import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const app = fs.readFileSync(new URL('../web/app.js', import.meta.url), 'utf8');

function runAction(action, nextAction, context) {
  const start = app.indexOf(`case '${action}':`);
  const end = app.indexOf(`case '${nextAction}':`, start);
  assert.ok(start >= 0 && end > start, `missing ${action} handler`);
  return vm.runInNewContext(`(async () => { switch ('${action}') { ${app.slice(start, end)} } })()`, context);
}

test('catalog administrator approval sends the artifact digest rather than the lifecycle document hash', async () => {
  for (const digests of [
    { approval_checksum: 'artifact', payload: { artifact_checksum: 'artifact' }, sha256: 'document' },
    { payload: { artifact_checksum: 'artifact' }, sha256: 'document' },
    { sha256: 'legacy-document' },
  ]) {
    const requests = [];
    await runAction('parser-action', 'rollout-approve', {
      id: 'candidate', value: 'approve',
      state: { user: { username: 'tenant-admin', role: 'admin' }, parserPacks: [{ id: 'candidate', revision: 7, created_by: 'tenant-admin', ...digests }] },
      prompt: () => 'Reviewed this exact parser revision.',
      api: async (path, options) => { requests.push({ path, ...options }); return {}; },
      toast: () => {}, navigate: async () => {},
    });
    assert.equal(requests.length, 1);
    assert.equal(requests[0].body.expected_checksum, digests.approval_checksum || digests.payload?.artifact_checksum || digests.sha256);
    assert.equal(requests[0].body.expected_revision, 7);
    assert.equal(requests[0].body.emergency_self_approval, true);
  }
});

test('administrator rollout approval covers other rollout types while retaining exact parser bundle binding', async () => {
  for (const type of ['config_push', 'package_update', 'restart', 'replay', 'parser_sync']) {
    const hash = 'a'.repeat(64), requests = [];
    const proposal = { id: 'proposal', revision: 3, created_by: 'tenant-admin', payload: { type, bundle_hash: hash } };
    await runAction('rollout-approve', 'rollout-cancel', {
      id: 'proposal',
      button: { closest: () => ({ querySelector: () => ({ value: 'Administrator reviewed this rollout.' }) }) },
      state: { page: 'rollouts', user: { username: 'tenant-admin', role: 'admin' }, parserRolloutProposals: [proposal], rolloutActionErrors: {} },
      page: { querySelectorAll: () => [{ dataset: { rolloutReason: 'proposal' }, value: 'Administrator reviewed this rollout.' }] },
      rolloutBundleHash: doc => doc.payload.bundle_hash,
      isParserSyncProposal: doc => doc.payload.type === 'parser_sync',
      rolloutApprovalProblem: () => '',
      api: async (path, options) => { requests.push({ path, ...options }); return {}; },
      toast: () => {}, navigate: async () => {}, refreshApprovalBadge: async () => {},
    });
    assert.equal(requests.length, 1, type);
    assert.equal(requests[0].body.reason, 'Administrator reviewed this rollout.');
    assert.equal(requests[0].body.emergency_self_approval, type === 'parser_sync' ? true : undefined);
    if (type === 'parser_sync') assert.equal(requests[0].body.expected_bundle_hash, hash);
  }
});

test('catalog UI retains non-admin separation of duties', async () => {
  let sent = false;
  await assert.rejects(runAction('parser-action', 'rollout-approve', {
    id: 'candidate', value: 'approve',
    state: { user: { username: 'reviewer', role: 'reviewer' }, parserPacks: [{ id: 'candidate', created_by: 'reviewer' }] },
    api: async () => { sent = true; },
  }), /different administrator/);
  assert.equal(sent, false);
});
