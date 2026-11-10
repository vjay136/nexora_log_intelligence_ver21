# Nexora Log Intelligence 22.7.0-0

This release upgrades 22.6.0-0. Use the matching Ver22 controller RPM, client RPM, and web app. The generated Supabase bootstrap reports 22.7.0-0 and keeps RPC schema version 1.

- **Suggest Mappings with AI** starts from the same FieldSense extraction used by **Suggest from raw log**. The provider receives masked evidence and can suggest labels or a revised native plan. NLI checks the result on the original record and displays exact JSON before import.
- If the provider is unavailable or its draft fails validation, NLI can offer a clearly labeled native draft when native extraction passes. Review any requested fields or transformations that the fallback did not apply.
- **Building Blocks** previews key/value fields through the same engine as FieldSense. Named object values appear on their cards; flattened keys appear in final JSON. Edits invalidate stale previews.
- Preview success describes the sample tested. Approval, publication, and client bundle acknowledgement remain separate steps.
- When expected JSON is supplied, the preview identifies missing, extra, wrong-type, and wrong-value fields. Ask AI about a single finding using the current raw record and field context.
