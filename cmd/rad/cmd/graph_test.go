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

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/radius-project/radius/pkg/cli/graph"
	corerpv20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/radius-project/radius/pkg/to"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalGraphArtifact_Outputs(t *testing.T) {
	t.Parallel()

	artifact := &graph.StaticGraphArtifact{
		Version:     "1.0.0",
		GeneratedAt: "2026-05-20T00:00:00Z",
		SourceFile:  "app.bicep",
		Application: corerpv20231001preview.ApplicationGraphResponse{
			Resources: []*corerpv20231001preview.ApplicationGraphResource{
				{
					ID:   to.Ptr("/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend"),
					Name: to.Ptr("frontend"),
					Type: to.Ptr("Applications.Core/containers"),
				},
			},
		},
	}

	t.Run("json", func(t *testing.T) {
		t.Parallel()
		data, err := marshalGraphArtifact(graphOutputJSON, artifact)
		require.NoError(t, err)

		var unmarshaled map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		_, hasApplication := unmarshaled["application"]
		assert.True(t, hasApplication)
	})

	t.Run("renderable", func(t *testing.T) {
		t.Parallel()
		data, err := marshalGraphArtifact(graphOutputRenderable, artifact)
		require.NoError(t, err)

		var unmarshaled map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		_, hasElements := unmarshaled["elements"]
		assert.True(t, hasElements)
	})

	t.Run("both", func(t *testing.T) {
		t.Parallel()
		data, err := marshalGraphArtifact(graphOutputBoth, artifact)
		require.NoError(t, err)

		var unmarshaled map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		_, hasApplication := unmarshaled["application"]
		_, hasElements := unmarshaled["elements"]
		assert.True(t, hasApplication)
		assert.True(t, hasElements)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		_, err := marshalGraphArtifact("invalid", artifact)
		require.Error(t, err)
	})
}
