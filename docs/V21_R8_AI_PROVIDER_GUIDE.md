# Ver21 R8 AI provider and key integration guide

AI assistance is optional, explicitly requested and deterministic-first. Native extraction, parser validation and approved runtime selection do not depend on a successful model response. AI advice remains diagnostic/review-only; it cannot publish or deploy a parser. Development qualification uses fake HTTP services and no live credentials or paid NVIDIA/provider calls.

## Supported profile kinds

| Kind | Intended adapter |
| --- | --- |
| `openai` | OpenAI chat-completion protocol |
| `nvidia_nim` | NVIDIA NIM OpenAI-compatible service |
| `anthropic` | Anthropic message protocol |
| `minimax` | MiniMax compatible service |
| `qwen_dashscope` | Qwen DashScope compatible service |
| `gemini` | Gemini generation protocol |
| `openai_compatible` | Explicit compatible endpoint |
| `ollama` | Operator-managed local Ollama service |

Create separate profile IDs for separate models, even when they share one endpoint. At most 16 profiles are supported. Lower numeric priority is tried first; fallback stays inside the same configured group, with at most three attempts within a 17-second request budget. AI reviews and connection tests share a maximum concurrency of two. A malformed/unavailable profile is isolated rather than taking native NLI parsing offline.

The user-supplied model IDs `deepseek-ai/deepseek-v4.1-flash` and `minimaxai/minimax-m3.1-flash` are editable NVIDIA examples using the same NVIDIA `/v1` endpoint configuration. For both examples served by NVIDIA, choose `nvidia_nim`: the kind describes the serving API, not the publisher in the model name. Use the separate `minimax` kind for its own service configuration. Their live availability/account access is **not verified** by this release. Confirm endpoint/model access with the account owner and a deliberate connection check; these IDs are not a live model-catalog guarantee or a recommendation to substitute another model.

## Key sources: values versus references

| Source | What to enter | Controller behavior |
| --- | --- | --- |
| `encrypted_inline` | Actual key in the password input | Encrypts the credential before persistence; GET/status does not return plaintext |
| `env_ref` | Plain variable name, for example `NVIDIA_API_KEY` | Resolves the value in the controller process environment |
| `file_ref` | Protected controller-side path under the secret root | Reads a regular non-symlink secret file with no group/other permissions |
| `none` | No credential | Available for the appropriate local Ollama configuration |

Ollama can use `none` or an appropriate protected credential when its deployment requires authentication.

`$ENV`, `${ENV}`, `YOUR_API_KEY` and a `Bearer `-prefixed value are not actual inline keys and are rejected. In `env_ref`, enter the variable name without `$`; in inline mode, enter the real key rather than the variable name. Authentication headers are constructed by the adapter; do not add `Bearer ` to the key input.

Controller environment values belong in `/etc/nli/controller.env`, edited through a protected administrative editor. Restart the controller after changing process environment, then test the exact saved profile again. Do not print environment values into support logs. `file_ref` defaults to the `/etc/nli/ai-secrets` root, configurable with `NLI_AI_SECRET_ROOT`; ensure the controller service identity can traverse that root and read its owner-only file. Secret files are bounded to 16,384 bytes. References resolve only on the controller, never in browser storage or on enrolled clients. Header secret values are not returned by profile/status responses. Retest after endpoint, model, mode, key/reference or key-value rotation changes.

## Configure and test

1. Open AI Resolution & Providers and create/select a unique profile ID.
2. Select the adapter kind and enter the intended endpoint, model, group and priority.
3. Choose the credential source and supply an actual value or the correct reference.
4. Save the profile, then explicitly run a connection check for that saved configuration. A draft check is labeled separately.
5. Inspect the exact `tested_config`: profile/model/host, configuration checksum, timestamp, saved-versus-draft source, response mode, optional image flag and token budget.
6. Resolve categorized errors rather than treating an HTTP 200 or a saved key as proof of usable model output.

The authenticated endpoints are `GET /api/v1/ai/capabilities` (telemetry permission; offline credential readiness), administrator `GET/PUT /api/v1/admin/ai-provider` (`GET?id=...` for a selected profile), and `POST /api/v1/admin/ai-provider/test`. Capability readiness is not a live provider-availability guarantee. Connection checks contact only the selected configured service when requested; development tests use local fake services.

