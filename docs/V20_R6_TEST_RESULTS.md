# NLI Ver20 Release 6 — Test Results

Release identity: `20.0.0-6`  
Upgrade and rollback baseline: Ver20 Release 4 (`20.0.0-4`)

## Corrective scope

- Approval and parser synchronization paths canonicalize the exact parser bundle before validating the proposal hash, so the approval decision remains bound to the reviewed immutable bundle.
- Quarantine Resolution Wizard candidate testing, review, deployment, replay, and verification remain governed by the existing approval and exact-client acknowledgement requirements.
- This report records source and browser contract evidence only. RPM build, package contents, and Rocky lifecycle results are not claimed here.

## Executed source evidence

| Gate | Result | Evidence |
|---|---|---|
| Complete Go suite | PASS | `go test ./...`; all packages passed. Command packages reporting `[no test files]` are entry points; exercised behavior is covered by their imported runtime packages and integration/browser contracts. |
| Web API contracts | PASS | 24/24 Node web contract scenarios passed. |
| Browser UI scenarios | PASS | 69/69 UI scenarios passed, including the Quarantine Resolution Wizard, approvals, parser catalog, navigation badges, and subsystem tabs. |

## Not yet verified

- RPM reproducibility, NEVRA, and package payload.
- Rocky Linux 9/10 fresh install, R4-to-R6 upgrade, rollback, and re-upgrade.
- Published GitHub workflow and release artifacts.

These checks require a frozen release revision and the genuine R4 RPM artifacts. Do not describe Release 6 as RPM- or Rocky-lifecycle-qualified until those runs complete and their logs and artifact hashes are attached.
