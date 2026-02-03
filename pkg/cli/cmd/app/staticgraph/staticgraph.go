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

package staticgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/radius-project/radius/pkg/cli/bicep"
	"github.com/radius-project/radius/pkg/cli/clierrors"
	"github.com/radius-project/radius/pkg/cli/framework"
	"github.com/radius-project/radius/pkg/cli/output"
	"github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/radius-project/radius/pkg/to"
	"github.com/spf13/cobra"
)

// NewCommand creates an instance of the command and runner for the `rad app staticgraph` command.
func NewCommand(factory framework.Factory) (*cobra.Command, framework.Runner) {
	runner := NewRunner(factory)
	cmd := &cobra.Command{
		Use:   "staticgraph",
		Short: "Generates a static dependency graph from a Bicep file.",
		Long: `Generates a static dependency graph from a Bicep file without deploying.

This command parses a Bicep file and constructs a dependency graph showing
resources and their connections. Each resource type becomes a node, and each
connection becomes an edge to the resource indicated by the source field.

The graph is output in JSON format compatible with the 'rad app graph' output.`,
		Example: `
# Generate a static graph from a Bicep file
rad app staticgraph --file app.bicep

# Generate a static graph from an ARM JSON template
rad app staticgraph --file app.json`,
		Args: cobra.NoArgs,
		RunE: framework.RunCommand(runner),
	}

	cmd.Flags().StringP("file", "f", "", "Path to the Bicep or ARM JSON template file (required)")
	_ = cmd.MarkFlagRequired("file")

	return cmd, runner
}

// Runner is the runner implementation for the `rad app staticgraph` command.
type Runner struct {
	Bicep    bicep.Interface
	Output   output.Interface
	FilePath string
}

// NewRunner creates a new instance of the `rad app staticgraph` runner.
func NewRunner(factory framework.Factory) *Runner {
	return &Runner{
		Bicep:  factory.GetBicep(),
		Output: factory.GetOutput(),
	}
}

// Validate runs validation for the `rad app staticgraph` command.
func (r *Runner) Validate(cmd *cobra.Command, args []string) error {
	filePath, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	if filePath == "" {
		return clierrors.Message("The --file flag is required.")
	}

	r.FilePath = filePath
	return nil
}

// Run runs the `rad app staticgraph` command.
func (r *Runner) Run(ctx context.Context) error {
	// Prepare the template (compile bicep to JSON if necessary)
	template, err := r.Bicep.PrepareTemplate(r.FilePath)
	if err != nil {
		return err
	}

	// Build the static graph from the template
	graph, err := buildStaticGraph(template)
	if err != nil {
		return err
	}

	// Output the graph as JSON
	jsonOutput, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal graph to JSON: %w", err)
	}

	r.Output.LogInfo("%s", string(jsonOutput))
	return nil
}

// resourceInfo holds information about a resource extracted from the template
type resourceInfo struct {
	Name        string
	Type        string
	Connections []connectionInfo
}

// connectionInfo holds information about a connection
type connectionInfo struct {
	Name   string
	Source string
}

