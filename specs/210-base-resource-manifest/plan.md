# Implementation Plan: Base Resource Manifest

**Branch**: `210-base-resource-manifest` | **Date**: 2026-06-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from [/specs/210-base-resource-manifest/spec.md](./spec.md)

## Summary

Introduce a single, repo-owned **base resource manifest** that declares the four common Radius properties (`application`, `environment`, `connections`, `codeReference`) so a resource-type author no longer has to repeat them in every per-type YAML. The schema validator's hardcoded global "environment is required" rule is removed (this is the documented breaking change; Radius is pre-1.0).

**OQ-001 is resolved as Approach A — user-facing `$ref` inheritance keyword.** The author writes `allOf: [{ $ref: "radius:base" }]` (sub-mechanism A.1) in the per-type YAML to opt the type into the four common properties; see [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for the exact grammar and placement. A future, separate POC can explore Approach B (implicit injection) if Approach A's discoverability cost or maintenance cost prove unacceptable — but the two are not pursued in parallel. This feature ships only Approach A.

`codeReference` is introduced in this feature as a new recognized common property (v1: optional string treated as a URI).

## Technical Context

**Language/Version**: Go (per existing Radius `go.mod` at repo root; honor `golang.instructions.md` conventions)
**Primary Dependencies**: existing in-repo packages only — `pkg/schema` (uses `github.com/getkin/kin-openapi/openapi3`), `pkg/cli/manifest` (YAML parser + validator), `pkg/dynamicrp/datamodel` (runtime adapters), `bicep-tools/pkg/converter` (Bicep emission). No new third-party dependencies.
**Storage**: N/A for the feature itself. The base manifest is a static YAML file shipped in the repo and loaded into memory. Resource-type registration continues to persist through the existing UCP store unchanged.
**Testing**: Go `testing` + `testify/require` for unit and integration; `magpiego` framework under `test/functional-portable/` for end-to-end. Existing converter integration test under `bicep-tools/test/integration_test.go` for the Bicep generator side.
**Target Platform**: Linux/macOS/Windows for the `rad` CLI; Linux (Kubernetes) for the control plane. No platform-specific code introduced.
**Project Type**: Cross-cutting Radius control-plane + CLI feature (Go) with a small additive contract on the `bicep-tools` Bicep extension generator.
**Performance Goals**: Manifest registration latency MUST NOT regress noticeably (base-manifest loading and `$ref` resolution is single-digit milliseconds and happens once per registration call). Runtime hot path is unaffected.
**Constraints**:
- The generated Bicep type wire format for a per-type YAML input that does NOT use `$ref` MUST be identical to today (regression-tested in `bicep-tools/pkg/converter/converter_test.go`).
- The set of reserved-and-forbidden property names (`status`, `recipe`) MUST stay unchanged.
- No backward-compatibility machinery (per FR-013): no per-type validator-version pinning, no schema snapshotting.
**Scale/Scope**: ~6–8 files touched; 1 new package (`pkg/schema/baseresource/`); 1 new YAML file (the base manifest); 1 new contributor doc; 1 new functional test.

## Constitution Check

*Gates from [.specify/memory/constitution.md](../../.specify/memory/constitution.md). Re-evaluated after Phase 1 — no new violations surfaced; Phase 1 preserved Phase 0 boundaries.*

| # | Principle | Status | Note |
|---|---|---|---|
| I | API-First Design | PASS | No new HTTP API. The manifest YAML schema is the contract and is captured in [contracts/base-manifest.schema.yaml](./contracts/base-manifest.schema.yaml) and [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md). |
| II | Idiomatic Code Standards (Go) | PASS | New `pkg/schema/baseresource/` follows existing package layout; gofmt, godoc on exported items, minimized surface area. |
| III | Multi-Cloud Neutrality | PASS | Pure schema/manifest feature; no cloud-provider touchpoints. |
| IV | Testing Pyramid (NON-NEGOTIABLE) | PASS | Unit: `pkg/schema/baseresource/loader_test.go`, updated `validator_test.go`, updated `validation_test.go`, updated `converter_test.go`. Integration: existing `pkg/cli/manifest/registermanifest_test.go` covers the CLI path. Functional: new `test/functional-portable/dynamicrp/noncloud/baseresource_test.go` registers a stripped manifest and deploys an instance. |
| V | Collaboration-Centric Design | PASS | Audience is resource-type authors (platform engineers + contributors). Developer experience improves indirectly via more consistent types; no new burden on application developers. |
| VI | Open Source / Community-First | PASS | Spec lives in `specs/`. Breaking change documented per FR-006 in a new contributor doc + release notes (see `docs/contributing/contributing-code/contributing-code-base-resource-manifest.md`). DCO sign-off applies to commits as usual. |
| VII | Simplicity Over Cleverness | PASS | Single implementation, single chokepoint, single new package. |
| VIII | Separation of Concerns & Modularity | PASS | The "make-base-manifest-part-of-effective-schema" logic is isolated in `pkg/schema/baseresource/`. |
| IX | Incremental Adoption & Backward Compat | PASS (with documented breaking change) | Pre-1.0 breaking change policy explicitly allows this. Documentation deliverable is in the plan; see FR-006 and the Breaking Changes section of the spec. |
| XII | Resource Type Schema Quality | PASS | Feature directly improves schema authoring ergonomics. |

Other principles (X, XI — Dashboard/TypeScript) do not apply to this feature.

## Project Structure

### Documentation (this feature)

```text
specs/210-base-resource-manifest/
├── spec.md                                                  # already exists
├── plan.md                                                  # this file
├── research.md                                              # Phase 0 (this command)
├── data-model.md                                            # Phase 1 (this command)
├── quickstart.md                                            # Phase 1 (this command)
├── contracts/
│   ├── base-manifest.schema.yaml                            # Phase 1 — what base.yaml must contain
│   └── inheritance-keyword.md                               # Phase 1 — $ref keyword grammar contract
├── checklists/
│   └── requirements.md                                      # already exists
└── tasks.md                                                 # Phase 2 — NOT generated by /speckit.plan
```

### Source code (repository root)

Files that change. New files marked `(NEW)`; everything else is an in-place edit.

```text
pkg/
├── schema/
│   ├── validator.go                                         # remove global env-required rule (lines 832–835)
│   ├── validator_test.go                                    # delete/invert the 3 env-required test cases
│   └── baseresource/                                        # (NEW package)
│       ├── base.yaml                                        # (NEW) the four common properties; embedded via go:embed
│       ├── loader.go                                        # (NEW) loads base.yaml; exposes Apply() that resolves $ref
│       └── loader_test.go                                   # (NEW) unit tests for the loader and resolver
├── resourceutil/
│   └── utils.go                                             # add "codeReference" to BasicProperties (line 28)
├── cli/
│   └── manifest/
│       ├── validation.go                                    # call baseresource.Apply() before validateManifestSchemas()
│       └── validation_test.go                               # new cases: $ref-using manifest validates; bad $ref errors
└── dynamicrp/
    └── datamodel/
        ├── dynamicresource.go                               # add CodeReference() accessor (mirrors EnvironmentID)
        └── dynamicresource_test.go                          # add Test_…_CodeReference

bicep-tools/
└── pkg/converter/
    ├── converter.go                                         # consume effective (post-Apply) schema
    └── converter_test.go                                    # assert four common props appear on $ref-using input

deploy/manifest/built-in-providers/
└── (unchanged in this feature)                              # cleanup of inline declarations is a follow-on

test/functional-portable/dynamicrp/noncloud/
└── baseresource_test.go                                     # (NEW) end-to-end: register $ref-using type, deploy, assert

docs/contributing/contributing-code/
└── contributing-code-base-resource-manifest.md              # (NEW) breaking-change notice + author how-to
```

**Structure Decision**: Use the existing Radius repository layout (no new top-level directories). Introduce one new package — `pkg/schema/baseresource/` — that owns the base manifest YAML file and exposes a single function `Apply(schema *openapi3.Schema) error` that resolves `$ref: "radius:base"` entries inside `allOf:` into the four common properties. Implementation happens on the existing `210-base-resource-manifest` branch.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| New package `pkg/schema/baseresource/` | The "apply base manifest to an effective schema" logic must live somewhere that both `pkg/cli/manifest/validation.go` and `bicep-tools/pkg/converter/converter.go` can call. A new small package gives a clean home for the loader, the embedded `base.yaml`, and the `$ref` resolver. | Inlining into either caller couples the two consumers. Putting it inside the validator conflates schema composition with validation (violates Principle VIII). |
