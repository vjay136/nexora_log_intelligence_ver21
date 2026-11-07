# Ver21 R10 release notes

Nexora Log Intelligence `21.0.0-10` uses released Ver21 R9 (`v21.0.0-9`) as the upgrade and rollback baseline. The release includes the client and controller RPMs, the WebApp and WebView, and their verification evidence.

## R10 maintenance update (21.0.0-10.1.1)

Install the controller and client RPMs from the same maintenance bundle, then use its WebApp or the controller-served WebView. This update uses the original R10 RPMs as the upgrade and rollback baseline.

- Both Guided Parser Repair Studio entry points support pasted samples alongside retained quarantine evidence. Sample verification uses the published parser and selected client acknowledgement without submitting a quarantine replay.
- The wizard separates parser analysis attempts, targeted replay attempts, exact bundle acknowledgement and the controller-reported result. Step 9 shows retained record history, including cases where replay is unavailable.
- Review Queue keeps lifecycle buttons beside a compact, expandable validation result. FieldSense Studio identifies the KV Intelligence Engine and shows where to inspect its grammar.

## KV Intelligence Engine

The FieldSense and quarantine repair workflows expose KV analysis where the operator edits the mapping. Analysis separates assignment operators from pair boundaries. This matters for tails such as `|status:failed|reason:notneeded|server:cloud`, where `:` assigns values and `|` separates pairs. A reviewed rule can live alongside the positional header fields in a single ParserPack.

The R9 mapping editor surfaced three related failures.

- Tail creation supplied an advanced assignment list containing only `=`. The Go extractor preferred that list over the visible single-assignment field. Changing the visible field to `:` could leave the executed grammar unchanged.
- The browser preview could label an extended mapping invalid before the native extractor tested it. That mixed an analysis requirement with a parse failure.
- The row's Preview action fell through into the add-tail handler. It raised a duplicate warning even when the operator only wanted a preview. Add dynamic KV tail now takes the operator to the existing tail and opens its inspector.

R10 places native controller analysis in the mapping workflow. **Inspect KV grammar** presents bounded extraction candidates and **Apply grammar** applies the selected settings to the draft. **Preview mapping** runs the draft against the source through the controller. Review the suggested grammar and extracted fields before generating a candidate. Approval, publication, rollout, applied-bundle verification, and replay remain separate operator actions.

For R10 mapping plans that opt into KVIE, inspection and execution use the shared native lexer. AutoSense compares a bounded set of grammars. It reports conflicting interpretations for explicit review. A configured grammar uses the chosen assignments and pair boundaries and can be reused for fields that share the same source span. Historical plans retain the grammar they were saved with.

Configured profiles support literal multi-character pair separators, quote and escape characters, custom bracket pairs, a bounded RE2 key pattern, and an empty-value policy. A named field can use the same source range and profile for explicit typing and ordered transformations. Selecting a narrower range or excluding a key is an operator decision; a failed structured-tail parse cannot be promoted into a generic normalized record by the candidate. Review independent matching parsers in the existing parser pool before assuming a quarantine outcome.

## Operations and qualification

Use the same R10 release archive for both RPMs and the web assets. Verify the external archive digest and internal `SHA256SUMS` before installation. Upgrade both sides before rolling out definitions that use R10 capabilities. Use Parser Fleet Sync and Status of All Communication to verify the target client and its acknowledged revision.

The [manager demo guide](V21_R10_MANAGER_DEMO.md) walks through extraction, failure recovery, operational tabs, and multi-endpoint delivery. The [qualification record](V21_R10_TEST_RESULTS.md) describes the tests and evidence needed for this release.
