# NLI Ver20 Release 4 — Quarantine Resolution Operator Guide

Release identity: `20.0.0-4`.

## One guided workspace

Use **Assurance & Release → Quarantine Resolution Wizard**. The highlighted card and **Next required action** identify the only required next step. Completed status comes from controller evidence and cannot be advanced by changing tabs.

1. **Select** — select one actionable retained record, then choose **Inspect**.
2. **Inspect** — verify the exact raw evidence. Enter desired field names in source-token order, separated by commas or new lines, and create mappings. Automatic suggestions remain editable; names do not fabricate values.
3. **Schema** — confirm names, types, required/optional policy, and null policy.
4. **Map** — confirm each value comes from a token position, key/value pair, safe regular-expression group, delimiter segment, or explicit default.
5. **Generate** — generate the deterministic candidate and validation case, then save the review-gated draft.
6. **Test** — run the candidate preview. Continue only when the exact candidate passes at least one selected retained case and returns its artifact checksum. This never activates the parser.
7. **Review & publish** — submit and approve the exact parser revision, then publish. Normal operation uses a different reviewer. If policy enables administrator emergency recovery, the administrator must explicitly select emergency approval, enter a reason, and bind approval to the displayed revision and checksum.
8. **Deploy & replay** — propose the exact published bundle to the origin client, approve the exact proposal, wait until desired and applied hashes match, and request replay. Emergency rollout approval has the same explicit reason and exact-hash requirements.
9. **Verify result** — confirm the controller reports the record resolved. Roll back or offboard if replay fails or the parser produces an incorrect structure.

## Notification behavior

Sidebar badges count unresolved operator work, not historical messages. **Mark read** clears an unread badge for the current user, **Mark unread** restores it, **Check later** removes it from the active feed until its snooze policy expires, and **Delete** dismisses it only for that user. Audit records and retained lifecycle evidence remain intact.

## Existing parser packs

Compatible historical records are assigned a deterministic current runtime identity and validated before synchronization. A parser shown as **blocked** is not silently activated: use **Repair as draft**, correct its runtime rules or identity, test it in this wizard, and complete the governed lifecycle above. Runtime-ID collisions remain blocked until the conflicting identities are resolved.

## Acceptance check

Use an intentionally unsupported sample. Confirm it enters quarantine once, map operator-chosen fields, obtain a passing non-zero candidate preview, publish the exact revision, deploy the exact bundle, observe the origin client acknowledge the same hash, replay once, and verify the normalized output count increases by one while quarantine decreases or moves to resolved. Preserve screenshots and the displayed revision/checksum/hash as deployment evidence.
