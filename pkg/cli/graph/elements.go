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
	corerpv20231001preview "github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/radius-project/radius/pkg/to"
)

// Element is a node or edge in the graph serialization format.
type Element struct {
	Group string      `json:"group"`
	Data  ElementData `json:"data"`
}

// ElementData holds node/edge properties.
type ElementData struct {
	ID       string `json:"id"`
	Resource string `json:"resourceId,omitempty"`
	Label    string `json:"label,omitempty"`
	Source   string `json:"source,omitempty"`
	Target   string `json:"target,omitempty"`

	// Provenance fields (nodes only). Populated when the upstream domain
	// model carries them; absent otherwise.
	CodeReference     string                  `json:"codeReference,omitempty"`
	AppDefinitionLine *int32                  `json:"appDefinitionLine,omitempty"`
	DiffHash          string                  `json:"diffHash,omitempty"`
	OutputResources   []ElementOutputResource `json:"outputResources,omitempty"`
}

// ElementOutputResource is a flattened view of an output resource backing a
// graph node. This is what a `rad deploy --dry-run` would populate.
type ElementOutputResource struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// GraphArtifact is the combined artifact containing both the application
// schema and the flat element graph.
type GraphArtifact struct {
	Version     string                                          `json:"version"`
	GeneratedAt string                                          `json:"generatedAt"`
	SourceFile  string                                          `json:"sourceFile"`
	Application corerpv20231001preview.ApplicationGraphResponse `json:"application"`
	Elements    []Element                                       `json:"elements"`
}

// BuildElements converts the static graph resources into a flat list of
// node and edge elements.
func BuildElements(artifact *StaticGraphArtifact) []Element {
	if artifact == nil {
		return nil
	}

	resources := artifact.Application.Resources
	elements := make([]Element, 0, len(resources))

	resourceByID := make(map[string]*corerpv20231001preview.ApplicationGraphResource, len(resources))
	for _, resource := range resources {
		resourceByID[to.String(resource.ID)] = resource
	}

	for _, resource := range resources {
		resourceID := to.String(resource.ID)
		resourceType := to.String(resource.Type)
		shortType := resourceType
		if idx := lastSlash(resourceType); idx >= 0 {
			shortType = resourceType[idx+1:]
		}

		data := ElementData{
			ID:                resourceID,
			Resource:          resourceID,
			Label:             to.String(resource.Name) + "\n" + shortType,
			CodeReference:     to.String(resource.CodeReference),
			AppDefinitionLine: resource.AppDefinitionLine,
			DiffHash:          to.String(resource.DiffHash),
			OutputResources:   convertOutputResources(resource.OutputResources),
		}

		elements = append(elements, Element{
			Group: "nodes",
			Data:  data,
		})
	}

	edgeIDs := map[string]struct{}{}
	for _, resource := range resources {
		sourceID := to.String(resource.ID)
		for _, connection := range resource.Connections {
			if connection.Direction == nil || *connection.Direction != corerpv20231001preview.DirectionOutbound {
				continue
			}

			targetID := to.String(connection.ID)
			if _, ok := resourceByID[targetID]; !ok {
				continue
			}

			edgeID := sourceID + "-->" + targetID
			if _, ok := edgeIDs[edgeID]; ok {
				continue
			}
			edgeIDs[edgeID] = struct{}{}

			elements = append(elements, Element{
				Group: "edges",
				Data: ElementData{
					ID:     edgeID,
					Source: sourceID,
					Target: targetID,
				},
			})
		}
	}

	return elements
}

// BuildGraphArtifact returns the combined artifact with both the application
// schema and the flat element graph.
func BuildGraphArtifact(artifact *StaticGraphArtifact) GraphArtifact {
	if artifact == nil {
		return GraphArtifact{}
	}

	return GraphArtifact{
		Version:     artifact.Version,
		GeneratedAt: artifact.GeneratedAt,
		SourceFile:  artifact.SourceFile,
		Application: artifact.Application,
		Elements:    BuildElements(artifact),
	}
}

func lastSlash(value string) int {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] == '/' {
			return i
		}
	}

	return -1
}

func convertOutputResources(in []*corerpv20231001preview.ApplicationGraphOutputResource) []ElementOutputResource {
	if len(in) == 0 {
		return nil
	}
	out := make([]ElementOutputResource, 0, len(in))
	for _, r := range in {
		if r == nil {
			continue
		}
		out = append(out, ElementOutputResource{
			ID:   to.String(r.ID),
			Name: to.String(r.Name),
			Type: to.String(r.Type),
		})
	}
	return out
}
