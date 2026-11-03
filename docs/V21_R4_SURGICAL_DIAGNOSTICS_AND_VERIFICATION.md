# Nexora Log Intelligence Ver21 R4 Surgical Diagnostics and Verification Runbook

## 1. Purpose and scope

This document is the durable engineering reference for diagnosing, repairing, validating, releasing, and rolling back Nexora Log Intelligence (NLI) Ver21 R4. It covers the client RPM, controller RPM, web application, parser lifecycle, Quarantine Resolution Wizard (QRW), live synchronization fabric, navigation pages, inner tabs, and release evidence.

It is intentionally evidence-driven. A page badge, parser state, or live indicator is not accepted as proof by itself. A parser is live only when the controller has published one canonical bundle, the intended client has acknowledged that exact bundle hash, the client runtime has loaded the pack, and a matching test record produces the expected normalized output exactly once.

This runbook is also the starting point for future surgical fixes. Preserve the last known-good release, reproduce the smallest failing contract, patch only the owning boundary, and run the consolidated release gate after focused tests pass.

## 2. Non-negotiable system invariants

1. **One canonical source of truth:** pages consume the same revisioned controller snapshot. A page must not independently invent fleet, parser, quarantine, rollout, or assurance state.
2. **One record, one terminal parsing outcome:** every input record ends as normalized, quarantined, or errored. Counts must not be duplicated by polling, replay, restart, or output delivery.
3. **Exact bundle identity:** bundle bytes, canonical manifest, revision, and SHA-256 hash remain identical from publication through client acknowledgement.
4. **Published is not the same as applied:** `published` describes the controller catalog; `active` requires an exact client acknowledgement and runtime load evidence.
5. **Replay is gated by correctness, not UI state:** replay is allowed only for an actionable retained record whose origin client has acknowledged the desired exact bundle.
6. **Parser changes do not interrupt telemetry:** parser publication, approval, rollout, acknowledgement, and replay use bounded background operations and must not block the synchronization snapshot or event stream.
7. **Raw evidence is explicit:** raw quarantine evidence is available only when client upload and controller retention are enabled. Metadata-only records remain visible but cannot be inspected as raw text.
8. **Resolved work does not remain actionable:** deleted, excluded, ignored, resolved, or acknowledged items may remain in audit history but must leave actionable navigation and notification counts.
9. **No fabricated telemetry:** unavailable measurements are identified by source/status metadata. Derived controller counts must be labelled as derived and never presented as client-reported runtime metrics.
10. **Rollback is always possible:** the client retains the last accepted parser bundle and restores it when validation, load, or acknowledgement fails.

## 3. Authoritative data flow

```text
raw log source
  -> nli-agent reader/framer
  -> active parser bundle in /var/lib/nli/parsers
  -> normalized | quarantined | errored terminal outcome
  -> nli-client-control heartbeat, metrics, quarantine metadata/evidence, ACK
  -> nli-controller-agent durable state and canonical revision
  -> NURSF snapshot + ordered incremental stream
  -> web application shared store
  -> every page, badge, graph, notification, and subsystem view
```

Primary service and configuration boundaries:

| Boundary | Service/configuration | Durable state |
|---|---|---|
| Client parser runtime | `nli-agent.service`, `/etc/nli/nli.json` | `/var/lib/nli`, `/var/lib/nli/parsers` |
| Client control plane | `nli-client-control.service`, `/etc/nli/client-control.json` | client bundle/ACK and retry state under `/var/lib/nli` |
| Controller | `nli-controller-agent.service`, `/etc/nli/controller.json` | `/var/lib/nli-controller` |
| Web application | controller `/api/v1/*` contracts | shared in-browser canonical store; no independent fleet polling |

Core live endpoints:

- `/api/v1/health`
- `/api/v1/sync/snapshot`
- `/api/v1/sync/diagnostics`
- `/api/v1/sync/stream`
- `/api/v1/parser-packs`
- `/api/v1/quarantine/records`
- `/api/v1/quarantine/reprocess`
- `/api/v1/node/quarantine/records`

## 4. Parser and QRW lifecycle contracts

### 4.1 Parser lifecycle

```text
draft -> deterministic validation -> reviewed candidate -> approved/published
      -> canonical bundle revision -> targeted/all-eligible rollout
      -> exact client hash acknowledgement -> runtime load evidence
      -> matching record parse proof -> active
```

