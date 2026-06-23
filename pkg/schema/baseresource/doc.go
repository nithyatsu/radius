/*
Copyright 2026 The Radius Authors.

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

// Package baseresource embeds the Radius base resource manifest and resolves
// the radius:base URI into per-type resource type schemas.
//
// A resource type author opts into the four common Radius properties
// (application, environment, connections, codeReference) by writing
//
//	allOf:
//	  - $ref: "radius:base"
//
// alongside their per-type properties: block. At schema-validation time, Apply
// walks the schema's AllOf array, removes the matching $ref entry, and merges
// the base manifest's properties into the local properties map using
// per-type-wins precedence (FR-004 of the base resource manifest spec).
//
// See specs/210-base-resource-manifest/contracts/inheritance-keyword.md for
// the full grammar, placement, and error contract.
package baseresource
