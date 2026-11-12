# NLI v22.11.0-0

- The controller allows up to 45 seconds for supported AI authoring requests. Start with 15 seconds for Gemini Flash-Lite and 30–40 seconds for slower models. Keep the 1 MiB response limit unless the controller reports an oversized response.
- Use AI Privacy and Knowledge under Controller Management to review the outbound policy, recent decisions and NLI workspace guidance. Production samples from retained evidence use native mapping. Explicit manual samples are masked before outbound requests.
- Use Add advanced dynamic KV tail in the shared mapping editor to compose fixed header fields with a complete, optional KV tail. For the eight-token mail header, the dynamic tail starts at token 9. Name a field for unassigned segments or empty keys only when your format requires those segments.
- Log Pattern Templates and KV Pair Templates can preview two exact saved revisions together. Paste up to 500 records or 2 MiB per request. The controller compiles once and reports line-numbered failures. Open the passing plan in FieldSense, add exact expected regression cases and follow the existing review and rollout flow.
- Model discovery uses the credential and endpoint of a saved profile. The catalog lists model IDs. A model test verifies generation access for the selected ID. Gemma 4 uses its supported minimum thinking setting and text responses, with native validation of the generated draft.
- Upgrade from v22.10.0-0 using the controller and client RPMs.
