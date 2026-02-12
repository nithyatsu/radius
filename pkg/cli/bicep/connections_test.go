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

package bicep

import (
	"testing"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/stretchr/testify/require"
)

func Test_DetectConnections_PropertyConnections(t *testing.T) {
	resources := []ExtractedResource{
		{
			Name: "frontend",
			Type: "Applications.Core/containers",
			Properties: map[string]any{
				"connections": map[string]any{
					"backend": map[string]any{
						"source": "backend-api",
					},
				},
			},
		},
		{
			Name: "backend-api",
			Type: "Applications.Core/containers",
			Properties: map[string]any{
				"connections": map[string]any{
					"redis": map[string]any{
						"source": "cache",
					},
				},
			},
		},
		{
			Name: "cache",
			Type: "Applications.Datastores/redisCaches",
		},
	}

	connections := DetectConnections(resources)
	require.Len(t, connections, 2)

	// frontend -> backend-api
	require.Equal(t, BuildResourceID(resources[0]), connections[0].SourceID)
	require.Equal(t, BuildResourceID(resources[1]), connections[0].TargetID)
	require.Equal(t, v20231001preview.ConnectionTypeConnection, connections[0].Type)

	// backend-api -> cache
	require.Equal(t, BuildResourceID(resources[1]), connections[1].SourceID)
	require.Equal(t, BuildResourceID(resources[2]), connections[1].TargetID)
	require.Equal(t, v20231001preview.ConnectionTypeConnection, connections[1].Type)
}

func Test_DetectConnections_RouteConnections(t *testing.T) {
	resources := []ExtractedResource{
		{
			Name: "gateway",
			Type: "Applications.Core/gateways",
			Properties: map[string]any{
				"routes": map[string]any{
					"frontend": map[string]any{
						"destination": "frontend",
					},
				},
			},
		},
		{
			Name: "frontend",
			Type: "Applications.Core/containers",
		},
	}

	connections := DetectConnections(resources)
	require.Len(t, connections, 1)

	require.Equal(t, BuildResourceID(resources[0]), connections[0].SourceID)
	require.Equal(t, BuildResourceID(resources[1]), connections[0].TargetID)
	require.Equal(t, v20231001preview.ConnectionTypeRoute, connections[0].Type)
}

func Test_DetectConnections_DependsOnConnections(t *testing.T) {
	resources := []ExtractedResource{
		{
			Name:      "backend",
			Type:      "Applications.Core/containers",
			DependsOn: []string{"cache"},
		},
		{
			Name: "cache",
			Type: "Applications.Datastores/redisCaches",
		},
	}

	connections := DetectConnections(resources)
	require.Len(t, connections, 1)

	require.Equal(t, BuildResourceID(resources[0]), connections[0].SourceID)
	require.Equal(t, BuildResourceID(resources[1]), connections[0].TargetID)
	require.Equal(t, v20231001preview.ConnectionTypeDependsOn, connections[0].Type)
}

func Test_DetectConnections_NoConnections(t *testing.T) {
	resources := []ExtractedResource{
		{
			Name: "frontend",
			Type: "Applications.Core/containers",
		},
		{
			Name: "backend",
			Type: "Applications.Core/containers",
		},
	}

	connections := DetectConnections(resources)
	require.Empty(t, connections)
}

func Test_DetectConnections_FullResourceIDSource(t *testing.T) {
	targetID := "/planes/radius/local/resourceGroups/default/providers/Applications.Datastores/redisCaches/cache"
	resources := []ExtractedResource{
		{
			Name: "backend",
			Type: "Applications.Core/containers",
			Properties: map[string]any{
				"connections": map[string]any{
					"redis": map[string]any{
						"source": targetID,
					},
				},
			},
		},
	}

	connections := DetectConnections(resources)
	require.Len(t, connections, 1)
	require.Equal(t, targetID, connections[0].TargetID)
}

func Test_DetectConnections_EmptyResources(t *testing.T) {
	connections := DetectConnections([]ExtractedResource{})
	require.Empty(t, connections)
}

func Test_DetectConnections_NilProperties(t *testing.T) {
	resources := []ExtractedResource{
		{
			Name: "cache",
			Type: "Applications.Datastores/redisCaches",
		},
	}

	connections := DetectConnections(resources)
	require.Empty(t, connections)
}

func Test_extractNameFromResourceID(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "standard resourceId expression",
			expr:     "resourceId('Applications.Datastores/redisCaches', 'cache')",
			expected: "cache",
		},
		{
			name:     "single argument",
			expr:     "resourceId('cache')",
			expected: "",
		},
		{
			name:     "empty",
			expr:     "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractNameFromResourceID(tc.expr)
			require.Equal(t, tc.expected, result)
		})
	}
}
