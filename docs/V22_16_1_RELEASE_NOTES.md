# NLI v22.16.1-0

- Make published Log Pattern Templates eligible for automatic parsing on the controller and clients. Fleet delivery still uses the approved parser-asset bundle.
- Parse the exact suffix after a header using compatible published KV profiles or the built-in Global AutoSense fallback. Require complete, unambiguous extraction whenever a suffix is present.
- Preview a saved header with Global AutoSense or an exact saved KV revision. Global AutoSense is visible in KV Pair Templates.
- Preserve source position ordering in normalized JSON and sample previews where field evidence is available.
- Fleet Sync shows the last confirmed inventory during temporary metadata outages and blocks actions that require current targets. Parser Sync reports stale bundle hashes and ineligible targets without overwriting the installed generation.
- Upgrade the controller and clients to v22.16.1-0 to use live header templates. In Parser Fleet Sync, confirm the client has applied the exact published bundle before retesting ingestion. For existing retained records, use the approved replay workflow. The Command Center keeps historical initial-routing totals separate from replay outcomes.