Required evidence for each state:

| State | Required evidence |
|---|---|
| Draft | durable definition, author, revision, validation fixtures |
| Validated | at least one reviewed regression case passes; no unsafe/executable rule fields |
| Published | parser is included in the controller's canonical manifest and exact bundle hash |
| Rolled out | rollout targets are persisted with the same desired hash |
| Acknowledged | target reports the exact applied hash, not a late hash from an older rollout |
| Runtime active | client runtime successfully loads the same pack and exposes load/match evidence |
| Proven | a representative record matches and normalizes to the expected schema once |

Late acknowledgements from older bundles must be retained as audit evidence but cannot replace the desired/current hash. Re-proposing an already active exact bundle is idempotent and must not create duplicate rollouts.

### 4.2 QRW nine-stage lifecycle

| Step | Name | Completion contract |
|---:|---|---|
| 1 | Select | an actionable quarantine record is selected; metadata-only evidence is clearly identified |
| 2 | Inspect | exact retained raw evidence is loaded, or a safe evidence-reacquisition action is offered |
| 3 | Schema | desired JSON fields and types are explicit |
| 4 | Map | every required field has an evidence-bearing extraction, approved default, or omission policy |
| 5 | Generate | one deterministic candidate and validation fixtures are saved |
| 6 | Test | the exact candidate passes deterministic preview/regression validation |
| 7 | Review & publish | authorized review records the reason and publishes the candidate into the canonical bundle |
| 8 | Deploy & replay | target acknowledges the exact desired hash; only then can targeted replay be requested |
| 9 | Verify result | replay reaches a terminal resolved/normalized result and history remains auditable |

The QRW is a unified operator surface over existing Assurance, Parser Catalog, Fleet Sync, rollout, and quarantine APIs. It does not maintain a second parser catalog or bypass the authoritative controller lifecycle.

## 5. Complete page and inner-tab inventory

Every route below must have a registered loader, use the shared API/store contract, render a stable empty/error state, and expose mutations only through an authenticated controller endpoint.

| Group | Route | Page | Primary authoritative concern |
|---|---|---|---|
| Command Center | `operations` | Command Center | canonical counts, history, throughput, latency |
| Command Center | `monitoring` | Monitoring | fleet/pipeline/destination/infrastructure observability |
| Command Center | `streams` | Live Streams | ordered incremental records and stream state |
| Assurance & Release | `assurance` | Assurance Lab | validation, fixtures, review, runs, evidence |
| Assurance & Release | `quarantine-wizard` | Quarantine Resolution Wizard | governed nine-stage recovery |
| Assurance & Release | `upes` | UPES · Parser Recovery | failure observatory and parser recovery health |
| Assurance & Release | `rollouts` | Approvals & Rollouts | review reasons, approvals, rollout results |
| Assurance & Release | `parser-sync` | Parser Fleet Sync | desired/applied hashes and fleet convergence |
| Assurance & Release | `artifacts` | Signed Artifacts & Upgrades | signed RPM/bundle artifacts and upgrade status |
| Assurance & Release | `parser-workspace` | Parser Creation Workspace | deterministic parser authoring and local validation |
| Assurance & Release | `nupire` | NUPIRE / EGPE | evidence-gated parser evolution |
| Assurance & Release | `field-integrity` | Field Integrity & Corpus Proof | schema/corpus integrity evidence |
| Assurance & Release | `ai` | AI Resolution & Providers | optional last-resort AI proposals and providers |
| Parser Management | `parsers` | Parser Catalog & Priority | canonical parser catalog, priority, runtime state |
| Parser Management | `studio` | Regex, Templates & Studio | bounded regex/template authoring |
| Parser Management | `regex` | Regex Pattern Library | reviewed expression inventory |
| Parser Management | `knownkeys` | Known Key Catalog | normalized field/key knowledge |
| Parser Management | `drift` | Template Drift | parser/template drift detection |
| Operations | `notifications` | Notification Manager | actionable event delivery and read/defer/delete state |
| Real-time Subsystems | `sync-fabric` | Unified Sync Fabric | NURSF transport, revision, freshness, diagnostics |
| Real-time Subsystems | `nhle-monitor` | NHLE Live | Hypersonic Log Engine ingestion/runtime state |
| Real-time Subsystems | `napi-monitor` | NAPI Live | Adaptive Parser Intelligence activity |
| Real-time Subsystems | `nal-monitor` | NAL Live | Assurance Lab execution state |
| Input | `input-paths` | Input Log Paths | source path configuration and rollout |
| Input | `migration` | Migration Center | controlled migration/import state |
| Output | `outputs` | Endpoint Configuration | destination profiles, credentials, tests, rollback |
| Output | `metadata` | Live Metadata | canonical normalized metadata |
| Output | `unknowns` | Unknown & Quarantine | retained queue, diagnosis, prepare replay |
| Output | `performance` | Performance & Backpressure | rates, queues, backpressure, latency |
| Parsing Engine | `kv` | KV Delimiter Config | key/value rules |
| Parsing Engine | `static` | Static Positional Patterns | positional parsers |
| Parsing Engine | `projection` | Field Capture & Projection | field capture/output projection |
| Parsing Engine | `semantic` | Semantic Extraction Policy | semantic field policy |
| Runtime Engines | `engine-network` | Network Identity Engine | IP/MAC/network extraction |
| Runtime Engines | `engine-url` | URL Engine | URL extraction/normalization |
| Runtime Engines | `engine-email` | Email Address Engine | email extraction/normalization |
| Runtime Engines | `engine-subject` | Encoded Subject Engine | encoded subject decoding |
| Client Management | `fleet` | Fleet Operations | node health and bulk operations |
| Client Management | `clients` | Client Agent Management | enrolment, configuration, heartbeat |
| Controller Management | `controller` | Controller Agent Management | controller configuration/status |
| Controller Management | `api-monitor` | API Monitor | endpoint health, latency, errors |
| Controller Management | `system` | System Health & Storage | resources, storage, retention |
| Controller Management | `communications` | Status of All Communication | client-controller-webapp path health |
| Controller Management | `security` | Users, RBAC & Audit | accounts, roles, login/audit history |

