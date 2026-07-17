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
	"strings"
	"testing"

	corerpv20250801preview "github.com/radius-project/radius/pkg/corerp/api/v20250801preview"
	"github.com/stretchr/testify/require"
)

func TestBuildModeledGraph_EmptyTemplate(t *testing.T) {
	t.Parallel()

	graph, err := BuildModeledGraph(map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, graph)
	require.NotNil(t, graph.Resources)
	require.Empty(t, graph.Resources)
}

func TestBuildModeledGraph_SkipsContainersAndRecipePacks(t *testing.T) {
	t.Parallel()

	template := map[string]any{
		"resources": []any{
			// Applications.Core (legacy) scope resources — excluded.
			map[string]any{"type": "Applications.Core/applications", "name": "myapp"},
			map[string]any{"type": "Applications.Core/environments", "name": "myenv"},
			// Radius.Core (new) scope resources — also excluded per FR-005.
			map[string]any{"type": "Radius.Core/applications", "name": "myapp2"},
			map[string]any{"type": "Radius.Core/environments", "name": "myenv2"},
			// Radius.Core catalog resource — excluded.
			map[string]any{"type": "Radius.Core/recipePacks", "name": "mypack"},
			// Only this container should survive as a graph node.
			map[string]any{"type": "Applications.Core/containers", "name": "frontend",
				"properties": map[string]any{"image": "nginx"}},
		},
	}

	graph, err := BuildModeledGraph(template)
	require.NoError(t, err)
	require.Len(t, graph.Resources, 1)
	require.Equal(t, "frontend", *graph.Resources[0].Name)
	require.Equal(t, "Applications.Core/containers", *graph.Resources[0].Type)
}

// TestBuildModeledGraph_ExcludesRadiusCoreApplicationsFromRabbitMQShape
// locks in the rabbitmq-app regression fix: a template that declares a
// Radius.Core/applications resource and whose children reference it
// (application: app.id) must produce exactly the two non-scope nodes
// (consumer + rabbitmq) with no Radius.Core/applications node.
//
// This is the concrete SC-001 fixture from
// specs/004-graph-dependency-edges/spec.md, using the languageVersion
// 2.0 symbolic-name codegen shape emitted by `bicep build`.
func TestBuildModeledGraph_ExcludesRadiusCoreApplicationsFromRabbitMQShape(t *testing.T) {
	t.Parallel()

	template := map[string]any{
		"languageVersion": "2.0",
		"resources": map[string]any{
			"app": map[string]any{
				"type": "Radius.Core/applications@2025-08-01-preview",
				"properties": map[string]any{
					"name": "rabbitmq-app",
					"properties": map[string]any{
						"environment": "[parameters('environment')]",
					},
				},
			},
			"rabbitmq": map[string]any{
				"type": "Radius.Messaging/rabbitMQ@2025-08-01-preview",
				"properties": map[string]any{
					"name": "rabbitmq",
					"properties": map[string]any{
						"application": "[reference('app').id]",
						"queue":       "jobs",
					},
				},
				"dependsOn": []any{"app"},
			},
			"consumer": map[string]any{
				"type": "Radius.Compute/containers@2025-08-01-preview",
				"properties": map[string]any{
					"name": "consumer",
					"properties": map[string]any{
						"application": "[reference('app').id]",
					},
				},
				"dependsOn": []any{"app", "rabbitmq"},
			},
		},
	}

	graph, err := BuildModeledGraph(template)
	require.NoError(t, err)
	require.Len(t, graph.Resources, 2, "Radius.Core/applications must not appear as a graph node")

	types := map[string]bool{}
	for _, r := range graph.Resources {
		require.NotNil(t, r.Type)
		types[*r.Type] = true
		require.NotEqual(t, "Radius.Core/applications", *r.Type)
	}
	require.True(t, types["Radius.Compute/containers"], "consumer node should survive")
	require.True(t, types["Radius.Messaging/rabbitMQ"], "rabbitmq node should survive")
}

func TestBuildModeledGraph_BuildsResourceID(t *testing.T) {
	t.Parallel()

	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type":       "Applications.Core/containers",
				"name":       "frontend",
				"properties": map[string]any{"image": "nginx"},
			},
		},
	}

	graph, err := BuildModeledGraph(template)
	require.NoError(t, err)
	require.Len(t, graph.Resources, 1)
	require.Equal(t,
		"/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend",
		*graph.Resources[0].ID,
	)
	require.Equal(t, "NotSpecified", *graph.Resources[0].ProvisioningState)
	require.True(t, strings.HasPrefix(*graph.Resources[0].DiffHash, "sha256:"))
}

