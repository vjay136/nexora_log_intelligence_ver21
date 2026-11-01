# NLI Ver20 Release 5 — Quarantine Resolution and Rollout Guide

Release identity: `20.0.0-5`.

## Approve and deploy the exact parser bundle

Use **Assurance & Release → Quarantine Resolution Wizard**. Steps 1–7 remain the evidence, mapping, candidate, deterministic test, review, and publish workflow. Release 5 strengthens Steps 8–9:

1. In **Step 8 · Deploy & replay**, create an origin-client or all-eligible-client proposal for the displayed current bundle.
2. Enter an approval reason in the proposal row. A different administrator or reviewer uses **Approve reviewed proposal**.
3. If the administrator is also the creator and emergency recovery is enabled, use **Emergency approve exact bundle**. The controller requires a 12–512 character reason and binds the action to the exact proposal revision and canonical SHA-256.
4. Wait for the origin client to report `applied_hash == desired_hash` with no drift. Approval alone never marks delivery complete.
5. Request targeted replay only after that exact acknowledgement. **Step 9 · Verify result** completes when the retained record is resolved by the deployed parser.

## Existing pending approvals

R5 verifies the embedded immutable bundle, not a UI abbreviation:

- A current canonical proposal is approved normally.
- A legacy full or shortened hash is migrated only when its embedded semantic snapshot is exactly the current published bundle. The audit trail retains both hashes.
- An older, malformed, or unverifiable proposal is not deployed. Cancel it and create a new proposal from the current published bundle; the UI shows the controller's repair guidance inline instead of producing repeated generic POST notifications.

This preserves old safe work without allowing a stale parser to reach clients.

## Approval and replay troubleshooting

- **Independent approval required** — sign in as a different authorized reviewer, or use the explicit administrator emergency action when policy permits.
- **Proposal is stale** — cancel and re-propose the current bundle; do not reuse an older hash.
- **Snapshot cannot be verified** — retain the audit record, cancel the pending action, and create a fresh current proposal.
- **Replay blocked: exact bundle not acknowledged** — verify the client is online and allow Parser Fleet Sync to apply the bundle. Retry only after desired and applied hashes match.
- **Client rejects the bundle** — compare controller and client package versions, the full 64-character hashes, and Parser Fleet Sync diagnostics. R5 controller and client RPMs must be upgraded together.

## Login presentation

The complete product banner appears in a larger responsive landscape stage above the credentials. It uses containment, so all source labels and artwork remain visible on desktop, tablet, and mobile screens.
