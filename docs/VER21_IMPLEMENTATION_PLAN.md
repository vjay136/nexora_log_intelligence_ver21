# Nexora Log Intelligence Ver21 implementation plan

## Source and release boundary

- Authoritative source: Ver20 R6 (Release 6) tag `v20.0.0-6` (`4f18ecb`).
- Working release: Ver21 `21.0.0-1` in the dedicated Ver21 repository.
- Compatibility boundary: retain RPM package identities, systemd units, state/configuration paths, and Ver20 artifact aliases so an in-place upgrade does not fork production state.
- No feature is permitted to fabricate parser readiness, telemetry, client acknowledgements, or AI results.

## High-priority implementation tracks

1. **Authentication and notification correctness**
   - Treat the anonymous login-page `GET /api/v1/auth/me` 401 as an expected session probe.
   - Keep genuine authenticated authorization, storage, and controller failures visible.
   - Test the browser probe and the controller notification classifier independently.

2. **One sealed parser lifecycle**
   - Converge Assurance Lab, Quarantine Resolution Wizard (QRW), Parser Creation Workspace, and Parser Catalog on one declarative runtime pack and retained regression-case contract.
   - Validate the exact runtime artifact before review.
   - Bind approval, publication, rollout, client acknowledgement, and replay to the same immutable SHA-256 bundle identity.
   - Supersede stale pending proposals when a new canonical bundle is published; keep the superseded record for audit but remove it from actionable counts.
   - Permit an administrator author to approve the exact revision/hash only with a recorded reason. Non-admin authors remain subject to independent administrator review.
   - Make publish additive. Offboard and rollback generate a new canonical bundle and require the normal exact-hash rollout acknowledgement.

3. **Synchronization isolation and recovery**
   - Use `/api/v1/sync/stream` consistently through controller, Vercel, and Cloudflare paths.
   - Parser draft/validation failures remain resource-local and must never stop the canonical telemetry stream.
   - Preserve last-known measurements during transport recovery without presenting them as live.
   - Keep snapshot polling as bounded recovery and deduplicate transient browser notifications.

4. **Client convergence**
   - Apply only controller-signed, exact-hash bundles.
   - Reconcile managed parser overlays completely, including removing a parser intentionally offboarded from the newer bundle.
   - Report desired/applied revision and hash so the controller can distinguish pending acknowledgement from drift.

5. **AI provider management**
   - Support OpenAI, NVIDIA NIM, Anthropic Claude, Google Gemini, Qwen, MiniMax, OpenAI-compatible endpoints, and local Ollama through provider adapters.
   - Accept credentials from encrypted controller storage, an environment-variable reference, or a bounded root-owned file reference. Never return a secret through the API or telemetry.
   - Provide template import, manual configuration, safe connection testing, priority/fallback ordering, enable/disable controls, and redacted status.
   - AI remains a final, review-only fallback after deterministic parsing fails; it cannot directly publish or roll out a parser.

6. **Release qualification**
   - Run Go unit/integration tests, race detection, vet, JavaScript contract tests, Vite build, browser UI tests, deterministic release packaging, Rocky Linux lifecycle validation, and artifact verification.
   - Exercise a known-format record and a new quarantined format through validation, approval, publication, exact-hash rollout, client acknowledgement, replay, and normalized output.
   - Publish only when the local commit, tag, generated artifacts, evidence manifest, primary repository, and Vercel repository agree.

## Verification and release gates

- A valid new parser cannot become `approved` while runtime-incompatible.
- A stale rollout proposal cannot approve a different current bundle.
- A valid administrator exact-revision approval succeeds with a reason and is audited.
- Parser creation or validation failure does not interrupt `/sync/stream`.
- A client never acknowledges a hash it did not apply.
- Offboarded managed parsers disappear from the next applied client overlay.
- Anonymous login probing creates no system-error notification.
- AI credentials never appear in API responses, logs, notifications, or downloaded configuration.
- All release workflows and published artifacts identify Ver21 `21.0.0-1` while retaining documented upgrade-compatible package/service names.
