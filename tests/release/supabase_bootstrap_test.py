"""Execute the setup helper's actual renderer with synthetic installation data."""
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
root=Path(sys.argv[1] if len(sys.argv)>1 else ".").resolve()
identity=dict(line.split("=",1) for line in (root/"packaging/release.env").read_text(encoding="utf-8").splitlines() if "=" in line and not line.startswith("#"))
release=identity["NLI_VERSION"]+"-"+identity["NLI_RELEASE"]
setup=(root/"scripts/setup-controller-rocky.sh").read_text(encoding="utf-8")
script=setup.split('NLI_SETUP_SUPABASE_HASH=$supabase_secret_sha256 python3 - ',1)[1].split("<<'PY'\n",1)[1].split('\nPY',1)[0]
with tempfile.TemporaryDirectory() as temporary:
 target=Path(temporary)/"bootstrap.sql"
 env=dict(os.environ,NLI_SETUP_SUPABASE_HASH="a"*64)
 subprocess.run([sys.executable,"-c",script,str(root/"supabase/nli_schema.sql"),str(target)],env=env,check=True)
 rendered=target.read_text(encoding="utf-8")
 assert "("+release+")" in rendered
 assert "'bootstrap_release', '"+release+"'" in rendered
 assert "Generated installation-specific bootstrap for Ver21 R"+identity["NLI_RELEASE"] in rendered
 assert "Generated installation-specific bootstrap" in rendered
 assert "Run this COMPLETE generated file" in rendered
 assert "Do not run this template directly" not in rendered
 assert "__NLI_CONTROLLER_SECRET_SHA256__" not in rendered
 assert "compatible RPC schema version 1" in rendered
 assert "bootstrap_release" in rendered and "nfi_mapping_plan_metadata" in rendered
 if os.name!="nt": assert target.stat().st_mode & 0o777 == 0o600
 env["NLI_SETUP_SUPABASE_HASH"]="invalid"
 result=subprocess.run([sys.executable,"-c",script,str(root/"supabase/nli_schema.sql"),str(target)],env=env,capture_output=True)
 assert result.returncode != 0
print("R8 generated Supabase bootstrap contract: PASS")
