# Ver21 R7 NFI field mapping guide

NFI — Native Field Intelligence uses a shared declarative mapping plan in the **Field Mapping** workspace (`field-mapping`), Quarantine Resolution Wizard Steps 3–4, Assurance candidate preview and the runtime parser. Editing a mapping does not publish a parser.

## Exact requested example

```text
20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud
```

The whitespace source positions are **1-based**, with an inclusive end position. Preserve the identifier and date/time strings until you deliberately select a conversion.

| Output field | Method | Source / operation | Expected value |
| --- | --- | --- | --- |
| `date` | position | token 1 | `"20250811"` |
| `time_timezone` | position | token 2 | `"080053105+0100"` |
| `hostname` | position | token 3 | `"aaa-gmailrgi009"` |
| `service` | position | token 4 | `"smtp"` |
| `pid` | position + integer type | token 5 | `3692` |
| `minor_pid` | position + integer type | token 6 | `0` |
| `thread_id` | position + string type | token 7 | `"140026854221568"` |
| `loglevel` | delimiter | token 8, `;`, part 1; confirm interpretation | `"Note"` |
| `event` | delimiter | token 8, `;`, part 2; confirm interpretation | `"SieveRuleMsg"` |
| `status` | key_value | key `status`, key/value separator `=`, colon pair boundary | `"failed"` |
| `reason` | key_value | key `reason`, same scanner | `"notneeded"` |
| `server` | key_value | key `server`, same scanner | `"cloud"` |

`Note;SieveRuleMsg` is intentionally split without retaining the semicolon in either output value. Field names and display labels are operator choices: use `loglevel`/`event` or your own schema. Suggestions remain reviewable and cannot know an arbitrary application's business semantics.

The semicolon/multiword variant `status=failed;reason=not needed` must preserve `not needed` as one value. A colon is a pair boundary only before a valid next key plus key/value separator. Colons inside a URL, IPv6 address or compound value remain part of the value until explicitly selected for splitting. Duplicate keys require an explicit policy; they must not silently disappear.

## Workbench sequence

1. Open Field Mapping, paste a representative raw record or inspect retained evidence in the wizard. Use source token chips and character offsets to locate the desired input. For a corpus, supply representative variants and held-out cases; review structural-frequency suggestions rather than treating “training” as automatic production readiness.
2. Use **Suggest from raw log**, then review names, labels, types and presence policies. Add a field when the intended schema is not suggested. Do not force absent evidence into a required output.
3. Select the extraction method and source. For `loglevel`/`event`, choose token 8 with a semicolon split, parts 1 and 2. For `status`, `reason` and `server`, choose their key names and the intended pair delimiter behavior.
4. Add ordered `operations` when further splitting, substring selection, replacement/removal, prefix/suffix handling or field merging is needed. Preview after each meaningful change. Order matters: split before selecting a part, strip a prefix before interpreting a nested payload, and convert types after obtaining the intended value.
5. Test all fields and use the authoritative controller preview. Compare exact JSON, absent/empty/null behavior and field-level errors. Local hints alone cannot certify a candidate.
6. Save/export the mapping draft or import a reviewed plan. Generate the parser candidate with the same plan and validate all cases in Step 6. Submit, approve, publish, obtain compatible-client exact-hash acknowledgement, replay and verify through the embedded workflow.

## Extraction and composition options

| Method | Operator intent |
| --- | --- |
| Position / word token range | Select one whitespace token or an inclusive sequence; join selected values with an explicit separator when configured |
| Delimiter | Split a selected token or whole raw value with an explicit delimiter and select a 1-based part |
| Character | Select Unicode character positions; use character counts for text, not byte counts |
| Byte | Select byte positions only when the format defines them; do not split a multibyte character |
| Line | Select one or more lines within an already framed multiline record |
| Key-value | Select a key with configured key/value and pair separators; preserve quotes/escapes, multiword values and internal equals |
| JSON path | Select a bounded object/array path; unrestricted JSONPath programs are not executable |
| Regex | Use a bounded RE2 pattern/group when a positional or structured selector cannot express the format |
| Constant | Add an explicit configured value; never describe it as observed raw evidence |
| Derived | Combine previously mapped fields with an explicit join separator; record derived provenance |

