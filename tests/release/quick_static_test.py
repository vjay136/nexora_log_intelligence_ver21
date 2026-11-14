#!/usr/bin/env python3
"""Fast, platform-neutral Ver22 release invariants for the cached quick suite."""

from __future__ import annotations

import json
from pathlib import Path
import re
import sys


def fail(message: str) -> None:
    raise SystemExit(message)


root = Path(sys.argv[1] if len(sys.argv) > 1 else Path(__file__).parents[2]).resolve()

required = (
    "scripts/install-controller-ver22.sh",
    "scripts/install-client-ver22.sh",
    "scripts/install-controller-aws-ver22.sh",
    "scripts/install-client-aws-ver22.sh",
    "scripts/install-controller-aws-cloudflare-ver22.sh",
    "scripts/setup-cloudflare-tunnel-v22.sh",
    "scripts/verify-cloudflare-tunnel-v22.sh",
    "scripts/uninstall-cloudflare-tunnel-v22.sh",
    "scripts/cloudflare-quick-diagnosis-v22.sh",
    "scripts/vercel-preflight.mjs",
    "wrangler.jsonc",
    "scripts/uninstall-controller-ver22.sh",
    "scripts/uninstall-client-ver22.sh",
    "scripts/release/create-bundle.sh",
    "configs/parser-catalog/nli-v20-parser-catalog.json",
    "configs/catalog/nli-v20-parser-catalog.json",
    "testdata/migration/sample-logs/manifest.json",
    "docs/V20_AWS_RED_HAT_INSTALLATION.md",
    "docs/V20_CLOUDFLARE_TUNNEL.md",
    "docs/release/V20_MIGRATION_AND_ROLLBACK.md",
    "docs/VERCEL_DEPLOYMENT.md",
    "docs/CLOUDFLARE_PAGES_DEPLOYMENT.md",
    "docs/TESTING.md",
    "configs/catalog/catalog_test.go",
    "configs/parsers/assets_test.go",
    "cmd/nli-bench/main_test.go",
    "cmd/nli-manager/main_test.go",
    "cmd/nli-migrate/main_test.go",
    "cmd/nli-throughput/main_test.go",
    "internal/confidence/confidence_test.go",
    "internal/manager/manager_test.go",
    "internal/registry/registry_test.go",
    "internal/typeinfer/typeinfer_test.go",
)
for name in required:
    if not (root / name).is_file():
        fail(f"missing Ver22 release input: {name}")

scan_roots = ("README.md", "docs", "packaging", "scripts", "web", ".github")
files: list[Path] = []
for name in scan_roots:
    path = root / name
    if path.is_file():
        files.append(path)
        continue
    for candidate in path.rglob("*"):
        if any(part in {"node_modules", "build", "dist", "__pycache__", ".playwright"} for part in candidate.parts):
            continue
        if candidate.is_file():
            files.append(candidate)

wrong_product = "Nexora Log Inetelligence V20"
publishable_key_pattern = re.compile(r"sb_publishable_[A-Za-z0-9_-]{20,}")
live_project_pattern = re.compile(r"https://[a-z0-9]{20}\.supabase\.co", re.IGNORECASE)
allowed_history = {
    root / "docs/NLI_V5_IMPLEMENTATION_SPEC.md",
    root / "CHANGELOG.md",
    root / "README.md",
    root / ".github/workflows/release.yml",
    root / "docs/V20_ACCEPTANCE_CHECKLIST.md",
    root / "docs/V20_ENGINEERING_AUDIT.md",
    root / "docs/V20_SUPPORT_MATRIX.md",
    root / "docs/V20_MASTER_SPEC_IMPLEMENTATION_MATRIX.md",
    root / "docs/release/README.md",
}
# Versioned baseline documents remain immutable evidence for upgrade and
# regression comparisons. They may legitimately name older releases, but are
# still scanned by the checks below for credentials and malformed addresses.
historical_release_docs = {
    path for path in (root / "docs").glob("*V18*") if path.is_file()
}
stale_release_terms = ("v" + "15", "ver" + "15", "15." + "0.0", "webview_ver" + "15", "webapp_ver" + "15", "nli-v" + "15")
stale_release_pattern = re.compile(r"(?i)(?:" + "|".join(re.escape(term) for term in stale_release_terms) + r")")
for path in files:
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue
    if "1210.0.0.1" in text:
        fail(f"invalid loopback address remains in {path.relative_to(root)}")
    if wrong_product in text:
        fail(f"incorrect product name remains in {path.relative_to(root)}")
    if path not in allowed_history and path not in historical_release_docs and stale_release_pattern.search(text):
        fail(f"stale release identity remains in {path.relative_to(root)}")
    if publishable_key_pattern.search(text):
        fail(f"live publishable key was committed in {path.relative_to(root)}")
    if live_project_pattern.search(text):
        fail(f"live Supabase project URL was committed in {path.relative_to(root)}")

