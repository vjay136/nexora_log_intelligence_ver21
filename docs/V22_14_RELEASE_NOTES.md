# NLI v22.14.0-0

- Account for the optional fourth component of the numeric-prefix mail event token. For `123456;abcd;Note;SieveRuleMsg`, the component `abcd` is captured as `extra_tag`, with separate evidence from the numeric prefix, level and event.
- Log Pattern Templates, KV Pair Templates and production parser execution recognize the same header boundary. A reviewed header and KV composition emits both sets of fields. A template that leaves a header component unaccounted for fails verification.
- Upgrade the controller and clients to v22.14 before rolling out a composed template that captures `extra_tag`. Regenerate or revise older templates and verify representative tagged and untagged records before approval and publication.