func TestBuildModeledGraph_OutboundConnectionsResolved(t *testing.T) {
	t.Parallel()

	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "frontend",
				"properties": map[string]any{
					"image": "nginx",
					"connections": map[string]any{
						"cache": map[string]any{
							"source": "[resourceId('Applications.Datastores/redisCaches', 'cache')]",
						},
					},
				},
			},
			map[string]any{
				"type":       "Applications.Datastores/redisCaches",
				"name":       "cache",
				"properties": map[string]any{},
			},
		},
	}

	graph, err := BuildModeledGraph(template)
	require.NoError(t, err)
	require.Len(t, graph.Resources, 2)

	frontend := findResource(t, graph, "frontend")
	require.Len(t, frontend.Connections, 1)
	require.Equal(t, corerpv20250801preview.DirectionOutbound, *frontend.Connections[0].Direction)
	require.Equal(t,
		"/planes/radius/local/resourcegroups/default/providers/Applications.Datastores/redisCaches/cache",
		*frontend.Connections[0].ID,
	)
}

func TestBuildModeledGraph_InboundConnectionsAreReciprocal(t *testing.T) {
	t.Parallel()

	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "frontend",
				"properties": map[string]any{
					"connections": map[string]any{
						"cache": map[string]any{
							"source": "[resourceId('Applications.Datastores/redisCaches', 'cache')]",
						},
					},
				},
			},
			map[string]any{
				"type":       "Applications.Datastores/redisCaches",
				"name":       "cache",
				"properties": map[string]any{},
			},
		},
	}

	graph, err := BuildModeledGraph(template)
	require.NoError(t, err)
	cache := findResource(t, graph, "cache")
	require.Len(t, cache.Connections, 1)
	require.Equal(t, corerpv20250801preview.DirectionInbound, *cache.Connections[0].Direction)
	require.Equal(t,
		"/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend",
		*cache.Connections[0].ID,
	)
}

func TestBuildModeledGraph_DropsUnresolvableConnections(t *testing.T) {
	t.Parallel()

	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "frontend",
				"properties": map[string]any{
					"connections": map[string]any{
						"dyn": map[string]any{"source": "[parameters('something')]"},
					},
				},
			},
		},
	}

	graph, err := BuildModeledGraph(template)
	require.NoError(t, err)
	require.Empty(t, graph.Resources[0].Connections)
}

func TestBuildModeledGraph_DependsOnAffectsDiffHash(t *testing.T) {
	t.Parallel()

	withDep := map[string]any{
		"resources": []any{
			map[string]any{
				"type":       "Applications.Core/containers",
				"name":       "frontend",
				"properties": map[string]any{"image": "nginx"},
				"dependsOn":  []any{"[resourceId('Applications.Datastores/redisCaches', 'cache')]"},
			},
		},
	}
	withoutDep := map[string]any{
		"resources": []any{
			map[string]any{
				"type":       "Applications.Core/containers",
				"name":       "frontend",
				"properties": map[string]any{"image": "nginx"},
			},
		},
	}

	g1, err := BuildModeledGraph(withDep)
	require.NoError(t, err)
	g2, err := BuildModeledGraph(withoutDep)
	require.NoError(t, err)

	require.NotEqual(t, *g1.Resources[0].DiffHash, *g2.Resources[0].DiffHash)
}

func TestResolveResourceIDExpression(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"valid", "[resourceId('Applications.Core/containers', 'web')]",
			"/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/web"},
		{"empty", "", ""},
		{"non-resourceid", "[parameters('foo')]", ""},
		{"missing args", "[resourceId('only-one')]", ""},
	}
	for _, tc := range cases {
		got := resolveResourceIDExpression(tc.in)
		require.Equal(t, tc.want, got, tc.name)
	}
}

func findResource(t *testing.T, g *corerpv20250801preview.ApplicationGraphResponse, name string) *corerpv20250801preview.ApplicationGraphResource {
	t.Helper()
	for _, r := range g.Resources {
		if r != nil && r.Name != nil && *r.Name == name {
			return r
		}
	}
	t.Fatalf("resource %q not found", name)
	return nil
}
