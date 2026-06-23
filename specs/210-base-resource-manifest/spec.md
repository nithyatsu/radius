# Feature Specification: Base Resource Manifest

**Feature Branch**: `210-base-resource-manifest`
**Created**: 2026-06-19
**Status**: Draft
**Input**: User description: "Add a 'base resource manifest' feature so a shared YAML file (e.g. baseresource.yaml) can supply a common set of schema properties — like `application`, `codeReference`, `connections`, and `environment` — that get inherited by every resource type defined in a Radius resource-type manifest. Users shouldn't have to supply these every time they need it."

## Purpose

Today, every Radius resource type a contributor authors must repeat the same boilerplate set of "Radius-aware" schema properties (`application`, `environment`, `connections`, and — proposed — `codeReference`) in its manifest YAML. One of them (`environment`) is also required by the schema validator, so every manifest must declare it explicitly even though Radius itself knows how to populate it. This is unnecessary repetition that contributors get wrong, and it leaks Radius framework concerns into every per-type schema.

This feature introduces a single, shared **base resource manifest** for these common Radius properties so that:

- A contributor authoring a new resource type does not have to redeclare `application`, `environment`, `connections`, or `codeReference` in every type they define.
- All four of those properties are **optional** from the contributor's point of view — the contributor opts in only when they want to customize, document, or constrain one of them.
- The properties continue to behave exactly the way Radius treats them today (e.g. `environment` and `application` resolve to Radius resource IDs, `connections` is the map of connection name → source ID, `codeReference` is the new per-resource pointer back to authoring source).
- Existing per-type manifests that still declare these properties explicitly continue to work without change.

The feature is for **resource type authors** (contributors and platform engineers publishing types via `rad resource-type create` or via the built-in providers loaded at install time). End-application developers using the resulting types in Bicep/Helm/etc. are not the audience for this feature, but they benefit indirectly because the types they consume become more consistent.

## Clarifications

### Session 2026-06-19

