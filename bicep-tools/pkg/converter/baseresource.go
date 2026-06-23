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

package converter

import (
	"fmt"
	"strings"

	"github.com/radius-project/radius/bicep-tools/pkg/manifest"
)

// baseResourceURIScheme is the custom URI scheme reserved for Radius schema
// base manifest references. See pkg/schema/baseresource/loader.go for the
// canonical resolver used by the schema validator; this file is the Bicep
// type generator's parallel implementation.
const baseResourceURIScheme = "radius:"

// baseResourceURI is the only legal value of the radius: scheme in this
// release.
const baseResourceURI = "radius:base"

// baseProperties is the hardcoded set of properties contributed by the
// radius:base manifest. The set is FROZEN per FR-012 of the base resource
// manifest spec. The canonical source of truth lives in
// pkg/schema/baseresource/base.yaml; this list is a deliberate parallel copy
// so that bicep-tools can remain a standalone module. A synchronization test
// (TestApplyBaseResource_PropertiesMatchCanonicalYAML) asserts that the two
// lists agree exactly.
func baseProperties() map[string]manifest.Schema {
	s := func(v string) *string { return &v }
	return map[string]manifest.Schema{
		"application": {
			Type:        "string",
			Description: s("The resource ID of the Radius Applications.Core/applications resource this resource belongs to."),
		},
		"environment": {
			Type:        "string",
			Description: s("The resource ID of the Radius Applications.Core/environments resource this resource is deployed into."),
		},
		"connections": {
			Type: "object",
			AdditionalProperties: &manifest.Schema{
				Type: "object",
			},
			Description: s("The map of connection name to source resource ID for this resource."),
		},
		"codeReference": {
			Type:        "string",
			Description: s("An optional URI that points back to the authoring source for this resource (for example, a Git URL with commit SHA and line range)."),
		},
	}
}

// applyBaseResource walks schema.AllOf, finds an entry whose $ref equals
// "radius:base", removes it, and merges the base properties into the schema
// using per-type-wins precedence. It mirrors the semantics of
// pkg/schema/baseresource.Apply().
//
// Behavior:
//   - nil schema or empty AllOf: no-op.
//   - allOf entries with no radius: ref: ignored.
//   - exactly one matching entry: properties merged, entry removed.
//   - any radius: $ref other than "radius:base": returns an error including
//     the allOf index path-pointer.
func applyBaseResource(schema *manifest.Schema) error {
	if schema == nil {
		return nil
	}
	if len(schema.AllOf) == 0 {
		return nil
	}

	matchIndex := -1
	for i := range schema.AllOf {
		ref := schema.AllOf[i].Ref
		if !strings.HasPrefix(ref, baseResourceURIScheme) {
			continue
		}
		if ref != baseResourceURI {
			return fmt.Errorf(
				"applyBaseResource: unsupported %s $ref %q at allOf[%d] — only %q is supported in this version",
				baseResourceURIScheme, ref, i, baseResourceURI,
			)
		}
		matchIndex = i
		break
	}

	if matchIndex == -1 {
		return nil
	}

	if schema.Properties == nil {
		schema.Properties = map[string]manifest.Schema{}
	}
	for name, baseProp := range baseProperties() {
		if _, alreadyDeclared := schema.Properties[name]; alreadyDeclared {
			continue
		}
		schema.Properties[name] = baseProp
	}

	schema.AllOf = append(schema.AllOf[:matchIndex], schema.AllOf[matchIndex+1:]...)
	return nil
}