## Clear a credential or remove a profile

Select the exact profile ID in the administrator workspace. **Clear credential** purges that profile's stored revision history, removes protected key/reference/header credentials, and retains disabled metadata for reconfiguration. **Delete profile** purges the profile and its stored revisions. These changes cannot be undone from NLI's revision history; exporting a profile response does not back up its secret.

The administrator endpoints are `POST /api/v1/admin/ai-provider?action=clear-credential&id=EXACT_ID` and `DELETE /api/v1/admin/ai-provider?id=EXACT_ID`. An exact valid ID is required; unrelated profiles are preserved. Successful responses report `stored_history_purged`, `credential_removed`, `profile_retained`, `configured` and `enabled`, without returning the key. A failed history purge returns an error and leaves the profile disabled; retry removal and review backup handling separately. Late connection-test bookkeeping cannot recreate a removed profile. Re-enabling a cleared profile requires a fresh valid credential; merely toggling its enabled state is rejected. The ordinary Disable checkbox preserves credentials and is separate from the clear/delete operations, whose UI confirmation identifies the exact profile ID.

Removal detaches an environment/file reference; it does **not** erase an external secret file or environment value, revoke the provider-issued key, erase external backups, or recall already outbound requests. Revoke/rotate the key at its issuer and remove external references separately when required. Other profiles may intentionally share the same external credential.

## Bounded JSON, SSE and optional image probes

JSON connection testing is supported across all eight adapters. SSE and an optional public HTTPS image URL are available only for compatible adapters/modes that advertise support. The image option is opt-in: NLI passes the validated URL to the provider rather than fetching the image itself. URLs containing user information, query strings or private/local hosts are rejected. Use only public nonsensitive images. Request acceptance does not prove semantic vision competence.

Probe output budgets are 16–1,024 tokens (default 64). Profile timeouts allow 1–120 seconds, while the request-wide budget remains 17 seconds. Response bounds are 1 KiB–8 MiB (default 1 MiB); SSE inspection additionally caps input at 1 MiB, 4,096 events and 128 choices. Raising a probe budget cannot bypass the request deadline.

A successful check requires a complete usable text response, not HTTP 200 alone. Reasoning-only output, a length-limited result or missing terminal completion is `acknowledged_incomplete` and does not set a green `LastTestAt`. Authentication/authorization, rate limit, missing model/endpoint, bad parameters, timeout, network and invalid-protocol outcomes remain distinct diagnostics. A thinking-heavy model may need an appropriate bounded probe configuration; do not mark its empty visible output as usable.

