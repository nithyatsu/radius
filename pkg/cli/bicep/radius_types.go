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

import "strings"

// radiusPrefixes contains the well-known Radius Bicep extension type prefixes.
var radiusPrefixes = []string{
	"Applications.Core/",
	"Applications.Datastores/",
	"Applications.Messaging/",
	"Applications.Dapr/",
}

// IsRadiusResourceType returns true if the given ARM resource type belongs to the
// Radius Bicep extension (starts with "Applications." prefix).
func IsRadiusResourceType(resourceType string) bool {
	for _, prefix := range radiusPrefixes {
		if strings.HasPrefix(resourceType, prefix) {
			return true
		}
	}

	// Generic fallback: any "Applications.*" type is a Radius type
	return strings.HasPrefix(resourceType, "Applications.")
}

// FilterRadiusResources returns only the resources with Radius-specific types.
func FilterRadiusResources(resources []ExtractedResource) []ExtractedResource {
	var result []ExtractedResource
	for _, r := range resources {
		if IsRadiusResourceType(r.Type) {
			result = append(result, r)
		}
	}
	return result
}

// ResourceCategory returns a human-readable category for the resource type,
// used for graph display and Mermaid diagram shapes.
func ResourceCategory(resourceType string) string {
	lower := strings.ToLower(resourceType)

	switch {
	case strings.Contains(lower, "/containers"):
		return "container"
	case strings.Contains(lower, "/gateways"):
		return "gateway"
	case strings.Contains(lower, "datastores/") || strings.Contains(lower, "databases"):
		return "datastore"
	case strings.Contains(lower, "/secretstores") || strings.Contains(lower, "secrets"):
		return "secret"
	case strings.Contains(lower, "/extenders"):
		return "extender"
	default:
		return "resource"
	}
}
