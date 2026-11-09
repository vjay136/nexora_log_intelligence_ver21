# Nexora Log Intelligence 22.4.0-0

This release builds on 22.3.0-0. The controller and client still execute the same deterministic parser. AI requests run only when an operator invokes them.

- **Ask AI** opens guidance for the current workspace. **Fix with AI** requests a reviewable draft where a supported artifact type is selected. Operators can enter the expected behavior, fields to keep, and edits to make before submitting.
- **Reference based help & Fix** uses a selected catalog definition and a representative sample to draft a new artifact. Review the result, open supported drafts in their editor, or download the draft.
- **Building Blocks** turns selected character ranges, literal values, splits, replacements, and merges into an editable FieldSense plan.
- **Reset Guided Parser Repair** clears the current authoring session. Published parsers and retained quarantine evidence remain available.
- Structured SMTP payload coverage checks reject candidates that silently collapse separate fields into one value. Keep repeated and positional fields in the expected output when testing a representative corpus.

Begin provider validation with the **Test connection** control. Gemini 3.5 Flash-Lite is a starting candidate for this workflow. Measure extraction against your own held-out samples and review any inferred field names before publication.