The official [Chat Completions API](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create) and [streaming event schema](https://developers.openai.com/api/reference/resources/chat/subresources/completions/streaming-events) describe complete responses and chunked delta/finish events. NLI's readiness classifications and bounds are its own tested policy, not a blanket guarantee about every compatible model/vendor.

## Suggest with AI: a separate native mapping proposal

Use **Suggest with AI** explicitly for a new or quarantined raw sample, including before a failed native verification. It is separate from native Suggest, manual mapping and the failed-Assurance advisory workflow. Shared controls cover FieldSense, Quarantine Resolution Wizard, Assurance helper/guided repair, Parser Creation and Regex/Templates; the last authoring route requires an explicitly supplied authorized raw sample. Capability readiness is checked first. An unavailable provider leaves native/manual tools usable.

`POST /api/v1/field-mapping/ai-suggest` uses the same ParsePreview permission guard as Assurance analysis. It accepts exact nonempty raw input up to 64 KiB with no NUL, selected service, optional desired JSON up to 64 KiB and an optional manual plan up to 2 MiB as context. Requests share the two-slot/17-second AI budget, with at most three same-group attempts, a 4,096-token generation budget and 256 KiB response limit. This explicit endpoint requires a configured enabled provider; unavailable/disabled configuration returns 503, invalid input 400 and saturated admission 429.

The model receives the actual Go-reflected native Plan/Field/Operation property schema, current methods/types/operations and two public examples, together with masked native structural evidence and approved-runtime context. Manual preview/example/default/constant and operation literal values are stripped from that context; the supplied manual plan is not overwritten. Supported schema is bounded native FieldSense, not arbitrary executable code or a promise to understand every possible format.

The response must be one strict root Plan or `{"plan": ...}` object. Unknown properties, trailing JSON, unsupported methods, JavaScript/invalid RE2, missing selectors and exact-source execution failures return 422. A proposal is compiled and mechanically executed against the original raw bytes before display, with trace and a canonical mapping-parser candidate. An expected-JSON mismatch remains an HTTP 200 reviewable proposal with a diff and `desired_match: false`; without desired JSON, `desired_match` is null. Neither outcome constitutes publication validation.

Review proposal JSON, evidence, validation and hash separately. Existing fields and manual names remain unchanged until **Import** or **Open FieldSense** is explicitly confirmed; stale raw samples and late responses cannot silently import. Dismiss rejects the ephemeral suggestion without changing a parser. Native suggestion accept/edit/reject/regenerate controls and manual naming remain independent.

Every returned field has `confirmed: false`. Mechanical preview may use a transient delimiter-confirmed clone solely to evaluate extractability; that clone is not an approval. The response has `operator_confirmation_required: true`, while `publication_validated`, `ready_for_publish`, `persisted` and `activated` remain false. After explicit import, inspect each mapping, confirm delimiter choices yourself, run native verification and the ordinary saved-draft validation/regression, approval, publication and exact-client-ACK workflow. AI never activates its proposal automatically.

## Reject advice or remove a generated parser

A retained advisory proposal can be rejected with a reason or removed using its separate queue actions. Reject/remove changes the proposal record, not an active parser pack. Ephemeral native-plan suggestions can simply be dismissed before import.

Once a parser has been saved or published, use its normal Catalog repair/offboard/rollback controls. Revocation/offboarding follows the existing authorized rollout flow; an approved published pack is not removed from a running client merely because its AI proposal was rejected. Retain the previously approved parser until the replacement has passed deterministic validation, approval and client acknowledgement. Embedded built-ins remain protected.

## Where the configured provider is used

FieldSense offers explicit AI help after native verification fails: capability readiness is checked, the failed Assurance evidence is recorded, and the existing `/api/v1/assurance/ai-proposal` workflow produces a bounded advisory proposal. Raw/actual/desired/error context is PII-masked. Production remediation requests use nonstreaming text with a 2,048-token budget; streaming/image probes are connection diagnostics rather than the production parser-generation path.

The advisory context combines the same failed analysis with the current approved runtime pool, selected parser and attempt trace, parser-helper evidence and the operator's manual mapping. For an exact raw sample of at most 64 KiB, native FieldSense structural suggestions are also computed under the existing request deadline and two-review admission limit, even when the parsed actual object is empty. This optional structural section is bounded to 4 KiB and contains only whitelisted structural metadata: name, type, method, source, delimiter, part, key/value separator and pair delimiters. Preview, example, default and raw values are excluded from that section; raw/actual/desired values travel only through the separately masked channels. It proposes extraction structure rather than asserting approved field names or changing retained evidence. Larger logs remain supported by native FieldSense within its normal limits, while this extra AI context is explicitly omitted. Cancellation does not create a proposal or alter mappings.

Review the proposed declarative change, run real deterministic validation against exact expected JSON, save a draft and follow the existing authorized parser approval/publication/rollout gates. AI approval or a connection test does not activate a parser, replace manual field names or certify a mapping.

Agent-local legacy AI enrichment remains a separate opt-in local-environment configuration. Controller provider keys are not distributed to agents. That path uses an asynchronous single worker and bounded retry queue, masks prompt PII and stages candidates rather than making them active. An unconfigured or failed provider leaves native parsing and review available.

## Qualification limits

Fake HTTP and focused Go/UI tests cover protocol extraction, routing, safe credential handling and error isolation. Focused removal tests cover history purge, required exact ID, sibling preservation, re-enable without a key, and late-probe protection. The actual local R8-controller/fake-provider browser flow also covers clear → disabled → delete while preserving the sibling profile. No user attachment, credential or private corpus is shipped. No live NVIDIA model, paid endpoint, semantic image capability or remote availability is certified. Final frozen R8 CI and packaged controller/browser evidence are recorded in `V21_R8_TEST_RESULTS.md`; earlier R7-controller visual checks do not qualify these new R8-only API fields.
