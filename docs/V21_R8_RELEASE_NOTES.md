# Nexora Log Intelligence Ver21 Release 8

Release `21.0.0-8` uses qualified R7 `v21.0.0-7`, source `bc6b17a20e1a7b2c11903df440a14ad9134697d8`, as its source, upgrade and rollback baseline. R7 source, tag and published assets remain preserved.

## Adaptive application design

R8 adds a centrally managed adaptive design engine to the existing WebApp and packaged WebView. The eight selectable design systems are Skeuomorphism, Minimalism, Glassmorphism, Neumorphism, Claymorphism, Bento Grid, Retro Y2K and Liquid Glass. The Appearance Design Studio previews and applies styles without a page reload, supports restore/revert and persists only nonsensitive appearance preferences. Existing routes and business workflows continue through their existing APIs and permission gates.

The design registry, resolver and tokens live in `design-engine.js`/`design-engine.css`. The same assets must ship in exported WebApp, WebView, controller RPM and release archive with source/hash provenance. Styles prioritize readable controls, keyboard focus, responsive layout, reduced motion and fallbacks over decoration. Liquid Glass is a CSS refraction-inspired treatment; blur is not represented as physically accurate optical refraction.

## Read-only record accounting

The shipped `nli-output-audit.py` distinguishes bytes from logical JSONL records and compares before/after output counts while preserving earlier hourly history. It reports real extras, in-flight/invalid/truncated output and explicit filtered records without rewriting files or deduplicating legitimate repeated text. Take a count-only baseline before the controlled input, stop/drain at EOF, then compare only the matching canonical normalized/quarantine scope. Quarantine journals, replay and fanout copies must be accounted separately. See the operator guide for commands.

## Compatibility

The mapping engine, canonical parser model, approval lifecycle and fleet protocols retain R7 compatibility. Supabase generated SQL now identifies R8 while keeping RPC schema version 1, existing tables and protected authorization. No theme preference contains credentials or requires a database schema change. The separately reproduced output retry/quarantine corrections are described below; UI styling does not invent operational data.

See `V21_R8_DESIGN_ENGINE_DESIGN.md` for the audit and architecture, `V21_R8_DESIGN_ENGINE_GUIDE.md` for operator controls, and `V21_R8_TEST_RESULTS.md` for observed qualification. Size/operation limits of the R7 FieldSense parser remain bounded and unchanged by appearance switching.

## Qualification boundary

Qualification is tied to the frozen R8 source revision: complete source/browser checks, reproducible genuine RPM/archive builds, checksum/provenance verification, Rocky R7→R8 upgrade/rollback and packaged two-node TLS/WebView acceptance. Observed results are recorded in the release workflow logs, generated validation evidence and accompanying qualification report. No live infrastructure, Supabase project, Vercel deployment or client fleet is changed by this release build.


## Surgical output retry and quarantine corrections

Two reproduced runtime paths are corrected alongside the presentation work. First, a required remote destination could fail after a local file append, leaving the input cursor uncommitted; retrying that same source occurrence then appended an extra local row. R8 identifies deliveries by source generation and logical record ordinal and uses a durable receipt/WAL for the file queue. A successful local append is idempotent for that occurrence while remote retry continues; receipts retire only after durable cursor acknowledgement. Different occurrences with identical text remain separate records, and customer JSON fields are unchanged.

Second, a legacy quarantine path could append the raw failed record and then route its envelope into the same quarantine output. The corrected path uses the dedicated raw quarantine queue as the sole writer for that canonical quarantine output.

The local guarantee is scoped to an exclusively owned writer and its durable state. Preserve queue/receipt/checkpoint state during restart/upgrade; arbitrary external rewriting, multiple independent writers or deleting durable state are outside that guarantee. Required remote transports retain at-least-once ambiguity: this change is not a blanket exactly-once guarantee for Loki or other remote services.

A size listing such as 48K or 738 represents bytes, not logical record counts. Larger enriched JSON output and earlier nonempty hourly history do not establish host-side duplication. Use the read-only count/baseline audit rather than deleting prior output or deduplicating repeated text.


See `V21_R8_OUTPUT_ACCOUNTING_GUIDE.md` for the dedicated byte/record/history audit, before/after commands and local/remote delivery limits.


## Additive parser availability and one output per source occurrence

R8 keeps embedded baseline profiles and approved managed parser packs in one bounded eligible candidate pool. Approval, service selectors, runtime compatibility and reviewed scope guards still determine eligibility. A successful embedded parser does not short-circuit the remaining approved matching pack attempts. Embedded profiles execute first to protect baseline availability under resource limits; failed managed candidates are isolated and cannot by themselves suppress a valid generic JSON/text fallback.

Selection produces one governed customer output after the eligible attempts. Reviewed native mapping scope/priority can intentionally select a custom schema; an ordinary regex match does not silently shadow structurally valid JSON. Original approved parsers remain available while new parsers are added or repaired. Multiple parser attempts are diagnostic activity, not multiple input records or permission to append a customer output for every matching parser. Record accounting compares source occurrences with canonical outputs separately from parser-attempt counters.


## Live approved-parser discovery

Automatic Assurance/helper and FieldSense suggestions now use current published add-ons together with the protected embedded baseline, rather than a built-ins-only startup adapter. Immutable analysis snapshots refresh by `pool_hash`; draft exclusion, rollback removal, invalid/conflicting identity isolation and context-bounded compilation preserve governance and availability. Suggestions expose read-only runtime matches and optional exact observed field-name proposals without overwriting manual mappings or writing metadata. R8 UI opts into exact-source Assurance bytes with `preserve_source: true`; legacy API normalization remains the default for existing clients.


The new automatic-pool discovery and exact-source request fields require an R8 controller. Existing API clients remain compatible with the R8 controller's default normalization, but an earlier R7 controller does not understand the new optional request fields. Earlier targeted R7-controller visual proofs are historical UI evidence; final R8 controller/package qualification remains separate.


See `V21_R8_AI_PROVIDER_GUIDE.md` for configured provider routing, protected key sources, bounded JSON/SSE/image connection checks and diagnostic-only AI review.


## Optional configured AI assistance

Eight provider kinds share protected inline/environment/file credential handling, capability-aware same-group fallback and bounded diagnostics. Connection checks distinguish complete usable JSON/SSE text from incomplete acknowledgements; optional public image-URL probes are operator-requested. Native verification stays first, AI proposals remain review-only and controller keys never propagate to agents. Fake-service qualification does not certify live model availability or semantic vision support. See the AI provider guide for exact limits and account-specific examples.

Explicit **Suggest with AI** offers a separate schema-aware native mapping proposal for new/quarantined raw samples, alongside native/manual controls. The controller strictly compiles and executes the generated bounded plan on exact source bytes; mismatched desired JSON remains visible for review. Import and delimiter confirmation are explicit, and no proposal is saved, approved or activated automatically. Retained advisory proposals support reject/remove; saved/published parsers continue through Catalog repair/offboard/rollback and governed rollout. Profile credential-clear and exact-profile delete purge stored revision history while preserving unrelated profiles; external key revocation remains a separate operator action.
