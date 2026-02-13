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

package output

import (
	"encoding/json"
	"sort"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
)

// MarshalDeterministicJSON serializes an AppGraph to JSON with deterministic
// ordering. Resources are sorted alphabetically by ID, connections are sorted
// by (sourceId, targetId, type), and map keys are sorted alphabetically.
// This ensures identical inputs always produce byte-identical output, which is
// essential for meaningful diffs in version control.
func MarshalDeterministicJSON(graph v20231001preview.AppGraph) ([]byte, error) {
	sorted := sortAppGraph(graph)
	return json.MarshalIndent(sorted, "", "  ")
}

// sortAppGraph returns a copy of the graph with resources and connections
// sorted deterministically.
func sortAppGraph(graph v20231001preview.AppGraph) v20231001preview.AppGraph {
	// Sort resources by ID
	resources := make([]v20231001preview.AppGraphResource, len(graph.Resources))
	copy(resources, graph.Resources)
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ID < resources[j].ID
	})

	// Sort properties maps within each resource for deterministic output
	for i := range resources {
		if resources[i].Properties != nil {
			resources[i].Properties = sortMapKeys(resources[i].Properties)
		}
	}

	// Sort connections by (sourceId, targetId, type)
	connections := make([]v20231001preview.AppGraphConnectionStatic, len(graph.Connections))
	copy(connections, graph.Connections)
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].SourceID != connections[j].SourceID {
			return connections[i].SourceID < connections[j].SourceID
		}
		if connections[i].TargetID != connections[j].TargetID {
			return connections[i].TargetID < connections[j].TargetID
		}
		return connections[i].Type < connections[j].Type
	})

	// Sort source files in metadata
	sourceFiles := make([]string, len(graph.Metadata.SourceFiles))
	copy(sourceFiles, graph.Metadata.SourceFiles)
	sort.Strings(sourceFiles)

	return v20231001preview.AppGraph{
		Metadata: v20231001preview.AppGraphMetadata{
			GeneratedAt:      graph.Metadata.GeneratedAt,
			RadiusCliVersion: graph.Metadata.RadiusCliVersion,
			SourceFiles:      sourceFiles,
			SourceHash:       graph.Metadata.SourceHash,
			GitCommit:        graph.Metadata.GitCommit,
		},
		Resources:   resources,
		Connections: connections,
	}
}

// sortMapKeys recursively sorts all map keys in a map[string]any. This ensures
// deterministic JSON output for properties maps that may contain nested maps.
func sortMapKeys(m map[string]any) map[string]any {
	sorted := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			sorted[k] = sortMapKeys(val)
		default:
			sorted[k] = v
		}
	}
	return sorted
}
