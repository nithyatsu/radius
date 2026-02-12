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

	"github.com/stretchr/testify/require"
)

func Test_ExtractResources_ValidTemplate(t *testing.T) {
	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "frontend",
				"properties": map[string]any{
					"container": map[string]any{
						"image": "myapp/frontend:v1",
					},
				},
			},
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "backend",
				"properties": map[string]any{
					"container": map[string]any{
						"image": "myapp/backend:v1",
					},
					"connections": map[string]any{
						"redis": map[string]any{
							"source": "cache",
						},
					},
				},
			},
			map[string]any{
				"type": "Applications.Datastores/redisCaches",
				"name": "cache",
			},
		},
	}

	resources, err := ExtractResources(template)
	require.NoError(t, err)
	require.Len(t, resources, 3)

	require.Equal(t, "frontend", resources[0].Name)
	require.Equal(t, "Applications.Core/containers", resources[0].Type)
	require.NotNil(t, resources[0].Properties)

	require.Equal(t, "backend", resources[1].Name)
	require.Equal(t, "Applications.Core/containers", resources[1].Type)

	require.Equal(t, "cache", resources[2].Name)
	require.Equal(t, "Applications.Datastores/redisCaches", resources[2].Type)
	require.Nil(t, resources[2].Properties)
}

func Test_ExtractResources_EmptyTemplate(t *testing.T) {
	template := map[string]any{}

	resources, err := ExtractResources(template)
	require.NoError(t, err)
	require.Empty(t, resources)
}

func Test_ExtractResources_NoResources(t *testing.T) {
	template := map[string]any{
		"parameters": map[string]any{},
	}

	resources, err := ExtractResources(template)
	require.NoError(t, err)
	require.Empty(t, resources)
}

func Test_ExtractResources_InvalidResourcesType(t *testing.T) {
	template := map[string]any{
		"resources": "not-an-array",
	}

	_, err := ExtractResources(template)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected resources to be an array or map")
}

func Test_ExtractResources_MissingType(t *testing.T) {
	template := map[string]any{
		"resources": []any{
			map[string]any{
				"name": "frontend",
			},
		},
	}

	_, err := ExtractResources(template)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing 'type' field")
}

func Test_ExtractResources_MissingName(t *testing.T) {
	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
			},
		},
	}

	_, err := ExtractResources(template)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing 'name' field")
}

func Test_ExtractResources_WithDependsOn(t *testing.T) {
	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "backend",
				"dependsOn": []any{
					"cache",
					"database",
				},
			},
			map[string]any{
				"type": "Applications.Datastores/redisCaches",
				"name": "cache",
			},
		},
	}

	resources, err := ExtractResources(template)
	require.NoError(t, err)
	require.Len(t, resources, 2)
	require.Equal(t, []string{"cache", "database"}, resources[0].DependsOn)
}

func Test_ExtractResources_ARMExpressionInName(t *testing.T) {
	template := map[string]any{
		"resources": []any{
			map[string]any{
				"type": "Applications.Core/containers",
				"name": "[format('{0}', 'frontend')]",
			},
		},
	}

	resources, err := ExtractResources(template)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	// ARM expression brackets are removed
	require.Equal(t, "format('{0}', 'frontend')", resources[0].Name)
}

func Test_BuildResourceID(t *testing.T) {
	resource := ExtractedResource{
		Name: "frontend",
		Type: "Applications.Core/containers",
	}

	id := BuildResourceID(resource)
	require.Equal(t, "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/frontend", id)
}

func Test_ConvertToAppGraphResources(t *testing.T) {
	extracted := []ExtractedResource{
		{
			Name:       "frontend",
			Type:       "Applications.Core/containers",
			Properties: map[string]any{"container": map[string]any{"image": "test:v1"}},
		},
		{
			Name: "cache",
			Type: "Applications.Datastores/redisCaches",
		},
	}

	resources := ConvertToAppGraphResources(extracted, "app.bicep")
	require.Len(t, resources, 2)

	require.Equal(t, "frontend", resources[0].Name)
	require.Equal(t, "Applications.Core/containers", resources[0].Type)
	require.Equal(t, "app.bicep", resources[0].SourceLocation.File)
	require.Equal(t, 1, resources[0].SourceLocation.Line)

	require.Equal(t, "cache", resources[1].Name)
	require.Nil(t, resources[1].Properties)
}