Pages with actionable navigation counts are: Assurance Lab, QRW, UPES, Approvals & Rollouts, Parser Creation Workspace, Parser Catalog & Priority, Input Log Paths, Endpoint Configuration, KV Delimiter Config, Static Positional Patterns, Field Capture & Projection, Regex/Templates/Studio, Regex Pattern Library, Known Key Catalog, Semantic Extraction Policy, and Unknown & Quarantine. Counts exclude completed, deleted, ignored, excluded, or otherwise non-actionable audit records.

### 5.1 Inner tabs

- **Monitoring (11):** Overview; Fleet & Nodes; Pipeline; Sources; Parser Performance; Destinations; Queues / Replay / Quarantine; Infrastructure; Latency & Synchronization; Errors & Incidents; Assurance & Release Health.
- **Assurance Lab (11):** Analyse; Corpus reports; Parser drafts; Review Queue; History; Fixtures; Guided Parser Repair Studio; Parser Creation Helper; NAPI Synthesis; Test Plans; Runs.
- **UPES (8):** Live Failure Observatory; Recovery Decision Trace; Parser Repair Workshop; Knowledge Base; Discovery Jobs; Parser Health & Drift; Deployment & Rollback; Runtime Health.
- **Endpoint Configuration (5):** Destinations; Security & credentials; Validation; Saved profiles; Live & rollback.
- **AI Resolution & Providers (7):** Provider; Status; Results; Parser; Regex; Template; Approval.
- **QRW (9):** Select; Inspect; Schema; Map; Generate; Test; Review & publish; Deploy & replay; Verify result.
- **Each real-time subsystem (4):** Overview; Live activity; Freshness; Diagnostics.

## 6. R4 verified defect analysis and owning boundaries

