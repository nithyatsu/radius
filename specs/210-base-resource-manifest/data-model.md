# Phase 1 — Data Model

**Feature**: Base Resource Manifest
**Date**: 2026-06-19

This feature is *schema-shaped* rather than data-shaped: it does not introduce new persisted entities, new database tables, or new HTTP request/response types. What it introduces is **a definition of how a resource type's effective schema is composed from two sources**: the per-type YAML the author writes, and the shared base manifest Radius ships. This document captures the entities involved in that composition and the rules that govern it.

---

## Entities

### Base resource manifest (NEW)

The single, repo-owned YAML file that names the four common Radius schema properties and declares their JSON-Schema shape.

| Field | Type | Notes |
|---|---|---|
| `properties.application` | `string` | Resource ID of a Radius `Applications.Core/applications` resource. Optional. |
| `properties.environment` | `string` | Resource ID of a Radius `Applications.Core/environments` resource. Optional (was globally required pre-feature). |
| `properties.connections` | `object` with `additionalProperties` (free-form connection map) | The map of connection name → source resource ID. Optional. |
| `properties.codeReference` | `string` (NEW property name) | Treated as a URI. Optional. v1 wire format is a flat string; structured form is a future additive change. |

**Storage location**: `pkg/schema/baseresource/base.yaml`, embedded into every Radius binary via `//go:embed`.

**Frozen forever** (FR-012, spec clarification): future Radius releases MUST NOT add, remove, rename, or change the type of a property in this file. Promoting any additional property to common status is a separate spec / separate file / separate feature.

**Forbidden additions**: the file MUST NOT declare `status` or `recipe` — these remain reserved-and-forbidden for all schemas including the base.

### Per-type resource manifest (EXISTING; behavior changes)

The YAML an author writes today to declare one resource type. Continues to follow the existing schema shape (the same one `pkg/cli/manifest` parses). After this feature:

- The four common properties may be omitted from the per-type `properties:` block.
- Any of the four that *are* declared per-type override the base manifest's version of that property (FR-004) — including making it required by listing it in the per-type `required:` array.
- Declaring a property named `status` or `recipe` remains a registration error.

### Effective schema (CONCEPTUAL; new term)

The merged schema that the validator validates against and that the bicep-tools generator emits Bicep for. Not a new persisted entity — exists in memory only.

**Composition rule**:
1. The composition runs only on a schema whose `allOf:` array contains an entry with `$ref: "radius:base"` (per [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md)). Schemas without that entry pass through unchanged (and therefore do NOT receive the four properties — they are "raw" types).
2. When the keyword is present: start from the per-type schema as parsed. For each of the four base properties: if the per-type schema does NOT already declare a property of that name (case-sensitive), copy the base manifest's declaration into the effective schema's `properties` map. If the per-type schema DOES declare it (case-sensitive), the per-type declaration wins entirely (the base's declaration is discarded for that property, including any base-side `required:` membership). The `$ref` node itself is removed from the effective schema.
3. The effective schema's `required:` array is the per-type `required:` array, unchanged. The base manifest contributes no entries to `required`.

### Common Radius property (NEW term)

A schema property whose presence, name, and runtime semantics are defined by Radius itself (in the base resource manifest) rather than by the resource type author. Set: `{application, environment, connections, codeReference}`.

Contrast with:
- **Type-specific property** (existing term, applies to every other property in a type's `properties:` block).
- **Reserved-and-forbidden property** (existing): `{status, recipe}` — cannot appear in any schema.

---

## State transitions

There are no state machines in this feature; YAML is loaded once at registration time and the resulting effective schema is persisted to the existing UCP store unchanged.

The single transition worth naming:

```
per-type YAML                                  effective schema (in memory)
+------------------+                          +-------------------------------+
| properties:      |   baseresource.Apply()   | properties:                   |
|   widgetSize: …  | -----------------------> |   widgetSize: …               |
| required:        |                          |   application: …  (from base) |
|   - widgetSize   |                          |   environment: …  (from base) |
+------------------+                          |   connections: …  (from base) |
                                              |   codeReference: …(from base) |
                                              | required:                     |
                                              |   - widgetSize                |
                                              +-------------------------------+
```

The transition runs:
- Once per resource type at `rad resource-type create` time (via `pkg/cli/manifest/validation.go`).
- Once per built-in type at server startup (via `pkg/ucp/initializer/service.go`).
- Once per resource type per Bicep generation (via `bicep-tools/pkg/converter/converter.go`).

The effective schema is what gets validated, persisted, and used to drive Bicep emission. The original per-type YAML is preserved on disk; the composition happens at load.

---

## Validation rules

Layered on top of the existing schema validation (`pkg/schema/validator.go::checkReservedProperties()` after the env-required block is removed per Decision 5 in [research.md](./research.md)):

| Rule | Source | Enforced where |
|---|---|---|
| `status` and `recipe` may not appear as property names | Existing (FR-008) | `checkReservedProperties()` — unchanged |
| `application`, `environment`, `connections`, `codeReference` if declared per-type must have compatible primitive types (e.g. `environment` as a string, `connections` as an object) | New (FR-007) | `checkReservedProperties()` — extended; emits actionable error at `rad resource-type create` time before any control-plane round-trip |
| Per-type schema's `required:` array honored as-is | New (replaces former env-always-required) | Implicit — by removing the env block at validator.go:832–835 (Decision 5) |
| Case-sensitive property name match for "is this an override of a base property?" | New (Edge Case in spec) | `pkg/schema/baseresource/loader.go::Apply()` — uses Go map lookup which is case-sensitive; documents that `Application` (capital A) is a *different* property and falls through to general name validation, which will likely produce a validation error since it doesn't match any registered shape |
| `$ref` value inside `allOf:` MUST be the exact literal `"radius:base"`; any other `radius:` URI is a registration error | New (this feature's keyword grammar — see [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md)) | `pkg/schema/baseresource/loader.go::Apply()` |

---

## Out-of-band entities NOT introduced

To prevent scope creep, the following are explicitly NOT entities of this feature even though related discussion may suggest them:

- A registered runtime resource type called `Radius.Core/baseResource` — rejected in OQ-001.
- A per-author / per-namespace base manifest. The spec ships only the single Radius-owned base; per-author bases are out of scope (spec § Out of Scope).
- A migration record / versioning marker per registered type. Forbidden by FR-013.
- A structured form of `codeReference` (`{repo, commit, path, line}`). Deferred per Decision 4.
