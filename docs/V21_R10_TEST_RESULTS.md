# Ver21 R10 qualification record

Release `21.0.0-10` is based on `v21.0.0-9` at `6fb870067631d27fbf8d2f178b393790e8d3df01`. The publication workflow binds qualification to the released revision through `SOURCE_REVISION`, the release manifest, and `SHA256SUMS`.

| Check | Evidence to review |
| --- | --- |
| Native parser correctness | Go tests cover supported KV grammars, malformed evidence, existing parser families, and parser definition compatibility. |
| Concurrency and isolation | Go race checks exercise the shared parser and control-plane test suite. |
| Browser workflows | The Web UI suite exercises navigation and the mapping, preview, validation, and approval surfaces. |
| Packaging | The release fixture and artifact verifier check repeatability, genuine RPM identity, source parity, document inclusion, and digests. |
| Platform and fleet behavior | Rocky Linux 9 and 10.2 gates use genuine R9 RPMs for upgrade and rollback. The two-node acceptance flow tests packaged client enrollment, controller communication, private TLS, and the packaged WebView. |

See the workflow run attached to the final tagged release for the results, test counts, and published artifact identifiers. Accuracy claims apply to cases with expected outputs. Target-host throughput and recovery times depend on record shape, input rate, storage, network, and destination settings.
