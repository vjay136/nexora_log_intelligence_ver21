# Nexora Log Intelligence Ver21 R8 test results

Release: `21.0.0-8`. Source and rollback baseline: `v21.0.0-7`, commit `bc6b17a20e1a7b2c11903df440a14ad9134697d8`.

## Source qualification

| Gate | Observed result |
| --- | --- |
| Read-only output audit | PASS: five tests cover historical delta 49=47+2, genuine extra output, invalid/in-flight/truncated uncertainty and lazy empty quarantine; no raw-content deduplication or output rewrite |
| Required-remote retry and canonical quarantine accounting | PASS in full agent/output/routing suites: source-occurrence receipt/WAL, interrupted-append recovery, conflict rejection, distinct equal records and single canonical quarantine writer. Remote at-least-once ambiguity is retained. |
| Eight registry entries, tokens and resolver | PASS: four unit contracts include 8 styles × 2 modes × 5 accents with 4.5:1 text/surface targets |
| Instant preview/apply/revert/default, mode/accent persistence | PASS: focused Chromium and Firefox cases; native 12383 controller proof covers 16 palettes, guarded draft save and no operational reposts (ordinary heartbeat remains allowed) |
| Existing route design matrix | PASS: 720 fixture cases (45 routes × 8 styles × dark desktop/light mobile); no root overflow. Tablet: 64 dense-page checks and 16 Design Studio dialog variants PASS. |
| Responsive/focus/contrast/motion/fallback | PASS for measured desktop/mobile/tablet and focused Chromium/Firefox; representative light chart series meet 3:1 contrast and retain exact hover labels. No universal WCAG or WebKit qualification is claimed. |
| Complete Go suite and vet | PASS: unfiltered `go test -timeout=20m ./... && go vet ./...` in pinned cached Go container, offline network and read-only source mount. Full log recorded in source qualification evidence. This checkpoint predates the final additive-pool/replay compatibility follow-ups; frozen CI must qualify those changes. |
| Adaptive engine contracts/Vite/source export | PASS: four design unit contracts, Vite/static 11-asset build and clean WebApp export; root web contract gate also passed all 45 checks at the local checkpoint |
| Ordinary UI regression coverage | 89 current scenarios covered across the full 88-test snapshot (86 initially passed), corrected chart-measurement rerun, unchanged legacy API-timing rerun and new tablet test. This was not a single clean 89-test run; frozen CI qualification remains pending. |
| Final full race and additive-pool/replay follow-ups | PENDING final frozen-source qualification. The earlier local full Go/vet/race checkpoints do not establish these later changes. |
| Release infrastructure contracts | PASS: static release invariants, source asset coverage, evidence fixture, bootstrap renderer and staged six-document/operator-helper packaging contracts |
| WebApp/WebView source assets and hashes | PENDING export/package verification |
| R8 bootstrap identity, RPCv1 compatibility and R7 metadata preservation | PASS: renderer release-identity guard; pinned local PostgreSQL fresh/rerun and separate-database genuine R7 SQL→R8 upgrade; unchanged prior document revisions and exact nested/null/array/object metadata |
| Reproducible two-pass genuine RPM/archive/evidence | PENDING frozen release pipeline |
| Rocky 9/10.2 R7→R8 lifecycle and packaged two-node TLS/browser | PENDING genuine R7 RPM baseline qualification |

## Artifact evidence

`SOURCE_REVISION`, `RELEASE_ID` and `RELEASE_MANIFEST.json` bind payloads to frozen R8 source. `SHA256SUMS` covers packaged payloads; the archive has a separate checksum sidecar. Final workflow results and downloaded artifact verification must be recorded before publication is described as qualified. No earlier R7 success establishes R8 correctness or live deployment status.


A new packaged browser scenario exercises eight styles × two modes, preserved draft DOM, no operational reposts and native mapping verification. Its targeted authoring-side run against the existing R7 controller passed in 26.4 seconds. The actual frozen R8 RPM/two-node packaged run remains pending; that earlier targeted result is not a substitute.


The final native-controller UI proof passed in 43.1 seconds across 16 style/mode variants, with guarded form save and unchanged DOM count (801→801). Measured appearance-switch paint latency in that proof had a 68.8 ms median and 169.8 ms maximum. These are one browser/controller/workload observations, not a universal interaction-performance guarantee. Nineteen final screenshots cover application variants, Design Studio and saved-draft state; final release evidence binds them to the qualified source rather than implying a live deployment.


The automatic-runtime focused gate passed (`TestAutomatic`, structural suggestions, FieldSense lifecycle and raw-source compatibility), including published refresh, draft exclusion, rollback eviction, immutable dependency copies, read-only suggestions, identical-alias deduplication, conflicting-ID isolation, strict fleet protection and client/controller CRLF/trailing-space parity. Controlplane vet passed. The final targeted automatic-runtime race gate passed in 15.289 seconds, including protected IDs, SMTP compatibility summary and exact-source parity; legacy normalization/helper, structural suggestion and FieldSense contracts passed in 1.930 seconds, followed by controlplane vet. Frozen CI still must qualify the unfiltered final source. The local full-auth race attempt timed out after 20 minutes under parallel workloads; it is not reported as PASS.


