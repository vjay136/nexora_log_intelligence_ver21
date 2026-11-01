# NLI Ver20 Release 6 — Test Results

Release identity: `20.0.0-6`  
Upgrade and rollback baseline: Ver20 Release 4 (`20.0.0-4`)

## Corrective scope

- Approval and parser synchronization paths canonicalize the exact parser bundle before validating the proposal hash, so the approval decision remains bound to the reviewed immutable bundle.
- Quarantine Resolution Wizard candidate testing, review, deployment, replay, and verification remain governed by the existing approval and exact-client acknowledgement requirements.
- Release artifacts are built twice from the same frozen revision. The resulting RPMs, archives, WebView bundles, installers, and documentation must be byte-identical before publication.

## Executed source evidence

| Gate | Result | Evidence |
|---|---|---|
| Complete Go suite | PASS | `go test ./...`; all packages passed. Command packages reporting `[no test files]` are entry points; exercised behavior is covered by their imported runtime packages and integration/browser contracts. |
| Go static analysis | PASS | `go vet ./...` completed without findings. |
| Go race suite | PASS | The release builder ran the repository race-test gate successfully. |
| Fuzz smoke test | PASS | The release builder completed the bounded parser fuzz gate successfully. |
| Web API contracts | PASS | 24/24 Node web contract scenarios passed. |
| Browser UI scenarios | PASS | 69/69 UI scenarios passed, including the Quarantine Resolution Wizard, approvals, parser catalog, navigation badges, and subsystem tabs. |
| Reproducible release build | PASS | Two clean containerized builds produced byte-identical release bundles and RPMs. All packaged SHA-256 manifests and installer identity checks passed. |
| RPM identity | PASS | `nli-client-20.0.0-6.el9.x86_64` and `nli-controller-agent-20.0.0-6.el9.x86_64` were built and inspected. |
| R4-to-R6 lifecycle — Rocky 9 | PASS | Genuine `20.0.0-4` RPMs installed, upgraded to R6, downgraded, re-upgraded, and removed successfully. Service and parser lifecycle checks passed. |
| R4-to-R6 lifecycle — Rocky 10.2 | PASS | The same genuine R4-to-R6 lifecycle passed on Rocky Linux 10.2. |
| Parsing regression accounting | PASS | The retained 49-record fixture produced 49 parsed outputs, zero quarantine records, and zero duplicates; restart produced no additional outputs. |
| Invalid managed-parser isolation | PASS | A malformed synchronized managed pack was isolated, the agent remained active, and a 50th record was parsed by the built-in path without duplicating prior output. |
| Two-node TLS and WebView acceptance | PASS | Separate Rocky 10.2 controller and client nodes enrolled successfully; heartbeat, exact TLS identity checks, certificate discovery/fallback, and six packaged WebView browser scenarios passed. |

## Release boundary

- Before the frozen-artifact gates executed, RPM and Rocky lifecycle qualification was not claimed here and was not yet verified. The completed, frozen-artifact results above now record both gates as PASS.
- The tests above qualify the locally built release artifacts. GitHub publication and hosted workflow status are reported separately after the exact tag and artifacts are uploaded.
- The release does not auto-approve parser changes. Independent approval remains the default; an administrator emergency path requires an explicit reason and exact revision/hash confirmation.
- A client must acknowledge the exact approved bundle hash before replay is enabled. This prevents replay against a different parser bundle while allowing canonical JSON-equivalent R4 proposal hashes to migrate safely.
