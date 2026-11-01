# NLI Ver20 Release 4 — Test Results

Release identity: `20.0.0-4`
Test date: 2026-11-01

## Corrective scope

- Quarantine Resolution Wizard stages use the selected record, evidence-backed field mappings, generated candidate, exact deterministic preview, parser revision, rollout bundle, origin-client acknowledgement, replay, and final resolution as their canonical state.
- Candidate preview validates the generated parser itself without persisting or activating it; validation reports non-zero case totals and an artifact checksum.
- Authorized administrators can use an explicit emergency same-user approval only when the request carries a reason and the exact revision plus parser checksum or rollout bundle hash. Normal independent approval remains the default.
- Historical compatible parser records are converted deterministically to current runtime packs. Invalid or colliding runtime identities remain blocked with an actionable repair reason.
- Notification badges use per-user unread state. Read, unread, check-later, and delete-from-active-feed actions do not delete immutable audit evidence.
- The login banner is contained above the sign-in form on the right side and remains square and responsive.

## Executed evidence

| Gate | Result | Evidence |
|---|---|---|
| Complete Go suite | PASS | `go test ./...` in `golang:1.24.9`; every tested package passed |
| Go static analysis | PASS | `go vet ./...` in the pinned Go image |
| Complete Chromium UI suite | PASS | 67/67 scenarios across all primary workspaces, visible inner tabs, QRW stages, notification state, login layout, live synchronization, and responsive layout |
| JavaScript syntax | PASS | `node --check web/app.js`; `node --check web/ui.spec.js` |

Command packages that report `[no test files]` contain entry-point wiring; their imported runtime packages and end-to-end behavior are exercised by the package and browser suites. That message is informational, not a failure.

The source-level gates above are verified. RPM reproducibility, package payload, Rocky lifecycle, and published CI results must be recorded from the frozen Release 4 commit and generated artifacts; they are not claimed by this document before those gates run.
