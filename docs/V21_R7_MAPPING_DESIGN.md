# Ver21 R7 NFI — Native Field Intelligence: audit and design

Baseline: R6 `v21.0.0-6`, source `905ce45bca9d54d2fd8fcf4505ba21e3e274a611`. This is an incremental native mapping extension, not a replacement of the NLI data path.

## R6 audit

The audit inspected `web/app.js` mapping helpers (`qrwSuggestedFields`, `qrwExtract`, `qrwBuildParser`), controller Assurance definition validation and `assuranceRuntimePack`, plus the ParserPack model, stage executor, schema checks, regression cases and source evidence. R6 already has bounded native parser stages, reviewed parser packs, typed schema checks, quarantine, exact artifact/bundle hashes, administrator review, client synchronization, replay, output routing and release qualification.

The mapping editor is **PRESENT_BUT_INCOMPLETE**: the suggestion helper recognizes two exact full-line samples, otherwise emits generic whitespace positions; key-value preview recognizes semicolon/comma/space boundaries but misses the requested colon boundary; delimiter preview splits one whitespace token; generating a regex rejects several controls offered by preview. `assuranceRuntimePack` permits only validated regex candidates. These are confirmed contract gaps; they do not prove that all native parser stages are incorrect.

Existing approval, disposition, streaming, framing, fleet identity and native parser paths are **PRESENT_AND_CORRECT for the R6 qualified fixtures** and are preserved. Broad unknown-format and capacity claims remain **NEEDS_INVESTIGATION** beyond measured fixtures. No legal compliance conclusion is inferred from code inspection.

## Issue-to-improvement matrix

| Current issue / root cause | Reproducible evidence | Expected behavior / NLI-native solution | Accuracy, performance and compatibility impact | Verification |
| --- | --- | --- | --- | --- |
| Colon KV not split by browser regex | `status=failed:reason=notneeded:server=cloud` | Native quote/escape-aware scanner recognizes separators only before a valid next key and keeps value-internal colons/equals | Prevents accidental merged values; bounded scan; existing parser stages unchanged | Golden sample, URL/IPv6, quotes, reordering and internal equals |
| Compound token label lost or mixed with a whole-token duplicate | `Note;SieveRuleMsg` | Select token 8, split `;`, choose parts 1 and 2, assign `loglevel` and `event` | Both fields retain explicit source evidence; overlaps are allowed when intentionally selected | Exact two-field split and explicit confirmation |
| Many editor controls cannot generate a publishable regex | R6 generator rejects transforms/default/null/object/array and several methods | Serialize an NFI mapping plan and execute it through one shared Go implementation in preview and runtime | Removes preview/runtime semantic drift; preserves R6 regex compatibility | Definition preview, saved candidate, sealed pack, normalized result equality |
| Token/character positions are difficult to identify | Spaces, punctuation, Unicode and multiline samples | Show token indices and character offsets; explicit extraction unit and inclusive ranges | Reduces off-by-one errors; bounds apply to each selector | Unicode, ranges, missing positions and split-byte boundaries |
| Repeated or nested delimiters need custom logic | Synthetic nested `msg=REPORT=action;keep|geo;ZZ` | Apply ordered strip-prefix and nested key-value extraction; retain outer raw value until operator maps it | Local deterministic composition; no executable expressions or automatic semantic invention | Nested-operation golden cases and malformed input |
| Exact full-line examples overfit suggestions | Same header shape with different dates/hosts/values | Generic structural inference and optional corpus field-frequency evidence | Suggestions are proposals, not proof; runtime stays deterministic for an approved plan | Variant and held-out synthetic records, false-positive/absence checks |
| Quoted/multiword/empty values are ambiguous with simple splitting | Synthetic `agent="Example Client/1.0" session=""` | Quoting/escaping and missing-versus-empty behavior are explicit | Preserves user-agent and empty evidence; required policy still enforces schema | Quoted delimiters, escapes, empty/null/default tests |
| Arbitrary normalization can hide evidence | Removal/replacement/merge requested by operator | Original raw evidence is retained according to policy; ordered transformations and derived provenance are explicit | Reversible draft editing; no mutation of original raw record | Plan round trip, output comparison and provenance tests |

## FieldSense directive: pre-expansion four-status audit

Operator-facing name: **NLI FieldSense Studio**, powered by the existing bounded NFI engine. The canonical route remains `field-mapping`, the 45th workspace; this is an extension of the shared parser model rather than a parallel parser store.

