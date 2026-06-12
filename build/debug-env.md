# Debug environment — `debug-env-secret` branch

Reference for the changes on branch `debug-env-secret` that make `make debug-start && drad deploy ...` work end-to-end against a local k3d cluster with dlv-attached host processes.

Branch base: `main` (`54122e720`). Branch head: `8d7a707e3` "add configs to make the debug env work".

All changes are confined to `build/configs/` and `build/scripts/`. No product (`pkg/`, `cmd/`) code is touched.

---

## Topology

```
host                                          k3d cluster (radius-debug)
├── ucp              :9000  (dlv :40001) ◀──┐  ┌── deployment-engine pod
├── controller       :7073  (dlv :40002)    │  │   container :6443  --kubernetes=true
├── applications-rp  :8080  (dlv :40003)    │  │   service   :6443  (ClusterIP)
├── dynamic-rp       :8082  (dlv :40004)    │  │
└── kubectl port-forward 5017:6443 ────────────┘  apiserver in-cluster: 10.43.0.1:443
                                            │
   ucp Microsoft.Resources plane ──► http://localhost:5017 ──► DE pod :6443
   DE pod ──► RADIUSBACKENDURL=http://host.k3d.internal:9000 ──► host ucp ─┘
```

---

## Required config (do not regress)

### `build/configs/ucp.yaml`
- Top-level key MUST be `environment:` (`testresourceenvironment:` is rejected).
- `initialization.planes` for `/planes/radius/local` sets:
  - `Applications.*` → `http://localhost:8080`
  - `Microsoft.Resources` → `http://localhost:5017` (port-forward to DE)
- `ucp.kind: direct`
- `ucp.direct.endpoint: http://localhost:9000/apis/api.ucp.dev/v1alpha3`
- `routing.defaultDownstreamEndpoint: http://localhost:8082` (dynamic-rp)

### `build/configs/deployment-engine.yaml`
- Container args: `--kubernetes=true` (NEVER `--local`).
- Container port: `6443`. `ASPNETCORE_URLS=http://+:6443`.
- Service `port: 6443`, `targetPort: 6443`.
- Env: `RADIUSBACKENDURL=http://host.k3d.internal:9000/apis/api.ucp.dev/v1alpha3`, `kubernetes=true`, `KUBERNETES_ENABLED=true`.

### `build/scripts/setup-deployment-engine-port-forward.sh`
- `kubectl ... port-forward ... service/deployment-engine 5017:6443`.

### `build/scripts/start-radius.sh`
- Auto-create secret `radius-system/radius-encryption-key` before launching dynamic-rp (32 random bytes; BSD/GNU `date`-compatible).
- `kubectl apply -f deploy/Chart/crds/radius/` and `.../crds/ucpd/` before launching controller.
- Per-port liveness via `lsof -nP -iTCP:<port> -sTCP:LISTEN`. Do NOT trust `curl /healthz` — dlv keeps the OS process alive after the inner binary crashes.

### `build/scripts/status-radius.sh`
- Bash 3.2-compatible `component_port()` case: ucp=9000, controller=7073, applications-rp=8080, dynamic-rp=8082.
- Three-state output per component: `✅ alive & listening`, `⚠️ dlv alive but binary not listening`, `❌ dead`.

---

## Root causes addressed

| Symptom | Cause | Fix |
|---|---|---|
| UCP fails to parse config | Top-level key `testresourceenvironment:` unknown | Renamed to `environment:` |
| DE recipe deploy hangs, then `RecipeDeploymentFailed` with `Connection refused (localhost:6443)` | `--local` makes DE assume kube-apiserver at its own pod's localhost | Use `--kubernetes=true` (matches Helm chart `bicep-de`) |
| `Microsoft.Resources` PUTs land nowhere | Plane pointed at in-cluster chart URL `http://bicep-de.radius-system:6443` from a host process | Point at `http://localhost:5017` (port-forward) |
| dynamic-rp dies at startup | Missing `radius-system/radius-encryption-key` secret | Auto-create in `start-radius.sh` |
| controller crash-loops | Missing CRDs (`recipes.radapp.io` et al.) | Apply `deploy/Chart/crds/{radius,ucpd}` in `start-radius.sh` |
| `status-radius.sh` claims components healthy after the binary crashed | Only checked dlv PID / `curl /healthz` | Check actual TCP listener via `lsof` |
| Port-forward broken after DE migrated to 6443 | Old `5017:5445` mapping | Updated to `5017:6443` |

---

## One-time / per-session manual steps

After `make debug-start`:

```bash
# 1. Register test resource types (per fresh cluster)
./drad resource-provider create --from-file \
  test/functional-portable/dynamicrp/noncloud/resources/testdata/testresourcetypes.yaml

# 2. Deploy the test app (uses user-specific registry)
./drad deploy \
  test/functional-portable/dynamicrp/noncloud/resources/testdata/udt2udt-connection.bicep \
  --parameters registry=ghcr.io/nithyatsu \
  --parameters version=local-dev
```

`ghcr.io/nithyatsu:local-dev` is user-specific because the recipe must be pushed by someone with write access. Replace with your own org if rebuilding.

---

## Validation

End-to-end works when:

```bash
./drad app graph -a udttoudtapp1 -o json \
  | jq '.resources[] | {name, type, properties}'
```

returns populated `properties` bags for both `udttoudtparent` (Test.Resources/userTypeAlpha, including `port`, `recipe`) and `udttoudtchild` (Test.Resources/externalResource, including `configMap`).

---

## Files changed on this branch

- `build/configs/ucp.yaml`
- `build/configs/deployment-engine.yaml`
- `build/scripts/start-radius.sh`
- `build/scripts/status-radius.sh`
- `build/scripts/setup-deployment-engine-port-forward.sh`

Generated this summary on 2026-06-11. Keep this in sync if any of the above files change.
