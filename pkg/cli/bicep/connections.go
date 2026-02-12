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
	"strings"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
)

// DetectConnections inspects extracted resources and identifies connections between
// them based on their properties. It examines:
//   - properties.connections (Radius container connections)
//   - properties.routes (Radius gateway routes)
//   - dependsOn (explicit ARM dependencies)
//
// It returns a slice of AppGraphConnectionStatic representing edges in the graph.
func DetectConnections(resources []ExtractedResource) []v20231001preview.AppGraphConnectionStatic {
	// Build a lookup map from resource name to resource ID for resolving references
	nameToID := make(map[string]string, len(resources))
	typeToResources := make(map[string][]ExtractedResource)
	for _, r := range resources {
		nameToID[r.Name] = BuildResourceID(r)
		typeToResources[r.Type] = append(typeToResources[r.Type], r)
	}

	var connections []v20231001preview.AppGraphConnectionStatic

	for _, resource := range resources {
		sourceID := BuildResourceID(resource)

		// Detect connections from properties.connections
		conns := detectPropertyConnections(sourceID, resource.Properties, nameToID)
		connections = append(connections, conns...)

		// Detect connections from properties.routes (gateways)
		routes := detectRouteConnections(sourceID, resource.Properties, nameToID)
		connections = append(connections, routes...)

		// Detect connections from dependsOn
		deps := detectDependsOnConnections(sourceID, resource.DependsOn, nameToID)
		connections = append(connections, deps...)
	}

	return connections
}

// detectPropertyConnections extracts connections from the properties.connections map.
// In Radius, containers have connections like:
//
//	"connections": {
//	  "redis": { "source": "<resourceID>" },
//	  "backend": { "source": "<resourceID>" }
//	}
func detectPropertyConnections(sourceID string, properties map[string]any, nameToID map[string]string) []v20231001preview.AppGraphConnectionStatic {
	if properties == nil {
		return nil
	}

	connectionsRaw, ok := properties["connections"]
	if !ok {
		return nil
	}

	connectionsMap, ok := connectionsRaw.(map[string]any)
	if !ok {
		return nil
	}

	var connections []v20231001preview.AppGraphConnectionStatic
	for _, connValue := range connectionsMap {
		connMap, ok := connValue.(map[string]any)
		if !ok {
			continue
		}

		targetID := resolveConnectionSource(connMap, nameToID)
		if targetID == "" {
			continue
		}

		connections = append(connections, v20231001preview.AppGraphConnectionStatic{
			SourceID: sourceID,
			TargetID: targetID,
			Type:     v20231001preview.ConnectionTypeConnection,
		})
	}

	return connections
}

// resolveConnectionSource extracts the target resource ID from a connection definition.
// It handles both direct resource IDs and resource name references.
func resolveConnectionSource(connMap map[string]any, nameToID map[string]string) string {
	source, ok := connMap["source"].(string)
	if !ok || source == "" {
		return ""
	}

	// If source is already a full resource ID, use it
	if strings.HasPrefix(source, "/") {
		return source
	}

	// Try to resolve as a resource name reference
	// ARM expressions like "[resourceId('Applications.Datastores/redisCaches', 'cache')]"
	// are resolved by trying to match against known resource names
	resolved := resolveARMResourceReference(source, nameToID)
	if resolved != "" {
		return resolved
	}

	return source
}

// detectRouteConnections extracts connections from properties.routes.
// In Radius, gateways have routes like:
//
//	"routes": {
//	  "frontend": { "destination": "<containerRef>" },
//	}
func detectRouteConnections(sourceID string, properties map[string]any, nameToID map[string]string) []v20231001preview.AppGraphConnectionStatic {
	if properties == nil {
		return nil
	}

	routesRaw, ok := properties["routes"]
	if !ok {
		return nil
	}

	routesMap, ok := routesRaw.(map[string]any)
	if !ok {
		return nil
	}

	var connections []v20231001preview.AppGraphConnectionStatic
	for _, routeValue := range routesMap {
		routeMap, ok := routeValue.(map[string]any)
		if !ok {
			continue
		}

		// Routes have a "destination" property pointing to a container
		destination, ok := routeMap["destination"].(string)
		if !ok || destination == "" {
			continue
		}

		targetID := destination
		if !strings.HasPrefix(destination, "/") {
			resolved := resolveARMResourceReference(destination, nameToID)
			if resolved != "" {
				targetID = resolved
			}
		}

		connections = append(connections, v20231001preview.AppGraphConnectionStatic{
			SourceID: sourceID,
			TargetID: targetID,
			Type:     v20231001preview.ConnectionTypeRoute,
		})
	}

	return connections
}

// detectDependsOnConnections creates connections from explicit dependsOn declarations.
func detectDependsOnConnections(sourceID string, dependsOn []string, nameToID map[string]string) []v20231001preview.AppGraphConnectionStatic {
	var connections []v20231001preview.AppGraphConnectionStatic

	for _, dep := range dependsOn {
		targetID := dep
		if !strings.HasPrefix(dep, "/") {
			// Try to resolve as resource name
			resolved := resolveARMResourceReference(dep, nameToID)
			if resolved != "" {
				targetID = resolved
			}
		}

		connections = append(connections, v20231001preview.AppGraphConnectionStatic{
			SourceID: sourceID,
			TargetID: targetID,
			Type:     v20231001preview.ConnectionTypeDependsOn,
		})
	}

	return connections
}

// resolveARMResourceReference attempts to match an ARM expression or resource name
// against known resources. It handles cases like:
//   - Simple name: "cache" -> lookup in nameToID
//   - resourceId expression: "resourceId('Type', 'name')" -> extract name, lookup
func resolveARMResourceReference(ref string, nameToID map[string]string) string {
	// Direct name lookup
	if id, ok := nameToID[ref]; ok {
		return id
	}

	// Try to extract name from resourceId() expression
	if strings.Contains(ref, "resourceId(") {
		name := extractNameFromResourceID(ref)
		if id, ok := nameToID[name]; ok {
			return id
		}
	}

	// Try to match the last segment of a path reference
	parts := strings.Split(ref, "/")
	lastName := parts[len(parts)-1]
	lastName = strings.Trim(lastName, "'\"")
	if id, ok := nameToID[lastName]; ok {
		return id
	}

	return ""
}

// extractNameFromResourceID extracts the resource name from a resourceId() expression.
// Example: "resourceId('Applications.Datastores/redisCaches', 'cache')" -> "cache"
func extractNameFromResourceID(expr string) string {
	// Find the last quoted string argument
	parts := strings.Split(expr, ",")
	if len(parts) < 2 {
		return ""
	}

	// The last parameter is typically the resource name
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	lastPart = strings.TrimRight(lastPart, ")")
	lastPart = strings.TrimSpace(lastPart)
	lastPart = strings.Trim(lastPart, "'\"")

	return lastPart
}