The following audit records the implementation immediately before the expanded FieldSense directive. Statuses describe observed executable behavior, not a visual control or a claim that every future format is supported. Final acceptance must be tied to the operation, controller and client tests below.

| Directive area / inspected implementation | Four-status finding at audit | Gap requiring executable closure |
| --- | --- | --- |
| Structural suggestions, corpus frequency, optionality (`field_mapping_suggest.go`) | Implemented and verified | Extend independently reviewable suggestion controls; never approve uncertain meanings |
| Custom field names, duplication/reorder, mapping presence/null policies (`model.go`, QRW editor) | Implemented and verified | Retain 128-field/name/reserved-metadata limits rather than advertise literal unlimited allocation |
| Positional, character, byte, line, delimiter, KV, regex, JSON extraction (`internal/fieldmapping`) | Implemented and verified | Preserve original-source Unicode/byte semantics and production parity |
| First/last/Nth/all occurrence, before/after/between, counts, reverse splits | Not implemented | Add bounded native operations plus controls and negative cases |
| Ordered split/replace/remove/trim/case/prefix/suffix/nested KV/join/merge | Implemented and verified | Extend conditional merge and explicit source omission without erasing original evidence |
| KV quote/escape protection, duplicate policy and selected keys | Implemented and verified | Add explicit aliases, all-key extraction and include/exclude lists |
| Mask/hash/redact in a mapping chain | Not implemented | Native data-only operations and exact preview/runtime tests |
| Integer/number/bool/timestamp/IP/port/object/array conversions | Implemented and verified | Extend date/email/URL/UUID/custom validator choices; invalid values remain errors |
| Shared field editor controls for native plans | Implemented and verified | Do not confuse locally rendered options with controller verification |
| Visual source selection with back-highlighting and automatic extraction trace | Implemented but incomplete | Token/character inspector exists; complete direct selection/actions and reverse evidence highlighting |
| Independent suggestion accept/edit/reject/regenerate and accept high-confidence | Implemented but incomplete | Existing corpus/profile import is coarse; per-field action history and review required |
| Contextual help for every advanced control | Implemented but incomplete | General hints exist; add hover and expanded examples with boundary/error explanations |
| Sixteen-step Guided Builder and Expert mode | Not implemented | Real state/errors/next-action progression plus shared expert plan editor |
| Native preview→candidate→approval→sealed pack→production adapter | Implemented and verified | Re-run the exact chain with expanded operations; generation alone is insufficient |
| Published valid parser→repair clone while prior active remains | Not implemented | Existing compatibility-repair endpoint handles blocked packs only; add canonical governed clone workflow |
| Catalog/priority, rollout/fleet, artifacts and replay integration | Implemented and verified for existing lifecycle fixtures | Verify expanded plan travels the same canonical bundle; browser handoffs are drafts, not deployment acknowledgements |
| Dedicated resource/cancellation/concurrency isolation for interactive analysis | Implemented but incomplete | Existing size/operation/time bounds remain; verify request cancellation, saturation and configurable limits |

No operation was classified as **UI present but backend functionality missing** merely because its label appeared: native plans were exercised through authoritative preview or rejected as unsupported. Any newly displayed operation without a passing controller/runtime test must receive that status until wired.

### Acceptance evidence map

| Requirement | Executable source / contract |
| --- | --- |
| Suggest & Sense proposals and evidence, quoted/compound/nested formats | `internal/controlplane/field_mapping_suggest.go`, `field_mapping_suggest_test.go`; private 51-record corpus test remains untracked/private |
| Native schema, extraction, transformations, presence and conversion | `internal/fieldmapping`, its unit/fuzz/benchmark tests; `parser-policy` mapping stage tests |
| Exact source positions in agent and scoped JSON override | `internal/adapter/nfi_source_parity_test.go`, `internal/engine/nfi_source_parity_test.go` |
| Scoped authority, canonical service inference and invalid guard rejection | `internal/controlplane/field_mapping_scope_test.go` |
| Exact preview, persistence, administrator review and publication | `internal/controlplane/field_mapping_api_test.go`; expanded `fieldsense_contract_test.go` acceptance supplements this baseline |
| Canonical bundle installation/hash acknowledgement and client adapter | `internal/controlclient/nfi_mapping_test.go`; existing synchronized-asset contracts |
| NFI JSONB, scope, nullable/object/array values and revision preservation | `tests/supabase/schema-contract.sh`, generated bootstrap test, genuine R6 SQL→R7 separate-database qualification |
| Raw selection, help, modes, guided progression and draft handoffs | Browser UI tests must execute the implemented controls; source-marker checks alone do not establish usability |
| Production parser protected during repair and rollback | Expanded FieldSense lifecycle acceptance must compare prior active bundle before publication and restore it after rollback |

