"""Execute the setup helper's actual renderer with synthetic installation data."""
import os
import shutil
from pathlib import Path
import subprocess
import sys
import tempfile
root=Path(sys.argv[1] if len(sys.argv)>1 else ".").resolve()
package_identity=dict(line.split("=", 1) for line in (root/"packaging/release.env").read_text(encoding="utf-8").splitlines() if line.startswith(("NLI_VERSION=", "NLI_RELEASE=")))
release=f'{package_identity["NLI_VERSION"]}-{package_identity["NLI_RELEASE"]}'
setup=(root/"scripts/setup-controller-rocky.sh").read_text(encoding="utf-8")
script=setup.split('NLI_SETUP_SUPABASE_HASH=$supabase_secret_sha256 python3 - ',1)[1].split("<<'PY'\n",1)[1].split('\nPY',1)[0]
with tempfile.TemporaryDirectory() as temporary:
 target=Path(temporary)/"bootstrap.sql"
 env=dict(os.environ,NLI_SETUP_SUPABASE_HASH="a"*64)
 subprocess.run([sys.executable,"-c",script,str(root/"supabase/nli_schema.sql"),str(target)],env=env,check=True)
 rendered=target.read_text(encoding="utf-8")
 assert "("+release+")" in rendered
 assert "'bootstrap_release', '"+release+"'" in rendered
 assert f"Generated installation-specific bootstrap for NLI {release}" in rendered
 assert "Run this COMPLETE generated file" in rendered
 assert "Do not run this template directly" not in rendered
 assert "__NLI_CONTROLLER_SECRET_SHA256__" not in rendered
 assert "compatible RPC schema version 1" in rendered
 assert "bootstrap_release" in rendered and "nfi_mapping_plan_metadata" in rendered
 if os.name!="nt": assert target.stat().st_mode & 0o777 == 0o600
 env["NLI_SETUP_SUPABASE_HASH"]="invalid"
 result=subprocess.run([sys.executable,"-c",script,str(root/"supabase/nli_schema.sql"),str(target)],env=env,capture_output=True)
 assert result.returncode != 0
# Exercise the actual setup probe against synthetic transport responses. An
# authorization rejection must be classified separately from an absent schema.
probe = setup.split("supabase_auth_rejected=false\nverify_supabase() {", 1)[1].split("\n}\n", 1)[0]
probe = "supabase_auth_rejected=false\nverify_supabase() {" + probe + "\n}\n"
bash = "C:/Program Files/Git/bin/bash.exe" if os.name == "nt" else shutil.which("bash")
assert bash, "bash is required for the setup probe contract"
for status, body, rejected, success in (
    (200, '{"status":"ok","schema_version":1}', "false", True),
    (401, '{"code":"42501","message":"NLI controller authentication failed"}', "true", False),
    (403, '{"message":"forbidden"}', "true", False),
    (404, '{"code":"PGRST202"}', "false", False),
    (503, '{"message":"unavailable"}', "false", False),
):
    harness = """set -euo pipefail
supabase_public_key=synthetic-public-key
supabase_controller_secret=synthetic-controller-secret
supabase_url=https://project.supabase.co
python3() { "$NLI_TEST_PYTHON" "$@"; }
curl() { printf '%s\\n%s' "$NLI_TEST_BODY" "$NLI_TEST_STATUS"; }
""" + probe + """if verify_supabase; then outcome=ok; else outcome=failed; fi
printf '%s %s' "$outcome" "$supabase_auth_rejected"
"""
    env = dict(os.environ, NLI_TEST_STATUS=str(status), NLI_TEST_BODY=body, NLI_TEST_PYTHON=sys.executable.replace("\\", "/"))
    result = subprocess.run([bash, "-c", harness], env=env, capture_output=True, text=True, check=True)
    assert result.stdout == ("ok" if success else "failed") + " " + rejected, result.stdout
