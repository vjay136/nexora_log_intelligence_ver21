# Nexora Log Intelligence Ver21 Release 1 — Test Results

Release identity: `21.0.0-1`  
Report date: 2026-11-02  
Evidence rule: PASS is recorded only for a command executed against this working tree or for a deterministic source invariant inspected here.

## Verification performed for the Ver21 release candidate

| Gate | Result | Evidence |
|---|---|---|
| Runtime identity | PASS | `internal/engine/engine.go` declares `21.0.0`. |
| Release identity | PASS | `packaging/release.env` declares `NLI_VERSION=21.0.0` and `NLI_RELEASE=1`. |
| RPM identity compatibility | PASS | Both RPM specs retain `Name: nli-client` and `Name: nli-controller-agent` while their default version/release and repository metadata identify Ver21. |
| Go module compatibility | PASS | `go.mod` remains `module nexora.local/nli-v20`; imports were not mass-renamed. |
| WebView/state compatibility | PASS | `/usr/share/nli/webview_ver20`, `/etc/nli`, systemd unit names, and durable state paths were not renamed. |
| Node/web deployment identity | PASS | root package metadata and generated web-only manifest identify `nexora-log-intelligence-v21` / `21.0.0`. |
| Embedded parser catalog product identity | PASS | Embedded and canonical catalog copies identify `Nexora Log Intelligence Ver21`. |
| Source whitespace/diff integrity | PASS | `git diff --check` completed without whitespace errors for this change set. |
| Residual identity scan | PASS with documented compatibility exceptions | `rg` confirms authoritative Ver21 fields; remaining `v20` strings are the preserved module, WebView/artifact aliases, historical docs/tests, or explicit compatibility references. |
| Complete Go test suite | PASS | `go test ./...` completed in the pinned Go 1.24 Linux builder, including the parser lifecycle, exact-hash rollout, authentication, AI provider, and synchronization packages. |
| Go static analysis | PASS | `go vet ./...` completed in the pinned Go 1.24 Linux builder. |
| Go race detector | PASS | `go test -race ./...` completed across the repository; the broad control-plane package completed in 412.583 seconds. |
| Web static and contract tests | PASS | `npm run web:check` completed with 24/24 tests passing. |
| Browser workspace and inner-tab suite | PASS | `npm run web:test:ui` completed with 70/70 Playwright tests passing. |
| Production web build | PASS | `npm run build:vite` produced the production bundle successfully. |

## Gates that must pass before publishing binary artifacts

These gates are release requirements, not claimed as completed by the documentation/version-only change:

| Gate | Current report status | Required evidence |
|---|---|---|
| Complete Go test suite | PASS locally; repeated by release CI | `go test ./...` in the pinned Go builder. |
| Race suite | PASS locally; repeated by release CI | `go test -race ./...` in the pinned Go builder. |
| Fuzz/contract suite | PENDING release CI | Configured bounded fuzz and API contract tests. |
| Web static/contract tests | PASS locally; repeated by release CI | `npm run web:check`. |
| Browser workspace/inner-tab suite | PASS locally; repeated by release CI | Playwright suite against a controller fixture. |
| Two-pass reproducible build | PENDING release CI | `packaging/build-release.ps1` or the Linux release target with byte comparisons. |
| RPM payload/metadata inspection | PENDING artifact build | `rpm -qpi`, `rpm -qpl`, digest, permissions, and dependency checks. |
| Rocky Linux lifecycle | PENDING release qualification | Fresh install, upgrade from the selected Ver20 baseline, restart, rollback, re-upgrade, and uninstall on every supported major image. |
| Two-node TLS and WebView | PENDING release qualification | Controller plus origin client enrollment, parsing, canonical telemetry, exact bundle acknowledgement, replay, and hosted/controller WebView checks. |
| Load/capacity qualification | SITE-SPECIFIC | Measured throughput, latency, memory, disk, queue, history, and destination durability on production-equivalent hardware. |

## Required end-to-end acceptance scenarios

1. **Known log path:** ingest a fixed line count; prove `input = normalized + quarantined + errors` according to the documented accounting contract, with no duplicate normalized output after restart.
2. **Unknown log path:** retain one unstructured record, inspect exact evidence, supply operator field names, generate a deterministic candidate, and obtain at least one passing fixture.
3. **Governed activation:** submit/publish the exact candidate, approve the exact canonical bundle according to policy, roll it to the origin client, and confirm applied revision/hash equality.
4. **Safe replay:** replay only after acknowledgement, prove one terminal result for the selected record, and verify the new parser remains available to subsequent matching live input.
5. **Rollback/offboarding:** remove or roll back the new parser through a governed bundle and prove prior parser behavior remains intact.
6. **Synchronization isolation:** create/save/approve parser work while a separate client continues to publish telemetry; prove snapshot and SSE remain responsive and no full-page refresh is required.
7. **Authentication:** anonymous session bootstrap must not create a false system-issue notification; login, logout, expiry, and role changes must produce bounded, accurate state.
8. **Notification lifecycle:** actionable counts clear when work reaches a terminal state; read/check-later/dismiss operations affect presentation without deleting audit evidence.
9. **Destination delivery:** distinguish local parse success from remote acknowledgement and exercise retry/backlog for every enabled destination adapter.
10. **Recovery:** interrupt the event stream and controller briefly; prove bounded reconnect/recovery restores the canonical revision without fabricating live values.

## Release decision

The Ver21 source candidate passes its complete local Go, race, vet, web-contract, browser, and production-build gates. Binary publication remains conditional on reproducible artifact/RPM verification and the tag-triggered release CI; Rocky lifecycle, two-node TLS/WebView, and site-capacity qualification remain environment-specific acceptance gates. This report intentionally makes no universal “100% accuracy” or “10 million events/second” claim.

