# NLI Ver20 Release 5 — Test Results

Release identity: `20.0.0-5`
Test date: 2026-11-01

## Corrective scope

- Parser rollout proposals use a semantic bundle digest shared by the controller and client. Equivalent JSON key order and numeric spelling no longer invalidate an immutable parser snapshot after a storage round-trip.
- A pending legacy full or shortened hash is upgraded only when its embedded snapshot can be verified as the exact current published bundle. The migration records the old and canonical hashes, operator, time, and validation mode.
- Malformed, mismatched, or stale proposals remain blocked with actionable `409 Conflict` guidance. R5 never bypasses the bundle-integrity gate.
- Independent and audited administrator emergency approvals carry an explicit reason. Emergency approval is additionally bound to the exact proposal revision and canonical bundle hash.
- Quarantine Resolution Wizard Steps 8–9 refresh from the controller after approval/cancellation and keep replay blocked until the origin client reports the exact applied hash.
- The login banner is enlarged in a responsive landscape stage while preserving the complete supplied artwork without cropping or distortion.

## Executed evidence

| Gate | Result | Evidence |
|---|---|---|
| Complete Go suite | PASS | `go test ./...` in `golang:1.23`; every tested package passed |
| Complete Chromium UI suite | PASS | 69/69 scenarios, including current, legacy, independent, emergency, rejected-hash, QRW refresh, notifications, and responsive login behavior |
| Focused controller/client compatibility | PASS | Current, legacy, stale, malformed, emergency, independent, and client wire-hash tests passed |
| JavaScript syntax | PASS | `node --check web/app.js` and `node --check web/ui.spec.js` |
| Release infrastructure static gate | PASS | Controller edge, SELinux, AWS, Cloudflare/Vercel, evidence, installer, parser provenance, RPM/systemd, and Supabase checks passed |
| Deterministic package-infrastructure fixture | PASS | Two independent R5 RPM and release-bundle builds matched; artifact, checksum, installer identity, metrics, and packaged-document verification passed |

Command packages that report `[no test files]` contain executable entry-point wiring. Their imported runtime packages and end-to-end behavior are exercised by the package and browser suites; this message is informational rather than a failure.

The source and package-infrastructure gates above are verified. Rocky lifecycle tests and published CI results are recorded only after they run from the frozen Release 5 commit.