# Changing TLS mode on an initialized host must keep its metadata backend.
# Execute the installer's actual selection and mode-default blocks.
backend = setup.split("# Preserve the metadata backend of an initialized controller", 1)[1].split("\n", 1)[1].split("\nsupabase_url=''", 1)[0]
mode_defaults = setup.split("offline_private_tls=false", 1)[1].split("\nadmin_password=''", 1)[0]
mode_defaults = "offline_private_tls=false" + mode_defaults
quick_defaults = "if [[ $mode == 10 ]]; then" + setup.split("if [[ $mode == 10 ]]; then", 1)[1].split("\nfi\n", 1)[0] + "\nfi\n"
project_guard = setup.split("supabase_url=${supabase_url%/}\n", 1)[1].split("\n[[ $(LC_ALL=C", 1)[0]
with tempfile.TemporaryDirectory() as temporary:
    state_dir = Path(temporary)
    config_path = state_dir / "controller.json"
    users_path = state_dir / "users.json"
    harness = """set -euo pipefail
python3() { "$NLI_TEST_PYTHON" "$@"; }
fail() { printf '%s' "$*" >&2; exit 1; }
info() { printf '%s' "$*" >&2; }
read_env() { if [[ $1 == NLI_ENROLLMENT_TOKEN ]]; then printf '%s' "${NLI_TEST_PRIOR_TOKEN:-}"; fi; }
PORT=12382
""" + backend + "\nmode=$NLI_TEST_MODE\nselected_mode=$mode\n" + mode_defaults + "\n" + quick_defaults + "\nprintf '%s' \"$storage_provider\"\n"
    def backend_probe(mode, *, explicit="", acknowledge="", token=""):
        env = dict(os.environ, CONFIG=str(config_path).replace("\\", "/"), STATE_DIR=str(state_dir).replace("\\", "/"), NLI_TEST_MODE=str(mode), NLI_TEST_PYTHON=sys.executable.replace("\\", "/"), NLI_SETUP_STORAGE_PROVIDER=explicit, NLI_SETUP_ALLOW_METADATA_BACKEND_CHANGE=acknowledge, NLI_TEST_PRIOR_TOKEN=token)
        return subprocess.run([bash, "-c", harness], env=env, capture_output=True, text=True)
    for provider in ("local", "supabase"):
        config_path.write_text('{"storage_provider":"'+provider+'"}', encoding="utf-8")
        users_path.write_text('{}', encoding="utf-8")
        for mode in range(1, 12):
            result = backend_probe(mode)
            assert result.returncode == 0 and result.stdout == provider, (provider, mode, result.stderr, result.stdout)
    users_path.unlink()
    config_path.write_text('{"storage_provider":"local"}', encoding="utf-8")
    result = backend_probe(11, token="saved-enrollment-token-0123456789abcdef")
    assert result.returncode == 0 and result.stdout == "local", result.stderr
    config_path.write_text('{"storage_provider":"supabase"}', encoding="utf-8")
    for mode in range(1, 12):
        result = backend_probe(mode)
        expected = "local" if mode in (7, 8, 10) else "supabase"
        assert result.returncode == 0 and result.stdout == expected, (mode, result.stderr, result.stdout)
    users_path.write_text('{}', encoding="utf-8")
    config_path.write_text('{"storage_provider":"local"}', encoding="utf-8")
    before = config_path.read_bytes()
    result = backend_probe(11, explicit="supabase")
    assert result.returncode != 0 and "NLI_SETUP_ALLOW_METADATA_BACKEND_CHANGE=1" in result.stderr, result.stderr
    assert config_path.read_bytes() == before
    result = backend_probe(11, explicit="supabase", acknowledge="1")
    assert result.returncode == 0 and result.stdout == "supabase", result.stderr
    config_path.write_text('{broken', encoding="utf-8")
    result = backend_probe(11)
    assert result.returncode != 0 and config_path.read_text(encoding="utf-8") == '{broken', result.stderr
    config_path.write_text('{"storage_provider":"supabase"}', encoding="utf-8")
    project_harness = harness.rsplit("printf '%s'", 1)[0] + "\ncurrent_supabase_url=https://original.supabase.co/\nsupabase_url=$NLI_TEST_PROJECT\n" + project_guard + "\nprintf ok\n"
    for project, ack, allowed in (("https://original.supabase.co", "", True), ("https://replacement.supabase.co", "", False), ("https://replacement.supabase.co", "1", True)):
        env = dict(os.environ, CONFIG=str(config_path).replace("\\", "/"), STATE_DIR=str(state_dir).replace("\\", "/"), NLI_TEST_MODE="11", NLI_TEST_PROJECT=project, NLI_TEST_PYTHON=sys.executable.replace("\\", "/"), NLI_SETUP_STORAGE_PROVIDER="", NLI_SETUP_ALLOW_METADATA_BACKEND_CHANGE=ack)
        result = subprocess.run([bash, "-c", project_harness], env=env, capture_output=True, text=True)
        assert (result.returncode == 0) == allowed, (project, ack, result.stderr)
print("Generated Supabase bootstrap release contract: PASS")
