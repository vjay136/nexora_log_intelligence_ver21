# Ver21 R8 adaptive Design Studio operator guide

Open **Appearance** from the application shell. The Design Studio offers eight styles with names, descriptions and previews. Preview before applying, apply instantly, revert the previous selection or restore the default. Choose the available light/dark mode and accent options; save nonsensitive appearance preferences locally. Refresh should retain saved appearance while leaving authentication under the existing NLI session mechanism.

## Eight design systems

| Style | Intended treatment |
| --- | --- |
| Skeuomorphism | Tactile raised/inset controls with controlled highlights |
| Minimalism | Restrained palette, whitespace and clear dividers |
| Glassmorphism | Frosted translucent surfaces with readable fallbacks |
| Neumorphism | Soft raised/inset surfaces with visible borders and focus |
| Claymorphism | Rounded colorful surfaces with restrained depth |
| Bento Grid | Responsive modular groups preserving all dashboard content |
| Retro Y2K | Professional chrome/blue/gloss accents without obstructive effects |
| Liquid Glass | Layered refraction-inspired gradients and reflections; not physical optical simulation |

## Safe operation

Theme switching changes presentation. Existing navigation, parser creation, FieldSense, approvals, quarantine, output, monitoring and fleet synchronization retain their functional controls. Unsaved form values must remain available during preview and switching. Status text and labels remain meaningful without color; keyboard focus remains visible. Use reduced-motion settings for quieter transitions. Tables may retain their own bounded scrolling region on narrow screens; the page should not overflow horizontally.

If an effect is unsupported, the application uses readable fallback surfaces. Restore the default appearance if a custom preference is unsuitable. Preference persistence stores no secrets and does not publish parser or output changes. A saved style is separate from any server-side configuration.

## Upgrade and rollback

R8 packages include `design-engine.js` and `design-engine.css` in both exported WebApp and WebView, with source/hash evidence. Verify `SHA256SUMS`, `SOURCE_REVISION` and the release manifest before installation. Use genuine `v21.0.0-7` RPMs as the prior baseline for rollback; keep R7 metadata, credentials and encrypted-fixture keys unchanged. The generated R8 Supabase bootstrap keeps compatible RPC schema version 1 and existing R7 JSONB documents. No live deployment is implied by a source export or archive build.


## Read-only record accounting

Output byte size is not a record count: normalized JSON adds field names and metadata, so it can be larger than the raw log while preserving the same number of logical records. Hourly output files may also contain earlier valid history. Preserve those files and compare a controlled before/after delta.

The shipped helper is available as `nli-output-audit.py` in the release folder and `/usr/share/doc/nli-client/nli-output-audit.py` or `/usr/share/doc/nli-controller-agent/nli-output-audit.py` in the RPMs. It reads files and writes only an explicitly selected count/hash snapshot; it never clears, rewrites or content-deduplicates log/output files. Repeated identical source lines can be legitimate separate occurrences.

Before injecting the controlled sample:

```bash
python3 /usr/share/doc/nli-client/nli-output-audit.py \
  --normalised '/var/log/nli/output/auto_normalised_*.jsonl' \
  --quarantine '/var/log/nli/output/auto_quarantine_*.jsonl' \
  --snapshot /root/nli-before.json
```

After input reaches EOF and the pipeline drains to a stable state:

```bash
python3 /usr/share/doc/nli-client/nli-output-audit.py \
  --input /path/to/controlled-sample.log \
  --normalised '/var/log/nli/output/auto_normalised_*.jsonl' \
  --quarantine '/var/log/nli/output/auto_quarantine_*.jsonl' \
  --baseline /root/nli-before.json --single-line
```

