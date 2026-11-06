# Nexora Log Intelligence Ver21 R6 test results

Release: `21.0.0-6`. Verified production baseline: R5 `v21.0.0-5` at `88e1abc86d7fcb0efca79654a0f278d75c6dba66`. The R6 checkout starts at `64dbb2f1292efe3fea417f0adf97acffcd026730`, including the subsequent idempotent release-workflow fix.

## Observed source qualification

| Check | Result |
| --- | --- |
| JavaScript syntax, approval, sync, export, and proxy contracts | PASS: 31 checks (`npm run web:check`) |
| Initial complete browser UI suite | PASS: 77 scenarios |
| Final affected browser scenarios after embedded-workspace and inspection-refresh fixes | PASS: 18 scenarios; one ambiguous preview-button selector was made exact and its scenario rerun successfully |
| Go controlplane, NUPIRE, middleware tests | PASS |
| Go vet for the changed packages | PASS |
| Vite production build and web-only export | PASS |
| R6 release identity, R5 baseline, and evidence-generation contracts | PASS |
| Git whitespace validation | PASS |

Coverage includes administrator-role usernames, non-admin separation of duties, parser artifact versus document checksums, retained validation requirements, quarantine reuploads after deletion/exclusion, pruning/restart persistence, stale/duplicate replay acknowledgements, exact origin-client bundle acknowledgement, Steps 7-9, and self-contained Assurance repair actions.

## Artifact and deployment qualification boundary

The frozen-source release pipeline additionally requires the full Go suite and vet, Linux race checks, deterministic two-pass binaries/RPMs/WebView/webapp assets, sample field accounting, real RPM payload inspection, source provenance, and archive and `SHA256SUMS` verification. Those post-freeze results belong to the build logs and final artifact evidence, not inferred source-test claims. The final binary revision is recorded in `SOURCE_REVISION`, `RELEASE_ID`, and `RELEASE_MANIFEST.json`.

Rocky Linux upgrade/rollback qualification uses the genuine R5 RPMs. GitHub publication and Vercel deployment acceptance are separate from source tests; a web-only source push alone does not prove a live deployment. No production capacity or live-fleet qualification is claimed here.