Final runtime-focused normal/race/vet checks passed for output, quarantine, routing, agent and CLI. Regression coverage includes `TestReplayRequiredRemoteRetryDoesNotDuplicateLocalPayload`, `TestLiveAgentAppendsSurviveReplaySnapshotAndQueueReplacement`, `TestConcurrentWriterInstancesShareReceiptsWithoutLosingOtherOwners` and `TestPrivilegedWriterPreservesAgentOwnership`. UID 1001/GID 1002 ownership/compaction checks executed without skip in the offline capabilities fixture; Windows test-binary cross-compilation passed, without claiming Windows runtime crash-durability qualification. Final append measurements are recorded in the output accounting guide; the buffered append-lock reference is not an R7 binary benchmark.


AI integration focused fake-HTTP Go/UI checks passed for the existing eight provider kinds, protected key sources, compatible JSON/SSE probes, same-group routing and diagnostic review. No live key or paid provider call was used. Final frozen source and packaged tests remain authoritative; supported adapter protocol does not certify live availability of the user's example NVIDIA models. See `V21_R8_AI_PROVIDER_GUIDE.md` for exact budgets and readiness limits.


The mock UI source inventory reached 94 scenarios after automatic-pool, AI-help and credential-removal additions. Earlier local counts above describe their observed checkpoints and reruns; a clean final inventory result must come from the frozen-source UI gate rather than being inferred from the scenario count. The later schema-aware AI checkpoint below covers the expanded local feature; frozen CI must qualify the final complete inventory.

The actual R8-controller AI browser proof passed in 15.5 seconds using a local fake provider: two NVIDIA profiles sharing an environment reference, base/full endpoint JSON and SSE/image probe responses, incomplete reasoning-only acknowledgement, HTTP 401, draft configuration binding without save, and reload with blank password/no returned key values. Affected Go package checks passed (controlplane 137 seconds), focused AI race checks passed (controlplane 37.653 seconds), and six mocked AI UI cases passed. Two additional structural-context regressions passed in 0.746 seconds, covering native proposals when actual parsing is empty, cancellation, masking and no evidence mutation. These are local synthetic-service checkpoints; final frozen CI requires its own qualification. The later removal proof below supersedes the earlier browser checkpoint for that expanded scenario.

The AI structural-context privacy follow-up passed its focused review and configured-fallback masking checks in 1.050 seconds. The optional FieldSense context uses a structural metadata whitelist and excludes preview/example/default/raw values; value-bearing raw/actual/desired channels remain separately masked.

Final profile-removal qualification passed: five focused Go tests in 2.146 seconds cover purge, missing-ID HTTP 400, sibling preservation, re-enable without a key HTTP 400, and delayed probes unable to resurrect cleared/deleted credentials. The mocked UI lifecycle passed in 3.8 seconds. The latest actual local R8 controller with fake NVIDIA-compatible service passed in 16.2 seconds across two profiles, JSON/SSE responses, incomplete reasoning, invalid key, clear → disabled → delete and sibling preservation. No live provider call or external secret revocation was performed. Critical review, final focused race and frozen release CI remain separate qualification gates.

Credential removal follow-ups passed: two focused race cases in 9.035 seconds and the exact legacy-case regression in 1.307 seconds. An exact uppercase legacy ID cannot delete a distinct lowercase sibling; newly created profile IDs use the normal lowercase convention.

Schema-aware explicit AI mapping qualification passed: five focused backend tests in 3.841 seconds cover the public compound-log 12-field output, reflected schema, privacy/no writes, stripped confirmations and operator-confirmed draft validation; valid JSON mapping and reviewable desired mismatch; rejection of invalid/trailing/unknown-property/JavaScript/RE2/missing-selector results; unavailable/disabled provider, raw bounds, NUL and two-slot admission. Parent aggregate context/syntax/privacy/proposal/offboard checks passed seven tests in 2.676 seconds, including unchanged real published mapping state after proposal rejection/removal and explicit offboard rollout governance.

The latest actual local R8 controller with fake provider passed both AI browser scenarios in 22.9 seconds. The new scenario preserved manual fields, showed an ephemeral five-field proposal, required explicit import and operator delimiter confirmation, then natively verified `SieveRuleMsg` and cleaned up the synthetic profile. Three new mocked UI scenarios passed in 11.2 seconds, and an enhanced authoring handoff passed in 5.9 seconds. Current mock UI inventory is 97; this is not a claim that all 97 passed in one final frozen run. Backend final focused race/vet and complete frozen CI remain separate gates.
