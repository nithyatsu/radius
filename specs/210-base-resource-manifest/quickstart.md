# Quickstart — Demo the feature

**Feature**: Base Resource Manifest
**Date**: 2026-06-19

This quickstart describes how to build a local `rad` CLI + control plane from the `210-base-resource-manifest` branch and demonstrate the new authoring experience end-to-end. The demo registers a resource type that uses `allOf: [{ $ref: "radius:base" }]` to inherit the four common Radius properties, then deploys an instance to show the inherited properties behave like they do on built-in types.

---

## Prerequisites

- A working Radius dev environment per [docs/contributing/contributing-code/contributing-code-prerequisites/](../../docs/contributing/contributing-code/contributing-code-prerequisites/) (Go toolchain, Kubernetes cluster, container registry).
- The `radius-build-cli`, `radius-build-images`, and `radius-install-custom` skills available (see `.github/skills/`).
- This repo cloned and the `210-base-resource-manifest` branch checked out and built.

---

## Build and install

```bash
git checkout 210-base-resource-manifest
make build
make docker-build DOCKER_TAG_VERSION=baseresource
# Install per the radius-install-custom skill, pointing at your registry
```

---

## Demo flow

1. **Author a `$ref`-using resource type**. Save as `mywidget.yaml`:

   ```yaml
   namespace: Demo.Examples
   types:
     widgets:
       description: A demo widget that uses the base resource manifest.
       apiVersions:
         "2026-06-19":
           schema:
             type: object
             allOf:
               - $ref: "radius:base"          # inherits the four common Radius properties
             properties:
               widgetSize:
                 type: integer
                 description: The size of the widget in arbitrary units.
             required:
               - widgetSize
   ```

   The YAML declares only one type-specific property (`widgetSize`). The four common properties (`application`, `environment`, `connections`, `codeReference`) are inherited from the base via the `allOf` + `$ref` composition. **Note**: `$ref` lives under `allOf:`, NOT under `properties:` — see [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for why.

2. **Register the type**:

   ```bash
   rad resource-type create -f mywidget.yaml
   ```

   Registration succeeds with no "missing required property `environment`" error. (Pre-feature, this would have failed.)

3. **Deploy an instance** with a Bicep file `widget-instance.bicep`:

   ```bicep
   extension radius

   resource widget 'Demo.Examples/widgets@2026-06-19' = {
     name: 'demo'
     properties: {
       widgetSize: 3
       application: 'app-resource-id-here'
       environment: 'env-resource-id-here'
       connections: {
         db: { source: 'db-resource-id-here' }
       }
       codeReference: 'https://github.com/myorg/myrepo/blob/abc123/widgets/demo.bicep#L1'
     }
   }
   ```

   ```bash
   rad deploy widget-instance.bicep
   ```

   Deployment is accepted, `application` / `environment` resolve through the existing Radius adapters, `connections` is extracted normally, `codeReference` round-trips and shows up in `rad resource show`.

4. **Show that the four common properties are optional**. Re-deploy a second instance that omits all four:

   ```bicep
   resource widgetBare 'Demo.Examples/widgets@2026-06-19' = {
     name: 'demo-bare'
     properties: {
       widgetSize: 1
     }
   }
   ```

   Deployment is accepted. (SC-004 demonstrated.)

---

## Demonstrate the error path (SC-005)

Edit `mywidget.yaml` to declare `environment` per-type but with a wrong primitive type:

```yaml
schema:
  type: object
  allOf:
    - $ref: "radius:base"
  properties:
    environment:
      type: integer        # WRONG — environment is a string
    widgetSize:
      type: integer
  required:
    - widgetSize
```

Run:

```bash
rad resource-type create -f mywidget.yaml
```

The CLI rejects the registration at command time with an actionable error naming `environment` and the type mismatch — no control-plane round-trip is needed.

---

## Verifying the breaking-change documentation

This feature ships `docs/contributing/contributing-code/contributing-code-base-resource-manifest.md` (SC-003). To verify:

```bash
test -f docs/contributing/contributing-code/contributing-code-base-resource-manifest.md
grep -q "environment .*no longer .*required" docs/contributing/contributing-code/contributing-code-base-resource-manifest.md
```

Both checks should pass.

---

## Troubleshooting

- **`rad resource-type create` errors with `unknown property 'codeReference'`**: you are running against a `rad` CLI from `main`, not from the `210-base-resource-manifest` branch. Re-build with `make build` on the branch and re-run.
- **`unsupported scheme radius:` from the validator**: the `$ref` value is not exactly `radius:base`. Check spelling — the literal is the only legal value in this version.
- **`unknown property '$ref'` from `kin-openapi`**: the `$ref` is misplaced under `properties:` instead of `allOf:`. Move it. See [contracts/inheritance-keyword.md](./contracts/inheritance-keyword.md) for the canonical placement.
- **Deployment fails with `missing required property 'environment'`**: the per-type YAML still has `environment` listed in its `required:` array. Remove it (or keep it if you want env to remain required for that type — FR-004).
