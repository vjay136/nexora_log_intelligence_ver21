# NLI Ver20 Release 6 — Quarantine Resolution Operator Guide

Release identity: `20.0.0-6`.

## Resolve one quarantined record

Use **Assurance & Release → Quarantine Resolution Wizard**. Keep the raw record and its originating client selected throughout recovery. The highlighted stage and **Next required action** panel show current progress; a stage changes only when the controller confirms its evidence.

1. **Select** — choose one actionable retained record and inspect it.
2. **Inspect** — review the exact raw line. Enter field names in source-token order, separated by commas or new lines. Suggestions are editable; field names do not invent values.
3. **Schema** — choose output type and required/optional and null policies for each field.
4. **Map** — map each field to an observed token/key-value pair/safe capture group or explicit default. Preview each mapping against the raw evidence.
5. **Generate** — generate and save the deterministic parser candidate as a review-gated draft.
6. **Test** — run the deterministic preview against the selected failing record and fixtures. Proceed only when at least one case passes and the exact candidate checksum is shown.
7. **Review & publish** — submit the candidate and approve/publish the same revision. Use an independent reviewer by default. If explicitly enabled by controller policy, emergency administrator approval must include its reason and bind to the displayed revision and checksum.
8. **Deploy & replay** — propose the published bundle to the originating client; approve the exact bundle proposal; wait for that client's applied hash to equal the desired bundle hash; then request replay for the selected record.
9. **Verify result** — confirm the controller reports the record resolved and the resulting normalized output is correct. If validation or replay is wrong, offboard/roll back the parser through the governed workflow before retrying.

## Approval and hash failures

An approval is valid only for the immutable bundle hash shown on its proposal. If a proposal reports a stale or invalid hash, refresh the proposal list and re-create it from the currently published bundle; do not approve a different revision to work around a mismatch. The proposal author cannot use normal independent approval on their own proposal. Each client acknowledgement must match the bundle hash exactly before replay becomes eligible.

## Preserve evidence and badges

Ignore/exclude, delete retained raw evidence, and successful replay are distinct audited actions. Terminal actions remove actionable pending counts when the controller confirms them while retaining permitted audit metadata. Use the active wizard and controller-reported stage state as the source of truth; old notifications are history and do not establish that an approval or deployment is still pending.

This guide explains operator flow; it is not evidence that RPM packaging or Rocky lifecycle qualification has passed. See `V20_R6_TEST_RESULTS.md` for the verified source/UI test boundary.
