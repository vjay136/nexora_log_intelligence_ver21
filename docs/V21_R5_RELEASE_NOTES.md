# Nexora Log Intelligence Ver21 Release 5

Release 5 is version `21.0.0-5`, based on Ver21 R4 (`v21.0.0-4`). It refines field mapping and parser lifecycle presentation in the Guided Parser Repair Wizard (QRW) and shared Parser Catalog.

## QRW field mapping

- Steps 3–4 accept operator-defined JSON field names and display labels.
- Deterministic extraction supports raw positions and inclusive token ranges, key-value fields, and delimiter-based values.
- Delimiter and string inputs have explicit size bounds. Key-value extraction retains multiword values.
- Automatic field population is capped at 128 suggestions. The wizard validates overlapping source ranges before accepting a mapping or generating a parser candidate.
- Mapping suggestions and previews remain tied to the inspected raw record. Unresolved or overlapping mappings require correction before candidate generation.

## Parser lifecycle visibility

The shared Parser Catalog presents a parser as Published / Active when its published revision is confirmed synchronized with the runtime. Blocked historical entries retain their repair path and status.

## Compatibility boundary

R5 leaves the R4 ingestion, SSE/live synchronization, approval, hashing, rollout, and replay contracts unchanged. Package names, services, module path, and durable configuration and state paths remain compatible with in-place upgrades. The previous production release is R4 (`v21.0.0-4`).

See [R5 test results](V21_R5_TEST_RESULTS.md) for the verification evidence associated with this release.
