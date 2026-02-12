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
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ComputeSourceHash computes a deterministic SHA256 hash of one or more source files.
// Files are sorted by path for determinism. The returned string has the format
// "sha256:<hex>".
func ComputeSourceHash(filePaths []string) (string, error) {
	if len(filePaths) == 0 {
		return "", fmt.Errorf("at least one file path is required")
	}

	// Sort paths for deterministic hashing
	sorted := make([]string, len(filePaths))
	copy(sorted, filePaths)
	sort.Strings(sorted)

	h := sha256.New()

	for _, fp := range sorted {
		content, err := os.ReadFile(fp)
		if err != nil {
			return "", fmt.Errorf("failed to read file %q for hashing: %w", fp, err)
		}

		// Include the file path in the hash so renames are detected
		fmt.Fprintf(h, "file:%s\n", fp)

		// Normalize line endings for cross-platform determinism
		normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
		h.Write([]byte(normalized))
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
