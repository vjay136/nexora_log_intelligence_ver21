# Nexora Log Intelligence Ver21 — Implemented Feature Inventory

Release identity: `21.0.0-7`
Document scope: behavior implemented in this repository; this is not a performance guarantee or a substitute for deployment qualification.

## NFI — Native Field Intelligence (R7)

NFI provides a dedicated **Field Mapping** workspace (`field-mapping`) and the shared mapping controls embedded in Quarantine Resolution Wizard Steps 3–4. An operator can inspect source token and character positions, rename fields/labels, select ranges, split compound tokens, extract quoted or escaped key-value data, read structured paths, combine fields, and apply ordered literal or bounded-regex operations before typed validation. The same declarative mapping plan is validated by the controller and used by the client ParserPack runtime; approval, exact bundle hashes, acknowledgement, replay and rollback remain governed.

The mapping guide documents supported methods, presence/null policies, bounds, and known limitations. It does not claim universal interpretation of unknown logs or change existing input framing and destination contracts. See [R7 field mapping guide](docs/V21_R7_FIELD_MAPPING_GUIDE.md) and [audit/design](docs/V21_R7_MAPPING_DESIGN.md).

## Product boundary and compatibility

Nexora Log Intelligence (NLI) is a distributed log parsing and parser-governance platform. A client agent reads configured local sources, converts accepted records to normalized JSON, quarantines records that cannot be parsed safely, and reports bounded operational telemetry. A controller owns identity, configuration, parser lifecycle, fleet rollouts, assurance evidence, audit records, and the authenticated WebApp.

Ver21 deliberately preserves the following compatibility identities:

- Go module: `nexora.local/nli-v20`.
- RPM package names: `nli-client` and `nli-controller-agent`.
- Client services: `nli-agent`, `nli-client-control`, and `nli-client-bootstrap`.
- Controller services: `nli-controller-agent` and `nli-controller-bootstrap`.
- Configuration roots: `/etc/nli`.
- Durable state roots: `/var/lib/nli`, `/var/lib/nli-control-client`, and `/var/lib/nli-controller`.
- Installed WebView root: `/usr/share/nli/webview_ver21`.
- Release artifacts use Ver21 names: `nli_client_ver21.rpm`, `nli_controller_agent_ver21.rpm`, `webview_ver21.zip`, `webapp_ver21.zip`, and `nli-v21.0.0-7-x86_64.tar.gz`.

These stable names allow RPM to perform an in-place major-version upgrade without creating duplicate services or abandoning existing configuration and state. They do not mean the product is still Ver20.

## Runtime data path

The normal record path is intentionally short:

1. A configured source reader discovers appended bytes and frames complete records.
2. The parser runtime evaluates the ordered, active parser set.
3. Accepted records are normalized to structured JSON.
4. Required-field and output-policy checks run.
5. The record is written to configured output routes.
6. Records with no safe match or a failed contract enter the bounded quarantine path with reason metadata.
7. Aggregate counters, rates, queue state, latency, and source progress are published independently of customer log bodies.

Governance subsystems do not run expensive synthesis or assurance work on every successfully parsed record. Parser discovery, candidate generation, assurance, approval, rollout, acknowledgement, replay, and verification are explicit recovery/control-plane workflows.

## Client runtime

The `nli-client` RPM contains:

- `nli-agent`: source reading, framing, parser execution, normalization, quarantine, output, progress logging, and telemetry.
- `nli-client-control`: restricted enrollment, heartbeat, desired-state polling, signed asset application, acknowledgement, and controlled fleet actions.
- `nli`: operator CLI and version reporting.
- `nli-bench`: local benchmark utility.
- Reviewed parser packs under `/etc/nli/parsers`.
- Systemd units, configuration examples, metric catalog, Prometheus scrape example, and Grafana dashboard seed.

Source handling includes configured file paths, file identity and offset tracking, rotation/truncation detection, bounded queues, backpressure reporting, and restart-safe state. Normalized and quarantined counts are separate; a record is not deliberately emitted to both success and quarantine routes.

## Parser model

Implemented parser inputs include:

- Ordered parser packs with revision, state, priority, declarative runtime rules, templates, projections, and regression cases.
- Static positional layouts.
- Key/value delimiter extraction.
- Safe regular expressions and named captures.
- Known-key catalogs and field mappings.
- Mixed static/key-value profiles used by the reviewed migration corpus.
- Runtime helpers for network identity, URL, email address, and encoded-subject extraction.

Parser definitions are treated as governed configuration, not executable user code. Validation rejects unsupported or unsafe definitions instead of silently activating them. Catalog state and runtime compatibility are displayed separately so an approved record cannot be mistaken for a deployable parser when its runtime contract is incomplete.

## Parser lifecycle

The parser lifecycle is revisioned and auditable:

