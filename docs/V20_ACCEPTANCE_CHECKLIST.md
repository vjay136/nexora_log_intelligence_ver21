# NLI Ver20 acceptance checklist

Updated: 2026-11-01

Current package identity: `20.0.0-5`. Release 5 canonicalizes parser bundle digests across controller/client round-trips, permits legacy hash migration only after exact embedded-snapshot verification, refreshes QRW deployment state after approval/cancellation, and preserves the exact applied-hash replay gate. Approval reasons are bound to the exact proposal revision and canonical SHA-256. The responsive login banner displays the complete supplied artwork. This checklist does not mark deployment-specific production acceptance. See [Release 5 test results](V20_R5_TEST_RESULTS.md) and the [QRW operator guide](V20_R5_QRW_OPERATOR_GUIDE.md).

## Release 5 source-level gates

| Gate | State | Evidence or remaining action |
|---|---|---|
| Complete Go suite | PASS | `go test ./...` passed in the pinned `golang:1.23` container. |
| Complete Chromium UI suite | PASS | 69/69 scenarios, including current/legacy hashes, approval paths, rejected hashes, QRW refresh, notifications, and responsive login behavior. |
| Focused controller/client compatibility | PASS | Current, legacy, stale, malformed, emergency, independent, and client wire-hash cases passed. |
| JavaScript syntax | PASS | `node --check web/app.js` and `node --check web/ui.spec.js`. |
| Reproducible RPMs, package payload, Rocky lifecycle, published CI | NOT YET RECORDED | Record evidence after these gates run from the frozen Release 5 commit. |

## Release 4 historical acceptance record

The following Release 4 details are retained as historical evidence; they do not describe the current package identity.

| Gate | State | Evidence or remaining action |
|---|---|---|
| Operator and manager demo guide covers current navigation and sync behavior | PASS | `docs/V20_OPERATOR_DEMO_GUIDE.md`; `tests/v20-operator-guide.test.js` checks `NAV` IDs/titles |
| Parser catalog compatibility and 530 recovery | PASS | Focused store/controller suite covers legacy conversion, malformed-record isolation, bounded read retry, no write retry, approval gating, and publication rollback. |
| GPRS checksum, withdrawal, independent approval, publish, rollback, and offboarding lifecycle | PASS | Focused controller lifecycle and browser QRW tests pass. Signed rollout/client acknowledgement remains governed by independent approval and exact-hash gates. See `docs/V20_R1_QRW_OPERATOR_TEST_GUIDE.md`. |
| Quarantine Resolution Wizard field mapping and guided projection | PASS | Exact required fixture, unresolved delimiter gate, per-field evidence, deterministic candidate generation, controller persistence, nine stages, Back/Next controls, and invalid/resource-exceeding candidate rejection are covered. See `docs/V20_QUARANTINE_RESOLUTION_WIZARD.md`. |
| Release 3 upgrade baseline preserved | PASS | Release 4 lifecycle CI requires the genuine published Ver20 Release 3 client and controller RPMs. |
| NUPIRE discovery/compiler/lifecycle | PASS | Unit, persistence, API, integration, and browser tests |
| Declarative artifact and resource limits | PASS | Compiler validation and stable error-code tests |
| Held-out evidence separation | PASS | Lifecycle and NAPI bridge tests |
| Approval/canary/promotion/drift/rollback | PASS | State-machine and controller API tests |
| Existing hot path and signed rollout preserved | PASS | Complete Go suite and architecture boundary review |
| Full Go suite and vet | PASS | `go test ./...` and `go vet ./...` passed with the repository's Go 1.25.4 toolchain |
| Browser suite | PASS | 66/66 Chromium scenarios passed across every primary workspace and visible subtab, including QRW operator field naming, terminal badge clearing, catalog compatibility/repair, sidebar order, and canonical badges. |
| Hosted deployment contracts | PASS | 24/24 Node contracts; standalone and Vite production builds passed. |
| Fuzz target | PASS | 110,438 executions / 3 seconds |
| Shell and installer contracts | PASS | Syntax, client, AWS/controller edge, provenance, SELinux fixture |
| Race detector | PASS | Complete Linux `go test -race ./...` passed in the pinned Go container. |
| 100 distinct upstream source reviews | PASS | 100 verified across the consolidated ledger and evidence batches; mirrors, aliases, duplicates, and incomplete inspections excluded |
| Deterministic RPM build and verification | PENDING | Runs from the frozen Release 4 commit in local release build and tag CI |
| Rocky 9/10.2 fresh install, upgrade, rollback | PENDING | Runs from the frozen Release 4 artifacts against genuine Release 3 RPMs in tag CI |
| Two-node TLS/enrollment lifecycle | PENDING | Linux release integration gate |
| Reproducible artifact comparison | PENDING | Two-pass comparison runs from the frozen Release 4 commit |
| Same-host legacy/Ver20 parser microbenchmark | PASS | Median hot-path timings level; see `evidence/V20_V18_LOCAL_PARSER_BENCHMARK.md` |
| Linux production throughput qualification | PENDING | Pinned workload required before production performance claim |

Release decision: **source release approved for tagging by the operator** after
the complete Go, browser, vet, Vite, and static packaging gates pass. Linux RPM
reproducibility, Rocky install/upgrade, race, multi-node TLS, and production
throughput qualification remain blocked or pending and must pass before the
release is described as production-qualified.
