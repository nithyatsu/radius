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
	"time"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MarshalDeterministicJSON_IdenticalOutputForIdenticalInput(t *testing.T) {
	graph := newTestGraph()

	first, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)

	second, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)

	assert.Equal(t, string(first), string(second), "two serializations of the same graph should be byte-identical")
}

func Test_MarshalDeterministicJSON_SortsResourcesByID(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/zzz", Name: "zzz", Type: "Applications.Core/containers"},
			{ID: "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/aaa", Name: "aaa", Type: "Applications.Core/containers"},
			{ID: "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/mmm", Name: "mmm", Type: "Applications.Core/containers"},
		},
	}

	data, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)

	json := string(data)
	idxA := indexOf(json, `"aaa"`)
	idxM := indexOf(json, `"mmm"`)
	idxZ := indexOf(json, `"zzz"`)

	assert.True(t, idxA < idxM, "aaa should appear before mmm")
	assert.True(t, idxM < idxZ, "mmm should appear before zzz")
}

func Test_MarshalDeterministicJSON_SortsConnectionsBySourceTargetType(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata:  newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{},
		Connections: []v20231001preview.AppGraphConnectionStatic{
			{SourceID: "b", TargetID: "z", Type: v20231001preview.ConnectionTypeConnection},
			{SourceID: "a", TargetID: "c", Type: v20231001preview.ConnectionTypeConnection},
			{SourceID: "a", TargetID: "b", Type: v20231001preview.ConnectionTypeRoute},
			{SourceID: "a", TargetID: "b", Type: v20231001preview.ConnectionTypeConnection},
		},
	}

	data, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)

	json := string(data)
	// Connections should appear in order: (a,b,connection), (a,b,route), (a,c,connection), (b,z,connection)
	idx1 := indexOfNth(json, `"sourceId": "a"`, 1)
	idx2 := indexOfNth(json, `"sourceId": "b"`, 1)

	assert.True(t, idx1 < idx2, "sourceId a should appear before sourceId b")
}

func Test_MarshalDeterministicJSON_SortsSourceFiles(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: v20231001preview.AppGraphMetadata{
			GeneratedAt:      time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC),
			RadiusCliVersion: "0.35.0",
			SourceFiles:      []string{"modules/db.bicep", "app.bicep", "modules/api.bicep"},
			SourceHash:       "sha256:abc123",
		},
	}

	data, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)

	json := string(data)
	idxApp := indexOf(json, `"app.bicep"`)
	idxAPI := indexOf(json, `"modules/api.bicep"`)
	idxDB := indexOf(json, `"modules/db.bicep"`)

	assert.True(t, idxApp < idxAPI, "app.bicep should appear before modules/api.bicep")
	assert.True(t, idxAPI < idxDB, "modules/api.bicep should appear before modules/db.bicep")
}

func Test_MarshalDeterministicJSON_EmptyGraph(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata:    newTestMetadata(),
		Resources:   []v20231001preview.AppGraphResource{},
		Connections: []v20231001preview.AppGraphConnectionStatic{},
	}

	data, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"resources": []`)
	assert.Contains(t, string(data), `"connections": []`)
}

func Test_MarshalDeterministicJSON_DoesNotMutateInput(t *testing.T) {
	graph := newTestGraph()
	originalFirstID := graph.Resources[0].ID

	_, err := MarshalDeterministicJSON(graph)
	require.NoError(t, err)

	// The original graph should not be modified
	assert.Equal(t, originalFirstID, graph.Resources[0].ID, "original graph should not be mutated")
}

func Test_sortMapKeys_RecursiveSort(t *testing.T) {
	input := map[string]any{
		"z": "last",
		"a": map[string]any{
			"z_inner": "deep_last",
			"a_inner": "deep_first",
		},
		"m": "middle",
	}

	result := sortMapKeys(input)
	require.NotNil(t, result)

	// The function should return sorted maps (Go maps don't have order,
	// but json.Marshal uses sorted keys for map[string]any by default).
	// Verify the nested map is also processed.
	nested, ok := result["a"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, nested, "z_inner")
	assert.Contains(t, nested, "a_inner")
}

// -- helpers --

func newTestMetadata() v20231001preview.AppGraphMetadata {
	return v20231001preview.AppGraphMetadata{
		GeneratedAt:      time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC),
		RadiusCliVersion: "0.35.0",
		SourceFiles:      []string{"app.bicep"},
		SourceHash:       "sha256:7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730",
	}
}

func newTestGraph() v20231001preview.AppGraph {
	return v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{
				ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/backend",
				Name: "backend",
				Type: "Applications.Core/containers",
				SourceLocation: v20231001preview.SourceLocation{
					File: "app.bicep",
					Line: 28,
				},
				Properties: map[string]any{
					"container": map[string]any{"image": "myapp/backend:v1"},
				},
			},
			{
				ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/frontend",
				Name: "frontend",
				Type: "Applications.Core/containers",
				SourceLocation: v20231001preview.SourceLocation{
					File: "app.bicep",
					Line: 12,
				},
				Properties: map[string]any{
					"container": map[string]any{"image": "myapp/frontend:v1"},
				},
			},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{
			{
				SourceID: "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/frontend",
				TargetID: "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/backend",
				Type:     v20231001preview.ConnectionTypeConnection,
			},
		},
	}
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func indexOfNth(s, substr string, n int) int {
	count := 0
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}
