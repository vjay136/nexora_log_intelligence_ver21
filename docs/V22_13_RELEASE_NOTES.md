# NLI v22.13.0-0

## Structural Field Integrity & Parser Assurance

An older native mapping could split `123456;Note;SieveRuleMsg` as if it had two components. It assigned `123456` to `loglevel`, assigned `Note` to `event`, and dropped `SieveRuleMsg`. The old completeness check began after that token, so successful KV extraction could leave the record eligible for normal output.

The recognized numeric-prefix mail header must account for each component before the result is eligible for selection. The built-in parser yields `random_number: "123456"`, `loglevel: "Note"`, `event: "SieveRuleMsg"`, `status: "failed"`, and `reason: "not needed"`, in addition to the other header fields. A reviewed template can name the level field `event_type`. Byte ranges use zero-based starts and exclusive ends. Semicolons and assignment characters are grammar boundaries where the selected grammar defines them.

## Reusable templates

Use **Log Pattern Templates** to build or verify a header template with pasted records. Use **KV Pair Templates** to build or verify a grammar for the payload after the header. Each verification accepts up to 500 records per request. Review the JSON and preview output before saving the drafts. Compose the approved revisions and publish the resulting parser pack through the existing review and fleet rollout workflow. A complete reviewed composition takes precedence over legacy candidates. Deploy the matching controller and client RPMs so the same selection rules are used in Assurance and live ingestion.
