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
)

func Test_GenerateMarkdown_ContainsHeader(t *testing.T) {
	graph := newTestGraph()

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "# Application Graph")
}

func Test_GenerateMarkdown_ContainsMetadata(t *testing.T) {
	graph := newTestGraph()

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "**CLI Version**: 0.35.0")
	assert.Contains(t, md, "**Source Files**: app.bicep")
}

func Test_GenerateMarkdown_ContainsResourceTable(t *testing.T) {
	graph := newTestGraph()

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "## Resources")
	assert.Contains(t, md, "| Name | Type | Source | Line |")
	assert.Contains(t, md, "| backend |")
	assert.Contains(t, md, "| frontend |")
}

func Test_GenerateMarkdown_ContainsConnectionTable(t *testing.T) {
	graph := newTestGraph()

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "## Connections")
	assert.Contains(t, md, "| Source | Target | Type |")
	assert.Contains(t, md, "| frontend | backend | connection |")
}

func Test_GenerateMarkdown_ContainsMermaidDiagram(t *testing.T) {
	graph := newTestGraph()

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "## Topology")
	assert.Contains(t, md, "```mermaid")
	assert.Contains(t, md, "flowchart TD")
}

func Test_GenerateMarkdown_NoConnectionSection_WhenEmpty(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{ID: "id1", Name: "webapp", Type: "Applications.Core/containers", SourceLocation: v20231001preview.SourceLocation{File: "app.bicep", Line: 10}},
		},
		Connections: []v20231001preview.AppGraphConnectionStatic{},
	}

	md := GenerateMarkdown(graph)

	assert.NotContains(t, md, "## Connections")
}

func Test_GenerateMarkdown_WithGitInfo(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{
				ID:   "id1",
				Name: "webapp",
				Type: "Applications.Core/containers",
				SourceLocation: v20231001preview.SourceLocation{
					File: "app.bicep",
					Line: 10,
				},
				GitInfo: &v20231001preview.GitInfo{
					CommitSHA:   "abc123def456789012345678901234567890abcd",
					CommitShort: "abc123d",
					Author:      "dev@example.com",
					Date:        time.Date(2026, 1, 29, 14, 30, 0, 0, time.UTC),
					Message:     "Add webapp container",
				},
			},
		},
	}

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "| Last Commit | Author |")
	assert.Contains(t, md, "abc123d")
	assert.Contains(t, md, "dev@example.com")
}

func Test_GenerateMarkdown_WithUncommittedChanges(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: newTestMetadata(),
		Resources: []v20231001preview.AppGraphResource{
			{
				ID:   "id1",
				Name: "webapp",
				Type: "Applications.Core/containers",
				SourceLocation: v20231001preview.SourceLocation{
					File: "app.bicep",
					Line: 10,
				},
				GitInfo: &v20231001preview.GitInfo{
					Uncommitted: true,
				},
			},
		},
	}

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "*uncommitted*")
}

func Test_GenerateMarkdown_GitCommitInMetadata(t *testing.T) {
	graph := v20231001preview.AppGraph{
		Metadata: v20231001preview.AppGraphMetadata{
			GeneratedAt:      time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC),
			RadiusCliVersion: "0.35.0",
			SourceFiles:      []string{"app.bicep"},
			SourceHash:       "sha256:abc",
			GitCommit:        "abc123",
		},
		Resources: []v20231001preview.AppGraphResource{},
	}

	md := GenerateMarkdown(graph)

	assert.Contains(t, md, "**Git Commit**: abc123")
}

func Test_GenerateMarkdown_DeterministicOutput(t *testing.T) {
	graph := newTestGraph()

	first := GenerateMarkdown(graph)
	second := GenerateMarkdown(graph)

	assert.Equal(t, first, second, "two calls should produce identical output")
}

func Test_formatCommitLink_TruncatesLongMessages(t *testing.T) {
	longMsg := "This is a very long commit message that exceeds the fifty character limit for display"
	result := formatCommitLink("abc1234", longMsg)

	assert.Contains(t, result, "abc1234")
	assert.Contains(t, result, "...")
	assert.True(t, len(result) < len(longMsg)+20, "result should be shorter than original message")
}

func Test_formatResourceType_WrapsInBackticks(t *testing.T) {
	result := formatResourceType("Applications.Core/containers")
	assert.Equal(t, "`Applications.Core/containers`", result)
}
