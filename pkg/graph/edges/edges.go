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

package edges

// Edge kind constants. Values match the ConnectionKind enum on the
// Radius.Core/2025-08-01-preview wire model; keeping them as untyped
// strings here avoids importing the generated API package into this
// otherwise dependency-free package. Callers convert Edge.Kind into the
// generated ConnectionKind pointer when materializing the wire model.
const (
	// KindConnection tags edges derived from a resource's
	// properties.connections block (author-declared).
	KindConnection = "Connection"

	// KindDependency tags edges derived from a resource's dependsOn
	// list (implicit, inferred from Bicep's compiled template or the
	// caller-supplied dependsOnEdges on the runtime GetGraphRequest
	// wire).
	KindDependency = "Dependency"
)

// Edge direction constants. Values match the Direction enum on the
// Radius.Core/2025-08-01-preview wire model.
const (
	// DirectionOutbound identifies the source resource's view of an
	// edge: "I depend on / connect to Target".
	DirectionOutbound = "Outbound"

	// DirectionInbound identifies the target resource's view of the
	// same edge: "Source depends on / connects to me". Every outbound
	// edge is mirrored by an inbound edge on the target with the same
	// Kind.
	DirectionInbound = "Inbound"
)

// Resource is a graph-eligible Radius resource as seen by the edge
// extractor. Callers convert their own representation (ARM JSON entry,
// stored resource record, etc.) into this shape before calling
// ExtractEdges.
type Resource struct {
	// ID is the canonical Radius resource ID
	// ("/planes/radius/local/resourcegroups/…/providers/{ns}/{type}/{name}").
	ID string

	// Type is the Radius resource type ("Radius.Compute/containers").
	Type string

	// Properties is the resource's authored properties. The extractor
	// inspects Properties["connections"] for author-declared
	// connections. Other keys are ignored in Phase 1.
	Properties map[string]any

	// DependsOn is the list of already-resolved canonical Radius
	// resource IDs the resource declares as build-time dependencies.
	// The static caller populates this from resolveDependsOn. The
	// runtime caller (Phase 2) populates it from caller-supplied
	// dependsOnEdges on the GetGraphRequest wire.
	//
	// Entries MUST be canonical resource IDs, not ARM
	// "[resourceId('T','N')]" expressions and not symbolic names.
	DependsOn []string
}

// Edge is a single directed edge in the application graph.
type Edge struct {
	// Source is the canonical resource ID of the edge's source node.
	Source string

	// Target is the canonical resource ID of the edge's target node.
	Target string

	// Direction is DirectionOutbound or DirectionInbound. Every
	// outbound edge Source→Target is mirrored by an inbound edge on
	// Target with the same Kind.
	Direction string

	// Kind is KindConnection (from properties.connections) or
	// KindDependency (from DependsOn). Case-sensitive; matches the
	// wire enum values on Radius.Core/2025-08-01-preview.
	Kind string
}

// ExtractEdges returns the deduplicated, mirrored edge list for the
// given resources, dropping any edge whose source or target type is
// present in the excluded set.
//
// excluded holds canonical "Namespace/type" strings that MUST NOT
// appear as graph nodes or edge targets. Callers control membership so
// the same primitive can be used with different exclusion sets (static
// vs runtime).
//
// The output is sorted deterministically by (Source, Target,
// Direction, Kind) so callers can compare or diff it.
//
// ExtractEdges never returns an error: unresolvable DependsOn entries
// and edges targeting excluded types are silently dropped. See the
// spec's edge-cases section for the exhaustive rules.
//
// Future extensibility: if a second configuration knob becomes
// necessary (for example, an option to emit diagnostic reasons for
// dropped edges), promote both arguments into an ExtractOptions
// struct. Keeping a single positional argument today (Constitution
// VII, Simplicity Over Cleverness) avoids paying that cost until a
// real need materializes.
//
// The stub implementation returns nil; the real implementation lands
// in a follow-up commit (Phase 5 / US1 of the tasks list).
func ExtractEdges(resources []Resource, excluded map[string]struct{}) []Edge {
	_ = resources
	_ = excluded
	return nil
}