Supported ordered transformations are exposed through the operations editor: split/select, substring, literal replace/remove, trim, lowercase/uppercase, add prefix/suffix, strip prefix/suffix, bounded regex extraction/replacement, and joins/merges. Nested key-value extraction is a separate data-only operation. Rename a field through its name control; remove, duplicate or reorder a mapping through its row actions. These edit the mapping and resulting JSON, not the retained original raw record.

For a synthetic nested example `msg=REPORT=eventType;summary|action;keep|geo;ZZ`, extract outer `msg`, strip prefix `REPORT=`, then extract `action` with key/value separator `;` and pair separator `|`. This gives `keep`; it must not treat every semicolon in the whole log as an outer pair boundary. Keep opaque user/address/identifier compounds intact unless an operator explicitly maps their components.

## Types, absence and validation

String values preserve leading zeros and identifier precision. Integer, finite number, boolean, timestamp, IP, port, object and array conversions must satisfy native validation. Set an explicit timestamp format and timezone policy when the raw value is not a complete unambiguous timestamp. Native timestamp layouts support Go time layouts and explicit epoch_seconds, epoch_milliseconds, epoch_microseconds or epoch_nanoseconds. Configured timezones require a valid IANA timezone_name; UI flows may expose preserve/UTC first. Port bounds remain 0–65535.

Choose required, optional, nullable, explicit-default or derived behavior deliberately. Missing, empty string and JSON null are different cases. Null behavior can omit, preserve null or reject according to the field policy. Defaults and constants are configured evidence, not observed source values. Numeric/length/allowed-value constraints must be representable by the shared runtime plan; unsupported rules return errors.

Duplicate-key policies are explicit (reject, first, last or array where applicable). An array result needs an appropriate output type. Derived fields reference existing mapped fields; forward references/cycles are not an arbitrary expression language.

## Service scope and safe publication

Review service scope alongside the output schema. Choose the intended service or services inferred from reviewed cases. Canonical aliases are `imapserv`/`imap`, `popserv`/`pop`, `webmail`/`webtop`, `pss`/`pabd`, and `mss`/`capd`; `smtp` includes inbound/outbound variants. Unknown auto formats use `__unclassified__`, keeping them separate from unrelated known services.

An optional `scope_pattern` constrains matching original raw records. It is a Go/RE2 expression of at most 1,024 bytes; every reviewed case must match. Constant/derived-only mappings receive an exact escaped case-input guard unless one is supplied explicitly. Wildcard scope is fallback by default. Authoritative global override requires both the explicit global-scope control and a nonempty reviewed pattern. Test unrelated records before publishing. Scope guards and extraction patterns have separate size limits.

## Boundaries, limits and rollback

The native schema allows at most 128 fields, 16 operations per field, 1 MiB raw input and 1 MiB aggregate mapped output, 65,536 tokens or split parts, 4,096-byte RE2 patterns, JSON nesting depth 32, 8,192 JSON nodes and 4,096 array elements. Field identifiers are 1–64 non-reserved characters. Native integers must fit the browser-exact range ±(2^53−1); preserve larger identifiers as strings. Corpus discovery accepts 1–100 samples in a 2 MiB request, each within the 1 MiB raw limit, and emits up to 16 structural profiles with 128 fields each. Diagnostic messages identify skipped/ambiguous samples or profile limits. These are hard limits, not a claim that every bounded input has a supported interpretation. Character selectors count Unicode code points with 1-based inclusive positions; byte selectors are separate. Regex is RE2-compatible and does not execute code, backreferences or look-around. JSON paths are bounded selectors. Malformed quotes, truncated required records, invalid numeric values or exceeded limits remain actionable errors/quarantine rather than invented output.

NFI mapping-stage packs require an R7-compatible target client. Existing R6 packs stay valid. Upgrade the target client before rollout, and verify desired/applied bundle hashes. Keep the previous reviewed pack and genuine R6 RPMs for rollback. Removing an active parser creates a governed bundle change; it does not silently alter fleet state.

The mapping guide documents behavior and workflow. `V21_R7_TEST_RESULTS.md` records executed evidence; unexecuted scenarios, broad protocol support, production capacity and an LLM weight-training claim must not be inferred from this guide.


The Supabase bootstrap generated by R7 identifies `21.0.0-7` while keeping compatible RPC schema version `1`. Its existing revisioned JSONB storage persists mapping plans, scopes and review metadata without new NFI tables. Run the complete installation-specific SQL file when instructed by controller setup; approval and validation remain NLI responsibilities. See `WEBVIEW_SUPABASE_DEPLOYMENT.md` for compatibility and secret-handling details.
