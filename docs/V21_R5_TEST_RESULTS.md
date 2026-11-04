# Nexora Log Intelligence Ver21 Release 5 Test Results

Release: `21.0.0-5` (R5)  
Previous production baseline: `v21.0.0-4` (R4)

## Verification evidence

| Check | Result |
| --- | --- |
| Web API contract check | PASS — 28 checks |
| Browser UI suite | PASS — 74 scenarios |
| Focused QRW field-mapping coverage | PASS — 3 scenarios |
| Vite production build | PASS |
| Dockerized Go suite (`go test ./...`) | PASS |
| Git diff validation | PASS |

The QRW mapping coverage exercises operator field names and labels, deterministic extraction and range behavior, mapping bounds, and overlap validation. The UI coverage includes wizard progression, exact-bundle replay gating, and parser lifecycle presentation. Release metadata names R4 as the production baseline.

## Evidence boundary

These results record the checks listed above. They do not claim additional production capacity, cross-version lifecycle qualification, or deployment acceptance beyond the reported checks. The ingestion, SSE/live sync, approval, hashing, rollout, and replay contracts remain those of R4.
