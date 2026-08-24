# Bug Reproduction

## Bug

Rulebook repository and validation service wrapped sentinel errors with percent-v, so duplicate, missing, withdrawn, and canceled cases became plain text. The HTTP response also exposed only the text and gave callers no stable error type.

## Trigger

Submit a duplicate rulebook, validate a revision that references a missing rulebook, evaluate after the referenced rulebook is withdrawn, or pass an already-canceled context to reevaluation. Before the fix, errors.Is could not identify the expected sentinel and the HTTP error response had no type field.

## Observed Error

save rulebook v1: rulebook version exists

get rulebook missing: rulebook version not found

get rulebook v1: rulebook version not effective or withdrawn
