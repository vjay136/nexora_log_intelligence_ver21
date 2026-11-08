# Nexora Log Intelligence Ver22.2

Release `22.2.0-0` upgrades Ver22 `22.1.0-0`. Extract `nli-v22.2.0-0-x86_64.tar.gz` and use the Ver22 installers. The archive includes the client and controller RPMs, the WebView and webapp ZIPs, and checksum files.

## Replace and split key/value input

In the mapping editor, add or edit a **Key value EOL** row. Set **Start position** to the token where the pairs begin. Open **Replace and split KV input** to edit the selected text before the controller extracts keys and values.

For the mail sample below, the adjacent bracketed pairs start at token `9`.

```text
20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg [status:failed][reason;notneeded][server=cloud]
```

1. Set **Start position** to `9`.
2. Select **Set up adjacent [key:value] pairs**. The preset removes the outer brackets, separates adjacent pairs, and accepts `:`, `;`, and `=` as assignment operators.
3. Select **Preview**. With **Output** set to **flatten**, the preview should show `status` as `failed`, `reason` as `notneeded`, and `server` as `cloud` alongside the header mappings.

For a custom format, enter the exact characters to find and replace. Leave the replacement empty to remove a match. For example, you can replace `#` with `|` in `status=failed#reason=timeout` and select `|` as the pair separator. If `#` already separates the pairs, you can also set it directly as the separator and skip the replacement. Use **Assignment operators** for characters between a key and its value. Use **Pair separators** for characters between one pair and the next. **Remove from start** and **Remove from end** target the edges of the selected text.

The literal controls take text directly. Enter one backslash to match a backslash, one space to match a space, or `[]` to match that exact two-character sequence. Braces, parentheses, quotes, asterisks, and other punctuation work the same way.

For a regular field mapping, use **Replace text** or **Split value**. First select the source with the extraction method, then apply the shortcut to the selected value. For example, split `smtp#cloud` on `#` and select part `2` to produce `cloud`. Use a separate mapping row for each named output.

Preview representative input before publishing a mapping. The same input preparation runs in the parser bundle, so clients must support Ver22.2 before receiving bundles that use it.

**Suggest from raw log** and **Inspect KV grammar** recognize complete adjacent square, round, curly, and angle wrappers, including mixed wrappers. They propose the literal edits and show the extracted pairs. Characters within values stay in the data unless an operator adds a replacement rule. Use separate samples to verify variations before publication.

FieldSense shows the purpose of each mapping input next to the control. The operation editor shows relevant inputs for common operations, with examples for literal replacement and splitting. Shared workspace forms offer expandable input help.

See [Installation and configuration](INSTALLATION_AND_CONFIGURATION.md) for upgrade steps.
