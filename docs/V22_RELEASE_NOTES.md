# Nexora Log Intelligence Ver22

Release 22.0.0-2 upgrades the existing controller and client RPMs. The installer and WebView filenames keep their compatibility names. Use the controller, clients and WebApp from the same release bundle. The upgrade and rollback baseline is Ver21 21.0.0-10.1.1.

## Quarantine replay

Step 8 uses the controller's explicit acknowledgement state, including inventory drift, as well as the exact hash. Matching hashes alone can appear while the inventory is pending or drifted. Replay rejection now preserves the controller's diagnostic details. Step 9 can show the retained record status and replay history before successful replay. The wizard only marks the record resolved when the controller reports resolution.

## Fleet operations

Generate a set of expiring, per-host enrollment grants with `sudo nli-controller-setup --create-enrollment-batch 100`. Deploy one protected bootstrap file per node. A retry on the same host uses the same enrolled identity. See [Fleet enrollment](V22_FLEET_ENROLLMENT.md) for installation and reissue.

Input rollouts affect the selected clients, or all eligible online clients when selected. Save, publish and approve the input profile through the existing review flow. Inspect each node's result before assuming it received the new configuration. Disabling retains the path in the profile for later reactivation. Removing a path from the collection configuration leaves the source file with its owning application and retention policy.

## Parser selection

ParserPack rule priority is sorted when the pack is compiled. Execution filters that immutable order for each record, keeping the existing priority and rule ID tie-break. Production throughput depends on the selected parser, log shape, hardware and output sink. The change does not establish a one-million-records-per-second capacity rating.

## Nested key value mapping

For deeply nested values, choose a dynamic KV tail, set its source range, and enable Kv Nested Objects under Advanced FieldSense parameters. For example, use AutoSense at token 1 on:

    status:ok,details:{phase:ready,env:{stage=prod;inner=[zone=west id=0042]}},note:(plain text)

The nested object keeps the identifier 0042 as a string. Ordinary bracketed text stays text. The parser bounds nesting and field count. The native preview reports source evidence coverage on the current sample separately from expected-output validation. A 100 percent evidence score covers the output fields in that sample.

The existing single-threaded production SMTP adapter benchmark ran at approximately 651 microseconds per record on a Ryzen 5 5600H Linux container with telemetry enabled and no external output sink. It measures that workload, rather than fleet capacity.
