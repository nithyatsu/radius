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

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/stretchr/testify/require"
)

func Test_EnrichResources_NonGitDirectory(t *testing.T) {
	// Use a temp dir that is not a git repo
	tmpDir := t.TempDir()

	resources := []v20231001preview.AppGraphResource{
		{
			ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/webapp",
			Name: "webapp",
			Type: "Applications.Core/containers",
			SourceLocation: v20231001preview.SourceLocation{
				File: "app.bicep",
				Line: 5,
			},
		},
	}

	bicepFile := filepath.Join(tmpDir, "app.bicep")
	err := os.WriteFile(bicepFile, []byte("// test"), 0644)
	require.NoError(t, err)

	result, err := EnrichResources(resources, bicepFile)
	require.NoError(t, err)
	require.Empty(t, result.HeadSHA)

	// Resources should not have git info
	require.Nil(t, resources[0].GitInfo)
}

func Test_EnrichResources_WithGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temp git repo with a committed Bicep file
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	bicepContent := `resource webapp 'Applications.Core/containers@2023-10-01-preview' = {
  name: 'webapp'
  properties: {
    container: {
      image: 'nginx:latest'
    }
  }
}

resource redis 'Applications.Datastores/redisCaches@2023-10-01-preview' = {
  name: 'redis'
  properties: {}
}
`
	bicepFile := filepath.Join(tmpDir, "app.bicep")
	err := os.WriteFile(bicepFile, []byte(bicepContent), 0644)
	require.NoError(t, err)

	runGit(t, tmpDir, "add", "app.bicep")
	runGit(t, tmpDir, "commit", "-m", "Add app resources")

	resources := []v20231001preview.AppGraphResource{
		{
			ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/webapp",
			Name: "webapp",
			Type: "Applications.Core/containers",
			SourceLocation: v20231001preview.SourceLocation{
				File: "app.bicep",
				Line: 1,
			},
		},
		{
			ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Datastores/redisCaches/redis",
			Name: "redis",
			Type: "Applications.Datastores/redisCaches",
			SourceLocation: v20231001preview.SourceLocation{
				File: "app.bicep",
				Line: 10,
			},
		},
	}

	result, err := EnrichResources(resources, bicepFile)
	require.NoError(t, err)
	require.NotEmpty(t, result.HeadSHA)
	require.False(t, result.ShallowClone)

	// Both resources should have git info
	require.NotNil(t, resources[0].GitInfo)
	require.NotEmpty(t, resources[0].GitInfo.CommitSHA)
	require.NotEmpty(t, resources[0].GitInfo.CommitShort)
	require.Equal(t, "test@example.com", resources[0].GitInfo.Author)
	require.Equal(t, "Add app resources", resources[0].GitInfo.Message)
	require.False(t, resources[0].GitInfo.Uncommitted)

	require.NotNil(t, resources[1].GitInfo)
	require.Equal(t, resources[0].GitInfo.CommitSHA, resources[1].GitInfo.CommitSHA)
}

func Test_EnrichResources_UncommittedChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	bicepFile := filepath.Join(tmpDir, "app.bicep")
	err := os.WriteFile(bicepFile, []byte("resource webapp 'Applications.Core/containers@2023-10-01-preview' = {\n  name: 'webapp'\n}\n"), 0644)
	require.NoError(t, err)

	runGit(t, tmpDir, "add", "app.bicep")
	runGit(t, tmpDir, "commit", "-m", "Initial commit")

	// Modify the file to create uncommitted changes
	err = os.WriteFile(bicepFile, []byte("resource webapp 'Applications.Core/containers@2023-10-01-preview' = {\n  name: 'webapp-modified'\n}\n"), 0644)
	require.NoError(t, err)

	resources := []v20231001preview.AppGraphResource{
		{
			ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/webapp",
			Name: "webapp",
			Type: "Applications.Core/containers",
			SourceLocation: v20231001preview.SourceLocation{
				File: "app.bicep",
				Line: 1,
			},
		},
	}

	result, err := EnrichResources(resources, bicepFile)
	require.NoError(t, err)
	require.NotEmpty(t, result.HeadSHA)

	// Resource should have git info with uncommitted flag
	require.NotNil(t, resources[0].GitInfo)
	require.True(t, resources[0].GitInfo.Uncommitted)
}

func Test_EnrichResources_NoLineNumber(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	bicepFile := filepath.Join(tmpDir, "app.bicep")
	err := os.WriteFile(bicepFile, []byte("resource webapp 'Applications.Core/containers@2023-10-01-preview' = {}\n"), 0644)
	require.NoError(t, err)

	runGit(t, tmpDir, "add", "app.bicep")
	runGit(t, tmpDir, "commit", "-m", "Initial")

	resources := []v20231001preview.AppGraphResource{
		{
			ID:   "/planes/radius/local/resourceGroups/default/providers/Applications.Core/containers/webapp",
			Name: "webapp",
			Type: "Applications.Core/containers",
			SourceLocation: v20231001preview.SourceLocation{
				File: "app.bicep",
				Line: 0, // No line number
			},
		},
	}

	result, err := EnrichResources(resources, bicepFile)
	require.NoError(t, err)
	require.NotEmpty(t, result.HeadSHA)

	// No line number means no blame match, and file is committed so no uncommitted flag
	require.Nil(t, resources[0].GitInfo)
}

func Test_shortSHA(t *testing.T) {
	require.Equal(t, "abc1234", shortSHA("abc123456789"))
	require.Equal(t, "abc", shortSHA("abc"))
	require.Equal(t, "", shortSHA(""))
}

// runGit is a test helper that executes a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, string(out))
	}
}
