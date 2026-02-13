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
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// BlameLine represents the git blame information for a single line.
type BlameLine struct {
	// CommitSHA is the full 40-character commit hash.
	CommitSHA string

	// Author is the commit author email.
	Author string

	// Date is the commit timestamp.
	Date time.Time

	// LineNumber is the 1-based line number in the file.
	LineNumber int
}

// BlameResult contains parsed blame information for a file.
type BlameResult struct {
	// Lines maps 1-based line numbers to their blame information.
	Lines map[int]BlameLine

	// ShallowClone indicates if blame data may be incomplete due to shallow clone.
	ShallowClone bool
}

// BlameFile runs git blame on the specified file and returns parsed results.
// The filePath should be absolute. The dir parameter is the working directory
// for the git command (typically the repo root).
func BlameFile(dir, filePath string) (*BlameResult, error) {
	// Use porcelain format for machine-readable output
	cmd := exec.Command("git", "blame", "--porcelain", filePath)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Check if this is a shallow clone issue
		if IsShallowClone(dir) {
			return &BlameResult{
				Lines:        make(map[int]BlameLine),
				ShallowClone: true,
			}, nil
		}
		return nil, fmt.Errorf("git blame failed for %q: %w", filePath, err)
	}

	return parsePorcelainBlame(string(out))
}

// parsePorcelainBlame parses the output of `git blame --porcelain`.
// Porcelain format:
//
//	<sha1> <orig-line> <final-line> [<num-lines>]
//	header-field: value
//	...
//	\t<content-line>
func parsePorcelainBlame(output string) (*BlameResult, error) {
	result := &BlameResult{
		Lines: make(map[int]BlameLine),
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentSHA string
	var currentLine int
	authors := make(map[string]string)   // sha -> author-mail
	timestamps := make(map[string]int64) // sha -> author-time

	for scanner.Scan() {
		line := scanner.Text()

		// Lines starting with a tab are content lines, marking the end of a block
		if strings.HasPrefix(line, "\t") {
			if currentSHA != "" && currentLine > 0 {
				bl := BlameLine{
					CommitSHA:  currentSHA,
					LineNumber: currentLine,
				}
				if author, ok := authors[currentSHA]; ok {
					bl.Author = author
				}
				if ts, ok := timestamps[currentSHA]; ok {
					bl.Date = time.Unix(ts, 0).UTC()
				}
				result.Lines[currentLine] = bl
			}
			continue
		}

		// Header line: <sha1> <orig-line> <final-line> [<num-lines>]
		parts := strings.Fields(line)
		if len(parts) >= 3 && len(parts[0]) == 40 && isHex(parts[0]) {
			currentSHA = parts[0]
			finalLine, err := strconv.Atoi(parts[2])
			if err == nil {
				currentLine = finalLine
			}
			continue
		}

		// Key-value header fields
		if strings.HasPrefix(line, "author-mail ") {
			email := strings.TrimPrefix(line, "author-mail ")
			email = strings.Trim(email, "<>")
			authors[currentSHA] = email
			continue
		}
		if strings.HasPrefix(line, "author-time ") {
			tsStr := strings.TrimPrefix(line, "author-time ")
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err == nil {
				timestamps[currentSHA] = ts
			}
			continue
		}
	}

	return result, scanner.Err()
}

// isHex checks if a string is valid hexadecimal.
func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
