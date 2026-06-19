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

### User Story 2 - Override one common property without losing the others (Priority: P2)

An author needs to constrain or document one of the common properties on a specific type — for example, adding a description to `environment` for documentation. They declare only that one property in the type's YAML. The other three common properties remain inherited from the base manifest unchanged.

**Why this priority**: Without per-type override, the only way to customize any common property is to fall back to declaring all four, which defeats the feature. This is the most common reason an author touches these fields at all.

**Independent Test**: Author a resource type that explicitly declares one of the four common properties (e.g. a more constrained `connections` schema) and omits the other three. Register the type. Verify the overridden property uses the author's schema and the other three behave as if inherited from the base manifest.

**Acceptance Scenarios**:

1. **Given** a resource type YAML that explicitly declares `connections` with a constrained schema and omits `application`, `environment`, and `codeReference`, **When** the type is registered and a developer deploys an instance with a `connections` value that does not match the author's constrained schema, **Then** the deployment is rejected with the author's constraint surfaced in the error.
2. **Given** the same type from scenario 1, **When** the developer deploys an instance setting `application` and `environment`, **Then** those values are accepted and behave as they do on any other Radius type.

---

### User Story 3 - Existing resource types keep working unchanged (Priority: P1)

The contributors who maintain the resource types under `pkg/resourcetypescontrib/` and the built-in types under `deploy/manifest/built-in-providers/` should not have to touch their existing YAML files for this feature to ship. Older author YAML that still explicitly declares `application`, `environment`, and `connections` must continue to validate and register exactly as it does today.

**Why this priority**: This is a backward-compatibility guarantee. Breaking the existing manifests in this repository or in any out-of-tree resource type would make the feature unshippable.

**Independent Test**: Without modifying any existing manifest YAML in `pkg/resourcetypescontrib/` or `deploy/manifest/`, run the standard manifest validation and registration paths for every existing built-in and contrib type. Verify every type registers exactly as it does on `main`.

**Acceptance Scenarios**:

