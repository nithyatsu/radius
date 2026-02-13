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

func Test_parseLogOutput_Valid(t *testing.T) {
	input := "abc1234567890123456789012345678901234567\nabc1234\ntest@example.com\n1700000000\nAdd webapp container"

	ci, err := parseLogOutput(input)
	require.NoError(t, err)
	require.Equal(t, "abc1234567890123456789012345678901234567", ci.SHA)
	require.Equal(t, "abc1234", ci.ShortSHA)
	require.Equal(t, "test@example.com", ci.Author)
	require.Equal(t, "Add webapp container", ci.Message)
	require.False(t, ci.Date.IsZero())
}

func Test_parseLogOutput_TooFewLines(t *testing.T) {
	input := "abc1234\ntest@example.com"

	_, err := parseLogOutput(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected 5 lines")
}

func Test_parseLogOutput_InvalidTimestamp(t *testing.T) {
	input := "abc1234567890123456789012345678901234567\nabc1234\ntest@example.com\nnot-a-number\nSome message"

	_, err := parseLogOutput(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse timestamp")
}

func Test_parseLogOutput_MultilineMessage(t *testing.T) {
	// The format uses %s which is first line only, but if newlines sneak through,
	// SplitN with limit 5 keeps the rest of the message in field 5
	input := "abc1234567890123456789012345678901234567\nabc1234\ntest@example.com\n1700000000\nFirst line\nSecond line"

	ci, err := parseLogOutput(input)
	require.NoError(t, err)
	require.Equal(t, "First line\nSecond line", ci.Message)
}
