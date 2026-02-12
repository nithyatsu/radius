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

import "time"

// AppGraph represents the complete application topology extracted from Bicep files.
// This is the primary output of the `rad app graph <file.bicep>` command.
type AppGraph struct {
	// Metadata contains generation context and provenance information.
	Metadata AppGraphMetadata `json:"metadata"`

	// Resources contains all resource nodes in the application.
	Resources []AppGraphResource `json:"resources"`

	// Connections contains all edges between resources.
	Connections []AppGraphConnectionStatic `json:"connections"`
}

// AppGraphMetadata contains generation context for the app graph.
type AppGraphMetadata struct {
	// GeneratedAt is the UTC timestamp when this graph was generated.
	GeneratedAt time.Time `json:"generatedAt"`

	// RadiusCliVersion is the version of rad CLI used to generate this graph.
	RadiusCliVersion string `json:"radiusCliVersion"`

	// SourceFiles lists all Bicep files that contributed to this graph.
	SourceFiles []string `json:"sourceFiles"`

	// SourceHash is a SHA256 hash of all source files for staleness detection.
	SourceHash string `json:"sourceHash"`

	// GitCommit is the current git commit SHA (if in a git repository).
	GitCommit string `json:"gitCommit,omitempty"`
}

// AppGraphResource represents a single resource in the application topology.
type AppGraphResource struct {
	// ID is the fully qualified Radius resource ID.
	// Format: /planes/radius/local/resourceGroups/{rg}/providers/{type}/{name}
	ID string `json:"id"`

	// Name is the human-readable resource name.
	Name string `json:"name"`

	// Type is the resource type (e.g., Applications.Core/containers).
	Type string `json:"type"`

	// SourceLocation indicates where this resource is defined.
	SourceLocation SourceLocation `json:"sourceLocation"`

	// GitInfo contains git metadata for this resource (optional).
	GitInfo *GitInfo `json:"gitInfo,omitempty"`

	// Properties contains type-specific resource configuration.
	// Stored as map for flexibility across resource types.
	Properties map[string]any `json:"properties,omitempty"`
}

// SourceLocation indicates the Bicep source file and line where a resource is defined.
type SourceLocation struct {
	// File is the path to the Bicep file (relative to repo root).
	File string `json:"file"`

	// Line is the 1-based line number where the resource begins.
	Line int `json:"line"`

	// Module is the module path if this resource is from an imported module.
	Module string `json:"module,omitempty"`
}

// GitInfo contains git commit information for a resource.
type GitInfo struct {
	// CommitSHA is the full commit hash that last modified this resource.
	CommitSHA string `json:"commitSha"`

	// CommitShort is the abbreviated commit hash for display.
	CommitShort string `json:"commitShort"`

	// Author is the commit author email.
	Author string `json:"author"`

	// Date is the commit timestamp in RFC3339 format.
	Date time.Time `json:"date"`

	// Message is the commit message (first line only).
	Message string `json:"message"`

	// Uncommitted indicates this resource has uncommitted changes.
	Uncommitted bool `json:"uncommitted,omitempty"`
}

// AppGraphConnectionStatic represents a directed edge between two resources in the static graph.
// Named AppGraphConnectionStatic to avoid collision with the auto-generated ApplicationGraphConnection.
type AppGraphConnectionStatic struct {
	// SourceID is the resource ID where the connection originates.
	SourceID string `json:"sourceId"`

	// TargetID is the resource ID where the connection terminates.
	TargetID string `json:"targetId"`

	// Type indicates the kind of connection.
	Type ConnectionType `json:"type"`
}

// ConnectionType enumerates the kinds of resource connections.
type ConnectionType string

const (
	// ConnectionTypeConnection represents a direct connection (e.g., container to database).
	ConnectionTypeConnection ConnectionType = "connection"

	// ConnectionTypeRoute represents a gateway route to a destination.
	ConnectionTypeRoute ConnectionType = "route"

	// ConnectionTypeDependsOn represents an explicit dependsOn relationship.
	ConnectionTypeDependsOn ConnectionType = "dependsOn"
)
