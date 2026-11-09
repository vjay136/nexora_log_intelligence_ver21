# Nexora Log Intelligence Ver22.3

Release `22.3.0-0` upgrades Ver22 `22.2.0-0`. Extract `nli-v22.3.0-0-x86_64.tar.gz` and use the Ver22 installers. The archive includes the client and controller RPMs, the WebView and webapp ZIPs, and checksum files.

## Parser coverage

Reviewed mail parsers recognize key/value payloads enclosed in an outer pair of square, curly, round, or angle brackets. The reported SMTP-OUT event `SmtpServerEHLO(2001/4)` now captures `id`, `domain`, and `host` from its braced payload. A candidate must account for the structured values at their original positions before it can normalize the record. If every eligible candidate fails coverage, the original record goes to quarantine with `incomplete_structured_tail`.

Verify new parser mappings against representative records from each source format before publishing. FieldSense preview shows the normalized values and their source evidence so you can check both before rolling out a parser.

## Hourly file output and replay

At startup and each UTC hour boundary, the agent prepares the configured local hourly files. The default file output creates a matching `*_normalised_YYYYMMDD_HH00.jsonl` and `*_quarantine_YYYYMMDD_HH00.jsonl` pair for each configured family, including families with no records that hour. Idle files are empty and the agent preserves any existing rows. Static filenames, remote sinks, and configurations that disable local normalized output keep their existing behavior.

For input accounting, stop adding test input and let the pipeline drain. Compare the normalized and quarantine file deltas for a controlled interval before starting replay. Replay output can add rows when it runs, so count those rows separately from original input.

The dashboard's historical **Quarantined** count records original routing. **Quarantine and replay** reports the current state of retained records as **Awaiting replay**, **Reprocessing**, **Reprocessed successfully**, or **Manual attention**. Selecting a client scopes both views. Changing the historical time range affects ingestion and routing history. A successfully reprocessed record remains part of its original routing history, and only records still retained by the controller contribute to the replay panel.

Use [Installation and configuration](INSTALLATION_AND_CONFIGURATION.md) for upgrade steps and the [output accounting guide](V21_R8_OUTPUT_ACCOUNTING_GUIDE.md) for controlled file-count checks.
