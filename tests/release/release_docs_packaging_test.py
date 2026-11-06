#!/usr/bin/env python3
"""Verify that the current Ver21 R6 release documents ship everywhere."""

from __future__ import annotations

import argparse
from pathlib import Path


DOCUMENTS = {
    "All_NLI_features.md": "All_NLI_features.md",
    "V21_R1_RELEASE_NOTES.md": "docs/V21_R1_RELEASE_NOTES.md",
    "V21_R1_TEST_RESULTS.md": "docs/V21_R1_TEST_RESULTS.md",
    "V21_R2_RELEASE_NOTES.md": "docs/V21_R2_RELEASE_NOTES.md",
    "V21_R2_TEST_RESULTS.md": "docs/V21_R2_TEST_RESULTS.md",
    "V21_R3_RELEASE_NOTES.md": "docs/V21_R3_RELEASE_NOTES.md",
    "V21_R3_TEST_RESULTS.md": "docs/V21_R3_TEST_RESULTS.md",
    "V21_R4_RELEASE_NOTES.md": "docs/V21_R4_RELEASE_NOTES.md",
    "V21_R4_TEST_RESULTS.md": "docs/V21_R4_TEST_RESULTS.md",
    "V21_R4_SURGICAL_DIAGNOSTICS_AND_VERIFICATION.md": "docs/V21_R4_SURGICAL_DIAGNOSTICS_AND_VERIFICATION.md",
    "V21_R5_RELEASE_NOTES.md": "docs/V21_R5_RELEASE_NOTES.md",
    "V21_R5_TEST_RESULTS.md": "docs/V21_R5_TEST_RESULTS.md",
    "V21_R6_RELEASE_NOTES.md": "docs/V21_R6_RELEASE_NOTES.md",
    "V21_R6_TEST_RESULTS.md": "docs/V21_R6_TEST_RESULTS.md",
    "VER21_IMPLEMENTATION_PLAN.md": "docs/VER21_IMPLEMENTATION_PLAN.md",
}


def fail(message: str) -> None:
    raise SystemExit(f"Ver21 R6 release-doc packaging: FAIL: {message}")


def validate(root: Path, output: Path, client: Path, controller: Path) -> None:
    sources = {name: root / relative for name, relative in DOCUMENTS.items()}
    for name, source in sources.items():
        if not source.is_file() or source.stat().st_size == 0:
            fail(f"source artifact is missing or empty: {source}")
        for target_dir in (output, client, controller):
            target = target_dir / name
            if not target.is_file() or target.read_bytes() != source.read_bytes():
                fail(f"{name} is missing or differs in {target_dir}")

    expected_claims = {
        "V21_R6_RELEASE_NOTES.md": ("21.0.0-6", "v21.0.0-5", "Administrator", "quarantine", "Steps 7–9"),
        "V21_R6_TEST_RESULTS.md": ("21.0.0-6", "v21.0.0-5", "SOURCE_REVISION", "SHA256SUMS"),
        "V21_R5_RELEASE_NOTES.md": (
            "21.0.0-5", "v21.0.0-4", "Steps 3–4", "128", "R4",
        ),
        "V21_R5_TEST_RESULTS.md": (
            "21.0.0-5", "v21.0.0-4", "28 checks", "74 scenarios", "3 scenarios",
            "Vite production build", "go test ./...", "PASS",
        ),
        "V21_R4_RELEASE_NOTES.md": (
            "21.0.0-4", "v21.0.0-3", "nli_client_ver21.rpm", "nli_controller_agent_ver21.rpm",
            "nli-v21.0.0-4-x86_64.tar.gz", "nexora.local/nli-v20", "capacity",
        ),
        "V21_R4_TEST_RESULTS.md": (
            "21.0.0-4", "v21.0.0-3", "be028bb", "28/28", "73/73", "Go `1.24.9`",
            "nli_client_ver21.rpm", "nli_controller_agent_ver21.rpm",
            "nli-v21.0.0-4-x86_64.tar.gz", "SHA256SUMS", "exact-source",
            "reproducibility", "PENDING FINAL GATE", "No production capacity",
        ),
        "VER21_IMPLEMENTATION_PLAN.md": (
            "Ver21 implementation plan", "21.0.0-6", "Verification and release gates",
        ),
    }
    for name, claims in expected_claims.items():
        content = sources[name].read_text(encoding="utf-8")
        for claim in claims:
            if claim not in content:
                fail(f"{name} does not document expected release evidence: {claim!r}")

    packaging_files = (
        "packaging/build-rpms.sh",
        "packaging/rpm/nli-client.spec",
        "packaging/rpm/nli-controller-agent.spec",
        "packaging/build-release.ps1",
        "scripts/release/build.sh",
        "scripts/release/generate-evidence.py",
        "scripts/release/verify-artifacts.sh",
        "tests/release/package-infra.sh",
    )
    for relative in packaging_files:
        content = (root / relative).read_text(encoding="utf-8")
        for name in sources:
            if name not in content:
                fail(f"{relative} does not reference packaged artifact {name}")

    workflow = (root / ".github/workflows/release.yml").read_text(encoding="utf-8")
    for claim in ("NLI V21 R6 release", "nli-v21-r6-unsigned-release", "docs/V21_R6_RELEASE_NOTES.md", "v21.0.0-5"):
        if claim not in workflow:
            fail(f"release workflow does not identify the R6 workflow/R5 baseline: {claim!r}")

    release_env = (root / "packaging/release.env").read_text(encoding="utf-8")
    if "NLI_VERSION=21.0.0" not in release_env or "NLI_RELEASE=6" not in release_env:
        fail("packaging release identity is not 21.0.0-6")

    for relative in (
        "README.md", "CHANGELOG.md", "docs/V21_R6_RELEASE_NOTES.md",
        "packaging/rpm/nli-client.spec", "packaging/rpm/nli-controller-agent.spec",
    ):
        if "21.0.0-6" not in (root / relative).read_text(encoding="utf-8"):
            fail(f"{relative} does not identify the current release")

    # Historical source evidence remains available for compatibility and audit.
    for relative in (
        "docs/V21_R4_RELEASE_NOTES.md", "docs/V21_R4_TEST_RESULTS.md",
        "docs/V21_R4_SURGICAL_DIAGNOSTICS_AND_VERIFICATION.md",
        "docs/V20_R4_TEST_RESULTS.md", "docs/V20_R6_TEST_RESULTS.md",
    ):
        if not (root / relative).is_file():
            fail(f"historical release evidence is missing: {relative}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--client", required=True, type=Path)
    parser.add_argument("--controller", required=True, type=Path)
    args = parser.parse_args()
    validate(args.root.resolve(), args.output.resolve(), args.client.resolve(), args.controller.resolve())
    print("Ver21 R6 release-doc packaging: PASS")


if __name__ == "__main__":
    main()
