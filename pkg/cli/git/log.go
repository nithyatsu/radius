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
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CommitInfo contains metadata about a single git commit.
type CommitInfo struct {
	// SHA is the full commit hash.
	SHA string

	// ShortSHA is the abbreviated commit hash (first 7 characters).
	ShortSHA string

	// Author is the commit author email.
	Author string

	// Date is the commit timestamp.
	Date time.Time

	// Message is the first line of the commit message.
	Message string
}

// LogForCommit retrieves commit metadata for a specific commit SHA.
// If sha is empty, retrieves the HEAD commit.
func LogForCommit(dir, sha string) (*CommitInfo, error) {
	args := []string{"log", "-1", "--format=%H%n%h%n%ae%n%at%n%s"}
	if sha != "" {
		args = append(args, sha)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return parseLogOutput(strings.TrimSpace(string(out)))
}

// parseLogOutput parses the output of `git log -1 --format=%H%n%h%n%ae%n%at%n%s`.
func parseLogOutput(output string) (*CommitInfo, error) {
	lines := strings.SplitN(output, "\n", 5)
	if len(lines) < 5 {
		return nil, fmt.Errorf("unexpected git log output: expected 5 lines, got %d", len(lines))
	}

	ts, err := strconv.ParseInt(lines[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %q: %w", lines[3], err)
	}

	return &CommitInfo{
		SHA:      lines[0],
		ShortSHA: lines[1],
		Author:   lines[2],
		Date:     time.Unix(ts, 0).UTC(),
		Message:  lines[4],
	}, nil
}