1. Create or repair a draft.
2. Attach retained evidence or explicit fixtures.
3. Define desired JSON fields and evidence-bearing mappings.
4. Generate a deterministic candidate.
5. Run validation and regression cases.
6. Submit the exact candidate for review.
7. Publish an approved parser into the canonical bundle.
8. Propose a rollout for the origin client, selected clients, or eligible online clients.
9. Approve the exact revision and bundle hash according to policy. An administrator may approve an administrator-created parser or rollout only with an explicit audited reason; non-administrator separation of duties remains enforced.
10. Deliver the immutable bundle and record each client's applied revision/hash acknowledgement.
11. Replay only eligible quarantined records after exact-hash acknowledgement.
12. Verify the replay outcome; retain audit evidence and permit rollback/offboarding through the same governed path.

Draft creation never changes the active parser set. A bundle hash binds approval to exact canonical bytes. Stale proposals are isolated rather than substituted with a newer bundle. Active parser packs expose edit, repair, download, delete/offboard, and review actions according to role and state.

## Quarantine Resolution Wizard and Guided Parser Repair Studio

The Quarantine Resolution Wizard (QRW) consolidates recovery into nine state-derived stages:

1. Select a retained failure.
2. Inspect exact evidence and the failure reason.
3. Define the desired JSON schema.
4. Map raw evidence to fields.
5. Generate the parser candidate and validation fixtures.
6. Run the deterministic preview without leaving QRW.
7. Review, approve, and publish using QRW's controller-owned lifecycle controls.
8. Deploy the exact bundle and request targeted replay after acknowledgement.
9. Verify the replay result.

Operator-defined field names, indexes, delimiters, safe regex, templates, type/presence policies, transforms, defaults, and validation constraints are available in the mapping flow. The wizard derives Completed, Current, Locked, Waiting, and Blocked states from controller records rather than from clicks alone. Ignore/exclude, return-to-queue, retained-evidence deletion, raw download, diagnosis, repair, replay, rollback, and offboarding remain explicit audited actions.

The Guided Parser Repair Studio exposes the same controller-owned evidence and parser lifecycle from Assurance Lab. The QRW duplicates relevant controls for operator convenience; it does not create a second approval or rollout system.

Ver21 R2 posts the client's exact applied revision and bundle hash immediately after persistence, while retaining the normal heartbeat as a recovery path. A replay attempted before that acknowledgement returns an inline, retryable QRW gate and does not mark canonical live synchronization stale. Quarantine deletion, exclusion, or resolution changes actionable queue state without erasing retained historical parsing measurements from Command Center charts.

## Assurance Lab (NAL)

Nexora Assurance Lab (NAL) provides:

- Exact raw-evidence analysis where retention and permissions allow it.
- Corpus reports and field-accounting evidence.
- Parser drafts, review queue, history, and fixtures.
- Guided Parser Repair Studio.
- Parser creation assistance and NAPI synthesis views.
- Test plans and persisted run outcomes.
- Deterministic preview before any publish/rollout action.

Assurance activity is durable control-plane evidence. A successful preview does not activate a parser; publication and rollout remain separate actions.

## Named subsystems

- **NHLE — Nexora Hypersonic Log Engine:** the normal client-side read, frame, parse, normalize, queue, and output runtime.
- **NURSF — Nexora Unified Real-Time Synchronization Fabric:** canonical snapshot and incremental control/telemetry distribution between clients, controller, and WebApp.
- **NAPI — Nexora Adaptive Parser Intelligence:** bounded parser discovery and synthesis metadata; it creates proposals, not autonomous production changes.
- **NAL — Nexora Assurance Lab:** deterministic validation, fixtures, regression evidence, review, and persisted run outcomes.
- **UPES — Unstoppable Parsing Engine Support:** failure observatory and governed recovery views built from canonical controller state.
- **GPRS — Guided Parser Repair Studio:** evidence-to-candidate operator workflow inside Assurance/QRW.
- **NUPIRE — Nexora Universal Parser Intelligence and Runtime Evolution:** evidence-gated parser candidate records and evaluation provenance.
- **EGPE — Evidence-Gated Parser Evolution:** held-out evaluation, approval, and provenance boundary around parser evolution.

AI is advisory and is evaluated after deterministic options. It cannot publish, approve, roll out, or replay a parser by itself.

## Controller and fleet management

The `nli-controller-agent` RPM provides:

- Authenticated `/api/v1` control-plane API and controller-served WebView.
- Client enrollment, one-use/bootstrap credentials, node registry, heartbeat, and health.
- Desired configuration, parser bundle, artifact, rollout, and acknowledgement state.
- Revisioned parser catalog and priority controls.
- Quarantine metadata, retained-evidence authorization, replay requests, and outcomes.
- Assurance, NAPI, NAL, UPES, NUPIRE/EGPE, migration, notification, audit, and communication status records.
- Canonical live telemetry snapshot, event stream, historical telemetry, OpenMetrics, and dashboard data.
- Optional Supabase metadata/private-storage adapter; normal customer event bodies are not stored there.

Fleet controls include origin/selected/eligible scopes, approval-gated changes, status, retry/cancel where implemented, desired versus applied revision/hash visibility, and rollback/offboarding proposals.

## Canonical telemetry and WebApp synchronization

All live workspaces consume the controller's canonical state rather than independently recalculating fleet totals. The controller supplies an initial snapshot plus incremental server-sent events (SSE); bounded recovery requests restore state after an interruption. The browser applies updates in place rather than requiring whole-page reloads.

