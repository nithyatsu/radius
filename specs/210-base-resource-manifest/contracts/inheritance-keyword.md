# Contract: Inheritance keyword

This contract specifies the grammar, placement, resolution semantics, and error behavior of the user-facing inheritance keyword that this feature adds to the resource-type manifest YAML.

---

## Grammar

The keyword is the standard JSON-Schema / OpenAPI `$ref` whose value is a URI in a Radius-owned custom URI scheme (RFC 3986 permits custom schemes). There is exactly one legal value in v1:

```yaml
$ref: "radius:base"
```

`radius:` is the scheme; `base` is the scheme-specific part. The URI does NOT point at a network location — it is resolved purely lexically by `pkg/schema/baseresource/loader.go` to the schema embedded in `base.yaml`. The reference is to the **whole base schema** (a JSON-Schema object that declares the four common properties); the URI deliberately omits any JSON-Pointer fragment (no `#/properties` suffix), because the composition keyword (`allOf`) takes whole schemas, not properties-map subtrees.

---

## Placement

`$ref` is a sibling of `type:` / `properties:` / `required:`, used inside a JSON-Schema composition keyword. It is **NOT** a key inside `properties:` — `properties:` is keyed by property names, so putting `$ref` there would declare a property literally named `$ref`, which is not what we want and which `kin-openapi` would reject.

The canonical placement is `allOf` with a single `$ref` entry, sitting alongside the local `properties:` block:

```yaml
schema:
  type: object
  allOf:
    - $ref: "radius:base"               # inherits the four common Radius properties
  properties:
    widgetSize:                          # type-specific properties as usual
      type: integer
  required:
    - widgetSize
```

`allOf` semantics: an instance of the type must satisfy *every* sub-schema in the list AND the local schema. After `Apply()` resolves `radius:base`, the effective schema's `properties` map contains the union of the base's four properties and the local `widgetSize` property (with per-type-wins precedence on any conflict — see [data-model.md](../data-model.md)). The effective schema's `required:` array is the local one — the base contributes no entries to `required:`.

A schema that omits the `allOf` / `$ref` entry is a **raw** type — it gets none of the four common properties. This is the way an author publishes a type that intentionally does not participate in Radius's app/env model.

---

## Resolution

`pkg/schema/baseresource/loader.go::Apply(schema)`:

1. Walks `schema.AllOf` looking for an entry whose `Ref` field is a URI with the `radius:` scheme.
2. If found AND the value is exactly `"radius:base"`: drops that entry from `AllOf` and merges the base schema's `properties` map into the local schema's `properties` map using the per-type-wins composition rule from [data-model.md](../data-model.md).
3. If found but the value is any other `radius:` URI (unknown sub-resource): returns the error in the table below.
4. If no `radius:`-scheme `$ref` is present in `AllOf`: returns the schema unchanged. No injection occurs.
5. Returns the (possibly mutated) schema and any error.

Resolution is **purely lexical** — the resolver does NOT round-trip through any external URI fetcher, does NOT consult the network, and does NOT consult the UCP store. The `radius:` scheme is recognized only as the literal `radius:base`; no other `radius:` URI is legal in v1.

---

## Errors

The resolver MUST produce actionable command-time errors (FR-007 SHOULD; this contract upgrades to MUST for the cases below since they are pure-format errors that benefit from early surfacing):

| Input | Error |
|---|---|
| `$ref: "radius:base/something"` (unknown sub-resource) | `unsupported radius: $ref "radius:base/something" — only "radius:base" is supported in this version` |
| `$ref: "radius:"` (empty scheme-specific part) | same wording, with the offending value substituted |
| `$ref: "radius:base"` placed under `properties:` instead of `allOf:` | NOT this resolver's concern — `properties:` keys are property names, so this declares a property literally called `$ref`. `kin-openapi` rejects it with its own parse error; the per-type author sees that error and corrects the placement. The contributor doc (see [research.md](../research.md) Decision 9) calls out this footgun. |
| `allOf: [{ $ref: "radius:base" }]` AND a sibling property named `application` etc. that has an incompatible primitive type | falls through to the existing FR-007 check inside the validator — the per-type declaration is what is checked, not the `$ref`. |
| `$ref: "http://example.com/schema.json"` (not a `radius:` scheme) | NOT this resolver's concern — passed through to whatever the underlying `openapi3` library does (today: error from `kin-openapi`, which is acceptable). |

Errors MUST include the YAML file path and a path-pointer to the offending `$ref` location (e.g. `allOf[0]`), so the author can locate it quickly.

---

## Non-goals (explicitly out of scope)

- Resolving `$ref` to user-authored base manifests (e.g. `$ref: "file://./myorg-base.yaml#/properties"`). Spec § Out of Scope.
- Supporting partial inheritance (`$ref` that resolves to only some of the four properties). The keyword is all-or-nothing — it brings in all four.
- Supporting `extends:` (sub-mechanism A.2). Rejected in [research.md](../research.md) Decision 2.
- Caching the resolved base across calls. The base.yaml is embedded; reloading it per call is microsecond-cheap.
- Implicit injection (Approach B — no keyword in the YAML). Documented in [research.md](../research.md) Decision 1 as a possible future POC if this feature's keyword approach proves unacceptable in practice. Not pursued here.
