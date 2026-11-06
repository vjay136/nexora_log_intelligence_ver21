#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
# shellcheck disable=SC1091
source "$ROOT/packaging/docker/images.env"
container="nli-supabase-schema-$$"
scratch=$(mktemp -d "${NLI_BUILD_SCRATCH:?NLI_BUILD_SCRATCH is required}/nli-supabase.XXXXXXXX")
secret='nli-schema-contract-secret-0123456789abcdef'
digest=$(printf '%s' "$secret" | sha256sum | awk '{print $1}')
payload_digest=$(printf '%s' '{"value":1}' | sha256sum | awk '{print $1}')
sed "s/__NLI_CONTROLLER_SECRET_SHA256__/$digest/" "$ROOT/supabase/nli_schema.sql" >"$scratch/schema.sql"

cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -rf "$scratch"
}
trap cleanup EXIT

docker run -d --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust "$POSTGRES_IMAGE" >/dev/null
for _ in $(seq 1 60); do
  docker exec "$container" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1 && break
  sleep 1
done
docker exec "$container" pg_isready -h 127.0.0.1 -U postgres >/dev/null
docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -U postgres -c 'create role anon nologin; create role authenticated nologin; create schema storage; create table storage.buckets (id text primary key, name text not null, public boolean not null, file_size_limit bigint, allowed_mime_types text[]);' >/dev/null
# Optional genuine prior-release SQL qualification; never connect to live DBs.
if [[ -n ${NLI_PREVIOUS_SUPABASE_SCHEMA:-} ]]; then
  [[ -f $NLI_PREVIOUS_SUPABASE_SCHEMA ]] || { echo 'Previous Supabase schema file is missing' >&2; exit 1; }
  docker exec "$container" createdb -h 127.0.0.1 -U postgres nli_r6_upgrade
  docker exec "$container" psql -d nli_r6_upgrade -h 127.0.0.1 -v ON_ERROR_STOP=1 -U postgres -c 'create schema storage; create table storage.buckets (id text primary key, name text not null, public boolean not null, file_size_limit bigint, allowed_mime_types text[]);' >/dev/null
  sed "s/__NLI_CONTROLLER_SECRET_SHA256__/$digest/" "$NLI_PREVIOUS_SUPABASE_SCHEMA" >"$scratch/previous-schema.sql"
  docker exec -i "$container" psql -d nli_r6_upgrade -h 127.0.0.1 -v ON_ERROR_STOP=1 -U postgres <"$scratch/previous-schema.sql" >/dev/null
  docker exec -i "$container" psql -d nli_r6_upgrade -h 127.0.0.1 -v ON_ERROR_STOP=1 -qAt -U postgres <<SQL >"$scratch/r6-before"
set role anon;
set request.headers = '{"x-nli-controller-secret":"$secret"}';
select public.nli_put_document('parser-packs','r6-preserved','published','{"parser":{"pattern_type":"regex","pattern":"(?P<message>.+)"},"baseline":"21.0.0-6"}'::jsonb,'$payload_digest','r6-admin',false);
reset role;
select md5(row_to_json(d)::text) from public.nli_documents d where collection='parser-packs' and document_id='r6-preserved' and revision=1;
SQL
  docker exec -i "$container" psql -d nli_r6_upgrade -h 127.0.0.1 -v ON_ERROR_STOP=1 -U postgres <"$scratch/schema.sql" >/dev/null
  docker exec -i "$container" psql -d nli_r6_upgrade -h 127.0.0.1 -v ON_ERROR_STOP=1 -qAt -U postgres <<SQL >"$scratch/r6-after"
select md5(row_to_json(d)::text) from public.nli_documents d where collection='parser-packs' and document_id='r6-preserved' and revision=1;
set role anon;
set request.headers = '{"x-nli-controller-secret":"$secret"}';
select public.nli_health()->>'bootstrap_release' = '21.0.0-7' and public.nli_get_document('parser-packs','r6-preserved',1)->'payload'->>'baseline' = '21.0.0-6';
SQL
  [[ $(tail -n1 "$scratch/r6-before") == $(head -n1 "$scratch/r6-after") && $(tail -n1 "$scratch/r6-after") == 't' ]] || { echo 'Genuine R6 to R7 SQL upgrade changed existing metadata' >&2; exit 1; }
fi

docker exec -i "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -U postgres <"$scratch/schema.sql" >/dev/null
[[ $(docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -At -U postgres -c "select count(*) from storage.buckets where id in ('nli-assurance-fixtures','nli-rollout-artifacts','nli-geoip-databases') and public=false") == '3' ]] || { echo 'NLI Storage buckets are not private or missing' >&2; exit 1; }
[[ $(docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -At -U postgres -c "select allowed_mime_types = array['application/octet-stream']::text[] and file_size_limit=8388608 from storage.buckets where id='nli-assurance-fixtures'") == 't' ]] || { echo 'Assurance bucket must accept only bounded encrypted octet-stream objects' >&2; exit 1; }
[[ $(docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -At -U postgres -c "select allowed_mime_types @> array['application/x-rpm','application/json']::text[] and cardinality(allowed_mime_types)=3 and file_size_limit=536870912 from storage.buckets where id='nli-rollout-artifacts'") == 't' ]] || { echo 'Rollout bucket MIME or size allowlist is invalid' >&2; exit 1; }
[[ $(docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -At -U postgres -c "select allowed_mime_types = array['application/octet-stream']::text[] and file_size_limit=134217728 from storage.buckets where id='nli-geoip-databases'") == 't' ]] || { echo 'GeoIP bucket must be private octet-stream with a 128 MiB object bound' >&2; exit 1; }

