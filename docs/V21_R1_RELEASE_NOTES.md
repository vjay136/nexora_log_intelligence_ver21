# Nexora Log Intelligence Ver21 Release 1

Release identity: `21.0.0-1`  
Release date: 2026-11-02

## Outcome

Ver21 Release 1 promotes the governed parser runtime and control plane to the Ver21 product line without renaming installed RPM packages, services, state paths, or the established Go module. The compatibility boundary allows an in-place RPM upgrade while the authoritative archive, RPM metadata, binary version output, embedded catalog identity, Node/web deployment metadata, and product documentation report Ver21.

## Included behavior

- Normal client parsing remains independent of parser-authoring and assurance workloads.
- Canonical controller telemetry feeds Command Center, Monitoring, subsystem workspaces, and external metrics from one source of truth.
- Assurance Lab and the nine-stage Quarantine Resolution Wizard retain deterministic candidate validation, review, exact-bundle rollout, acknowledgement-gated replay, and result verification.
- Parser Catalog exposes draft, approved, active, blocked, repair, download, delete/offboard, and runtime-compatibility state without treating approval alone as deployability.
- Fleet Sync records desired and applied revision/hash state per target.
- RBAC, audit, notification state, tab badges, and retained-evidence permissions remain controller-owned.
- Optional AI assistance remains advisory and cannot bypass deterministic validation, authorization, publication, rollout, or replay gates.

See [`All_NLI_features.md`](../All_NLI_features.md) for the complete implemented inventory and boundaries.

## Compatibility contract

The following names intentionally remain unchanged:

| Surface | Stable value |
|---|---|
| Go module | `nexora.local/nli-v20` |
| Client RPM | `nli-client` |
| Controller RPM | `nli-controller-agent` |
| WebView installation root | `/usr/share/nli/webview_ver20` |
| Main configuration root | `/etc/nli` |
| Client/controller service names | Existing `nli-*` systemd units |

The release archive is `nli-v21.0.0-1-x86_64.tar.gz`. Compatibility artifact aliases that contain `ver20` may remain in the archive; RPM metadata and `RELEASE_ID` are authoritative.

## Upgrade boundary

1. Back up `/etc/nli`, controller/client state, TLS material, and the current verified release artifacts.
2. Verify the Ver21 archive `.sha256` and internal `SHA256SUMS`.
3. Inspect `RELEASE_ID`; it must contain `NLI_VERSION=21.0.0` and `NLI_RELEASE=1`.
4. Use the included installers in verify-only mode before installation.
5. Upgrade the controller first, then a canary client, then the remaining clients.
6. Confirm service health, login, canonical snapshot/SSE, parser bundle desired/applied hashes, ordinary parsing, quarantine, and output delivery before expanding the rollout.
7. Keep the prior verified RPMs until rollback has been rehearsed on the target OS image.

## Evidence boundary

Historical Ver20 reports are retained as evidence for the behavior they actually tested. They do not by themselves prove Ver21 packaging, upgrade, Rocky lifecycle, or production capacity. Ver21 results are recorded only in [`V21_R1_TEST_RESULTS.md`](V21_R1_TEST_RESULTS.md); incomplete gates remain explicitly pending.

## Known deployment responsibilities

- Configure a trusted TLS endpoint and preserve same-origin `/api/v1/*` proxy semantics for hosted WebApps.
- Keep controller, Supabase, output, enrollment, TLS, and AI secrets server-side.
- Validate output acknowledgements against the selected destination adapter.
- Size queue, retention, disk, CPU, and memory for the actual source mix and throughput.
- Treat a quiet source as zero activity, not automatically as a synchronization failure.

