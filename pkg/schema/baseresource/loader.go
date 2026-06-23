/*
Copyright 2026 The Radius Authors.

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

package baseresource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"sigs.k8s.io/yaml"
)

// URIScheme is the custom URI scheme reserved for references to the Radius
// base resource manifest. RFC 3986 permits arbitrary custom URI schemes.
const URIScheme = "radius:"

// URI is the only supported value for the inheritance keyword. The reference
// targets the whole base schema; the JSON-Pointer fragment is intentionally
// omitted because the composition keyword (allOf) takes whole schemas, not
// properties-map subtrees.
const URI = "radius:base"

//go:embed base.yaml
var baseYAML []byte

var (
	baseSchemaOnce sync.Once
	baseSchema     *openapi3.Schema
	baseSchemaErr  error
)

// loadBaseSchema decodes the embedded base.yaml once and caches the result.
func loadBaseSchema() (*openapi3.Schema, error) {
	baseSchemaOnce.Do(func() {
		// Convert YAML to JSON first so kin-openapi's JSON-tagged struct fields
		// populate correctly. sigs.k8s.io/yaml does this transparently via
		// its YAMLToJSON helper invoked under the hood.
		jsonBytes, err := yaml.YAMLToJSON(baseYAML)
		if err != nil {
			baseSchemaErr = fmt.Errorf("baseresource: failed to convert embedded base.yaml to JSON: %w", err)
			return
		}

		var schema openapi3.Schema
		if err := json.Unmarshal(jsonBytes, &schema); err != nil {
			baseSchemaErr = fmt.Errorf("baseresource: failed to parse embedded base.yaml: %w", err)
			return
		}

		baseSchema = &schema
	})
	return baseSchema, baseSchemaErr
}

// PropertyNames returns the names of the four common Radius properties in the
// order they appear in the embedded base manifest. The set is frozen per
// FR-012 of the base resource manifest spec.
func PropertyNames() []string {
	return []string{"application", "environment", "connections", "codeReference"}
}

// Apply resolves a radius:base $ref entry inside the given schema's allOf
// array. If a matching entry is found, it is removed from allOf and the four
// base properties are merged into the schema's properties map using
// per-type-wins precedence (any per-type declaration of one of the four
// properties keeps its own shape; properties not declared per-type are
// copied from the base).
//
// If the schema's allOf array contains no entry with a "radius:"-scheme $ref,
// Apply is a no-op and the schema is returned unchanged.
//
// If the schema's allOf array contains a "radius:"-scheme $ref with any value
// other than "radius:base", Apply returns an actionable error per the
// inheritance keyword contract.
//
// Apply is safe to call on a nil schema (returns nil) and on a schema with no
// allOf array (returns nil).
func Apply(schema *openapi3.Schema) error {
	if schema == nil {
		return nil
	}
	if len(schema.AllOf) == 0 {
		return nil
	}

	// Walk allOf to locate a radius:-scheme $ref entry. We scan the whole array
	// rather than short-circuiting so an author who placed multiple radius:
	// refs gets a deterministic error on the first one.
	matchIndex := -1
	for i, entry := range schema.AllOf {
		if entry == nil {
			continue
		}
		if !strings.HasPrefix(entry.Ref, URIScheme) {
			continue
		}
		// Found a radius: scheme reference. Validate that it is the only legal value.
		if entry.Ref != URI {
			return fmt.Errorf(
				`baseresource: unsupported %s $ref %q at allOf[%d] — only %q is supported in this version`,
				URIScheme, entry.Ref, i, URI,
			)
		}
		matchIndex = i
		break
	}

	if matchIndex == -1 {
		// No radius: $ref in this schema's allOf — pass through unchanged.
		return nil
	}

	base, err := loadBaseSchema()
	if err != nil {
		return err
	}

	// Per-type-wins merge of base properties into the local properties map.
	if schema.Properties == nil {
		schema.Properties = openapi3.Schemas{}
	}
	for _, name := range PropertyNames() {
		if _, alreadyDeclared := schema.Properties[name]; alreadyDeclared {
			// The per-type declaration wins (FR-004). Discard the base's
			// version of this property entirely for this schema.
			continue
		}
		baseRef, ok := base.Properties[name]
		if !ok {
			// Defensive: every name in PropertyNames() must be present in the
			// embedded base.yaml. A missing entry indicates a developer error
			// in this package, not a user error.
			return fmt.Errorf(
				"baseresource: embedded base.yaml is missing property %q (this is a bug in pkg/schema/baseresource)",
				name,
			)
		}
		schema.Properties[name] = baseRef
	}

	// Drop the matched radius: $ref entry from allOf.
	schema.AllOf = append(schema.AllOf[:matchIndex], schema.AllOf[matchIndex+1:]...)

	return nil
}
