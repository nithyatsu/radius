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
	"fmt"
	"sort"
	"strings"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
)

// GenerateMermaid creates a Mermaid flowchart diagram from an AppGraph.
// Resource types determine the node shape:
//   - Containers use rectangles: [name]
//   - Gateways use diamonds: {name}
//   - Databases/data stores use cylinders: [(name)]
//   - All other types use rounded rectangles: (name)
//
// Connections are rendered as directed edges with labels indicating the
// connection type.
func GenerateMermaid(graph v20231001preview.AppGraph) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\nflowchart TD\n")

	// Build a map from resource ID to a sanitized Mermaid node ID
	nodeIDs := make(map[string]string, len(graph.Resources))

	// Sort resources for deterministic output
	resources := make([]v20231001preview.AppGraphResource, len(graph.Resources))
	copy(resources, graph.Resources)
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ID < resources[j].ID
	})

	for i, r := range resources {
		nodeID := fmt.Sprintf("n%d", i)
		nodeIDs[r.ID] = nodeID

		shape := mermaidShape(r.Name, r.Type)
		sb.WriteString(fmt.Sprintf("    %s%s\n", nodeID, shape))
	}

	// Sort connections for deterministic output
	connections := make([]v20231001preview.AppGraphConnectionStatic, len(graph.Connections))
	copy(connections, graph.Connections)
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].SourceID != connections[j].SourceID {
			return connections[i].SourceID < connections[j].SourceID
		}
		if connections[i].TargetID != connections[j].TargetID {
			return connections[i].TargetID < connections[j].TargetID
		}
		return connections[i].Type < connections[j].Type
	})

	for _, c := range connections {
		srcNode, srcOK := nodeIDs[c.SourceID]
		tgtNode, tgtOK := nodeIDs[c.TargetID]
		if !srcOK || !tgtOK {
			continue
		}

		label := mermaidEdgeLabel(c.Type)
		if label != "" {
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", srcNode, label, tgtNode))
		} else {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", srcNode, tgtNode))
		}
	}

	sb.WriteString("```")
	return sb.String()
}

// mermaidShape returns the Mermaid node definition with appropriate shape for
// the resource type.
func mermaidShape(name string, resourceType string) string {
	sanitized := sanitizeMermaidLabel(name)
	typeLower := strings.ToLower(resourceType)

	switch {
	case strings.Contains(typeLower, "/gateways") || strings.Contains(typeLower, "/httpRoutes"):
		// Gateways use diamond shape
		return fmt.Sprintf("{%s}", sanitized)
	case isDataStoreType(typeLower):
		// Databases/data stores use cylinder shape
		return fmt.Sprintf("[(%s)]", sanitized)
	case strings.Contains(typeLower, "/containers"):
		// Containers use rectangle shape
		return fmt.Sprintf("[%s]", sanitized)
	default:
		// Default: rounded rectangle
		return fmt.Sprintf("(%s)", sanitized)
	}
}

// isDataStoreType returns true if the resource type represents a data store or database.
func isDataStoreType(typeLower string) bool {
	dataStorePatterns := []string{
		"/rediscaches",
		"/mongodatabases",
		"/sqldatabases",
		"/mysqldatabases",
		"/postgresqldatabases",
		"/rabbitmqqueues",
		"datastores",
		"/databases",
		"storage",
	}
	for _, pattern := range dataStorePatterns {
		if strings.Contains(typeLower, pattern) {
			return true
		}
	}
	return false
}

// mermaidEdgeLabel returns a display label for the connection type.
func mermaidEdgeLabel(connType v20231001preview.ConnectionType) string {
	switch connType {
	case v20231001preview.ConnectionTypeRoute:
		return "route"
	case v20231001preview.ConnectionTypeDependsOn:
		return "dependsOn"
	default:
		return ""
	}
}

// sanitizeMermaidLabel escapes characters that have special meaning in Mermaid syntax.
func sanitizeMermaidLabel(label string) string {
	// Replace characters that could break Mermaid syntax
	replacer := strings.NewReplacer(
		"[", "&#91;",
		"]", "&#93;",
		"{", "&#123;",
		"}", "&#125;",
		"(", "&#40;",
		")", "&#41;",
		"|", "&#124;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(label)
}
