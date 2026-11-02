# Nexora Log Intelligence Ver21 Release 2 Test Results

Release identity: `21.0.0-2`

## Executed verification

| Check | Observed result |
|---|---|
| Go tests | PASS — `go test ./...` |
| Go static analysis | PASS — `go vet ./...` |
| Go race suite | PASS — `go test -race ./...` in the digest-pinned Go image |
| Browser UI | PASS — 71/71 Playwright cases |
| Focused QRW/dashboard regression | PASS — acknowledgement gating, in-wizard diagnosis, retained history, and canonical snapshot cases |
| Release environment | PASS — `NLI_VERSION=21.0.0`, `NLI_RELEASE=2`, `NLI_ARCH=x86_64` |
| RPM metadata contract | PASS — `nli-client-21.0.0-2.x86_64` and `nli-controller-agent-21.0.0-2.x86_64` |
| Public filenames | PASS — Ver21-only RPM, archive, installer, uninstaller, AWS, and Cloudflare names |
| Typo regression | PASS — no generated `nli_clinet_*` artifact or installer fallback |
| Release evidence contract | PASS — manifest, SBOM, provenance, checksums, and archive use the authoritative R2 identity |
| Upgrade compatibility | PASS — stable RPM/service/config/state identities and WebView migration retained |

The full Rocky Linux installation lifecycle remains a deployment-environment qualification and is not represented as executed by the Windows/Docker repository gates above.

