# Nexora Log Intelligence Ver21 Release 4 Test Results

Release identity: `21.0.0-4`

Planned Git tag: `v21.0.0-4`

Previous release baseline: `v21.0.0-3`

Verified implementation revision: `be028bb`

## Verification record

The completed source and local artifact gates below were run against implementation revision `be028bb`. The release build produced both RPMs and the versioned archive; checksum verification, installer exact-source checks, release-document packaging, and reproducibility verification passed. The exhaustive browser suite passed `73/73`. Rocky Linux 9.6 and 10.2 lifecycle validation passed, followed by the isolated two-node TLS, enrollment, packaged-WebView, certificate-discovery, and fallback acceptance scenario. External GitHub/Vercel publication remains explicitly pending until the final evidence commit is pushed.

| Check | Result |
|---|---|
| Web source and contract checks | PASS — `npm run web:check`; exit `0`; `28/28` checks passed |
| Release infrastructure static checks | PASS — `bash tests/release/static.sh full`; exit `0` |
| Go unit, vet, and race checks | PASS — pinned Go `1.24.9`; `go test ./...`, `go vet ./...`, and `go test -race ./...` passed across 63 packages |
| Exhaustive UI behavior | PASS — `npm --prefix web run test:ui`; exit `0`; `73/73` Playwright tests passed, including all pages and inner tabs, QRW metadata-only evidence recovery, stages 6–9, replay gates, notification-count clearing, parser Published / Active state, SSE reconnect/fallback, and mobile navigation |
| Release document packaging contract | PASS — source documents were verified byte-identical in the release output and both RPM payload staging trees |
| Evidence generator deterministic contract | PASS — exercised by the full release infrastructure static gate |
| Release quick-static contract | PASS — exercised by the full release infrastructure static gate |
| Runtime parser proof from built artifacts | PASS — the Rocky artifact matrix exercised the packaged client parser and controller path with exact input/output accounting and no duplicate normalized records; the isolated two-node acceptance then verified live enrollment, parser/control services, controller health, and no injected sample telemetry |
| Reproducible RPM and artifact pipeline | PASS — `nli_client_ver21.rpm`, `nli_controller_agent_ver21.rpm`, `nli-v21.0.0-4-x86_64.tar.gz`, `SHA256SUMS`, installer exact-source checks, and reproducibility verification passed at `be028bb` |
| Rocky Linux install, upgrade from R3, rollback, and re-upgrade | PASS — `tests/rocky/validate-release-matrix.sh` completed on Rocky Linux 9.6 and 10.2, including clean install, R3→R4 upgrade, rollback/re-upgrade, service handling, and parser accounting; `tests/rocky/validate-two-node-webview.sh` passed the final Rocky 10.2 controller/client TLS and packaged-WebView scenario (`6/6` browser tests) |
| GitHub and Vercel tag, release, and workflow publication | PENDING FINAL GATE |

Completed source, browser, build, artifact, Rocky lifecycle, and two-node gates establish integrity for the tested source and artifacts only. They do not establish external publication or production capacity. No production capacity, throughput, or latency claim is made by this report. Pending rows remain release blockers until their exact commands, exit status, evidence, and limitations are recorded here.
