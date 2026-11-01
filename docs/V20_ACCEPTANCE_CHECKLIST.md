# NLI Ver20 acceptance checklist

Updated: 2026-11-01

Current package identity: `20.0.0-6`. Release 6 is based on the R4 parsing/runtime behavior and fixes canonical parser bundle hash handling across proposal creation and approval. This note does not mark deployment-specific or RPM lifecycle production acceptance.

| Gate | State | Evidence or remaining action |
|---|---|---|
| Operator and manager demo guide covers current navigation and sync behavior | PASS | `docs/V20_OPERATOR_DEMO_GUIDE.md`; `tests/v20-operator-guide.test.js` checks `NAV` IDs/titles |
| Parser catalog compatibility and 530 recovery | PASS | Focused store/controller suite covers legacy conversion, malformed-record isolation, bounded read retry, no write retry, approval gating, and publication rollback. |
| GPRS checksum, withdrawal, independent approval, publish, rollback, and offboarding lifecycle | PASS | Focused controller lifecycle and browser QRW tests pass. Signed rollout/client acknowledgement remains governed by independent approval and exact-hash gates. See `docs/V20_R1_QRW_OPERATOR_TEST_GUIDE.md`. |
| Quarantine Resolution Wizard field mapping and guided projection | PASS | Exact required fixture, unresolved delimiter gate, per-field evidence, deterministic candidate generation, controller persistence, nine stages, Back/Next controls, and invalid/resource-exceeding candidate rejection are covered. See `docs/V20_QUARANTINE_RESOLUTION_WIZARD.md`. |
| Release 4 upgrade baseline preserved | PASS | Release 6 lifecycle CI must use the genuine published Ver20 Release 4 client and controller RPMs. |
| NUPIRE discovery/compiler/lifecycle | PASS | Unit, persistence, API, integration, and browser tests |
| Declarative artifact and resource limits | PASS | Compiler validation and stable error-code tests |
| Held-out evidence separation | PASS | Lifecycle and NAPI bridge tests |
| Approval/canary/promotion/drift/rollback | PASS | State-machine and controller API tests |
| Existing hot path and signed rollout preserved | PASS | Complete Go suite and architecture boundary review |
| Full Go suite | PASS | `go test ./...`; all packages passed. |
| Browser suite | PASS | 69/69 UI scenarios passed across the covered workspaces, including QRW, approvals, parser catalog, notifications, navigation badges, and subsystem tabs. |
| Web contracts | PASS | 24/24 Node web contract scenarios passed. |
| Fuzz target | PASS | 110,438 executions / 3 seconds |
| Shell and installer contracts | PASS | Syntax, client, AWS/controller edge, provenance, SELinux fixture |
| Race detector | PASS | Complete Linux `go test -race ./...` passed in the pinned Go container. |
| 100 distinct upstream source reviews | PASS | 100 verified across the consolidated ledger and evidence batches; mirrors, aliases, duplicates, and incomplete inspections excluded |
| Deterministic RPM build and verification | PENDING | Must run from the frozen Release 6 commit and verify package identity `20.0.0-6`. |
| Rocky 9/10.2 fresh install, R4 upgrade, rollback | PENDING | Must run from frozen Release 6 artifacts against genuine Release 4 RPMs in tag CI. |
| Two-node TLS/enrollment lifecycle | PENDING | Linux release integration gate |
| Reproducible artifact comparison | PENDING | Two-pass comparison runs from the frozen Release 6 commit. |
| Same-host legacy/Ver20 parser microbenchmark | PASS | Median hot-path timings level; see `evidence/V20_V18_LOCAL_PARSER_BENCHMARK.md` |
| Linux production throughput qualification | PENDING | Pinned workload required before production performance claim |

Release decision: **source and UI gates recorded as passing** for the listed
Go, browser, and web-contract results. RPM reproducibility, Rocky install/upgrade,
race, multi-node TLS, and production throughput qualification remain pending and
must pass before the release is described as package/lifecycle or production
qualified.
