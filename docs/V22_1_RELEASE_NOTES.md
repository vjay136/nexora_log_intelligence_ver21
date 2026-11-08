# Nexora Log Intelligence Ver22.1

Release 22.1.0-0 upgrades Ver22 22.0.0-2. Extract the complete versioned archive and use its Ver22 installers. RPM package identities and service names support in-place upgrades.

## Quarantine recovery

In Step 8, follow **8.1** to deploy to the origin client or verify its acknowledgement. An approved bundle must be applied and acknowledged by the originating client. When it is ready, use **8.2 · Request targeted replay**. Step 9 shows the controller's replay result. Quarantine badges clear when a record reaches a resolved state. An older failed replay can still appear in history.

The controller preserves the desired parser state when a client with a dotted node ID checks in. It can also recover previously lost state from an exact client inventory and a recorded successful delivery. An active proposal with a finished job no longer blocks a targeted retry.

## Mapping and Known Key Catalog

For a **Key value EOL** row, use the key-edit controls to omit a key, set it to an empty string, or replace its value. For example, omit `session_id`, blank `account`, or replace `reason` with `redacted`. Run the native preview and verify the exact output before publishing. Parser bundles containing these edits require Ver22.1 clients.

The Known Key Catalog supports editing key lists without hand-writing JSON. Copy a published entry into a new draft to revise it, then submit, approve and publish. Published definitions provide evidence for keys actually observed by FieldSense. Revoke a published entry and review the resulting parser asset rollout to deliver the change to clients.

## Fleet and inputs

In **Input Log Paths**, review each client's configured paths and last reported file presence. Configuration delivery reports when the client has acknowledged the change and when the corresponding configuration hash appears in a heartbeat. A rollout to selected enrolled clients can wait for offline clients to check in. Removing an input from a profile stops collection once the revised configuration is applied.

Use a separate approved **Managed input files** operation to copy or delete a file in `/var/log/nli/input` on selected clients. For deletion, first remove the path from the configuration and verify the client has applied it. Copies are limited to 64 MiB and require a new destination filename. The client reports the operation outcome and its latest managed file inventory separately.

**Fleet Operations** separates heartbeat, input, parser, resource, desired-state and issue views. **Client Agent Management** shows reported agent and parser state, offers targeted agent restart and parser synchronization, and tracks per-client job results. **Controller Agent Management** shows controller status and links to related fleet and rollout controls.

## Installation files

The archive is `nli-v22.1.0-0-x86_64.tar.gz`. Its RPMs, web ZIPs and installer names use Ver22. The controller retains its existing on-host WebView location for compatibility with installed configuration and rollback. Historical documents retain the version numbers they describe.

See [Installation and configuration](INSTALLATION_AND_CONFIGURATION.md) and [Fleet enrollment](V22_FLEET_ENROLLMENT.md).
