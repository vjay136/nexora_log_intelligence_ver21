# Nexora Log Intelligence v22.21.1-0

- A Supabase bootstrap using a different controller secret now aborts without changing the registered secret, stored documents or storage buckets. An upgrade with the existing secret remains supported.
- Metadata API failures expose safe, actionable backend diagnostics. A rejected controller secret is separate from the operator login; it does not sign the operator out or imply that catalogs were deleted.
- Repeated metadata-authentication failures produce one incident notification. Individual failures remain visible in API Monitor.
- Catalog refresh failures retain the currently displayed data with editing disabled. A first-load failure displays an unavailable state rather than an empty catalog.
- All installation modes preserve an initialized controller's metadata backend. Provider or project changes require an explicit migration acknowledgement rather than silently selecting an empty backend.
- Controller and client retain up to ten verified prior synchronized bundles in a private `parser-assets-history` directory before replacement. History is recovery evidence and never activates automatically.
- Regression coverage exercises all reported catalog GET paths and notification POST actions through rejected credentials and restored access. Full release gates cover Linux tests, race detection, browser workspaces and RPM lifecycle compatibility.

## Existing deployments

Use one intended controller for the Webapp and clients. A client node must not run an unintended second controller against the same project. Preserve the primary controller's `/etc/nli/controller.env` during upgrades. Do not apply an older bootstrap from a second node to replace the project's registered secret.

The repair does not recreate records absent from the connected Supabase project. Verify the project reference and use an existing backup or catalog export for missing historical records. A healthy connection to a newly created or different project cannot recover records from the previous project automatically.