1. **Given** the set of resource type manifests that exist in the repo before this feature lands, **When** they are validated and registered after the feature is enabled, **Then** every type registers successfully with no warnings or errors that weren't present before.
2. **Given** an existing manifest that declares `environment` as required at the per-type level, **When** a deployment omits `environment`, **Then** the deployment is rejected (the per-type declaration wins over the base manifest's optional-by-default).

---

### Edge Cases

- A per-type YAML declares one of the four common properties with a schema that is **incompatible** with the base manifest's shape (e.g. declares `connections` as a string instead of a map). The author should get a clear, command-time error that names the offending property and explains the conflict — not a confusing runtime validation failure later.
- A per-type YAML attempts to declare a property named `status` or `recipe` (already forbidden today). Behavior is unchanged: the registration fails with the existing reserved-property error. The base manifest does **not** silently permit those.
- A per-type YAML uses an exact-but-differently-cased spelling such as `Application` or `CodeReference`. The author should get a clear error or warning rather than silently creating a new property that shadows the inherited common property.
- The user attempts to deploy a resource that omits `environment` for a type that did not opt out of the base manifest. The deployment is accepted (environment is optional from the author's contract) and Radius applies its existing default / scope-based environment resolution behavior.
- An out-of-tree resource type built against an older Radius CLI is registered against a newer control plane that has this feature. The control plane treats the four properties exactly as before for that type — i.e. backward compatibility holds in both directions.
- Two different types in the **same** manifest YAML interact differently with the common properties (one overrides `connections`, one does not). Each type's effective schema is computed independently; the override on one does not leak into the other.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Resource type authors MUST be able to register a resource type whose manifest YAML declares **none** of `application`, `environment`, `connections`, or `codeReference`, and have the resulting registered type still accept and validate those four properties at deployment time with their standard Radius semantics.
- **FR-002**: For any resource type that does not explicitly declare them, all four common properties (`application`, `environment`, `connections`, `codeReference`) MUST be **optional** at deployment time — deployments that omit any or all of them MUST be accepted.
- **FR-003**: Authors MUST be able to override exactly one of the four common properties on a per-type basis — declaring it explicitly in the type's YAML — without having to redeclare the other three.
- **FR-004**: An explicit per-type declaration of one of the four common properties MUST take precedence over the corresponding declaration in the base resource manifest. (E.g. an author who declares `environment` as required at the type level keeps that requirement.)
- **FR-005**: `codeReference` MUST be introduced as a recognized common property in this feature. Its v1 shape is an **optional string treated as a URI** that points back to authoring source (e.g. a Git URL with commit SHA and line range). Radius validates that the value is a string at registration time; it does not enforce a specific URI format. Tooling that surfaces resource metadata (graph, status, `rad resource show`) MAY render the value as a clickable link when it parses as an HTTP(S) URL. Richer structured shapes (e.g. `{repo, commit, path, line}`) are an additive future change that can be introduced without breaking the v1 string wire format.
- **FR-006**: Manifest YAML that exists in this repository (under `pkg/resourcetypescontrib/` and `deploy/manifest/built-in-providers/`) and any out-of-tree manifest that already declares the four common properties explicitly MUST continue to validate and register without modification. Deployment-time behavior for these existing types (including "environment is required" for types whose YAML declares it) MUST NOT change as a side effect of this feature — their behavior is preserved by FR-004's per-type-explicit-wins rule, not by any per-type validator-version tracking.
- **FR-007**: When a per-type YAML declares one of the four common properties with a schema that is **incompatible** with the shared base (e.g. wrong primitive type), the author MUST receive an actionable command-time error from `rad resource-type create` that names the offending property and explains the conflict — the registration MUST NOT silently succeed.
- **FR-008**: The set of reserved property names that authors are **forbidden** to use as their own custom properties (today: `status`, `recipe`) MUST be preserved unchanged. The base resource manifest MUST NOT introduce a new collision class.
- **FR-009**: The semantics of each of the four common properties at deployment / runtime MUST be unchanged from today:
  - `application` and `environment` continue to resolve to Radius resource IDs via the existing application/environment ID adapters.
  - `connections` continues to be extracted as the map of connection name → source resource ID by the existing connection-extraction path.
  - `codeReference` is treated as an optional URI string surfaced through the same mechanism wherever resource metadata is exposed (graph, status, etc.). See FR-005 for its v1 shape.
- **FR-010**: Tooling that consumes the resource type manifest to produce client artifacts (i.e. the Bicep extension generator) MUST emit a working Bicep type definition for a resource type that does not explicitly declare the four common properties — i.e. an author writing only type-specific properties in YAML MUST still get a Bicep type where a consumer can set `application`, `environment`, `connections`, and `codeReference` on a resource of that type.
- **FR-011**: The `rad resource-type create` command MUST make the base resource manifest available to every registration **without the author passing an additional flag or file path** at the command line — `rad resource-type create -f <file>.yaml` (with no extra arguments) MUST be sufficient. Whether the author's YAML itself must contain a keyword that references the base (e.g. `$ref`, `extends:`) or whether the inclusion is invisible to the YAML is the user-experience choice tracked by **OQ-001** in **Open Questions** below — both options satisfy this requirement.
- **FR-012**: The base resource manifest's definition (the source of truth for the shape and behavior of the four common properties) MUST live at one well-known location in the repository, shipped with Radius. The set of properties in the base is **fixed at the four named in this spec** (`application`, `environment`, `connections`, `codeReference`) — future Radius releases MUST NOT silently add, remove, rename, or change the type of a property in this base (see Clarifications § Session 2026-06-19). Out-of-tree resource type authors MUST NOT need to fetch or co-author the base themselves.
- **FR-013**: The prototype implementation MUST NOT introduce per-type validator-version tracking, schema snapshotting, re-registration triggers, or any other mechanism whose only purpose is to preserve the old "env required" behavior for already-registered types (see Clarifications § Session 2026-06-19). The new validator behavior MUST be a simple uniform code change; backward compatibility for existing types is delivered entirely through FR-004 (per-type explicit declarations win) operating on existing types' already-present `environment` declarations.

### Open Questions

The feature has two viable approaches that differ primarily in **how visible the base manifest is to the author**. The choice is significant enough to call out as a clarification rather than guess. Both approaches are tracked as open until a decision is made — neither has been pre-selected.

- **OQ-001**: [NEEDS CLARIFICATION: Should the base resource manifest be **user-facing** (approach A: the author writes a keyword in their YAML that references the base) or **non-user-facing** (approach B: the base is applied invisibly by Radius with no keyword in the author's YAML at all)? The two approaches are equally aligned with the goal of removing per-type boilerplate; they differ on whether the inheritance is visible to anyone reading a type's YAML.

  - **Approach A — user-facing / explicit inheritance**. The author's YAML contains a keyword that opts the type into the base manifest. Common properties only appear on the type if the author opts in. Authors can publish a "raw" type by omitting the keyword. Two candidate mechanisms for the keyword have been considered:
    - **A.1 — `$ref` with a Radius-owned scheme**, e.g. `$ref: "radius:base#/properties"`. Reuses standard JSON-Schema vocabulary that schema-literate authors and tooling already understand. Cost: requires defining the `radius:` scheme and how the CLI / control plane resolves it. *(Recommended sub-mechanism if Approach A is chosen.)*
    - **A.2 — a Radius-specific keyword**, e.g. `extends: Radius.Core/baseResource` or `inherit: …`. More opinionated and more obviously "a Radius thing" in docs; no URI-resolution machinery to design. Cost: non-standard syntax that generic JSON-Schema tooling does not recognize.
    - **Considered and rejected**: making `Radius.Core/baseResource` a real registered runtime resource type that application developers instantiate in Bicep. That conflates schema composition ("does this type understand `environment`?") with runtime data ("which environment is this widget bound to?"), adds boilerplate to every Bicep file instead of removing it, and does not address the resource-type author's pain point that motivates this feature.

  - **Approach B — non-user-facing / implicit injection**. There is no keyword anywhere in the author's YAML. Every resource type that goes through `rad resource-type create` automatically picks up the four common properties as optional, and the author writes only type-specific properties. The Bicep extension generator and the control plane validator both already know the four properties exist on every type. The author cannot opt out, but can still **override** individual common properties per FR-003 / FR-004 by declaring them explicitly.

  - **What is the same in both approaches**: FR-001 through FR-010 hold identically. The four common properties become optional. The runtime semantics do not change. Existing manifests keep working without edits. The set of forbidden reserved names (`status`, `recipe`) is unchanged.

  - **What differs**: the wording of FR-011 (whether the author's YAML must contain an inheritance keyword); whether a new manifest-syntax token (`$ref` resolution scheme, `extends:`) becomes a permanent part of the format and has to be documented; the migration story for in-repo manifests that currently declare the four properties inline (they could be cleaned up in either approach, but only Approach A gives them a syntactic marker showing they now inherit); and discoverability — Approach A is self-documenting in the YAML, Approach B requires the reader to consult Radius docs to know the four properties exist.]

### Key Entities

- **Base resource manifest**: a single, repo-owned definition of the four common properties (`application`, `environment`, `connections`, `codeReference`) and how Radius treats them. The source of truth for what "every Radius resource type knows how to do."
- **Resource type manifest** (existing): the per-type YAML an author writes to declare a resource type's name, API version, and schema. After this feature, this YAML may declare the four common properties only when it needs to override them.
- **Common Radius property** (new term, applies to: `application`, `environment`, `connections`, `codeReference`): a schema property whose presence, name, and runtime semantics are defined by Radius itself rather than by the resource type author. Contrasted with **type-specific property** (everything else the author declares).
- **Reserved property name** (existing, unchanged): a property name authors are forbidden to use (today: `status`, `recipe`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new resource type can be authored, registered, and deployed end-to-end with a manifest YAML that contains **zero** lines mentioning `application`, `environment`, `connections`, or `codeReference`. (Measured by line-grep on a representative new-type YAML against an end-to-end functional test that deploys an instance of the type and asserts the four common properties behave correctly.)
- **SC-002**: For a newly authored resource type, the author's YAML for the schema section is at least **15 lines shorter** than the equivalent YAML would be today (measuring on a representative single-property type — i.e. the boilerplate four-property block is gone).
- **SC-003**: Every resource type manifest YAML that exists in the repository on the commit before this feature lands continues to register successfully on the commit after, with **zero** required edits to those YAML files. (Measured by running existing manifest validation against the unmodified files.)
- **SC-004**: A deployment that omits all four common properties on a type that did not explicitly require any of them is accepted by the control plane in **100%** of cases. (Measured by a deployment test that exercises this path.)
- **SC-005**: When an author writes a per-type schema for one of the four common properties whose shape is incompatible with the base (e.g. declares `environment` as an integer), `rad resource-type create` rejects the registration with an actionable error message **at command time**, before any control-plane round-trip. (Measured by a CLI test that asserts the error path is surfaced locally.)
- **SC-006**: Documentation that explains how to author a new resource type drops the "you must declare these four properties" section, replaced by a single one-sentence note that Radius supplies them — i.e. the author-facing how-to is measurably shorter.

## Assumptions

- The four common properties named in the input (`application`, `codeReference`, `connections`, `environment`) are the complete initial scope. `status` and `recipe` remain reserved-and-forbidden (not common). Any expansion of the common set (e.g. promoting more properties to "common") is a follow-on feature, not in scope here.
- Today's behavior of `environment` being **schema-required** on every type is treated as boilerplate that this feature explicitly removes from the **validator's hardcoded global rule**: after this feature, the validator does not enforce a universal env-required check. Per-type schemas that explicitly declare `environment` (which is every existing type today) keep requiring it via FR-004. New types authored without declaring `environment` get the base manifest's optional-by-default behavior. (If preserving today's per-type "environment is required" behavior is desired for a new type, the author can simply declare it explicitly per FR-004.)
- The prototype is **scoped to the new authoring experience** — it deliberately ships no migration tooling, no per-type validator-version tracking, and no retroactive re-evaluation of stored types' schemas. The user-visible change for existing apps deploying existing types is **zero**, because all existing in-repo and out-of-tree types declare `environment` in their YAML and that declaration is preserved by FR-004. This kept the implementation small and the rollout boring.
- "Inheritance" is used informally in the feature description; the spec treats the relationship as "the base manifest contributes properties to a per-type effective schema." Whether the implementation uses JSON-Schema `allOf`, runtime composition, code-generated injection, or another mechanism is a design decision and is out of scope for this spec.
- The introduction of `codeReference` as a recognized Radius property is part of this feature's scope. Its v1 shape — an **optional string treated as a URI** — is fixed by FR-005 (see Clarifications § Session 2026-06-19). Defining a richer structured form (`{repo, commit, path, line}`, etc.) is deferred to a future additive feature.
- The bicep-tools generator's existing behavior of injecting the standard envelope (`name`, `location`, `properties`, `apiVersion`, `type`, `id`) on every type is unchanged. The base resource manifest applies to the **schema properties under `properties:`**, not to that envelope.

## Out of Scope

- Defining a richer structured form for `codeReference` (e.g. a `{repo, commit, path, line}` object). v1 ships as an optional URI string (FR-005); a structured form is an additive future feature.
- Promoting any property other than the four named ones (e.g. `provisioningState`, `secrets`, `tags`) to the "common Radius property" set. The base is frozen at four properties (FR-012); any future expansion is a separate feature and a separate mechanism, not an evolution of this base.
- Changing the existing reserved-and-forbidden list (`status`, `recipe`).
- Defining a generic resource-type-manifest inheritance mechanism that authors could use for **their own** shared blocks (e.g. one author defining a "myorg-base" YAML and reusing it across several of their types). This feature ships only the single Radius-owned base manifest; user-defined bases can be a follow-on.
- Migration tooling that rewrites existing manifest YAML to remove the now-redundant declarations of the four common properties. Existing manifests keep working as-is; cleanup is optional and manual.
- Changes to the consumer-side Bicep authoring experience beyond what falls out of the schema change automatically — no new author-time Bicep keywords, no new CLI commands for end-application developers.
