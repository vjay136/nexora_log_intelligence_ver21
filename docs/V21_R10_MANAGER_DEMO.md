# Nexora Log Intelligence Ver21 R10 manager demo

NLI takes raw logs from enrolled Linux clients, applies reviewed parser rules, and delivers structured records to the configured destinations. Operators can see which parser revision is active on each client and use quarantined evidence to recover unsupported formats.

## Demonstrate a complete parser in ten minutes

1. In **Command Center**, select a recent time range and one enrolled client. Show the parsed, quarantined, and error totals and the source freshness indicator. Explain that counters apply to the selected scope.
2. In **FieldSense Studio** or the **Quarantine Resolution Wizard**, open an approved demo sample. This record has a positional header and a pipe-delimited key/value tail that starts at token 9.

   ```text
   20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg |status:failed|reason:notneeded|server:cloud
   ```

3. Select **Key value EOL** at token `9`. **Add dynamic KV tail** opens the existing tail if one is already mapped. Select **Inspect KV grammar** to open **KV AutoSense and Failure Inspector**. Use **Apply grammar** on the candidate that accounts for the three pairs, or use **Configured separators** with `:` for the assignment and `|` for pair boundaries. Select **Preview mapping** to compare the extracted values with the raw text. The tail fields should appear as follows, in addition to the header fields you map.

   ```json
   {
     "status": "failed",
     "reason": "notneeded",
     "server": "cloud"
   }
   ```

4. Review another representative record with the same shape, a supported mixed assignment example such as `status:failed reason;notneeded server;cloud` at the same start position, and an unterminated quoted value. To support both reported formats with one candidate, keep **AutoSense** and validate both complete records. To lock a known source to one grammar, apply a candidate or choose configured separators. Use the numbered steps and highlighted action in the wizard to continue. Show where diagnostics identify an unrecognized pair or an incompatible field. In **Assurance Lab**, compare actual output with the expected fields and values. Preserve the evidence when the rule needs correction.
5. Select **Generate + validate complete parser pack** to build one candidate containing the header and tail mappings. Save it as a draft. The wizard highlights the next available review, approval or rollout action. **Approvals & Rollouts** provides the fleet-wide view of the same workflow.
6. In **Parser Fleet Sync**, compare the desired and applied bundle hashes, check the client heartbeat, and inspect any failure details. After the originating client acknowledges the required bundle, use the **Quarantine Resolution Wizard** to replay the retained record if it is eligible.
7. In **Endpoint Configuration**, show the output profiles. Follow the resulting activity in **Live Metadata** and **Performance & Backpressure**. Finish by opening **Users, RBAC & Audit** to show who changed the parser and who approved the rollout.

## What the mapping controls do

- **Start** selects the first token, character, byte, or line in the configured range. Positions start at 1. For the sample above the tail begins at token 9. **Key value EOL** ends at the first line break.
- **Assignment operators** define how a key gets its value. Use the JSON list `[":"]` for the pipe example. **Pair separators** define candidate boundaries. Use `|` for the pipe example, or whitespace in the mixed example where `;` assigns a value.
- For a source with custom literals, use **Configured separators**. A multi-character pair list such as `["||", " "]` preserves the delimiter as one boundary. Use a key pattern to constrain recognition, or explicit aliases and include/exclude lists to control the output. For nested objects, select a named value and use the existing type and ordered extraction controls.
- Quoted, escaped and balanced bracketed values can contain punctuation. For example, `reason="retry; still queued"` keeps the text inside the quotes. Use multiple sample shapes to check that punctuation in a URL or IPv6 address remains part of its value.
- **Duplicate keys** controls which occurrence is kept, whether occurrences become an array, or whether the record fails validation. Review cases such as `id=001 id=002`. Preserve identifiers as strings when leading zeros matter.
- Choose the required, optional, type, and transformation settings for your output. Use expected JSON to catch missing fields, extra fields and type differences before review.

## Where to find each capability