// buildStaticGraph constructs a static dependency graph from an ARM template
func buildStaticGraph(template map[string]any) (*v20231001preview.ApplicationGraphResponse, error) {
	resources, err := extractResources(template)
	if err != nil {
		return nil, err
	}

	// Build a map of resource names to their info for lookup
	resourcesByName := make(map[string]*resourceInfo)
	for i := range resources {
		resourcesByName[strings.ToLower(resources[i].Name)] = &resources[i]
	}

	// Build the graph
	graphResources := []*v20231001preview.ApplicationGraphResource{}

	for _, res := range resources {
		graphResource := &v20231001preview.ApplicationGraphResource{
			Name:              to.Ptr(res.Name),
			Type:              to.Ptr(res.Type),
			ProvisioningState: to.Ptr("NotDeployed"),
			Connections:       []*v20231001preview.ApplicationGraphConnection{},
			OutputResources:   []*v20231001preview.ApplicationGraphOutputResource{},
		}

		// Process connections
		for _, conn := range res.Connections {
			targetName, targetType := resolveConnectionTarget(conn.Source, resourcesByName)
			if targetName != "" {
				connection := &v20231001preview.ApplicationGraphConnection{
					ID:        to.Ptr(fmt.Sprintf("%s/%s", targetType, targetName)),
					Direction: to.Ptr(v20231001preview.DirectionOutbound),
				}
				graphResource.Connections = append(graphResource.Connections, connection)
			}
		}

		// Sort connections for stable output
		sort.Slice(graphResource.Connections, func(i, j int) bool {
			return to.String(graphResource.Connections[i].ID) < to.String(graphResource.Connections[j].ID)
		})

		graphResources = append(graphResources, graphResource)
	}

	// Sort resources by type then name for stable output
	sort.Slice(graphResources, func(i, j int) bool {
		if *graphResources[i].Type != *graphResources[j].Type {
			return *graphResources[i].Type < *graphResources[j].Type
		}
		return *graphResources[i].Name < *graphResources[j].Name
	})

	return &v20231001preview.ApplicationGraphResponse{
		Resources: graphResources,
	}, nil
}

// extractResources extracts resource information from an ARM template
func extractResources(template map[string]any) ([]resourceInfo, error) {
	resourcesRaw, ok := template["resources"]
	if !ok {
		return []resourceInfo{}, nil
	}

	var resources []resourceInfo

	// Handle both array format (older ARM) and object format (newer ARM/Bicep languageVersion 2.0)
	switch res := resourcesRaw.(type) {
	case []any:
		// Older ARM template format: resources is an array
		for _, resRaw := range res {
			resMap, ok := resRaw.(map[string]any)
			if !ok {
				continue
			}
			info := extractResourceFromArrayFormat(resMap)
			if info.Name != "" && info.Type != "" {
				resources = append(resources, info)
			}
		}
	case map[string]any:
		// Newer ARM template format (languageVersion 2.0): resources is an object
		for _, resRaw := range res {
			resMap, ok := resRaw.(map[string]any)
			if !ok {
				continue
			}
			info := extractResourceFromObjectFormat(resMap)
			if info.Name != "" && info.Type != "" {
				resources = append(resources, info)
			}
		}
	default:
		return nil, fmt.Errorf("invalid resources format in template")
	}

	return resources, nil
}

// extractResourceFromArrayFormat extracts resource info from older ARM template array format
func extractResourceFromArrayFormat(res map[string]any) resourceInfo {
	info := resourceInfo{}

	// Extract name
	if name, ok := res["name"].(string); ok {
		info.Name = extractNameFromExpression(name)
	}

	// Extract type
	if resType, ok := res["type"].(string); ok {
		info.Type = normalizeResourceType(resType)
	}

	// Extract connections from properties
	if props, ok := res["properties"].(map[string]any); ok {
		info.Connections = extractConnections(props)
	}

	return info
}

// extractResourceFromObjectFormat extracts resource info from newer ARM template object format (languageVersion 2.0)
func extractResourceFromObjectFormat(res map[string]any) resourceInfo {
	info := resourceInfo{}

	// Extract type (in object format, type is at the resource level)
	if resType, ok := res["type"].(string); ok {
		info.Type = normalizeResourceType(resType)
	}

	// In object format, the actual resource properties are nested under "properties"
	if propsWrapper, ok := res["properties"].(map[string]any); ok {
		// Extract name from properties.name
		if name, ok := propsWrapper["name"].(string); ok {
			info.Name = extractNameFromExpression(name)
		}

		// Connections are nested under properties.properties.connections
		if innerProps, ok := propsWrapper["properties"].(map[string]any); ok {
			info.Connections = extractConnections(innerProps)
		}
	}

	return info
}

