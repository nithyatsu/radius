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
	"bufio"
	"os"
	"regexp"
	"strings"
)

// resourceDeclPattern matches Bicep resource declarations like:
//
//	resource app 'Applications.Core/applications@2023-10-01-preview' = {
//	resource frontendContainer 'Applications.Core/containers@2023-10-01-preview' = {
//
// Captures the symbolic name (group 1).
var resourceDeclPattern = regexp.MustCompile(`^\s*resource\s+(\w+)\s+'[^']+'\s*`)

// ParseBicepSourceLines reads a .bicep file and returns a map of symbolic resource
// name to the 1-based line number where it is declared. This is used to enrich
// the static app graph with source location info.
//
// The parser scans for lines matching the pattern:
//
//	resource <symbolicName> '<type>@<apiVersion>' = {
//
// Only the first occurrence of each symbolic name is recorded.
func ParseBicepSourceLines(filePath string) (map[string]int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := map[string]int{}
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Quick check before running regex
		if !strings.Contains(line, "resource") {
			continue
		}

		matches := resourceDeclPattern.FindStringSubmatch(line)
		if len(matches) >= 2 {
			symbolicName := matches[1]
			// Only record the first occurrence
			if _, exists := result[symbolicName]; !exists {
				result[symbolicName] = lineNum
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ApplyLineNumbers enriches extracted resources with line numbers from the Bicep
// source file. It matches resources by symbolic name (for map-format ARM JSON)
// or by resource name (for array-format ARM JSON as a fallback).
func ApplyLineNumbers(resources []ExtractedResource, lineMap map[string]int) {
	for i := range resources {
		// Try matching by symbolic name first (map-format ARM JSON)
		if resources[i].SymbolicName != "" {
			if line, ok := lineMap[resources[i].SymbolicName]; ok {
				resources[i].Line = line
				continue
			}
		}

		// Fallback: try matching by resource name (less reliable but works for simple cases)
		if line, ok := lineMap[resources[i].Name]; ok {
			resources[i].Line = line
		}
	}
}
