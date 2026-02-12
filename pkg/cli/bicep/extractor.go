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
	"fmt"
	"strings"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
)

// ExtractedResource holds a parsed ARM JSON resource with its relevant fields.
type ExtractedResource struct {
	// Name is the resource name from the ARM JSON.
	Name string

	// Type is the fully qualified resource type (e.g., "Applications.Core/containers").
	Type string

	// SymbolicName is the Bicep symbolic name (key in map-format ARM JSON).
	// Empty when extracted from array-format ARM JSON.
	SymbolicName string

	// Properties is the raw properties map from the ARM JSON resource.
	Properties map[string]any

	// DependsOn lists the resource names this resource explicitly depends on.
	DependsOn []string

	// Line is the 1-based line number where this resource is defined in the Bicep source.
	// Zero means line info is not available.
	Line int
}

// ExtractResources parses the ARM JSON template returned by PrepareTemplate and
// extracts all resources into a structured list. It handles both array-format
// resources (legacy ARM JSON) and map-format resources (newer Bicep with
// "extension radius" where keys are symbolic names).
func ExtractResources(template map[string]any) ([]ExtractedResource, error) {
	resourcesRaw, ok := template["resources"]
	if !ok {
		return []ExtractedResource{}, nil
	}

	switch resources := resourcesRaw.(type) {
	case []any:
		return extractResourcesFromArray(resources)
	case map[string]any:
		return extractResourcesFromMap(resources)
	default:
		return nil, fmt.Errorf("expected resources to be an array or map, got %T", resourcesRaw)
	}
}

// extractResourcesFromArray handles the legacy ARM JSON format where resources
// is a flat array of resource objects.
func extractResourcesFromArray(resources []any) ([]ExtractedResource, error) {
	var result []ExtractedResource
	for i, r := range resources {
		resourceMap, ok := r.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected resource at index %d to be an object, got %T", i, r)
		}

		extracted, err := extractSingleResource(resourceMap, "")
		if err != nil {
			return nil, fmt.Errorf("failed to extract resource at index %d: %w", i, err)
		}

		result = append(result, extracted)
	}

	return result, nil
}

// extractResourcesFromMap handles the newer ARM JSON format (produced by Bicep
// with "extension radius") where resources is a map keyed by symbolic name.
func extractResourcesFromMap(resources map[string]any) ([]ExtractedResource, error) {
	var result []ExtractedResource
	for symbolicName, r := range resources {
		resourceMap, ok := r.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected resource %q to be an object, got %T", symbolicName, r)
		}

		extracted, err := extractSingleResource(resourceMap, symbolicName)
		if err != nil {
			return nil, fmt.Errorf("failed to extract resource %q: %w", symbolicName, err)
		}

		result = append(result, extracted)
	}

	return result, nil
}

