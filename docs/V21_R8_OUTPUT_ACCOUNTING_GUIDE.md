# Ver21 R8 output record accounting guide

R8 is based on qualified R7 `v21.0.0-7` at `bc6b17a20e1a7b2c11903df440a14ad9134697d8`. This guide distinguishes a record-accounting investigation from file-size comparisons and describes the scope of the separately reproduced retry/quarantine corrections. It does not rewrite existing output or imply a live-host diagnosis.

## Bytes, physical lines and logical records

A size listing such as 48K or 738 measures bytes. Normalized JSON adds field names and metadata, so a larger file does not prove duplicate records. Hourly output may already contain earlier legitimate history. Input physical lines can also differ from logical records because of multiline framing, intentional filtering or blank-line handling. Identical text at different source positions remains distinct legitimate records; do not remove it using content-only deduplication.

Count only a controlled test interval. Stop input at EOF and allow the pipeline to drain before comparing. Select the canonical normalized and quarantine outputs for that interval. Do not add separate quarantine journals, replay outputs, copied files or fanout destination copies to the same input count. Record intentional filtering explicitly.

## Read-only before/after audit

Run the shipped release asset `nli-output-audit.py`, or the RPM copy `/usr/share/doc/nli-client/nli-output-audit.py` (controller copy: `/usr/share/doc/nli-controller-agent/nli-output-audit.py`). Python 3 is already an RPM dependency. The helper reads source/output files and writes only an explicitly requested count/hash snapshot. It never clears, rewrites or content-deduplicates those files.

Before the controlled input:

```bash
python3 /usr/share/doc/nli-client/nli-output-audit.py \
  --normalised '/var/log/nli/output/auto_normalised_*.jsonl' \
  --quarantine '/var/log/nli/output/auto_quarantine_*.jsonl' \
  --snapshot /root/nli-before.json
```

After EOF and drain, with the same output scope:

```bash
python3 /usr/share/doc/nli-client/nli-output-audit.py \
  --input /path/to/controlled-sample.log \
  --normalised '/var/log/nli/output/auto_normalised_*.jsonl' \
  --quarantine '/var/log/nli/output/auto_quarantine_*.jsonl' \
  --baseline /root/nli-before.json --single-line
```

Use `--single-line` only when each nonblank physical input line is exactly one logical record. Otherwise pass a known framed count with `--expected-records 49`; add `--filtered-records N` for known intentionally filtered records. Do not guess a multiline logical count from `wc -l`.

A verified synthetic helper test preserves earlier hourly history while balancing 49 new records as 47 normalized plus 2 quarantine records. A genuine extra 50th output reports mismatch. Invalid JSONL, partial in-flight rows, blank output rows or truncated/replaced historical files yield unknown/mismatch rather than a false balance. A quarantine output not yet created because nothing failed counts as zero.

## Reproduced retry and quarantine defects

A required remote error could occur after a successful local append, leaving the source cursor uncommitted. Retrying that same occurrence appended an extra local row. R8 uses source generation plus logical ordinal and a durable delivery receipt/WAL so local append is idempotent across that retry. It does not deduplicate raw text, and customer JSON fields are unchanged.

A separate legacy path wrote the raw failed record and then a routed envelope into the same canonical quarantine file. The corrected canonical quarantine path uses its dedicated raw queue once. A separate quarantine journal may still exist for recovery; it is not another canonical output record to add to the test count.

## Durable state and delivery limits

The local guarantee requires one owning writer and intact queue/receipt/checkpoint state. Preserve these together during restart, upgrade, backup and restore. Deleting state, externally rewriting output or allowing multiple independent writers is outside that guarantee. Receipts retire after durable source cursor acknowledgement. The receipt journal is `<agent state_dir>/delivery/<SHA256(output.Path + NUL + output.PathTemplate)>.wal`. It stores intent/commit/abort metadata (delivery ID, target path, offset, length and SHA-256), not raw record values. The source cursor persists generation before the first identified delivery and tracks the logical event ordinal. A delivery ID combines a 192-bit generation prefix with a 64-bit logical ordinal. Quarantine record IDs derive from delivery identity, so equal raw records at distinct source offsets/generations remain independently retained. Source identity includes node, path, device, inode and generation, so separate physical occurrences remain distinct.

Before any new append, startup/retry recovery resolves pending intents. A complete append with matching hash is recovered as delivered. An incomplete tail owned by that intent is truncated only to its recorded pre-append offset and marked aborted; conflicting completed bytes fail closed. This is runtime crash recovery of its own partial append, not the audit helper rewriting historical files. After a durable cursor checkpoint, receipts at or below the acknowledged logical ordinal are pruned. Startup repeats that pruning if a crash occurred between cursor acknowledgement and cleanup.

The agent state directory `.agent.lock` prevents concurrent cursor owners; duplicate physical input aliases/hardlinks and duplicate file-output queue owners are rejected. External writers or agents with unrelated state directories still violate exclusive ownership.

Take a consistent backup of agent state and its owned output files together while writers are quiescent; restore them together. Do not edit/delete a WAL or cursor independently or mix state and output snapshots from different instants. Restoring earlier state may replay remote deliveries because remote semantics remain at-least-once. File and directory synchronization support the qualified POSIX/Linux durability path; Windows directory-handle limitations do not establish equivalent crash-durability qualification. Never delete state as an accounting workaround.

