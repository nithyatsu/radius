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

package baseresource

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestApply_NilSchema(t *testing.T) {
	require.NoError(t, Apply(nil))
}

func TestApply_NoAllOf(t *testing.T) {
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: openapi3.Schemas{
			"widgetSize": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
		},
	}

	require.NoError(t, Apply(schema))

	// Properties should be unchanged; no base properties injected.
	require.Len(t, schema.Properties, 1)
	require.Contains(t, schema.Properties, "widgetSize")
	for _, base := range PropertyNames() {
		require.NotContains(t, schema.Properties, base, "raw type should not gain base property %q", base)
	}
}

func TestApply_AllOfWithoutRadiusRef(t *testing.T) {
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		AllOf: openapi3.SchemaRefs{
			{Ref: "#/components/schemas/Something"},
		},
		Properties: openapi3.Schemas{
			"widgetSize": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
		},
	}

	require.NoError(t, Apply(schema))

	// No radius: ref => Apply is a no-op. AllOf entry preserved; no base properties merged.
	require.Len(t, schema.AllOf, 1)
	require.Equal(t, "#/components/schemas/Something", schema.AllOf[0].Ref)
	require.Len(t, schema.Properties, 1)
}

func TestApply_InjectsAllFourBaseProperties(t *testing.T) {
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		AllOf: openapi3.SchemaRefs{
			{Ref: URI},
		},
		Properties: openapi3.Schemas{
			"widgetSize": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
		},
	}

	require.NoError(t, Apply(schema))

	// The radius:base entry is removed from AllOf.
	require.Empty(t, schema.AllOf)

	// All four base properties are now present, plus the original type-specific one.
	require.Len(t, schema.Properties, 5)
	require.Contains(t, schema.Properties, "widgetSize")
	for _, name := range PropertyNames() {
		require.Contains(t, schema.Properties, name, "base property %q should be merged into the schema", name)
		require.NotNil(t, schema.Properties[name].Value, "base property %q should have a non-nil value", name)
	}

	// Spot-check a couple of expected shapes against the embedded base.yaml.
	require.True(t, schema.Properties["application"].Value.Type.Is("string"))
	require.True(t, schema.Properties["environment"].Value.Type.Is("string"))
	require.True(t, schema.Properties["codeReference"].Value.Type.Is("string"))
	require.True(t, schema.Properties["connections"].Value.Type.Is("object"))
}

func TestApply_PerTypeDeclarationWins(t *testing.T) {
	// Author declares environment per-type with a custom description; the
	// per-type declaration must win entirely (FR-004) — the base manifest's
	// version of environment is discarded for this schema.
	customEnvironment := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"string"},
			Description: "Custom per-type environment override",
		},
	}
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		AllOf: openapi3.SchemaRefs{
			{Ref: URI},
		},
		Properties: openapi3.Schemas{
			"environment": customEnvironment,
			"widgetSize":  &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
		},
		Required: []string{"environment"},
	}

	require.NoError(t, Apply(schema))

	// Per-type environment is preserved exactly.
	require.Same(t, customEnvironment, schema.Properties["environment"])
	require.Equal(t, "Custom per-type environment override", schema.Properties["environment"].Value.Description)

	// The other three base properties are still merged in.
	for _, name := range []string{"application", "connections", "codeReference"} {
		require.Contains(t, schema.Properties, name)
	}

	// Per-type required: array is untouched by Apply.
	require.Equal(t, []string{"environment"}, schema.Required)
}

func TestApply_UnsupportedRadiusRefReturnsError(t *testing.T) {
	cases := []struct {
		name string
		ref  string
	}{
		{name: "unknown sub-resource", ref: "radius:base/something"},
		{name: "empty scheme-specific part", ref: "radius:"},
		{name: "unknown name", ref: "radius:other"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				AllOf: openapi3.SchemaRefs{
					{Ref: tc.ref},
				},
				Properties: openapi3.Schemas{
					"widgetSize": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
				},
			}

			err := Apply(schema)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.ref)
			require.Contains(t, err.Error(), URI, "error must point the author at the only legal value")
			require.Contains(t, err.Error(), "allOf[0]", "error must include a path pointer to the offending entry")
		})
	}
}

func TestApply_RadiusRefAtNonZeroIndex(t *testing.T) {
	// The radius:base entry doesn't have to be the first allOf entry.
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		AllOf: openapi3.SchemaRefs{
			{Ref: "#/components/schemas/SomethingElse"},
			{Ref: URI},
		},
		Properties: openapi3.Schemas{
			"widgetSize": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
		},
	}

	require.NoError(t, Apply(schema))

	// The radius:base entry is removed; the other entry is preserved.
	require.Len(t, schema.AllOf, 1)
	require.Equal(t, "#/components/schemas/SomethingElse", schema.AllOf[0].Ref)

	// All four base properties merged.
	for _, name := range PropertyNames() {
		require.Contains(t, schema.Properties, name)
	}
}

func TestPropertyNames_FrozenSet(t *testing.T) {
	// Regression guard for FR-012 (the set of common properties is frozen).
	// If you are changing this assertion you are changing the wire format and
	// need a separate spec / mechanism — see specs/210-base-resource-manifest.
	require.Equal(t, []string{"application", "environment", "connections", "codeReference"}, PropertyNames())
}

func TestLoadBaseSchema_HasAllFourProperties(t *testing.T) {
	base, err := loadBaseSchema()
	require.NoError(t, err)
	require.NotNil(t, base)

	for _, name := range PropertyNames() {
		require.Contains(t, base.Properties, name, "embedded base.yaml must declare property %q", name)
		require.NotNil(t, base.Properties[name].Value, "embedded base.yaml property %q must have a value", name)
	}

	// FR-012: base manifest must NOT declare any required: entries.
	require.Empty(t, base.Required, "embedded base.yaml must not declare any properties as required (FR-002)")
}
