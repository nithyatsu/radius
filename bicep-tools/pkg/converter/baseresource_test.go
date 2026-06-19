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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/radius-project/radius/bicep-tools/pkg/manifest"
	"gopkg.in/yaml.v3"
)

func TestApplyBaseResource_NilSchema(t *testing.T) {
	if err := applyBaseResource(nil); err != nil {
		t.Fatalf("expected no error for nil schema, got %v", err)
	}
}

func TestApplyBaseResource_NoAllOf(t *testing.T) {
	schema := &manifest.Schema{
		Type:       "object",
		Properties: map[string]manifest.Schema{"name": {Type: "string"}},
	}
	if err := applyBaseResource(schema); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, hasApplication := schema.Properties["application"]; hasApplication {
		t.Fatalf("base properties must not be merged when allOf is absent")
	}
}

func TestApplyBaseResource_AllOfWithoutRadiusRef(t *testing.T) {
	schema := &manifest.Schema{
		Type:  "object",
		AllOf: []manifest.Schema{{Ref: "#/components/schemas/Other"}},
	}
	if err := applyBaseResource(schema); err != nil {
		t.Fatalf("expected no error for unrelated allOf entries, got %v", err)
	}
	if len(schema.AllOf) != 1 {
		t.Fatalf("non-radius allOf entries must be preserved; got %d entries", len(schema.AllOf))
	}
}

func TestApplyBaseResource_InjectsAllFourBaseProperties(t *testing.T) {
	schema := &manifest.Schema{
		Type: "object",
		AllOf: []manifest.Schema{
			{Ref: "radius:base"},
		},
		Properties: map[string]manifest.Schema{
			"size": {Type: "string"},
		},
	}

	if err := applyBaseResource(schema); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{"application", "environment", "connections", "codeReference", "size"}
	for _, name := range want {
		if _, ok := schema.Properties[name]; !ok {
			t.Errorf("expected property %q to be present after applyBaseResource", name)
		}
	}

	if len(schema.AllOf) != 0 {
		t.Errorf("radius:base entry should have been stripped from AllOf; got %d entries", len(schema.AllOf))
	}
}

func TestApplyBaseResource_PerTypeDeclarationWins(t *testing.T) {
	narrowed := "narrowed by the per-type schema"
	schema := &manifest.Schema{
		Type: "object",
		AllOf: []manifest.Schema{
			{Ref: "radius:base"},
		},
		Properties: map[string]manifest.Schema{
			"environment": {Type: "string", Description: &narrowed},
		},
	}

	if err := applyBaseResource(schema); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	envProp := schema.Properties["environment"]
	if envProp.Description == nil || *envProp.Description != narrowed {
		t.Errorf("per-type description must win; got %v", envProp.Description)
	}
}

func TestApplyBaseResource_UnsupportedRadiusRefReturnsError(t *testing.T) {
	cases := []string{
		"radius:base/something",
		"radius:",
		"radius:other",
	}
	for _, ref := range cases {
		t.Run(ref, func(t *testing.T) {
			schema := &manifest.Schema{
				Type: "object",
				AllOf: []manifest.Schema{
					{Ref: ref},
				},
			}
			err := applyBaseResource(schema)
			if err == nil {
				t.Fatalf("expected error for ref %q, got nil", ref)
			}
			if !strings.Contains(err.Error(), "radius:base") {
				t.Errorf("error message must mention the supported URI; got %q", err.Error())
			}
			if !strings.Contains(err.Error(), "allOf[0]") {
				t.Errorf("error message must include the JSON-Pointer path; got %q", err.Error())
			}
		})
	}
}

// TestApplyBaseResource_PropertiesMatchCanonicalYAML asserts that the
// hardcoded baseProperties() list in bicep-tools matches the canonical
// pkg/schema/baseresource/base.yaml. If this test ever fails, the canonical
// list and the bicep-tools copy have drifted apart and one of them needs to be
// brought back in sync.
func TestApplyBaseResource_PropertiesMatchCanonicalYAML(t *testing.T) {
	// Resolve the canonical YAML path relative to the workspace root. The
	// bicep-tools/pkg/converter directory is three levels below the repo root.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	canonicalPath := filepath.Join(wd, "..", "..", "..", "pkg", "schema", "baseresource", "base.yaml")

	bytes, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("failed to read canonical base.yaml at %s: %v", canonicalPath, err)
	}

	var canonical manifest.Schema
	if err := yaml.Unmarshal(bytes, &canonical); err != nil {
		t.Fatalf("failed to parse canonical base.yaml: %v", err)
	}

	got := baseProperties()

	canonicalNames := make([]string, 0, len(canonical.Properties))
	for name := range canonical.Properties {
		canonicalNames = append(canonicalNames, name)
	}
	sort.Strings(canonicalNames)

	gotNames := make([]string, 0, len(got))
	for name := range got {
		gotNames = append(gotNames, name)
	}
	sort.Strings(gotNames)

	if strings.Join(canonicalNames, ",") != strings.Join(gotNames, ",") {
		t.Fatalf("canonical YAML property names %v and bicep-tools baseProperties() names %v have drifted apart", canonicalNames, gotNames)
	}

	for _, name := range canonicalNames {
		canonProp := canonical.Properties[name]
		gotProp := got[name]
		if canonProp.Type != gotProp.Type {
			t.Errorf("property %q type differs: canonical=%q bicep-tools=%q", name, canonProp.Type, gotProp.Type)
		}
		canonHasAdditional := canonProp.AdditionalProperties != nil
		gotHasAdditional := gotProp.AdditionalProperties != nil
		if canonHasAdditional != gotHasAdditional {
			t.Errorf("property %q additionalProperties presence differs: canonical=%v bicep-tools=%v", name, canonHasAdditional, gotHasAdditional)
		}
		if canonHasAdditional && gotHasAdditional {
			if canonProp.AdditionalProperties.Type != gotProp.AdditionalProperties.Type {
				t.Errorf("property %q additionalProperties.type differs: canonical=%q bicep-tools=%q",
					name, canonProp.AdditionalProperties.Type, gotProp.AdditionalProperties.Type)
			}
		}
	}
}