Remote transports retain at-least-once acknowledgement ambiguity. A remote server may accept a request while the client loses its acknowledgement; this local correction does not promise exactly-once Loki delivery. Compare each remote/fanout destination separately and use its own acknowledgement/idempotency capabilities where available.

Final runtime regression results, frozen revision and artifact checksums belong in `V21_R8_TEST_RESULTS.md` and release qualification evidence. Existing nonempty hourly files or byte sizes alone do not establish that the user's live host duplicated records.


## Regression evidence inventory

The named runtime gates are `Test49PhysicalRecordsRequiredRemoteRetryPreservesLocalCounts`, `TestIdentifiedDeliveryRecoversInterruptedAppend`, `TestIdentifiedDeliveryRejectsConflictingCompletedAppend` and `TestDuplicatePhysicalInputAliasesAreRejected`. They target retry identity, crash-tail recovery, fail-closed conflicts and duplicate physical input aliases. Final PASS results are recorded only after execution; they do not prove delivery on an uninspected live host or remote exactly-once behavior.


## Loki acknowledgement limits and durability cost

Normal successful Loki push handling uses HTTP 204. HTTP status class alone is insufficient: when ingestion is blocked, Loki documents `block_ingestion_status_code`, whose default is 260 even though it lies in the 2xx range. NLI treats that blocked response and HTTP 202 as non-acknowledgements and keeps required-delivery retry behavior; native 204 and compatibility 200 are accepted. If the server deliberately configures blocked ingestion to return 200 without an error message, the client cannot distinguish that response from an apparent success. Preserve an unambiguous server-side blocking status; an HTTP acknowledgement is still not proof of remote exactly-once ingestion. See the official [Loki HTTP API](https://grafana.com/docs/loki/latest/reference/loki-http-api/) description of `block_ingestion_until` and `block_ingestion_status_code`.

Durable local receipts and coordinated append locks add persistence/locking work. Final scoped 1,000-row local Docker measurements observed 127.28–144.67 milliseconds for identified durable append (about 127–145 microseconds per row), versus 69.49–77.96 milliseconds for a buffered append reference using the same append-lock mechanism. That reference is **not the R7 binary**. Earlier pre-coordination figures are superseded. These measurements compare specific durability modes and one filesystem/workload; they are not a parser benchmark, production capacity figure or a universal slowdown estimate. Measure the deployed storage and fanout configuration before setting throughput expectations.


## Downgrading to R7

The R7 binary predates R8 identified delivery and does not honor or recover R8 pending receipt/WAL state. For count continuity, quiesce ingestion, restore required destination availability, let R8 drain to a durable cursor checkpoint, confirm there are no pending receipts and journals are empty, stop R8 completely, then back up state and owned outputs together before downgrade. Do not run both binaries or downgrade during an unresolved required-destination outage. Keep R8 as the recovery path for pending intents. R7 may omit additive generation/ordinal fields at later checkpoints; on a later upgrade, R8 assigns a new generation at the current durable offset. Package rollback/state preservation is distinct from R8 retry-idempotency guarantees.


## Additive parser availability and one output per source occurrence

R8 keeps embedded baseline profiles and approved managed parser packs in one bounded eligible candidate pool. Approval, service selectors, runtime compatibility and reviewed scope guards still determine eligibility. A successful embedded parser does not short-circuit the remaining approved matching pack attempts. Embedded profiles execute first to protect baseline availability under resource limits; failed managed candidates are isolated and cannot by themselves suppress a valid generic JSON/text fallback.

Selection produces one governed customer output after the eligible attempts. Reviewed native mapping scope/priority can intentionally select a custom schema; an ordinary regex match does not silently shadow structurally valid JSON. Original approved parsers remain available while new parsers are added or repaired. Multiple parser attempts are diagnostic activity, not multiple input records or permission to append a customer output for every matching parser. Record accounting compares source occurrences with canonical outputs separately from parser-attempt counters.


## Governed replay while ingestion remains active

One agent owns its source cursor, while the agent and governed replay CLI can coordinate through the same state-directory delivery WAL. Append/WAL locks are short and released during remote HTTP calls; a separate replay-operation lock rejects simultaneous replay CLIs for the same queue. Live writers reload changed journal inode/size before append/prune. Atomic replay queue merge preserves new quarantine records appended by the running agent, and the raw writer follows a replacement queue inode.

Replay delivery identity includes quarantine record identity, approved candidate revision and a persisted delivery-attempt ordinal. Failed required delivery persists queue/history attempt state before returning an error; retry suppresses the prior local append while honestly retrying remote delivery. Replay remains a deliberate additional output scope, so exclude it from a fresh-ingestion-only baseline count. WAL/lock files use mode 0600; existing queue/output mode 0640 and owner/group are preserved through coordinated replacement. Consistent backup still requires all cooperating writers to be quiescent.

Copytruncate detection covers an observable file size shrink. A file truncated and regrown past the old offset while the agent was stopped remains a standard tailer limitation; no text-based deduplication or universal lossless rotation claim is made.


Empty data-sidecar lock files use names `.nli-append-<hash>.lock` and `.nli-replay-<hash>.lock`. They hold no event payload and use mode 0600 with inherited parent owner/group. Do not delete or replace a lock file while any cooperating writer/replay process is active. Queue/WAL compaction and replacement preserve the agent owner/group; regression checks include a privileged writer creating/compacting data for UID 1001/GID 1002.
