# Phase 0 — Research

**Feature**: Base Resource Manifest
**Branch**: `210-base-resource-manifest`
**Date**: 2026-06-19

This document records the decisions taken before design (Phase 1) to resolve unknowns.

---

## Decision 1 — Resolve OQ-001 by choosing Approach A (user-facing `$ref`)

**Decision**: Implement Approach A only — the per-type YAML uses `allOf: [{ $ref: "radius:base" }]` to opt into the four common properties (see [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for the grammar). Approach B (implicit injection) is documented as a possible alternative but is NOT pursued in this feature. If Approach A proves unacceptable in practice (discoverability surprises, doc maintenance overhead), a separate future POC can revisit Approach B; the two are not pursued in parallel.

**Rationale**: Approach A's discoverability — the keyword appears in the YAML so a reader sees that the type inherits the base — is the dominant value the team wants from this feature. Approach B's "the YAML is shorter" advantage is real but the cost (no signal in the file that four properties exist; reader must consult docs) is judged worse. Building both in parallel was considered and rejected because parallel implementations double the review burden, complicate the spec/plan/research/contracts artifacts, and force a comparison decision later instead of letting the team commit to one approach now.

**Alternatives considered**:
- *Build Approach B only*: rejected as above — no in-YAML signal of inheritance.
- *Build both approaches in parallel POCs and compare*: considered and rejected — the team prefers committing to one approach and iterating, with Approach B available as a future POC if needed.
- *Make `Radius.Core/baseResource` a registered runtime resource type*: explicitly rejected in spec OQ-001 (conflates schema composition with runtime data; adds boilerplate to consumer Bicep instead of removing it).

---

## Decision 2 — Approach A keyword: `allOf: [{ $ref: "radius:base" }]` (sub-mechanism A.1)

**Decision**: The user-facing keyword is JSON-Schema's / OpenAPI's standard `$ref`, placed inside the composition keyword `allOf:`, with a Radius-owned custom URI scheme. The only legal value in v1 is `radius:base`. The custom-keyword alternative (sub-mechanism A.2: `extends: Radius.Core/baseResource`) is rejected. See [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for the full grammar and placement contract.

**Rationale**: `$ref` is part of the JSON-Schema vocabulary that `pkg/schema/validator.go` and the underlying `kin-openapi/openapi3` library already understand. `allOf` is the JSON-Schema-idiomatic composition keyword (a sibling-`$ref` under `properties:` is broken JSON-Schema because `properties:` is keyed by property names). Reusing these standard keywords means generic schema tooling (linters, IDE plugins, JSON-Schema-aware editors) can at least *see* that the type composes with another schema even if they cannot resolve the `radius:` scheme. The cost is registering a `radius:` URI loader in our new `pkg/schema/baseresource/` package, which is a localized change. A.2 would introduce a permanent non-standard keyword that no third-party tool understands.

**URI shape**: `radius:base` (no JSON-Pointer fragment). The URI references the whole base schema object, not just its `properties:` subtree, because `allOf` expects schemas. The earlier sketch `radius:base#/properties` was rejected once placement was nailed down: a fragment pointing at a properties-map cannot be the operand of `allOf` (the operand must be a schema).

**Alternatives considered**:
- *A.2 — `extends: Radius.Core/baseResource`*: rejected as above. Could be re-evaluated later if `$ref` resolution proves surprising.

---

## Decision 3 — Where the base manifest YAML lives

**Decision**: The single source-of-truth file is `pkg/schema/baseresource/base.yaml`, embedded into the Radius binary via `//go:embed`. There is one file, not one per environment (dev/self-hosted).

**Rationale**: FR-012 requires a single well-known location, shipped with Radius, with out-of-tree authors not needing to fetch or co-author it. Embedding via `go:embed` guarantees this — the file is part of every Radius binary, the loader cannot fail at runtime due to a missing file, and out-of-tree resource-type authors never see the file in their own repos. Placing the file inside the `baseresource` package keeps the "definition" and the "loader" co-located, which Principle VIII (Separation of Concerns) favors.

The file is plain YAML matching the existing per-type schema shape so the loader can parse it with the same `openapi3` machinery already used by the validator.

**Alternatives considered**:
- *Put `base.yaml` under `deploy/manifest/built-in-providers/`*: rejected. That directory holds manifests *loaded at server startup*; the base manifest is conceptually different (it is part of the type system, not a registered type) and confusing the two would mislead readers.
- *Define the four properties in pure Go (no YAML file)*: rejected. The four properties are schema content; expressing them as Go literals would duplicate `openapi3` field definitions verbosely and would not be a *manifest* in any meaningful sense, undermining the feature name. A YAML file is what FR-012's "well-known location" implies.

---

## Decision 4 — `codeReference` v1 shape: optional string treated as a URI

**Decision** (already locked in spec — recorded here for design completeness): `codeReference` is a single optional string field. Radius validates only that the value is a string. Tooling MAY treat it as a URI when rendering (`rad resource show`, graph view). The structured form `{repo, commit, path, line}` is deferred to a future additive feature.

**Rationale**: FR-005. A string wire format keeps the v1 surface minimal and lets the structured form ship later as an *additive* change (a future Radius release can teach the validator to also accept a struct, with the string form remaining valid). Doing the structured form now would require designing four sub-fields' validation rules and Bicep-side ergonomics in this feature — out of scope.

**Source touchpoints (additive)**:
- `pkg/schema/baseresource/base.yaml` — declares `codeReference: {type: string}`
- `pkg/resourceutil/utils.go` — append `"codeReference"` to `BasicProperties`
- `pkg/dynamicrp/datamodel/dynamicresource.go` — add a `CodeReference()` accessor mirroring `EnvironmentID()`, returning `string`

---

## Decision 5 — Validator change strategy: delete the global env-required rule outright

**Decision**: In `pkg/schema/validator.go`, the block (lines 832–835 per the touchpoint map) that enforces "`environment` must always be present in the schema regardless of the Required array" is removed entirely. Per-type `required:` arrays in YAML continue to be honored (so an author who explicitly declares `environment` as required keeps that behavior — FR-004).

**Rationale**: FR-013 forbids backward-compat machinery (no validator-version pinning, no schema snapshotting). A clean delete is the simplest implementation and matches the spec's "simple uniform code change" stance. The breaking-change documentation (FR-006) covers what changes for existing authors.

The three corresponding tests in `validator_test.go` (lines 1788, 1819, 1840 per the touchpoint map: "environment property always required", "environment property missing from any schema should fail", "environment property present") are *inverted*, not deleted — they become regression tests asserting that env is now optional unless declared required.

**Alternatives considered**:
- *Keep the env-required rule and exempt manifests that opt into the base*: rejected. Adds conditional logic to the validator (knowing whether a given schema "uses the base") that contradicts FR-013 and Principle VII.

---

## Decision 6 — Apply chokepoint location

**Decision**: `baseresource.Apply(schema)` is called from inside `pkg/cli/manifest/validation.go::validateManifestSchemas()`, immediately before the call to `ValidateSchema` at line 87 (per the touchpoint map). The same `Apply()` call is also added on the bicep-tools side, inside `bicep-tools/pkg/converter/converter.go::addResourceTypeForAPIVersion()` immediately before the call to `addSchemaType()` at line 147.

**Rationale**: This is the single chokepoint that both the server-side built-in loader (`pkg/ucp/initializer/service.go`) and the CLI registration path (`pkg/cli/manifest/registermanifest.go`) already converge through (both call `ValidateManifest` which calls `validateManifestSchemas`). Putting `Apply()` there guarantees identical treatment for built-in types and user-registered types, eliminating a class of "works for one path but not the other" bugs. The Bicep generator-side call ensures the generated Bicep types include the four common properties whenever the YAML uses `$ref` — closing FR-010.

`Apply()` behavior: walk the schema's `AllOf` array looking for an entry with `$ref: "radius:base"`; if found, drop it from `AllOf` and merge the base schema's `properties` into the local schema's `properties` (per-type declarations that name any of the four win — FR-004); if absent, `Apply()` is a no-op and the schema passes through unchanged. See [contracts/inheritance-keyword.md](../contracts/inheritance-keyword.md) for the full grammar and error contract.

**Alternatives considered**:
- *Apply inside the validator itself*: rejected. Couples schema composition with validation; violates Principle VIII.
- *Apply at YAML-parse time*: rejected. The parser doesn't have access to the resolved `openapi3.Schema` form; calling at validation time uses the type already used by every downstream consumer.

---

## Decision 7 — Built-in manifests stay as-is in this feature

**Decision**: The YAML files under `deploy/manifest/built-in-providers/dev/` and `deploy/manifest/built-in-providers/self-hosted/` are NOT edited as part of this feature. Existing inline declarations of `application`, `environment`, `connections` in those files remain.

**Rationale**: The point of this feature is the *authoring experience for new types*, not a sweep through existing types. FR-004 guarantees that explicit declarations win, so leaving them in place changes nothing functional. Migrating each built-in type's YAML to use `$ref` is mechanical follow-on cleanup that can land in a separate PR once the validator change has stabilized.

**Note for testing**: the functional test (Decision 8) must register a *new, `$ref`-using* type rather than rely on a built-in.

---

## Decision 8 — Functional test pattern

**Decision**: Add one new functional test at `test/functional-portable/dynamicrp/noncloud/baseresource_test.go` following the `magpiego` pattern. The test registers a fresh resource type whose YAML uses `allOf: [{ $ref: "radius:base" }]` and declares only one type-specific property (no per-type declarations of the four common ones), then deploys an instance that sets `environment`, `application`, `connections`, and `codeReference`, and asserts each is honored by the runtime (resource IDs resolved correctly, connections returned by the existing extraction helper, `codeReference` round-trips).

**Rationale**: SC-001, SC-004, and the FR-001 / FR-002 / FR-009 / FR-010 chain all need an end-to-end demonstration that "`$ref`-using YAML works at runtime." This test is the demonstrable artifact.

**Alternatives considered**:
- *Add only unit tests*: insufficient — the spec's success criteria are end-to-end, and the bicep-tools generator integration is not exercised by unit tests alone.

---

## Decision 9 — Documentation deliverable

**Decision**: One new contributor doc, `docs/contributing/contributing-code/contributing-code-base-resource-manifest.md`, with three sections matching the Breaking Changes & Documentation Impact list in the spec: (1) What changed, (2) Who is affected and what action they take, (3) How to author a new resource type using the base via `allOf: [{ $ref: "radius:base" }]` (including the common footgun of putting `$ref` under `properties:` instead of `allOf:`). The release notes for the Radius release that ships this feature get a one-paragraph entry pointing at this doc.

**Rationale**: SC-003 and FR-006 both require a documentation deliverable; the spec specifies *what* must be documented but not *where*. A single contributor doc keeps the change discoverable from the rest of `docs/contributing/contributing-code/`.

**Source touchpoints**: `docs/contributing/contributing-code/`.

---

## Summary of NEEDS CLARIFICATION resolutions

| Source | What was unresolved | Resolution location |
|---|---|---|
| Technical Context: Language/Version | Was the feature in Go? | Resolved as Go in plan.md Technical Context |
| Technical Context: Storage | Did this need a store? | Resolved as N/A in plan.md Technical Context |
| Spec OQ-001 | A vs B | Decision 1 (Approach A — user-facing `$ref`); Decision 2 (sub-mechanism A.1) |
| Spec FR-005 | codeReference shape | Decision 4 — already locked by spec clarification |
| Spec FR-012 | Base file location | Decision 3 |
| Spec FR-006 / SC-003 | Where to document | Decision 9 |

All NEEDS CLARIFICATION markers in plan.md's Technical Context have been replaced with concrete answers above. OQ-001 in spec.md is updated in the same commit as this plan to record the Approach A decision and close the question.
