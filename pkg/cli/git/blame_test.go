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
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parsePorcelainBlame_SingleLine(t *testing.T) {
	// Porcelain format for a single line
	input := `abc1234567890123456789012345678901234567 1 1 1
author Test User
author-mail <test@example.com>
author-time 1700000000
author-tz +0000
committer Test User
committer-mail <test@example.com>
committer-time 1700000000
committer-tz +0000
summary Initial commit
filename app.bicep
	resource webapp 'Applications.Core/containers@2023-10-01-preview' = {
`

	result, err := parsePorcelainBlame(input)
	require.NoError(t, err)
	require.Len(t, result.Lines, 1)

	bl := result.Lines[1]
	require.Equal(t, "abc1234567890123456789012345678901234567", bl.CommitSHA)
	require.Equal(t, "test@example.com", bl.Author)
	require.Equal(t, 1, bl.LineNumber)
}

func Test_parsePorcelainBlame_MultipleLines(t *testing.T) {
	input := `aaaa111122223333444455556666777788889999 1 1 2
author Alice
author-mail <alice@example.com>
author-time 1700000000
author-tz +0000
summary Add webapp
filename app.bicep
	resource webapp 'Applications.Core/containers@2023-10-01-preview' = {
aaaa111122223333444455556666777788889999 2 2
	  name: 'webapp'
bbbb111122223333444455556666777788889999 3 3 1
author Bob
author-mail <bob@example.com>
author-time 1700100000
author-tz +0000
summary Add redis
filename app.bicep
	resource redis 'Applications.Datastores/redisCaches@2023-10-01-preview' = {
`

	result, err := parsePorcelainBlame(input)
	require.NoError(t, err)
	require.Len(t, result.Lines, 3)

	// Line 1 and 2 should have Alice's commit
	require.Equal(t, "alice@example.com", result.Lines[1].Author)
	require.Equal(t, "alice@example.com", result.Lines[2].Author)

	// Line 3 should have Bob's commit
	require.Equal(t, "bob@example.com", result.Lines[3].Author)
	require.NotEqual(t, result.Lines[1].CommitSHA, result.Lines[3].CommitSHA)
}

func Test_parsePorcelainBlame_Empty(t *testing.T) {
	result, err := parsePorcelainBlame("")
	require.NoError(t, err)
	require.Empty(t, result.Lines)
}

func Test_isHex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "abc123def", true},
		{"valid uppercase", "ABC123DEF", true},
		{"valid mixed", "aAbBcC123", true},
		{"all digits", "1234567890", true},
		{"invalid char", "xyz123", false},
		{"empty", "", true},
		{"40 char sha", "abc1234567890123456789012345678901234567", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, isHex(tc.input))
		})
	}
}
