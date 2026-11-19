# Nexora Log Intelligence v22.20.1-0

- Browser synchronization recognizes each controller process, accepts its new revision sequence after a restart, and rejects late snapshots from the previous process.
- Controller connection checks retry while the login screen is unavailable. An explicit CSRF rejection refreshes protection and retries once.
- Controller and client systemd services continue retrying after startup failures. Clients use bounded, jittered reconnect delays while preserving their enrollment identity.
- The Disaster Recovery workspace exposes protected catalog checkpoints, local asset locations, recovery status, draft restoration and a verified fleet resynchronization proposal. Supabase deployments retain catalog checkpoints every five minutes while the store is reachable.
- Encrypted controller identity exports preserve credentials, TLS trust, enrollment and signing state for offline disaster recovery. Root performs a consistent export or restore with the controller stopped; the webapp stores and downloads only encrypted exports.
- Client Agent Management adds searchable inventory, online/drifted selection, synchronization details and inventory downloads. Existing command approvals continue to apply.
- The shared mapping workbench can combine an exact saved header revision with a KV template or Global AutoSense, show the source byte boundary, and import the verified mapping for further editing.
- Installation option 11 provides certificate-verified TLS using a stable IPv4 address and private CA without internal DNS names. Existing installation options remain available. Nodes still require network reachability; an IP change requires updating the endpoint and its certificate.

## Recovery boundaries

Published metadata remains authoritative in Supabase. Checkpoint restoration stages reviewable drafts; it does not silently activate historical configuration. Fleet recovery uses the existing approval and per-client acknowledgement flow. Offline clients remain pending until they reconnect.

Keep an encrypted host export and its passphrase in separate protected locations before a disaster. Automatic catalog checkpoints do not replace host identities or a backup of raw log files. The webapp accepts encrypted host exports up to 6 MiB; larger offline exports require protected external backup storage. Recovery requires the original state/configuration paths and reachable, valid Supabase credentials when Supabase is configured.