Use `--single-line` only when one nonblank input line is one logical record. Otherwise supply a known framed count such as `--expected-records 49`; include known intentional filtering with `--filtered-records`. For example, a controlled 49-record test can balance as 47 normalized plus 2 quarantine records. A true 50th output is a mismatch, not something to delete automatically. In-flight partial JSONL, invalid rows or a truncated/replaced historical file produce an unknown/mismatch result rather than a false pass. A quarantine file not yet created because no records failed counts as zero.

Select only the canonical normalized and quarantine outputs for the same test interval. Do not count a separate quarantine journal, replay results, copied files or multiple fanout destinations as additional unique input records. Remote transports can retain at-least-once delivery semantics; compare their delivery scope separately. No theme preview or helper command changes parser routing, source checkpoints or operational data.


## Retry-state preservation

R8 prevents a successful local append from being repeated when a required remote failure retries the same source occurrence. Its identity is source generation plus logical record ordinal, not raw text; identical records at different positions remain valid. Keep the file queue, delivery receipts and durable input cursor state intact during restart/upgrade. The canonical quarantine output is owned by its dedicated raw queue, so a failed record is not also appended as an envelope to that same file. The guarantee assumes one owning writer and intact durable state; remote destinations retain at-least-once retry ambiguity. Do not delete receipt/queue/checkpoint state as a counting workaround.


See `V21_R8_OUTPUT_ACCOUNTING_GUIDE.md` for the dedicated byte/record/history audit, before/after commands and local/remote delivery limits.


Approved custom parsers join the bounded eligible pool alongside embedded parsers. One source occurrence may produce several parser-attempt diagnostics while still producing one selected customer output. Do not interpret a larger attempt counter as duplicated normalized records. Service/approval/scope restrictions still apply; see the output accounting guide for selection and counting boundaries.


## Automatic discovery uses the current approved pool

Automatic Assurance and Parser Creation Helper requests now read the current published/active canonical parser-pack metadata and use an immutable compiled runtime cache. Drafts and unapproved revisions stay excluded; rollback removes an add-on from the next automatic snapshot without a controller restart. Each request receives its own engine-dependency copy, preserving shared baseline dependencies and explicit manual parser selection.

The analysis-only reader tolerates identical canonical aliases and excludes conflicting runtime identities, invalid packs or attempts to replace protected embedded IDs while keeping unrelated valid add-ons and built-ins available. Canonical fleet publication and atomic bundle synchronization remain strict. Its `pool_hash` identifies an analysis snapshot; it is not a desired/applied fleet bundle hash or a client acknowledgement.

**Suggest from raw** calls the controller with the exact sample and selected service. Structural proposals remain separate from read-only runtime evidence: embedded/published pool IDs, attempted parsers, selected matches and bounded observed fields. Runtime naming proposals are offered only when a source-backed unique primitive value and explicit byte span reproduce the approved parser's value exactly. They do not overwrite operator names, labels or transformation chains. Raw edits invalidate older evidence; canceled/stale responses are ignored. An unavailable controller yields an explicitly unverified local structural fallback, not fabricated runtime proof. No suggestion call persists metadata, creates a parser or changes approval/deployment state.

Runtime matching evidence is limited to 16 samples and about 64 KiB while structural corpus analysis retains its existing 100-sample bound. Existing admission deadlines and cancellation cover contextual pack normalization, approved regression compilation and execution. To preserve source byte/character positions, R8 UI Assurance/helper calls send `preserve_source: true`; this retains CRLF and trailing spaces in the parsed raw/hash. Legacy API clients default to their prior canonicalization unless they opt in. Exact-source HTTP input remains bounded to 1 MiB.


The new automatic-pool discovery and exact-source request fields require an R8 controller. Existing API clients remain compatible with the R8 controller's default normalization, but an earlier R7 controller does not understand the new optional request fields. Earlier targeted R7-controller visual proofs are historical UI evidence; final R8 controller/package qualification remains separate.


See `V21_R8_AI_PROVIDER_GUIDE.md` for configured provider routing, protected key sources, bounded JSON/SSE/image connection checks and diagnostic-only AI review.
