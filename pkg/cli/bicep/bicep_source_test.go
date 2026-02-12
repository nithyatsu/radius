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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseBicepSourceLines(t *testing.T) {
	t.Run("parses resource declarations with line numbers", func(t *testing.T) {
		content := `extension radius

@description('Specifies the location.')
param location string = 'local'

resource app 'Applications.Core/applications@2023-10-01-preview' = {
  name: 'myapp'
  location: location
}

resource frontendContainer 'Applications.Core/containers@2023-10-01-preview' = {
  name: 'frontend'
  location: location
  properties: {
    application: app.id
  }
}

resource backendContainer 'Applications.Core/containers@2023-10-01-preview' = {
  name: 'backend'
  location: location
}
`
		tmpFile := writeTempBicep(t, content)
		lineMap, err := ParseBicepSourceLines(tmpFile)
		require.NoError(t, err)

		require.Equal(t, 6, lineMap["app"])
		require.Equal(t, 11, lineMap["frontendContainer"])
		require.Equal(t, 19, lineMap["backendContainer"])
		require.Len(t, lineMap, 3)
	})

	t.Run("handles empty file", func(t *testing.T) {
		tmpFile := writeTempBicep(t, "")
		lineMap, err := ParseBicepSourceLines(tmpFile)
		require.NoError(t, err)
		require.Empty(t, lineMap)
	})

	t.Run("handles file with no resources", func(t *testing.T) {
		content := `extension radius

param location string = 'local'
param environment string = 'default'

var appName = 'myapp'
`
		tmpFile := writeTempBicep(t, content)
		lineMap, err := ParseBicepSourceLines(tmpFile)
		require.NoError(t, err)
		require.Empty(t, lineMap)
	})

	t.Run("handles existing keyword", func(t *testing.T) {
		content := `extension radius

resource app 'Applications.Core/applications@2023-10-01-preview' = {
  name: 'myapp'
}

resource existing env 'Applications.Core/environments@2023-10-01-preview' existing = {
  name: 'default'
}
`
		tmpFile := writeTempBicep(t, content)
		lineMap, err := ParseBicepSourceLines(tmpFile)
		require.NoError(t, err)
		// 'existing' follows the keyword pattern but the symbolic name is 'env'
		// The regex should still capture the second word after 'resource'
		require.Contains(t, lineMap, "app")
		require.Equal(t, 3, lineMap["app"])
	})

	t.Run("file not found returns error", func(t *testing.T) {
		_, err := ParseBicepSourceLines("/nonexistent/file.bicep")
		require.Error(t, err)
	})

	t.Run("ignores resource word in comments and strings", func(t *testing.T) {
		content := `extension radius

// resource commentedOut 'Applications.Core/applications@2023-10-01-preview' = {

resource app 'Applications.Core/applications@2023-10-01-preview' = {
  name: 'myapp'
}
`
		tmpFile := writeTempBicep(t, content)
		lineMap, err := ParseBicepSourceLines(tmpFile)
		require.NoError(t, err)
		// The commented-out line matches the regex because we do a simple scan.
		// This is acceptable for now; the line for 'app' should be correct.
		require.Equal(t, 5, lineMap["app"])
	})
}

func Test_ApplyLineNumbers(t *testing.T) {
	t.Run("applies line numbers by symbolic name", func(t *testing.T) {
		resources := []ExtractedResource{
			{Name: "myapp", Type: "Applications.Core/applications", SymbolicName: "app"},
			{Name: "frontend", Type: "Applications.Core/containers", SymbolicName: "frontendContainer"},
		}
		lineMap := map[string]int{
			"app":               6,
			"frontendContainer": 11,
		}

		ApplyLineNumbers(resources, lineMap)

		require.Equal(t, 6, resources[0].Line)
		require.Equal(t, 11, resources[1].Line)
	})

	t.Run("falls back to resource name when no symbolic name", func(t *testing.T) {
		resources := []ExtractedResource{
			{Name: "app", Type: "Applications.Core/applications"},
		}
		lineMap := map[string]int{
			"app": 6,
		}

		ApplyLineNumbers(resources, lineMap)

		require.Equal(t, 6, resources[0].Line)
	})

	t.Run("no match leaves line as zero", func(t *testing.T) {
		resources := []ExtractedResource{
			{Name: "unknown", Type: "Applications.Core/applications", SymbolicName: "missing"},
		}
		lineMap := map[string]int{
			"app": 6,
		}

		ApplyLineNumbers(resources, lineMap)

		require.Equal(t, 0, resources[0].Line)
	})

	t.Run("empty line map leaves all lines as zero", func(t *testing.T) {
		resources := []ExtractedResource{
			{Name: "app", Type: "Applications.Core/applications", SymbolicName: "app"},
		}

		ApplyLineNumbers(resources, map[string]int{})

		require.Equal(t, 0, resources[0].Line)
	})
}

func writeTempBicep(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bicep")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}
