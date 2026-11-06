# Nexora Log Intelligence Ver21 Release 7

Release `21.0.0-7` uses verified R6 `v21.0.0-6` at `905ce45bca9d54d2fd8fcf4505ba21e3e274a611` as its source, upgrade and rollback baseline. R6 remains available as the previous release.

## NFI — Native Field Intelligence

NFI is the native, bounded field mapping subsystem. Its dedicated **Field Mapping** workspace (`field-mapping`) shares mapping definitions with Quarantine Resolution Wizard Steps 3–4, Assurance previews and the parser candidate lifecycle. The mapping plan follows the artifact through review, publication and client execution; preview-only behavior must never be silently substituted in production.

Operators can identify numbered source tokens and character positions, assign custom field names and labels, split compound values, extract key-value data, select token/character/byte/line ranges or structured paths, combine fields, and compose ordered transformations. Presence, null/default, data-type, duplicate-key and validation rules are explicit. Unsupported or ambiguous definitions fail with field-level errors rather than fabricated values. See `V21_R7_FIELD_MAPPING_GUIDE.md` for the supported operations and exact sample.

Suggestions use observed structure rather than exact full-line examples. A corpus analysis or “training” action means deterministic structural discovery, field-frequency evidence and reviewable mapping proposals. It does not train external model weights, reinterpret opaque values automatically or publish a parser without review. Supplied corpus bodies are not added to public release artifacts; regression cases use synthetic or explicitly authorized examples.

## Compatibility and governance

Existing native parsers and validated R6 regex packs retain their runtime path. NFI adds a mapping stage rather than replacing all parser implementations. Controller/agent RBAC, administrator approval, signed exact-hash bundles, client acknowledgement, quarantine disposition, replay limits and rollback remain in force. Raw-log framing, transport and destination behavior keep their existing contracts.

Use the genuine R6 RPMs for upgrade and rollback qualification. Package/service names, module path and durable configuration/state locations remain compatible. R7 release assets include `nli_client_ver21.rpm`, `nli_controller_agent_ver21.rpm`, WebView/webapp ZIPs, installers, checksums/provenance and `nli-v21.0.0-7-x86_64.tar.gz`. The R7 release notes, results, mapping design and guide ship in both RPMs and the archive.

## Evidence and limits

`V21_R7_TEST_RESULTS.md` records measured qualification. The design document contains the R6 audit and issue-to-improvement matrix, compatibility boundary, original implementation provenance, and the broader engineering-prompt capability checklist. No claim is made that every unknown log format has a unique interpretation, that all listed protocol parsers were rewritten, or that NFI has unmeasured universal accuracy or throughput.


## Supabase bootstrap identity and persistence

The installation-specific SQL file now identifies Ver21 R7 (`21.0.0-7`) and clearly directs operators to run the complete generated file. The packaged template remains a template until the setup helper renders its controller digest. RPC schema version 1 is intentionally unchanged: existing revisioned JSONB already stores NFI plans, ordered operations, scopes, nullable/object/array values and review metadata. R6 document revisions remain compatible. Authenticated SQL health adds release and persistence-capability metadata; controller/runtime validation and approvals remain authoritative. No live project is changed by the release build.