- Q: What shape does the `codeReference` common property have in v1? → A: String treated as a URI (e.g. `https://github.com/org/repo/blob/<sha>/path/file.bicep#L42`). Radius validates it as a string at registration time; richer structured shapes can be a later additive change without breaking the wire format.
- Q: How does the base manifest evolve over time? → A: Frozen forever — the base ships with exactly the four common properties named in this spec (`application`, `environment`, `connections`, `codeReference`) and is not extended by future Radius releases. Promoting any additional property to common status requires a separate mechanism (a new spec / a parallel base / a different feature), not an evolution of this base.
- Q: When this feature ships, do already-registered types' deployments get the new env optionality, or stay as they are? → A: Scope the prototype to new authoring experiences — do **not** build per-type validator versioning, snapshotting, or migration tooling. The new "env optional" behavior applies uniformly via the updated validator, but in practice only flows to types whose YAML does not declare `environment` (i.e. types authored after this feature). Existing types' YAML continues to declare `environment`, so FR-004 (per-type explicit declaration wins) preserves their current require-env behavior at deployment time without any additional engineering. Edge cases involving stored types that somehow lack an explicit `environment` declaration are out of scope for the prototype.
- Q: Are User Story 2 (per-type override of a single common property) and User Story 3 (existing resource types keep working unchanged) in scope for the prototype? → A: No. Radius is in incubation and does not guarantee backward compatibility, so the prototype explicitly accepts breaking changes for in-repo and out-of-tree resource types. User Story 2 is **deferred** to a follow-on feature. User Story 3 is **dropped**: backward compatibility is no longer a requirement. The breaking change itself MUST be captured in the feature's documentation (see new "Breaking Changes & Documentation Impact" section).
- Q: OQ-001 (user-facing vs. non-user-facing inheritance) — which approach does this feature ship? → A: **Approach A — user-facing `$ref`**. The author writes `allOf: [{ $ref: "radius:base" }]` (sub-mechanism A.1) in the per-type YAML to opt the type into the base. See [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for the keyword's grammar and placement. Approach B (implicit injection) is preserved as a possible future POC if Approach A proves unacceptable in practice — but the two are not pursued in parallel. OQ-001 is closed by this answer.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Author a new resource type without restating Radius boilerplate (Priority: P1)

A platform engineer is publishing a new resource type (e.g. `MyOrg.Examples/widgets`). Today they have to copy the standard `application`, `environment`, and `connections` property declarations into the type's YAML schema, get the `environment` field marked required, and remember exactly how each is spelled — or schema validation fails. With this feature, they author only the properties that are unique to widgets (e.g. `size`, `color`, `replicaCount`). The four common Radius properties are supplied by the base resource manifest. Registration succeeds. A consumer can then write a Bicep resource of type `MyOrg.Examples/widgets` and set `environment`, `application`, and `connections` exactly as they would on any other Radius resource.

**Why this priority**: This is the core value of the feature. If new resource type authors still have to hand-write these four properties, the feature has not delivered.

**Independent Test**: Author a resource type whose YAML declares only type-specific properties. Run `rad resource-type create` against it. Verify registration succeeds, verify the registered type accepts and validates `environment`, `application`, `connections`, and `codeReference` from a deployment, and verify that values set on those properties at deployment time are honored by Radius (e.g. `connections` resolves source resource IDs the same way it does for built-in types).

**Acceptance Scenarios**:

1. **Given** a resource type manifest YAML that declares only type-specific properties (no `application`, `environment`, `connections`, or `codeReference`), **When** the author runs `rad resource-type create -f <file>.yaml`, **Then** registration succeeds without a "missing required property `environment`" error.
2. **Given** a registered type authored without declaring the four common properties, **When** a developer deploys an instance of the type that sets `environment`, `application`, and `connections`, **Then** Radius accepts the deployment and the runtime treats those fields with the same semantics as it does on built-in types.
3. **Given** a registered type authored without declaring the four common properties, **When** a developer deploys an instance that omits all four optional common properties, **Then** the deployment is accepted (the four are optional and Radius applies its existing defaults / lookup behavior for absent values).

---

### Deferred Stories & Breaking Changes

The following stories were considered and removed from the prototype scope (see Clarifications § Session 2026-06-19). They are documented here so a future contributor can pick them up without re-discovering the trade-off.

- **Deferred — per-type override workflow** (was User Story 2, P2). An author overriding exactly one common property on a specific type (e.g. constraining `connections` to a closed set of named keys) is **not** a prototype goal. The underlying mechanism that makes per-type explicit declarations win is still present (see FR-004), but the prototype does not invest in command-time conflict diagnostics, override-shape validation, or supporting documentation for this workflow. A follow-on feature is expected to restore this story.
- **Dropped — backward compatibility for existing resource types** (was User Story 3, P1). Radius is in incubation and ships breaking changes between releases. This feature ships a breaking change to the validator's contract for any resource type whose author relied on the global "environment is required" rule. There is no engineering invested in preserving that contract for existing types and no migration tooling. The deliverable that accompanies the breaking change is **documentation** (see "Breaking Changes & Documentation Impact" below), not preservation.

---

### Edge Cases

- A per-type YAML attempts to declare a property named `status` or `recipe` (already forbidden today). Behavior is unchanged: the registration fails with the existing reserved-property error. The base manifest does **not** silently permit those.
- A per-type YAML uses an exact-but-differently-cased spelling such as `Application` or `CodeReference`. The author should get a clear error or warning rather than silently creating a new property that shadows the inherited common property.
- The user attempts to deploy a resource that omits `environment` for a type that did not opt out of the base manifest. The deployment is accepted (environment is optional from the author's contract) and Radius applies its existing default / scope-based environment resolution behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Resource type authors MUST be able to register a resource type whose manifest YAML declares **none** of `application`, `environment`, `connections`, or `codeReference`, and have the resulting registered type still accept and validate those four properties at deployment time with their standard Radius semantics.
- **FR-002**: For any resource type that does not explicitly declare them, all four common properties (`application`, `environment`, `connections`, `codeReference`) MUST be **optional** at deployment time — deployments that omit any or all of them MUST be accepted.
- **FR-003**: *(Deferred from prototype.)* The mechanism that lets an author override exactly one of the four common properties on a per-type basis is delivered structurally by FR-004 (explicit declarations win), but the prototype does not commit to a polished override workflow — no command-time conflict diagnostics, no override-shape validation, no supporting docs are required. A follow-on feature restores this as a first-class story.
- **FR-004**: An explicit per-type declaration of one of the four common properties MUST take precedence over the corresponding declaration in the base resource manifest. (E.g. an author who declares `environment` as required at the type level keeps that requirement.)
- **FR-005**: `codeReference` MUST be introduced as a recognized common property in this feature. Its v1 shape is an **optional string treated as a URI** that points back to authoring source (e.g. a Git URL with commit SHA and line range). Radius validates that the value is a string at registration time; it does not enforce a specific URI format. Tooling that surfaces resource metadata (graph, status, `rad resource show`) MAY render the value as a clickable link when it parses as an HTTP(S) URL. Richer structured shapes (e.g. `{repo, commit, path, line}`) are an additive future change that can be introduced without breaking the v1 string wire format.
- **FR-006**: This feature is permitted to be a **breaking change** for any existing in-repo or out-of-tree resource type manifest. Radius is in incubation and does not guarantee backward compatibility (see Clarifications § Session 2026-06-19). The implementation MUST NOT invest engineering in preserving the validator's previous global "environment is required" rule for existing types, but it MUST produce documentation describing what changed (see "Breaking Changes & Documentation Impact" section below).
- **FR-007**: When a per-type YAML declares one of the four common properties with a schema that is **incompatible** with the shared base (e.g. wrong primitive type), the author SHOULD receive an actionable command-time error from `rad resource-type create` that names the offending property and explains the conflict — the registration MUST NOT silently succeed. (This requirement is softened to SHOULD-quality because the polished override workflow it serves is deferred per FR-003; the must-not-silently-succeed half is retained as a correctness guard.)
- **FR-008**: The set of reserved property names that authors are **forbidden** to use as their own custom properties (today: `status`, `recipe`) MUST be preserved unchanged. The base resource manifest MUST NOT introduce a new collision class.
- **FR-009**: The semantics of each of the four common properties at deployment / runtime MUST be unchanged from today:
  - `application` and `environment` continue to resolve to Radius resource IDs via the existing application/environment ID adapters.
  - `connections` continues to be extracted as the map of connection name → source resource ID by the existing connection-extraction path.
  - `codeReference` is treated as an optional URI string surfaced through the same mechanism wherever resource metadata is exposed (graph, status, etc.). See FR-005 for its v1 shape.
- **FR-010**: Tooling that consumes the resource type manifest to produce client artifacts (i.e. the Bicep extension generator) MUST emit a working Bicep type definition for a resource type that does not explicitly declare the four common properties — i.e. an author writing only type-specific properties in YAML MUST still get a Bicep type where a consumer can set `application`, `environment`, `connections`, and `codeReference` on a resource of that type.
- **FR-011**: The `rad resource-type create` command MUST make the base resource manifest available to every registration **without the author passing an additional flag or file path** at the command line — `rad resource-type create -f <file>.yaml` (with no extra arguments) MUST be sufficient. The author's YAML opts into the base by adding `allOf: [{ $ref: "radius:base" }]` to the per-type schema (per OQ-001 resolution in Clarifications § Session 2026-06-19; see [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for the exact grammar and placement). A YAML that omits the keyword publishes a "raw" type that does not inherit the four common properties.
- **FR-012**: The base resource manifest's definition (the source of truth for the shape and behavior of the four common properties) MUST live at one well-known location in the repository, shipped with Radius. The set of properties in the base is **fixed at the four named in this spec** (`application`, `environment`, `connections`, `codeReference`) — future Radius releases MUST NOT silently add, remove, rename, or change the type of a property in this base (see Clarifications § Session 2026-06-19). Out-of-tree resource type authors MUST NOT need to fetch or co-author the base themselves.
- **FR-013**: The prototype implementation MUST NOT introduce per-type validator-version tracking, schema snapshotting, re-registration triggers, or any other backward-compatibility machinery. The new validator behavior MUST be a simple uniform code change. Any divergence in behavior for existing in-repo or out-of-tree types is acceptable, is recorded in the Breaking Changes section below, and is communicated via documentation rather than preserved via code.

### Open Questions

None open. OQ-001 was resolved in Clarifications § Session 2026-06-19 to **Approach A — user-facing `$ref`** (sub-mechanism A.1: `allOf: [{ $ref: "radius:base" }]`; see [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md)). Approach B (implicit injection) is documented in [research.md](./research.md) Decision 1 as a possible future POC if Approach A proves unacceptable, but is not pursued in this feature.

### Key Entities

- **Base resource manifest**: a single, repo-owned definition of the four common properties (`application`, `environment`, `connections`, `codeReference`) and how Radius treats them. The source of truth for what "every Radius resource type knows how to do."
- **Resource type manifest** (existing): the per-type YAML an author writes to declare a resource type's name, API version, and schema. After this feature, this YAML may declare the four common properties only when it needs to override them.
- **Common Radius property** (new term, applies to: `application`, `environment`, `connections`, `codeReference`): a schema property whose presence, name, and runtime semantics are defined by Radius itself rather than by the resource type author. Contrasted with **type-specific property** (everything else the author declares).
- **Reserved property name** (existing, unchanged): a property name authors are forbidden to use (today: `status`, `recipe`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new resource type can be authored, registered, and deployed end-to-end with a manifest YAML that contains **zero** lines mentioning `application`, `environment`, `connections`, or `codeReference`. (Measured by line-grep on a representative new-type YAML against an end-to-end functional test that deploys an instance of the type and asserts the four common properties behave correctly.)
- **SC-002**: For a newly authored resource type, the author's YAML for the schema section is at least **15 lines shorter** than the equivalent YAML would be today (measuring on a representative single-property type — i.e. the boilerplate four-property block is gone).
- **SC-003**: The feature ships with a documented breaking-change notice (in the changelog or release notes, and in contributor docs) that names every contract that changed (today: the validator's global "environment required" rule), states who is affected, and lists what action — if any — each affected author must take. (Measured by the presence of these items in the PR's documentation diff at merge time.)
- **SC-004**: A deployment that omits all four common properties on a type that did not explicitly require any of them is accepted by the control plane in **100%** of cases. (Measured by a deployment test that exercises this path.)
- **SC-005**: When an author writes a per-type schema for one of the four common properties whose shape is incompatible with the base (e.g. declares `environment` as an integer), `rad resource-type create` rejects the registration with an actionable error message **at command time**, before any control-plane round-trip. (Measured by a CLI test that asserts the error path is surfaced locally.)
- **SC-006**: Documentation that explains how to author a new resource type drops the "you must declare these four properties" section, replaced by a single one-sentence note that Radius supplies them — i.e. the author-facing how-to is measurably shorter.

## Assumptions

- The four common properties named in the input (`application`, `codeReference`, `connections`, `environment`) are the complete initial scope. `status` and `recipe` remain reserved-and-forbidden (not common). Any expansion of the common set (e.g. promoting more properties to "common") is a follow-on feature, not in scope here.
- **Radius is in incubation and this feature is permitted to ship breaking changes** for existing in-repo and out-of-tree resource types. No backward compatibility is promised, no migration tooling is shipped, and no per-type behavior preservation is engineered. The deliverable that accompanies the breaking change is **documentation** (see "Breaking Changes & Documentation Impact" section).
- After this feature, the validator does not enforce a universal "environment is required" check. Per-type schemas that explicitly declare `environment` keep requiring it via FR-004. New types authored without declaring `environment` get the base manifest's optional-by-default behavior.
- "Inheritance" is used informally in the feature description; the spec treats the relationship as "the base manifest contributes properties to a per-type effective schema." Whether the implementation uses JSON-Schema `allOf`, runtime composition, code-generated injection, or another mechanism is a design decision and is out of scope for this spec.
- The introduction of `codeReference` as a recognized Radius property is part of this feature's scope. Its v1 shape — an **optional string treated as a URI** — is fixed by FR-005 (see Clarifications § Session 2026-06-19). Defining a richer structured form (`{repo, commit, path, line}`, etc.) is deferred to a future additive feature.
- The bicep-tools generator's existing behavior of injecting the standard envelope (`name`, `location`, `properties`, `apiVersion`, `type`, `id`) on every type is unchanged. The base resource manifest applies to the **schema properties under `properties:`**, not to that envelope.
- The introduction of `codeReference` as a recognized Radius property is part of this feature's scope. Its v1 shape — an **optional string treated as a URI** — is fixed by FR-005 (see Clarifications § Session 2026-06-19). Defining a richer structured form (`{repo, commit, path, line}`, etc.) is deferred to a future additive feature.
- The bicep-tools generator's existing behavior of injecting the standard envelope (`name`, `location`, `properties`, `apiVersion`, `type`, `id`) on every type is unchanged. The base resource manifest applies to the **schema properties under `properties:`**, not to that envelope.

## Out of Scope

- Defining a richer structured form for `codeReference` (e.g. a `{repo, commit, path, line}` object). v1 ships as an optional URI string (FR-005); a structured form is an additive future feature.
- Promoting any property other than the four named ones (e.g. `provisioningState`, `secrets`, `tags`) to the "common Radius property" set. The base is frozen at four properties (FR-012); any future expansion is a separate feature and a separate mechanism, not an evolution of this base.
- Changing the existing reserved-and-forbidden list (`status`, `recipe`).
- Defining a generic resource-type-manifest inheritance mechanism that authors could use for **their own** shared blocks (e.g. one author defining a "myorg-base" YAML and reusing it across several of their types). This feature ships only the single Radius-owned base manifest; user-defined bases can be a follow-on.
- Migration tooling that rewrites existing manifest YAML to remove the now-redundant declarations of the four common properties. Existing manifests are left untouched; cleanup is optional and manual.
- Changes to the consumer-side Bicep authoring experience beyond what falls out of the schema change automatically — no new author-time Bicep keywords, no new CLI commands for end-application developers.

## Breaking Changes & Documentation Impact

This feature is permitted to break existing in-repo and out-of-tree resource type manifests (see Clarifications § Session 2026-06-19 and the Assumptions section). It ships **documentation**, not preservation. The following items MUST be captured in the feature's documentation diff at merge time — they are the deliverable that accompanies the breaking change.

- **What changed**: The schema validator's hardcoded global rule that `environment` is required on every resource type schema has been removed. The base resource manifest now declares `application`, `environment`, `connections`, and `codeReference` as optional by default; the per-type schema can override any of them (FR-004).
- **Who is affected**:
  - Resource type authors whose YAML declared `environment` *only* because the previous validator required it (i.e. they did not actually want it required at deployment time) — their types now have a more permissive deployment contract than the original YAML may have implied.
  - Out-of-tree tooling that hard-coded an assumption that every Radius type schema includes a required `environment` property.
- **What action affected authors take**:
  - If you want `environment` to remain required for your type after this feature: explicitly declare it as required in your YAML (FR-004 makes that declaration win).
  - If you are happy with the new optional behavior: no action needed.
  - If your tooling assumed env-required: update it to treat `environment` as optional on incoming type schemas.
- **What is NOT promised**: There is no per-type validator-version pinning, no fallback path for older types, and no automated migration. Out-of-tree CLIs built against the old contract MAY produce warnings or surprising behavior; fixing those is a follow-on if needed.
- **Where this is documented**: The change MUST appear in (a) the release notes / changelog for the Radius release that ships this feature, and (b) the contributor-facing resource-type authoring docs under `docs/contributing/` (whatever doc currently tells authors that `environment` is required is updated to remove or invert that statement).
