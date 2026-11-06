# Nexora Log Intelligence Ver21 Release 6

Release 6 is version `21.0.0-6`, based on the verified Ver21 R5 release `v21.0.0-5`. The R5 tag resolves to `88e1abc86d7fcb0efca79654a0f278d75c6dba66`; the R6 checkout starts at `64dbb2f1292efe3fea417f0adf97acffcd026730`, which adds the idempotent release workflow fix.

## Administrator review

Authenticated users assigned the administrator role can approve their own governed changes across Parser Catalog, Assurance, parser assets, AI proposals, rollouts, and NUPIRE. The decision remains audited. Parser validation, exact revision/artifact or bundle binding, and lifecycle evidence remain required. Non-administrator creators still require an independent authorized reviewer. Approval follows the assigned role rather than a literal username.

## Durable quarantine disposition

Exclusion and deletion of retained raw evidence persist when a client uploads the same record again and when bounded queue pruning removes its visible metadata. Deleted/excluded records do not contribute actionable recovery badges or re-enter the wizard. Explicit inclusion can return an excluded record to work. Metadata and audit history remain available. Disposed records cannot be reopened by replay completion callbacks; stale replay results cannot replace a later replay outcome.

## Quarantine Resolution Wizard

Steps 7–9 keep candidate review/publication, rollout approval, exact origin-client acknowledgement, targeted replay, and final verification in the same workspace. Numbered substeps and distinct progress colors identify the next required action. An approved rollout for another client does not satisfy origin deployment. Replay waits for a real acknowledgement of the current desired bundle; an accepted request is shown separately from a resolved record. Validation and replay errors remain local to the workflow.

## Packaging and upgrade

Use the verified R5 RPMs as the upgrade and rollback baseline. Package/service names, module path, configuration paths, and durable state remain compatible. Release artifacts include `nli_client_ver21.rpm`, `nli_controller_agent_ver21.rpm`, WebView/webapp ZIPs, installers, `SHA256SUMS`, provenance, and `nli-v21.0.0-6-x86_64.tar.gz` with an external SHA-256 sidecar. Release and test notes ship in both RPMs and the archive. Consult `V21_R6_TEST_RESULTS.md` for measured qualification status.