Freshness distinguishes transport recency from source activity. A quiet source can be healthy with a zero rate. Last-known values are labeled when the authoritative transport or measurement is stale. Diagnostics carry bounded component, stage, reason, and recommended-action metadata; sensitive log bodies are excluded from shared telemetry.

Historical telemetry supports controller-time ranges used by Command Center. The metric catalog, Prometheus scrape example, and Grafana seed provide an external-observability starting point. Metric availability remains evidence-based: a missing measurement is not fabricated.

## WebApp workspace inventory

The authenticated WebApp groups work by operator intent:

- **Command Center:** live totals, historical series, source throughput, output volume, propagation latency, and reporting-node health.
- **Monitoring:** fleet/nodes, pipeline, sources, parser performance, destinations, queues/replay/quarantine, infrastructure, latency/synchronization, errors/incidents, and assurance/release health.
- **Live Streams:** bounded live operational event views.
- **Assurance & Release:** Assurance Lab, Quarantine Resolution Wizard, UPES Parser Recovery, Approvals & Rollouts, Parser Fleet Sync, Signed Artifacts & Upgrades, Parser Creation Workspace, NUPIRE/EGPE, Field Integrity & Corpus Proof, and AI Resolution & Providers.
- **Parser Management:** catalog/priority, regex templates/studio, regex library, known keys, and template drift.
- **Input/Output:** input paths, migration, endpoints, live metadata, unknown/quarantine, and performance/backpressure.
- **Client Management:** fleet operations and client-agent management.
- **Controller Management:** controller agent, API monitor, system health/storage, communication status, and users/RBAC/audit.
- **Real-time subsystems:** Unified Sync Fabric, NHLE Live, NAPI Live, and NAL Live with activity, freshness, and diagnostics views.
- **Operations:** Notification Manager and actionable tab badges.

Visibility and write actions are RBAC-controlled. Empty states, zero activity, blocked states, and unavailable measurements have different meanings and are not interchangeable.

## Identity, RBAC, audit, and notifications

The controller owns login sessions and role evaluation. Implemented roles include administrator, operations, parser author, reviewer, and viewer permission sets. User creation/deactivation, login time, review actions, emergency actions, parser changes, rollout changes, downloads, and recovery actions are auditable controller operations.

Notifications support severity/state, actionable tab counts, read/unread state, check-later, and dismissal/deletion of the user's notification presentation while retaining authoritative audit evidence. Expected unauthenticated bootstrap responses are part of login/session establishment and should not be presented as production system failures.

## Outputs and endpoint controls

The runtime supports local normalized JSON and controller-managed output configuration. Endpoint workspaces expose configured routes and their state; adapters represented by the UI include Loki, ClickHouse, SIEM/HTTP-style destinations, Kafka, file/SFTP, cloud/data-lake/database, and custom API/webhook patterns where the corresponding adapter/configuration is installed. Connection test, enable/disable, credentials, delivery acknowledgement, retry, and backlog visibility depend on the selected adapter's implemented contract.

No document or dashboard should claim a remote delivery acknowledgement from local parse success alone.

## AI provider boundary

AI provider configuration is server-side. Provider secrets must remain in root-owned controller configuration or an approved secret store and must never be shipped to clients, embedded in browser JavaScript, committed to Git, or included in telemetry. The provider layer is designed for multiple OpenAI-compatible and vendor-native endpoints, connection testing, model selection, bounded timeouts, and redacted diagnostics. AI output remains a proposal requiring deterministic validation and normal authorization.

## Packaging and release engineering

The release pipeline builds static Linux binaries, genuine x86_64 RPMs, a controller WebView ZIP, a web-deployment ZIP, installers, checksums, an SBOM, provenance, and a versioned archive. Two-pass release builds compare deterministic artifacts. Artifact verification checks RPM identity/version/release/architecture, payload ownership/modes, WebView bytes, archive containment, checksums, and operator-script syntax.

Authoritative Ver21 identity is:

- Product: Nexora Log Intelligence Ver21.
- Runtime version: `21.0.0`.
- Package release: `2`.
- Release archive root: `nli-v21.0.0-2-x86_64`.
- Git repositories: `nexora_log_intelligence_ver21` and its dedicated web-deployment repository.

Historical Ver20 documents remain as evidence only. Current public artifacts, helper scripts, `RELEASE_ID`, RPM metadata, binary version output, and release notes use the Ver21 R2 identity.

## Operational safety boundaries

- Never fabricate counts, rates, acknowledgements, resource measurements, or destination delivery state.
- Never replay a quarantined record before the target client reports the exact desired parser bundle hash.
- Never treat a draft or preview as an active parser.
- Never expose session cookies, node tokens, controller keys, TLS private keys, Supabase secret keys, output credentials, retained raw logs, or AI API keys in telemetry or browser bundles.
- Retain the previous verified release artifacts until upgrade validation and rollback rehearsal complete.
- Performance numbers are host/workload evidence, not universal guarantees. Validate throughput, latency, memory, queue behavior, output durability, and replay safety on the target deployment.

