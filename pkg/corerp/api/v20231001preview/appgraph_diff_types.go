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

package v20231001preview

// GraphDiff represents the differences between two app graphs.
type GraphDiff struct {
	// BaseCommit is the commit SHA of the base graph (optional).
	BaseCommit string `json:"baseCommit,omitempty"`

	// HeadCommit is the commit SHA of the head graph (optional).
	HeadCommit string `json:"headCommit,omitempty"`

	// AddedResources are resources present in head but not in base.
	AddedResources []AppGraphResource `json:"addedResources"`

	// RemovedResources are resources present in base but not in head.
	RemovedResources []AppGraphResource `json:"removedResources"`

	// ModifiedResources are resources with changed properties.
	ModifiedResources []ResourceDiff `json:"modifiedResources"`

	// AddedConnections are connections present in head but not in base.
	AddedConnections []AppGraphConnectionStatic `json:"addedConnections"`

	// RemovedConnections are connections present in base but not in head.
	RemovedConnections []AppGraphConnectionStatic `json:"removedConnections"`

	// Summary provides a human-readable overview.
	Summary DiffSummary `json:"summary"`
}

// ResourceDiff captures changes to a single resource.
type ResourceDiff struct {
	// ID is the resource ID (same in both base and head).
	ID string `json:"id"`

	// Name is the resource name.
	Name string `json:"name"`

	// Type is the resource type.
	Type string `json:"type"`

	// ChangedProperties lists the property paths that changed.
	ChangedProperties []PropertyChange `json:"changedProperties"`
}

// PropertyChange describes a single property modification.
type PropertyChange struct {
	// Path is the JSON path to the changed property (e.g., "properties.container.image").
	Path string `json:"path"`

	// OldValue is the value in the base graph.
	OldValue any `json:"oldValue,omitempty"`

	// NewValue is the value in the head graph.
	NewValue any `json:"newValue,omitempty"`
}

// DiffSummary provides aggregate statistics for the diff.
type DiffSummary struct {
	TotalChanges       int `json:"totalChanges"`
	ResourcesAdded     int `json:"resourcesAdded"`
	ResourcesRemoved   int `json:"resourcesRemoved"`
	ResourcesModified  int `json:"resourcesModified"`
	ConnectionsAdded   int `json:"connectionsAdded"`
	ConnectionsRemoved int `json:"connectionsRemoved"`
}