## Shared architecture

`raw record → bounded NFI plan → extraction → ordered operations → presence/null/type/validation → typed values and evidence`.

The dedicated `field-mapping` workspace is the 45th primary workspace. It uses the same plan as embedded Steps 3–4. Controller preview validates the plan with the same native implementation used by the client runtime mapping stage. Saved candidates contain the plan; reviewed regression cases, canonical hashes and signed bundle delivery keep the existing lifecycle. Parser management and output handoffs reuse that definition rather than translating transformations to an unrelated browser-only regex.

There is no per-record LLM, external network lookup, arbitrary evaluation, executable plugin, shell command or unbounded configuration program in NFI. Existing source framing, parser dispatch, input checkpointing, quarantine persistence and output routing stay separate. Unknown or invalid mappings fail closed without stopping the shared telemetry stream.

## Corpus discovery and “training”

The supplied corpus identifies header-plus-colon-KV mail shapes, quoted whitespace KV, compound `user` values, nested message payloads, optional/empty fields and truncated records. Corpus analysis counts observed structures/fields and proposes a bounded plan for review. It does not train model weights or silently publish a parser. Date/host/address/user values are not treated as stable structure, and an opaque `user=address:identifier` remains one value until an explicit split is approved. Missing required fields in truncated examples remain visible errors or quarantined records; NFI must not fill invented values.

Only structural findings and independently constructed synthetic examples are published here. The supplied corpus bodies are not copied into the public repository or release evidence.

## Bounds and compatibility

The plan has a versioned data-only schema, unique bounded field names, finite field/operation counts, bounded input and regex sizes, explicit type conversion and finite numeric values. Unicode character indexing and byte indexing are distinct. JSON paths are bounded selectors, not an unrestricted query language. Regex uses the runtime's RE2 semantics; unsupported look-around/backreferences are errors. Derived references cannot execute arbitrary expressions or create dependency cycles. Exact implemented limits belong in the operator guide and native validation tests.

R7 can load old R6 packs. An R6 client cannot be assumed to understand the new mapping stage: upgrade the target client to R7 before proposing a mapping-stage rollout, and require its exact bundle acknowledgement. Do not pretend a completed controller preview is a deployed parser. Keep an R6-compatible prior bundle available for rollback; do not remove R6 packages or overwrite their tag/assets.

## Reviewed scope and dispatch isolation

Authoritative mappings are scoped to explicit services or services inferred from reviewed cases. Canonical aliases match native dispatch: `imapserv` to `imap`, `popserv` to `pop`, `webmail` to `webtop`, `pss` to `pabd`, and `mss` to `capd`. Broad `smtp` scope includes `smtp-in` and `smtp-out`; unknown auto formats use `__unclassified__`. Inference must agree with runtime dispatch and every reviewed case must lie inside the scope.

A scope guard is a reviewed Go/RE2 expression limited to 1,024 bytes, evaluated against original raw input. Constant/derived-only plans without a guard receive an anchored, escaped guard matching only reviewed case inputs. An oversized generated guard requires an explicit bounded guard. Wildcard scope remains fallback by default. Authoritative global override requires both `reviewed_global_scope: true` and valid nonempty `scope_pattern`. Scope is validated and sealed with the runtime pack; approval and bundle acknowledgement still apply.

Character, byte and line extraction preserve original input through preview, adapter and agent; legacy parsers retain trimmed-input behavior. Matching reviewed managed mappings can precede generic JSON parsing, while unrelated JSON retains its fast path. Truncated reviewed mappings cannot be verified or routed as complete normal records.

## Master engineering-prompt capability checklist

