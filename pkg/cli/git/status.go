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
	"strings"
)

// UncommittedFiles returns a set of files that have uncommitted changes
// (modified, added, deleted, or untracked). The returned map keys are
// paths relative to the git repository root.
func UncommittedFiles(dir string) (map[string]bool, error) {
	// git status --porcelain returns machine-readable status.
	// Each line is: XY <path>
	// Where X is the index status and Y is the working tree status.
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		// Status is the first 2 chars, then a space, then the path
		path := strings.TrimSpace(line[3:])
		// Handle renamed files: "R  old -> new"
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		if path != "" {
			result[path] = true
		}
	}

	return result, nil
}
