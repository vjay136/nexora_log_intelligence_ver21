# Nexora Log Intelligence Ver21 R7 test results

Release: `21.0.0-7`. Verified source and rollback baseline: R6 `v21.0.0-6`, commit `905ce45bca9d54d2fd8fcf4505ba21e3e274a611`.

## Source qualification

| Check | Observed result |
| --- | --- |
| Final expanded complete Go suite and `go vet ./...` | PASS in pinned Go 1.24.9 container after FieldSense runtime/API freeze; all packages tested without filtering, including full controlplane and agent/runtime suites |
| Full Linux `-race ./...` | Earlier R7 baseline PASS; final expanded full-race qualification is recorded by the frozen-source workflow and final qualification evidence |
| Native mapping selectors, ordered operations, types, null/default, bounds and malformed input | PASS; includes Unicode byte boundaries, quoted CSV/KV, nested MISREPORT, duplicate policies and exact requested SMTP example |
| Preview → saved candidate → administrator review → publication → production parity | PASS; same native plan and exact typed JSON output |
| Supabase generated bootstrap and persistence | PASS: R7 header and generated-file instructions, release-identity guard, fresh install, rerun, genuine R6 SQL upgrade, unchanged R6 revisions, exact NFI plan/scope/null/array/object roundtrip, controller authentication and direct-table denial |
| Client bundle installation and runtime execution | PASS; exact hash acknowledgement, atomic install, tampered bundle rejection |
| Original-source and service isolation | PASS; whitespace positions, scoped JSON mapping, unrelated JSON fast path, truncated mapping quarantine, canonical service inference and guarded global override |
| Operator corpus qualification | PASS: all 51 private records execute through individual and merged corpus plans across seven structural profiles; corpus content excluded from published source/assets |
| Mapping fuzzing | PASS: initial targets 163,550 and 133,221 executions; optional-field fix 109,086; expanded toolbox 23,737 and hostile configuration target 111,913 executions. Bounded runs, not exhaustive proof |
| Affected mapping/runtime/client race checks | PASS; adapter and engine race checks also passed after source-parity fixes |
| Browser UI scenarios | Expanded FieldSense suite: all 83 scenarios PASS, including all 45 workspaces, guided/expert/manual controls, context help, live output, suggestion review and isolated published-parser repair; earlier 81-scenario baseline also passed |
| JavaScript contracts, syntax, Vite build and clean web-only export | PASS: 40 contract tests; Vite production build and export succeed |
| Release scope/provenance | R6 artifacts and tag preserved; original native implementation, no new parser dependency |

The packaged-controller browser gate traverses all 45 primary workspaces and includes real compound-token/colon-KV candidate-save and manual word/help/native-trace scenarios. Packaged acceptance is recorded after building the frozen revision.

Expanded FieldSense acceptance additionally verifies native email conversion, SHA-256, filtered all-key/value extraction, Nth boundary selection, intermediate fields and conditional merge through preview, guarded save, administrator approval, sealed fleet bundle and the actual agent parser. Repair acceptance preserves the original published revision and bundle while editing a distinct draft, rejects submission before revalidation, selects the approved correction by reviewed priority, and restores the exact original bundle on rollback. Saturation/deadline/cancellation tests verify bounded analysis slots recover without an input queue; production parsing remains trace-free.

## Scoped benchmark

Same synthetic twelve-field mail record and identical output through compiled ParserPack executors, three final expanded-engine runs under concurrent qualification: R6 regex 76.0–105.3 µs/op, 53,486–53,492 B/op and 576 allocations; NFI 27.9–31.9 µs/op, 16,483 B/op and 156 allocations. Both requests use the production-normalized service; this prevents an invalid comparison against an ineligible parser rule. This is one fixture and execution path, not a fleet capacity or universal speed claim. Core mapping-only execution is measured separately and is not directly comparable with the complete executor.

## Frozen-revision artifact qualification

Release publication requires complete Go race checks, reproducible two-pass genuine RPM/WebView builds, archive/checksum/provenance verification, and Rocky 9/10.2 R6→R7 upgrade/restart/rollback/state-preservation plus real two-node TLS/browser acceptance. These run against the frozen source after this document is committed. Their observed results, source revision, artifact hashes and GitHub workflow outcome are recorded in the accompanying `R7_QUALIFICATION.json` release asset and build logs. No result is inferred merely from an earlier R6 success.

`SHA256SUMS` covers the packaged payloads, and the archive has a separate SHA-256 sidecar. `SOURCE_REVISION`, `RELEASE_ID` and `RELEASE_MANIFEST.json` bind artifacts to the frozen source. A WebApp source push alone does not prove a functioning live Vercel deployment. No live-fleet, universal-format or capacity guarantee follows from these regression checks.