// extractSingleResource parses a single ARM JSON resource object.
// symbolicName is the Bicep symbolic name (from map-format ARM JSON), or empty for array-format.
//
// The function handles two ARM JSON layouts:
//
// Array-format (legacy):
//
//	{"type": "Applications.Core/containers", "name": "webapp", "properties": {...}}
//
// Map-format (extension radius, languageVersion 2.0):
//
//	{"type": "Applications.Core/containers@2023-10-01-preview", "properties": {"name": "webapp", "properties": {...}}}
//
// In the map-format, name is nested inside properties, the type includes the API
// version, and resource-level properties are under properties.properties.
func extractSingleResource(resource map[string]any, symbolicName string) (ExtractedResource, error) {
	resourceType, _ := resource["type"].(string)
	if resourceType == "" {
		return ExtractedResource{}, fmt.Errorf("resource missing 'type' field")
	}

	// Strip API version from type if present (e.g., "Applications.Core/containers@2023-10-01-preview")
	resourceType = stripAPIVersion(resourceType)

	// Look for name at the top level first (array-format)
	name, _ := resource["name"].(string)

	// Get the top-level properties wrapper
	var topProps map[string]any
	if propsRaw, ok := resource["properties"]; ok {
		if propsMap, ok := propsRaw.(map[string]any); ok {
			topProps = propsMap
		}
	}

	// In map-format, name is inside properties
	if name == "" && topProps != nil {
		name, _ = topProps["name"].(string)
	}

	if name == "" {
		return ExtractedResource{}, fmt.Errorf("resource missing 'name' field")
	}

	// Clean up name: remove ARM expression brackets if present
	name = cleanARMExpression(name)

	// Determine the actual resource properties:
	// - Array-format: properties are at resource["properties"]
	// - Map-format: properties are at resource["properties"]["properties"]
	var properties map[string]any
	if topProps != nil {
		if innerProps, ok := topProps["properties"]; ok {
			if innerMap, ok := innerProps.(map[string]any); ok {
				// Map-format: nested properties.properties contains the real resource config
				properties = innerMap
			}
		} else if symbolicName == "" {
			// Array-format: top-level properties is the resource config
			properties = topProps
		}
	}

	var dependsOn []string
	if depsRaw, ok := resource["dependsOn"]; ok {
		if depsSlice, ok := depsRaw.([]any); ok {
			for _, dep := range depsSlice {
				if depStr, ok := dep.(string); ok {
					dependsOn = append(dependsOn, cleanARMExpression(depStr))
				}
			}
		}
	}

	return ExtractedResource{
		Name:         name,
		Type:         resourceType,
		SymbolicName: symbolicName,
		Properties:   properties,
		DependsOn:    dependsOn,
	}, nil
}

// stripAPIVersion removes the @apiVersion suffix from a resource type string.
// For example, "Applications.Core/containers@2023-10-01-preview" becomes
// "Applications.Core/containers".
func stripAPIVersion(resourceType string) string {
	if idx := strings.Index(resourceType, "@"); idx != -1 {
		return resourceType[:idx]
	}
	return resourceType
}

// cleanARMExpression removes surrounding ARM template expression brackets like
// [format('{0}', ...)] and returns a simplified name. For simple string values,
// it returns them as-is.
func cleanARMExpression(value string) string {
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := value[1 : len(value)-1]
		// Try to extract a simple string from format() expressions
		if strings.HasPrefix(inner, "format(") || strings.HasPrefix(inner, "concat(") {
			// Return the expression as-is for display purposes
			return inner
		}
		return inner
	}
	return value
}

// BuildResourceID constructs a Radius resource ID from an extracted resource.
// Format: /planes/radius/local/resourceGroups/default/providers/{type}/{name}
func BuildResourceID(resource ExtractedResource) string {
	return fmt.Sprintf("/planes/radius/local/resourceGroups/default/providers/%s/%s", resource.Type, resource.Name)
}

// ConvertToAppGraphResources converts extracted resources into AppGraphResource types
// for inclusion in the static app graph.
func ConvertToAppGraphResources(extracted []ExtractedResource, sourceFile string) []v20231001preview.AppGraphResource {
	resources := make([]v20231001preview.AppGraphResource, 0, len(extracted))
	for _, e := range extracted {
		line := e.Line
		if line == 0 {
			line = 1 // Default to line 1 when line info is unavailable
		}

		resource := v20231001preview.AppGraphResource{
			ID:   BuildResourceID(e),
			Name: e.Name,
			Type: e.Type,
			SourceLocation: v20231001preview.SourceLocation{
				File: sourceFile,
				Line: line,
			},
			Properties: e.Properties,
		}
		resources = append(resources, resource)
	}
	return resources
}

// BuildExternalPlaceholder creates an AppGraphResource for an external module
// reference that could not be resolved during static analysis.
func BuildExternalPlaceholder(name string, resourceType string, modulePath string) v20231001preview.AppGraphResource {
	return v20231001preview.AppGraphResource{
		ID:   fmt.Sprintf("/planes/radius/local/resourceGroups/default/providers/%s/%s", resourceType, name),
		Name: name,
		Type: resourceType,
		SourceLocation: v20231001preview.SourceLocation{
			File:   modulePath,
			Line:   1,
			Module: modulePath,
		},
	}
}
