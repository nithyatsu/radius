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

package graph

import (
	"testing"

	corerpv20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/radius-project/radius/pkg/to"
	"github.com/stretchr/testify/assert"
)

func TestBuildElements(t *testing.T) {
	t.Parallel()

	artifact := &StaticGraphArtifact{
		Application: corerpv20231001preview.ApplicationGraphResponse{
			Resources: []*corerpv20231001preview.ApplicationGraphResource{
				{
					ID:   to.Ptr("/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend"),
					Name: to.Ptr("frontend"),
					Type: to.Ptr("Applications.Core/containers"),
					Connections: []*corerpv20231001preview.ApplicationGraphConnection{
						{ID: to.Ptr("/planes/radius/local/resourcegroups/default/providers/Applications.Core/redisCaches/cache"), Direction: to.Ptr(corerpv20231001preview.DirectionOutbound)},
					},
				},
				{
					ID:   to.Ptr("/planes/radius/local/resourcegroups/default/providers/Applications.Core/redisCaches/cache"),
					Name: to.Ptr("cache"),
					Type: to.Ptr("Applications.Core/redisCaches"),
					Connections: []*corerpv20231001preview.ApplicationGraphConnection{
						{ID: to.Ptr("/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend"), Direction: to.Ptr(corerpv20231001preview.DirectionInbound)},
					},
				},
			},
		},
	}

	elements := BuildElements(artifact)
	assert.Len(t, elements, 3)

	var nodeCount, edgeCount int
	for _, element := range elements {
		switch element.Group {
		case "nodes":
			nodeCount++
		case "edges":
			edgeCount++
		}
	}

	assert.Equal(t, 2, nodeCount)
	assert.Equal(t, 1, edgeCount)

	edge := elements[2]
	assert.Equal(t, "edges", edge.Group)
	assert.Equal(t, "/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend", edge.Data.Source)
	assert.Equal(t, "/planes/radius/local/resourcegroups/default/providers/Applications.Core/redisCaches/cache", edge.Data.Target)
}

func TestBuildElements_Provenance(t *testing.T) {
	t.Parallel()

	artifact := &StaticGraphArtifact{
		Application: corerpv20231001preview.ApplicationGraphResponse{
			Resources: []*corerpv20231001preview.ApplicationGraphResource{
				{
					ID:                to.Ptr("/planes/radius/local/resourcegroups/default/providers/Applications.Core/containers/frontend"),
					Name:              to.Ptr("frontend"),
					Type:              to.Ptr("Applications.Core/containers"),
					CodeReference:     to.Ptr("test/infra/azure/main.bicep#L42"),
					AppDefinitionLine: to.Ptr(int32(42)),
					DiffHash:          to.Ptr("a3f1c9"),
					OutputResources: []*corerpv20231001preview.ApplicationGraphOutputResource{
						{
							ID:   to.Ptr("/subscriptions/.../deployments/frontend"),
							Name: to.Ptr("frontend"),
							Type: to.Ptr("apps/Deployment"),
						},
					},
				},
			},
		},
	}

	elements := BuildElements(artifact)
	assert.Len(t, elements, 1)

	data := elements[0].Data
	assert.Equal(t, "test/infra/azure/main.bicep#L42", data.CodeReference)
	if assert.NotNil(t, data.AppDefinitionLine) {
		assert.Equal(t, int32(42), *data.AppDefinitionLine)
	}
	assert.Equal(t, "a3f1c9", data.DiffHash)
	assert.Len(t, data.OutputResources, 1)
	assert.Equal(t, "frontend", data.OutputResources[0].Name)
	assert.Equal(t, "apps/Deployment", data.OutputResources[0].Type)
}

func TestBuildGraphArtifact(t *testing.T) {
	t.Parallel()

	artifact := &StaticGraphArtifact{
		Version:     "1.0.0",
		GeneratedAt: "2026-05-20T00:00:00Z",
		SourceFile:  "app.bicep",
		Application: corerpv20231001preview.ApplicationGraphResponse{
			Resources: []*corerpv20231001preview.ApplicationGraphResource{},
		},
	}

	combined := BuildGraphArtifact(artifact)
	assert.Equal(t, artifact.Version, combined.Version)
	assert.Equal(t, artifact.GeneratedAt, combined.GeneratedAt)
	assert.Equal(t, artifact.SourceFile, combined.SourceFile)
	assert.Equal(t, artifact.Application, combined.Application)
	assert.NotNil(t, combined.Elements)
}
