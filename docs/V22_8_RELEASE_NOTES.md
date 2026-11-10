# Nexora Log Intelligence 22.8.0-0

This release upgrades 22.7.0-0. Use the matching Ver22 controller RPM, client RPM, and web app. The generated Supabase bootstrap reports 22.8.0-0 and keeps RPC schema version 1.

- **Ask NLI AI** accepts a workspace, step, observed issue, and optional raw sample from the Reference based help & Fix workspace. The controller provides guidance tailored to all 50 workspaces.
- **Suggest Mappings with AI** is available in the raw-sample editors, including Building Blocks. Direct authoring requests in Ask NLI AI can build a reviewable draft without redirecting you to another tab. The current raw record is required for mapping and draft generation.
- FieldSense recognizes the reported period/comma mail tail with mixed assignment operators. The regression checks exact source spans for status, reason, and server.
- Verified native seeds use a short structured Gemini request for label suggestions. The browser distinguishes an extraction issue, provider rejection, and controller transport failure.
- Live Gemini smoke tests on synthetic records covered mapping, parser drafting, KV template drafting, regex drafting, and workspace guidance. No production data or saved provider profile was used.
