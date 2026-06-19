# Authoring base properties into a resource type manifest

> Status: Available in Radius v0.55 (or newer) — covers feature spec [specs/210-base-resource-manifest](/specs/210-base-resource-manifest/).

This page is for **resource type authors** writing a YAML manifest for `rad resource-type create`. It describes the *base resource manifest* — a small, frozen set of common Radius properties that you can opt into instead of restating in every manifest.

## What changed

Radius previously **required** every resource type schema to include the `environment` property. This requirement has been removed.

In its place, Radius now publishes a single **base resource manifest** that contributes the following four properties, all optional, to any resource type that opts in:

- `application` (string) — the application this resource belongs to.
- `environment` (string) — the environment this resource is deployed into.
- `connections` (object) — the map of connection name to source resource ID.
- `codeReference` (string) — an optional URI back to the authoring source (e.g. a Git URL with commit SHA and line range).

The canonical YAML for this base lives at [pkg/schema/baseresource/base.yaml](/pkg/schema/baseresource/base.yaml). The set is **frozen** — future Radius releases will not add, remove, rename, or change the type of any property in this file. Promoting any additional property to common status requires a separate spec, not an evolution of this base.

## Who is affected

- **Resource type authors** writing a new manifest: if you want any of the four base properties, opt in via the `allOf` keyword shown below. If you do not opt in, your schema is validated exactly as you wrote it. The `environment` property is no longer special.
- **Existing resource type authors**: nothing changes. Your manifest continues to work as written. If your manifest previously listed all four base properties by hand, you can shorten it by opting into the base — but you do not have to.
- **Downstream consumers** (custom controllers, recipes that read resource properties): when a resource type opts into the base, every resource of that type carries the four properties at runtime in the standard places (`properties.application`, `properties.environment`, etc.). The runtime path now also exposes `codeReference` alongside the existing accessors on the dynamic resource adapter; see `pkg/dynamicrp/datamodel/dynamicresource.go::CodeReference()`.

## How to author

Opt in by declaring `allOf` with a `$ref` to the special URI `radius:base` at the **same level as `properties:`**, not inside it:

```yaml
namespace: MyCompany.Resources
types:
  widgets:
    apiVersions:
      "2025-01-01":
        schema:
          type: object
          allOf:
            - $ref: "radius:base"
          properties:
            size:
              type: string
              description: How big the widget is.
            color:
              type: string
              enum: ["red", "green", "blue"]
          required:
            - size
```

When this manifest is registered, the four base properties are merged into the schema and your type-specific properties (`size`, `color`) sit alongside them.

### Per-type-wins precedence

If you redeclare a base property in your own `properties:` block, your declaration **wins** and the base contributes nothing for that name. Use this when you want to narrow the description, mark a property as required, or both:

```yaml
schema:
  type: object
  allOf:
    - $ref: "radius:base"
  properties:
    # Narrow environment and mark it required for THIS type.
    environment:
      type: string
      description: The Radius environment that hosts this widget. Required.
  required:
    - environment
```

### The `allOf:` / `properties:` footgun

> ⚠️ The `$ref` MUST go inside `allOf:`, **not** inside `properties:`.

JSON Schema validators reject `$ref` placed under `properties:` because there it is interpreted as a literal property name. This will not register and you will see a confusing validator error. Always write:

```yaml
# CORRECT
schema:
  type: object
  allOf:
    - $ref: "radius:base"
  properties:
    size:
      type: string
```

and never:

```yaml
# WRONG — will fail to register
schema:
  type: object
  properties:
    $ref: "radius:base"   # treated as a property literally named "$ref"
    size:
      type: string
```

### Reserved property names

The names `status` and `recipe` remain reserved across all resource types and **must not** appear in your `properties:` block, whether or not you opt into the base. The base itself does not declare them.

### Unsupported `radius:` URIs

`radius:base` is the **only** legal value for the `radius:` URI scheme in this release. Anything else — `radius:base/foo`, `radius:other`, an empty `radius:` — produces a registration error scoped to the resource type and API version you are registering, naming the bad value and pointing at its `allOf[N]` index.

## Verification

After updating your manifest, register it with:

```bash
rad resource-type create -f my-manifest.yaml
```

A successful registration prints the namespace, resource type, and API version. To confirm the base properties were merged, read the type definition back:

```bash
rad resource-type show MyCompany.Resources/widgets
```

The output should include `application`, `environment`, `connections`, `codeReference`, and your type-specific properties.

## Troubleshooting

- **"environment is missing" error on a pre-existing manifest** — this no longer fires. If you still see it, you are running a Radius release older than v0.55. Upgrade or remove the manifest's `required: ["environment"]` if you do not actually need that constraint.
- **`unsupported radius: $ref ... — only "radius:base" is supported`** — change the `$ref` value to exactly `"radius:base"`. Quoted strings are fine; the value is matched literally with no JSON Pointer fragment.
- **`allOf is not supported`** — your `allOf:` contains an entry the validator does not understand (e.g. a literal subschema, not a `$ref` to `radius:base`). The base resource manifest is the only allowed `allOf` usage. Move any inline composition into the per-type `properties:` block.

## Related links

- Spec: [specs/210-base-resource-manifest/spec.md](/specs/210-base-resource-manifest/spec.md)
- Inheritance keyword contract: [specs/210-base-resource-manifest/contracts/inheritance-keyword.md](/specs/210-base-resource-manifest/contracts/inheritance-keyword.md)
- Canonical base YAML: [pkg/schema/baseresource/base.yaml](/pkg/schema/baseresource/base.yaml)
- Resolver implementation: [pkg/schema/baseresource/loader.go](/pkg/schema/baseresource/loader.go)
- Bicep type generator parallel implementation: [bicep-tools/pkg/converter/baseresource.go](/bicep-tools/pkg/converter/baseresource.go)
