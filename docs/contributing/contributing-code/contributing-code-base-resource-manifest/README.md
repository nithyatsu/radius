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

## How to test

This section walks a contributor through end-to-end verification of the feature. It covers both surfaces the change touches: the schema validator that backs `rad resource-type create` and the Bicep extension generator (`bicep-tools/cmd/manifest-to-bicep`) that powers `rad bicep publish-extension`.

### Prerequisites

Rebuild both binaries from the branch under test:

```bash
make build-cli
# main rad binary lands at ./dist/<os>_<arch>/release/rad

cd bicep-tools && go build -o ../dist/manifest-to-bicep ./cmd/manifest-to-bicep && cd -
```

> Use `./dist/<os>_<arch>/release/rad` (or add it to your `PATH`) so the manual steps below pick up the freshly built binary rather than the system-installed `rad`.

### Test fixtures

Save these three YAML files in a scratch directory.

`baseline.yaml` — opts into the base, no overrides:

```yaml
namespace: Test.Resources
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
          required:
            - size
```

`override.yaml` — per-type narrows `environment` and marks it required:

```yaml
namespace: Test.Resources
types:
  gadgets:
    apiVersions:
      "2025-01-01":
        schema:
          type: object
          allOf:
            - $ref: "radius:base"
          properties:
            environment:
              type: string
              description: The Radius environment hosting this gadget. Required.
          required:
            - environment
```

`bad-uri.yaml` — should fail with an actionable error:

```yaml
namespace: Test.Resources
types:
  doodads:
    apiVersions:
      "2025-01-01":
        schema:
          type: object
          allOf:
            - $ref: "radius:base/not-real"
          properties:
            label:
              type: string
```

### Path A — `rad resource-type create`

1. **Happy path — opt-in succeeds.**

   ```bash
   rad resource-type create -f baseline.yaml
   rad resource-type show Test.Resources/widgets
   ```

   Expect a success message naming `Test.Resources/widgets@2025-01-01`. The `show` output must list **all** of `application`, `environment`, `connections`, `codeReference`, **and** `size`.

2. **Per-type-wins override succeeds.**

   ```bash
   rad resource-type create -f override.yaml
   rad resource-type show Test.Resources/gadgets
   ```

   `environment` appears with the narrowed description and is listed in `required:`. The other three base properties (`application`, `connections`, `codeReference`) are still present.

3. **Unsupported `radius:` URI fails cleanly.**

   ```bash
   rad resource-type create -f bad-uri.yaml
   ```

   Expect a non-zero exit and an error message containing `Test.Resources/doodads@2025-01-01`, `radius:base/not-real`, and `allOf[0]`. The message must mention that `radius:base` is the only supported value.

4. **Backward compatibility — environment is no longer auto-required.** Author a manifest with neither the `allOf` opt-in nor an `environment` property:

   ```yaml
   namespace: Test.Resources
   types:
     plain:
       apiVersions:
         "2025-01-01":
           schema:
             type: object
             properties:
               name: { type: string }
   ```

   `rad resource-type create` must succeed. Before this change it failed with `property 'environment' must be included in schema`.

5. **Reserved names still rejected.** Add a property called `status` or `recipe` to any of the schemas — registration must still fail with the existing reserved-property error.

6. **Round-trip `codeReference` on a real resource.** Deploy a Bicep app that sets `codeReference: "https://github.com/example/repo/blob/abc1234/app.bicep#L10-L20"` on a widget instance, then:

   ```bash
   rad resource show Test.Resources/widgets <name> -o json | jq .properties.codeReference
   ```

   The value must round-trip exactly.

### Path B — Bicep extension generation

`rad bicep publish-extension` builds an extension index by invoking the bicep-tools generator under the hood. Exercise the generator directly so failures are easier to diagnose, then confirm the higher-level command works.

1. **Happy path — emitted extension contains the base properties.**

   ```bash
   ./dist/manifest-to-bicep --manifest baseline.yaml --output ./out/
   ```

   Inspect the generated Bicep types JSON. The `widgetsProperties` object type must include `application`, `environment`, `connections`, `codeReference`, and `size`. The `allOf` / `$ref` keyword must be absent from the emitted output — `applyBaseResource` resolves it away before the Bicep emitter runs.

2. **Override is honoured.** Run the generator against `override.yaml` and confirm the emitted `gadgetsProperties` type carries the narrowed `environment` description (not the base description) and that `environment` is marked required.

3. **Bad URI fails.** Run against `bad-uri.yaml`. The tool must exit non-zero with an error message that includes `Test.Resources/doodads@2025-01-01`, `radius:base/not-real`, and `allOf[0]`.

4. **End-to-end `rad bicep publish-extension`.**

   ```bash
   rad bicep publish-extension --manifest baseline.yaml --target ./out/extension.tgz
   ```

   The command must succeed. Reference the resulting extension in a small Bicep file that creates an instance of `Test.Resources/widgets` and sets `codeReference`. `rad deploy` of that file must accept it (the base properties resolve correctly in the Bicep type system) and the resource must reach the success state.

