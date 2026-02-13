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

// GenerateMarkdown creates a Markdown document from an AppGraph that includes:
//   - A resource table with name, type, source file, and optional git metadata
//   - An embedded Mermaid diagram for topology visualization
//
// The Markdown is designed to render correctly in GitHub.
func GenerateMarkdown(graph v20231001preview.AppGraph) string {
	var sb strings.Builder

	sb.WriteString("# Application Graph\n\n")

	// Metadata section
	sb.WriteString(fmt.Sprintf("**Generated**: %s  \n", graph.Metadata.GeneratedAt.Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("**CLI Version**: %s  \n", graph.Metadata.RadiusCliVersion))
	if len(graph.Metadata.SourceFiles) > 0 {
		sb.WriteString(fmt.Sprintf("**Source Files**: %s  \n", strings.Join(graph.Metadata.SourceFiles, ", ")))
	}
	if graph.Metadata.GitCommit != "" {
		sb.WriteString(fmt.Sprintf("**Git Commit**: %s  \n", graph.Metadata.GitCommit))
	}
	sb.WriteString("\n")

	// Resource table
	sb.WriteString("## Resources\n\n")
	sb.WriteString(generateResourceTable(graph.Resources))
	sb.WriteString("\n")

	// Connections table (if any)
	if len(graph.Connections) > 0 {
		sb.WriteString("## Connections\n\n")
		sb.WriteString(generateConnectionTable(graph.Connections, graph.Resources))
		sb.WriteString("\n")
	}

	// Mermaid diagram
	sb.WriteString("## Topology\n\n")
	sb.WriteString(GenerateMermaid(graph))
	sb.WriteString("\n")

	return sb.String()
}

// generateResourceTable creates a Markdown table of resources.
func generateResourceTable(resources []v20231001preview.AppGraphResource) string {
	// Sort for deterministic output
	sorted := make([]v20231001preview.AppGraphResource, len(resources))
	copy(sorted, resources)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	// Determine if any resource has git info
	hasGitInfo := false
	for _, r := range sorted {
		if r.GitInfo != nil {
			hasGitInfo = true
			break
		}
	}

	var sb strings.Builder

	// Header row
	if hasGitInfo {
		sb.WriteString("| Name | Type | Source | Line | Last Commit | Author |\n")
		sb.WriteString("|------|------|--------|------|-------------|--------|\n")
	} else {
		sb.WriteString("| Name | Type | Source | Line |\n")
		sb.WriteString("|------|------|--------|------|\n")
	}

	// Data rows
	for _, r := range sorted {
		if hasGitInfo {
			commitInfo := ""
			authorInfo := ""
			if r.GitInfo != nil {
				if r.GitInfo.Uncommitted {
					commitInfo = "*uncommitted*"
					authorInfo = "-"
				} else {
					commitInfo = formatCommitLink(r.GitInfo.CommitShort, r.GitInfo.Message)
					authorInfo = r.GitInfo.Author
				}
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s |\n",
				r.Name, formatResourceType(r.Type), r.SourceLocation.File, r.SourceLocation.Line,
				commitInfo, authorInfo))
		} else {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n",
				r.Name, formatResourceType(r.Type), r.SourceLocation.File, r.SourceLocation.Line))
		}
	}

	return sb.String()
}

// generateConnectionTable creates a Markdown table of connections.
func generateConnectionTable(connections []v20231001preview.AppGraphConnectionStatic, resources []v20231001preview.AppGraphResource) string {
	// Build lookup from ID to name
	idToName := make(map[string]string, len(resources))
	for _, r := range resources {
		idToName[r.ID] = r.Name
	}

	// Sort for deterministic output
	sorted := make([]v20231001preview.AppGraphConnectionStatic, len(connections))
	copy(sorted, connections)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SourceID != sorted[j].SourceID {
			return sorted[i].SourceID < sorted[j].SourceID
		}
		return sorted[i].TargetID < sorted[j].TargetID
	})

	var sb strings.Builder
	sb.WriteString("| Source | Target | Type |\n")
	sb.WriteString("|--------|--------|------|\n")

	for _, c := range sorted {
		srcName := idToName[c.SourceID]
		if srcName == "" {
			srcName = c.SourceID
		}
		tgtName := idToName[c.TargetID]
		if tgtName == "" {
			tgtName = c.TargetID
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", srcName, tgtName, string(c.Type)))
	}

	return sb.String()
}

// formatResourceType shortens a resource type for display by removing the
// provider prefix if it starts with "Applications.".
func formatResourceType(resourceType string) string {
	// Keep the full type for clarity in tables
	return fmt.Sprintf("`%s`", resourceType)
}

// formatCommitLink creates a markdown-formatted commit reference.
// Format: [shortSHA](../../commit/fullSHA) message
func formatCommitLink(commitShort string, message string) string {
	if commitShort == "" {
		return ""
	}

	// Truncate long messages
	truncatedMsg := message
	if len(truncatedMsg) > 50 {
		truncatedMsg = truncatedMsg[:47] + "..."
	}

	return fmt.Sprintf("`%s` %s", commitShort, truncatedMsg)
}
