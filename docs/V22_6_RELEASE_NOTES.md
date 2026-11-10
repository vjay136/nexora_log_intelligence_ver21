# Nexora Log Intelligence 22.6.0-0

This release upgrades 22.5.0-0. Install from the versioned archive with its companion SHA-256 file and use the Ver22 installers.

For Supabase deployments, generate the bootstrap with the installed Ver22 controller setup and apply the complete generated SQL in the corresponding project. The health RPC reports bootstrap release `22.6.0-0` and RPC schema version `1`. Reapplying the bootstrap preserves existing document revisions. Retain the controller Assurance encryption key so encrypted fixtures stay readable.

- Guided Parser Repair verifies the published parser, automatic output, and the exact client bundle separately. If another parser wins automatic selection with the same output, the wizard says so. Confirmed replay results show a completed state.
- Enable encrypted retention on a pasted sample to save it to Test Sample History and reopen it in Guided Parser Repair. Supabase deployments store these encrypted fixtures under the private test-sample-history prefix. Existing retention settings still apply.
- Log Pattern Templates and KV Pair Templates provide separate workspaces for fixed prefixes and reusable KV grammars. They use the existing revision, approval and publishing workflows.
- Building Blocks shows the final field order in a separate column. Drag tiles or move them with the arrow buttons. The order is preserved in the mapping plan; JSON object key order is not guaranteed.
- Auto Fix with AI can draft from the current sample without opening a dialog. Review and apply the editable draft explicitly; a draft cannot publish or change a live parser.
- Assurance distinguishes rejected provider credentials, throttling and invalid model drafts. Editor validation stays inline, repeated backend failures are grouped in Notifications, and audit records remain available.
- The reviewed SMTP complete-tail pack handles continued assignments, adjacent KV wrappers, and pipe-delimited tails. It keeps captured source values and records ambiguous or incomplete input rather than accepting a header-only parse. Continued assignment content can use the configurable tailed_field output name.
- For example, JSON escape sequences around a sender address represent the original angle brackets. Derived network fields are reported separately from source evidence.