| Area | R7 treatment / evidence boundary |
| --- | --- |
| Existing-engine audit and issue matrix | Focused audit above; preserve correct paths and record concrete mapping failures |
| Original algorithms and bounded stages | Native composition/scanning extension; no copied parser implementation or new third-party parser dependency |
| Parser registry and semantic correctness | Existing versioned pack registry, validation, golden cases and reviewed lifecycle retained; unknown meanings not fabricated |
| Format detection and generic suggestions | Structural proposals and corpus frequency evidence; operator validates variants and held-out examples |
| Typed normalized JSONL and quarantine | Existing routes retained; shared mapping result must satisfy its selected schema; failed input remains observable |
| Framing, multiline, rotation and streaming | Existing bounded implementation retained; line extraction does not redefine record boundaries |
| Timestamp/timezone behavior | Explicit selected conversion; no invented timezone for incomplete values |
| Security and hostile input | No executable expressions; field/input/depth/operation/regex limits and negative/fuzz cases |
| Performance | Benchmark native mapping against the scoped R6 baseline; record allocations/time and avoid unmeasured throughput claims |
| RPM, service, restart and rollback | Preserve identities and durable paths; qualify genuine R6→R7 upgrade/rollback |
| Provenance, ownership and licenses | Original implementation trace below; existing dependency inventory retained; no legal certification |
| Long-running production capacity and every listed protocol variant | Not established by this scoped release; require separately measured deployments and labeled fixtures |

## Original implementation and provenance

The new NFI implementation is developed in this R7 change from repository contracts and the user's authorized mapping requirements. No external parser pack, source implementation or third-party corpus is imported. The exact user-provided short mapping example is an authorized regression requirement; broader sample structures are represented by synthetic fixtures. Final implementation commits are traceable through the R7 source/tag, `SOURCE_REVISION`, SBOM, release manifest and test evidence. Public algorithmic concepts, protocol names and field names are not claimed as exclusive property. Existing dependency/license inventory remains the basis for redistribution review; ownership and legal approval are not certified by this document.

## Public specification references

These references explain interoperability and standard-library behavior; they are not imported parser implementations or blanket compliance certification.

