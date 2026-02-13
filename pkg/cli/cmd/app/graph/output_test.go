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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/radius-project/radius/pkg/cli/output"
	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DefaultJSONPath(t *testing.T) {
	result := DefaultJSONPath("/home/user/project/app.bicep")
	expected := filepath.Join("/home/user/project", ".radius", "app-graph.json")
	assert.Equal(t, expected, result)
}

func Test_DefaultMarkdownPath(t *testing.T) {
	result := DefaultMarkdownPath("/home/user/project/app.bicep")
	expected := filepath.Join("/home/user/project", ".radius", "app-graph.md")
	assert.Equal(t, expected, result)
}

func Test_WriteGraphOutput_StdoutMode(t *testing.T) {
	graph := testOutputGraph()
	mockOutput := &output.MockOutput{}

	files, err := WriteGraphOutput(graph, OutputConfig{
		Stdout:        true,
		BicepFilePath: "/tmp/test/app.bicep",
	}, mockOutput)

	require.NoError(t, err)
	assert.Nil(t, files, "stdout mode should not return file paths")
	require.Len(t, mockOutput.Writes, 1, "should write JSON to stdout")

	// Verify it wrote JSON content
	logOutput, ok := mockOutput.Writes[0].(output.LogOutput)
	require.True(t, ok)
	assert.Contains(t, logOutput.Format, "%s")
}

func Test_WriteGraphOutput_DefaultFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	bicepPath := filepath.Join(tmpDir, "app.bicep")
	graph := testOutputGraph()
	mockOutput := &output.MockOutput{}

	files, err := WriteGraphOutput(graph, OutputConfig{
		BicepFilePath: bicepPath,
	}, mockOutput)

	require.NoError(t, err)
	require.Len(t, files, 1)

	expectedPath := filepath.Join(tmpDir, ".radius", "app-graph.json")
	assert.Equal(t, expectedPath, files[0])

	// Verify file exists and contains valid JSON
	data, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"metadata"`)
	assert.Contains(t, string(data), `"resources"`)
}

func Test_WriteGraphOutput_CustomOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom", "output.json")
	graph := testOutputGraph()
	mockOutput := &output.MockOutput{}

	files, err := WriteGraphOutput(graph, OutputConfig{
		OutputPath:    customPath,
		BicepFilePath: filepath.Join(tmpDir, "app.bicep"),
	}, mockOutput)

	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, customPath, files[0])

	data, err := os.ReadFile(customPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"metadata"`)
}

func Test_WriteGraphOutput_MarkdownFormat(t *testing.T) {
	tmpDir := t.TempDir()
	bicepPath := filepath.Join(tmpDir, "app.bicep")
	graph := testOutputGraph()
	mockOutput := &output.MockOutput{}

	files, err := WriteGraphOutput(graph, OutputConfig{
		Format:        "markdown",
		BicepFilePath: bicepPath,
	}, mockOutput)

	require.NoError(t, err)
	require.Len(t, files, 2)

	// First file is JSON
	jsonPath := filepath.Join(tmpDir, ".radius", "app-graph.json")
	assert.Equal(t, jsonPath, files[0])

	// Second file is Markdown
	mdPath := filepath.Join(tmpDir, ".radius", "app-graph.md")
	assert.Equal(t, mdPath, files[1])

	// Verify markdown content
	mdData, err := os.ReadFile(mdPath)
	require.NoError(t, err)
	assert.Contains(t, string(mdData), "# Application Graph")
	assert.Contains(t, string(mdData), "```mermaid")
}

func Test_WriteGraphOutput_DeterministicJSON(t *testing.T) {
	tmpDir := t.TempDir()
	bicepPath := filepath.Join(tmpDir, "app.bicep")
	graph := testOutputGraph()
	mockOutput := &output.MockOutput{}

	// Write once
	_, err := WriteGraphOutput(graph, OutputConfig{
		BicepFilePath: bicepPath,
	}, mockOutput)
	require.NoError(t, err)

	jsonPath := filepath.Join(tmpDir, ".radius", "app-graph.json")
	first, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	// Write again to a new directory
	tmpDir2 := t.TempDir()
	bicepPath2 := filepath.Join(tmpDir2, "app.bicep")
	mockOutput2 := &output.MockOutput{}

	_, err = WriteGraphOutput(graph, OutputConfig{
		BicepFilePath: bicepPath2,
	}, mockOutput2)
	require.NoError(t, err)

	jsonPath2 := filepath.Join(tmpDir2, ".radius", "app-graph.json")
	second, err := os.ReadFile(jsonPath2)
	require.NoError(t, err)

	assert.Equal(t, string(first), string(second), "identical graphs should produce identical JSON")
}

func testOutputGraph() v20231001preview.AppGraph {
	return v20231001preview.AppGraph{
		Metadata: v20231001preview.AppGraphMetadata{
			GeneratedAt:      time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC),
			RadiusCliVersion: "0.35.0",
			SourceFiles:      []string{"app.bicep"},
			SourceHash:       "sha256:abc123",
		},
		Resources: []v20231001preview.AppGraphResource{
			{
				ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/webapp",
				Name: "webapp",
				Type: "Applications.Core/containers",
				SourceLocation: v20231001preview.SourceLocation{
					File: "app.bicep",
					Line: 10,
				},
			},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{},
	}
}
