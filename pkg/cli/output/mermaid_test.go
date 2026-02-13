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
	"testing"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateMermaid_ContainerRectangleShape(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "webapp", Type: "Applications.Core/containers"},
		},
	}

	result := GenerateMermaid(graph)
	assert.Contains(t, result, "[webapp]")
	assert.Contains(t, result, "flowchart TD")
	assert.Contains(t, result, "```mermaid")
}

func Test_GenerateMermaid_GatewayDiamondShape(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "gateway", Type: "Applications.Core/gateways"},
		},
	}

	result := GenerateMermaid(graph)
	assert.Contains(t, result, "{gateway}")
}

func Test_GenerateMermaid_DatabaseCylinderShape(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
	}{
		{"redis cache", "Applications.Datastores/redisCaches"},
		{"mongo database", "Applications.Datastores/mongoDatabases"},
		{"sql database", "Applications.Datastores/sqlDatabases"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := v20231001preview.AppGraph{
				Metadata: newTestMetadata(),
				Resources: []v20231001preview.AppGraphResource{
					{ID: "id1", Name: "mydb", Type: tt.resourceType},
				},
			}

			result := GenerateMermaid(graph)
			assert.Contains(t, result, "[(mydb)]")
		})
	}
}

func Test_GenerateMermaid_DefaultRoundedRectangle(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "unknown", Type: "Applications.Core/extenders"},
		},
	}

	result := GenerateMermaid(graph)
	assert.Contains(t, result, "(unknown)")
}

func Test_GenerateMermaid_ConnectionEdges(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "frontend", Type: "Applications.Core/containers"},
			{ID: "id2", Name: "backend", Type: "Applications.Core/containers"},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{
			{SourceID: "id1", TargetID: "id2", Type: v20231001preview.ConnectionTypeConnection},
		},
	}

	result := GenerateMermaid(graph)
	// Connection type "connection" should not have a label
	assert.Contains(t, result, "-->")
	assert.NotContains(t, result, "-->|connection|")
}

func Test_GenerateMermaid_RouteEdgeLabel(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "gateway", Type: "Applications.Core/gateways"},
			{ID: "id2", Name: "frontend", Type: "Applications.Core/containers"},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{
			{SourceID: "id1", TargetID: "id2", Type: v20231001preview.ConnectionTypeRoute},
		},
	}

	result := GenerateMermaid(graph)
	assert.Contains(t, result, "-->|route|")
}

func Test_GenerateMermaid_DependsOnEdgeLabel(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "a", Type: "Applications.Core/containers"},
			{ID: "id2", Name: "b", Type: "Applications.Core/containers"},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{
			{SourceID: "id1", TargetID: "id2", Type: v20231001preview.ConnectionTypeDependsOn},
		},
	}

	result := GenerateMermaid(graph)
	assert.Contains(t, result, "-->|dependsOn|")
}

func Test_GenerateMermaid_EmptyGraph(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata:    newTestMetadata(),
		Resources:   []v20231001preview.AppGraphResource{},
		Connections: []v20231001preview.AppGraphConnectionStatic{},
	}

	result := GenerateMermaid(graph)
	require.Contains(t, result, "```mermaid")
	require.Contains(t, result, "flowchart TD")
}

func Test_GenerateMermaid_DeterministicOutput(t *testing.T) {
	graph := newTestGraph()

	first := GenerateMermaid(graph)
	second := GenerateMermaid(graph)

	assert.Equal(t, first, second, "two calls should produce identical output")
}

func Test_sanitizeMermaidLabel_SpecialCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"has[brackets]", "has&#91;brackets&#93;"},
		{"has{braces}", "has&#123;braces&#125;"},
		{"has(parens)", "has&#40;parens&#41;"},
		{"has|pipe", "has&#124;pipe"},
		{"has<angle>", "has&lt;angle&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeMermaidLabel(tt.input))
		})
	}
}

func Test_GenerateMermaid_SkipsConnectionsWithUnknownNodes(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "frontend", Type: "Applications.Core/containers"},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{
			{SourceID: "id1", TargetID: "id_unknown", Type: v20231001preview.ConnectionTypeConnection},
		},
	}

	result := GenerateMermaid(graph)
	// Connection referencing unknown node should be skipped
	assert.NotContains(t, result, "-->")
}
