#!/usr/bin/env python3
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import tempfile
import unittest


class EvidenceGeneratorTest(unittest.TestCase):
    def test_deterministic_valid_evidence(self) -> None:
        root = pathlib.Path(__file__).resolve().parents[2]
        generator = root / "scripts" / "release" / "generate-evidence.py"
        with tempfile.TemporaryDirectory() as directory:
            artifacts = pathlib.Path(directory)
            names = (
                "nli_client_ver22.rpm",
                "nli_controller_agent_ver22.rpm",
                "webview_ver22.zip",
                "webapp_ver22.zip",
                "install-controller-ver22.sh",
                "install-client-ver22.sh",
                "install-controller-aws-ver22.sh",
                "install-client-aws-ver22.sh",
                "install-controller-aws-cloudflare-ver22.sh",
                "aws-rhel-preflight-ver22.sh",
                "aws-security-group-plan-ver22.sh",
                "setup-cloudflare-tunnel-v22.sh",
                "verify-cloudflare-tunnel-v22.sh",
                "uninstall-cloudflare-tunnel-v22.sh",
                "cloudflare-quick-diagnosis-v22.sh",
                "vercel-preflight.mjs",
                "uninstall-controller-ver22.sh",
                "uninstall-client-ver22.sh",
                "INSTALLATION_AND_CONFIGURATION.md",
                "VERCEL_DEPLOYMENT.md",
                "CLOUDFLARE_PAGES_DEPLOYMENT.md",
                "WEBAPP_OPERATOR_AND_PARSER_GUIDE.md",
                "PARSER_CATALOG.md",
                "V22_6_RELEASE_NOTES.md",
                "All_NLI_features.md",
                "V21_R1_RELEASE_NOTES.md",
                "V21_R1_TEST_RESULTS.md",
                "V21_R2_RELEASE_NOTES.md",
                "V21_R2_TEST_RESULTS.md",
                "V21_R3_RELEASE_NOTES.md",
                "V21_R3_TEST_RESULTS.md",
                "V21_R4_RELEASE_NOTES.md",
                "V21_R4_TEST_RESULTS.md",
                "V21_R4_SURGICAL_DIAGNOSTICS_AND_VERIFICATION.md",
                "V21_R5_RELEASE_NOTES.md",
                "V21_R5_TEST_RESULTS.md",
                "V21_R6_RELEASE_NOTES.md",
                "V21_R6_TEST_RESULTS.md",
                "V21_R7_RELEASE_NOTES.md",
                "V21_R7_TEST_RESULTS.md",
                "V21_R7_MAPPING_DESIGN.md",
                "V21_R7_FIELD_MAPPING_GUIDE.md", "V21_R8_RELEASE_NOTES.md", "V21_R8_TEST_RESULTS.md", "V21_R8_DESIGN_ENGINE_DESIGN.md", "V21_R8_DESIGN_ENGINE_GUIDE.md", "V21_R8_OUTPUT_ACCOUNTING_GUIDE.md", "V21_R8_AI_PROVIDER_GUIDE.md", "V21_R9_RELEASE_NOTES.md", "V21_R9_TEST_RESULTS.md", "V21_R9_FIELD_MAPPING_GUIDE.md", "V21_R10_RELEASE_NOTES.md", "V21_R10_TEST_RESULTS.md", "V21_R10_MANAGER_DEMO.md", "nli-output-audit.py",
                "VER21_IMPLEMENTATION_PLAN.md",
                "BENCHMARKS.txt",
                "SAMPLE_FIELD_ACCOUNTING.json",
                "RELEASE_ID",
            )
            for index, name in enumerate(names):
                if name == "BENCHMARKS.txt":
                    content = b"BenchmarkFixture-8  5  100 ns/op\n"
                elif name == "SAMPLE_FIELD_ACCOUNTING.json":
                    content = b'{"verdict":"pass"}\n'
                elif name == "RELEASE_ID":
                    content = b"NLI_VERSION=21.0.0\nNLI_RELEASE=7\nNLI_ARCH=x86_64\nSOURCE_REVISION=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"
                else:
                    content = f"fixture-{index}".encode()
                (artifacts / name).write_bytes(content)
            (artifacts / "SOURCE_BINARY_PROVENANCE.txt").write_text(
                "\tmod\tnexora.local/nli-v20\t(devel)\n"
                "\tdep\texample.invalid/module\tv1.2.3\th1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n",
                encoding="utf-8",
            )
            images = artifacts / "images.env"
            images.write_text("GO_IMAGE=example.invalid/go@sha256:abc\n", encoding="utf-8")
            command = [
                sys.executable,
                str(generator),
                "--artifacts", str(artifacts),
                "--version", "21.0.0",
                "--release", "7",
                "--revision", "a" * 40,
                "--source-date-epoch", "1791200000",
                "--images", str(images),
            ]
            subprocess.run(command, check=True)
            evidence_names = ("SHA256SUMS", "RELEASE_MANIFEST.json", "SBOM.spdx.json", "provenance.intoto.json")
            first = {name: (artifacts / name).read_bytes() for name in evidence_names}
            subprocess.run(command, check=True)
            self.assertEqual(first, {name: (artifacts / name).read_bytes() for name in evidence_names})

            checksums = (artifacts / "SHA256SUMS").read_text(encoding="ascii").splitlines()
            self.assertEqual(len(checksums), len(names))
            for line, name in zip(checksums, names):
                expected = hashlib.sha256((artifacts / name).read_bytes()).hexdigest()
                self.assertEqual(line, f"{expected}  {name}")

            sbom = json.loads((artifacts / "SBOM.spdx.json").read_text(encoding="utf-8"))
            ids = {package["SPDXID"] for package in sbom["packages"]} | {"SPDXRef-DOCUMENT"}
            for relationship in sbom["relationships"]:
                self.assertIn(relationship["spdxElementId"], ids)
                self.assertIn(relationship["relatedSpdxElement"], ids)
            for package in sbom["packages"]:
                for checksum in package.get("checksums", []):
                    self.assertRegex(checksum["checksumValue"], re.compile(r"^[0-9a-f]{64}$"))

            provenance = json.loads((artifacts / "provenance.intoto.json").read_text(encoding="utf-8"))
            self.assertEqual(provenance["_type"], "https://in-toto.io/Statement/v1")
            self.assertEqual(len(provenance["subject"]), len(names))


if __name__ == "__main__":
    unittest.main()
