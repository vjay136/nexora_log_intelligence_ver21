# Ver21 R8 adaptive design engine audit and design

## R7 audit and compatibility boundary

The qualified R7 baseline is `v21.0.0-7` at `bc6b17a20e1a7b2c11903df440a14ad9134697d8`. It has a Vite-built vanilla JavaScript application, a shared global stylesheet, reusable render helpers and 45 primary workspaces. Existing shell, cards, tables, forms, dialogs, charts, log/code views and permission-dependent actions must be extended rather than replaced. R7 FieldSense, parser APIs, auth, metadata and fleet lifecycle remain the functional baseline.

## Issue-to-improvement matrix

| Requirement / baseline gap | R8 implementation target | Verification boundary |
| --- | --- | --- |
| One fixed visual language | Central registry with eight independent token sets | Registry uniqueness/token completeness and actual computed styles |
| Disconnected cosmetic demos would not affect workflows | Root theme attributes and shared component selectors | Switch while real forms, tables and dialogs remain usable |
| Preferences need safe persistence | Nonsensitive appearance-only local preferences | Refresh/revert/default and malformed-storage tests; no auth data |
| Blur/shadows can harm performance/readability | Controlled surfaces, CSS fallbacks and reduced-motion rules | Representative desktop/mobile checks; no claim of physical refraction |
| Theme effects can obscure statuses | Preserve text labels, focus and semantic state | Keyboard, contrast and selected/disabled/error states |
| New source assets could be omitted from RPM or WebView | Explicit design-engine JS/CSS assets in export/staging/manifests | Source-to-build/export/WebView/RPM checksum verification |

## Architecture

`design-engine.js` owns the registry, token resolver, safe persistence and Appearance controls. `design-engine.css` adapts the existing shell and reusable components using centralized variables and selected design attributes. The existing component/render architecture remains in place; introducing a React provider or a new component framework would add unnecessary replacement work. Global preferences apply to all existing routes. Page-level overrides are appropriate only when they preserve predictable global state.

Token coverage includes colors/surfaces, typography, borders/radii, elevation, spacing, gradients/blur/opacity, motion/easing, focus and component states. The eight design languages are Skeuomorphism, Minimalism, Glassmorphism, Neumorphism, Claymorphism, Bento Grid, Retro Y2K and Liquid Glass. The registry must expose valid complete designs, not labels disconnected from actual application styling.

## Original implementation and data boundary

The design extension is repository-native JavaScript/CSS built on existing components and dependencies. No external theme implementation, private corpus or new parser dependency is imported. Appearance settings contain only style/mode/accent preferences. They do not store credentials, alter roles, deploy parsers, change backend APIs or invent dashboard statistics. NLI metadata remains under its existing protected RPCv1 protocol; updating the bootstrap release label is not a storage schema migration.

## Limits and acceptance

Design transitions must not reload routes or erase form state. Unsupported backdrop effects need opaque readable fallbacks. Respect reduced motion and preserve explicit focus. Browser checks establish the tested matrix rather than every browser/device. Full WCAG 2.2 AA certification, physically accurate optical refraction and universal performance claims are not established by source inspection. Final test counts, contrast results and packaged acceptance belong in `V21_R8_TEST_RESULTS.md` after execution.


## Separate accounting failure audit

The visual design engine does not change parser or transport semantics. Two independently reproduced output defects are a separate surgical runtime correction: local append before required-remote failure could be repeated when the uncommitted cursor retried, and a legacy quarantine path could write both raw record and routed envelope to the same quarantine file. Source-generation/logical-ordinal durable receipts protect local replay without content-based deduplication; canonical quarantine uses its raw queue once. Existing customer JSON remains unchanged. This is a scoped local guarantee with durable state and an exclusive writer, while remote transports remain at-least-once.

Observed byte-size differences are not proof of duplicate logical records. The count-only operator helper compares a stable before/after test interval, preserves historical output and explicitly excludes unrelated journal, replay and fanout copies. Runtime regression names and final results belong in `V21_R8_TEST_RESULTS.md` after execution.


### Implemented design engine inventory

The implementation exports an eight-entry registry, preference normalizer, token resolver, contrast helper, design provider and optional registration hook. The provider updates root CSS variables/data attributes without accessing application APIs or rebuilding business-route DOM. The CSS adapter covers the existing semantic shell/cards/forms/tables/status/chart/log components; blur is limited to smaller topbar/popover surfaces. Reduced-motion, forced-colors and unsupported-blur fallbacks preserve usability. The Appearance dialog's component/chart examples are explicitly illustrative, not operational telemetry.

Preference data is `{version, style, mode, accent}` only. Corrupt/unavailable browser storage falls back safely; the original R7 dark appearance remains recoverable. Registry/unit tests cover eight styles, two modes and five accents against text/surface contrast targets. Browser/responsive/route results are recorded after their actual tests, rather than inferred from those token calculations.


## Additive parser availability and one output per source occurrence

R8 keeps embedded baseline profiles and approved managed parser packs in one bounded eligible candidate pool. Approval, service selectors, runtime compatibility and reviewed scope guards still determine eligibility. A successful embedded parser does not short-circuit the remaining approved matching pack attempts. Embedded profiles execute first to protect baseline availability under resource limits; failed managed candidates are isolated and cannot by themselves suppress a valid generic JSON/text fallback.

Selection produces one governed customer output after the eligible attempts. Reviewed native mapping scope/priority can intentionally select a custom schema; an ordinary regex match does not silently shadow structurally valid JSON. Original approved parsers remain available while new parsers are added or repaired. Multiple parser attempts are diagnostic activity, not multiple input records or permission to append a customer output for every matching parser. Record accounting compares source occurrences with canonical outputs separately from parser-attempt counters.


## Automatic discovery uses the current approved pool

Automatic Assurance and Parser Creation Helper requests now read the current published/active canonical parser-pack metadata and use an immutable compiled runtime cache. Drafts and unapproved revisions stay excluded; rollback removes an add-on from the next automatic snapshot without a controller restart. Each request receives its own engine-dependency copy, preserving shared baseline dependencies and explicit manual parser selection.

The analysis-only reader tolerates identical canonical aliases and excludes conflicting runtime identities, invalid packs or attempts to replace protected embedded IDs while keeping unrelated valid add-ons and built-ins available. Canonical fleet publication and atomic bundle synchronization remain strict. Its `pool_hash` identifies an analysis snapshot; it is not a desired/applied fleet bundle hash or a client acknowledgement.

**Suggest from raw** calls the controller with the exact sample and selected service. Structural proposals remain separate from read-only runtime evidence: embedded/published pool IDs, attempted parsers, selected matches and bounded observed fields. Runtime naming proposals are offered only when a source-backed unique primitive value and explicit byte span reproduce the approved parser's value exactly. They do not overwrite operator names, labels or transformation chains. Raw edits invalidate older evidence; canceled/stale responses are ignored. An unavailable controller yields an explicitly unverified local structural fallback, not fabricated runtime proof. No suggestion call persists metadata, creates a parser or changes approval/deployment state.

Runtime matching evidence is limited to 16 samples and about 64 KiB while structural corpus analysis retains its existing 100-sample bound. Existing admission deadlines and cancellation cover contextual pack normalization, approved regression compilation and execution. To preserve source byte/character positions, R8 UI Assurance/helper calls send `preserve_source: true`; this retains CRLF and trailing spaces in the parsed raw/hash. Legacy API clients default to their prior canonicalization unless they opt in. Exact-source HTTP input remains bounded to 1 MiB.