| Symptom | Root cause/owning boundary | Surgical correction contract |
|---|---|---|
| QRW Step 1 shows `Not retained` and cannot inspect | raw upload and/or controller retention disabled; UI treated metadata-only records as a dead end | expose evidence status and bounded reacquisition guidance/action; never pretend metadata contains raw text |
| Catalog says Published/Active but logs do not match | publication/ACK state was treated as runtime match proof; adapter service selection could reject dynamic/wildcard packs before evaluating them | load the canonical applied bundle in runtime, evaluate eligible wildcard/dynamic packs, and publish load/match evidence |
| UPES displays Not reported/0 beside waiting rows | cards used optional heartbeat counters while rows used the controller replay queue | reconcile controller-derived actionable/waiting counters and label runtime-only metrics separately |
| Assurance and Parser Catalog disagree | Assurance returned raw parser documents while catalog/fleet used projected runtime compatibility state | use the same revisioned parser-document projection in all parser views |
| QRW replay waits despite completed rollouts | UI selected stale/older rollout or late ACK rather than desired exact hash | select current desired bundle, ignore late old ACKs for gating, and unlock only exact-hash records |
| Live synchronization reports 530 during mutations | mutation failure/retry leaked into global stream health or a long mutation blocked shared controller work | keep mutations bounded and independent; stream recovery uses snapshot fallback without clearing last-known state |

## 7. Failure signature matrix

| Signature | First checks | Do not do |
|---|---|---|
| `Not retained` | client `quarantine_raw_upload_enabled`; controller `quarantine_raw_retention_enabled`; per-record evidence status; retention limits/key | do not label metadata as raw evidence |
| Published/Active but no matching output | desired/applied hash, parser files in `/var/lib/nli/parsers`, runtime load journal, service/family eligibility, match counter, normalized output | do not infer runtime activity from catalog state alone |
| UPES waiting rows with zero cards | canonical revision, replay queue state, `controller_records_status`, runtime metric status | do not manufacture client counters |
| `bundle hash is invalid` | canonical manifest bytes, stored proposal hash, current controller hash, request revision, JSON canonicalization | do not recompute from display JSON or an unordered map |
| `target client has not acknowledged` | rollout target, desired hash, heartbeat applied hash, late ACK audit, client download/load journal | do not bypass ACK gating for ordinary replay |
| `stream reconnecting: 530` | `/health`, sync diagnostics, stream response headers, controller latency, storage readiness, reverse proxy timeout | do not refresh the entire page or erase last-known canonical state |
| Counts fall after deleting quarantine evidence | distinguish cumulative telemetry/history from actionable queue; verify history aggregation | do not subtract evidence deletion from historical input/parsed totals |

## 8. Surgical diagnostic commands

Run on the appropriate Rocky Linux client/controller. Replace placeholders; never paste real secrets into tickets or documentation.

```bash
sudo systemctl status nli-agent nli-client-control nli-controller-agent --no-pager
sudo journalctl -u nli-agent -u nli-client-control --since "15 minutes ago" --no-pager
sudo journalctl -u nli-controller-agent --since "15 minutes ago" --no-pager

curl -fsS http://127.0.0.1:<controller-port>/api/v1/health
curl -fsS http://127.0.0.1:<controller-port>/api/v1/sync/diagnostics
curl -fsS http://127.0.0.1:<controller-port>/api/v1/sync/snapshot
curl -fsS http://127.0.0.1:<controller-port>/api/v1/parser-packs

sudo rpm -q nli-client nli-controller-agent
sudo rpm -ql nli-client
sudo rpm -ql nli-controller-agent
sudo sha256sum /path/to/downloaded/artifact.rpm

sudo grep -E 'quarantine_raw_upload_enabled|quarantine_raw_retention_enabled' \
  /etc/nli/client-control.json /etc/nli/controller.json
sudo find /var/lib/nli/parsers -maxdepth 2 -type f -printf '%p %s bytes\n'
```

For a bundle incident, record these as one evidence set: proposal ID, parser revision, desired hash, rollout ID, target node ID, client applied hash, runtime-loaded hash, first/last ACK timestamps, replay record ID, and terminal replay outcome.

## 9. Surgical repair workflow

1. Freeze the last known-good tag, commit, RPM checksums, and release folder.
2. Reproduce the smallest failing contract with one deterministic fixture.
3. Identify the owning boundary: runtime adapter, client-control delivery, controller state, API projection, or UI rendering.
4. Add a focused regression test that fails before the patch.
5. Patch only the owning boundary and preserve compatibility identifiers unless a migration is explicitly designed.
6. Run the focused test and adjacent package tests.
7. Inspect the diff for unrelated changes and generated artifacts.
8. Run the consolidated full release gate once after all focused fixes are merged.
9. Build reproducible artifacts, validate installation/service lifecycle, and compare checksums.
10. Publish only after source and Vercel repositories are clean and both main/tag workflows are green.