### Automated regression coverage already in tree

These cases are exercised by `go test -count=1` on every CI run; you do not have to drive them by hand, but they are useful when narrowing down a failure:

```bash
go test -count=1 \
  ./pkg/schema/... \
  ./pkg/cli/manifest/... \
  ./pkg/resourceutil/... \
  ./pkg/dynamicrp/datamodel/... \
  ./bicep-tools/pkg/converter/...
```

What each package covers:

- [pkg/schema/baseresource](/pkg/schema/baseresource/) — nil schema, no `allOf`, non-radius refs, four-property merge, per-type-wins, unsupported `radius:` URIs at index 0 and non-zero, frozen property names, embedded YAML load.
- [pkg/cli/manifest](/pkg/cli/manifest/) — opt-in succeeds, override succeeds, unsupported URI error names the resource type and API version.
- [pkg/schema](/pkg/schema/) — inverted `environment`-required tests confirm a schema without `environment` now validates.
- [pkg/dynamicrp/datamodel](/pkg/dynamicrp/datamodel/) — `CodeReference()` accessor covers nil, missing, non-string, and valid string cases.
- [bicep-tools/pkg/converter](/bicep-tools/pkg/converter/) — parallel `applyBaseResource` tests plus `TestApplyBaseResource_PropertiesMatchCanonicalYAML`, which fails loudly if the hardcoded list in `bicep-tools` ever drifts from [pkg/schema/baseresource/base.yaml](/pkg/schema/baseresource/base.yaml).

### Worked example: migrating `testresourcetypes.yaml`

The repository ships a real multi-type fixture at [test/functional-portable/dynamicrp/noncloud/resources/testdata/testresourcetypes.yaml](/test/functional-portable/dynamicrp/noncloud/resources/testdata/testresourcetypes.yaml) that pre-dates the base resource manifest. Every type in that file restates `application` and `environment` (and sometimes `connections`) by hand. This is the recommended end-to-end test plan: rewrite the file to use the `$ref` opt-in, then verify both `rad resource-type create` and `rad bicep publish-extension`.

#### What to rewrite, and what NOT to

Each type declares some combination of `application`, `environment`, and `connections`. Substitute those base properties with `allOf: [{$ref: "radius:base"}]`. **Keep** type-specific properties (`port`, `database`, `username`, ...) and any per-type `required:` array.

Two caveats before you mass-replace:

- **`userTypeAlpha`** declares a richer `connections` schema than the base (nested `additionalProperties.properties.source`). The base contributes a plain `connections: {type: object, additionalProperties: {type: object}}`. To keep the richer shape, **leave the `connections` block in `properties:`** — per-type-wins precedence keeps your version and the base contributes nothing for that name. If you delete it, the type gets the simpler base shape.
- **`postgres`** has no `connections` field today. After opting in, `connections` will appear on `postgres` because the base provides it. This is intended.

#### Before → after migration

**`userTypeAlpha`** — keeps the richer `connections`:

```yaml
namespace: Test.Resources
types:
  userTypeAlpha:
    apiVersions:
      "2023-10-01-preview":
        schema:
          type: "object"
          allOf:
            - $ref: "radius:base"
          properties:
            # Override the base connections shape with the richer per-type one.
            connections:
              type: object
              additionalProperties:
                type: object
                properties:
                  source:
                    type: string
                    description: The resourceID of the source of the connection.
            port:
              type: string
              description: The port number exposed by the application.
          required:
            - application
            - environment
      "2025-01-01-preview":
        schema:
          type: "object"
          allOf:
            - $ref: "radius:base"
          properties:
            connections:
              type: object
              additionalProperties:
                type: object
                properties:
                  source:
                    type: string
                    description: The resourceID of the source of the connection.
          required:
            - application
            - environment
```

**`postgres`** — base provides app/env, the type-specific block stays:

```yaml
  postgres:
    apiVersions:
      "2025-01-01-preview":
        schema:
          type: object
          allOf:
            - $ref: "radius:base"
          properties:
            database:
              type: string
              description: The name of the database.
            host:
              type: string
              description: The host name of the database.
            port:
              type: string
              description: The port number of the database.
            username:
              type: string
              description: The username for the database.
            password:
              type: string
              description: The password for the database.
```

**`externalResource`** — base + `configMap` + extended `required`:

```yaml
  externalResource:
    capabilities: ["ManualResourceProvisioning"]
    apiVersions:
      "2023-10-01-preview":
        schema:
          type: "object"
          allOf:
            - $ref: "radius:base"
          properties:
            configMap:
              type: string
              description: The data contained in a config map.
          required:
            - application
            - environment
            - configMap
```