func Test_BuildExternalPlaceholder(t *testing.T) {
	resource := BuildExternalPlaceholder("ext-db", "Microsoft.Sql/servers", "modules/db.bicep")

	require.Equal(t, "ext-db", resource.Name)
	require.Equal(t, "Microsoft.Sql/servers", resource.Type)
	require.Equal(t, "modules/db.bicep", resource.SourceLocation.File)
	require.Equal(t, "modules/db.bicep", resource.SourceLocation.Module)
}

func Test_ExtractResources_MapFormat(t *testing.T) {
	// This mirrors the actual ARM JSON output from Bicep with "extension radius"
	// (languageVersion 2.0): name is inside properties, type includes @apiVersion,
	// and resource config is at properties.properties.
	template := map[string]any{
		"resources": map[string]any{
			"app": map[string]any{
				"import": "radius",
				"type":   "Applications.Core/applications@2023-10-01-preview",
				"properties": map[string]any{
					"name": "corerp-application-simple1",
					"properties": map[string]any{
						"environment": "default",
					},
				},
			},
			"frontendContainer": map[string]any{
				"import": "radius",
				"type":   "Applications.Core/containers@2023-10-01-preview",
				"properties": map[string]any{
					"name": "http-front-ctnr-simple1",
					"properties": map[string]any{
						"application": "[reference('app').id]",
						"connections": map[string]any{
							"backend": map[string]any{
								"source": "http://http-back-ctnr-simple1:3000",
							},
						},
					},
				},
				"dependsOn": []any{"app"},
			},
			"backendContainer": map[string]any{
				"import": "radius",
				"type":   "Applications.Core/containers@2023-10-01-preview",
				"properties": map[string]any{
					"name": "http-back-ctnr-simple1",
				},
				"dependsOn": []any{"app"},
			},
		},
	}

	resources, err := ExtractResources(template)
	require.NoError(t, err)
	require.Len(t, resources, 3)

	// Map iteration order is non-deterministic, so look up by symbolic name
	bySymbolic := map[string]ExtractedResource{}
	for _, r := range resources {
		bySymbolic[r.SymbolicName] = r
	}

	app := bySymbolic["app"]
	require.Equal(t, "corerp-application-simple1", app.Name)
	require.Equal(t, "Applications.Core/applications", app.Type) // API version stripped
	require.Equal(t, "app", app.SymbolicName)
	require.NotNil(t, app.Properties)
	require.Equal(t, "default", app.Properties["environment"])

	frontend := bySymbolic["frontendContainer"]
	require.Equal(t, "http-front-ctnr-simple1", frontend.Name)
	require.Equal(t, "Applications.Core/containers", frontend.Type)
	require.Equal(t, "frontendContainer", frontend.SymbolicName)
	require.NotNil(t, frontend.Properties)
	require.Equal(t, []string{"app"}, frontend.DependsOn)

	backend := bySymbolic["backendContainer"]
	require.Equal(t, "http-back-ctnr-simple1", backend.Name)
	require.Equal(t, "Applications.Core/containers", backend.Type)
	require.Equal(t, "backendContainer", backend.SymbolicName)
	require.Nil(t, backend.Properties) // No nested properties.properties
	require.Equal(t, []string{"app"}, backend.DependsOn)
}

func Test_ExtractResources_MapFormat_MissingType(t *testing.T) {
	template := map[string]any{
		"resources": map[string]any{
			"app": map[string]any{
				"properties": map[string]any{
					"name": "myapp",
				},
			},
		},
	}

	_, err := ExtractResources(template)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing 'type' field")
}

func Test_ExtractResources_MapFormat_MissingName(t *testing.T) {
	template := map[string]any{
		"resources": map[string]any{
			"app": map[string]any{
				"type":       "Applications.Core/applications@2023-10-01-preview",
				"properties": map[string]any{
					// No name field
				},
			},
		},
	}

	_, err := ExtractResources(template)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing 'name' field")
}

func Test_ConvertToAppGraphResources_WithLineNumbers(t *testing.T) {
	extracted := []ExtractedResource{
		{
			Name:         "frontend",
			Type:         "Applications.Core/containers",
			SymbolicName: "frontendContainer",
			Line:         24,
		},
		{
			Name: "cache",
			Type: "Applications.Datastores/redisCaches",
			Line: 0, // No line info
		},
	}

	resources := ConvertToAppGraphResources(extracted, "app.bicep")
	require.Len(t, resources, 2)

	require.Equal(t, 24, resources[0].SourceLocation.Line)
	require.Equal(t, "app.bicep", resources[0].SourceLocation.File)

	// Line 0 defaults to 1
	require.Equal(t, 1, resources[1].SourceLocation.Line)
}
