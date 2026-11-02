# Nexora Log Intelligence Ver21 Release 3

Release identity: `21.0.0-3`  
Git tag: `v21.0.0-3`

## Release identity and upgrades

The public RPM aliases are `nli_client_ver21.rpm` and `nli_controller_agent_ver21.rpm`; the release archive is `nli-v21.0.0-3-x86_64.tar.gz`. WebView and webapp bundles are `webview_ver21.zip` and `webapp_ver21.zip`. Installer, uninstaller, AWS, and Cloudflare helper names remain Ver21-specific. RPM package names (`nli-client`, `nli-controller-agent`), systemd unit names, configuration paths, durable state paths, and the Go module path remain stable for in-place upgrades. The installed WebView stays at `/usr/share/nli/webview_ver21`.

## Fleet synchronization and telemetry

The authenticated `nli.sync.v1` client publication carries node and process identity, a monotonic sequence, collection/publication timestamps and intervals, a full-snapshot marker, and changed source telemetry. The canonical snapshot exposes a schema version, revision, generation time, aggregate counters/rates, current source samples, and synchronization diagnostics. Diagnostics include tracked nodes/sources, freshness, duplicate/out-of-order/gap/reconnect counts, coalesced updates, rejected batches, and observed processing/delay percentiles where clocks permit.

Clients publish through `POST /api/v1/node/stream-telemetry` (up to 256 source samples per request). Operators can read the authoritative snapshot and diagnostics from `GET /api/v1/sync/snapshot` and `GET /api/v1/sync/diagnostics`; `GET /api/v1/sync/stream` starts with a full `sync_snapshot`, then sends revisioned deltas, critical transitions, and heartbeats. The versioned batch sequence is scoped to the node and process generation so retries, gaps, and restarts are distinguishable.

The controller retains bounded latest state for up to 10,000 sources. Each stream subscriber has a one-item latest-state queue; slow readers receive a coalesced update marked for authoritative resynchronization rather than blocking ingestion. The existing in-process sync contract test publishes 1,000 nodes and 32 concurrent subscribers. That is a correctness test, not a production capacity or latency qualification. Capacity depends on source counts, publication rate, storage, controller resources, and browser subscribers; qualify those limits on the target deployment.

Raw log bodies and parsed customer fields are not part of the sync schema. Missing measurements remain unavailable instead of being synthesized as zero.

See `V21_R3_TEST_RESULTS.md` for the release verification record and commands.
