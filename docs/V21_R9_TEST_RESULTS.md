# Nexora Log Intelligence Ver21 R9 qualification

Release: `21.0.0-9`. Source/upgrade/rollback baseline: `v21.0.0-8`, commit `82ce826aaf32a34b7aa46127ffaa1f7064e77886`.

## Observed runtime evidence

- Offline pinned Go test suites passed for `internal/fieldmapping`, `parser-policy`, `internal/adapter` and `internal/registry`, including existing embedded corpus compatibility.
- The same four packages passed `go test -race` after completeness, dynamic output and explicit assignment handling were added. The explicit EOL selector also passed CR/LF/CRLF regression and affected-package race checks.
- The four reported mail records are rejected by a header-only managed mapping. With an approved dynamic `=` tail, the first two records normalize all observed pairs plus a previously unseen `future_key`. The malformed mixed grammar and colon-only input quarantine under that default. Explicit colon named/dynamic mappings pass source-completeness validation.
- Shared backend suggestions discover all observed suffix keys for semicolon, colon and pipe samples; ambiguous mixed assignments are not invented.
- Native dynamic fields survive a fixed static output schema only when actually produced by a configured flattened tail; other unknown fields retain schema rejection.

## Synchronization fixtures

Focused controlplane, controlclient and agent synchronization suites passed. `TestR9CatalogDeletionUnloadsActualClientAndRestoreStaysDraft` exercises real client HTTP bundle application, production adapter reload, managed file removal, preserved embedded profiles, tampered-file acknowledgement rejection and restored-draft validation. Legacy aliases map to one runtime identity. Replay rejects a parser-generation change before execution. The million-counter burst/coalescing fixture checks exact final aggregate transport state; it is not a million-record disk ingestion, Loki delivery or throughput-capacity benchmark.

## Local dashboard, UI and persistence evidence

- Browser coverage exercised all 45 workspaces and visible inner tabs, all eight appearance styles on desktop/mobile, tablet controls, and existing AI/provider/output/approval workflows. Three defects found in the first complete run were corrected: asynchronous wizard navigation, incremental queue updates, and empty custom-date controls. The final focused rerun passed all 14 affected/new cases, including dynamic EOL and combined generation success/failure, merge/remove-sources behavior, inventory/restore, selected client/range counts, late response rejection, and live updates without replacing a focused control.
- One million cumulative counter increments reached the telemetry publisher in bounded snapshots with exact final totals. This is a telemetry/coalescing test, not a million-line disk/Loki parsing throughput benchmark. Retained interval tests verify one million additional records, per-node isolation, process resets, restart persistence, decimal-string precision beyond JavaScript safe integers, and effective minute-boundary accounting.
- An isolated local PostgreSQL test applied genuine R8 SQL followed by R9, preserving existing document revisions and testing protected RPC access, dynamic EOL plans, parser-trash recovery metadata and repeated bootstrap application. The generated installation bootstrap renderer identifies R9 while retaining compatible RPC schema version 1.

## Release evidence

Final frozen-source build and qualification must bind `SOURCE_REVISION`, `SOURCE_BINARY_PROVENANCE.txt`, `RELEASE_MANIFEST.json` and `SHA256SUMS` to the published commit. Client/controller RPMs, WebApp/WebView archives, documentation and the release bundle are verified together. The historical R7/R8 documents describe their original qualification and are not rewritten as R9 results.

Final packaging, upgrade/rollback and complete UI evidence is recorded by the release gate before publication. No production deployment or throughput capacity claim is made by these local tests.