| Workspace group | What to show |
| --- | --- |
| Command Center | Historical totals, current rates, fleet health, live streams, and time-range scope. |
| Assurance & Release | NAL validates expected JSON and regressions. GPRS guides quarantine repair. UPES surfaces failure evidence and recovery. NUPIRE and EGPE support evidence-based candidate evaluation. Review and verify rollouts, approved artifacts, and client acknowledgements in this group. |
| Parser Management | Author one mapping plan in FieldSense. Review parser catalog and priority, regex and templates, known keys, and template drift. |
| Real-Time Subsystems | NURSF shows freshness and synchronization. NHLE shows local processing activity. NAPI and NAL show discovery and validation activity. |
| Parsing Engine | Review KV delimiters, static positions, field projection, and semantic extraction policies. KVIE provides key/value analysis for native mapping plans. |
| Runtime Engines | Show network identity, URL, email, and encoded subject extraction using representative input. |
| Client Management | Inspect enrolled nodes, client health and service state before choosing a rollout target. |
| Controller Management | Inspect API, storage, and communication health. Review permissions and audit events. |
| Input | Define exact source paths and parser families. Use Migration Center for evidence and review of imported configurations. |
| Output | Configure destinations, inspect delivered-record metadata and quarantine, and monitor queues or delivery errors. |
| Operations | Review durable notifications and delivery attempts. |

## Failure recovery and delivery

| Scenario | Operator action |
| --- | --- |
| A record has a new or malformed shape | Open Unknown & Quarantine or UPES. Check the error, parser revision, and retained-evidence availability. Map the sample in the wizard and validate a candidate before publishing it. |
| A destination is unavailable | Check Performance & Backpressure and the endpoint's delivery status. Loki and ClickHouse have destination-specific batching and retry controls. Check local receipts and required delivery status before replaying. |
| A client is offline or drifted | Check the last heartbeat, command result, and desired/applied hashes. Wait for the exact bundle acknowledgement before retrying a quarantined record. |
| A proposed parser fails validation or rollout | Inspect the failing fixture or client error. Repair the draft or use the governed rollback workflow. Confirm the restored client hash. |
| The WebApp or telemetry link is unavailable | Check Status of All Communication and the client and controller services. The client parses using its locally applied bundle and the configured output pipeline. WebApp freshness reflects the available telemetry. |
| A deterministic suggestion needs more evidence | Add representative inputs and exact expected outputs in Assurance. Use AI-assisted proposals only through the configured provider and the same review and validation path. |

NLI supports local file, Loki, ClickHouse, and controller output profiles. Each configured destination receives records through its own delivery path. Exact-raw quarantine is retained locally. Optional raw evidence on the controller depends on configured retention and operator access. Delivery confirmation reflects destination acknowledgements. Remote retries can result in duplicate delivery when the remote outcome is unknown.

For a minimal health check, inspect **Monitoring**, **Performance & Backpressure**, and **Status of All Communication**. On the Linux hosts, review recent service failures with `journalctl -u nli-agent -u nli-client-control --since "15 minutes ago"` on the client and `journalctl -u nli-controller-agent --since "15 minutes ago"` on the controller. Correlate failures with timestamps, record IDs, and bundle hashes. Use redacted samples in shared demos.

The authenticated FieldSense status (`/api/v1/field-mapping`) and the NAL section of the canonical sync snapshot show aggregate KVIE inspection and no-match counts. The mapping editor shows the sample's diagnostics. The aggregate status does not store raw logs, and the analysis admission limit is separate from ingestion and fleet commands.

Show accuracy using the expected output for each representative sample, including malformed examples. Use measured throughput and queue metrics from the demo environment. Verify installer and upgrade results against the qualification record and its associated release workflow.

Further detail is available in the [operator and parser guide](WEBAPP_OPERATOR_AND_PARSER_GUIDE.md), [installation guide](INSTALLATION_AND_CONFIGURATION.md), and [R10 qualification record](V21_R10_TEST_RESULTS.md).
