/*
Copyright 2023 The Radius Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package graph

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	v1 "github.com/radius-project/radius/pkg/armrpc/api/v1"
	corerpv20250801preview "github.com/radius-project/radius/pkg/corerp/api/v20250801preview"
	"github.com/radius-project/radius/pkg/graph/edges"
	"github.com/radius-project/radius/pkg/to"
)

// Default scope segments used when constructing fully-qualified Radius
// resource IDs for a modeled graph. The modeled graph is built from a Bicep
// definition without contacting any control plane, so the actual plane and
// resource group are unknown at build time.
const (
	defaultPlane             = "local"
	defaultResourceGroup     = "default"
	applicationsResourceType = "Applications.Core/applications"
	environmentsResourceType = "Applications.Core/environments"

	// Radius.Core control-plane types are containment scopes (see FR-005
	// in specs/004-graph-dependency-edges/spec.md). They are never graph
	// nodes and never edge targets. Adding a new excluded type is a
	// one-line edit here plus one new test case in modeled_test.go.
	radiusCoreApplicationsType = "Radius.Core/applications"
	radiusCoreEnvironmentsType = "Radius.Core/environments"
	recipePacksResourceType    = "Radius.Core/recipePacks"
)

// resourceIDExpression matches an ARM template resourceId() expression
// produced by `bicep build` for Radius resource references, e.g.
//
//	[resourceId('Applications.Datastores/redisCaches', 'cache')]
//
// Only the literal-argument form is supported; expressions that compute
// types or names dynamically are left unresolved (the connection edge is
// dropped from the modeled graph).
var resourceIDExpression = regexp.MustCompile(`^\[resourceId\(([^)]*)\)\]$`)

// excludedTypes is the set of Radius resource types that are neither
// graph nodes nor edge targets. See FR-005 in
// specs/004-graph-dependency-edges/spec.md. The map value is a zero-size
// struct because only membership matters. Add a new excluded type by
// adding an entry here plus one new test case in modeled_test.go.
var excludedTypes = map[string]struct{}{
	applicationsResourceType:   {},
	environmentsResourceType:   {},
	radiusCoreApplicationsType: {},
	radiusCoreEnvironmentsType: {},
	recipePacksResourceType:    {},
}

// BuildModeledGraph parses an ARM JSON template (typically the output of
// `bicep build` on an application's app.bicep) and returns the corresponding
// modeled application graph. The graph contains application resources,
// their connections and dependsOn relationships, and a stable diff hash for
// each resource. It does not contain output resources or runtime status —
// those are only available for planned and deployed graphs.
//
// Edges are surfaced from two authored sources:
//
//   - properties.connections[*].source — author-declared, tagged as Kind
//     Connection on the wire.
//   - dependsOn — implicit (Bicep-emitted), tagged as Kind Dependency.
//
// When both sources signal the same (source, target) pair for the same
// resource, Connection wins and a single edge is emitted. Every outbound
// edge is mirrored by a reciprocal inbound edge on the target. See
// pkg/graph/edges/ for the shared primitive that implements the rules.
func BuildModeledGraph(template map[string]any) (*corerpv20250801preview.ApplicationGraphResponse, error) {
	rawResources := collectResources(template["resources"])
	if rawResources == nil {
		return emptyGraph(), nil
	}

	graphResources := make([]*corerpv20250801preview.ApplicationGraphResource, 0, len(rawResources))
	extractInputs := make([]edges.Resource, 0, len(rawResources))
	for _, entry := range rawResources {
		resource, err := buildModeledResource(entry)
		if err != nil {
			return nil, err
		}
		if resource == nil {
			continue
		}
		graphResources = append(graphResources, resource)

		properties, _ := entry["properties"].(map[string]any)
		rawDependsOn, _ := entry["dependsOn"].([]any)
		extractInputs = append(extractInputs, edges.Resource{
			ID:          to.String(resource.ID),
			Type:        to.String(resource.Type),
			Connections: resolveConnectionSources(properties),
			DependsOn:   resolveDependsOn(rawDependsOn),
		})
	}

	graph := &corerpv20250801preview.ApplicationGraphResponse{Resources: graphResources}
	applyEdges(graph, edges.ExtractEdges(extractInputs, excludedTypes))
	return graph, nil
}

// applyEdges attaches each Edge returned by the extractor to the
// Connections slice of the resource that owns it (Source for Outbound,
// Target for Inbound). ExtractEdges already returns entries sorted
// deterministically, so the resulting Connections slices are ordered
// stably across runs.
func applyEdges(graph *corerpv20250801preview.ApplicationGraphResponse, extracted []edges.Edge) {
	if graph == nil || len(extracted) == 0 {
		return
	}
	byID := make(map[string]*corerpv20250801preview.ApplicationGraphResource, len(graph.Resources))
	for _, r := range graph.Resources {
		if r != nil && r.ID != nil {
			byID[*r.ID] = r
		}
	}
	for _, e := range extracted {
		owner, ok := byID[e.Owner()]
		if !ok {
			continue
		}
		owner.Connections = append(owner.Connections, &corerpv20250801preview.ApplicationGraphConnection{
			ID:        to.Ptr(e.Peer()),
			Direction: to.Ptr(corerpv20250801preview.Direction(e.Direction)),
			Kind:      to.Ptr(corerpv20250801preview.ConnectionKind(e.Kind)),
		})
	}
}

// collectResources normalizes the "resources" section of an ARM JSON
// template into a slice of resource entries in the classic (flat) shape
// expected by buildModeledResource. Bicep emits two layouts:
//
//   - languageVersion 1.x: an array of entries, each with top-level
//     "type", "name", and "properties".
//   - languageVersion 2.0 (symbolic-name codegen): an object keyed by
//     symbolic name. Each entry's "type" includes the API version suffix
//     ("Foo/bar@2024-01-01"), the resource's name and authored properties
//     are nested under "properties.name" / "properties.properties", and
//     "dependsOn" plus connection "[reference('sym').id]" expressions
//     refer to other resources by symbolic name.
//
// In the symbolic case we strip the @version, hoist the inner properties,
// and rewrite symbolic references to the equivalent
// [resourceId('TYPE', 'NAME')] form so the rest of the pipeline (and the
// stable diff hash) is independent of codegen mode.
func collectResources(raw any) []map[string]any {
	switch v := raw.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if entry, ok := item.(map[string]any); ok {
				out = append(out, entry)
			}
		}
		return out
	case map[string]any:
		symbols := buildSymbolTable(v)
		// Iterate symbolic-name keys in sorted order so the resulting
		// resource slice (and the stable diff hash derived from it) is
		// deterministic across runs. Go map iteration is randomized,
		// which would otherwise produce noisy diffs.
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]map[string]any, 0, len(v))
		for _, k := range keys {
			entry, ok := v[k].(map[string]any)
			if !ok {
				continue
			}
			out = append(out, normalizeSymbolicEntry(entry, symbols))
		}
		return out
	default:
		return nil
	}
}

// symbolicReference matches a reference() expression that points to a
// resource by symbolic name in languageVersion 2.0 templates.
var symbolicReference = regexp.MustCompile(`^\[reference\('([^']+)'\)\.[^\]]+\]$`)

// symbolEntry holds the version-stripped resource type and authored name
// for a single symbolic-name entry.
type symbolEntry struct {
	resourceType string
	name         string
}

func buildSymbolTable(resources map[string]any) map[string]symbolEntry {
	table := make(map[string]symbolEntry, len(resources))
	for symbol, item := range resources {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typeWithVersion, _ := entry["type"].(string)
		wrapper, _ := entry["properties"].(map[string]any)
		name, _ := wrapper["name"].(string)
		table[symbol] = symbolEntry{
			resourceType: stripAPIVersion(typeWithVersion),
			name:         name,
		}
	}
	return table
}

func normalizeSymbolicEntry(entry map[string]any, symbols map[string]symbolEntry) map[string]any {
	resourceType := stripAPIVersion(stringAt(entry, "type"))
	wrapper, _ := entry["properties"].(map[string]any)
	name, _ := wrapper["name"].(string)
	innerProps, _ := wrapper["properties"].(map[string]any)
	rewriteSymbolicConnections(innerProps, symbols)

	rawDeps, _ := entry["dependsOn"].([]any)
	newDeps := make([]any, 0, len(rawDeps))
	for _, d := range rawDeps {
		s, ok := d.(string)
		if !ok {
			continue
		}
		if sym, ok := symbols[s]; ok && sym.resourceType != "" && sym.name != "" {
			newDeps = append(newDeps, fmt.Sprintf("[resourceId('%s', '%s')]", sym.resourceType, sym.name))
			continue
		}
		newDeps = append(newDeps, s)
	}

	return map[string]any{
		"type":       resourceType,
		"name":       name,
		"properties": innerProps,
		"dependsOn":  newDeps,
	}
}

// rewriteSymbolicConnections rewrites each connection's "source" expression
// from [reference('sym').id] (or .properties.id) to the equivalent
// [resourceId('TYPE','NAME')] form, in place.
func rewriteSymbolicConnections(properties map[string]any, symbols map[string]symbolEntry) {
	if properties == nil {
		return
	}
	connections, ok := properties["connections"].(map[string]any)
	if !ok {
		return
	}
	for _, raw := range connections {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		source, ok := entry["source"].(string)
		if !ok {
			continue
		}
		matches := symbolicReference.FindStringSubmatch(source)
		if len(matches) != 2 {
			continue
		}
		sym, ok := symbols[matches[1]]
		if !ok || sym.resourceType == "" || sym.name == "" {
			continue
		}
		entry["source"] = fmt.Sprintf("[resourceId('%s', '%s')]", sym.resourceType, sym.name)
	}
}

// stripAPIVersion removes the trailing "@version" suffix from an ARM type
// string. languageVersion 2.0 emits types as "Foo/bar@2024-01-01" while
// the classic codegen uses a separate "apiVersion" field; the modeled
// graph stores only the bare type.
func stripAPIVersion(t string) string {
	if before, _, ok := strings.Cut(t, "@"); ok {
		return before
	}
	return t
}

func stringAt(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

// buildModeledResource converts a single ARM JSON resource entry into an
// ApplicationGraphResource. It returns (nil, nil) for entries that should
// not appear as graph nodes (Radius applications and environments are
// containers rather than graph members, and recipe packs are catalog
// resources defined only in Radius.Core that hold reusable recipes).
func buildModeledResource(entry map[string]any) (*corerpv20250801preview.ApplicationGraphResource, error) {
	resourceType, _ := entry["type"].(string)
	name, _ := entry["name"].(string)
	if resourceType == "" || name == "" {
		return nil, nil
	}
	if strings.EqualFold(resourceType, applicationsResourceType) ||
		strings.EqualFold(resourceType, environmentsResourceType) ||
		strings.EqualFold(resourceType, radiusCoreApplicationsType) ||
		strings.EqualFold(resourceType, radiusCoreEnvironmentsType) ||
		strings.EqualFold(resourceType, recipePacksResourceType) {
		return nil, nil
	}

	properties, _ := entry["properties"].(map[string]any)
	rawDependsOn, _ := entry["dependsOn"].([]any)
	dependsOn := resolveDependsOn(rawDependsOn)

	hash, err := ComputeDiffHash(properties, dependsOn...)
	if err != nil {
		return nil, fmt.Errorf("compute diffHash for %s/%s: %w", resourceType, name, err)
	}

	return &corerpv20250801preview.ApplicationGraphResource{
		ID:                to.Ptr(buildResourceID(resourceType, name)),
		Name:              to.Ptr(name),
		Type:              to.Ptr(resourceType),
		ProvisioningState: to.Ptr(string(v1.ProvisioningStateNotSpecified)),
		Connections:       []*corerpv20250801preview.ApplicationGraphConnection{},
		OutputResources:   []*corerpv20250801preview.ApplicationGraphOutputResource{},
		DiffHash:          to.Ptr(hash),
	}, nil
}

// resolveDependsOn resolves each ARM expression in an ARM JSON dependsOn
// list to a fully-qualified Radius resource ID. Unresolvable entries are
// dropped so the diffHash remains stable across builds.
func resolveDependsOn(in []any) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		expr, ok := v.(string)
		if !ok {
			continue
		}
		if resolved := resolveResourceIDExpression(expr); resolved != "" {
			out = append(out, resolved)
		}
	}
	return out
}

// resolveConnectionSources walks the resource's properties.connections
// map and returns each entry's `source` after resolving it from an ARM
// [resourceId(...)] expression to a fully-qualified Radius resource ID.
// Unresolvable sources are dropped; the caller sees a Connections
// slice of pre-resolved canonical IDs suitable for edges.ExtractEdges.
func resolveConnectionSources(properties map[string]any) []string {
	if properties == nil {
		return nil
	}
	connections, ok := properties["connections"].(map[string]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(connections))
	for _, raw := range connections {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		source, _ := entry["source"].(string)
		resolved := resolveResourceIDExpression(source)
		if resolved == "" {
			continue
		}
		out = append(out, resolved)
	}
	return out
}

// resolveResourceIDExpression converts an ARM expression of the form
// [resourceId('TYPE', 'NAME')] into a fully-qualified Radius resource ID.
// Returns an empty string if the input is not a recognised literal-argument
// resourceId expression.
func resolveResourceIDExpression(expr string) string {
	if expr == "" {
		return ""
	}
	matches := resourceIDExpression.FindStringSubmatch(expr)
	if len(matches) != 2 {
		return ""
	}
	args := splitResourceIDArgs(matches[1])
	if len(args) < 2 {
		return ""
	}
	resourceType := strings.Trim(strings.TrimSpace(args[0]), "'")
	name := strings.Trim(strings.TrimSpace(args[1]), "'")
	if resourceType == "" || name == "" {
		return ""
	}
	return buildResourceID(resourceType, name)
}

// splitResourceIDArgs splits the comma-separated argument list of an ARM
// resourceId() expression, treating commas inside single-quoted strings as
// literal characters.
func splitResourceIDArgs(s string) []string {
	parts := []string{}
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			inQuote = !inQuote
			current.WriteByte(c)
			continue
		}
		if c == ',' && !inQuote {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(c)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// buildResourceID returns the fully-qualified Radius resource ID under the
// default plane and resource group used for modeled graphs.
func buildResourceID(resourceType, name string) string {
	return fmt.Sprintf("/planes/radius/%s/resourcegroups/%s/providers/%s/%s", defaultPlane, defaultResourceGroup, resourceType, name)
}

// emptyGraph returns a fresh graph with an empty (non-nil) Resources slice
// so it serializes to "resources": [] rather than "resources": null.
func emptyGraph() *corerpv20250801preview.ApplicationGraphResponse {
	return &corerpv20250801preview.ApplicationGraphResponse{
		Resources: []*corerpv20250801preview.ApplicationGraphResource{},
	}
}