## 10. Verification layers and R4 release gate

| Layer | Required evidence | R4 result |
|---|---|---|
| Static/web contract | all 44 routes, loaders, actionable badge pages, inner-tab inventories, API wiring | SOURCE GATES PASS — revision `be028bb`; `npm run web:check` passed `28/28`; full release infrastructure static gate passed |
| UI behavior | exhaustive page/sub-tab navigation, stable refresh, QRW recovery, actions, empty/error states | PASS — `73/73` Playwright tests passed, including QRW metadata-only evidence recovery, stages 6–9, replay gating, parser Published / Active state, notification-count clearing, SSE reconnect/fallback, and every visible inner tab |
| Go unit/integration | all packages plus focused parser, sync, quarantine, rollout, late-ACK tests | PASS — pinned Go `1.24.9`; test, vet, and race gates passed across 63 packages |
| Runtime parser proof | delivered pack loads and matches a representative previously unsupported record | PASS — the packaged Rocky artifact matrix verified exact parser accounting without duplicate normalized records, and the isolated controller/client scenario verified enrolled live parser/control services |
| Client/controller lifecycle | clean install, upgrade, restart, rollback, uninstall checks on supported Rocky versions | PASS — Rocky Linux 9.6 and 10.2 passed clean install, R3→R4 upgrade, rollback/re-upgrade, service handling, TLS identity, enrollment, and health checks; the final Rocky 10.2 two-node WebView scenario passed `6/6` browser workflows plus trusted-certificate discovery and offline-TLS fallback |
| Release reproducibility | RPM/archive checksums and release evidence verification | PASS — both RPMs, the versioned archive, `SHA256SUMS`, installer exact-source checks, packaged-document byte identity, and reproducibility verification passed from revision `be028bb` |
| GitHub/Vercel | clean source, tag/release, green source and deployment workflows | PENDING FINAL GATE |

The local artifact gate verified both RPMs, the versioned archive, `SHA256SUMS`, installer exact-source checks, packaged-document byte identity, and reproducibility from revision `be028bb`. The exhaustive UI gate passed `73/73`; Rocky 9.6/10.2 lifecycle and the isolated two-node acceptance also passed. These results do not substitute for external publication or production capacity qualification.

The release gate includes, at minimum:

```text
go test ./...
npm run web:check
npm run web:test:ui
python tests/release/release_docs_packaging_test.py <release-folder>
bash tests/release/static.sh
make release-build
```

Exact command output, counts, commit IDs, tag, workflow URLs, artifact checksums, and any qualified limitations belong in `V21_R4_TEST_RESULTS.md`.

## 11. Rollback and recovery

- Do not use destructive repository resets on an unreviewed worktree.
- If a parser rollout fails validation or load, keep the previous parser directory and applied hash authoritative, restore it atomically, restart only the parser runtime if needed, and report the failed desired hash separately.
- If the controller stream is interrupted, retain the last-known canonical snapshot, reconnect with bounded backoff, fetch a fresh snapshot, then resume ordered deltas from the new revision.
- If a release artifact fails lifecycle validation, do not promote the tag. Retain the previous RPMs and checksums and document the failure in test results.
- Audit records remain durable; only actionable badge counts and queues are cleared when work is resolved or intentionally dismissed.

## 12. Capacity and accuracy qualification

Passing functional tests proves the tested contracts only. A 1,000-node correctness simulation does not prove production throughput, end-to-end latency, or a claimed ten-million-events-per-second rate. Those claims require a representative capacity environment, defined payload distribution, sustained load duration, resource limits, failure injection, and recorded percentiles. R4 release notes must state the measured scope and must not convert design targets into achieved results.

## 13. Future incident record template

For every surgical issue, append or link a record containing:

- release/tag/commit and artifact checksum;
- environment, node count, OS/RPM versions, and topology;
- exact UTC and local time window;
- affected route/inner tab, API, service, and record/parser/rollout IDs;
- expected versus observed behavior;
- sanitized logs and canonical revision/hash evidence;
- root cause and owning boundary;
- regression test added;
- files changed and compatibility impact;
- focused/full verification results;
- rollback result and remaining limitations.

