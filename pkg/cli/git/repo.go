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
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepo checks whether the given directory is inside a git repository.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

// RepoRoot returns the root directory of the git repository containing dir.
// Returns empty string and error if not in a git repo.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(out))

	// Resolve symlinks for consistent path comparison (e.g., macOS /tmp -> /private/tmp)
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return root, nil
	}
	return resolved, nil
}

// HeadCommit returns the full SHA of the current HEAD commit.
// Returns empty string if not in a git repo or no commits exist.
func HeadCommit(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RelativePath returns the path of file relative to the git repository root.
// If the file is not within the repo, returns the original path.
func RelativePath(repoRoot, filePath string) string {
	rel, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return filePath
	}
	return rel
}

// IsShallowClone checks whether the current git repo is a shallow clone.
func IsShallowClone(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-shallow-repository")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}
