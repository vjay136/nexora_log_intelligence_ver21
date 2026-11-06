# Nexora Log Intelligence Ver21 R7 test results

Release: `21.0.0-7`. Verified source and rollback baseline: R6 `v21.0.0-6`, commit `905ce45bca9d54d2fd8fcf4505ba21e3e274a611`.

## Source qualification

| Check | Observed result |
| --- | --- |
| Complete Go suite, `go vet ./...` and full Linux `-race ./...` | PASS in pinned Go 1.24.9 container; final follow-up additionally covered by the published source workflow |
| Native mapping selectors, ordered operations, types, null/default, bounds and malformed input | PASS; includes Unicode byte boundaries, quoted CSV/KV, nested MISREPORT, duplicate policies and exact requested SMTP example |
| Preview → saved candidate → administrator review → publication → production parity | PASS; same native plan and exact typed JSON output |
| Client bundle installation and runtime execution | PASS; exact hash acknowledgement, atomic install, tampered bundle rejection |
| Original-source and service isolation | PASS; whitespace positions, scoped JSON mapping, unrelated JSON fast path, truncated mapping quarantine, canonical service inference and guarded global override |
| Operator corpus qualification | PASS: all 51 private records execute through individual and merged corpus plans across seven structural profiles; corpus content excluded from published source/assets |
| Mapping fuzzing | PASS: initial 163,550 and final 133,221 executions in bounded six-second runs |
| Affected mapping/runtime/client race checks | PASS; adapter and engine race checks also passed after source-parity fixes |
| Browser UI scenarios | All 81 scenarios passed across the full run and focused reruns; a trace-output collision required a unique-directory rerun |
| JavaScript contracts, syntax, Vite build and clean web-only export | PASS: 39 contract tests; Vite production build and export succeed |
| Release scope/provenance | R6 artifacts and tag preserved; original native implementation, no new parser dependency |

The packaged-controller browser gate now traverses all 45 primary workspaces and includes a real NFI compound-token/colon-KV candidate-save scenario. Packaged acceptance is recorded after building the frozen revision.

## Scoped benchmark

Same synthetic twelve-field mail record and identical output through compiled ParserPack executors, three runs: R6 regex 72.3–76.0 µs/op, approximately 53,475 B/op and 576 allocations; NFI 24.6–25.3 µs/op, approximately 15,290 B/op and 153 allocations. This is a comparison of one fixture and execution path, not a fleet capacity or universal speed claim. Core mapping-only execution is measured separately and is not directly comparable with the complete executor.

## Frozen-revision artifact qualification

Release publication requires complete Go race checks, reproducible two-pass genuine RPM/WebView builds, archive/checksum/provenance verification, and Rocky 9/10.2 R6→R7 upgrade/restart/rollback/state-preservation plus real two-node TLS/browser acceptance. These run against the frozen source after this document is committed. Their observed results, source revision, artifact hashes and GitHub workflow outcome are recorded in the accompanying `R7_QUALIFICATION.json` release asset and build logs. No result is inferred merely from an earlier R6 success.

`SOURCE_REVISION`, `RELEASE_ID` and `RELEASE_MANIFEST.json` bind artifacts to the frozen source. A WebApp source push alone does not prove a functioning live Vercel deployment. No live-fleet, universal-format or capacity guarantee follows from these regression checks.
