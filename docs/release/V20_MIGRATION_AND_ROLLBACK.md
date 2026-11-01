# Ver20 migration and rollback note

## Release identity

Ver20 Release 5 is version `20.0.0-5`, Go module `nexora.local/nli-v20`, and WebView root `/usr/share/nli/webview_ver20`. The release archive, RPM aliases, web bundles, installer filenames, and CI release tag use the Ver20 identity. Ver20 Release 4 is the immediate release-to-release upgrade and rollback baseline; Ver19 release 8 remains the supported major-version migration baseline. Release 4's specific behavior is documented in the [release notes](README.md).

## Upgrade path

1. Keep the exact currently installed Ver20 client and controller RPMs available. For a Release 4-to-5 upgrade, retain the genuine Release 4 client and controller RPMs and verified archive. Confirm the Release 5 archive checksums before extraction.
2. Back up `/etc/nli`, `/var/lib/nli`, `/var/lib/nli-control-client`, and `/var/lib/nli-controller` using the site's approved protected backup process. Preserve TLS material, credentials, node enrollment state, parser assets, and audit-chain keys.
3. Use the Release 5 installer and RPMs. Upgrade the controller and managed clients as a coordinated release pair because parser-bundle integrity and client acknowledgement are checked across both components. Do not remove existing packages first; stable package names and service identities allow an in-place upgrade.
4. Verify RPM versions, systemd service health, controller health/API identity, client enrollment, configured input/output paths, and WebView access. Verify representative parsing/output and review quarantine before declaring completion.

## Rollback

If service or compatibility checks fail, stop the affected NLI services, reinstall the saved Ver20 Release 4 client/controller RPMs with the site's normal package manager downgrade procedure, then start services and verify health, enrollment, and data paths. Keep the Release 5 archive and logs for diagnosis. If rolling back across more than one Ver20 release, use the saved RPMs that match the actual pre-upgrade installation and follow the same coordinated procedure.

The RPM downgrade restores executable files but does not automatically reverse configuration or database changes. Before rollback, compare the saved `/etc/nli` backup and database schema/state with the Release 2 baseline. Restore a database backup only when the migration actually changed persistent schema and the approved recovery plan requires it; coordinate client/controller rollback so their protocol and parser revisions remain compatible. Never purge `/var/lib/nli`, secrets, certificates, parser assets, or audit history as part of rollback.

## Evidence boundary

This note describes the release identity and recovery procedure. It is not proof that Release 5 RPM packaging or a Release 4-to-5 lifecycle run passed. Release 5's executed source-level evidence is in [the test results](../V20_R5_TEST_RESULTS.md); reproducible RPMs, payload checks, Rocky lifecycle tests, and published CI results must be recorded after they run from the frozen Release 5 commit. Record exact RPM hashes, source revision, test results, and upgrade/rollback outcomes in release evidence before publishing.
