"""Prevent source-only adaptive design assets from disappearing in release paths."""
from pathlib import Path
import sys
root=Path(sys.argv[1] if len(sys.argv)>1 else ".").resolve()
assets=("design-engine.js","design-engine.css")
for asset in assets:
 source=root/"web"/asset
 assert source.is_file() and source.stat().st_size>0, f"missing runtime asset: {source}"
 for relative in ("scripts/build-webapp.mjs","scripts/export-webapp.mjs","scripts/release/build-webapp.py","scripts/release/preflight.sh","packaging/build-rpms.sh","scripts/release/verify-artifacts.sh"):
  assert asset in (root/relative).read_text(encoding="utf-8"),f"{relative} omits {asset}"
 assert asset in (root/"web/index.html").read_text(encoding="utf-8"),f"entry point omits {asset}"
print("R8 adaptive design runtime asset packaging contracts: PASS")
