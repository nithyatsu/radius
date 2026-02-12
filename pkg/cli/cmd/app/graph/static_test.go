// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/radius-project/radius/pkg/cli/bicep"
	"github.com/radius-project/radius/pkg/cli/output"
	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_StaticRunner_Validate(t *testing.T) {
	t.Run("valid bicep file path", func(t *testing.T) {
		runner := &StaticRunner{
			FilePath: "app.bicep",
		}
		err := runner.Validate(nil, nil)
		require.NoError(t, err)
		// Validate resolves to absolute path
		require.NotEmpty(t, runner.FilePath)
		require.True(t, len(runner.FilePath) > len("app.bicep"))
	})

	t.Run("empty file path", func(t *testing.T) {
		runner := &StaticRunner{
			FilePath: "",
		}
		err := runner.Validate(nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Bicep file path is required")
	})
}

func Test_StaticRunner_Run(t *testing.T) {
	t.Run("valid template with resources and connections", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockBicep := bicep.NewMockInterface(ctrl)
		outputSink := &output.MockOutput{}

		// ARM JSON template with container and redis
		template := map[string]any{
			"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
			"contentVersion": "1.0.0.0",
			"resources": []any{
				map[string]any{
					"type":       "Applications.Core/containers",
					"apiVersion": "2023-10-01-preview",
					"name":       "webapp",
					"properties": map[string]any{
						"container": map[string]any{
							"image": "ghcr.io/radius-project/webapp:latest",
						},
						"connections": map[string]any{
							"redis": map[string]any{
								"source": "[resourceId('Applications.Datastores/redisCaches', 'redis')]",
							},
						},
					},
				},
				map[string]any{
					"type":       "Applications.Datastores/redisCaches",
					"apiVersion": "2023-10-01-preview",
					"name":       "redis",
					"properties": map[string]any{
						"environment": "[resourceId('Applications.Core/environments', 'default')]",
					},
				},
			},
		}

		mockBicep.EXPECT().
			PrepareTemplate(gomock.Any()).
			Return(template, nil).
			Times(1)

		runner := &StaticRunner{
			Output:   outputSink,
			Bicep:    mockBicep,
			FilePath: "/tmp/app.bicep",
		}

		err := runner.Run(context.Background())
		require.NoError(t, err)

		// Verify output was written
		require.Len(t, outputSink.Writes, 1)

		// Parse the JSON output
		logOutput := outputSink.Writes[0].(output.LogOutput)
		params := logOutput.Params
		require.Len(t, params, 1)

		jsonStr, ok := params[0].(string)
		require.True(t, ok, "expected string output, got %T", params[0])

		var graph v20231001preview.AppGraph
		err = json.Unmarshal([]byte(jsonStr), &graph)
		require.NoError(t, err)

		// Verify metadata
		require.NotEmpty(t, graph.Metadata.RadiusCliVersion)
		require.Equal(t, []string{"app.bicep"}, graph.Metadata.SourceFiles)

		// Verify resources
		require.Len(t, graph.Resources, 2)

		// Verify connections (webapp -> redis via property connection)
		require.NotEmpty(t, graph.Connections)

		// Find the connection from webapp to redis using resource IDs
		found := false
		for _, conn := range graph.Connections {
			if strings.Contains(conn.SourceID, "webapp") && strings.Contains(conn.TargetID, "redis") {
				found = true
				require.Equal(t, v20231001preview.ConnectionTypeConnection, conn.Type)
			}
		}
		require.True(t, found, "expected connection from webapp to redis")
	})

	t.Run("bicep compilation failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockBicep := bicep.NewMockInterface(ctrl)
		outputSink := &output.MockOutput{}

		mockBicep.EXPECT().
			PrepareTemplate(gomock.Any()).
			Return(nil, fmt.Errorf("syntax error in Bicep file")).
			Times(1)

		runner := &StaticRunner{
			Output:   outputSink,
			Bicep:    mockBicep,
			FilePath: "/tmp/invalid.bicep",
		}

		err := runner.Run(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to compile Bicep file")
	})

	t.Run("empty template produces empty graph", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockBicep := bicep.NewMockInterface(ctrl)
		outputSink := &output.MockOutput{}

		template := map[string]any{
			"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
			"contentVersion": "1.0.0.0",
			"resources":      []any{},
		}

		mockBicep.EXPECT().
			PrepareTemplate(gomock.Any()).
			Return(template, nil).
			Times(1)

		runner := &StaticRunner{
			Output:   outputSink,
			Bicep:    mockBicep,
			FilePath: "/tmp/empty.bicep",
		}

		err := runner.Run(context.Background())
		require.NoError(t, err)

		// Verify output was written
		require.Len(t, outputSink.Writes, 1)

		logOutput := outputSink.Writes[0].(output.LogOutput)
		params := logOutput.Params
		require.Len(t, params, 1)

		jsonStr, ok := params[0].(string)
		require.True(t, ok)

		var graph v20231001preview.AppGraph
		err = json.Unmarshal([]byte(jsonStr), &graph)
		require.NoError(t, err)
		require.Empty(t, graph.Resources)
		require.Empty(t, graph.Connections)
	})

	t.Run("template with required parameters but none provided", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockBicep := bicep.NewMockInterface(ctrl)
		outputSink := &output.MockOutput{}

		template := map[string]any{
			"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
			"contentVersion": "1.0.0.0",
			"parameters": map[string]any{
				"environmentName": map[string]any{
					"type": "string",
				},
			},
			"resources": []any{},
		}

		mockBicep.EXPECT().
			PrepareTemplate(gomock.Any()).
			Return(template, nil).
			Times(1)

		runner := &StaticRunner{
			Output:   outputSink,
			Bicep:    mockBicep,
			FilePath: "/tmp/parameterized.bicep",
		}

		err := runner.Run(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "environmentName")
	})

	t.Run("map-format resources from extension radius", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockBicep := bicep.NewMockInterface(ctrl)
		outputSink := &output.MockOutput{}

		// ARM JSON template with resources as a map (newer Bicep with extension radius)
		template := map[string]any{
			"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
			"contentVersion": "1.0.0.0",
			"resources": map[string]any{
				"app": map[string]any{
					"import": "radius",
					"type":   "Applications.Core/applications@2023-10-01-preview",
					"properties": map[string]any{
						"name": "myapp",
						"properties": map[string]any{
							"environment": "default",
						},
					},
				},
				"webapp": map[string]any{
					"import": "radius",
					"type":   "Applications.Core/containers@2023-10-01-preview",
					"properties": map[string]any{
						"name": "webapp",
						"properties": map[string]any{
							"application": "[reference('app').id]",
							"container": map[string]any{
								"image": "ghcr.io/radius-project/webapp:latest",
							},
						},
					},
					"dependsOn": []any{"app"},
				},
			},
		}

		mockBicep.EXPECT().
			PrepareTemplate(gomock.Any()).
			Return(template, nil).
			Times(1)

		runner := &StaticRunner{
			Output:   outputSink,
			Bicep:    mockBicep,
			FilePath: "/tmp/app.bicep",
		}

		err := runner.Run(context.Background())
		require.NoError(t, err)

		// Verify output was written
		require.Len(t, outputSink.Writes, 1)

		logOutput := outputSink.Writes[0].(output.LogOutput)
		params := logOutput.Params
		require.Len(t, params, 1)

		jsonStr, ok := params[0].(string)
		require.True(t, ok)

		var graph v20231001preview.AppGraph
		err = json.Unmarshal([]byte(jsonStr), &graph)
		require.NoError(t, err)

		// Verify resources
		require.Len(t, graph.Resources, 2)
	})
}

func Test_IsBicepFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"bicep extension", "app.bicep", true},
		{"bicep with path", "/path/to/app.bicep", true},
		{"relative bicep path", "./app.bicep", true},
		{"uppercase extension", "app.BICEP", true},
		{"no extension", "myapp", false},
		{"json extension", "template.json", false},
		{"empty string", "", false},
		{"bicep in name no extension", "bicep", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsBicepFile(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