docker exec -i "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -qAt -U postgres <<SQL >"$scratch/result"
set role anon;
set request.headers = '{"x-nli-controller-secret":"$secret"}';
select public.nli_health()->>'status';
select public.nli_put_document('contract','item','draft','{"value":1}'::jsonb,'$payload_digest','contract-test',false)->>'revision';
select jsonb_array_length(public.nli_list_documents('contract','',10,0));
select public.nli_get_document('contract','item',0)->'payload'->>'value';
select public.nli_delete_document('contract','item',true,'contract-test')->>'status';
SQL
[[ $(tr '\n' ' ' <"$scratch/result") == 'ok 1 1 1 purged ' ]] || {
  echo 'Supabase protected RPC contract returned unexpected results' >&2
  cat "$scratch/result" >&2
  exit 1
}
if docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -At -U postgres -c "set role anon; set request.headers = '{\"x-nli-controller-secret\":\"wrong-secret\"}'; select public.nli_health();" >/dev/null 2>&1; then
  echo 'Supabase RPC accepted an incorrect controller secret' >&2
  exit 1
fi
if docker exec "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -At -U postgres -c 'set role anon; select count(*) from public.nli_documents;' >/dev/null 2>&1; then
  echo 'Supabase anon role bypassed direct-table denial' >&2
  exit 1
fi
# R7 is a persistence-compatible RPC v1 extension, not a new document schema.
# Preserve a pre-existing R6-shaped parser document and the exact NFI JSONB
# plan (including null/array/object values) across a complete bootstrap rerun.
docker exec -i "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -qAt -U postgres <<SQL >"$scratch/r7-before"
set role anon;
set request.headers = '{"x-nli-controller-secret":"$secret"}';
select public.nli_put_document('parser-packs','r6-preserved','published','{"parser":{"pattern_type":"regex","pattern":"(?P<message>.+)"},"baseline":"21.0.0-6"}'::jsonb,'$payload_digest','r6-admin',false)->>'revision';
select public.nli_put_document('parser-packs','nfi-r7','draft','{"parser":{"pattern_type":"mapping","mapping_plan":{"version":1,"fields":[{"name":"event","method":"position","source":"8","operations":[{"op":"split","delimiter":";","part":2}]},{"name":"optional_value","method":"json_path","source":"value","policy":"nullable","null_policy":"preserve_null"}]},"service_scope":["smtp-in"],"scope_pattern":"Note;","reviewed_global_scope":false},"typed":{"empty":"","null":null,"array":[1,null,"0042"],"object":{"ok":true}}}'::jsonb,'$payload_digest','r7-admin',false)->>'revision';
select public.nli_put_document('parser-packs','nfi-r7','in_review',public.nli_get_document('parser-packs','nfi-r7',1)->'payload','$payload_digest','r7-admin',false)->>'revision';
reset role;
select md5(string_agg(row_to_json(d)::text, '|' order by collection,document_id,revision)) from public.nli_documents d;
SQL
# Direct table access for the checksum requires the test's postgres owner role.
docker exec -i "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -U postgres <"$scratch/schema.sql" >/dev/null
docker exec -i "$container" psql -h 127.0.0.1 -v ON_ERROR_STOP=1 -qAt -U postgres <<SQL >"$scratch/r7-after"
select md5(string_agg(row_to_json(d)::text, '|' order by collection,document_id,revision)) from public.nli_documents d;
set role anon;
set request.headers = '{"x-nli-controller-secret":"$secret"}';
select public.nli_health()->>'bootstrap_release' = '21.0.0-7' and public.nli_health()->>'schema_version' = '1' and public.nli_health()->'persistence_capabilities' ? 'nfi_mapping_plan_metadata';
select public.nli_get_document('parser-packs','r6-preserved',1)->'payload'->>'baseline' = '21.0.0-6';
select public.nli_get_document('parser-packs','nfi-r7',0)->>'state' = 'in_review' and public.nli_get_document('parser-packs','nfi-r7',0)->>'revision' = '2';
select public.nli_get_document('parser-packs','nfi-r7',1)->'payload' = public.nli_get_document('parser-packs','nfi-r7',2)->'payload';
select public.nli_get_document('parser-packs','nfi-r7',2)->'payload'->'typed' = '{"empty":"","null":null,"array":[1,null,"0042"],"object":{"ok":true}}'::jsonb;
select public.nli_get_document('parser-packs','nfi-r7',2)->'payload'->'parser'->'mapping_plan'->'fields'->0->'operations'->0->>'delimiter' = ';';
SQL
[[ $(tail -n1 "$scratch/r7-before") == $(head -n1 "$scratch/r7-after") ]] || { echo 'Bootstrap rerun changed existing document revisions' >&2; exit 1; }
[[ $(tail -n +2 "$scratch/r7-after" | tr '\n' ' ') == 't t t t t t ' ]] || { echo 'R7 NFI JSONB/review/R6 persistence contract failed' >&2; exit 1; }

echo 'Supabase schema/RLS/RPC contract: PASS'
