# Nexora Log Intelligence Ver21 Release 9

Release `21.0.0-9` uses released R8 `v21.0.0-8`, source `82ce826aaf32a34b7aa46127ffaa1f7064e77886`, for source, upgrade and rollback qualification. R7 and R8 tags, historical documents and published assets remain preserved.

## Complete mapping and key/value tails

FieldSense mapping success now requires source evidence for observed structured suffix values. A nine-field mail header alone cannot mark a record containing an unhandled key/value payload as normalized. Explicitly mapped hidden/omitted fields and retained whole-message fields remain operator choices. Existing reviewed native parser projections retain their established contracts.

The shared native `kv_all` mapping supports a configured 1-based token/word/character/byte/line start and optional inclusive end. The UI's **Key value EOL — End of line** option stops at the first CR/LF. Expert multiline extraction remains explicit. Root flattening emits actual keys, including future keys absent a fixed preview schema, while rejecting reserved names, unsafe names, collisions and excessive output fields. Named typed key/value extraction remains available beside the dynamic tail, using explicit include/exclude rules to prevent collisions.

Suggestions use the same bounded scanner as extraction, preserving multiword values and discovering semicolon, colon and pipe-separated pairs. Consistent colon assignments can be proposed for review; ambiguous mixed assignment grammars require explicit operator configuration. AI proposals remain governed and separate from deterministic execution.

## Dashboard and parser lifecycle

Dashboard totals use the selected retained historical range and client scope. History declares minute-bucket precision and returns effective bucket-aligned bounds alongside requested bounds. Legacy bounded parser-metadata history is separately labelled and used only where no canonical checkpoint exists; it is never added to canonical counters. Live rates remain current measurements. Per-node history attribution starts with R9 observations; old fleet-only buckets cannot be assigned retrospectively to a node. Fresh client configurations publish changed telemetry at 250 ms, with 1-second idle keepalives; existing explicit second-based poll settings remain honored. Parser deletion/restoration updates the controller's canonical active generation; clients receive the corresponding exact bundle and keep a persisted inspectable runtime inventory. Replay is authorized against the published candidate and acknowledged origin bundle. Administrator-role approval uses the authenticated role rather than a literal username.

See `V21_R9_FIELD_MAPPING_GUIDE.md` for controls, evidence and limitations, and `V21_R9_TEST_RESULTS.md` for observed qualification. The release retains package names, service units, configuration and durable metadata paths. Manual deployment remains the operator's responsibility; release qualification does not claim a live customer deployment or production capacity result.
