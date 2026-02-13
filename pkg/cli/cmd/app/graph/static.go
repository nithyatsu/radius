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
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/radius-project/radius/pkg/cli/bicep"
	"github.com/radius-project/radius/pkg/cli/git"
	"github.com/radius-project/radius/pkg/cli/output"
	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/radius-project/radius/pkg/version"
	"github.com/spf13/cobra"
)

// StaticRunner is the runner for generating an app graph from Bicep files
// without deployment. It implements the framework.Runner interface.
type StaticRunner struct {
	Output output.Interface
	Bicep  bicep.Interface

	// FilePath is the path to the Bicep file to analyze.
	FilePath string

	// ParameterFile is the optional path to a parameter file.
	ParameterFile string

	// Stdout controls whether output goes to stdout instead of a file.
	Stdout bool

	// OutputPath is an optional custom file path for JSON output.
	OutputPath string

	// Format controls additional output formats (e.g., "markdown").
	Format string

	// NoGit disables git metadata enrichment when true.
	NoGit bool
}

// Validate checks that the Bicep file path is valid and accessible.
func (r *StaticRunner) Validate(cmd *cobra.Command, args []string) error {
	if r.FilePath == "" {
		return fmt.Errorf("Bicep file path is required")
	}

	// Resolve to absolute path for deterministic behavior
	absPath, err := filepath.Abs(r.FilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve file path %q: %w", r.FilePath, err)
	}
	r.FilePath = absPath

	return nil
}

// Run executes the static graph generation pipeline:
//  1. Compile Bicep file to ARM JSON via PrepareTemplate
//  2. Validate required parameters
//  3. Extract resources from ARM JSON
//  4. Detect connections between resources
//  5. Compute source hash for staleness detection
//  6. Build the AppGraph structure
//  7. Write output (JSON file, stdout, or Markdown)
func (r *StaticRunner) Run(ctx context.Context) error {
	// Step 1: Compile Bicep to ARM JSON
	template, err := r.Bicep.PrepareTemplate(r.FilePath)
	if err != nil {
		return fmt.Errorf("failed to compile Bicep file: %w", err)
	}

	// Step 2: Validate required parameters
	var providedParams map[string]any
	if r.ParameterFile != "" {
		paramTemplate, err := bicep.ReadARMJSON(r.ParameterFile)
		if err != nil {
			return fmt.Errorf("failed to read parameter file %q: %w", r.ParameterFile, err)
		}
		// Parameter files can have a "parameters" wrapper or be flat
		if params, ok := paramTemplate["parameters"]; ok {
			if paramsMap, ok := params.(map[string]any); ok {
				providedParams = paramsMap
			}
		} else {
			providedParams = paramTemplate
		}
	}

	if err := bicep.ValidateRequiredParameters(template, providedParams); err != nil {
		return err
	}

	// Step 3: Extract resources from ARM JSON
	extracted, err := bicep.ExtractResources(template)
	if err != nil {
		return fmt.Errorf("failed to extract resources: %w", err)
	}

	// Step 3b: Parse Bicep source for resource line numbers
	lineMap, err := bicep.ParseBicepSourceLines(r.FilePath)
	if err == nil && len(lineMap) > 0 {
		bicep.ApplyLineNumbers(extracted, lineMap)
	}
	// Non-fatal: continue without line numbers if parsing fails

	// Step 4: Detect connections
	connections := bicep.DetectConnections(extracted)

	// Step 5: Compute source hash
	sourceFile := filepath.Base(r.FilePath)
	sourceFiles := []string{r.FilePath}
	sourceHash, err := bicep.ComputeSourceHash(sourceFiles)
	if err != nil {
		// Non-fatal: continue without hash
		sourceHash = ""
	}

	// Step 6: Build AppGraph
	appGraph := v20231001preview.AppGraph{
		Metadata: v20231001preview.AppGraphMetadata{
			GeneratedAt:      time.Now().UTC(),
			RadiusCliVersion: version.Version(),
			SourceFiles:      []string{sourceFile},
			SourceHash:       sourceHash,
		},
		Resources:   bicep.ConvertToAppGraphResources(extracted, sourceFile),
		Connections: connections,
	}

	// Step 6b: Enrich with git metadata (unless --no-git)
	if !r.NoGit {
		enrichResult, err := git.EnrichResources(appGraph.Resources, r.FilePath)
		if err == nil && enrichResult.HeadSHA != "" {
			appGraph.Metadata.GitCommit = enrichResult.HeadSHA
		}
		// Non-fatal: continue even if enrichment fails
	}

	// Step 7: Write output (file or stdout)
	config := OutputConfig{
		Stdout:        r.Stdout,
		OutputPath:    r.OutputPath,
		Format:        r.Format,
		BicepFilePath: r.FilePath,
	}

	_, err = WriteGraphOutput(appGraph, config, r.Output)
	return err
}
