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
	"path/filepath"

	v20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
)

// EnrichResult contains the results of git metadata enrichment.
type EnrichResult struct {
	// HeadSHA is the current HEAD commit SHA.
	HeadSHA string

	// ShallowClone indicates the repo is a shallow clone with limited history.
	ShallowClone bool
}

// EnrichResources populates GitInfo on each resource using git blame and status.
// The bicepFilePath should be an absolute path to the Bicep source file.
// Resources must have SourceLocation.Line set for blame matching.
//
// This function degrades gracefully:
//   - Non-git directories: returns empty result without error
//   - Shallow clones: marks result as ShallowClone, skips blame
//   - Uncommitted files: sets GitInfo.Uncommitted = true
//   - Blame failures: skips individual resources silently
func EnrichResources(resources []v20231001preview.AppGraphResource, bicepFilePath string) (*EnrichResult, error) {
	dir := filepath.Dir(bicepFilePath)

	// Check if we're in a git repo
	if !IsGitRepo(dir) {
		return &EnrichResult{}, nil
	}

	repoRoot, err := RepoRoot(dir)
	if err != nil {
		return &EnrichResult{}, nil
	}

	// Resolve symlinks in the bicep file path for consistent comparison
	// (e.g., macOS /tmp -> /private/tmp)
	resolvedBicep, err := filepath.EvalSymlinks(bicepFilePath)
	if err != nil {
		resolvedBicep = bicepFilePath
	}

	result := &EnrichResult{
		HeadSHA: HeadCommit(dir),
	}

	// Check for shallow clone
	if IsShallowClone(dir) {
		result.ShallowClone = true
		// In shallow clones, blame data may be unreliable, but we can still
		// try to get basic info. We'll mark the result but continue.
	}

	// Get uncommitted files
	uncommitted, err := UncommittedFiles(repoRoot)
	if err != nil {
		// Non-fatal: continue without uncommitted detection
		uncommitted = make(map[string]bool)
	}

	// Get blame data for the Bicep file
	blameResult, err := BlameFile(repoRoot, resolvedBicep)
	if err != nil {
		// Non-fatal: continue without blame data
		blameResult = &BlameResult{Lines: make(map[int]BlameLine)}
	}
	if blameResult.ShallowClone {
		result.ShallowClone = true
	}

	// Cache commit info to avoid repeated git log calls for the same SHA
	commitCache := make(map[string]*CommitInfo)

	// Enrich each resource
	relFile := RelativePath(repoRoot, resolvedBicep)
	for i := range resources {
		res := &resources[i]
		line := res.SourceLocation.Line

		// Check if the source file has uncommitted changes
		isUncommitted := uncommitted[relFile]

		// Look up blame info for this resource's line
		if line > 0 {
			if bl, ok := blameResult.Lines[line]; ok {
				// Get full commit info (cached)
				ci, ok := commitCache[bl.CommitSHA]
				if !ok {
					ci, err = LogForCommit(repoRoot, bl.CommitSHA)
					if err != nil {
						// If log fails, build partial info from blame
						ci = &CommitInfo{
							SHA:      bl.CommitSHA,
							ShortSHA: shortSHA(bl.CommitSHA),
							Author:   bl.Author,
							Date:     bl.Date,
						}
					}
					commitCache[bl.CommitSHA] = ci
				}

				res.GitInfo = &v20231001preview.GitInfo{
					CommitSHA:   ci.SHA,
					CommitShort: ci.ShortSHA,
					Author:      ci.Author,
					Date:        ci.Date,
					Message:     ci.Message,
					Uncommitted: isUncommitted,
				}
				continue
			}
		}

		// No blame data for this line: resource might be new/uncommitted
		if isUncommitted {
			res.GitInfo = &v20231001preview.GitInfo{
				Uncommitted: true,
			}
		}
	}

	return result, nil
}

// shortSHA returns the first 7 characters of a SHA.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
