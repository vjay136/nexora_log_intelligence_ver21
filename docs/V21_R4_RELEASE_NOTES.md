# Nexora Log Intelligence Ver21 Release 4

Release identity: `21.0.0-4`  
Git tag: `v21.0.0-4`  
Upgrade baseline: `v21.0.0-3`

## Release scope

Release 4 advances the Ver21 RPM release number to 4 and updates the release workflow, reproducible build manifests, evidence generation, artifact verification, and Rocky lifecycle checks to use the published Ver21 R3 packages as the upgrade and rollback baseline.

The release includes the R4 surgical diagnostics and verification runbook. It records the system invariants, parser and Quarantine Resolution Wizard lifecycle evidence, page and inner-tab inventory, failure signatures, and release qualification boundaries for operators and maintainers.

## Artifact and compatibility identity

The public RPM aliases remain `nli_client_ver21.rpm` and `nli_controller_agent_ver21.rpm`; the release archive is `nli-v21.0.0-4-x86_64.tar.gz`. WebView and webapp bundles remain `webview_ver21.zip` and `webapp_ver21.zip`. Installers, uninstallers, AWS helpers, and Cloudflare helpers retain their Ver21 names.

RPM package names (`nli-client`, `nli-controller-agent`), systemd unit names, configuration and durable state paths, and the Go module path (`nexora.local/nli-v20`) remain stable for in-place compatibility. The installed WebView remains under `/usr/share/nli/webview_ver21`. Historical R1, R2, and R3 notes and test evidence remain included alongside the current R4 documents.

## Verification boundary

The R4 test-results document is the record of commands actually run, their outcomes, and remaining qualification. Entries that have not been completed are marked pending. Functional simulations and static checks do not establish universal production capacity; capacity and latency claims require representative, measured environments and must be reported with their scope.
