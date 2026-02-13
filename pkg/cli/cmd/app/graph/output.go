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
	"fmt"
	"os"
	"path/filepath"

	"github.com/radius-project/radius/pkg/cli/output"
	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
)

const (
	// defaultOutputDir is the directory name where graph artifacts are written.
	defaultOutputDir = ".radius"

	// defaultJSONFilename is the default JSON output filename.
	defaultJSONFilename = "app-graph.json"

	// defaultMarkdownFilename is the default Markdown output filename.
	defaultMarkdownFilename = "app-graph.md"
)

// OutputConfig controls where and how the app graph is written.
type OutputConfig struct {
	// Stdout indicates the graph should be written to stdout only (no file).
	Stdout bool

	// OutputPath is a custom file path for JSON output. If empty, the default
	// .radius/app-graph.json is used.
	OutputPath string

	// Format controls additional output formats. Valid values: "", "markdown".
	Format string

	// BicepFilePath is the path to the input Bicep file, used to determine
	// the default output directory.
	BicepFilePath string
}

// DefaultJSONPath returns the default output path for the app graph JSON file.
// The path is .radius/app-graph.json relative to the Bicep file's directory.
func DefaultJSONPath(bicepFilePath string) string {
	dir := filepath.Dir(bicepFilePath)
	return filepath.Join(dir, defaultOutputDir, defaultJSONFilename)
}

// DefaultMarkdownPath returns the default output path for the app graph Markdown file.
// The path is .radius/app-graph.md relative to the Bicep file's directory.
func DefaultMarkdownPath(bicepFilePath string) string {
	dir := filepath.Dir(bicepFilePath)
	return filepath.Join(dir, defaultOutputDir, defaultMarkdownFilename)
}

// WriteGraphOutput writes the app graph to the configured output destinations.
// It returns the paths of files written (empty if stdout-only).
func WriteGraphOutput(graph v20231001preview.AppGraph, config OutputConfig, out output.Interface) ([]string, error) {
	// Always generate deterministic JSON
	jsonBytes, err := output.MarshalDeterministicJSON(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize app graph: %w", err)
	}

	// Stdout mode: write JSON to stdout and return
	if config.Stdout {
		out.LogInfo("%s", string(jsonBytes))
		return nil, nil
	}

	// Determine JSON output path
	jsonPath := config.OutputPath
	if jsonPath == "" {
		jsonPath = DefaultJSONPath(config.BicepFilePath)
	}

	// Ensure output directory exists
	dir := filepath.Dir(jsonPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %q: %w", dir, err)
	}

	// Write JSON file
	if err := os.WriteFile(jsonPath, jsonBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write JSON to %q: %w", jsonPath, err)
	}

	writtenFiles := []string{jsonPath}
	out.LogInfo("Wrote app graph to %s", jsonPath)

	// Write Markdown if requested
	if config.Format == "markdown" {
		mdContent := output.GenerateMarkdown(graph)
		mdPath := DefaultMarkdownPath(config.BicepFilePath)
		if config.OutputPath != "" {
			// If custom output path specified, put markdown alongside it
			mdDir := filepath.Dir(config.OutputPath)
			mdPath = filepath.Join(mdDir, defaultMarkdownFilename)
		}

		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			return writtenFiles, fmt.Errorf("failed to write Markdown to %q: %w", mdPath, err)
		}

		writtenFiles = append(writtenFiles, mdPath)
		out.LogInfo("Wrote app graph Markdown to %s", mdPath)
	}

	return writtenFiles, nil
}