Apply the same pattern to `sharedAPITestTypeA`, `sharedAPITestTypeB`, `testResourceSchema`, `testValidPlatformOptionsSchema`, and `sensitiveResource`. The mechanical rule is: delete the `application`, `environment`, and (if it matches the base shape) `connections` entries, then add `allOf: [{$ref: "radius:base"}]` at the same indentation as `properties:`.

#### Verify with `rad resource-type create`

```bash
# Use your freshly built rad.
RAD=./dist/darwin_arm64/release/rad   # adjust for your OS/arch
$RAD version

# Wipe any prior registrations so we read a clean state.
$RAD resource-type delete Test.Resources/userTypeAlpha --yes 2>/dev/null || true
$RAD resource-type delete Test.Resources/postgres --yes 2>/dev/null || true
$RAD resource-type delete Test.Resources/externalResource --yes 2>/dev/null || true

# Register everything in the migrated file.
$RAD resource-type create -f test/functional-portable/dynamicrp/noncloud/resources/testdata/testresourcetypes.yaml
```

Expected: a success line per type. No `property 'environment' must be included in schema` errors and no `allOf is not supported` errors.

Inspect each migrated type:

```bash
# userTypeAlpha — app/env/connections/codeReference + port
$RAD resource-type show Test.Resources/userTypeAlpha

# postgres — app/env/connections/codeReference + database/host/port/username/password
$RAD resource-type show Test.Resources/postgres

# externalResource — app/env/connections/codeReference + configMap,
# and required: [application, environment, configMap]
$RAD resource-type show Test.Resources/externalResource
```

Pass criteria:

- All four base properties (`application`, `environment`, `connections`, `codeReference`) are present.
- All per-type properties are present.
- For `userTypeAlpha`, `connections.additionalProperties.properties.source` is preserved (per-type override won).
- The `required:` array matches exactly what you wrote.

Negative test in the same file — temporarily change one type's `$ref` to `radius:base/typo` and re-run `resource-type create`. You should get an error containing the type name, the API version, `radius:base/typo`, `allOf[0]`, and a hint that `radius:base` is the only supported value. Revert before continuing.

#### Verify with `rad bicep publish-extension`

`rad bicep publish-extension` calls the same `bicep-tools/generator` code path the feature wired through, so the four base properties appear in the generated Bicep types.

```bash
# Publish to a local tgz so you can crack it open.
$RAD bicep publish-extension \
  --from-file test/functional-portable/dynamicrp/noncloud/resources/testdata/testresourcetypes.yaml \
  --target ./out/test-resources.tgz \
  --force

# Unpack.
mkdir -p out/unpacked
tar -xzf out/test-resources.tgz -C out/unpacked
ls out/unpacked
```

Every `*Properties` object type emitted from a migrated resource type should declare the four base properties:

```bash
for prop in application environment connections codeReference; do
  echo "== ${prop} =="
  grep -l "\"${prop}\"" out/unpacked/*.json
done
```

You should see entries for `userTypeAlphaProperties`, `postgresProperties`, `externalResourceProperties`, and so on.

The `$ref` / `allOf` must be resolved away in the emitted output:

```bash
grep -E '"\$ref"|"allOf"|radius:base' out/unpacked/*.json \
  && echo "FAIL: \$ref leaked into emitted Bicep types"
```

A clean run prints nothing and the `&&` branch is skipped.

Negative test — same as for `resource-type create`: change one `$ref` to `radius:base/typo`, re-run `rad bicep publish-extension`. The command must fail with an error that includes the resource type, API version, `radius:base/typo`, `allOf[0]`, and `radius:base` as the only legal value. Revert before continuing.

#### End-to-end smoke test (optional)

Once both surfaces accept the migrated file, deploy a tiny Bicep template that uses the extension and instantiates one of the migrated types (e.g. `Test.Resources/userTypeAlpha`) with a `codeReference` value set. Then:

```bash
$RAD resource show Test.Resources/userTypeAlpha <name> -o json | jq '.properties'
```

The output must include `application`, `environment`, `connections`, **and** `codeReference` round-tripping the value you set. This confirms the runtime path (the `CodeReference()` accessor on the dynamic adapter) is wired correctly.

## Related links

- Spec: [specs/210-base-resource-manifest/spec.md](/specs/210-base-resource-manifest/spec.md)
- Inheritance keyword contract: [specs/210-base-resource-manifest/contracts/inheritance-keyword.md](/specs/210-base-resource-manifest/contracts/inheritance-keyword.md)
- Canonical base YAML: [pkg/schema/baseresource/base.yaml](/pkg/schema/baseresource/base.yaml)
- Resolver implementation: [pkg/schema/baseresource/loader.go](/pkg/schema/baseresource/loader.go)
- Bicep type generator parallel implementation: [bicep-tools/pkg/converter/baseresource.go](/bicep-tools/pkg/converter/baseresource.go)
