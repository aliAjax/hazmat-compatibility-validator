# Hazmat compatibility and loading manifest validator

This is a pure Go service for closed-world dangerous-goods compatibility decisions. It validates substances, UN numbers, hazard classes, packaging groups, physical state, temperature/concentration intervals, containers, compartments and transport manifests. It does not manage products, shipments or commercial workflows.

## Run

Go 1.23 or newer is required.

```sh
./scripts/run-local.sh
```

The management API listens on `127.0.0.1:32710`. Every route except `/healthz` requires `Authorization: Bearer hazmat-admin-token` in the sample configuration. Run the real HTTP simulator in another terminal:

```sh
./scripts/smoke.sh
```

The simulator imports a rulebook and three versioned substances, freezes four manifest revisions, and verifies allowed, prohibited, conditional and indeterminate outcomes. It checks rule hits, conflict paths, unit conversions, result idempotency, streaming NDJSON quarantine and withdrawal/re-evaluation behavior.

## Decision semantics

The rulebook is a closed set of typed predicates: `class_pair`, `same_container`, `minimum_separation`, `temperature_required`, `quantity_limit`, `transport_mode`, `container_type` and `concentration_range`. Unknown predicates are rejected at import; no expression interpreter or open-ended script is evaluated.

Rules produce explicit `prohibited`, `conditional`, `allowed` or `indeterminate` outcomes. Higher priority wins per conflict path; a prohibited result is never downgraded by a lower-priority allow. Missing substance versions, missing quantities, unknown units, missing compartments or absent pair rules force `indeterminate`. The service never treats absent data as permission.

Intervals are preserved as `[min,max]` values. Unit conversion records input, canonical output and formula in evidence. An interval that crosses a threshold yields `conditional` where a point value could have yielded `allowed` or `prohibited`; overlapping concentration intervals and unresolved separation are retained as uncertainty rather than rounded away.

Substances and rulebooks are immutable versions. Effective dates are checked at validation time. A rulebook can be withdrawn; later re-evaluation against its withdrawn/effective period is rejected. Manifest revisions are append-only, must be frozen before validation, and freeze stores a SHA-256 input digest. Frozen content or a digest mismatch cannot be silently changed.

## API and batch behavior

OpenAPI is in [api/openapi.yaml](api/openapi.yaml), and the protobuf contract is [api/hazmat.proto](api/hazmat.proto). `POST /v1/validate` returns a result containing facts, conversions, rule hits, conflicts, missing paths and the input digest. Repeating the same frozen digest and rulebook version returns the stored result with `X-Idempotent-Replay: true`.

`POST /v1/batches/validate` consumes newline-delimited JSON with a required `Idempotency-Key`. It uses bounded workers, scanner line limits and context cancellation. Decode/validation errors become quarantine rows and do not abort valid rows. Quarantine rows are available through `/v1/quarantine?batch_id=...`.

The management API supports `/healthz`, `/readyz`, `/metrics`, substance/rulebook import, rulebook withdrawal, manifest draft creation and freeze, validation, re-evaluation, batch validation, quarantine listing and `/admin/drain`. SIGINT/SIGTERM drains and gracefully closes the listener.

## Control-plane persistence

The PostgreSQL migrations store rulebook versions, substance versions, manifest revisions, validation results and audit events. The in-memory runtime does not require PostgreSQL; block or chemical payloads are never placed in a database table outside their JSON metadata payloads. Apply migrations with `DATABASE_URL=... ./scripts/migrate.sh` when integrating a control plane.

## Explicit boundaries

This implementation does not claim regulatory certification, complete ADR/IMDG/IATA coverage, automatic legal interpretation, physical container certification, live sensor integration, product ordering, shipment tracking or commercial inventory behavior. Sample rules are illustrative only. A rulebook must be supplied and versioned by the deploying authority.

The project contains more than 2,000 non-test Go source lines split across domain, application and infrastructure packages. Test files, SQL, configuration, examples, scripts, generated protobuf output and build artifacts are excluded from that count.
