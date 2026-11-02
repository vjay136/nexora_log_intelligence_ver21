# Nexora Log Intelligence Ver21 Release 2

Release identity: `21.0.0-2`  
Git tag: `v21.0.0-2`

## Packaging identity

- Published RPM aliases are `nli_client_ver21.rpm` and `nli_controller_agent_ver21.rpm`.
- Published archives are `nli-v21.0.0-2-x86_64.tar.gz`, `webview_ver21.zip`, and `webapp_ver21.zip`.
- Installer, uninstaller, AWS, and Cloudflare helper filenames consistently use `ver21`.
- The misspelled historical `nli_clinet_*` compatibility file is no longer emitted. RPM package names, service names, configuration paths, and state paths remain stable for in-place upgrades.
- RPM metadata is `Version: 21.0.0` and `Release: 2`.

## Upgrade boundary

The Go module path, RPM package identities (`nli-client`, `nli-controller-agent`), systemd unit names, `/etc/nli`, and `/var/lib/nli*` remain stable. The controller RPM migrates older WebView roots to `/usr/share/nli/webview_ver21`; compatibility is provided through package metadata and migration logic, not legacy release filenames.

## Release evidence

See `V21_R2_TEST_RESULTS.md` for the executed release gates.

## Parser lifecycle and replay reliability

- Parser bundles now carry one immutable revision/hash contract from publication through approval, rollout, client application, acknowledgement, and replay.
- Client rollout results update the parser-fleet state without replacing or stalling canonical live telemetry.
- A successful exact-hash acknowledgement immediately clears drift and unlocks the selected retained-record replay.
- An acknowledgement gate is returned as a structured, retryable workflow state. It remains inline in QRW and never creates a generic API notification or marks live synchronization stale.
- Any authenticated administrator may approve an administrator-created parser proposal, including their own proposal, when a 12–512 character audit reason is supplied. Non-administrator separation-of-duty rules remain enforced.

## One-workspace quarantine recovery

- Quarantine diagnosis, evidence inspection, field mapping, deterministic validation, approval, exact-bundle rollout, acknowledgement, replay, and verification remain inside the Quarantine Resolution Wizard.
- Assurance Lab keeps its independent parser-authoring workflow and uses the same publish/rollout contract.
- Terminal quarantine actions remove actionable badges but retain historical telemetry. Command Center uses retained history when the current live interval is empty, so deleting or excluding evidence cannot erase previously parsed counts.