- [RFC 8259 section 6, JSON numbers](https://www.rfc-editor.org/rfc/rfc8259#section-6) describes interoperable exact integers in the binary64 range. NFI uses the browser-safe interval ±(2^53−1), preserving larger identifiers as strings.
- [RFC 4180 section 2, CSV quoting](https://www.rfc-editor.org/rfc/rfc4180#section-2) documents quoted comma-containing fields and doubled quotes. NFI uses explicit quote-aware selectors and Go standard-library CSV discovery; non-comma delimiters are NFI options rather than a claim that every dialect is RFC 4180 CSV.
- [Go regexp documentation](https://pkg.go.dev/regexp) documents RE2 syntax and matching time linear in input size. NFI combines this standard-library behavior with finite pattern/input/operation limits; it does not copy a third-party regexp implementation or accept executable expressions.

## Limitations

NFI does not infer a uniquely correct meaning for every arbitrary log. Suggestions need review, new mapping stages require compatible clients, and malformed quoting, missing required evidence, invalid types, unsupported expressions and exceeded bounds remain failures. Current corpus evidence does not prove universal protocol coverage, zero legal risk or global performance superiority. Validation, fuzz, benchmark, browser, packaging and lifecycle results are recorded only after execution in `V21_R7_TEST_RESULTS.md` and final release evidence.


## Canonical integration boundaries

| NLI workspace / route | Shared model or behavior | Acceptance boundary |
| --- | --- | --- |
| FieldSense Studio (`field-mapping`) | `mapping_plan` and source cases, same native engine as embedded editor | Draft edits invalidate preview; controller verification is authoritative |
| Parser Creation (`parser-workspace`) | Handoff of the same parser definition and exact cases | Handoff populates a draft rather than creating a second parser identity |
| Catalog & Priority (`parsers`) | Revisioned `parser-packs` metadata and sealed runtime pack | Persistence makes draft visible; priority and service scope apply only through governed publication |
| Assurance (`assurance`) | Mapping preview, complete candidate preview, exact structural output comparison | Generated configuration alone is not a pass |
| Quarantine Resolution (`quarantine-wizard`) | Selected retained source/evidence ID and shared Steps 3–4 editor | Exact source link, compatible-client acknowledgement and replay remain required |
| UPES (`upes`) | Existing quarantine/recovery state and persisted candidates | Recovery views do not bypass validation or replace active parser state |
| Approvals & Rollouts (`rollouts`) | Exact submitted/approved checksum and bundle proposal | Authorized review/publication plus separate rollout approval gates |
| Parser Fleet Sync (`parser-sync`) | Canonical payload bytes, SHA-256 and desired/applied bundle hash | A successful controller save is not a client acknowledgement |
| Signed Artifacts & Upgrades (`artifacts`) | Existing signed RPM/config distribution and compatible client release | Mapping definitions do not invent a new signing or installation channel |
| Field Integrity & Corpus Proof (`field-integrity`) | Existing evidence/corpus checks over actual typed output | Validate generated field coverage rather than counting editor rows |
| Monitoring/Live Streams (`monitoring`, `streams`) | Existing parser evidence, failure and runtime telemetry | Interactive errors are isolated; existing ingestion/runtime paths remain operational |
| Output (`outputs`) | Existing destination draft maps generated field names | Output handoff does not silently publish destination or parser changes |

FieldSense changes no existing service identity, primary workspace count or canonical artifact ownership. The dedicated Studio is an on-demand authoring client of bounded native preview APIs; record ingestion does not call the suggestion service. Saved repairs use a separate draft identity, leaving the prior active parser available until the replacement is validated, approved and deployed.


### Expanded native toolbox test inventory

`TestFieldSenseToolboxSelectors`, `TestFieldSenseOrderedToolbox`, `TestFieldSenseOutputPoliciesTypesTraceAndImmutability` and `TestFieldSenseEmptyAndBoundedSelections` exercise the expanded native selectors, operation ordering, visibility/type behavior, trace bounds and failed/empty selections. `TestFieldSenseAdvancedPlanLifecycleProductionParity` crosses the controller, storage, exact review/publication, canonical bundle and actual agent engine with email conversion, identifier hashing, filtered all-key extraction, Nth boundary extraction and an intermediate conditional merge. `TestFieldSenseRepairClonePreservesActiveUntilApprovedAndRollsBack` protects the original published parser/hash/bundle, rejects overwrite or unvalidated submission, selects only the approved higher-priority repair and restores the prior exact bundle on rollback. Results are recorded only after these tests execute.


## Expanded FieldSense implementation ledger

This ledger follows the pre-expansion audit above. “Implemented and verified” applies to the named fixture/tests, not a guarantee for every unfamiliar format or deployment. Broad capacity and universal-format statements remain outside qualification.

| Capability / four-status result | Executable evidence |
| --- | --- |
| Expanded character/word/token, occurrence/boundary/count and delimiter toolbox — Implemented and verified | Native `TestFieldSenseToolboxSelectors` and `TestFieldSenseEmptyAndBoundedSelections`; same engine in preview and production |
| KV aliases/assignment variants/filtered all-key extraction, quoted/escaped/empty handling — Implemented and verified | Native toolbox tests and advanced controller→actual-agent filtered-object acceptance |
| Regex groups/all matches and production-compatible syntax — Implemented and verified | Native selector tests; Go RE2 compilation rejects unsupported syntax |
| Ordered splitting/join/conditional merge/copy/normalize/hash/mask/redact/omit — Implemented and verified | `TestFieldSenseOrderedToolbox`, type/visibility tests and advanced publication acceptance |
| Custom names/metadata/type/empty/null/intermediate policies — Implemented and verified | Native output-policy/type/immutability tests; email/hash/intermediate exact JSON acceptance |
| Published mapping repair clone and rollback — Implemented and verified | New distinct identity and reset validation; unchanged original revision/hash/bundle; approved higher-priority repair selected; exact prior bundle restored on rollback |
| Preview resource isolation — Implemented and verified | Admission concurrency/deadline/cancellation tests; no queued saturation; production execution does not generate authoring traces |
| Suggest & Sense / Manual Sense, guided/expert controls, help and live source interactions — Implemented and verified in browser fixtures | Expanded browser suite and contracts; advanced-only operations require controller verification |
| Exact source highlighting — Implemented and verified for authoritative evidence | Native UTF-8 byte spans drive original-source highlights; absent/derived source has no highlight. Local repeated-value search is explicitly a hint until controller verification |
| Literal unlimited fields, every arbitrary format and every unbounded regex dialect — Not implemented | Intentionally excluded: 128 fields/16 operations and documented bounds; Go RE2 only; ambiguous/malformed inputs remain errors |

No advertised advanced operation should remain **UI present but backend functionality missing**: native-only local previews are explicitly unverified and must cross the authoritative controller gate. A fresh failure or unsupported option must be recorded against this ledger rather than described as validated merely because its control renders.


Canonical published parser repair dispatch preserves the original identity: native mapping clones open FieldSense Studio; canonical regex clones open Parser Creation Workspace. Repair metadata and scope remain in the existing parser document. Mapping-only reviewed priority is intentionally separate from legacy parser priority; legacy repair must use the existing Parser Creation/Catalog priority controls and production tests for its chosen selector.
