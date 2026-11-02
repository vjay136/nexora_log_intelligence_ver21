# Nexora Log Intelligence Ver21 Release 3 Test Results

Release identity: `21.0.0-3`

## Required release checks

| Check | Command / evidence |
|---|---|
| Release documentation and metadata contract | `python tests/release/release_docs_packaging_test.py --root . --output <built-output> --client <client-stage> --controller <controller-stage>` |
| Sync schema, 1,000-node aggregation, 32 subscribers, coalescing, resync, ordering, and freshness | `go test ./internal/controlplane -run 'TestSyncFabric' -count=1` |
| Release static gates | `bash tests/release/static.sh` |
| Reproducible package and evidence gates | `make release-build OUTPUT_DIR=<empty-output-directory>` |

## Local source verification

| Check | Result |
|---|---|
| Full Go suite in Go 1.24 container: `go test ./...` | Passed; all tested packages passed. Command-only packages correctly reported no test files. |
| Full browser/UI regression suite: `npm --prefix web run test:ui` | Passed: 72 tests. |
| Web contract suite: `npm run web:check` | Passed: 24 tests. |
| Step 8/9, fleet sync, and Parser Catalog focused UI regression | Passed: 3 tests. |
| Late parser acknowledgement and recovery handoff Go regression | Passed. |
| Web production build | Passed; 9 public assets built. |
| Release quick-static and evidence contracts | Passed. |
| `git diff --check` | Passed. |

The final release artifact directory is accepted only after the reproducible
package builder and artifact verifier complete successfully from the tagged,
clean source revision. GitHub release publication remains gated by the same
checks plus Rocky Linux lifecycle jobs in the release workflow.

The 1,000-node test validates canonical state and subscriber behavior in-process. It does not establish production capacity, throughput, or p95 latency. Rocky Linux install/upgrade/rollback and target-fleet load qualification must be recorded separately when run; this document does not imply those deployment gates have passed.