// extractConnections extracts connection information from a properties map
func extractConnections(props map[string]any) []connectionInfo {
	var connections []connectionInfo

	// Check for connections
	if conns, ok := props["connections"].(map[string]any); ok {
		for connName, connValue := range conns {
			if connMap, ok := connValue.(map[string]any); ok {
				if source, ok := connMap["source"].(string); ok {
					connections = append(connections, connectionInfo{
						Name:   connName,
						Source: source,
					})
				}
			}
		}
	}

	// Also check for routes (used by gateways)
	if routes, ok := props["routes"].([]any); ok {
		for _, routeRaw := range routes {
			if route, ok := routeRaw.(map[string]any); ok {
				if destination, ok := route["destination"].(string); ok {
					connections = append(connections, connectionInfo{
						Name:   "route",
						Source: destination,
					})
				}
			}
		}
	}

	return connections
}

// normalizeResourceType removes API version suffix from resource type if present
func normalizeResourceType(resType string) string {
	// Types may be in format "Applications.Core/containers@2023-10-01-preview"
	// Remove the @version suffix
	if idx := strings.Index(resType, "@"); idx != -1 {
		return resType[:idx]
	}
	return resType
}

// extractNameFromExpression extracts the resource name from an ARM template expression
// ARM templates often have names like "[parameters('name')]" or literal strings
func extractNameFromExpression(expr string) string {
	// If it's a literal string (no brackets), return as-is
	if !strings.HasPrefix(expr, "[") {
		return expr
	}

	// Try to extract name from common patterns
	// Pattern: [format('{0}-{1}', parameters('x'), 'name')]
	// Pattern: ['name']
	// For simplicity, if we can't parse it, return the expression cleaned up
	expr = strings.TrimPrefix(expr, "[")
	expr = strings.TrimSuffix(expr, "]")

	// Check for simple string literal
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		return strings.Trim(expr, "'")
	}

	// For complex expressions, just return a placeholder or the expression itself
	return expr
}

// resolveConnectionTarget resolves the target resource from a connection source
// Returns the target name and type
func resolveConnectionTarget(source string, resourcesByName map[string]*resourceInfo) (string, string) {
	// Check if source is a resource ID (starts with / or contains /providers/)
	if strings.HasPrefix(source, "/") || strings.Contains(source, "/providers/") {
		return extractFromResourceID(source)
	}

	// Check if source is a URL (http:// or https://)
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return extractFromURL(source, resourcesByName)
	}

	// Try parsing as URL without scheme
	if !strings.Contains(source, "//") {
		// Add scheme to help URL parsing
		parsedURL, err := url.Parse("//" + source)
		if err == nil && parsedURL.Hostname() != "" {
			return extractFromURL("http://"+source, resourcesByName)
		}
	}

	return "", ""
}

// extractFromResourceID extracts name and type from a resource ID
func extractFromResourceID(id string) (string, string) {
	// Resource IDs look like:
	// /subscriptions/{sub}/resourceGroups/{rg}/providers/{provider}/{type}/{name}
	// or /planes/radius/local/resourceGroups/{rg}/providers/{provider}/{type}/{name}

	parts := strings.Split(id, "/")

	// Find the providers segment and extract type/name after it
	for i := 0; i < len(parts)-2; i++ {
		if strings.EqualFold(parts[i], "providers") && i+3 <= len(parts) {
			// parts[i+1] is the provider namespace (e.g., Applications.Core)
			// parts[i+2] is the resource type (e.g., containers)
			// parts[i+3] is the resource name
			if i+3 < len(parts) {
				resourceType := parts[i+1] + "/" + parts[i+2]
				resourceName := parts[i+3]
				return resourceName, resourceType
			}
		}
	}

	// Fallback: try to get the last segment as name
	if len(parts) >= 2 {
		name := parts[len(parts)-1]
		resType := parts[len(parts)-2]
		return name, resType
	}

	return "", ""
}

// extractFromURL extracts the target name from a URL source
// and determines the type based on whether a matching resource exists
func extractFromURL(source string, resourcesByName map[string]*resourceInfo) (string, string) {
	parsedURL, err := url.Parse(source)
	if err != nil {
		return "", ""
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", ""
	}

	// Check if there's a resource with this name
	if res, ok := resourcesByName[strings.ToLower(hostname)]; ok {
		return res.Name, res.Type
	}

	// If no matching resource found, use a generic type
	// Check if the hostname looks like a Radius resource name
	// Default to Applications.Core/containers as a common case
	return hostname, "Applications.Core/containers"
}
