# Ver22 fleet enrollment

Run the regular controller setup first so it writes the verified HTTPS URL and any private CA into the controller bootstrap file. From the controller, run as root:

```sh
sudo nli-controller-setup --create-enrollment-batch 100
```

The command prints the batch directory under `/var/lib/nli-controller/enrollment/bundles/` and the expiry. Each run accepts 1 to 2,000 files. The default lifetime is 168 hours. You can pass a third argument, such as `--create-enrollment-batch 100 48`, to set a lifetime of 1 to 720 hours.

Distribute exactly one `client-NNNN.env` per intended node through a secure inventory or configuration management channel. Preserve the root-only file permissions. Record which file went to each host. The controller retains root-only export copies and reads grant fingerprints from the adjacent `grants` directory.

After verifying and installing the Ver22 client RPM, run on the client:

```sh
sudo nli-client-setup /root/client-0001.env
```

Or use the release installer's `--bootstrap /root/client-0001.env` option. Setup normally removes its local bootstrap copy after success. Retain the protected controller copy to recover an interrupted setup or reinstall on the same host. Each grant binds to one NLI identity and a separate host proof on first valid use. The two are derived from the host's unique Linux machine ID, which stays local. Ensure cloned images initialize a unique machine ID before deployment. Older enrolled clients retain their original identities and use their existing enrollment method.

An interrupted client can retry with the same unexpired bundle. If the slot expired or the file was lost, recover the original host using its exact `nli-fleet-…` ID:

```sh
sudo nli-controller-setup --reissue-enrollment nli-fleet-REPLACE_WITH_FULL_ID
```

Distribute the generated replacement only to the original host. If a local node token already exists and you intend to rotate it, run `sudo env NLI_REENROLL=1 nli-client-setup /root/client-0001.env` with the new file. A valid replacement supersedes the prior grant. Reissue requires the original machine identity, so a rebuilt machine that lost its OS identity needs a new slot. Keep `fleet-enrollment-claims.json`, the grants, and the protected exports in controller state backups.

The batch approach uses the per-device identity and trusted HTTPS bootstrap principles described in [BRSKI](https://www.rfc-editor.org/info/rfc8995/) and separates initial enrollment and re-enrollment as in [EST](https://www.rfc-editor.org/rfc/rfc7030.html). NLI's own protocol uses per-host bootstrap grants and node tokens. Application-specific identity derivation follows the [systemd machine identity guidance](https://freedesktop.org/software/systemd/man/latest/sd_id128_get_machine.html).