catalog_a = (root / "configs/parser-catalog/nli-v20-parser-catalog.json").read_bytes()
catalog_b = (root / "configs/catalog/nli-v20-parser-catalog.json").read_bytes()
if catalog_a != catalog_b:
    fail("canonical and embedded V20 parser catalogs differ")

manifest = json.loads((root / "testdata/migration/sample-logs/manifest.json").read_text(encoding="utf-8"))
if manifest.get("sample_log_record_count") != 728:
    fail(
        "sample manifest expected 728 records, got "
        f"{manifest.get('sample_log_record_count')!r}"
    )

workflow = (root / ".github/workflows/release.yml").read_text(encoding="utf-8")
for marker in (
    "npm run test:ui",
    "npm run build",
    "gh release view",
    "Expected genuine client and controller RPMs",
    "REQUIRE_PREVIOUS=1",
):
    if marker not in workflow:
        fail(f"release workflow is missing mandatory gate {marker!r}")
if not re.search(r"go test(?: -timeout=\d+[smh])? -race \./\.\.\.", workflow):
    fail("release workflow is missing the unfiltered all-package Linux race gate")
if "REQUIRE_PREVIOUS=0" in workflow:
    fail("release workflow may silently skip genuine previous-RPM acceptance")

rocky_validation = (root / "tests/rocky/validate.sh").read_text(encoding="utf-8")
for marker in (
    "previous client must be the genuine Ver22 22.15.1-0 RPM",
    "previous controller must be the genuine Ver22 22.15.1-0 RPM",
    "$previous_client_release =~ ^0([.]el[0-9]+)?$ && $current_client_release =~ ^0([.]el[0-9]+)?$",
    "$previous_controller_release =~ ^0([.]el[0-9]+)?$ && $current_controller_release =~ ^0([.]el[0-9]+)?$",
):
    if marker not in rocky_validation:
        fail(f"Rocky lifecycle validation is missing previous-release gate {marker!r}")
if "PREVIOUS_RELEASE_TAG: ${{ inputs.previous_release_tag || 'v22.15.1-0' }}" not in workflow:
    fail("Ver22 lifecycle validation does not use the published Ver22 22.15.1-0 baseline")
if "name: nli-v22-unsigned-release" not in workflow or "--notes-file docs/V22_16_1_RELEASE_NOTES.md" not in workflow:
    fail("Ver22 lifecycle workflow does not publish Ver22.5 artifacts and release notes")
release_env = (root / "packaging/release.env").read_text(encoding="utf-8")
if "NLI_VERSION=22.16.1" not in release_env or "NLI_RELEASE=0" not in release_env.splitlines():
    fail("Ver22 release identity is not 22.16.1-0")
if "PREVIOUS_RELEASE_REPO: ${{ github.repository }}" not in workflow:
    fail("Ver22 lifecycle validation does not use the current published repository")

research_ledger = (root / "docs/V20_RESEARCH_AND_LICENSE_LEDGER.md").read_text(encoding="utf-8")
if "| **Total** | **100** |" not in research_ledger:
    fail("research ledger does not prove the required exact total of 100 reviews")
for batch in range(1, 8):
    name = f"docs/research/V20_RESEARCH_GAP_D{batch}.md"
    if not (root / name).is_file() or f"`research/V20_RESEARCH_GAP_D{batch}.md`" not in research_ledger:
        fail(f"research evidence batch is missing from the consolidated ledger: {name}")

for installer_name in required[:2]:
    installer = (root / installer_name).read_text(encoding="utf-8")
    if "rpm_release == 1" in installer or "not release 1" in installer:
        fail(f"hard-coded RPM release validation remains in {installer_name}")
    for marker in ("validate_rpm_release", "^[0-9]+([.][0-9]+)*$", "--verify-only"):
        if marker not in installer:
            fail(f"{installer_name} is missing release validation marker {marker!r}")
    for marker in ("RELEASE_ID is missing", "release bundle checksum verification failed", "does not match this bundle release"):
        if marker not in installer:
            fail(f"{installer_name} is missing mixed-release protection {marker!r}")

print("Ver22 quick static release invariants: PASS")
