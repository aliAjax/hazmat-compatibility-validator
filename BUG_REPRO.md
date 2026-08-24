# Bug Reproduction

## Bug

Quantity interval, unit, and dimension errors retain readable text but lose their
sentinel causes while crossing validation, conversion, and importer boundaries.
Callers therefore cannot classify the failures with `errors.Is`.

## Trigger

Run the focused regression checks:

```bash
go test ./internal/importaudit -run '^TestH010IntervalValidationPreservesSentinel$' -count=1
go test ./internal/importaudit -run '^TestH010ConverterPreservesIntervalCause$' -count=1
go test ./internal/importaudit -run '^TestH010ManifestImporterPreservesUnknownUnit$' -count=1
go test ./internal/importaudit -run '^TestH010SubstanceImporterPreservesDimensionCause$' -count=1
```

## Observed Errors

The buggy baseline exits with status 1 for all four commands and reports:

```text
invalid interval cause was lost: invalid interval: min=NaN max=10 unit="kg"
converter discarded interval cause: input interval rejected: invalid interval: min=+Inf max=+Inf unit="kg"
manifest importer discarded unknown-unit cause: container drum capacity: unit "stone" is unknown
substance importer discarded dimension cause: temperature: unit "kg" is not valid for temperature
```
