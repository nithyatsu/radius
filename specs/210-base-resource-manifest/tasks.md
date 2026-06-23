---
description: "Task list for Base Resource Manifest"
---

# Tasks: Base Resource Manifest

**Input**: Design documents from [/specs/210-base-resource-manifest/](./)
**Prerequisites**: [plan.md](./plan.md) (required), [spec.md](./spec.md) (required), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Tests are included. The spec's success criteria (SC-001, SC-004, SC-005) and Constitution Principle IV (Testing Pyramid, NON-NEGOTIABLE) both require unit + integration + functional coverage. Test tasks are listed alongside the code they cover and MUST be written so they fail before the corresponding implementation lands.

**Organization**: Only one user story is in scope (User Story 1 — "Author a new resource type without restating Radius boilerplate"). User Story 2 was deferred and User Story 3 was dropped per Clarifications § Session 2026-06-19. All implementation tasks therefore live under Phase 3 (US1) with no parallel-story considerations.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[US1]**: Task belongs to User Story 1
- File paths are absolute from the repository root and follow the structure laid out in [plan.md § Project Structure](./plan.md#project-structure)

---

## Phase 1: Setup

**Purpose**: No new tooling, no new dependencies, no scaffolding. This feature lives entirely in the existing Radius repository with the existing Go toolchain. The single Setup task confirms the working tree is on the right branch and clean before edits begin.

- [ ] T001 Confirm working tree is on branch `210-base-resource-manifest` and clean (`git status`); pull latest `upstream/main` if not already up to date

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Ship the new `pkg/schema/baseresource/` package — the embedded `base.yaml`, the loader, and the `Apply(schema)` function — and its unit tests. Every other phase depends on the `baseresource.Apply` symbol being callable and correct.

**⚠️ CRITICAL**: No US1 work can begin until this phase is complete.

- [ ] T002 Create new package directory `pkg/schema/baseresource/` with a package doc-comment in a new file `pkg/schema/baseresource/doc.go` describing the package as "embeds the Radius base resource manifest and resolves the `radius:base` URI into per-type schemas"
- [ ] T003 [P] Author `pkg/schema/baseresource/base.yaml` per [contracts/base-manifest.schema.yaml](./contracts/base-manifest.schema.yaml): declares exactly the four common properties (`application`, `environment`, `connections`, `codeReference`) with the JSON-Schema shapes from [data-model.md § Entities](./data-model.md#entities); no `required:` array; no `status`/`recipe` keys
- [ ] T004 Implement `pkg/schema/baseresource/loader.go` exposing `Apply(schema *openapi3.Schema) error` per [contracts/inheritance-keyword.md § Resolution](./contracts/inheritance-keyword.md#resolution) — embed `base.yaml` via `//go:embed`, walk `schema.AllOf` for a `radius:`-scheme `$ref`, merge base properties into the local schema's `Properties` map using per-type-wins precedence (FR-004), drop the matched `$ref` entry from `AllOf`, and return the contract's actionable errors for unsupported `radius:` URIs
- [ ] T005 [P] [US1] Write `pkg/schema/baseresource/loader_test.go` covering: (a) schema with `allOf: [{$ref: "radius:base"}]` gains the four properties and loses the `$ref` entry; (b) schema with a per-type declaration of one of the four properties keeps its own declaration and gains the other three; (c) schema with no `radius:` `$ref` passes through unchanged; (d) schema with `$ref: "radius:base/something"` returns the contract's actionable error; (e) schema with `$ref: "radius:"` returns the contract's actionable error

**Checkpoint**: `go test ./pkg/schema/baseresource/...` passes. `baseresource.Apply` is now callable from elsewhere in the tree.

---

## Phase 3: User Story 1 — Author a new resource type without restating Radius boilerplate (Priority: P1) 🎯 MVP

**Story goal**: A resource-type author writes a YAML that opts into the base with `allOf: [{ $ref: "radius:base" }]`, declares only type-specific properties, runs `rad resource-type create -f <file>.yaml`, sees registration succeed, and can deploy an instance that sets any of the four common properties. The same path works end-to-end through CLI validation, the bicep-tools generator, and the dynamic-rp runtime.

**Independent test**: Run the new functional test `test/functional-portable/dynamicrp/noncloud/baseresource_test.go` (created in T013) — it registers a stripped manifest, deploys an instance, asserts the four common properties round-trip.

### Implementation for User Story 1

The task ordering reflects dependency: validator + `BasicProperties` + runtime accessor are leaf edits that can land in parallel; the CLI and bicep-tools wire-ups call `baseresource.Apply` (so they depend only on Phase 2); the functional test depends on all of the above.

- [ ] T006 [P] [US1] Remove the global env-required block in `pkg/schema/validator.go` (the lines that enforce "`environment` must always be present regardless of `Required`" — research.md Decision 5 cites lines 832–835 as the touchpoint; verify with `grep -n 'environment' pkg/schema/validator.go` before editing). Leave the reserved-property check (`status`, `recipe`) and the existing per-type `required:`-array honoring intact.
- [ ] T007 [P] [US1] Update `pkg/schema/validator_test.go`: invert the three pre-existing env-required test cases (research.md Decision 5 cites lines 1788, 1819, 1840) so they now assert env is optional unless the per-type schema declares it in `required:`. Add one new positive case: per-type schema that declares `environment` in `required:` still produces a required-env contract.
- [ ] T008 [P] [US1] Add `"codeReference"` to the `BasicProperties` list in `pkg/resourceutil/utils.go` (research.md Decision 4 cites line 28 as the touchpoint; verify with `grep -n 'BasicProperties' pkg/resourceutil/utils.go`). No code calling this slice should need changes — it is the canonical list of common property names.
- [ ] T009 [US1] Call `baseresource.Apply(schema)` from `pkg/cli/manifest/validation.go::validateManifestSchemas()` immediately before `ValidateSchema` (research.md Decision 6 cites line 87). Import `pkg/schema/baseresource`. Surface the error from `Apply` with the YAML file path and a path-pointer (`allOf[N]`) per [contracts/inheritance-keyword.md § Errors](./contracts/inheritance-keyword.md#errors).
- [ ] T010 [US1] Extend `pkg/cli/manifest/validation_test.go` with two new test cases: (a) a YAML that uses `allOf: [{ $ref: "radius:base" }]` and declares only one type-specific property validates successfully; (b) a YAML with `$ref: "radius:base/something"` returns the contract's actionable error including the YAML file path.
- [ ] T011 [US1] Call `baseresource.Apply(schema)` from `bicep-tools/pkg/converter/converter.go::addResourceTypeForAPIVersion()` immediately before `addSchemaType()` (research.md Decision 6 cites line 147). Import the new package; mirror the error-surfacing pattern from T009.
- [ ] T012 [US1] Extend `bicep-tools/pkg/converter/converter_test.go` with one new case: a YAML that uses `allOf: [{ $ref: "radius:base" }]` and declares only one type-specific property emits a Bicep type whose properties include all four common properties plus the type-specific one. Verify pre-existing tests (YAML without `$ref`) continue to emit byte-identical Bicep (FR-010 wire-format invariance).
- [ ] T013 [P] [US1] Add `CodeReference()` accessor to `pkg/dynamicrp/datamodel/dynamicresource.go` mirroring the shape of `EnvironmentID()` (research.md Decision 4 — return type `string`; pulls from the resource's properties map using key `"codeReference"`).
- [ ] T014 [P] [US1] Add `Test_DynamicResource_CodeReference` to `pkg/dynamicrp/datamodel/dynamicresource_test.go` mirroring the structure of the existing `Test_…_EnvironmentID` cases.
- [ ] T015 [US1] Create `test/functional-portable/dynamicrp/noncloud/baseresource_test.go` per research.md Decision 8: register a fresh resource type whose YAML uses `allOf: [{ $ref: "radius:base" }]` and declares only one type-specific property (e.g. `widgetSize`); deploy an instance that sets `environment`, `application`, `connections`, and `codeReference`; assert all four are honored by the runtime (resource IDs resolved correctly, connections returned, `codeReference` round-trips). Follow the `magpiego` pattern used by sibling tests in `test/functional-portable/dynamicrp/noncloud/`.

**Checkpoint**: `go test ./pkg/schema/... ./pkg/cli/manifest/... ./pkg/resourceutil/... ./pkg/dynamicrp/datamodel/... ./bicep-tools/...` passes. The new functional test (T015) is gated by the standard functional-test framework — confirmed to pass against a locally-built control plane per [quickstart.md](./quickstart.md).

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Ship the documentation deliverable required by FR-006 / SC-003. No code changes in this phase.

- [ ] T016 [P] Create `docs/contributing/contributing-code/contributing-code-base-resource-manifest.md` per research.md Decision 9 with the three sections from the spec's "Breaking Changes & Documentation Impact" section: (1) What changed, (2) Who is affected and what action they take, (3) How to author a new resource type using the base (including the `allOf:` vs `properties:` footgun call-out)
- [ ] T017 [P] Add a one-paragraph entry to the next release-note draft under `docs/release-notes/` pointing at the new contributor doc; call out the breaking change to the env-required validator rule and link to FR-006 / the new doc
- [ ] T018 [P] Run `make lint` (or the project's equivalent linter target) on the changed Go files and fix any lint findings
- [ ] T019 Run the markdown linters per the `radius-markdown-lint` skill on the new contributor doc and the updated spec / plan / research / data-model / quickstart / contracts files; fix any findings
- [ ] T020 Run `quickstart.md` end-to-end: build the CLI per `radius-build-cli`, run `rad resource-type create -f mywidget.yaml`, deploy an instance, assert the demo step-by-step passes. Record any divergence from quickstart in a follow-up edit to `quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Setup. Blocks every Phase 3 task. The `Apply()` function and the embedded `base.yaml` MUST exist before any other consumer can call it.
- **Phase 3 (US1)**: Depends on Phase 2. Within Phase 3, T006–T008 + T013–T014 are leaf edits with no cross-task dependency; T009–T012 depend on Phase 2 only; T015 (functional test) depends on T006, T009, T011, T013, and T014 all being in place
- **Phase 4 (Polish)**: Depends on Phase 3 — the doc describes the now-shipped behavior

### Within Phase 3

- T015 (functional test) is the integration point — depends on every code-edit task being done
- T010 depends on T009 (its `Apply()` call site)
- T012 depends on T011 (its `Apply()` call site)
- T007 depends on T006 (the inverted assertions reflect the new validator behavior)
- T014 depends on T013 (the test exercises the new accessor)

### Parallel Opportunities

- T003 in Phase 2 is parallelizable with T002 (different file)
- T005 (unit tests) parallelizes with T004 (implementation) only in the sense that the test file can be drafted first and watched fail; on a single-developer flow, write the test, watch it fail, then implement T004
- In Phase 3: T006, T007, T008, T013, T014 all touch different files and can be edited in parallel; T009 and T011 touch different files but both depend on Phase 2 — once Phase 2 lands they can be done in parallel
- All four Phase 4 tasks (T016–T019) touch different artifacts and can run in parallel

---

## Parallel Example: Phase 3 leaf edits

```bash
# After Phase 2 is in, these five can be edited in parallel:
Task: "Remove global env-required block in pkg/schema/validator.go"            # T006
Task: "Invert env-required tests in pkg/schema/validator_test.go"              # T007
Task: "Add codeReference to BasicProperties in pkg/resourceutil/utils.go"      # T008
Task: "Add CodeReference accessor to pkg/dynamicrp/datamodel/dynamicresource.go" # T013
Task: "Add Test_DynamicResource_CodeReference"                                 # T014
```

---

## Implementation Strategy

### MVP First (US1 only — the only in-scope story)

1. Complete Phase 1 (Setup — single git check)
2. Complete Phase 2 (Foundational — new package + base.yaml + Apply + its unit tests)
3. Complete Phase 3 (US1 — validator change, codeReference, CLI + bicep-tools wire-ups, runtime accessor, functional test)
4. **STOP and VALIDATE**: Run the new functional test (T015) — this is the spec's "Independent Test" for US1
5. Complete Phase 4 (Polish — docs + lint + quickstart walkthrough)

### Incremental Delivery

The work is small enough that all four phases land as a single PR. The natural commit boundaries are:

1. Phase 2 package + tests as one commit
2. Phase 3 validator change + tests as one commit
3. Phase 3 CLI wire-up + tests as one commit
4. Phase 3 bicep-tools wire-up + tests as one commit
5. Phase 3 runtime accessor + functional test as one commit
6. Phase 4 docs as one commit

---

## Notes

- [P] tasks = different files, no dependencies on an incomplete task
- The single user-story scope means no inter-story parallelization — the parallelization that matters is within Phase 3's leaf edits
- Functional test (T015) is the spec's measurable Independent Test for US1; it is the gate for declaring US1 done
- Documentation (T016–T017) is the deliverable that accompanies the breaking change (FR-006 / SC-003) — not optional
- Commit after each logical group (see Incremental Delivery above)
