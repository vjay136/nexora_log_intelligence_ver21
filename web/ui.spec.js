import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { contrast as designContrast } from './design-engine.js';

// These browser regressions verify frontend behavior independently of network
// and database availability. e2e.spec.js separately exercises a real controller.
const origin = 'http://127.0.0.1:4176';
const tabs = ['sample-history', 'header-templates', 'kv-templates', 'operations', 'monitoring', 'streams', 'sync-fabric', 'nhle-monitor', 'napi-monitor', 'nal-monitor', 'assurance', 'quarantine-wizard', 'upes', 'parser-workspace', 'nupire', 'api-monitor', 'field-integrity', 'fleet', 'parser-sync', 'clients', 'controller', 'parsers', 'reference-ai', 'building-blocks', 'field-mapping', 'migration', 'input-paths', 'kv', 'static', 'projection', 'studio', 'regex', 'knownkeys', 'semantic', 'engine-network', 'engine-url', 'engine-email', 'engine-subject', 'outputs', 'metadata', 'unknowns', 'drift', 'performance', 'rollouts', 'artifacts', 'security', 'system', 'communications', 'notifications', 'ai'];
const assets = Object.fromEntries(await Promise.all(['index.html', 'app.js', 'styles.css', 'design-engine.js', 'design-engine.css', 'nli-mark.svg', 'nexora-logo-v21.png', 'login-banner-v21.png', 'nexora-poster-v21.png'].map(async name => [name, await readFile(new URL(name, import.meta.url))])));

test('contextual Ask AI and visual blocks start from an empty mapping', async ({page})=>{
  await mockController(page);
  await page.route('**/api/v1/ai/workspace',route=>{
    const body=route.request().postDataJSON();
    expect(body).toMatchObject({workspace:'building-blocks',mode:'ask',question:'How do I split a field?'});
    return route.fulfill({json:{answer:'Select the exact source range.',review_only:true}});
  });
  await login(page);
  await page.getByTestId('nav-building-blocks').click();
  await page.getByRole('button',{name:'Ask AI',exact:true}).click();
  const dialog=page.getByRole('dialog');
  await dialog.getByRole('textbox',{name:'What do you need help with?'}).fill('How do I split a field?');
  await dialog.getByRole('button',{name:'Get guidance'}).click();
  await expect(dialog.getByRole('status')).toHaveText('Select the exact source range.');
  await dialog.getByRole('button',{name:'Close',exact:true}).click();
  const raw=page.locator('#block-raw');
  await raw.fill('smtp status=failed');
  await raw.evaluate(input=>input.setSelectionRange(5,18));
  await page.getByRole('button',{name:'Split at selected text'}).click();
  await page.locator('[data-block-name]').fill('status');
  await page.locator('[data-block-delimiter]').fill('=');
  await page.locator('[data-block-part]').fill('2');
  await page.getByRole('button',{name:'Open blocks in FieldSense'}).click();
  await expect(page.locator('#nfi-raw')).toHaveValue('smtp status=failed');
  await expect(page.locator('#qrw-field-rows [data-qrw-key=name]')).toHaveValue('status');
});

test('Ask NLI AI sends selected workspace, issue, and current raw sample to the live gateway',async({page})=>{
  await mockController(page);
  const raw='smtp status=failed';
  await page.route('**/api/v1/ai/workspace',route=>{
    expect(route.request().postDataJSON()).toMatchObject({
      workspace:'quarantine-wizard',tab:'Step 4',mode:'ask',raw,
      question:'How do I map status?\n\nObserved issue or result: The raw tail is still unmapped.',
    });
    return route.fulfill({json:{answer:'Use Key Value with the source key status.'}});
  });
  await login(page);
  await page.getByTestId('nav-reference-ai').click();
  const help=page.locator('[data-ai-help-desk]');
  await help.getByLabel('NLI workspace').selectOption('quarantine-wizard');
  await help.getByLabel('Tab or step (optional)').fill('Step 4');
  await help.getByLabel('What happened? (optional)').fill('The raw tail is still unmapped.');
  await help.getByLabel('What do you need help with?').fill('How do I map status?');
  await help.getByLabel('Sample record').fill(raw);
  await help.getByRole('button',{name:'Get guidance'}).click();
  await expect(help.getByRole('status')).toHaveText('Use Key Value with the source key status.');
});

test('refresh shows only a spinner while AI is pending and keeps the returned result',async({page})=>{
  await mockController(page);
  let pending;
  await page.route('**/api/v1/ai/workspace',route=>{pending=route;});
  await login(page);
  await page.getByTestId('nav-reference-ai').click();
  const help=page.locator('[data-ai-help-desk]');
  await help.getByLabel('What do you need help with?').fill('How do I map status?');
  await help.getByRole('button',{name:'Get guidance'}).click();
  await expect.poll(()=>!!pending).toBe(true);
  const refresh=page.getByRole('button',{name:'Refresh',exact:true});
  await refresh.click();
  await expect(refresh.locator('.refresh-spinner')).toBeVisible();
  await expect(refresh.locator('[data-refresh-label]')).toBeHidden();
  await expect(refresh).toHaveAttribute('aria-busy','true');
  await pending.fulfill({json:{answer:'Select the status source key.'}});
  await expect(help.getByRole('status')).toHaveText('Select the status source key.');
  await expect(refresh.locator('.refresh-spinner')).toBeHidden();
  await expect(refresh).toBeEnabled();
});

test('AI request errors distinguish provider, controller transport, and field extraction',async({page})=>{
  await mockController(page);
  let issue=0;
  await page.route('**/api/v1/ai/workspace',route=>{
    issue++;
    if(issue===1)return route.fulfill({status:502,contentType:'text/plain',body:'Bad Gateway'});
    if(issue===2)return route.fulfill({status:502,json:{error:'Provider rejected the key',error_code:'http_401'}});
    return route.fulfill({status:422,json:{error:'Mapping could not extract this record',errors:['status: key not found']}});
  });
  await login(page);
  await page.getByTestId('nav-reference-ai').click();
  const help=page.locator('[data-ai-help-desk]');
  await help.getByLabel('What do you need help with?').fill('Map status from this sample');
  const submit=help.getByRole('button',{name:'Get guidance'});
  const result=help.getByRole('status');
  await submit.click();
  await expect(result).toContainText('transport error');
  await submit.click();
  await expect(result).toHaveText('Provider rejected the key');
  await submit.click();
  await expect(result).toContainText('status: key not found');
});

test('Building Blocks mapping button sends the current sample directly and offers an editable draft', async ({page})=>{
  await mockController(page);
  const raw='smtp status=failed';
  let calls=0;
  await page.route('**/api/v1/ai/workspace',route=>{
    calls++;
    expect(route.request().postDataJSON()).toMatchObject({workspace:'building-blocks',mode:'fix',artifact:'mapping',raw});
    return route.fulfill({json:{
      answer:'Status mapped from exact evidence.',
      proposal:{plan:{version:1,fields:[{name:'status',type:'string',method:'key_value',source:'status'}]}},
      validation:{sample_matched:true},review_only:true,persisted:false,activated:false,
    }});
  });
  await login(page);
  await page.getByTestId('nav-building-blocks').click();
  await page.locator('#block-raw').fill(raw);
  await page.getByRole('button',{name:'Suggest Mappings with AI',exact:true}).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  const draft=page.locator('[data-workspace-ai-auto-panel]');
  await expect(draft.getByRole('status')).toContainText('Status mapped from exact evidence.');
  page.once('dialog',dialog=>dialog.accept());
  await draft.getByRole('button',{name:'Apply mapping to editor'}).click();
  await expect(page.locator('#block-fields [data-block-name]')).toHaveValue('status');
  expect(calls).toBe(1);
});

test('Suggest Mappings with AI previews a labeled native fallback before import', async ({page})=>{
  await mockController(page);
  const raw='smtp status=failed';
  let calls=0;
  await page.route('**/api/v1/ai/workspace',route=>{
    const request=route.request().postDataJSON();
    if(request.mode==='ask'){
      expect(request.question).toContain('fix only field status');
      return route.fulfill({json:{answer:'Use the key/value method with source key status.'}});
    }
    calls++;
    expect(route.request().postDataJSON()).toMatchObject({workspace:'field-mapping',mode:'fix',artifact:'mapping',raw,native_fallback:true});
    return route.fulfill({json:{
      answer:'The provider returned an invalid mapping plan. This is a native FieldSense draft.',
      proposal_origin:'native_fallback',
      proposal:{plan:{version:1,fields:[{name:'status',type:'string',method:'key_value',source:'status'}]}},
      validation:{actual:{status:'failed'},syntax_valid:true,extraction_valid:true,diff:[{field:'status',status:'value',actual:'failed',desired:'ok'}]},
      review_only:true,persisted:false,activated:false,
    }});
  });
  await login(page);
  await page.getByTestId('nav-field-mapping').click();
  await page.locator('#nfi-raw').fill(raw);
  await page.getByRole('button',{name:'Suggest Mappings with AI',exact:true}).click();
  const draft=page.locator('[data-workspace-ai-auto-panel]');
  await expect(draft.getByRole('status')).toContainText('native FieldSense draft');
  await expect(draft.getByRole('status')).toContainText('"status": "failed"');
  await expect(page.locator('#qrw-field-rows [data-qrw-key=name]')).toHaveCount(0);
  await expect(draft.getByRole('heading',{name:'Fields that need attention'})).toBeVisible();
  await draft.getByRole('button',{name:'Ask AI about this field'}).click();
  const fieldHelp=page.getByRole('dialog');
  await fieldHelp.getByRole('button',{name:'Get guidance'}).click();
  await expect(fieldHelp.getByRole('status')).toContainText('source key status');
  await fieldHelp.getByRole('button',{name:'Close',exact:true}).click();
  page.once('dialog',dialog=>dialog.accept());
  await draft.getByRole('button',{name:'Apply mapping to editor'}).click();
  await expect(page.locator('#qrw-field-rows [data-qrw-key=name]')).toHaveValue('status');
  expect(calls).toBe(1);
});

test('Building Blocks adds pieces by drag and keyboard, reorders, and checks exact output',async({page})=>{
  await mockController(page);
  let request;
  await page.route('**/api/v1/assurance/mapping-preview',route=>{
    request=route.request().postDataJSON();
    return route.fulfill({json:{passed:true,actual:{separator:'=',status:'failed'}}});
  });
  await login(page);
  await page.getByTestId('nav-building-blocks').click();
  await page.locator('#block-raw').fill('smtp=failed');
  const tray=page.locator('#block-fields');
  await page.locator('#block-source button').filter({hasText:'failed'}).dragTo(tray);
  await page.locator('#block-source button').filter({hasText:'='}).press('Enter');
  await expect(tray.locator('[data-block-card]')).toHaveCount(2);
  await tray.locator('[data-block-name]').nth(0).fill('status');
  await tray.locator('[data-block-name]').nth(1).fill('separator');
  await page.locator('#block-order').getByRole('button',{name:'Move separator up'}).press('Enter');
  await expect(page.locator('#block-order li').first()).toContainText('separator');
  await page.getByRole('button',{name:'Verify output preview'}).click();
  await expect(page.locator('#block-preview')).toContainText('"status": "failed"');
  expect(request.plan.fields.map(({name,source,source_end})=>[name,source,source_end])).toEqual([['separator','5','5'],['status','6','11']]);
});

test('Building Blocks automatically previews native KV output and refreshes after a policy edit',async({page})=>{
  await mockController(page);
  const raw='smtp status=failed|reason=greylisted';
  await page.route('**/api/v1/field-mapping/suggest',route=>route.fulfill({json:{fields:[{
    name:'details',method:'kv_eol',type:'object',policy:'required',source:'2',unit:'token',
    kv_pair_separators:['|'],assignment_separators:['='],
  }]}}));
  await page.route('**/api/v1/assurance/mapping-preview',route=>{
    const request=route.request().postDataJSON();
    expect(request).toMatchObject({raw,plan:{version:1,fields:[{
      name:'details',method:'kv_all',type:'object',source:'2',unit:'token',end_of_line:true,
      kv_pair_separators:['|'],assignment_separators:['='],
    }]}});
    const fields={status:'failed',reason:'greylisted'};
    const actual=request.plan.fields[0].output_policy==='flatten'?fields:{details:fields};
    return route.fulfill({json:{passed:true,actual}});
  });
  await login(page);
  await page.getByTestId('nav-building-blocks').click();
  await page.getByLabel('Raw record').fill(raw);
  await page.getByRole('button',{name:'Sense and populate fields'}).click();
  await expect(page.locator('#block-fields .block-value')).toContainText('"reason": "greylisted"');
  await expect(page.locator('#block-preview')).toContainText('"details": {');
  await page.getByText('Edit piece',{exact:true}).click();
  await page.locator('[data-block-output]').selectOption('flatten');
  await expect(page.locator('#block-order')).toContainText('Keys merged into the output object');
  await expect(page.locator('#block-preview')).toHaveText('{\n  "status": "failed",\n  "reason": "greylisted"\n}');
});

test('Manual test samples can be retained and reopened through encrypted fixture history',async({page})=>{
  await mockController(page);
  const raw='smtp accepted';
  let saved;
  const id='test-sample-guided-repair-1234';
  await page.route('**/api/v1/assurance/cases',route=>{
    if(route.request().method()==='POST'){
      saved=route.request().postDataJSON();
      return route.fulfill({json:{id}});
    }
    return route.fulfill({json:{cases:saved?[{id,name:saved.name}]:[]}});
  });
  await page.route('**/api/v1/assurance/cases/'+id,route=>route.fulfill({json:{fixture:saved.case}}));
  await login(page);
  await page.getByTestId('nav-sample-history').click();
  await page.getByLabel('Automatically save pasted test samples during this sign-in').check();
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByLabel('Manual raw sample').fill(raw);
  await expect(page.getByLabel('Save an encrypted copy to Test Sample History')).toBeChecked();
  await page.getByRole('button',{name:/Use manual sample/}).click();
  await expect.poll(()=>saved?.case.raw).toBe(raw);
  expect(saved.retain).toBe(true);
  await page.getByTestId('nav-sample-history').click();
  await page.getByRole('button',{name:'Open in Guided Repair'}).click();
  await expect(page.locator('#gprs-preview-form [name=raw_evidence]')).toHaveValue(raw);
});

test('Provider storage failure keeps the provider manager and recovery control visible',async({page})=>{
  await mockController(page);
  await page.route('**/api/v1/admin/ai-provider',route=>route.fulfill({status:503,json:{error:'provider store unavailable'}}));
  await login(page);
  await page.getByTestId('nav-ai').click();
  await expect(page.getByRole('heading',{name:'AI connection manager'})).toBeVisible();
  await expect(page.getByText('Provider storage needs attention.')).toBeVisible();
  await expect(page.getByRole('button',{name:'Retry loading profiles'})).toBeVisible();
});

test('AI model order and exclusive default can be saved independently',async({page})=>{
  await mockController(page);
  let pinned='';
  let order=[];
  const first={id:'first',name:'First model',model:'first-model',provider:'openai',enabled:true,priority:1};
  const profile={id:'second',name:'Second model',model:'example-model',provider:'openai',enabled:true};
  await page.route(/\/api\/v1\/admin\/ai-provider(?:\?|$)/,route=>{
    const query=new URL(route.request().url()).searchParams;
    if(route.request().method()==='POST'){
      if(query.get('action')==='set-order')order=route.request().postDataJSON().profile_order;
      else pinned=query.get('action')==='set-default'?query.get('id'):'';
    }
    return route.fulfill({json:{provider:profile,profiles:[first,profile],default_profile_id:pinned,profile_order:order}});
  });
  await login(page);
  await page.getByTestId('nav-ai').click();
  await page.locator('#ai-profile-choice').selectOption('second');
  await page.getByRole('button',{name:'Set as Default AI Profile'}).click();
  await expect(page.getByText('NLI uses only profile', {exact:false})).toBeVisible();
  expect(pinned).toBe('second');
  const second=page.locator('#ai-route-order [data-profile-id=second]');
  await second.getByRole('button',{name:'Move down'}).click();
  await page.getByRole('button',{name:'Save model order'}).click();
  await expect(page.locator('#ai-route-order li').first()).toContainText('First model');
  expect(order).toEqual(['first','second']);
  expect(pinned).toBe('second');
  await page.getByRole('button',{name:'Use automatic routing'}).click();
  await expect(page.getByText('Automatic routing is active.', {exact:false})).toBeVisible();
  expect(pinned).toBe('');
  expect(order).toEqual(['first','second']);
});

async function mockController(page, { role = 'admin', healthIdentity = 'v21', upesSnapshot = null, assurancePassed = true, nupireCandidates = [], quarantineRaw = 'exact raw source line', qrwParsers = [], qrwRecords = null, qrwSyncAssets = null, navigationCounts = null, managedParsers = [], controllerNotifications = [], rolloutSeed = [], rolloutApprovalFailure = null, replayFailure = null, replayPending = false } = {}) {
  const observed = [];
  const cases = [];
  const uploads = [];
  const assuranceRequests = [];
  const assuranceCorpora = [];
  const assurancePlans = [];
  const assuranceRuns = [];
  const outputSaves = [];
  const inputSaves = [];
  const outputProfiles = [];
  const inputProfiles = [];
  const inputRolloutSaves = [];
  const rolloutSaves = [];
  const rolloutProposals = rolloutSeed.map(item => structuredClone(item));
  const rolloutApprovalRequests = [];
  const historyQueries = [];
  const historyCounts = {};
  const users = [];
  const auditEvents = [];
  const notificationPolicySaves = [];
  const notificationActions = [];
  const qrwParserActions = [];
  const qrwReplays = [];
  const qrwSyncVerifications = [];
  const qrwRolloutRequests = [];
  let currentBundleHash = qrwSyncAssets?.[0]?.desired?.sha256 || '';
  let currentQRWParsers = qrwParsers.map(item => structuredClone(item));
  const currentManagedParsers = managedParsers.map(item => structuredClone(item));
  let currentQRWRecords = qrwRecords ? structuredClone(qrwRecords) : null;
  let currentQRWSyncAssets = qrwSyncAssets ? structuredClone(qrwSyncAssets) : null;
  let currentQRWSyncClients = currentQRWSyncAssets ? currentQRWSyncAssets.flatMap(asset => asset.clients || []).map(client => structuredClone(client)) : null;
  const parserSyncMockClients = () => currentQRWSyncClients || [{ node_id: 'node-1', hostname: 'mail-node-1', desired_revision: 3, desired_hash: 'a'.repeat(64), applied_revision: 3, applied_hash: 'a'.repeat(64), last_seen: '2026-10-09T10:00:00Z', status: 'online', drifted: false }];
  const currentControllerNotifications = controllerNotifications.map(item => structuredClone(item));
  const currentQRWHistory = [];
  let authenticated = false;
  await page.route(`${origin}/**`, async route => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (!path.startsWith('/api/v1/')) {
      const name = path === '/' ? 'index.html' : path.slice(1);
      return route.fulfill({ status: assets[name] ? 200 : 404, contentType: name.endsWith('.js') ? 'application/javascript' : name.endsWith('.css') ? 'text/css' : name.endsWith('.svg') ? 'image/svg+xml' : name.endsWith('.png') ? 'image/png' : 'text/html', body: assets[name] || 'Not found' });
    }
    observed.push(`${request.method()} ${path}`);
    const headers = { 'content-type': 'application/json', 'x-nli-controller': healthIdentity, 'x-nli-upstream': 'controller' };
    const fulfill = (value, status = 200) => {
      if (path === '/api/v1/sync/snapshot' && upesSnapshot) value = { ...value, subsystems: { ...(value.subsystems || {}), upes: upesSnapshot } };
      return route.fulfill({ status, headers, body: JSON.stringify(value) });
    };
    if (path === '/api/v1/health') return fulfill({ status: 'ok', service: 'nli-controller-agent', api_version: 'v1' });
    if (path === '/api/v1/navigation/counts') {
      const lifecycle = items => items.filter(item => ['draft','rejected','in_review','approved'].includes(item.state)).length;
      const waiting = (currentQRWRecords || [{ status: 'waiting', raw_available: true }]).filter(item => ['waiting','replay_failed'].includes(item.status) && item.raw_available === true).length;
      const parserActions = lifecycle(currentQRWParsers);
      const counts = {
        assurance: { total: assuranceRuns.filter(item => ['queued','running'].includes(item.state || item.payload?.state)).length },
        'quarantine-wizard': { total: waiting }, unknowns: { total: waiting }, upes: { total: waiting },
        rollouts: { total: rolloutProposals.filter(item => item.state === 'pending_approval').length },
        parsers: { total: parserActions }, 'parser-workspace': { total: parserActions },
        'input-paths': { total: lifecycle(inputProfiles) }, outputs: { total: lifecycle(outputProfiles) },
        kv: { total: 0 }, static: { total: 0 }, projection: { total: 0 }, studio: { total: 0 }, regex: { total: 0 }, knownkeys: { total: 0 }, semantic: { total: 0 },
        ...(navigationCounts || {}),
      };
      return fulfill({ generated_at: new Date().toISOString(), counts, source: 'controller lifecycle stores' });
    }
    if (path === '/api/v1/auth/login' && request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers: { ...headers, allow: 'POST, OPTIONS' } });
    if (path === '/api/v1/auth/login') { authenticated = true; return fulfill({ status: 'ok' }); }
    if (path === '/api/v1/auth/me') return authenticated ? fulfill({ username: 'test-operator', role, permissions: ['assurance:run'] }) : fulfill({ error: 'unauthorized' }, 401);
    if (path === '/api/v1/field-mapping') return fulfill({engine:'native-field-mapping-v1',profile_count:0,profiles:[],methods:['position','key_value','delimiter','character'],operations:['split','merge','replace'],limits:{max_fields:128,max_operations:16,max_input_bytes:1048576}});
    if (path === '/api/v1/parser-packs/deleted') return fulfill({deleted_parsers:[],count:0});
    if (path === '/api/v1/parser-sync/runtime') return fulfill({controller:{status:'unavailable',parsers:[]},clients:[]});
    if (path === '/api/v1/auth/csrf') return fulfill({ csrf_token: 'browser-test-csrf' });
    if (path === '/api/v1/auth/logout') { authenticated = false; return fulfill({ status: 'ok' }); }
    if (path === '/api/v1/nupire/snapshot') return fulfill({ candidates: nupireCandidates, count: nupireCandidates.length, evolution_state: 'available' });
    if (path === '/api/v1/assurance/run') {
      const requestBody = request.postDataJSON();
      assuranceRequests.push(requestBody);
      return fulfill({ passed: assurancePassed, route: assurancePassed ? 'normal' : 'quarantine', actual: { message: requestBody.raw, service: requestBody.service }, parser: { name: assurancePassed ? 'test-parser' : 'unknown', version: '7', strategy: 'json' }, counts: { actual: 2, attempts: 1 }, trace: [{ stage: 'parser_attempt', parser: assurancePassed ? 'test-parser' : 'unknown', success: assurancePassed, fields: 2 }], evidence_spans: [{ field: 'message', source_backed: true, start: 0, end: requestBody.raw.length }], parser_creation_helper: { normalized_sample: requestBody.raw.replace(/\r\n/g,'\n').split('\n').map(line=>line.replace(/[\t ]+$/,'')).join('\n'), sample_lines: 2, detected_format: 'key_value', requested_family: requestBody.service, selected_parser: assurancePassed ? 'test-parser' : 'unknown', selected_strategy: 'key_value', match_scope: { kind: 'line', prefix: 'level=' }, suggested_regex: 'level=(?P<level>\\S+) user=(?P<user>\\S+)', kv_delimiters: ['='], fields: [{ name: 'level', source: 'group:level', type: 'string' }, { name: 'user', source: 'group:user', type: 'string' }], projection: ['level','user'], template: { level: '{level}', user: '{user}' }, requires_human_approval: true } });
    }
    if (path === '/api/v1/assurance/corpus' && request.method() === 'POST') { const body=request.postDataJSON();assuranceCorpora.push(body);return fulfill({total:body.cases.length,passed:body.cases.length,failed:0,results:[]}); }
    if (path === '/api/v1/assurance/parser-preview' && request.method() === 'POST') { const body=request.postDataJSON();assuranceCorpora.push(body);const cases=body.cases||[],valid=assurancePassed&&!!body.parser?.pattern&&Array.isArray(body.parser?.fields)&&body.parser.fields.length>0&&cases.length>0&&cases.every(item=>item.raw&&item.desired&&Object.keys(item.desired).length>0);return fulfill({passed:valid,persisted:false,activated:false,artifact_checksum:'f'.repeat(64),validation:{status:valid?'passed':'failed',total:cases.length,passed:valid?cases.length:0,failed:valid?0:cases.length,score:valid?100:0,results:cases.map((item,index)=>({index,passed:valid,actual:valid?item.desired:{},diff:[],errors:valid?[]:['candidate did not match']}))}}); }
    if (path === '/api/v1/assurance/upload' && request.method() === 'POST') { uploads.push(request.postData() || ''); return fulfill({ retained: true, fixture_id: 'fixture-42', encrypted: true, filename: 'sample.log', content_type: 'text/plain', size_bytes: 28, sha256: 'a'.repeat(64), parser_revision: 'smtp@18.0.0', storage_status: 'encrypted_supabase' }); }
    if (path === '/api/v1/assurance/cases/fixture-42/download') return route.fulfill({ status: 200, headers: { ...headers, 'content-type': 'application/octet-stream' }, body: 'service=example user=operator' });
    if (path === '/api/v1/assurance/cases' && request.method() === 'POST') { const body = request.postDataJSON(); cases.push({ id: 'case-1', name: body.name, revision: 1, manual_retention: true, parser_id: 'test-parser', passed: true }); return fulfill(cases.at(-1)); }
    if (path === '/api/v1/assurance/cases') return fulfill({ cases: [{ id: 'fixture-42', name: 'Retained fixture', payload: { name: 'Retained fixture' } }, ...cases] });
    if (path === '/api/v1/assurance/parsers' && request.method() === 'POST') { const body=request.postDataJSON(),doc={id:'draft-1',revision:1,state:'draft',payload:body};currentQRWParsers.push(doc);return fulfill({id:doc.id,state:doc.state,revision:doc.revision},201); }
    if (path.match(/^\/api\/v1\/assurance\/parsers\/[^/]+\/(validate|submit|approve|reject|cancel|publish|rollback|offboard)$/) && request.method() === 'POST') { const [,id,action]=path.match(/^\/api\/v1\/assurance\/parsers\/([^/]+)\/(validate|submit|approve|reject|cancel|publish|rollback|offboard)$/),doc=currentQRWParsers.find(item=>item.id===id);qrwParserActions.push({id,action,body:request.postDataJSON()});if(doc){doc.state=({validate:doc.state,submit:'in_review',approve:'approved',reject:'rejected',cancel:'draft',publish:'published',rollback:'approved',offboard:'approved'})[action];doc.payload={...doc.payload,...(action==='publish'?{activated:true}:action==='rollback'?{activated:false,rolled_back_at:'2026-10-28T00:00:00Z'}:action==='offboard'?{activated:false,offboarded_at:'2026-10-28T00:00:00Z'}:{})};if(action==='publish')currentBundleHash=doc.payload.artifact_checksum||currentBundleHash;}return fulfill({id,state:doc?.state||action,validation:{score:100},bundle_hash:currentBundleHash,rollout_proposal:['publish','rollback','offboard'].includes(action)?{id:`proposal-${action}-${id}`,state:'pending_approval'}:undefined}); }
    if (path === '/api/v1/assurance/parsers') return fulfill({ parsers: currentQRWParsers });
    if (path === '/api/v1/assurance/napi/synthesize' && request.method() === 'POST') { const body=request.postDataJSON();return fulfill({id:body.id,revision:1,state:'draft',draft_created:true,selected_candidate:'anchored-regex',strategy:'regex',validation:{score:100,held_out_passed:body.held_out.length},regex_safety:{safe:true,engine:'go-re2'},provenance:{discovery_count:body.discovery.length,held_out_count:body.held_out.length,raw_samples_persisted:false},requires_human_approval:true,published:false},201); }
    if (path === '/api/v1/assurance/ai-proposals') return fulfill({ proposals: [], raw_samples_retained: false });
    if (path === '/api/v1/assurance/history') return fulfill({ events: [] });
    if (path === '/api/v1/assurance/plans' && request.method() === 'POST') { const body=request.postDataJSON();const doc={id:body.id,revision:1,state:'active',sha256:'a'.repeat(64),payload:body};assurancePlans.push(doc);return fulfill({id:body.id,revision:1,state:'active'},201); }
    if (path === '/api/v1/assurance/runs' && request.method() === 'POST') { const body=request.postDataJSON(),queued={id:'run-1',plan_id:body.plan_id,plan_revision:body.plan_revision,plan_sha256:'a'.repeat(64),case_id:body.case_id,scenario:body.scenario,state:'queued',queued_at:'2026-10-19T00:00:00Z'},run={...queued,state:'passed',started_at:'2026-10-19T00:00:01Z',finished_at:'2026-10-19T00:00:02Z',evidence:{raw_sha256:'b'.repeat(64),passed:true,score:100}};assuranceRuns.push({id:run.id,state:run.state,payload:run});return fulfill(queued,202); }
    if (path === '/api/v1/assurance/runs') return fulfill({runs:assuranceRuns,count:assuranceRuns.length});
    if (path === '/api/v1/assurance/runs/run-1') return assuranceRuns.length?fulfill(assuranceRuns[0]):fulfill({error:'not found'},404);
    if (path === '/api/v1/assurance/plans') return fulfill({ plans: assurancePlans, count: assurancePlans.length });
    if (path.startsWith('/api/v1/assurance/plans/')) { const id=decodeURIComponent(path.split('/').at(-1)),index=assurancePlans.findIndex(item=>item.id===id);if(request.method()==='PUT'){const body=request.postDataJSON(),previous=assurancePlans[index],doc={id,revision:(previous?.revision||0)+1,state:'active',payload:body};if(index>=0)assurancePlans[index]=doc;else assurancePlans.push(doc);return fulfill({id,revision:doc.revision,state:'active'},201);}if(request.method()==='DELETE'){if(index>=0)assurancePlans.splice(index,1);return fulfill({status:'deleted'});}return index>=0?fulfill(assurancePlans[index]):fulfill({error:'not found'},404); }
    if (path === '/api/v1/quarantine/records/q-1/raw') return route.fulfill({ status: 200, headers: { ...headers, 'content-type': 'text/plain' }, body: quarantineRaw });
    if (path === '/api/v1/quarantine/records/q-1/analyze' && request.method() === 'POST') return fulfill({ raw_sha256: 'c'.repeat(64), record: { id: 'q-1', node_id: 'node-1', raw_available: true }, result: { raw: quarantineRaw, passed: false, parser: { name: 'no-match' } }, parser_changed: false, review_state: 'analysis_only' });
    if (path === '/api/v1/admin/ai-provider') return fulfill({ provider: { id: 'default', name: 'Default AI provider', provider: 'openai', model: 'test-model', endpoint: 'https://api.openai.com/v1', configured: false, enabled: false, credential_source: 'encrypted_inline', priority: 0 }, credential_modes: ['encrypted_inline','env_ref','file_ref','none'], templates: [{ id:'openai', name:'OpenAI', kind:'openai', base_url:'https://api.openai.com/v1', model_hint:'gpt-4.1-mini', credential_source:'encrypted_inline' },{ id:'nvidia-nim', name:'NVIDIA NIM', kind:'nvidia_nim', base_url:'https://integrate.api.nvidia.com/v1', model_hint:'provider/model', credential_source:'encrypted_inline' },{ id:'anthropic', name:'Anthropic Claude', kind:'anthropic', base_url:'https://api.anthropic.com/v1', model_hint:'claude-sonnet', credential_source:'encrypted_inline' },{ id:'minimax', name:'MiniMax', kind:'minimax', base_url:'https://api.minimax.io/v1', model_hint:'MiniMax model', credential_source:'encrypted_inline' },{ id:'qwen-dashscope', name:'Qwen / DashScope', kind:'qwen_dashscope', base_url:'https://dashscope-intl.aliyuncs.com/compatible-mode/v1', model_hint:'qwen model', credential_source:'encrypted_inline' },{ id:'gemini', name:'Gemini / Google', kind:'gemini', base_url:'https://generativelanguage.googleapis.com/v1beta', model_hint:'gemini model', credential_source:'encrypted_inline' },{ id:'openai-compatible', name:'OpenAI-compatible endpoint', kind:'openai_compatible', base_url:'https://example.invalid/v1', model_hint:'model name', credential_source:'encrypted_inline' },{ id:'ollama', name:'Ollama (controller-local)', kind:'ollama', base_url:'http://127.0.0.1:11434', model_hint:'local model', credential_source:'none' }] });
    if (path === '/api/v1/sync/snapshot') return fulfill({ schema_version: 'nli.sync.v1', revision: 7, generated_at: new Date().toISOString(), target_interval_ms: 1000, aggregate: { nli_source_records_read_total: 120, nli_source_bytes_read_total: 4096, nli_parser_records_succeeded_total: 116, nli_parser_records_failed_total: 4, nli_normalized_records_total: 116, nli_quarantined_records_total: 4, nli_validation_attempts_total: 120, nli_validation_passed_total: 116, nli_validation_failed_total: 4, nli_queue_depth: 4, read_lines_per_second: 90, normalized_per_second: 87, quarantined_per_second: 3, validation_attempts_per_second: 90, validation_passed_per_second: 87, validation_failed_per_second: 3, bytes_per_second: 4096, backpressure_source_count: 0, nli_telemetry_propagation_delay_seconds_p95: 0.08 }, sources: [{ node_id: 'node-1', source: '/var/log/mail.log', process_generation: 'gen-1', sequence: 7, phase: 'tailing', counters: { nli_source_records_read_total: 120, nli_parser_records_succeeded_total: 116, nli_parser_records_failed_total: 4, nli_normalized_records_total: 116, nli_quarantined_records_total: 4, nli_validation_attempts_total: 120, nli_validation_passed_total: 116, nli_validation_failed_total: 4 }, rates: { read_lines_per_second: 90, normalized_per_second: 87, quarantined_per_second: 3, validation_attempts_per_second: 90, validation_passed_per_second: 87, validation_failed_per_second: 3, bytes_per_second: 4096 }, queue: { nli_queue_depth: 4, queue_capacity: 128, backpressure: false }, controller_received_at: new Date().toISOString(), controller_processing_ms: 2, observed_propagation_delay_ms: 80, freshness: 'live', freshness_age_ms: 120, freshness_reason: 'within target', critical: false, diagnostic_codes: [] }], diagnostics: { subscribers: 1, tracked_sources: 1, live: 1, delayed: 0, stale: 0, disconnected: 0, accepted: 7, duplicate: 0, out_of_order: 0, sequence_gaps: 0, reconnects: 0, coalesced: 0, rejected: 0, missed_deadlines: 0, clock_skew: 0, propagation_delay_ms_p50: 35, propagation_delay_ms_p95: 80, propagation_delay_ms_p99: 95, issues: [{ code: 'parser_latency', stage: 'parser', severity: 'warning', component: 'NHLE', node_id: 'node-1', source: '/var/log/mail.log', reason: 'parser latency exceeded warning threshold', recommended_action: 'Inspect parser trace and queue saturation.', last_seen: new Date().toISOString(), occurrences: 1 }] }, controller: { health: 'ok', version: '20.0.0', uptime_seconds: 600, process_memory_alloc_bytes: 33554432, process_memory_sys_bytes: 67108864, cpu_count: 4, goroutines: 12, open_file_descriptors: 21 }, infrastructure: { clients: [{ node_id: 'node-1', hostname: 'mail-node-1', cpu_percent: 18, memory_percent: 31, memory_bytes: 1073741824, load_1m: .4, load_5m: .3, load_15m: .2, updated_at: new Date().toISOString() }] }, subsystems: { nhle: { status: 'live', updated_at: new Date().toISOString(), source_count: 1, active_nodes: 1, records_read_total: 120, normalized_total: 116, quarantined_total: 4, queue_depth: 4, backpressure_sources: 0, read_lines_per_second: 90, normalized_per_second: 87 }, napi: { status: 'live', updated_at: new Date().toISOString(), requests_total: 3, failures_total: 0, pending_drafts: 1, latest_proposal_at: new Date().toISOString() }, nal: { status: 'live', updated_at: new Date().toISOString(), total: 5, queued: 1, running: 1, passed: 3, failed: 0, cancelled: 0, latest_run_at: new Date().toISOString() }, delivery: { status: 'ready', updated_at: new Date().toISOString(), acknowledgements_by_destination: { Loki: { delivered: 116, failed: 1 } }, retry_backlog_latency_status: 'not_instrumented' }, replay: { status: 'ready', state: 'queued', updated_at: new Date().toISOString(), queued: 1, reprocessing: 1, resolved: 2, oldest_queued_age_seconds: 45, attempts_total: 3 }, release_readiness: { status: 'ready', state: 'ready', updated_at: new Date().toISOString(), managed_nodes: 1, release_drifted_nodes: 0, evidence: 'persisted desired and reported active release hashes' } } });
    if (path === '/api/v1/sync/stream') return route.fulfill({ status: 200, headers: { ...headers, 'content-type': 'text/event-stream', 'cache-control': 'no-cache' }, body: '' });
    if (path === '/api/v1/dashboard/snapshot') return fulfill({ overview: { fleet: { total: 1, online: 1 }, metadata: {}, storage: { mode: 'supabase', status: 'connected' } }, metadata: { records: [{ record: { node_id: 'node-1', service: 'smtp-in', parser_id: 'smtp-in/reviewed', route: 'normal', result: 'success', latency_ms: 2, timestamp: '2026-10-17T05:00:00Z' } }], routes: { normal: 1 } }, system: { health: 'ok', version: '20.0.0', storage: { mode: 'supabase', status: 'connected' } }, ai_usage: { requests: 0 }, fleet_sync: { nodes: [{ node: { id: 'node-1', hostname: 'mail-node-1', status: 'online', last_seen: '2026-10-17T05:00:00Z', issues: [], metrics: { logs_received: 120, logs_parsed: 116, logs_quarantined: 4, parse_rate_pct: 96.67, parse_latency_ms: 2.4, cpu_percent: 18, memory_percent: 31 } }, sync: { drifted: false } }] }, timeseries: { from: '2026-10-17T04:58:00Z', to: '2026-10-17T05:00:00Z', points: [{ timestamp: '2026-10-17T04:59:00Z', logs_received: 60, logs_parsed: 58, logs_quarantined: 2, errors: 1, parse_latency_ms_avg: 2.2 }, { timestamp: '2026-10-17T05:00:00Z', logs_received: 60, logs_parsed: 58, logs_quarantined: 2, errors: 0, parse_latency_ms_avg: 2.6 }], system: { controller_cpu_count: 4, controller_goroutines: 12, controller_memory_alloc_bytes: 33554432 } }, stream_files: [{ node_id: 'node-1', source: '/var/log/nli/input/smtp-in.log', phase: 'tailing', read_lines_per_second: 90, normalized_per_second: 87, quarantined_per_second: 3, queue_depth: 4, queue_capacity: 128, backpressure: false, read_duration_ns: 1000000, parse_duration_ns: 2000000, output_duration_ns: 3000000, updated_at: '2026-10-17T05:00:00Z' }] });
    if (path === '/api/v1/overview') return fulfill({ fleet: {}, metadata: {}, storage: { mode: 'supabase', status: 'connected' } });
    if (path === '/api/v1/telemetry/stats') return fulfill({ received: 120, parsed: 116, quarantined: 4 });
    if (path === '/api/v1/telemetry/stream/snapshot') return fulfill({ bounded: true, files: [{ node_id: 'node-1', source: '/var/log/mail.log', phase: 'tailing', read_lines_per_second: 90, normalized_per_second: 80, quarantined_per_second: 5, bytes_per_second: 4096, queue_depth: 4, queue_capacity: 128, backpressure: false, read_duration_ns: 1000000, parse_duration_ns: 2000000, output_duration_ns: 3000000, total_duration_ns: 6000000, read_lines: 900, updated_at: '2026-10-14T00:00:00Z', thresholds: { read_lines_per_second: 80, bytes_per_second: 5000, quarantined_per_second: 10, queue_depth: 100 } }] });
    if (path === '/api/v1/telemetry/query') return fulfill({ telemetry: [] });
    if (path === '/api/v1/quarantine/summary') return fulfill({ waiting: 1, reprocessing: 0, resolved: 2, raw_available: 1 });
    if (path === '/api/v1/quarantine/records') return fulfill({ records: currentQRWRecords || [{ id: 'q-1', node_id: 'node-1', service: 'smtp', first_seen: '2026-10-10T00:00:00Z', attempt_count: 2, status: 'waiting', error_codes: ['no_match'], raw_available: true }] });
    if (path === '/api/v1/quarantine/history') return fulfill({ events: currentQRWHistory });
    if (path.match(/^\/api\/v1\/quarantine\/records\/[^/]+\/exclude$/) && request.method() === 'POST') { const id=decodeURIComponent(path.split('/').at(-2)),record=(currentQRWRecords||[]).find(item=>item.id===id);if(record)record.status='excluded';return fulfill({ status:'excluded', record_id:id }); }
    if (path.match(/^\/api\/v1\/quarantine\/records\/[^/]+\/raw$/) && request.method() === 'DELETE') { const id=decodeURIComponent(path.split('/').at(-2)),record=(currentQRWRecords||[]).find(item=>item.id===id);if(record){record.status='evidence_deleted';record.raw_available=false;}return fulfill({ status:'evidence_deleted', record_id:id }); }
    if (path === '/api/v1/quarantine/reprocess' && request.method() === 'POST') { const body=request.postDataJSON();qrwReplays.push(body);if(replayFailure)return fulfill({error:replayFailure.message},replayFailure.status);if(!replayPending){for(const record of currentQRWRecords||[])if(body.record_ids.includes(record.id))record.status='resolved';currentQRWHistory.push({record_id:body.record_ids[0],status:'resolved',outcome:'success'});}return fulfill({ rollout_id: 'replay-1', status: 'pending' }, 202); }
    if (path.startsWith('/api/v1/metrics/timeseries')) { const params=new URL(request.url()).searchParams;historyQueries.push(params);const key=params.toString(),call=historyCounts[key]=(historyCounts[key]||0)+1,read=key.includes('range=24h')&&call>1?'42':'9007199254740993',parsed=read==='42'?'40':'9007199254740990'; return fulfill({ from: '2026-10-25T00:00:00Z', to: '2026-10-25T01:00:00Z', canonical_history: { bucket_width_seconds: 60, points: [{timestamp:'2026-10-25T00:59:00Z',read_delta:read,parsed_delta:parsed,quarantine_delta:'2',errors:1}] }, points: [{ timestamp: '2026-10-25T00:59:00Z', received: read, parsed, quarantine: 2, errors: 1, logs_received: read, logs_parsed: parsed, logs_quarantined: 2 }] }); }
    if (path === '/api/v1/verification/sample-corpus') return fulfill({ verdict: 'pass', known_key_inventory: { declared_family_keys: 278, covered_family_keys: 278, unmapped_family_keys: 0 }, totals: { records: 728, accounted_records: 728, parsed_records: 723, exact_parity_records: 723, failing_records: 0, expected_quarantined_records: 5, unexpected_quarantined_records: 0, parsed_comparable_raw_fields: 13101, parsed_generated_json_fields: 13101 }, families: [{ family: 'IMAP', records: 52, passing_records: 52, comparable_raw_fields: 1154, generated_json_fields: 1154, duplicate_occurrences: 0 }], records: [{ corpus: 'samples.log', source_line: 50, family: 'SMTP-OUT', expected_disposition: 'quarantined', disposition_reason: 'truncated record', record_sha256: 'a'.repeat(64) }] });
    if (path === '/api/v1/admin/ai-usage') return fulfill({ requests: 0, tokens: 0, pending_drafts: 0 });
    if (path === '/api/v1/fleet') return fulfill({ nodes: [{ id: 'node-1', hostname: 'mail-node-1', status: 'online', last_seen: '2026-10-17T05:00:00Z' }] });
    if (path === '/api/v1/fleet/sync') return fulfill({ nodes: parserSyncMockClients().map(client=>({node:{id:client.node_id,hostname:client.hostname||client.node_id,status:client.status||'online',last_seen:'2026-10-17T05:00:00Z'},sync:{drifted:!!client.drifted}})) });
    if (path === '/api/v1/parser-sync' && request.method() === 'GET') { const assets=(currentQRWSyncAssets || [{ asset_class: 'regex', asset_id: 'nginx-access', name: 'Nginx access pattern', desired: { revision: 3, sha256: 'a'.repeat(64), state: 'published' }, controller: { revision: 3, sha256: 'a'.repeat(64), status: 'applied' }, auto_rollout: { enabled: true, status: 'enabled' }, rollout: { id: 'rollout-1', status: 'complete', total_nodes: 1, done_nodes: 1, failed_nodes: 0, errors: [] } }]).map(({clients,...asset})=>asset),clients=parserSyncMockClients();return fulfill({ bundle: qrwSyncAssets ? { desired_hash: currentBundleHash, parser_ids: currentQRWParsers.filter(item=>item.state==='published').map(item=>item.id), revision: 2 } : undefined, count: assets.length, assets, clients, telemetry:{total_clients:clients.length,online_clients:clients.filter(item=>item.status==='online').length,offline_clients:clients.filter(item=>item.status!=='online').length,exact_acknowledged_clients:clients.filter(item=>!item.drifted&&item.applied_hash===item.desired_hash).length,pending_clients:clients.filter(item=>!item.applied_hash).length,drifted_clients:clients.filter(item=>item.drifted).length,failed_clients:0,acknowledgement_pct:clients.length?100*clients.filter(item=>!item.drifted&&item.applied_hash===item.desired_hash).length/clients.length:0,desired_bundle_hash:currentBundleHash,controller_status:'ready',target_limit:10000} }); }
    if (path === '/api/v1/parser-sync/rollout' && request.method() === 'POST') { const body=request.postDataJSON(),proposal={id:`qrw-rollout-${qrwRolloutRequests.length+1}`,revision:1,created_by:'test-operator',state:'pending_approval',payload:{type:'parser_sync',name:'Parser bundle rollout',target_mode:body.target_mode,target_nodes:body.target_mode==='all_eligible'?parserSyncMockClients().map(item=>item.node_id):[body.origin_node_id],bundle_hash:body.bundle_hash}};qrwRolloutRequests.push(body);rolloutProposals.push(proposal);return fulfill({proposal},201); }
    if (path === '/api/v1/parser-sync/verify' && request.method() === 'POST') { const body=request.postDataJSON();qrwSyncVerifications.push(body);const origin=parserSyncMockClients().find(item=>item.node_id===(body.node_ids?.[0]||body.node_id||'node-1'));if(origin){origin.applied_hash=origin.desired_hash;origin.drifted=false;}return fulfill({ verified: true, message: 'Reported state refreshed', client_acknowledgements:origin?[{node_id:origin.node_id,exact_acknowledgement:true}]:[] }); }
    if (path === '/api/v1/parser-sync/retry' && request.method() === 'POST') return fulfill({ proposal_id: 'proposal-1', state: 'pending_approval', message: 'Rollout proposal created; independent approval is required.' });
    if (path.match(/^\/api\/v1\/parser-packs\/[^/]+\/(submit|approve|reject|activate|repair)$/) && request.method() === 'POST') { const [,id,action]=path.match(/^\/api\/v1\/parser-packs\/([^/]+)\/(submit|approve|reject|activate|repair)$/),doc=currentManagedParsers.find(item=>item.id===id);if(!doc)return fulfill({error:'not found'},404);doc.state=({submit:'in_review',approve:'approved',reject:'rejected',activate:'published',repair:'rejected'})[action];doc.runtime_sync=action==='repair'?{status:'not_published'}:doc.runtime_sync;return fulfill(doc); }
    if (path === '/api/v1/parser-packs') return fulfill({ parser_packs: currentManagedParsers });
    if (path === '/api/v1/parsers') return fulfill([{ id: 'smtp', version: 7, format: 'syslog', description: 'SMTP parser', fields: ['message'] }]);
    if (path === '/api/v1/migration/reports') return fulfill({ reports: [] });
    if (path === '/api/v1/config/output-destinations' && request.method() === 'POST') { outputSaves.push(request.postDataJSON()); return fulfill({ id: 'customer-output', state: 'draft', revision: 1 }); }
    if (path === '/api/v1/config/output-destinations') return fulfill({ items: outputProfiles, count: outputProfiles.length });
    if (path.startsWith('/api/v1/config/output-destinations/')) { const item=outputProfiles.find(profile=>path.endsWith(`/${profile.id}`));return item?fulfill(item):fulfill({error:'not found'},404); }
    if (path.match(/^\/api\/v1\/config\/input-paths\/[^/]+\/(submit|approve|publish)$/) && request.method() === 'POST') { const [,id,action]=path.match(/^\/api\/v1\/config\/input-paths\/([^/]+)\/(submit|approve|publish)$/),doc=inputProfiles.find(item=>item.id===id);if(!doc)return fulfill({error:'not found'},404);doc.state=({submit:'in_review',approve:'approved',publish:'active'})[action];return fulfill({id,state:doc.state,revision:doc.revision}); }
    if (path === '/api/v1/config/input-paths' && request.method() === 'POST') { inputSaves.push(request.postDataJSON()); return fulfill({ id: 'mail-paths', state: 'draft', revision: 1 }); }
    if (path === '/api/v1/config/input-paths') return fulfill({ items: inputProfiles, count: inputProfiles.length });
    if (path.match(/^\/api\/v1\/config\/input-paths\/[^/]+\/rollout$/) && request.method() === 'POST') { inputRolloutSaves.push({ path, body: request.postDataJSON() }); return fulfill({ id: `input-proposal-${inputRolloutSaves.length}`, state: 'pending_approval' }, 201); }
    if (path.match(/^\/api\/v1\/config\/input-paths\/[^/]+\/rollout$/)) { const id=decodeURIComponent(path.split('/').at(-2));return fulfill({ profile_id:id, profile_state:'active', history:[], clients:[{node_id:'node-1',hostname:'mail-node-1',status:'online',input_path_sync:{status:'not_targeted'}}] }); }
    if (path.startsWith('/api/v1/config/input-paths/')) { const item=inputProfiles.find(profile=>path.endsWith(`/${profile.id}`));return item?fulfill(item):fulfill({error:'not found'},404); }
    if (path.startsWith('/api/v1/config/')) return fulfill({ items: [] });
    if (path === '/api/v1/metadata') return fulfill({ records: [], stats: {}, payload_retained: false });
    if (path === '/api/v1/rollouts' && request.method() === 'POST') { rolloutSaves.push(request.postDataJSON()); return fulfill({ id: 'proposal-test', state: 'pending_approval' }, 201); }
    if (path.match(/^\/api\/v1\/rollouts\/[^/]+\/(approve|cancel)$/) && request.method() === 'POST') { const [,id,action]=path.match(/^\/api\/v1\/rollouts\/([^/]+)\/(approve|cancel)$/),item=rolloutProposals.find(doc=>doc.id===id),body=request.postDataJSON();if(action==='approve'){rolloutApprovalRequests.push({id,body});if(rolloutApprovalFailure)return fulfill({error:rolloutApprovalFailure},409);}if(item)item.state=action==='approve'?'approved':'cancelled';return fulfill({status:item?.state||action}); }
    if (path === '/api/v1/rollouts') return fulfill({ proposals: rolloutProposals, live: [] });
    if (path === '/api/v1/artifacts') return fulfill({ artifacts: [], signing_keys_configured: 0 });
    if (path === '/api/v1/rbac/roles') return fulfill({ roles: { viewer: ['assurance:run'], admin: ['*'] } });
    if (path === '/api/v1/users') return fulfill({ users });
    if (path === '/api/v1/audit') return fulfill({ verification: 'ok', events: auditEvents });
    if (path === '/api/v1/communications/browser-heartbeat' && request.method() === 'POST') return fulfill({ status: 'recorded', observed_at: new Date().toISOString() });
    if (path === '/api/v1/communications/status') return fulfill({ generated_at: new Date().toISOString(), path: [{ hop: 'client_to_controller', state: 'live', evidence: 'client heartbeat', clients: [{ node_id: 'node-1', hostname: 'mail-node-1', state: 'live', last_seen: new Date().toISOString(), age_seconds: 0, heartbeat_status: 'live' }], client_count: 1 }, { hop: 'controller_internal', state: 'live', evidence: 'controller health', observed_at: new Date().toISOString() }, { hop: 'controller_to_webapp', state: 'live', evidence: 'browser heartbeat', browser: { state: 'live', status: 'recorded', observed_at: new Date().toISOString(), age_seconds: 0, username: 'test-operator', evidence: 'heartbeat accepted' } }], canonical: { revision: 7, generated_at: new Date().toISOString(), schema_version: 'nli.sync.v1', target_interval_ms: 1000 } });
    if (path === '/api/v1/notifications') {
      const visible=currentControllerNotifications.filter(item=>item.status!=='dismissed'&&item.status!=='snoozed');
      return fulfill({ notifications: visible, count: visible.length, unread_count: visible.filter(item=>!item.read&&item.status!=='read').length, delivery_history: [], policy: {} });
    }
    if (path === '/api/v1/notifications/config' && request.method() === 'PUT') { const body=request.postDataJSON();notificationPolicySaves.push(body);return fulfill({policy:body,channels:{in_app:{enabled:true,state:'available',status:'available'},email:{enabled:body.channels.email.enabled,state:body.channels.email.enabled?'ready':'not_configured',status:body.channels.email.enabled?'ready':'not_configured'},pagerduty:{enabled:body.channels.pagerduty.enabled,state:body.channels.pagerduty.enabled?'ready':'not_configured',status:body.channels.pagerduty.enabled?'ready':'not_configured'}}}); }
    if (path === '/api/v1/notifications/config') return fulfill({ policy: { channels: { in_app: { enabled: true }, email: { enabled: false, server_env: 'NLI_SMTP_SERVER', sender_env: 'NLI_NOTIFICATION_EMAIL_FROM', recipient_env: 'NLI_NOTIFICATION_EMAIL_RECIPIENTS', username_env:'NLI_SMTP_USERNAME', password_env:'NLI_SMTP_PASSWORD' }, pagerduty: { enabled: false, secret_env: 'NLI_PAGERDUTY_ROUTING_KEY' } }, admin_fallback: true, recipients: ['admin'], events: ['approval','quarantine','sync','system','delivery','security'] }, channels: { in_app: { state: 'available', status: 'available' }, email: { state: 'not_configured', status: 'not_configured' }, pagerduty: { state: 'not_configured', status: 'not_configured' } } });
    if (path.match(/^\/api\/v1\/notifications\/audit-[^/]+$/) && ['POST','DELETE'].includes(request.method())) {
      const sequence=Number(path.split('audit-').at(-1)),action=request.method()==='DELETE'?'dismissed':request.postDataJSON().action;
      notificationActions.push({sequence,action,method:request.method()});
      const item=currentControllerNotifications.find(note=>Number(note.sequence)===sequence);
      if(item){item.status=action==='snooze'?'snoozed':action;item.read=action==='read';}
      return fulfill({ status: action });
    }
    if (path === '/api/v1/system') return fulfill({ version: '20.0.0', health: 'ok', storage: { mode: 'supabase', status: 'connected' }, security: { audit_chain: 'ok' } });
    return fulfill({ error: `Unmocked API: ${path}` }, 500);
  });
  return { observed, uploads, assuranceRequests, assuranceCorpora, assurancePlans, assuranceRuns, outputSaves, inputSaves, outputProfiles, inputProfiles, inputRolloutSaves, rolloutSaves, rolloutProposals, rolloutApprovalRequests, historyQueries, users, auditEvents, notificationPolicySaves, notificationActions, qrwParserActions, qrwReplays, qrwSyncVerifications, qrwRolloutRequests, qrwRecords:currentQRWRecords, qrwHistory:currentQRWHistory };
}

async function login(page) {
  await page.goto(origin);
  await expect(page.getByRole('button', { name: 'Sign in securely' })).toBeEnabled();
  await page.getByLabel('Username').fill('test-operator');
  await page.getByLabel('Password').fill('test-only-password');
  await page.getByRole('button', { name: 'Sign in securely' }).click();
  await expect(page.getByTestId('nav-operations')).toBeVisible();
  await expect(page.locator('.loading-state')).toHaveCount(0);
}

const designStyles=['skeuomorphism','minimalism','glassmorphism','neumorphism','claymorphism','bento-grid','retro-y2k','liquid-glass'];

test('R9 historical dashboard counts follow range and client and reject late responses',async({page})=>{
 await mockController(page);let pending;
 const history=count=>({from:'2026-11-06T00:00:00Z',to:'2026-11-06T01:00:00Z',points:[{timestamp:'2026-11-06T00:59:00Z'}],canonical_history:{points:[{timestamp:'2026-11-06T00:59:00Z',read_delta:String(count),parsed_delta:String(count-2),quarantine_delta:'2',errors:'0'}]}});
 await page.route('**/api/v1/metrics/timeseries?**',async route=>{const query=new URL(route.request().url()).searchParams;if(query.get('range')==='6h'){pending=route;return;}const count=query.get('node_id')?17:query.get('range')==='24h'?49:9;await route.fulfill({contentType:'application/json',body:JSON.stringify(history(count))});});
 await login(page);const input=page.getByTestId('live-dashboard-kpis').locator('.dashboard-kpi strong').nth(0);await expect(input).toHaveText('49');await page.locator('#dashboard-range').selectOption('1h');await expect(input).toHaveText('9');
 await page.locator('#dashboard-range').selectOption('6h');await expect.poll(()=>!!pending).toBe(true);await page.locator('#dashboard-range').selectOption('24h');await expect(input).toHaveText('49');await pending.fulfill({contentType:'application/json',body:JSON.stringify(history(999))});await expect(page.locator('#dashboard-range')).toHaveValue('24h');await expect(input).toHaveText('49');
 await page.locator('#dashboard-client').selectOption('node-1');await expect(input).toHaveText('17');await expect(page.locator('#dashboard-client')).toHaveValue('node-1');
});

test('R9 dashboard adds post-checkpoint live counts without replacing a focused range control',async({page})=>{
 await mockController(page);await page.addInitScript(()=>{window.__r9Sources=[];window.EventSource=class extends EventTarget{constructor(){super();window.__r9Sources.push(this);setTimeout(()=>this.onopen?.(),0);}close(){}emit(value){this.dispatchEvent(new MessageEvent('sync_delta',{data:JSON.stringify(value)}));}};});
 const observed=new Date(Date.now()-1000).toISOString();await page.route('**/api/v1/metrics/timeseries?**',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({from:'2026-11-06T00:00:00Z',to:'2026-11-07T00:00:00Z',points:[{timestamp:'2026-11-06T00:59:00Z'}],canonical_history:{observed_at:observed,counter_baseline:{['node-1\0/var/log/mail.log']:{generation:'gen-1',read:'120',parsed:'116',quarantine:'4',errors:'0'}},points:[{timestamp:'2026-11-06T00:59:00Z',read_delta:'49',parsed_delta:'47',quarantine_delta:'2',errors:'0'}]}})}));
 await login(page);const input=page.getByTestId('live-dashboard-kpis').locator('.dashboard-kpi strong').nth(0);await expect(input).toHaveText('49');await page.locator('#dashboard-range').focus();await page.locator('#dashboard-range').evaluate(node=>{window.__r9Range=node;window.__r9Root=document.querySelector('#page').firstElementChild;});
 await page.evaluate(()=>window.__r9Sources.at(-1).emit({schema_version:'nli.sync.v1',revision:8,generated_at:new Date().toISOString(),source:{node_id:'node-1',source:'/var/log/mail.log',process_generation:'gen-1',sequence:8,counters:{nli_source_records_read_total:125,nli_parser_records_succeeded_total:120,nli_normalized_records_total:120,nli_quarantined_records_total:5,nli_parser_records_failed_total:0},controller_received_at:new Date().toISOString()},aggregate:{nli_source_records_read_total:125,nli_normalized_records_total:120,nli_quarantined_records_total:5}}));
 await expect(input).toHaveText('54');await expect(page.getByTestId('live-dashboard-kpis').locator('.dashboard-kpi strong').nth(1)).toHaveText('51');await expect.poll(()=>page.evaluate(()=>({focused:document.activeElement===window.__r9Range,connected:window.__r9Range.isConnected,root:document.querySelector('#page').firstElementChild===window.__r9Root}))).toEqual({focused:true,connected:true,root:true});
});

test('R9 dynamic tail keeps future keys in verified JSON and out of fixed projection',async({page})=>{
 await mockController(page);let plan;
 await page.route('**/api/v1/assurance/mapping-preview',async route=>{plan=route.request().postDataJSON().plan;await route.fulfill({contentType:'application/json',body:JSON.stringify({passed:true,actual:{status:'failed',reason:'notneeded',server:'cloud',future_key:'preserved'},errors:[]})});});
 await login(page);await page.getByTestId('nav-field-mapping').click();await page.locator('#nfi-raw').fill('20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud');
 await page.getByRole('button',{name:'Add dynamic KV tail',exact:true}).click();
 await expect(page.locator('[data-qrw-row]')).toHaveCount(1);await page.getByRole('button',{name:'Preview mapping',exact:true}).click();await expect(page.locator('#nfi-live-json')).toContainText('future_key');
 await page.locator('[data-action=nfi-generate]').first().click();await expect(page.locator('#nfi-parser-json')).not.toHaveValue('{}');const parser=JSON.parse(await page.locator('#nfi-parser-json').inputValue());
 expect(plan.fields[0]).toMatchObject({method:'kv_all',source:'9',source_end:'',unit:'token',output_policy:'flatten',assignment_separators:['='],require_complete:true});expect(parser.fields).toEqual([]);expect(parser.field_mapping).toEqual({});expect(parser.output_schema.fields).toEqual([]);expect(parser.mapping_plan.fields[0]).toMatchObject({method:'kv_all',output_policy:'flatten'});expect(JSON.parse(await page.locator('#nfi-cases-json').inputValue())[0].desired.future_key).toBe('preserved');
});

test('bracket preset previews three literal edits and supports hash replacement', async ({page}) => {
  await mockController(page);
  let plan;
  await page.route('**/api/v1/assurance/mapping-preview',async route=>{
    plan=route.request().postDataJSON().plan;
    await route.fulfill({json:{passed:true,actual:{status:'failed',reason:'notneeded',server:'cloud'},errors:[]}});
  });
  await login(page);
  await page.getByTestId('nav-field-mapping').click();
  await page.locator('#nfi-raw').fill('20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg [status:failed][reason;notneeded][server=cloud#]');
  await page.getByRole('button',{name:'Add dynamic KV tail',exact:true}).click();
  const row=page.locator('[data-qrw-row]');
  await expect(row.getByText('Extraction method',{exact:true})).toBeVisible();
  await row.getByText('Replace and split KV input',{exact:true}).click();
  await row.getByRole('button',{name:'Set up adjacent [key:value] pairs',exact:true}).click();
  await row.getByRole('button',{name:'Add literal replacement',exact:true}).click();
  await row.getByLabel('Find literal text',{exact:true}).last().fill('#');
  await row.getByRole('button',{name:'Preview',exact:true}).click();
  await expect.poll(()=>plan.fields[0]).toMatchObject({
    method:'kv_all',source:'9',kv_profile:'configured',require_complete:true,kv_pair_separators:['|'],
    assignment_separators:[':', ';', '='],
    kv_input_edits:[{action:'replace',find:'][',replace:'|'},{action:'strip_prefix',find:'['},
      {action:'strip_suffix',find:']'},{action:'replace',find:'#',replace:''}]
  });
});

test('R10 KV row preview reuses the tail and applies an inspected grammar', async ({page}) => {
  await mockController(page);
  const raw='20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg |status:failed|reason:notneeded|server:cloud';
  const actual={status:'failed',reason:'notneeded',server:'cloud'};
  let inspected, plan, inspectionCount=0;
  await page.route('**/api/v1/assurance/kv-inspect',async route=>{
    inspected=route.request().postDataJSON();
    inspectionCount++;
    await route.fulfill({json:{profiles:[{
      assignment_separators:[':'],pair_delimiters:'|',
      observations:[{key:'status',value:'failed',start:86,end:99}]
    }]}});
  });
  await page.route('**/api/v1/assurance/mapping-preview',async route=>{
    plan=route.request().postDataJSON().plan;
    await route.fulfill({json:{passed:true,actual,errors:[]}});
  });
  await login(page);
  await page.getByTestId('nav-field-mapping').click();
  await page.locator('#nfi-raw').fill(raw);
  const add=page.getByRole('button',{name:'Add dynamic KV tail',exact:true});
  await add.click();
  const row=page.locator('[data-qrw-row]');
  await expect(row).toHaveCount(1);
  expect(inspected).toMatchObject({raw,field:{method:'kv_all',source:'9',end_of_line:true,kv_profile:'auto'}});
  await expect(page.locator('#nfi-live-json')).toContainText('notneeded');
  await row.getByRole('button',{name:'Preview',exact:true}).click();
  await expect(row).toHaveCount(1);
  expect(inspectionCount).toBe(1);
  await add.click();
  await expect(row).toHaveCount(1);
  expect(inspectionCount).toBe(2);
  await expect(page.locator('#kvie-inspector')).toContainText('status = failed');
  await page.getByRole('button',{name:'Apply grammar 1',exact:true}).click();
  await expect(row.locator('[data-qrw-key=kv_profile]')).toHaveValue('configured');
  await expect(row.locator('[data-qrw-key=assignment_separators]')).toHaveValue('[":"]');
  await expect.poll(()=>plan.fields[0]).toMatchObject({kv_profile:'configured',assignment_separators:[':'],pair_delimiters:'|'});
  await row.getByText('Replace and split KV input',{exact:true}).click();
  await row.locator('[data-qrw-key=kv_pair_separators]').fill('["||"]');
  await expect.poll(()=>plan.fields[0].kv_pair_separators).toEqual(['||']);
  await row.getByText('Edit individual KV output fields',{exact:true}).click();
  await row.getByRole('button',{name:'Add key edit',exact:true}).click();
  await row.getByLabel('Output key',{exact:true}).fill('reason');
  await row.getByLabel('Output key',{exact:true}).blur();
  await row.getByLabel('Key edit action',{exact:true}).selectOption('replace');
  await row.getByLabel('Replacement text',{exact:true}).fill('redacted');
  await row.getByLabel('Replacement text',{exact:true}).blur();
  await expect.poll(()=>plan.fields[0].key_value_edits).toEqual([{key:'reason',action:'replace',value:'redacted'}]);
  await row.getByLabel('Key edit action',{exact:true}).selectOption('blank');
  await expect(row.getByLabel('Replacement text',{exact:true})).toHaveCount(0);
  await expect.poll(()=>plan.fields[0].key_value_edits).toEqual([{key:'reason',action:'blank'}]);
  await row.getByRole('button',{name:'Remove key edit 1',exact:true}).click();
  await expect.poll(()=>plan.fields[0].key_value_edits).toBeNull();

});

test('KV mode switch clears named-only state and shows native evidence coverage',async({page})=>{
  await mockController(page);
  const raw='prefix status:failed,reason:notneeded,server:cloud';
  const actual={status:'failed',reason:'notneeded',server:'cloud'};
  const submitted=[];
  await page.route('**/api/v1/assurance/mapping-preview',async route=>{
    const field=route.request().postDataJSON().plan.fields[0];
    submitted.push(field);
    await route.fulfill({json:{passed:true,actual,errors:[],assessment:{
      source_evidence_pct:100,observed_fields:3,output_fields:3,structured_coverage_passed:true
    }}});
  });
  await login(page);
  await page.getByTestId('nav-field-mapping').click();
  await page.locator('#nfi-raw').fill(raw);
  await page.getByText('Import or edit a mapping plan',{exact:true}).click();
  const importField=async field=>{
    await page.locator('#nfi-plan-import').fill(JSON.stringify({version:1,fields:[field]}));
    await page.getByRole('button',{name:'Import mapping',exact:true}).click();
  };
  const named={name:'tail',method:'key_value',type:'string',policy:'required',source:'status',kv_profile:'configured',
    kv_start:'2',unit:'token',assignment_separators:[':'],pair_delimiters:','};
  await importField(named);
  const row=page.locator('[data-qrw-row]');
  await row.locator('[data-qrw-key=method]').selectOption('kv_eol');
  await expect(row.locator('[data-qrw-key=source]')).toHaveValue('2');
  await page.getByRole('button',{name:'Preview mapping',exact:true}).click();
  expect(submitted.at(-1)).toMatchObject({method:'kv_all',source:'2',end_of_line:true,assignment_separators:[':'],pair_delimiters:','});
  expect(submitted.at(-1)).not.toHaveProperty('kv_start');
  await expect(page.locator('#nfi-live-state')).toContainText('Source evidence coverage: 100% (3/3');
  await importField({...named,method:'kv_all',source:'2',type:'object',end_of_line:true});
  await page.getByRole('button',{name:'Preview mapping',exact:true}).click();
  expect(submitted.at(-1)).not.toHaveProperty('kv_start');
});

test('R9 catalog exposes actual inventory and restores archived parsers as inactive drafts',async({page})=>{
 await mockController(page);let restored;
 await page.route('**/api/v1/parser-packs/deleted',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({deleted_parsers:[{id:'retired',name:'Retired parser',original_state:'published',restorable:true}]})}));
 await page.route('**/api/v1/parser-sync/runtime',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({controller:{status:'materialized',bundle_path:'/var/lib/nli/parser-sync/bundle.json',parser_directory:'/var/lib/nli/parser-sync/parsers',parsers:[{id:'live',path:'/var/lib/nli/parser-sync/parsers/live.json',sha256:'actual-sha'}]},clients:[{node_id:'node-1',status:'pending',exact_acknowledgement:false,blocking_code:'bundle_ack_pending',next_action:'Approve the removal rollout'}]})}));
 await page.route('**/api/v1/parser-packs/deleted/retired/restore',async route=>{restored=route.request().postDataJSON();await route.fulfill({status:201,contentType:'application/json',body:JSON.stringify({id:restored.id,state:'draft',activated:false})});});
 await login(page);await page.getByTestId('nav-parsers').click();await expect(page.locator('#page')).toContainText('/var/lib/nli/parser-sync/parsers/live.json');await expect(page.locator('#page')).toContainText('bundle_ack_pending');page.once('dialog',dialog=>dialog.accept('retired-restored'));await page.getByRole('button',{name:'Restore as draft',exact:true}).click();await expect.poll(()=>restored?.id).toBe('retired-restored');expect(restored).not.toHaveProperty('activated');
});

for(const passes of [true,false])test(`R9 combined wizard generation validates static and Key value EOL together (${passes?'passed':'blocked'})`,async({page})=>{
 const raw='mailnode status=failed:server=cloud';const mock=await mockController(page,{quarantineRaw:raw,assurancePassed:passes});let nativePlan;
 await page.route('**/api/v1/assurance/mapping-preview',async route=>{nativePlan=route.request().postDataJSON().plan;await route.fulfill({contentType:'application/json',body:JSON.stringify({passed:true,actual:{hostname:'mailnode',status:'failed',server:'cloud'},errors:[]})});});
 await login(page);await page.getByTestId('nav-quarantine-wizard').click();await page.getByRole('button',{name:'Inspect',exact:true}).click();await page.getByRole('button',{name:/Review desired schema/}).click();await page.getByText('Import or edit a mapping plan',{exact:true}).click();await page.locator('#nfi-plan-import').fill(JSON.stringify({version:1,fields:[{name:'hostname',method:'position',source:'1',type:'string',policy:'required'},{name:'tail_fields',method:'kv_eol',source:'2',unit:'token',type:'object',policy:'required',output_policy:'flatten',assignment_separators:['='],pair_delimiters:':;| \t\r\n',require_complete:true}]}));await page.getByRole('button',{name:'Import mapping',exact:true}).click();
 const method=page.locator('[data-qrw-key=method]').nth(1);await expect(method.locator('option:checked')).toHaveText('Key value EOL — End of line');await expect(method.locator('option[value=key_value]')).toHaveCount(1);
 await page.getByRole('button',{name:'Generate + validate complete parser pack',exact:true}).click();await expect(page.locator('#gprs-parser-helper-form')).toBeVisible();expect(nativePlan.fields[1]).toMatchObject({method:'kv_all',end_of_line:true,source:'2',output_policy:'flatten'});const parser=JSON.parse(await page.locator('#gprs-parser-helper-form [name=parser]').inputValue());expect(parser.fields.map(field=>field.name)).toEqual(['hostname']);expect(parser.mapping_plan.fields).toHaveLength(2);await expect(page.locator('#gprs-parser-helper-form [data-parser-helper-status]')).toContainText(passes?'Complete parser validation passed':'0/1 passed');expect(mock.observed).toContain('POST /api/v1/assurance/parser-preview');expect(mock.observed.some(item=>item.endsWith('/publish'))).toBe(false);
});

test('R9 native merge removes consumed output schema fields and conditional non-consumption preserves them',async({page})=>{
 await mockController(page);let actual={merged:'mailnode/smtp'};
 await page.route('**/api/v1/assurance/mapping-preview',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({passed:true,actual,errors:[]})}));
 await login(page);await page.getByTestId('nav-field-mapping').click();await page.locator('#nfi-raw').fill('mailnode smtp');await page.getByText('Import or edit a mapping plan',{exact:true}).click();
 const fields=[{name:'host',method:'position',source:'1',type:'string',policy:'required'},{name:'service',method:'position',source:'2',type:'string',policy:'required'},{name:'merged',method:'constant',default_value:'',type:'string',policy:'required',operations:[{op:'merge',fields:['host','service'],separator:'/',remove_sources:true}]}];
 await page.locator('#nfi-plan-import').fill(JSON.stringify({version:1,fields}));await page.getByRole('button',{name:'Import mapping',exact:true}).click();await page.getByRole('button',{name:'Preview mapping',exact:true}).click();await page.locator('[data-action=nfi-generate]').first().click();let parser=JSON.parse(await page.locator('#nfi-parser-json').inputValue());expect(parser.fields.map(field=>field.name)).toEqual(['merged']);expect(parser.mapping_plan.fields).toHaveLength(3);
 fields[2].operations[0]={op:'conditional_merge',fields:['host','service'],separator:'/',remove_sources:true,condition_field:'host',condition_operator:'equals',condition_value:'different',else_value:'unchanged'};actual={host:'mailnode',service:'smtp',merged:'unchanged'};await page.locator('#nfi-plan-import').fill(JSON.stringify({version:1,fields}));await page.getByRole('button',{name:'Import mapping',exact:true}).click();await page.getByRole('button',{name:'Preview mapping',exact:true}).click();await page.locator('[data-action=nfi-generate]').first().click();parser=JSON.parse(await page.locator('#nfi-parser-json').inputValue());expect(parser.fields.map(field=>field.name)).toEqual(['host','service','merged']);expect(parser.fields.find(field=>field.name==='host').required).toBe(false);expect(parser.output_schema.fields.find(field=>field.name==='host').policy).toBe('optional');
});

test('R9 a delayed wizard load cannot replace a newer quarantine inbox navigation',async({page})=>{
 await mockController(page);let pending;
 await page.route('**/api/v1/assurance/cases',route=>{pending=route;});await login(page);await page.getByTestId('nav-quarantine-wizard').click();await expect.poll(()=>!!pending).toBe(true);await page.getByTestId('nav-unknowns').click();await expect(page.locator('#page')).toHaveAttribute('data-page','unknowns');await pending.fulfill({contentType:'application/json',body:'{"cases":[]}'});await expect(page.locator('.card.metric').filter({hasText:'Manual attention'}).locator('.metric-value')).toHaveText('0');await expect(page.locator('#page')).toHaveAttribute('data-page','unknowns');await expect(page.locator('#page h1').filter({hasText:'Quarantine Resolution Wizard'})).toHaveCount(0);
});

test('R8 Design Studio previews, applies, reverts and persists appearance without replacing a working draft',async({page})=>{
 await mockController(page);await login(page);await page.getByTestId('nav-field-mapping').click();await page.locator('#nfi-raw').fill('mailnode smtp 007');await page.locator('#nfi-raw').evaluate(node=>node.dataset.identity='preserved');
 await page.getByRole('button',{name:'Appearance',exact:true}).click();const studio=page.locator('#design-studio');await expect(studio).toBeVisible();await expect(studio.locator('[data-option]')).toHaveCount(8);
 await studio.locator('[data-design-action=preview][data-style=glassmorphism]').click();await expect(page.locator('html')).toHaveAttribute('data-design','glassmorphism');expect(await page.evaluate(()=>localStorage.getItem('nli.appearance.v1'))).toBe(null);await studio.getByRole('button',{name:'Cancel preview',exact:true}).click();await expect(page.locator('html')).toHaveAttribute('data-design','original');
 for(const style of designStyles){await studio.locator(`[data-design-action=apply][data-style=${style}]`).click();await expect(page.locator('html')).toHaveAttribute('data-design',style);await studio.locator('#design-mode').selectOption('light');await studio.locator('#design-accent').selectOption('teal');await studio.getByRole('button',{name:'Apply and save preferences',exact:true}).click();await expect(page.locator('html')).toHaveAttribute('data-color-mode','light');expect(JSON.parse(await page.evaluate(()=>localStorage.getItem('nli.appearance.v1')))).toMatchObject({style,mode:'light',accent:'teal'});await studio.locator('#design-mode').selectOption('dark');}
 await studio.getByRole('button',{name:'Apply and save preferences',exact:true}).click();await studio.getByRole('button',{name:'Revert to previous design',exact:true}).click();await expect(page.locator('html')).toHaveAttribute('data-color-mode','light');await page.keyboard.press('Escape');await expect(studio).toBeHidden();await expect(page.locator('#nfi-raw')).toHaveValue('mailnode smtp 007');await expect(page.locator('#nfi-raw')).toHaveAttribute('data-identity','preserved');await expect(page.getByRole('button',{name:'Appearance',exact:true})).toBeFocused();
 await page.reload();await expect(page.locator('html')).toHaveAttribute('data-design','liquid-glass');await expect(page.locator('html')).toHaveAttribute('data-color-mode','light');await page.getByRole('button',{name:'Appearance',exact:true}).click();await studio.getByRole('button',{name:'Restore NLI default',exact:true}).click();await expect(page.locator('html')).toHaveAttribute('data-design','original');await expect(page.locator('html')).toHaveAttribute('data-color-mode','dark');
});

test('R8 eight designs cover every route in dark desktop and light mobile layouts',async({page})=>{
 test.setTimeout(240000);await mockController(page);await login(page);const errors=[];page.on('pageerror',error=>errors.push(error.message));
 for(const style of designStyles){await page.evaluate(style=>window.NLIDesign.provider.apply({style,mode:'dark',accent:'blue'}),style);for(const tab of tabs){await page.getByTestId(`nav-${tab}`).click();await expect(page.locator('.error-state')).toHaveCount(0);await expect(page.locator('.loading-state')).toHaveCount(0);await expect(page.locator('#page')).toHaveAttribute('data-page',tab);}await page.setViewportSize({width:390,height:844});await page.evaluate(style=>window.NLIDesign.provider.apply({style,mode:'light',accent:'violet'}),style);for(const tab of tabs){await page.getByLabel('Toggle navigation').click();await page.getByTestId(`nav-${tab}`).click();await expect(page.locator('.error-state')).toHaveCount(0);await expect(page.locator('.loading-state')).toHaveCount(0);await expect(page.locator('#page')).toHaveAttribute('data-page',tab);if(['streams','outputs'].includes(tab)){await expect(page.locator('.record-accounting-help')).toContainText('File size is not a record count');await expect(page.locator('.record-accounting-help')).toContainText('nli-output-audit.py');}const overflow=await page.evaluate(()=>({width:innerWidth,scroll:document.documentElement.scrollWidth,offenders:[...document.querySelectorAll('body *')].map(node=>({tag:node.tagName,cls:node.className,right:node.getBoundingClientRect().right,width:node.getBoundingClientRect().width})).filter(item=>item.right>innerWidth+1&&item.width>0).slice(0,12)}));expect(overflow.scroll<=overflow.width,`${style}/${tab}: mobile overflow ${JSON.stringify(overflow)}`).toBe(true);}await page.setViewportSize({width:1280,height:720});}expect(errors).toEqual([]);
});

test('R8 all designs retain dense tablet forms, tables and Studio controls at768px',async({page})=>{
 test.setTimeout(90000);await page.setViewportSize({width:768,height:1024});await mockController(page);await login(page);
 for(const style of designStyles)for(const mode of ['dark','light']){await page.evaluate(({style,mode})=>window.NLIDesign.provider.apply({style,mode,accent:'blue'}),{style,mode});for(const tab of ['monitoring','field-mapping','input-paths','outputs']){await page.getByLabel('Toggle navigation').click();await page.getByTestId(`nav-${tab}`).click();await expect(page.locator('#page')).toHaveAttribute('data-page',tab);await expect(page.locator('.error-state')).toHaveCount(0);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`${style}/${mode}/${tab} tablet overflow`).toBe(true);}await page.getByRole('button',{name:'Appearance',exact:true}).click();const studio=page.locator('#design-studio');await expect(studio).toBeVisible();expect(await studio.evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);await page.keyboard.press('Escape');}
});

test('R8 light chart series use distinct contrast-aware colours and preserve exact hover labels',async({page})=>{
 await mockController(page);await page.route('**/api/v1/metrics/timeseries?**',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({from:'2026-11-01T00:00:00Z',to:'2026-11-01T00:02:00Z',points:[{timestamp:'2026-11-01T00:00:00Z',received:12,parsed:10,quarantine:1,errors:1},{timestamp:'2026-11-01T00:01:00Z',received:14,parsed:12,quarantine:2,errors:0}],canonical_history:{bucket_width_seconds:60,points:[{timestamp:'2026-11-01T00:00:00Z',read_delta:'12',parsed_delta:'10',quarantine_delta:'1',errors:1},{timestamp:'2026-11-01T00:01:00Z',read_delta:'14',parsed_delta:'12',quarantine_delta:'2',errors:0}]}})}));await login(page);await expect(page.locator('.multi-chart polyline').first()).toBeVisible();
 for(const style of designStyles){await page.evaluate(style=>window.NLIDesign.provider.apply({style,mode:'light',accent:'teal'}),style);const evidence=await page.evaluate(()=>{const node=document.querySelector('.multi-chart'),tokens=window.NLIDesign.resolve(window.NLIDesign.provider.rendered).tokens,hex=value=>{const parts=value.match(/[\d.]+/g);return parts?'#'+parts.slice(0,3).map(n=>Number(n).toString(16).padStart(2,'0')).join(''):value;},colours=[...node.querySelectorAll('polyline')].map(line=>hex(getComputedStyle(line).stroke));return {colours,connected:node.isConnected,raw:[...node.querySelectorAll('polyline')].map(line=>({connected:line.isConnected,attr:line.getAttribute('stroke'),computed:getComputedStyle(line).stroke})),tokens,labels:[...node.querySelectorAll('.chart-point')].map(point=>point.getAttribute('aria-label'))};});expect(evidence.colours.every(value=>/^#[\da-f]{6}$/i.test(value)),JSON.stringify(evidence)).toBe(true);expect(new Set(evidence.colours).size).toBe(4);const ratios=evidence.colours.map(colour=>Math.min(...['panel','panel2','panel3'].map(surface=>designContrast(colour,evidence.tokens[surface]))));expect(ratios.every(ratio=>ratio>=3),style+JSON.stringify({colours:evidence.colours,ratios})).toBe(true);expect(evidence.labels.some(label=>label.includes('12'))).toBe(true);}
});

test('R8 appearance respects reduced motion, visible keyboard focus and unavailable storage',async({page})=>{
 await page.addInitScript(()=>{Storage.prototype.setItem=function(){throw new Error('Storage unavailable for test');};});await page.emulateMedia({reducedMotion:'reduce'});await mockController(page);await login(page);await page.getByRole('button',{name:'Appearance',exact:true}).click();const studio=page.locator('#design-studio');await studio.locator('[data-design-action=apply][data-style=neumorphism]').click();await expect(studio.locator('#design-status')).toContainText('storage is unavailable');await studio.getByLabel('Preview sample input',{exact:true}).focus();expect(await studio.getByLabel('Preview sample input',{exact:true}).evaluate(node=>getComputedStyle(node).outlineWidth)).not.toBe('0px');expect(await studio.getByLabel('Preview sample input',{exact:true}).evaluate(node=>getComputedStyle(node).transitionDuration)).toBe('0s');await page.keyboard.press('Escape');await expect(studio).toBeHidden();await expect(page.locator('html')).toHaveAttribute('data-design','neumorphism');
});

test('all workspaces render and audit is visible', async ({ page }) => {
  const { observed } = await mockController(page);
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await login(page);
  await expect(page.locator('.topbar-brand')).toContainText('Nexora Log Intelligence');
  await expect(page.locator('.topbar-brand')).toContainText('Infinite Logs. One Engine. Intelligent Parsing.');
  await expect(page.locator('#stream-state')).toHaveText(/Live sync|Reconnecting|Resynchronizing|Streaming unavailable/);
  for (const tab of tabs) {
    await page.getByTestId(`nav-${tab}`).click();
    await expect(page.locator('#page-title')).not.toBeEmpty();
    await expect(page.locator('.loading-state')).toHaveCount(0);
    await expect(page.locator('.error-state')).toHaveCount(0);
    await expect(page.locator('#page')).not.toContainText('undefined');
    if (tab === 'fleet') await expect(page.getByText('mail-node-1')).toBeVisible();
    if (tab === 'operations') await expect(page.locator('#page').getByRole('heading', { name: 'Command Center' })).toBeVisible();
    if (tab === 'api-monitor') await expect(page.getByRole('heading', { name: 'API monitor', exact: true })).toBeVisible();
    if (tab === 'unknowns') await expect(page.getByRole('heading', { name: 'Quarantine and reprocessing' })).toBeVisible();
    if (tab === 'parsers') await expect(page.getByText('SMTP parser', { exact: true }).first()).toBeVisible();
    if (tab === 'security') await expect(page.getByRole('heading', { name: 'Immutable audit chain' })).toBeVisible();
  }
  expect(observed.filter(item => item.endsWith('/api/v1/stream'))).toEqual([]);
  const fleetSyncRequests = observed.filter(item => item === 'GET /api/v1/fleet/sync');
  expect(fleetSyncRequests.length).toBeGreaterThanOrEqual(1);
  expect(fleetSyncRequests.length).toBeLessThanOrEqual(6);
  expect(errors).toEqual([]);
  await page.locator('#notification-bell').click();
  await expect(page.getByText('Notifications', { exact: true })).toBeVisible();
});

test('every visible inner tab activates without browser, request, or render errors', async ({ page }) => {
  test.setTimeout(120_000);
  const { observed } = await mockController(page);
  const errors = [];
  const failedResponses = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => { if (response.status() >= 500) failedResponses.push(`${response.status()} ${new URL(response.url()).pathname}`); });
  await login(page);
  for (const workspace of tabs) {
    await page.getByTestId(`nav-${workspace}`).click();
    await expect(page.locator('#page')).toHaveAttribute('data-page', workspace);
    await expect(page.locator('.loading-state')).toHaveCount(0);
    await expect(page.locator('.error-state')).toHaveCount(0);
    const innerTabNames = await page.locator('#page [role=tab]:visible').allTextContents();
    for (const rawName of innerTabNames) {
      const name = rawName.trim();
      const tab = page.locator('#page').getByRole('tab', { name, exact: true }).first();
      await expect(tab, `inner tab ${workspace} / ${name} remains mounted`).toBeVisible();
      await tab.click();
      await expect(tab).toHaveAttribute('aria-selected', 'true');
      await expect(page.locator('.error-state')).toHaveCount(0);
      await expect(page.locator('#page')).not.toContainText(/rows\.join is not a function|Request failed|NaN|undefined/);
    }
  }
  expect(errors).toEqual([]);
  expect(failedResponses).toEqual([]);
  expect(observed.length).toBeGreaterThan(40);
});

test('one canonical synchronization snapshot feeds Command Center, NHLE, NAPI and NAL', async ({ page }) => {
  await mockController(page);
  await login(page);
  await expect(page.getByTestId('live-dashboard-kpis').locator('.dashboard-kpi strong').first()).toHaveText('9,007,199,254,740,993');
  await expect(page.locator('.chart-legend').first()).toContainText('Input (records / bucket)');
  const firstPoint=page.locator('[data-chart-category="Input"]').first();
  await expect(firstPoint).toBeVisible();
  const readout=await firstPoint.evaluate(point=>{
    point.dispatchEvent(new FocusEvent('focusin',{bubbles:true}));
    return point.closest('.multi-chart')?.querySelector('.chart-hover-readout')?.textContent||'';
  });
  await expect.poll(async()=>readout.match(/Input .* records/)?readout:page.locator('[data-chart-category="Input"]').first().evaluate(point=>{point.dispatchEvent(new FocusEvent('focusin',{bubbles:true}));return point.closest('.multi-chart')?.querySelector('.chart-hover-readout')?.textContent||'';})).toMatch(/Input .* records/);

  await page.getByTestId('nav-sync-fabric').click();
  await expect(page.getByRole('heading',{name:'Unified Real-Time Synchronization Fabric'})).toBeVisible();
  await expect(page.getByText('nli.sync.v1')).toBeVisible();
  await page.getByRole('tab',{name:'Diagnostics'}).click();
  await expect(page.getByText('parser latency exceeded warning threshold')).toBeVisible();
  await expect(page.getByText('Inspect parser trace and queue saturation.')).toBeVisible();

  await page.getByTestId('nav-nhle-monitor').click();
  await expect(page.getByRole('heading',{name:'Nexora Hypersonic Log Engine'})).toBeVisible();
  await expect(page.locator('#page')).toContainText('120');
  await expect(page.locator('#page')).toContainText('90 lines/s');
  await page.getByRole('tab',{name:'Diagnostics'}).click();
  await expect(page.getByText('parser latency exceeded warning threshold')).toBeVisible();

  await page.getByTestId('nav-napi-monitor').click();
  await expect(page.getByRole('heading',{name:'Nexora Adaptive Parser Intelligence'})).toBeVisible();
  await expect(page.locator('#page')).toContainText('Pending drafts');
  await expect(page.locator('#page')).toContainText('3');

  await page.getByTestId('nav-nal-monitor').click();
  await expect(page.getByRole('heading',{name:'Nexora Assurance Lab'})).toBeVisible();
  await expect(page.locator('#page')).toContainText('Total runs');
  await expect(page.locator('#page')).toContainText('5');
  await page.getByTestId('nav-operations').click();
  await expect(page.locator('.throughput-gauges')).toContainText('90.0/s');
});

test('Monitoring exposes eleven canonical dashboards using only the shared recovery path', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  const before = observed.length;
  await page.getByTestId('nav-monitoring').click();
  await expect(page.getByRole('heading', { name: 'NLI Monitoring' })).toBeVisible();
  await expect(page.locator('.monitoring-tabs [role=tab]')).toHaveCount(11);
  await expect(page.locator('.monitoring-tabs [role=tab][aria-controls]')).toHaveCount(11);
  await expect(page.locator('[data-monitoring-panel=overview]')).toContainText('Ingestion');
  await expect(page.locator('[data-monitoring-panel=overview]')).toContainText('90 rec/s');
  await expect(page.locator('.monitoring-freshness')).toContainText('Revision 7');

  await page.getByRole('tab', { name: 'Pipeline' }).click();
  const validateRow=page.locator('[data-monitoring-panel=pipeline] tr').filter({hasText:'VALIDATE'});
  await expect(validateRow).toContainText('90 rec/s');
  await expect(validateRow).toContainText('87 rec/s');
  await expect(validateRow).toContainText('3 rec/s failed validation');
  await expect(validateRow).not.toContainText('NOT REPORTED');

  await page.getByRole('tab', { name: 'Latency & Synchronization' }).click();
  await expect(page.locator('[data-monitoring-panel=latency]')).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Latency & Synchronization' })).toHaveAttribute('aria-selected','true');
  await expect(page.locator('[data-monitoring-panel=latency]')).toContainText('parser latency exceeded warning threshold');
  await expect(page.locator('[data-monitoring-panel=latency]')).toContainText('Inspect parser trace and queue saturation.');

  await page.getByRole('tab', { name: 'Infrastructure' }).click();
  await expect(page.locator('[data-monitoring-panel=infrastructure]')).toContainText('Client CPU avg18%');
  await expect(page.locator('[data-monitoring-panel=infrastructure]')).toContainText('File descriptors21');
  await expect(page.locator('.filter-contract-note')).toContainText('Component, destination and environment labels are not included');

  await page.getByRole('tab', { name: 'Destinations' }).click();
  await expect(page.locator('[data-monitoring-panel=destinations]')).toContainText('Loki');
  await expect(page.locator('[data-monitoring-panel=destinations]')).toContainText('delivered: 116');
  await page.getByRole('tab', { name: 'Queues / Replay / Quarantine' }).click();
  await expect(page.locator('[data-monitoring-panel=queues]')).toContainText('Oldest pending45 s');
  await page.getByRole('tab', { name: 'Assurance & Release Health' }).click();
  await expect(page.locator('[data-monitoring-panel=assurance]')).toContainText('Release readinessready');
  const newRequests = observed.slice(before).filter(item => !item.endsWith('/api/v1/sync/stream'));
  const allowed=['GET /api/v1/sync/snapshot','GET /api/v1/metrics/timeseries','GET /api/v1/navigation/counts','GET /api/v1/notifications','POST /api/v1/communications/browser-heartbeat'];
  expect(newRequests.filter(item=>!allowed.includes(item))).toEqual([]);
});

test('diagnostics span the dashboard grid and remain usable at narrow widths', async ({ page }) => {
  await mockController(page);
  await login(page);
  const diagnostics=page.locator('.diagnostic-panel').last();
  const desktop=await diagnostics.boundingBox();
  expect(desktop?.width).toBeGreaterThan(700);
  await page.setViewportSize({width:390,height:844});
  const mobile=await diagnostics.boundingBox();
  expect(mobile?.width).toBeGreaterThan(300);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await expect(diagnostics.locator('.table-wrap')).toBeVisible();
});

test('empty telemetry uses reported state and never renders NaN', async ({ page }) => {
  await mockController(page);
  await page.route(`${origin}/api/v1/sync/snapshot`, route => route.fulfill({
    status:200,
    headers:{'content-type':'application/json','x-nli-controller':'v21','x-nli-upstream':'controller'},
    body:JSON.stringify({schema_version:'nli.sync.v1',revision:0,generated_at:new Date().toISOString(),target_interval_ms:1000,aggregate:{},sources:[],diagnostics:{issues:[]},subsystems:{}}),
  }));
  await login(page);
  await expect(page.locator('.diagnostic-panel')).toContainText('No active latency or error diagnostics');
  await expect(page.locator('.dashboard-kpi-strip').first()).toContainText('—');
  await expect(page.locator('#page')).not.toContainText('NaN');
  await page.setViewportSize({width:390,height:844});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('NUPIRE workspace displays an explicit empty candidate state', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-nupire').click();
  await expect(page.getByRole('heading', { name: 'NUPIRE / EGPE workspace' })).toBeVisible();
  await expect(page.getByText('No NUPIRE candidates are persisted yet.')).toBeVisible();
  await expect(page.getByText('No candidates')).toHaveCount(5);
});

test('NUPIRE candidate registry renders non-empty controller results', async ({ page }) => {
  await mockController(page, { nupireCandidates: [{ id: 'candidate-structured-mail', state: 'evaluated', digest: 'a'.repeat(64), metrics: { samples: 12, exact_matches: 12 }, updated_at: '2026-10-26T12:00:00Z' }] });
  await login(page);
  await page.getByTestId('nav-nupire').click();
  await expect(page.getByRole('button', { name: 'candidate-structured-mail' })).toBeVisible();
  await expect(page.locator('#page')).toContainText('12 exact / 12 samples');
  await expect(page.locator('.error-state')).toHaveCount(0);
});

test('navigation leads with Command Center, Assurance Lab, approvals and notifications', async ({ page }) => {
  await mockController(page);
  await login(page);
  const ids=await page.locator('.nav-item').evaluateAll(items=>items.map(item=>item.dataset.page));
  expect(ids.slice(0,24)).toEqual(['operations','monitoring','streams','assurance','quarantine-wizard','upes','rollouts','parser-sync','artifacts','parser-workspace','nupire','field-integrity','ai','sample-history','header-templates','kv-templates','parsers','reference-ai','building-blocks','field-mapping','studio','regex','knownkeys','drift']);
  expect(ids[24]).toBe('notifications');
  const input=ids.indexOf('input-paths'),output=ids.indexOf('outputs'),parsing=ids.indexOf('kv');
  expect(ids[input+1]).toBe('migration');
  expect(ids[input+2]).toBe('outputs');
  expect(ids[output+1]).toBe('metadata');
  expect(ids[output+2]).toBe('unknowns');
  expect(ids[output+3]).toBe('performance');
  expect(ids[parsing]).toBe('kv');
  expect(ids.indexOf('parsers')).toBeLessThan(parsing);
  expect(ids.indexOf('parsers')).toBe(ids.indexOf('kv-templates')+1);
  expect(ids.indexOf('engine-network')).toBeGreaterThan(ids.indexOf('drift'));
});

test('UPES observatory uses controller failure and candidate data with explicit unavailable views', async ({ page }) => {
  const mock = await mockController(page, { upesSnapshot: { status: 'live', clients_total: 1, clients_reporting: 1, unique_failure_signatures: 2, observed_failures: 7, discovery_queued: 2, discovery_dropped: 0, discovery_pending: 1 } });
  await login(page);
  await page.getByTestId('nav-upes').click();
  await expect(page.getByRole('heading', { name: 'Unstoppable Parsing Engine Support' })).toBeVisible();
  await expect(page.getByText('q-1', { exact: true }).first()).toBeVisible();
  await expect(page.locator('[data-subsystem-panel=failures]')).toContainText('Observed failures7');
  expect(mock.observed).toContain('GET /api/v1/sync/snapshot');
  expect(mock.observed).not.toContain('GET /api/v1/upes/snapshot');
  await expect(page.getByRole('tab', { name: 'Knowledge Base' })).toBeVisible();
  await page.getByRole('tab', { name: 'Discovery Jobs' }).click();
  await expect(page.locator('[data-subsystem-panel=jobs]')).toContainText('"observed_failures": 7');
  await page.getByRole('tab', { name: 'Knowledge Base' }).click();
  await expect(page.getByText('Knowledge base telemetry unavailable.')).toBeVisible();
});

test('Guided Parser Repair Studio inspects retained failures, uses AI only after failure, and opens the candidate editor', async ({ page }) => {
  const mock = await mockController(page, { assurancePassed: false });
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab', { name: 'Guided Parser Repair Studio' }).click();
  await expect(page.getByText('q-1', { exact: true }).first()).toBeVisible();
  await expect(page.locator('.quarantine-wizard [data-wizard-step]')).toHaveCount(9);
  await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await expect(page.locator('#gprs-preview-form [name=manual_field_name]')).toBeVisible();
  await expect(page.locator('#gprs-preview-form [name=manual_field_index]')).toBeVisible();
  await expect(page.locator('#gprs-preview-form [name=manual_delimiter]')).toBeVisible();
  await expect(page.locator('#gprs-preview-form [name=manual_template]')).toBeVisible();
  await expect(page.locator('#gprs-preview-form [name=manual_regex]')).toBeVisible();
  await page.locator('[data-gprs-record="q-1"]').check();
  const desired = page.locator('#gprs-preview-form [name=gprs_desired]');
  await page.locator('#gprs-preview-form [name=manual_field_names]').fill('user');
  await page.getByRole('button',{name:/Create mappings from my field names/}).click();
  await page.locator('[data-wizard-step="2"] button').click();
  await desired.fill('{"user":"alice"}');
  await page.locator('#gprs-preview-form [name=mapping]').fill('{"username":"user"}');
  await page.locator('[data-wizard-step="5"] button').click();
  await page.getByRole('button', { name: /Prepare candidate from evidence/ }).click();
  await page.locator('[data-wizard-step="6"] button').click();
  await page.getByRole('button', { name: /Run deterministic preview/ }).click();
  await expect(page.getByTestId('gprs-preview')).toContainText('records_tested');
  await expect(page.getByRole('button', { name: 'Ask AI to diagnose and propose a fix' })).toBeVisible();
  expect(mock.observed).toContain('POST /api/v1/quarantine/records/q-1/analyze');
  expect(mock.assuranceCorpora.at(-1).cases[0].desired).toEqual({ user: 'exact' });
  await page.getByRole('button', { name: 'Edit generated candidate' }).click();
  await expect(page.getByRole('tab', { name: 'Guided Parser Repair Studio' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('#assurance-panel-gprs #gprs-parser-helper-form')).toBeVisible();
  await expect(page.locator('#gprs-parser-helper-form [name=samples]')).toHaveValue('exact raw source line');
});

test('Quarantine inbox groups only controller-verified signatures and isolates unverified records', async ({ page }) => {
  await mockController(page, { qrwRecords: [
    { id:'q-unverified-1', node_id:'node-1', service:'smtp', first_seen:'2026-10-10T00:00:00Z', attempt_count:1, status:'waiting', error_codes:['no_match'], raw_available:true },
    { id:'q-unverified-2', node_id:'node-1', service:'smtp', first_seen:'2026-10-10T00:01:00Z', attempt_count:1, status:'waiting', error_codes:['no_match'], raw_available:true },
    { id:'q-verified-1', node_id:'node-1', service:'smtp', first_seen:'2026-10-10T00:02:00Z', attempt_count:1, status:'waiting', error_codes:['no_match'], raw_available:true, failure_signature:'smtp-shape-a' },
    { id:'q-verified-2', node_id:'node-2', service:'smtp', first_seen:'2026-10-10T00:03:00Z', attempt_count:1, status:'waiting', error_codes:['no_match'], raw_available:true, failure_signature:'smtp-shape-a' },
  ] });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.getByText('Individual record · shared cause unverified')).toHaveCount(2);
  await expect(page.getByText('2 signature-matched records')).toHaveCount(1);
});

test('Quarantine Resolution Wizard maps the required unstructured fixture without fabricating delimiter values and saves a governed candidate', async ({ page }) => {
  const raw='20250812 130035122+0100 aaa-gmailrgo009 mail 3609 0 140293768701696 Note;systemlogs_for_data';
  const mock=await mockController(page,{quarantineRaw:raw,assurancePassed:true});
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('#page').getByRole('heading',{name:'Quarantine Resolution Wizard',exact:true})).toBeVisible();
  await expect(page.locator('.quarantine-wizard [data-wizard-step]')).toHaveCount(9);
  await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await expect(page.locator('#gprs-preview-form [name=raw_evidence]')).toHaveValue(raw);
  await page.getByRole('button',{name:/Review desired schema/}).click();
  await page.getByRole('button',{name:'Suggest from raw log'}).click();
  await expect(page.locator('[data-qrw-key=name]')).toHaveCount(9);
  await expect(page.locator('[data-qrw-key=name]').nth(0)).toHaveValue('date');
  await expect(page.locator('[data-qrw-key=name]').nth(7)).toHaveValue('event_type');
  await expect(page.getByRole('button',{name:/Back to evidence/})).toBeVisible();
  await expect(page.getByRole('button',{name:/Generate parser candidate/})).toBeVisible();
  await expect(page.locator('[data-qrw-key=timestamp_format]')).toHaveCount(9);
  await expect(page.locator('[data-qrw-key=null_policy]')).toHaveCount(9);
  const editor=page.locator('#gprs-parser-helper-form');
  const initialParser=await editor.locator('[name=parser]').inputValue();
  await page.getByRole('button',{name:'Test all fields'}).click();
  await expect(page.locator('#qrw-field-status')).toContainText('2 field mapping error');
  await expect(page.locator('[data-wizard-step="5"] button')).toBeDisabled();
  await expect(editor.locator('[name=parser]')).toHaveValue(initialParser);
  const confirmations=page.getByLabel('Confirm delimiter interpretation');
  await expect(confirmations).toHaveCount(2);
  await confirmations.nth(0).check();
  await confirmations.nth(1).check();
  await expect(confirmations.nth(0)).toBeChecked();
  await expect(confirmations.nth(1)).toBeChecked();
  await page.getByRole('button',{name:'Test all fields'}).click();
  await expect(page.locator('#qrw-field-status')).toContainText('9 fields mapped');
  const desired=JSON.parse(await page.locator('#gprs-preview-form [name=gprs_desired]').inputValue());
  expect(desired).toMatchObject({date:'20250812',hostname:'aaa-gmailrgo009',service:'mail',pid:3609,status:0,minor_pid:140293768701696,event_type:'Note',event_name:'systemlogs_for_data'});
  await page.locator('[data-qrw-key=transform]').first().selectOption('lowercase');
  await page.getByRole('button',{name:'Test all fields'}).click();
  await page.locator('[data-wizard-step="5"] button').click();
	await page.getByRole('button',{name:/Prepare candidate from evidence/}).click();
  const transformed=JSON.parse(await editor.locator('[name=parser]').inputValue());
  expect(transformed.pattern_type).toBe('mapping');
  expect(transformed.mapping_plan.fields[0].transform).toBe('lowercase');
	await page.getByRole('button',{name:/Back to mapping/}).click();
  await page.locator('[data-qrw-key=transform]').first().selectOption('none');
  await page.getByRole('button',{name:'Test all fields'}).click();
  await page.locator('[data-wizard-step="5"] button').click();
  await page.getByRole('button',{name:/Prepare candidate from evidence/}).click();
  await expect(editor).toBeVisible();
  const parser=JSON.parse(await editor.locator('[name=parser]').inputValue());
  expect(parser.pattern_type).toBe('mapping');
  expect(parser.mapping_plan.fields.find(field=>field.name==='event_type')).toMatchObject({method:'delimiter',source:'8',delimiter:';',part:'1',confirmed:true});
  expect(parser.provenance).toMatchObject({kind:'quarantine_resolution_wizard',generated_from_verified_evidence:true});
  await editor.getByRole('button',{name:'Save review-gated draft'}).click();
  await expect.poll(()=>mock.observed.includes('POST /api/v1/assurance/parsers')).toBe(true);
});

test('FieldSense guided and expert modes preserve edits, show contextual help and keep live source traces',async({page})=>{
  await mockController(page);await login(page);await page.getByTestId('nav-field-mapping').click();
  await expect(page.locator('#nfi-guide-steps li')).toHaveCount(16);
  await page.getByRole('button',{name:'Manual Sense Mode',exact:true}).click();await page.locator('#nfi-raw').fill('mailnode smtp 007');
  await page.getByRole('button',{name:'Add Custom Field',exact:true}).click();
  const row=page.locator('[data-qrw-row]').first();await row.locator('[data-qrw-key=name]').fill('mail_service');await row.locator('[data-qrw-key=method]').selectOption('word');await row.locator('[data-qrw-key=source]').fill('2');await row.locator('[data-qrw-key=policy]').selectOption('required');
  await expect(page.locator('#nfi-live-json')).toContainText('smtp');await expect(page.locator('#nfi-live-state')).toContainText('local preview passed');
  expect(JSON.parse(await page.locator('#nfi-parser-json').inputValue()).mapping_plan.fields[0].name).toBe('mail_service');
  await row.locator('[data-qrw-key=source] + .nfi-info').click();await expect(page.locator('#nfi-help-dialog')).toBeVisible();await expect(page.locator('#nfi-help-content')).toContainText('1-based');await page.getByRole('button',{name:'Close help',exact:true}).click();
  await page.locator('[data-action=nfi-highlight]').first().click();await expect(page.locator('#nfi-raw-selection mark')).toHaveText('smtp');await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:test.info().outputPath('fieldsense-studio-guided.png'),fullPage:true});await page.setViewportSize({width:390,height:844});await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:test.info().outputPath('fieldsense-studio-mobile.png'),fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.setViewportSize({width:1280,height:720});
  await page.getByRole('button',{name:'Expert mode',exact:true}).click();await expect(page.locator('#nfi-guide-steps')).toBeHidden();await expect(row.locator('[data-qrw-key=name]')).toHaveValue('mail_service');await page.getByRole('button',{name:'Switch to Guided Builder',exact:true}).click();await expect(page.locator('#nfi-guide-steps')).toBeVisible();
  await page.getByRole('button',{name:'Suggest & Sense Mode',exact:true}).click();await page.getByRole('button',{name:'Suggest from raw log',exact:true}).click();await expect(page.locator('[data-action=nfi-suggestion-accept]')).not.toHaveCount(0);await page.locator('[data-action=nfi-suggestion-edit]').first().click();await expect(page.locator('[data-qrw-key=name]').first()).toBeFocused();await page.locator('[data-action=nfi-suggestion-regenerate]').first().click();await page.locator('[data-action=nfi-suggestion-accept]').first().click();await expect(page.locator('[data-action=nfi-suggestion-accept]').first()).toHaveText('Accepted');let requested=false;await page.route('**/api/v1/assurance/mapping-preview',async route=>{requested=true;await new Promise(resolve=>setTimeout(resolve,600));try{await route.fulfill({contentType:'application/json',body:JSON.stringify({passed:true,actual:{},fields:{},errors:[]})});}catch{}});await page.getByRole('button',{name:'Preview mapping',exact:true}).click();await expect.poll(()=>requested).toBe(true);await page.getByRole('button',{name:'Cancel analysis',exact:true}).click();await expect(page.locator('#nfi-live-state')).toContainText('Analysis cancelled');await expect(page.getByRole('button',{name:'Preview mapping',exact:true})).toBeEnabled();
});

test('FieldSense published mappings reopen as isolated repair drafts and retain scoped priority',async({page})=>{
  const fields=[{name:'message',method:'raw',source:'',type:'string',policy:'required',operations:[]}],parser={pattern_type:'mapping',pattern:'.*',service_scope:'custom',mapping_priority:2,mapping_plan:{version:1,fields},fields:[{name:'message',source:'mapping:message',type:'string',required:true}]},original={id:'published-native',revision:4,state:'published',payload:{name:'Published native',version:'1.0.0',parser,cases:[{raw:'hello',service:'custom',desired:{message:'hello'}}]}};let cloned=false,updated;
  const repair={id:'repair-native',revision:1,state:'draft',payload:{...original.payload,name:'Native repair',parser:{...parser,mapping_priority:3},repair_of:'published-native'}};await mockController(page,{managedParsers:[original]});
  await page.route('**/api/v1/assurance/parsers',route=>route.request().method()==='GET'?route.fulfill({contentType:'application/json',body:JSON.stringify({parsers:cloned?[original,repair]:[original]})}):route.fallback());
  await page.route('**/api/v1/assurance/parsers/published-native/clone-draft',async route=>{cloned=true;await route.fulfill({status:201,contentType:'application/json',body:JSON.stringify({id:repair.id,state:'draft',original_active_preserved:true})});});
  await page.route('**/api/v1/assurance/parsers/repair-native',async route=>{if(route.request().method()==='PUT'){updated=route.request().postDataJSON();return route.fulfill({contentType:'application/json',body:JSON.stringify({id:repair.id,state:'draft'})});}return route.fulfill({contentType:'application/json',body:JSON.stringify(repair)});});
  await login(page);await page.getByTestId('nav-parsers').click();await page.getByRole('button',{name:'Repair as new draft',exact:true}).click();await expect(page.locator('[data-qrw-key=name]')).toHaveValue('message');await expect(page.locator('#nfi-mapping-priority')).toHaveValue('3');await page.locator('[data-action=nfi-generate]').first().click();await page.getByRole('button',{name:'Validate and save review-gated draft',exact:true}).click();await expect.poll(()=>updated?.parser?.mapping_priority).toBe(3);expect(original.state).toBe('published');expect(original.payload.parser.mapping_priority).toBe(2);
});

test('Parser Catalog canonical regex repair opens editable Parser Creation JSON without changing production',async({page})=>{
  const parser={pattern_type:'regex',pattern:'^(?P<message>.*)$',fields:[{name:'message',source:'message',type:'string',required:true}]},original={id:'canonical-regex',revision:2,state:'published',payload:{name:'Canonical regex',version:'1.0.0',parser,cases:[{raw:'hello',service:'custom',desired:{message:'hello'}}]}},repair={id:'regex-repair',revision:1,state:'draft',payload:{...original.payload,name:'Regex repair'}};await mockController(page,{managedParsers:[original]});let cloned=false;
  await page.route('**/api/v1/assurance/parsers/canonical-regex/clone-draft',async route=>{cloned=true;await route.fulfill({status:201,contentType:'application/json',body:JSON.stringify({id:repair.id,state:'draft',original_active_preserved:true})});});
  await page.route('**/api/v1/assurance/parsers/regex-repair',route=>route.fulfill({contentType:'application/json',body:JSON.stringify(repair)}));
  await login(page);await page.getByTestId('nav-parsers').click();await page.getByRole('button',{name:'Repair as new draft',exact:true}).click();await expect(page.locator('#page-title')).toHaveText('Parser Creation Workspace');await expect(page.locator('#parser-workspace-form [name=edit_id]')).toHaveValue('regex-repair');expect(JSON.parse(await page.locator('#parser-workspace-form [name=parser]').inputValue()).pattern_type).toBe('regex');expect(cloned).toBe(true);expect(original.state).toBe('published');
});

test('AI manager tests the exact unsaved NVIDIA model with explicit SSE vision options and safe acknowledgements',async({page})=>{
 await mockController(page);let tested,saved;
 const base={id:'nvidia-first',name:'NVIDIA primary',provider:'nvidia_nim',model:'deepseek-ai/deepseek-v4.1-flash',endpoint:'https://integrate.api.nvidia.com/v1',configured:true,enabled:true,credential_source:'env_ref',credential_ref:'NVIDIA_API_KEY',priority:0,fallback_group:'review',timeout_seconds:15,max_response_bytes:1048576};
 await page.route('**/api/v1/admin/ai-provider',async route=>{if(route.request().method()==='PUT'){saved=route.request().postDataJSON();return route.fulfill({contentType:'application/json',body:JSON.stringify({provider:{...base,...saved}})});}return route.fulfill({contentType:'application/json',body:JSON.stringify({provider:base,profiles:[base],capabilities:{available:true,status:'configured_not_connection_verified'},templates:[{kind:'nvidia_nim',name:'NVIDIA NIM',base_url:base.endpoint,model_hint:'Exact NVIDIA model ID'}]})});});
 await page.route('**/api/v1/admin/ai-provider/test',async route=>{tested=route.request().postDataJSON();await route.fulfill({contentType:'application/json',body:JSON.stringify({connected:false,acknowledged:true,status:'acknowledged_incomplete',message:'Provider acknowledged the model; bounded output is incomplete.',error_code:'incomplete_output',tested_config:{configuration:'unsaved_draft',profile_id:tested.id,model:tested.model,endpoint_host:'integrate.api.nvidia.com',test_mode:'sse'}})});});
 await login(page);await page.getByTestId('nav-ai').click();const form=page.locator('#ai-provider-form');await page.getByRole('button',{name:'New model profile',exact:true}).click();await form.locator('[name=id]').fill('nvidia-vision');await form.locator('[name=name]').fill('Vision draft');await form.locator('[name=model]').fill('minimaxai/minimax-m3.1-flash');await form.locator('[name=endpoint]').fill(base.endpoint+'/chat/completions');await form.locator('[name=credential_ref]').fill('NVIDIA_API_KEY');await page.getByText('Explicit connection-test options',{exact:true}).click();await form.locator('[name=test_mode]').selectOption('stream');await form.locator('[name=test_image_url]').fill('https://assets.ngc.nvidia.com/products/example.jpg');await page.getByRole('button',{name:'Test connection',exact:true}).click();await expect(page.locator('#ai-provider-test-status')).toContainText('acknowledged_incomplete');await expect(page.locator('#ai-provider-test-status')).toContainText('unsaved_draft');await expect(form.locator('[name=model]')).toHaveValue('minimaxai/minimax-m3.1-flash');expect(tested).toMatchObject({id:'nvidia-vision',model:'minimaxai/minimax-m3.1-flash',credential_source:'env_ref',credential_ref:'NVIDIA_API_KEY',stream:true,test_max_tokens:64,image_url:'https://assets.ngc.nvidia.com/products/example.jpg'});expect(saved).toBeUndefined();expect(await page.evaluate(()=>JSON.stringify(localStorage))).not.toContain('NVIDIA_API_KEY');
 await form.getByRole('button',{name:'Save provider settings',exact:true}).click();await expect.poll(()=>saved?.id).toBe('nvidia-vision');expect(saved).toMatchObject({enabled:true,credential_source:'env_ref',credential_ref:'NVIDIA_API_KEY',fallback_group:'review'});expect(saved).not.toHaveProperty('api_key');expect(saved).not.toHaveProperty('image_url');expect(saved).not.toHaveProperty('stream');
});

test('FieldSense requests guidance before native preview and preserves the editable plan',async({page})=>{
  await mockController(page);
  const writes=[];
  page.on('request',request=>{if(request.method()==='POST')writes.push(new URL(request.url()).pathname);});
  await page.route('**/api/v1/ai/workspace',route=>{
    expect(route.request().postDataJSON()).toMatchObject({mode:'ask',workspace:'field-mapping',raw:'status=FAILED'});
    return route.fulfill({json:{answer:'Map the status key from the raw record.',review_only:true}});
  });
  await login(page);
  await page.getByTestId('nav-field-mapping').click();
  await page.locator('#nfi-raw').fill('status=FAILED');
  await page.getByRole('button',{name:'Add Custom Field',exact:true}).click();
  await page.locator('[data-qrw-key=name]').fill('my_status');
  await page.getByText('Optional configured AI help · review only',{exact:true}).click();
  await page.getByRole('button',{name:'Ask configured AI for review-only help',exact:true}).click();
  const dialog=page.getByRole('dialog');
  await dialog.getByRole('textbox',{name:'What do you need help with?'}).fill('How should I map status?');
  await dialog.getByRole('button',{name:'Get guidance'}).click();
  await expect(dialog.getByRole('status')).toHaveText('Map the status key from the raw record.');
  await dialog.getByRole('button',{name:'Close',exact:true}).click();
  await expect(page.locator('[data-qrw-key=name]')).toHaveValue('my_status');
  expect(writes).toContain('/api/v1/ai/workspace');
  expect(writes.some(path=>path.endsWith('/publish')||path.endsWith('/assurance/parsers')||path.endsWith('/rollouts/proposals'))).toBe(false);
});

test('FieldSense controller suggestions preserve manual mappings and expose optional read-only parser-pool evidence',async({page})=>{
 const mock=await mockController(page),raw='mailnode smtp 007';let request;
 const structural=[{name:'service',label:'Suggested service',method:'position',source:'2',type:'string',policy:'required',operations:[]}],runtime={available:true,builtin_profiles:[{id:'builtin/smtp',eligible:true}],managed_profiles:[{id:'published/custom-smtp',state:'published',eligible:true}],matches:[{parser_id:'published/custom-smtp',fields:{observed_host:'mailnode'}}],attempts:[{parser_id:'builtin/smtp',matched:false},{parser_id:'published/custom-smtp',matched:true}]},observed=[{name:'my_mail_service',method:'byte',source:'1',source_end:'8',type:'string',policy:'required'},{name:'observed_host',method:'byte',source:'1',source_end:'8',type:'string',policy:'required'}];
 await page.route('**/api/v1/field-mapping/suggest',async route=>{request=route.request().postDataJSON();await route.fulfill({contentType:'application/json',body:JSON.stringify({sample_count:1,fields:structural,runtime,runtime_field_suggestions:observed})});});
 await login(page);await page.getByTestId('nav-field-mapping').click();await page.locator('#nfi-raw').fill(raw);await page.getByText('Import or edit a mapping plan',{exact:true}).click();
 const manual={name:'my_mail_service',label:'Operator service label',method:'position',source:'2',type:'string',policy:'required',operations:[{op:'uppercase'},{op:'prefix',value:'svc:'}]};await page.locator('#nfi-plan-import').fill(JSON.stringify({version:1,service_scope:'smtp',fields:[manual]}));await page.getByRole('button',{name:'Import mapping',exact:true}).click();await page.getByRole('button',{name:'Suggest from raw log',exact:true}).click();
 await expect.poll(()=>request?.service).toBe('smtp');expect(request.samples).toEqual([raw]);await expect(page.locator('[data-qrw-row]')).toHaveCount(1);await expect(page.locator('[data-qrw-key=name]')).toHaveValue('my_mail_service');await expect(page.locator('[data-qrw-key=label]')).toHaveValue('Operator service label');await expect(page.locator('#nfi-runtime-evidence')).toBeVisible();await page.locator('#nfi-runtime-evidence summary').click();await expect(page.locator('#nfi-runtime-evidence-json')).toContainText('builtin/smtp');await expect(page.locator('#nfi-runtime-evidence-json')).toContainText('published/custom-smtp');
 await page.getByRole('button',{name:'Add observed fields from the existing parser',exact:true}).click();await expect(page.locator('[data-qrw-row]')).toHaveCount(2);await page.getByRole('button',{name:'Test all fields',exact:true}).click();await page.locator('[data-action=nfi-generate]').first().click();const parser=JSON.parse(await page.locator('#nfi-parser-json').inputValue());expect(parser.mapping_plan.fields.find(field=>field.name==='my_mail_service')).toMatchObject({label:'Operator service label',method:'position',source:'2',operations:manual.operations});expect(parser.mapping_plan.fields.find(field=>field.name==='observed_host')).toMatchObject({method:'byte',source:'1',source_end:'8'});await expect(page.locator('#nfi-runtime-evidence-json')).toContainText('"matched": true');expect(mock.observed.some(item=>item.endsWith('/assurance/parsers')&&item.startsWith('POST'))).toBe(false);expect(mock.observed.some(item=>item.endsWith('/publish'))).toBe(false);
 await page.locator('#nfi-raw').fill('different smtp 009');await expect(page.locator('#nfi-runtime-evidence')).toBeHidden();
});

test('FieldSense ignores stale and cancelled controller suggestions without overwriting the working plan',async({page})=>{
 await mockController(page);const pending=[];await page.route('**/api/v1/field-mapping/suggest',route=>{pending.push(route);});await login(page);await page.getByTestId('nav-field-mapping').click();await page.locator('#nfi-raw').fill('first smtp');await page.getByRole('button',{name:'Add Custom Field',exact:true}).click();await page.locator('[data-qrw-key=name]').fill('keep_manual');await page.locator('[data-qrw-key=policy]').selectOption('required');
 await page.getByRole('button',{name:'Suggest from raw log',exact:true}).click();await expect.poll(()=>pending.length).toBe(1);await page.locator('#nfi-raw').fill('second smtp');await pending[0].fulfill({contentType:'application/json',body:JSON.stringify({fields:[{name:'stale_forbidden',method:'position',source:'1',type:'string',policy:'required'}],runtime:{matches:[{parser_id:'stale/parser'}]}})});await expect(page.getByRole('button',{name:'Suggest from raw log',exact:true})).toBeEnabled();await expect(page.locator('[data-qrw-key=name]')).toHaveValue('keep_manual');await expect(page.locator('#nfi-runtime-evidence')).toBeHidden();
 await page.getByRole('button',{name:'Suggest from raw log',exact:true}).click();await expect.poll(()=>pending.length).toBe(2);await page.getByRole('button',{name:'Cancel analysis',exact:true}).click();await expect(page.getByRole('button',{name:'Suggest from raw log',exact:true})).toBeEnabled();try{await pending[1].fulfill({contentType:'application/json',body:JSON.stringify({fields:[{name:'cancelled_forbidden',method:'position',source:'1',type:'string',policy:'required'}],runtime:{matches:[{parser_id:'cancelled/parser'}]}})});}catch{}await expect(page.locator('[data-qrw-row]')).toHaveCount(1);await expect(page.locator('[data-qrw-key=name]')).toHaveValue('keep_manual');await expect(page.locator('#nfi-runtime-evidence')).toBeHidden();
});

test('NFI maps compound tokens and colon key/value fields into a transform-capable native candidate', async ({page})=>{
  const raw='20261201 102030123+0530 mailnode smtp 42 0 140026854221568 Note;DeliveryEvent status=failed:reason=notneeded:server=cloud';
  const mock=await mockController(page,{quarantineRaw:raw});await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await page.getByRole('button',{name:/Review desired schema/}).click();await page.getByRole('button',{name:'Suggest from raw log',exact:true}).click();
  await expect(page.locator('[data-qrw-row]')).toHaveCount(12);
  const confirmations=page.getByLabel('Confirm delimiter interpretation');await confirmations.nth(0).check();await confirmations.nth(1).check();
  const event=page.locator('[data-qrw-row]').nth(8);await event.locator('.nfi-operations summary').click();await event.getByRole('button',{name:'Add operation',exact:true}).click();
  await event.locator('[data-qrw-op-key=op]').selectOption('uppercase');await page.getByRole('button',{name:'Test all fields',exact:true}).click();
  const desired=JSON.parse(await page.locator('[name=gprs_desired]').inputValue());expect(desired).toMatchObject({loglevel:'Note',event:'DELIVERYEVENT',status:'failed',reason:'notneeded',server:'cloud',thread_id:'140026854221568'});
  await page.getByRole('button',{name:/5 · Auto-populate complete parser/}).click();
  const form=page.locator('#gprs-parser-helper-form'),parser=JSON.parse(await form.locator('[name=parser]').inputValue());expect(parser.pattern_type).toBe('mapping');expect(parser.mapping_plan.fields.find(field=>field.name==='event').operations[0].op).toBe('uppercase');
  await form.getByRole('button',{name:'Save review-gated draft',exact:true}).click();await expect.poll(()=>mock.observed.includes('POST /api/v1/assurance/parsers')).toBe(true);expect(mock.observed.some(request=>request.endsWith('/publish'))).toBe(false);
});

test('NFI standalone workbench verifies ordered splits, Unicode selections and merged names and saves one governed plan',async({page})=>{
  const mock=await mockController(page);let verifiedPlan;
  await page.route('**/api/v1/assurance/mapping-preview',async route=>{verifiedPlan=route.request().postDataJSON().plan;await route.fulfill({contentType:'application/json',body:JSON.stringify({passed:true,actual:{kind:'event:x',symbol:'α',combined:'event:x/α'},fields:[],errors:[]})});});
  await login(page);await page.getByTestId('nav-field-mapping').click();await expect(page.locator('#page h1')).toHaveText('NLI FieldSense Studio');
  await page.locator('#nfi-raw').fill('α beta;X|Y');await expect(page.locator('.nfi-token').first()).toContainText('chars 1–1');
  await page.getByText('Import or edit a mapping plan',{exact:true}).click();
  const fields=[{name:'kind',type:'string',policy:'required',method:'position',source:'2',operations:[{op:'split',delimiter:';',part:2},{op:'split',delimiter:'|',part:1},{op:'lowercase'},{op:'prefix',value:'event:'}]},{name:'symbol',type:'string',policy:'required',method:'character',source:'1',source_end:'1'},{name:'combined',type:'string',policy:'derived',method:'derived',source:'kind,symbol',join_separator:'/'}];
  await page.locator('#nfi-plan-import').fill(JSON.stringify({version:1,fields,service_scope:'mapping-test',scope_pattern:'^α',reviewed_global_scope:false}));await page.getByRole('button',{name:'Import mapping',exact:true}).click();await page.getByRole('button',{name:'Test all fields',exact:true}).click();
  await expect(page.locator('[data-qrw-row]').nth(2).locator('[data-qrw-preview]')).toHaveText('event:x/α');
  await page.getByRole('button',{name:'Preview mapping',exact:true}).click();await expect(page.locator('#qrw-field-status')).toContainText('Verified by controller');expect(verifiedPlan.fields[0].operations).toHaveLength(4);
  await page.locator('[data-action=nfi-generate]').first().click();const parser=JSON.parse(await page.locator('#nfi-parser-json').inputValue());expect(parser.mapping_plan.fields[2].method).toBe('derived');expect(parser.service_scope).toBe('mapping-test');expect(parser.scope_pattern).toBe('^α');
  await page.getByRole('button',{name:'Validate and save review-gated draft',exact:true}).click();await expect.poll(()=>mock.observed.includes('POST /api/v1/assurance/parsers')).toBe(true);await expect(page.locator('#page-title')).toHaveText('NLI FieldSense Studio');await expect(page.locator('#page tr').filter({hasText:'native-field-mapping'})).toContainText('draft');
  await page.getByRole('button',{name:'Use candidate in Parser Creation Workspace',exact:true}).click();expect(JSON.parse(await page.locator('#parser-workspace-form [name=parser]').inputValue()).mapping_plan.fields).toHaveLength(3);
  await page.getByTestId('nav-field-mapping').click();await page.getByRole('button',{name:'Use mapped names in output delivery',exact:true}).click();expect(JSON.parse(await page.locator('#output-profile-form [name=field_columns]').inputValue())).toEqual({kind:'kind',symbol:'symbol',combined:'combined'});
});

test('NFI corpus profiles report optional evidence and stay explicit review suggestions',async({page})=>{
  await mockController(page);let request;
  const fields=[{name:'level',label:'Level',type:'string',policy:'required',method:'key_value',source:'level',confidence:'observed',evidence:'2/2 samples'},{name:'detail',label:'Detail',type:'string',policy:'optional',method:'key_value',source:'detail',confidence:'observed',evidence:'1/2 samples'}];
  await page.route('**/api/v1/field-mapping/suggest',async route=>{request=route.request().postDataJSON();await route.fulfill({contentType:'application/json',body:JSON.stringify({sample_count:2,fields,profiles:[{name:'key/value structure',sample_count:2,sample:'level=INFO detail=ready',fields}]})});});
  await login(page);await page.getByTestId('nav-field-mapping').click();await page.getByText('Learn structural suggestions from multiple samples',{exact:true}).click();await page.locator('#nfi-corpus').fill('level=INFO detail=ready\nlevel=WARN');await page.getByRole('button',{name:'Learn sample structures',exact:true}).click();
  await expect(page.locator('#nfi-learning-status')).toContainText('2 samples inspected');expect(request.samples).toEqual(['level=INFO detail=ready','level=WARN']);await page.getByRole('button',{name:'Review this profile',exact:true}).click();await expect(page.locator('[data-qrw-key=policy]').nth(1)).toHaveValue('optional');await expect(page.locator('#nfi-profile-list')).toContainText('1/2 samples');
  await page.getByRole('button',{name:'Test all fields',exact:true}).click();await expect(page.locator('#qrw-field-status')).toContainText('2 fields mapped');
});

test('QRW preserves the supplied SMTP sample structure, multiword key-value reason, and complete-parser handoff', async ({ page }) => {
  const raw = '20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed;reason=not needed';
  await mockController(page, { quarantineRaw: raw });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button', { name: 'Inspect', exact: true }).click();
  await page.getByRole('button', { name: /Review desired schema/ }).click();
  await page.getByRole('button', { name: 'Suggest from raw log' }).click();

  const rows = page.locator('[data-qrw-row]');
  await expect(rows).toHaveCount(11);
  const statusRow = rows.nth(9);
  const reasonRow = rows.nth(10);
  await expect(statusRow.locator('[data-qrw-key=name]')).toHaveValue('status');
  await expect(statusRow.locator('[data-qrw-key=method]')).toHaveValue('key_value');
  await expect(statusRow.locator('[data-qrw-key=source]')).toHaveValue('status');
  await expect(statusRow.locator('[data-qrw-preview]')).toContainText('failed');
  await expect(reasonRow.locator('[data-qrw-key=name]')).toHaveValue('reason');
  await expect(reasonRow.locator('[data-qrw-key=method]')).toHaveValue('key_value');
  await expect(reasonRow.locator('[data-qrw-key=source]')).toHaveValue('reason');
  await expect(reasonRow.locator('[data-qrw-preview]')).toContainText('not needed');

  const confirmations = page.locator('[data-qrw-key=confirmed]');
  await expect(confirmations).toHaveCount(2);
  await confirmations.nth(0).check();
  await confirmations.nth(1).check();
  await page.getByRole('button', { name: 'Test all fields' }).click();
  await expect(page.getByText('11 fields mapped from actual evidence. Review the generated JSON before testing.')).toBeVisible();

  const desired = JSON.parse(await page.locator('#gprs-preview-form [name=gprs_desired]').inputValue());
  expect(desired).toMatchObject({ status: 'failed', reason: 'not needed', event_type: 'SieveRuleMsg' });

  const autoPopulate = page.getByRole('button', { name: /5 · Auto-populate complete parser/ });
  await expect(autoPopulate).toBeVisible();
  await expect(autoPopulate).toBeEnabled();
  await autoPopulate.click();
  await expect(page.locator('#gprs-parser-helper-form')).toBeVisible();
  const parser = await page.locator('#gprs-parser-helper-form [name=parser]').inputValue();
  expect(parser).toContain('"status"');
  expect(parser).toContain('"reason"');
});

test('QRW accepts operator field names in Inspect and updates live completion state', async ({ page }) => {
  await mockController(page,{quarantineRaw:'2026-10-30 mail01 ERROR delivery failed'});
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await expect(page.locator('[data-wizard-step="1"] .qrw-step-state')).toHaveText('Complete');
  await expect(page.locator('[data-wizard-step="2"] .qrw-step-state')).toHaveText('Complete');
  await page.locator('[name=manual_field_names]').fill('event_date, host, severity, message');
  await page.getByRole('button',{name:/Create mappings from my field names/}).click();
  await expect(page.locator('[data-qrw-key=name]')).toHaveCount(4);
  await expect.poll(async () => page.locator('[data-qrw-key=name]').evaluateAll((els) => els.map((el) => el.value))).toEqual(['event_date', 'host', 'severity', 'message']);
  await expect.poll(async () => page.locator('[data-qrw-key=source]').evaluateAll((els) => els.map((el) => el.value))).toEqual(['1', '2', '3', '4']);
  await expect(page.locator('[data-qrw-key=source_end]')).toHaveCount(4);
  await expect(page.getByText('Positions are 1-based; end token is inclusive.').first()).toBeVisible();
  const messageRow = page.locator('[data-qrw-row]').nth(3);
  await messageRow.locator('[data-qrw-key=source_end]').fill('5');
  await page.getByRole('button', { name: 'Test all fields' }).click();
  await expect(messageRow.locator('[data-qrw-key=name]')).toHaveValue('message');
  await expect(messageRow.locator('[data-qrw-key=source_end]')).toHaveValue('5');
  await expect(page.locator('[data-wizard-step="3"] .qrw-step-state')).toHaveText('Complete');
  await expect(page.locator('[data-wizard-step="4"] .qrw-step-state')).toHaveText('Complete');
  const desired=JSON.parse(await page.locator('#gprs-preview-form [name=gprs_desired]').inputValue());
  expect(desired).toEqual({event_date:'2026-10-30',host:'mail01',severity:'ERROR',message:'delivery failed'});
});

test('metadata-only quarantine records expose evidence recovery without actionable badges', async ({ page }) => {
  await mockController(page, { qrwRecords: [{
    id: 'q-meta',
    node_id: 'node-meta',
    service: 'custom',
    first_seen: '2026-11-03T00:00:00Z',
    attempt_count: 1,
    status: 'waiting',
    error_codes: ['no_match'],
    raw_available: false,
  }] });
  await login(page);

  await expect(page.locator('[data-nav-count="quarantine-wizard"]')).toBeHidden();
  await expect(page.locator('[data-nav-count="unknowns"]')).toBeHidden();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.getByRole('button', { name: 'Acquire evidence' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Inspect', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: 'Acquire evidence' }).click();
  await expect(page.locator('.toast').last()).toContainText('Enable bounded quarantine evidence retention');
});

test('terminal quarantine actions clear actionable navigation badges while retaining records', async ({ page }) => {
  await mockController(page,{qrwRecords:[{id:'q-1',node_id:'node-1',service:'smtp',first_seen:'2026-10-30T00:00:00Z',attempt_count:1,status:'waiting',error_codes:['no_match'],raw_available:true}]});
  await login(page);
  // Historical KPIs are retained telemetry, not a projection of the current
  // quarantine queue. A terminal queue action must never rewrite them.
  await expect(page.locator('[data-chart-category="Parsed"]').first()).toHaveAttribute('data-chart-value','9007199254740990');
  await expect(page.locator('[data-nav-count="quarantine-wizard"]')).toHaveText('1');
  await page.getByTestId('nav-quarantine-wizard').click();
  page.once('dialog',dialog=>dialog.accept('not applicable to this source'));
  await page.getByRole('button',{name:'Ignore / exclude',exact:true}).click();
  await expect(page.locator('[data-nav-count="quarantine-wizard"]')).toBeHidden();
  await expect(page.locator('[data-nav-count="upes"]')).toBeHidden();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('#page')).toContainText(/excluded/i);
  await page.getByTestId('nav-operations').click();
  await expect(page.locator('[data-chart-category="Parsed"]').first()).toHaveAttribute('data-chart-value','9007199254740990');
});

test('terminal quarantine metadata stays out of manual attention and repair actions after refresh', async ({ page }) => {
  await mockController(page,{qrwRecords:['excluded','evidence_deleted','resolved'].map(status=>({id:'q-'+status,node_id:'node-1',service:'smtp',first_seen:'2026-10-30T00:00:00Z',attempt_count:7,status,error_codes:['no_match'],raw_available:status!=='evidence_deleted'}))});
  await login(page);
  await page.getByTestId('nav-unknowns').click();
  await expect(page.locator('.card.metric').filter({hasText:'Manual attention'}).locator('.metric-value')).toHaveText('0');
  for(const status of ['excluded','evidence_deleted','resolved']){
    const row=page.locator('#page tr').filter({hasText:'q-'+status});
    await expect(row.locator('.status')).toHaveText(status);
    await expect(row.locator('[data-action="quarantine-diagnose"]')).toHaveCount(0);
  }
  await page.evaluate(()=>sessionStorage.setItem('nli.qrw.selected-record','q-excluded'));
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.getByRole('button',{name:'Inspect',exact:true})).toHaveCount(0);
  await expect(page.locator('[data-wizard-step="1"] .qrw-step-state')).toHaveText('Current · do this now');
  await page.reload();
  await page.getByTestId('nav-unknowns').click();
  await expect(page.locator('.card.metric').filter({hasText:'Manual attention'}).locator('.metric-value')).toHaveText('0');
  for(const name of ['quarantine-wizard','upes','unknowns'])await expect(page.locator(`[data-nav-count="${name}"]`)).toBeHidden();
});

test('QRW stages 6–9 give the next action and gate replay on the exact origin bundle before verify and rollback', async ({ page }) => {
  const checksum='d'.repeat(64),wrong='e'.repeat(64);
  const mock=await mockController(page,{quarantineRaw:'alpha beta',replayPending:true,
    qrwParsers:[{id:'qr-candidate',revision:1,state:'in_review',payload:{name:'reviewed-smtp',source_quarantine_record_id:'q-1',artifact_checksum:checksum,validation:{score:100}}},{id:'qr-offboard',revision:2,state:'published',payload:{name:'retired-smtp',source_quarantine_record_id:'q-2',artifact_checksum:checksum,activated:true,validation:{score:100}}}],
    qrwRecords:[{id:'q-1',node_id:'node-1',service:'smtp',first_seen:'2026-10-10T00:00:00Z',attempt_count:2,status:'waiting',error_codes:['no_match'],raw_available:true}],
    qrwSyncAssets:[{asset_class:'parser',asset_id:'qr-candidate',name:'reviewed-smtp',desired:{revision:1,sha256:checksum,state:'published'},rollout:{id:'global-qr-rollout',status:'complete',target_mode:'all_eligible',total_nodes:2,done_nodes:2,failed_nodes:0},clients:[{node_id:'node-1',desired_revision:1,desired_hash:checksum,applied_revision:0,applied_hash:wrong,status:'online',drifted:true},{node_id:'node-2',desired_revision:1,desired_hash:checksum,applied_revision:1,applied_hash:checksum,status:'online',drifted:false}]}]
  });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  const lifecycle=page.locator('#page tr').filter({hasText:'reviewed-smtp'});
  await expect(page.locator('.qrw-next-action')).toContainText('Choose a retained failure or paste a new log');
  await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await page.getByRole('button',{name:/Review desired schema/}).click();
  await page.getByRole('button',{name:'Suggest from raw log'}).click();
  await page.getByRole('button',{name:'Test all fields'}).click();
  await page.locator('[data-wizard-step="5"] button').click();
  await page.getByRole('button',{name:/Prepare candidate from evidence/}).click();
  await page.locator('[data-wizard-step="6"] button').click();
  await page.getByRole('button',{name:'Run deterministic preview',exact:true}).click();
  await expect(page.locator('[data-wizard-step="7"] button')).toBeEnabled();
  await page.locator('[data-wizard-step="7"] button').click();
  await expect(page.locator('[data-wizard-step="7"]')).toHaveClass(/active/);
  await expect(page.locator('[data-wizard-step="8"] button')).toBeDisabled();
  await expect(page.locator('[data-wizard-step="9"] button')).toBeDisabled();
  await page.getByTestId('nav-operations').click();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('.qrw-next-action')).toContainText('Review the candidate independently');
  await page.locator('[data-wizard-step="7"] button').click();
  await expect(lifecycle).toContainText('1/2 clients acknowledged');
  await lifecycle.getByRole('button',{name:'Approve',exact:true}).click();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('.qrw-next-action')).toContainText('Publish the approved candidate');
  await lifecycle.getByRole('button',{name:'Publish',exact:true}).click();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('.qrw-next-action')).toContainText('Create the origin-client rollout proposal');
  await expect(page.getByRole('button',{name:/Request targeted replay/})).toBeDisabled();
  await expect(page.locator('#gprs-replay-form [name=record_ids] option[value="q-1"]')).toHaveAttribute('disabled','');
  await expect(page.getByTestId('qrw-deployment-status')).toContainText('Step 8 · Deploy and replay');
  await page.getByText('Deployment details and fleet controls',{exact:true}).click();
  await expect(page.getByRole('list',{name:'Deployment and replay checklist'})).toContainText('Published bundle available');
  await page.getByRole('button',{name:'Propose to all eligible clients'}).click();
  await expect.poll(()=>mock.qrwRolloutRequests).toHaveLength(1);
  expect(mock.qrwRolloutRequests[0]).toEqual({target_mode:'all_eligible',origin_node_id:'node-1',node_ids:[],bundle_hash:checksum});
  await page.getByText('Deployment details and fleet controls',{exact:true}).click();
  const rolloutRow=page.locator('#page tr').filter({hasText:'qrw-rollout-1'});
  await expect(rolloutRow).toContainText('all_eligible · 2 client(s)');
  await expect(rolloutRow.getByRole('button',{name:'Administrator approve exact bundle'})).toBeEnabled();
  await expect(page.getByRole('button',{name:'Propose to origin client'})).toBeDisabled();
  await expect(page.getByRole('button',{name:'Propose to all eligible clients'})).toBeDisabled();
  await expect(page.locator('.qrw-next-action')).toContainText('Complete the audited rollout approval');
  await rolloutRow.locator('[data-rollout-reason]').fill('Restore this exact reviewed bundle to the origin client');
  await rolloutRow.getByRole('button',{name:'Administrator approve exact bundle'}).click();
  await expect.poll(()=>mock.rolloutApprovalRequests).toHaveLength(1);
  expect(mock.rolloutApprovalRequests[0]).toEqual({id:'qrw-rollout-1',body:{reason:'Restore this exact reviewed bundle to the origin client',emergency_self_approval:true,expected_revision:1,expected_bundle_hash:checksum}});
  await expect(page.locator('.qrw-next-action')).toContainText('Verify the origin client acknowledgement');
  await page.locator('.qrw-next-action').getByRole('button',{name:/Verify origin acknowledgement/}).click();
  await expect.poll(()=>mock.qrwSyncVerifications).toHaveLength(1);
  expect(mock.qrwSyncVerifications[0]).toEqual({bundle_hash:checksum,node_ids:['node-1']});
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(lifecycle).toContainText('2/2 clients acknowledged');
  await expect(page.locator('[data-gprs-record="q-1"]').locator('xpath=ancestor::tr/following-sibling::tr[1]')).toContainText('Exact published bundle acknowledged by this client');
  await expect(page.locator('#gprs-replay-form [name=record_ids] option[value="q-1"]')).not.toHaveAttribute('disabled');
  const replay=page.locator('#gprs-replay-form');
  await replay.locator('[name=record_ids]').selectOption('q-1');
  await expect(replay.locator('[name=node_id]')).toHaveValue('node-1');
  await replay.getByRole('button',{name:/Request targeted replay/}).click();
  await expect.poll(()=>mock.qrwReplays).toHaveLength(1);
  expect(mock.qrwReplays[0]).toEqual({record_ids:['q-1'],node_id:'node-1',expected_bundle_hash:checksum});
  await expect(page.locator('[data-wizard-step="9"]')).toHaveClass(/active/);
  await expect(page.locator('[data-qrw-panel="9"]')).toBeVisible();
  await expect(page.locator('[data-qrw-panel="9"]')).toContainText('Replay request accepted; the final controller outcome is still pending.');
  mock.qrwRecords[0].status='resolved';
  mock.qrwHistory.push({record_id:'q-1',status:'resolved',outcome:'success'});
  await page.getByRole('button',{name:/Refresh controller result/}).click();
  await expect(page.locator('[data-qrw-panel="9"] .qrw-runbook')).toContainText('Origin record resolved');
  await expect(page.locator('[data-qrw-panel="9"]')).not.toContainText('the final controller outcome is still pending');
  await expect(page.locator('[data-qrw-panel="8"] .qrw-runbook .qrw-runbook-state').filter({hasText:'Do this now'})).toHaveCount(0);
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('.qrw-next-action')).toContainText('Recovery complete');
  await page.locator('[data-wizard-step="7"] button').click();
  await lifecycle.getByRole('button',{name:'Rollback',exact:true}).click();
  await expect.poll(()=>mock.qrwParserActions.some(item=>item.id==='qr-candidate'&&item.action==='rollback')).toBe(true);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.locator('[data-wizard-step="7"] button').click();
  const offboardRow=page.locator('#page tr').filter({hasText:'retired-smtp'});
  await offboardRow.getByRole('button',{name:'Offboard',exact:true}).click();
  await expect.poll(()=>mock.qrwParserActions.some(item=>item.id==='qr-offboard'&&item.action==='offboard')).toBe(true);
  expect(mock.qrwParserActions.map(item=>item.action)).toEqual(['approve','publish','rollback','offboard']);
});

test('QRW restores saved candidate validation before rendering review after a browser reload', async ({ page }) => {
  await mockController(page,{
    qrwParsers:[{id:'saved-candidate',revision:4,state:'in_review',created_by:'independent-author',payload:{name:'Saved parser',source_quarantine_record_id:'q-1',artifact_checksum:'a'.repeat(64),validation_passed:true,validation:{total:1,failed:0}}}],
    qrwRecords:[{id:'q-1',node_id:'node-1',status:'waiting',raw_available:true,attempt_count:0}]
  });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await expect(page.locator('[data-wizard-step="2"]')).toHaveClass(/active/);
  await expect.poll(()=>page.evaluate(()=>sessionStorage.getItem('nli.qrw.selected-record'))).toBe('q-1');
  await page.reload();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('[data-wizard-step="6"]')).toHaveClass(/done/);
  await expect(page.locator('[data-wizard-step="7"] button')).toBeEnabled();
  await expect(page.locator('[data-qrw-panel="7"]')).toBeVisible();
  const candidate=page.locator('[data-qrw-panel="7"] tr').filter({hasText:'saved-candidate'});
  await candidate.getByRole('button',{name:'Approve',exact:true}).click();
  await expect(candidate.getByRole('button',{name:'Publish',exact:true})).toBeVisible();
});

test('QRW retries the drifted origin after a completed rollout and blocks replay without the current bundle hash', async ({ page }) => {
  const hash='a'.repeat(64);
  const mock=await mockController(page,{
    qrwParsers:[{id:'origin-parser',revision:2,state:'published',payload:{source_quarantine_record_id:'q-1',artifact_checksum:hash,validation_passed:true,activated:true}}],
    qrwRecords:[{id:'q-1',node_id:'node-1',status:'waiting',raw_available:true,attempt_count:0}],
    qrwSyncAssets:[{asset_class:'parser',asset_id:'origin-parser',desired:{sha256:hash},clients:[{node_id:'node-1',desired_hash:hash,applied_hash:hash,drifted:true,status:'online'}]}],
    rolloutSeed:[{id:'other-client',state:'approved',payload:{type:'parser_sync',bundle_hash:hash,target_mode:'all_eligible',target_nodes:['node-2']}},{id:'completed-origin',state:'active',payload:{type:'parser_sync',bundle_hash:hash,target_mode:'origin',target_nodes:['node-1'],live_rollout_id:'done-job'}}]
  });
  await page.route('**/api/v1/rollouts',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({proposals:mock.rolloutProposals,live:[{id:'done-job',type:'parser_sync',status:'completed'}]})}));
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button',{name:'Inspect',exact:true}).click();
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.locator('.qrw-next-action')).toContainText('Create the origin-client rollout proposal');
  await page.locator('[data-wizard-step="8"] button').click();
  await expect(page.getByRole('button',{name:/Deploy to origin client/})).toBeEnabled();
  await page.route('**/api/v1/parser-sync',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({assets:[{asset_class:'parser',asset_id:'origin-parser'}],clients:[{node_id:'node-1',desired_hash:hash,applied_hash:hash,drifted:false}]})}));
  await page.getByTestId('nav-quarantine-wizard').click();
  await expect(page.getByRole('button',{name:/Request targeted replay/})).toBeDisabled();
  await expect(page.locator('#gprs-replay-form option[value="q-1"]')).toHaveAttribute('disabled','');
});

test('Assurance repair keeps deployment, approval, retry, replay and refresh inside the studio', async ({ page }) => {
  const hash='f'.repeat(64);
  const mock=await mockController(page,{
    qrwParsers:[{id:'studio-parser',revision:3,state:'published',payload:{name:'Studio parser',source_quarantine_record_id:'q-1',artifact_checksum:hash,validation_passed:true,activated:true}}],
    qrwRecords:[{id:'q-1',node_id:'node-1',service:'smtp',status:'waiting',attempt_count:1,raw_available:true}],
    qrwSyncAssets:[{asset_class:'parser',asset_id:'studio-parser',desired:{sha256:hash,state:'published'},clients:[{node_id:'node-1',desired_hash:hash,applied_hash:'',drifted:true,status:'online'}]}]
  });
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab',{name:'Guided Parser Repair Studio',exact:true}).click();
  const studio=page.locator('#assurance-panel-gprs');
  const expectStudio=async()=>{
    await expect(page.locator('#page-title')).toHaveText('Assurance Lab');
    await expect(page.getByRole('tab',{name:'Guided Parser Repair Studio',exact:true})).toHaveAttribute('aria-selected','true');
    await expect(studio).toBeVisible();
  };
  await studio.getByRole('button',{name:'Inspect',exact:true}).click();
  await studio.locator('[data-wizard-step="8"] button').click();
  await studio.getByRole('button',{name:/Deploy to origin client/}).click();
  await expect.poll(()=>mock.qrwRolloutRequests).toHaveLength(1);
  await expectStudio();
  await studio.getByText('Deployment details and fleet controls',{exact:true}).click();
  await studio.locator('tr').filter({hasText:'qrw-rollout-1'}).locator('[data-action="rollout-cancel"]').click();
  await expectStudio();
  await expect.poll(()=>mock.rolloutProposals[0].state).toBe('cancelled');
  await studio.getByRole('button',{name:/Deploy to origin client/}).click();
  await expect.poll(()=>mock.qrwRolloutRequests).toHaveLength(2);
  await expectStudio();
  const rolloutRow=studio.locator('.qrw-next-action');
  await rolloutRow.locator('[data-rollout-reason]').fill('Administrator reviewed this exact studio deployment');
  await rolloutRow.locator('[data-action="rollout-approve"]').click();
  await expect.poll(()=>mock.rolloutApprovalRequests).toHaveLength(1);
  await expectStudio();
  await studio.locator('.qrw-next-action').getByRole('button',{name:/Verify origin acknowledgement/}).click();
  await expect.poll(()=>mock.qrwSyncVerifications).toHaveLength(1);
  await expectStudio();
  const replay=studio.locator('#gprs-replay-form');
  await replay.locator('[name=record_ids]').selectOption('q-1');
  await replay.getByRole('button',{name:/Request targeted replay/}).click();
  await expect.poll(()=>mock.qrwReplays).toHaveLength(1);
  await expectStudio();
  await studio.getByRole('button',{name:/Refresh controller result/}).click();
  await expectStudio();
  await expect(studio.locator('[data-qrw-panel="9"]')).toBeVisible();
});

test('Approvals workspace sends reason-bearing independent and exact emergency approval contracts', async ({ page }) => {
  const hash='a'.repeat(64),sync=[{asset_class:'parser',asset_id:'reviewed-parser',desired:{revision:4,sha256:hash,state:'published'},clients:[{node_id:'node-1',desired_hash:hash,applied_hash:'',status:'online',drifted:true}]}];
  const mock=await mockController(page,{qrwSyncAssets:sync,rolloutSeed:[
    {id:'independent-proposal',revision:7,created_by:'other-reviewer',state:'pending_approval',payload:{type:'parser_sync',name:'Independent parser rollout',bundle_hash:hash,target_mode:'selected',target_nodes:['node-1']}},
    {id:'emergency-proposal',revision:8,created_by:'test-operator',state:'pending_approval',payload:{type:'parser_sync',name:'Emergency parser rollout',bundle_hash:hash,target_mode:'selected',target_nodes:['node-1']}},
  ]});
  await login(page);
  await page.getByTestId('nav-rollouts').click();
  const independent=page.locator('tr').filter({hasText:'Independent parser rollout'});
  await independent.locator('[data-rollout-reason]').fill('Validated against the approved parser bundle');
  await independent.getByRole('button',{name:'Approve reviewed proposal'}).click();
  await expect.poll(()=>mock.rolloutApprovalRequests).toHaveLength(1);
  expect(mock.rolloutApprovalRequests[0]).toEqual({id:'independent-proposal',body:{reason:'Validated against the approved parser bundle'}});
  // The approval handler refreshes the workspace asynchronously. Wait for that
  // refresh to commit before operating on the second proposal so the following
  // click can never race the outgoing DOM on slower CI runners.
  await expect(independent.getByRole('button',{name:'Approve reviewed proposal'})).toHaveCount(0);
  const emergency=page.locator('tr').filter({hasText:'Emergency parser rollout'});
  await emergency.locator('[data-rollout-reason]').fill('Production recovery approved by administrator');
  await emergency.getByRole('button',{name:'Administrator approve exact bundle'}).click();
  await expect.poll(()=>mock.rolloutApprovalRequests).toHaveLength(2);
  expect(mock.rolloutApprovalRequests[1]).toEqual({id:'emergency-proposal',body:{reason:'Production recovery approved by administrator',emergency_self_approval:true,expected_revision:8,expected_bundle_hash:hash}});
});

test('legacy parser proposals reach controller verification and a hash rejection becomes one actionable inline state', async ({ page }) => {
  const hash='b'.repeat(64),legacyHash=hash.slice(0,20),sync=[{asset_class:'parser',asset_id:'legacy-parser',desired:{revision:2,sha256:hash,state:'published'},clients:[{node_id:'node-1',desired_hash:hash,applied_hash:'',status:'online',drifted:true}]}];
  const mock=await mockController(page,{qrwSyncAssets:sync,rolloutApprovalFailure:'parser sync proposal bundle hash is invalid',rolloutSeed:[{id:'legacy-proposal',revision:2,created_by:'other-reviewer',state:'pending_approval',payload:{type:'parser_sync',name:'Legacy parser rollout',bundle_hash:legacyHash,payload:{asset_bundle:{schema_version:1}},target_mode:'selected',target_nodes:['node-1']}}]});
  await login(page);
  await page.getByTestId('nav-rollouts').click();
  const row=page.locator('tr').filter({hasText:'Legacy parser rollout'});
  await expect(row).toContainText('Legacy snapshot · controller verification required');
  await expect(row.getByRole('button',{name:'Approve reviewed proposal'})).toBeEnabled();
  await row.locator('[data-rollout-reason]').fill('Reviewed legacy snapshot evidence');
  await row.getByRole('button',{name:'Approve reviewed proposal'}).click();
  await expect.poll(()=>mock.rolloutApprovalRequests).toHaveLength(1);
  const refreshed=page.locator('tr').filter({hasText:'Legacy parser rollout'});
  await expect(refreshed).toContainText('Cancel this proposal and create a new one from the current published bundle');
  await expect(refreshed.getByRole('button',{name:'Approve reviewed proposal'})).toBeDisabled();
  await expect(page.locator('.toast.bad')).toHaveCount(0);
  await expect(refreshed.getByRole('button',{name:'Cancel'})).toBeEnabled();
});

test('premium login presents the complete V21 banner in a responsive landscape stage', async ({ page }) => {
  await mockController(page);
  await page.goto(origin);
  const lockup=page.locator('.showcase-header .brand-emblem');
  const banner=page.locator('.login-banner-image');
  await expect(lockup).toHaveAttribute('src','./nexora-poster-v21.png');
  await expect(banner).toHaveAttribute('src','./login-banner-v21.png');
  await expect(page.locator('.showcase-header .brand-wordmark b')).toHaveText('Nexora Log Intelligence');
  await expect(page.locator('.showcase-header .brand-wordmark strong')).toHaveText('Nexora');
  await expect.poll(()=>banner.evaluate(image=>image.naturalWidth)).toBeGreaterThan(0);
  await expect(banner).toHaveCSS('object-fit','contain');
  const backdrop=await page.locator('.login-form-banner').evaluate(stage=>getComputedStyle(stage,'::before').backgroundImage);
  expect(backdrop).toContain('login-banner-v21.png');
  const desktop=await page.locator('.login-form-banner').evaluate(stage=>({width:stage.getBoundingClientRect().width,height:stage.getBoundingClientRect().height,imageWidth:stage.querySelector('img').naturalWidth,imageHeight:stage.querySelector('img').naturalHeight}));
  expect(desktop.width).toBeGreaterThan(500);
  expect(desktop.width/desktop.height).toBeCloseTo(1.6,1);
  expect(desktop.imageWidth).toBe(desktop.imageHeight);
  await page.setViewportSize({width:390,height:844});
  await expect(page.locator('.auth-showcase')).toBeHidden();
  await expect(page.locator('#login-form .login-form-banner')).toBeVisible();
  await expect(page.getByRole('button', {name:'Sign in securely'})).toBeVisible();
  const geometry=await page.locator('.login-form-banner').evaluate(stage=>({width:stage.getBoundingClientRect().width,height:stage.getBoundingClientRect().height,scrollWidth:document.documentElement.scrollWidth,clientWidth:document.documentElement.clientWidth}));
  expect(geometry.width).toBeGreaterThan(330);
  expect(geometry.width/geometry.height).toBeCloseTo(1.6,1);
  expect(geometry.scrollWidth).toBeLessThanOrEqual(geometry.clientWidth);
});

test('browser brand, Assurance selection, and account/audit timestamps stay explicit', async ({ page }) => {
  const mock = await mockController(page);
  mock.users.push({ username: 'test-operator', email: 'operator@example.test', role: 'admin', active: true, created_at: '2026-10-24T00:34:56Z', last_login: '2026-10-24T01:02:03Z' });
  mock.users.push({ username: 'never-signed-in', email: 'new@example.test', role: 'viewer', active: true, created_at: '0001-01-01T00:00:00Z', last_login: '0001-01-01T00:00:00Z' });
  mock.auditEvents.push({ sequence: 9, timestamp: '2026-10-24T02:03:04Z', actor: 'admin', role: 'admin', action: 'user.approved', resource: 'test-operator', outcome: 'approved' });
  await login(page);
  await expect(page.locator('link[rel="icon"]')).toHaveAttribute('href', './nli-mark.svg');
  await page.getByTestId('nav-assurance').click();
  const corpusTab = page.locator('[data-assurance-tab="corpus"]');
  await corpusTab.click();
  await expect(corpusTab).toHaveClass(/active/);
  await expect(corpusTab).toHaveAttribute('aria-selected', 'true');
  await page.getByTestId('nav-security').click();
  const usersPanel = page.locator('.panel').filter({ has: page.getByRole('heading', { name: 'Users' }) });
  const created = usersPanel.locator('tr', { hasText: 'test-operator' }).locator('td').nth(4);
  const expected = await page.evaluate(() => new Intl.DateTimeFormat(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', timeZoneName: 'short' }).format(new Date('2026-10-24T00:34:56Z')));
  await expect(created).toHaveText(expected);
  await expect(usersPanel.locator('tr', { hasText: 'never-signed-in' }).locator('td').nth(5)).toHaveText('—');
  expect(expected).toMatch(/\d{2}:\d{2}:\d{2}/);
  const auditPanel = page.locator('.panel').filter({ has: page.getByRole('heading', { name: 'Immutable audit chain' }) });
  const approvalTime = auditPanel.locator('tr', { hasText: 'user.approved' }).locator('td').nth(1);
  const approvalExpected = await page.evaluate(() => new Intl.DateTimeFormat(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', timeZoneName: 'short' }).format(new Date('2026-10-24T02:03:04Z')));
  await expect(approvalTime).toHaveText(approvalExpected);
  await page.getByTestId('nav-communications').click();
  await expect(page.getByText('Client → Controller', { exact: true })).toBeVisible();
  await page.getByTestId('nav-notifications').click();
  await expect(page.getByText('not_configured').first()).toBeVisible();
  await page.getByLabel('Enable SMTP email').check();
  await page.getByLabel('Enable PagerDuty').check();
  await page.getByLabel('Recipient usernames, comma separated').fill('admin, oncall-1');
  await page.getByLabel('SMTP server environment variable').fill('NLI_SMTP_SERVER_ENV');
  await page.getByLabel('Email sender environment variable').fill('NLI_MAIL_FROM_ENV');
  await page.getByLabel('Email recipient-list environment variable').fill('NLI_MAIL_TO_ENV');
  await page.getByLabel('SMTP username environment variable').fill('NLI_SMTP_USER_ENV');
  await page.getByLabel('SMTP password environment variable').fill('NLI_SMTP_PASS_ENV');
  await page.getByLabel('PagerDuty routing-key environment variable').fill('NLI_PD_KEY_ENV');
  await page.getByRole('button', { name: 'Save notification policy' }).click();
  await expect.poll(() => mock.notificationPolicySaves.length).toBe(1);
  expect(mock.notificationPolicySaves[0]).toEqual({ channels: { in_app: { enabled: true }, email: { enabled: true, server_env: 'NLI_SMTP_SERVER_ENV', sender_env: 'NLI_MAIL_FROM_ENV', recipient_env: 'NLI_MAIL_TO_ENV', username_env:'NLI_SMTP_USER_ENV', password_env:'NLI_SMTP_PASS_ENV' }, pagerduty: { enabled: true, secret_env: 'NLI_PD_KEY_ENV' } }, admin_fallback: true, recipients: ['admin','oncall-1'], events: ['approval','quarantine','sync','system','delivery','security'] });
  expect(JSON.stringify(mock.notificationPolicySaves[0])).not.toMatch(/secret-value|routing-key-value/i);
  expect(mock.observed).toContain('POST /api/v1/communications/browser-heartbeat');
});

test('notification count and per-user read, check-later, and delete actions stay synchronized', async ({ page }) => {
  const mock=await mockController(page,{controllerNotifications:[
    {id:'audit-41',sequence:41,title:'Parser approval required',resource:'parser-41',severity:'warning',status:'unread',read:false,timestamp:'2026-10-30T02:00:00Z'},
    {id:'audit-42',sequence:42,title:'Quarantine review required',resource:'record-42',severity:'critical',status:'unread',read:false,timestamp:'2026-10-30T02:01:00Z'},
  ]});
  await login(page);
  await expect(page.locator('#notification-count')).toHaveText('2');
  await page.locator('#notification-bell').click();
  await page.locator('[data-notification-id="audit-41"] [data-notification-action="read"]').click();
  await expect(page.locator('#notification-count')).toHaveText('1');
  await page.locator('[data-notification-id="audit-42"] [data-notification-action="snooze"]').click();
  await expect(page.locator('#notification-count')).toBeHidden();
  await page.getByTestId('nav-notifications').click();
  await expect(page.getByText('0 unread operator action(s).')).toBeVisible();
  expect(mock.notificationActions).toEqual([
    {sequence:41,action:'read',method:'POST'},
    {sequence:42,action:'snooze',method:'POST'},
  ]);
});

test('Assurance Lab tabs isolate each workflow and expose complete generated outputs', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab', {name:'Parser Creation Helper'}).click();
  await expect(page.locator('#assurance-panel-helper')).toBeVisible();
  await expect(page.locator('#assurance-panel-analysis')).toBeHidden();
  const form=page.locator('#parser-helper-form');
  await form.locator('[name=samples]').fill('level=INFO user=alice');
  await page.getByRole('button', {name:'Generate parser draft'}).click();
  await expect(page.locator('.parser-helper-output')).toHaveCount(6);
  for(const label of ['Generated JSON','Regex','Templates','KV rules','Field mappings','Field definitions']) await expect(page.getByRole('heading',{name:label})).toBeVisible();
  await expect(page.getByRole('button',{name:'Copy Regex'})).toBeVisible();
});

test('NAPI Synthesis validates separate datasets and exposes review-gated evidence', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab',{name:'NAPI Synthesis',exact:true}).click();
  const form=page.getByTestId('napi-synthesis-form');
  await expect(form).toBeVisible();
  await form.getByRole('button',{name:'Synthesize and validate draft'}).click();
  const result=page.getByTestId('napi-synthesis-result');
  await expect(result).toContainText('anchored-regex');
  await expect(result).toContainText('requires_human_approval');
  await expect(result).toContainText('raw_samples_persisted');
  await expect(result).toContainText('false');
});

test('Assurance Test Plans saves versioned declarative plans through the controller API', async ({ page }) => {
  const { assurancePlans } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab', {name:'Test Plans'}).click();
  await expect(page.locator('#assurance-panel-plans')).toBeVisible();
  const form=page.locator('#assurance-plan-form');
  await form.locator('[name=id]').fill('smtp-certification');
  await form.locator('[name=name]').fill('SMTP certification');
  await form.locator('[name=version]').fill('1.0.0');
  await form.getByRole('button',{name:'Save test-plan revision'}).click();
  await expect.poll(()=>assurancePlans.length).toBe(1);
  expect(assurancePlans[0].payload.plan.scenarios).toEqual(['parser-regression','failure-recovery','replay','destination-integration','performance','security']);
  await expect(page.getByText('SMTP certification',{exact:true})).toBeVisible();
});


test('Assurance Runs use an exact plan revision and retained fixture and display persisted evidence', async ({ page }) => {
  const { assuranceRuns } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab', {name:'Test Plans'}).click();
  const planForm=page.locator('#assurance-plan-form');
  await planForm.locator('[name=id]').fill('parser-certification');
  await planForm.locator('[name=plan]').fill(JSON.stringify({objective:'Certify parser',environment:'synthetic-lab',scenarios:['parser-regression'],pass_criteria:['exact expected output']}));
  await planForm.getByRole('button',{name:'Save test-plan revision'}).click();
  await page.getByRole('tab',{name:'Runs',exact:true}).click();
  await expect(page.locator('#assurance-run-form')).toBeVisible();
  await expect(page.getByTestId('assurance-run-scenario').locator('option')).toHaveCount(6);
  await page.getByTestId('assurance-run-start').click();
  await expect.poll(()=>assuranceRuns.length).toBe(1);
  expect(assuranceRuns[0].payload).toMatchObject({plan_id:'parser-certification',plan_revision:1,case_id:'fixture-42',state:'passed'});
  await expect(page.getByTestId('assurance-run-row')).toContainText('parser-certification');
  await expect(page.getByTestId('assurance-run-row')).toContainText('b'.repeat(64));
});

test('AI workspace exposes provider, status, results, parser, regex, template and approval categories', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-ai').click();
  for(const category of ['Provider','Status','Results','Parser','Regex','Template','Approval']){
    const tab=page.getByRole('tab',{name:category,exact:true});
    await tab.click();
    await expect(tab).toHaveAttribute('aria-selected','true');
    await expect(page.locator(`#ai-panel-${category.toLowerCase()}`)).toBeVisible();
  }
});

test('AI provider manager exposes every supported provider and redacted credential mode', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-ai').click();
  await page.getByRole('tab',{name:'Provider',exact:true}).click();
  const form=page.locator('#ai-provider-form');
  await expect(form).toBeVisible();
  await expect(form.locator('[name=provider] option')).toHaveCount(8);
  for(const value of ['openai','nvidia_nim','anthropic','minimax','qwen_dashscope','gemini','openai_compatible','ollama']) {
    await expect(form.locator(`[name=provider] option[value="${value}"]`)).toHaveCount(1);
  }
  await expect(form.locator('[name=credential_source] option')).toHaveCount(4);
  await expect(page.locator('#ai-panel-provider').getByRole('row',{name:/NVIDIA NIM/})).toBeVisible();
  await expect(page.locator('#ai-panel-provider').getByRole('row',{name:/Gemini \/ Google/})).toBeVisible();
  await expect(page.locator('body')).not.toContainText('ciphertext');
});

test('V21 security identity and task navigation remain accessible', async ({ page }) => {
  await mockController(page);
  await page.goto(origin);
  const officialLogo = page.getByAltText('Nexora Log Intelligence logo').first();
  await expect(officialLogo).toBeVisible();
  await expect(officialLogo).toHaveJSProperty('complete', true);
  expect(await officialLogo.evaluate(image => image.naturalWidth)).toBeGreaterThan(0);
  await login(page);
  const groups = await page.locator('.nav-group').allTextContents();
  for (const group of ['COMMAND CENTER','ASSURANCE & RELEASE','OPERATIONS','REAL-TIME SUBSYSTEMS','INPUT', 'PARSING ENGINE', 'OUTPUT', 'CLIENT MANAGEMENT', 'CONTROLLER MANAGEMENT', 'PARSER MANAGEMENT']) {
    expect(groups).toContain(group);
  }
  expect(groups[groups.indexOf('ASSURANCE & RELEASE') + 1]).toBe('PARSER MANAGEMENT');
  await page.getByTestId('nav-fleet').click();
  await expect(page.getByText('Enrolled clients', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Client monitoring dimensions')).toContainText('Parser activity');
});

test('eagle identity honors reduced-motion preference', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await mockController(page);
  await page.goto(origin);
  const logo = page.locator('.eagle-brand .brand-logo');
  await expect(logo).toBeVisible();
  await expect(logo).toHaveCSS('animation-name', 'none');
  await expect(page.locator('.eagle-orbit')).toHaveCSS('animation-name', 'none');
});

test('streams render live per-node rates and file processing timings', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-streams').click();
  await expect(page.getByRole('heading', { name: 'Streams', exact: true, level: 1 })).toBeVisible();
  await expect(page.locator('#stream-live-table')).toContainText('/var/log/mail.log');
  await expect(page.locator('#stream-live-table')).toContainText('90.0');
  await expect(page.locator('#stream-live-table')).toContainText('4 / 128');
  await expect(page.locator('#stream-live-table')).toContainText('tailing');
  await page.getByTestId('nav-performance').click();
  await expect(page.getByRole('heading', { name: 'Canonical throughput and latency' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Per-source processing and freshness' })).toBeVisible();
  await expect(page.locator('#page')).toContainText('parser latency exceeded warning threshold');
});

test('operations reuses canonical sync and only fetches one aggregate history series', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  const start = observed.length;
  await page.getByTestId('nav-operations').click();
  const requests = observed.slice(start);
  expect(requests.filter(item => item.startsWith('GET /api/v1/dashboard/snapshot'))).toHaveLength(0);
  // The shared snapshot recovery loop may race the navigation while the
  // mocked EventSource is opening.  It is the one permitted canonical
  // fallback. The retained timeline is one aggregate controller query; it
  // must not fan out into fleet, metadata, system, or overview polling.
  expect(requests.filter(item => item === 'GET /api/v1/sync/snapshot').length).toBeLessThanOrEqual(1);
  expect(requests.filter(item => item.startsWith('GET /api/v1/metrics/timeseries')).length).toBeLessThanOrEqual(1);
  expect(requests.some(item => /GET \/api\/v1\/(overview|metadata\?|system|fleet\/sync)/.test(item))).toBe(false);
});

test('live dashboard renders controller metrics, client filter, pipeline timing and safe failures', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await expect(page.locator('#page').getByRole('heading', { name: 'Command Center' })).toBeVisible();
  await expect(page.getByTestId('live-dashboard-kpis').locator('.dashboard-kpi strong').first()).toHaveText('9,007,199,254,740,993');
  await expect(page.getByText('100.00% success')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Pipeline time travel' })).toBeVisible();
  await expect(page.getByText('/var/log/mail.log').first()).toBeVisible();
  await expect(page.locator('#page')).not.toContainText('service=');
  const before=observed.length;
  await page.locator('#dashboard-client').selectOption('node-1');
  expect(observed.slice(before).some(item=>item.startsWith('GET /api/v1/dashboard/snapshot'))).toBe(false);
  await page.getByTestId('nav-monitoring').click();
  await page.getByRole('tab', { name: 'Parser Performance' }).click();
  await expect(page.locator('[data-monitoring-panel=parsers]')).toContainText('Parser successes');
  await expect(page.locator('[data-monitoring-panel=parsers]')).toContainText('Parse p95');
  await expect(page.locator('[data-monitoring-panel=parsers]')).toContainText('Source workers');
});

test('canonical SSE reconnect resumes from the last accepted revision without legacy polling', async ({ page }) => {
  await page.clock.install();
  await mockController(page);
  await page.addInitScript(() => {
    const instances=[];
    class FailingEventSource extends EventTarget {
      constructor(url){super();this.url=url;this.closed=false;instances.push(this);window.__failingSources=instances;setTimeout(()=>this.onerror?.(new Event('error')),0);}
      close(){this.closed=true;}
    }
    window.EventSource=FailingEventSource;
  });
  await login(page);
  await page.getByTestId('nav-streams').click();
  await expect.poll(() => page.evaluate(() => window.__failingSources?.length||0)).toBe(1);
  await page.evaluate(() => window.__failingSources[0].onerror?.(new Event('error')));
  await page.clock.fastForward(5_100);
  await expect.poll(() => page.evaluate(() => window.__failingSources.length)).toBeGreaterThan(1);
  expect(await page.evaluate(() => window.__failingSources.at(-1).url)).toContain('last_revision=7');
  await expect(page.locator('#stream-live-table')).toContainText('/var/log/mail.log');
});

test('canonical SSE deltas update stream rows and reconnect from the accepted revision', async ({ page }) => {
  await mockController(page);
  await page.addInitScript(() => {
    const instances=[];
    class FakeEventSource extends EventTarget {
      constructor(url){super();this.url=url;this.closed=false;instances.push(this);window.__streamSources=instances;setTimeout(()=>this.onopen?.(),0);}
      close(){this.closed=true;}
      emit(type,data,id=''){const event=new MessageEvent(type,{data:JSON.stringify(data)});Object.defineProperty(event,'lastEventId',{value:id});this.dispatchEvent(event);}
    }
    window.EventSource=FakeEventSource;
  });
  await login(page);
  await page.getByTestId('nav-streams').click();
  await expect(page.locator('#stream-live-table')).toContainText('node-1');
  await expect.poll(() => page.evaluate(() => window.__streamSources?.length||0)).toBeGreaterThan(0);
  await page.evaluate(() => window.__streamSources.at(-1).emit('sync_delta',{schema_version:'nli.sync.v1',revision:8,generated_at:new Date().toISOString(),target_interval_ms:1000,aggregate:{nli_source_records_read_total:132,nli_parser_records_succeeded_total:126,nli_parser_records_failed_total:1,nli_normalized_records_total:126,nli_quarantined_records_total:6,nli_queue_depth:5,read_lines_per_second:102,normalized_per_second:97,quarantined_per_second:5,bytes_per_second:4608},source:{node_id:'node-live',source:'/var/log/live.log',process_generation:'gen-live',sequence:1,phase:'tailing',counters:{nli_source_records_read_total:12,nli_parser_records_succeeded_total:10,nli_parser_records_failed_total:0,nli_normalized_records_total:10,nli_quarantined_records_total:2},rates:{read_lines_per_second:12,normalized_per_second:10,quarantined_per_second:2,bytes_per_second:512},queue:{nli_queue_depth:1,queue_capacity:20,backpressure:false},controller_received_at:new Date().toISOString(),controller_processing_ms:1,observed_propagation_delay_ms:5,freshness:'live',freshness_age_ms:5,freshness_reason:'within target',critical:false,diagnostic_codes:[]},diagnostics:{subscribers:1,tracked_sources:2,live:2,delayed:0,stale:0,disconnected:0,issues:[]},subsystems:{nhle:{status:'live',source_count:2,active_nodes:2,records_read_total:132,normalized_total:126,quarantined_total:6,queue_depth:5,read_lines_per_second:102,normalized_per_second:97},napi:{status:'live'},nal:{status:'live'}}},'8'));
  await expect(page.locator('#stream-live-table')).toContainText('node-live');
  await expect(page.locator('#stream-live-table')).toContainText('12.0');
  await page.evaluate(() => window.__streamSources.at(-1).emit('sync_delta',{schema_version:'nli.sync.v1',revision:9,generated_at:new Date().toISOString(),target_interval_ms:1000,aggregate:{nli_source_records_read_total:140,nli_parser_records_succeeded_total:134,nli_parser_records_failed_total:1,nli_normalized_records_total:134,nli_quarantined_records_total:5,nli_queue_depth:4,read_lines_per_second:110,normalized_per_second:105,quarantined_per_second:4,bytes_per_second:4796},source:{node_id:'node-live',source:'/var/log/live.log',process_generation:'gen-live',sequence:2,phase:'eof',counters:{nli_source_records_read_total:20,nli_parser_records_succeeded_total:18,nli_parser_records_failed_total:0,nli_normalized_records_total:18,nli_quarantined_records_total:1},rates:{read_lines_per_second:20,normalized_per_second:18,quarantined_per_second:1,bytes_per_second:700},queue:{nli_queue_depth:0,queue_capacity:20,backpressure:false},controller_received_at:new Date().toISOString(),controller_processing_ms:1,observed_propagation_delay_ms:5,freshness:'live',freshness_age_ms:5,freshness_reason:'within target',critical:false,diagnostic_codes:[]},diagnostics:{subscribers:1,tracked_sources:2,live:2,delayed:0,stale:0,disconnected:0,issues:[]},subsystems:{nhle:{status:'live',source_count:2,active_nodes:2,records_read_total:140,normalized_total:134,quarantined_total:5,queue_depth:4,read_lines_per_second:110,normalized_per_second:105},napi:{status:'live'},nal:{status:'live'}}},'9'));
  await expect(page.locator('#stream-live-table')).toContainText('20.0');
	await page.getByTestId('nav-operations').click();
	await expect(page.locator('.throughput-gauges')).toContainText('110.0/s');
	await page.evaluate(() => window.__streamSources.at(-1).onerror?.(new Event('error')));
	await new Promise(resolve => setTimeout(resolve, 1_100));
  expect(await page.evaluate(() => window.__streamSources.at(-1).url)).toContain('last_revision=9');
});

test('equal-revision canonical heartbeat refreshes freshness without replacing the focused workspace', async ({ page }) => {
  await mockController(page);
  await page.addInitScript(() => {
    const instances=[];
    class FakeEventSource extends EventTarget {
      constructor(url){super();this.url=url;instances.push(this);window.__streamSources=instances;setTimeout(()=>this.onopen?.(),0);}
      close(){this.closed=true;}
      emit(type,data){this.dispatchEvent(new MessageEvent(type,{data:JSON.stringify(data)}));}
    }
    window.EventSource=FakeEventSource;
  });
  const snapshotResponse=page.waitForResponse(response=>new URL(response.url()).pathname==='/api/v1/sync/snapshot');
  await login(page);
  const snapshot=await (await snapshotResponse).json();
  await page.getByTestId('nav-operations').click();
  const range=page.locator('#dashboard-range');
  await range.focus();
  await range.evaluate(node=>{window.__heartbeatInput=node;window.__heartbeatRoot=document.querySelector('#page').firstElementChild;});
  const historical=page.getByTestId('live-dashboard-kpis').locator('.dashboard-kpi').filter({hasText:'Quarantined'}).locator('strong');
  const originalCount=await historical.textContent();
  const replay=page.getByTestId('dashboard-replay');
  await expect(replay.locator('[data-replay-counter="reprocessing"]')).toHaveText('1');
  await page.evaluate(value=>window.__streamSources.at(-1).emit('sync_heartbeat',{schema_version:value.schema_version,revision:value.revision,generated_at:new Date().toISOString(),target_interval_ms:value.target_interval_ms,aggregate:value.aggregate,sources:value.sources,diagnostics:value.diagnostics,subsystems:{replay:{status:'ready',queued:1,reprocessing:0,resolved:3,manual_attention:0,by_node:{'node-1':{queued:1,resolved:2}}}}}),snapshot);
  await expect(replay.locator('[data-replay-counter="reprocessing"]')).toHaveText('0');
  await expect(replay.locator('[data-replay-counter="resolved"]')).toHaveText('3');
  await expect(historical).toHaveText(originalCount);
  await page.evaluate(value=>{
    const {subsystems,...heartbeat}=value;
    window.__streamSources.at(-1).emit('sync_heartbeat',heartbeat);
  },snapshot);
  await expect(replay.locator('[data-replay-counter="resolved"]')).toHaveText('3');
  await expect.poll(()=>page.evaluate(()=>({focused:document.activeElement===window.__heartbeatInput,connected:window.__heartbeatInput?.isConnected,root:document.querySelector('#page').firstElementChild===window.__heartbeatRoot}))).toEqual({focused:true,connected:true,root:true});
  await expect(page.locator('#freshness')).toContainText('live');
  await page.locator('#dashboard-client').selectOption('node-1');
  await expect(replay.locator('[data-replay-counter="resolved"]')).toHaveText('2');
  await page.getByTestId('nav-napi-monitor').click();
  await expect(page.locator('#page')).toContainText('Pending drafts');
  await expect(page.locator('#page')).toContainText('3');
});

test('canonical updates defer a global dashboard rerender while a select is focused', async ({ page }) => {
  await mockController(page);
  await page.addInitScript(() => {
    const instances=[];
    class FakeEventSource extends EventTarget {
      constructor(url){super();this.url=url;instances.push(this);window.__streamSources=instances;setTimeout(()=>this.onopen?.(),0);}
      close(){}
      emit(type,data){this.dispatchEvent(new MessageEvent(type,{data:JSON.stringify(data)}));}
    }
    window.EventSource=FakeEventSource;
  });
  const response=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/sync/snapshot');
  await login(page);
  const snapshot=await (await response).json();
  await page.getByTestId('nav-operations').click();
  const range=page.locator('#dashboard-range');
  await range.focus();
  await range.evaluate(node=>{window.__focusedSelect=node;});
  await page.evaluate(value=>window.__streamSources.at(-1).emit('sync_delta',{schema_version:value.schema_version,revision:Number(value.revision)+1,generated_at:new Date().toISOString(),target_interval_ms:value.target_interval_ms,aggregate:{...value.aggregate,nli_queue_depth:17},source:{node_id:'node-1',source:'/var/log/mail.log',sequence:1,phase:'tailing',counters:{},rates:{},queue:{nli_queue_depth:17,queue_capacity:128},freshness:'live',freshness_age_ms:0},diagnostics:value.diagnostics,subsystems:value.subsystems}),snapshot);
  await expect.poll(()=>page.evaluate(()=>({connected:window.__focusedSelect?.isConnected,focused:document.activeElement===window.__focusedSelect}))).toEqual({connected:true,focused:true});
  await range.evaluate(node=>node.blur());
  await expect(page.locator('#dashboard-range')).toHaveCount(1);
  await expect(page.locator('.throughput-gauges')).toContainText('17');
});

test('shared snapshot recovery keeps canonical freshness live when SSE heartbeats stop', async ({ page }) => {
  await page.clock.install();
  const { observed } = await mockController(page);
  await page.addInitScript(() => {
    class QuietEventSource extends EventTarget {
      constructor(){super();setTimeout(()=>this.onopen?.(),0);}
      close(){}
    }
    window.EventSource=QuietEventSource;
  });
  await login(page);
  await page.clock.fastForward(6_100);
  await expect(page.locator('#stream-state')).toContainText('Live sync');
  await expect.poll(()=>observed.filter(item=>item==='GET /api/v1/sync/snapshot').length).toBeGreaterThan(1);
  await expect(page.locator('#freshness')).toContainText('live');
  await expect(page.locator('.stale-banner')).toHaveCount(0);
});

test('production Assurance response fills helper using normalized backend proposal and saves a draft', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByTestId('assurance-raw').fill('level=INFO user=alice   \r\nlevel=WARN user=bob\t  \r\n');
  await page.getByRole('button', { name: 'Run assurance', exact: true }).click();
  await page.getByRole('tab', { name: 'Parser Creation Helper' }).click();
  const form=page.locator('#parser-helper-form');
  await expect(form.locator('[name=samples]')).toHaveValue('level=INFO user=alice\nlevel=WARN user=bob');
  const parser=JSON.parse(await form.locator('[name=parser]').inputValue());
  expect(parser.known_fields).toEqual(['level', 'user']);
  expect(parser.header_pattern).toMatchObject({ kind: 'line', prefix: 'level=' });
  expect(parser.template).toEqual({ level: '{level}', user: '{user}' });
  await expect(page.locator('#parser-helper-overview')).toContainText('key_value');
  await form.getByRole('button', { name: 'Save review-gated draft' }).click();
  await expect.poll(()=>observed.includes('POST /api/v1/assurance/parsers')).toBe(true);
  expect(observed.some(path=>/assurance\/parsers\/.*\/(approve|publish)/.test(path))).toBe(false);
});

test('Assurance helper trims trailing whitespace, generates editable parser, validates and saves only a draft', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await expect(page.getByRole('tablist', { name: 'Assurance Lab tools' })).toBeVisible();
  const labels = await page.getByRole('tablist', { name: 'Assurance Lab tools' }).getByRole('tab').allTextContents();
  expect(labels).toEqual(['Analyse', 'Corpus reports', 'Parser drafts', 'Review Queue', 'History', 'Fixtures', 'Guided Parser Repair Studio', 'Parser Creation Helper', 'NAPI Synthesis', 'Test Plans', 'Runs']);
  await page.getByRole('tab', { name: 'Guided Parser Repair Studio' }).click();
  await expect(page.getByText(/active parser is unchanged until the existing approval and deployment flow completes/i)).toBeVisible();
  await page.getByRole('tab', { name: 'Parser Creation Helper' }).click();
  const form = page.locator('#parser-helper-form');
  await form.locator('[name=samples]').fill('level=INFO user=alice   \r\nlevel=WARN user=bob\t  \r\n');
  await page.getByRole('button', { name: 'Generate parser draft' }).click();
  await expect(form.locator('[name=samples]')).toHaveValue('level=INFO user=alice\nlevel=WARN user=bob');
  const parser = JSON.parse(await form.locator('[name=parser]').inputValue());
  expect(parser.pattern_type).toBe('regex');
  expect(parser.known_fields).toEqual(['level', 'user']);
  await expect(page.locator('#parser-helper-overview')).toContainText('key_value_regex');
  await form.locator('[name=cases]').fill(JSON.stringify([
    { raw: 'level=INFO user=alice', service: 'key_value', desired: { level: 'INFO', user: 'alice' } },
    { raw: 'level=WARN user=bob', service: 'key_value', desired: { level: 'WARN', user: 'bob' } },
  ], null, 2));
  await page.getByRole('button', { name: 'Validate draft', exact: true }).click();
  await expect.poll(() => observed.includes('POST /api/v1/assurance/parser-preview')).toBe(true);
  await form.getByRole('button', { name: 'Save review-gated draft' }).click();
  await expect.poll(() => observed.includes('POST /api/v1/assurance/parsers')).toBe(true);
  expect(observed.some(path => /assurance\/parsers\/.*\/(approve|publish)/.test(path))).toBe(false);
});

test('quarantine diagnosis stays in the self-contained Resolution Wizard', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-unknowns').click();
  await page.getByRole('button', { name: 'Open in Resolution Wizard' }).click();
  await expect.poll(() => observed.includes('POST /api/v1/quarantine/records/q-1/analyze')).toBe(true);
  await expect(page.locator('#page h1').filter({hasText:'Quarantine Resolution Wizard'})).toBeVisible();
  await expect(page.locator('#gprs-preview-form [name=raw_evidence]')).toHaveValue('exact raw source line');
  await expect(page.locator('#gprs-parser-helper-form')).toHaveCount(1);
  await expect(page.getByRole('heading', { name: 'Assurance Lab' })).toHaveCount(0);
});

test('Parser Creation Workspace builds a complete editable draft without publishing', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-parser-workspace').click();
  const form = page.locator('#parser-workspace-form');
  await form.locator('[name=samples]').fill('level=INFO user=alice   \r\nlevel=WARN user=bob\t  \r\n');
  const autoPopulate = page.getByRole('button', { name: 'Auto-populate complete parser' });
  await expect(autoPopulate).toBeVisible();
  await expect(autoPopulate).toBeEnabled();
  await autoPopulate.click();
  await expect(form.locator('[name=samples]')).toHaveValue('level=INFO user=alice\nlevel=WARN user=bob');
  const parser = JSON.parse(await form.locator('[name=parser]').inputValue());
  expect(parser).toMatchObject({ pattern_type: 'regex', known_fields: ['level', 'user'], projection: ['level', 'user'] });
  await page.getByRole('button', { name: 'Validate locally' }).click();
  await expect(form.locator('[data-parser-status]')).toContainText('Local structure valid');
  await page.getByRole('button', { name: 'Save approval-gated draft' }).click();
  await expect.poll(() => observed.includes('POST /api/v1/assurance/parsers')).toBe(true);
  expect(observed.some(path => /assurance\/parsers\/.*\/(approve|publish)/.test(path))).toBe(false);
});

test('operations does not fan out beyond shared canonical recovery', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  const before = observed.length;
  await page.getByTestId('nav-operations').click();
  const requests = observed.slice(before);
  expect(requests.filter(item => item.startsWith('GET /api/v1/dashboard/snapshot'))).toHaveLength(0);
  expect(requests.filter(item => item === 'GET /api/v1/sync/snapshot').length).toBeLessThanOrEqual(1);
  expect(requests.filter(item => item.startsWith('GET /api/v1/metrics/timeseries')).length).toBeLessThanOrEqual(1);
  expect(requests.some(item => /\/api\/v1\/(overview|metadata|system|fleet\/sync)/.test(item))).toBe(false);
});

test('missing EventSource uses shared recovery only and signout aborts it cleanly', async ({ page }) => {
  await page.clock.install();
  const { observed } = await mockController(page);
  await page.addInitScript(() => { window.EventSource = undefined; });
  await login(page);
  await expect(page.getByTestId('nav-streams')).toBeVisible();
  await page.clock.fastForward(5_100);
  expect(observed.filter(item => item === 'GET /api/v1/telemetry/stream/snapshot')).toHaveLength(0);
  await page.getByRole('button', { name: 'Sign out' }).click();
  const afterSignout = observed.filter(item => item === 'GET /api/v1/sync/snapshot').length;
  await page.clock.fastForward(15_000);
  expect(observed.filter(item => item === 'GET /api/v1/sync/snapshot')).toHaveLength(afterSignout);
});

test('API monitor starts its independent probes concurrently and displays latency percentiles', async ({ page }) => {
  const starts = [];
  await mockController(page);
  await page.route(`${origin}/api/v1/**`, async route => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/api/v1/auth/login' || path === '/api/v1/auth/csrf' || path === '/api/v1/auth/me' || path === '/api/v1/health' || path === '/api/v1/sync/snapshot' || path === '/api/v1/sync/stream') return route.fallback();
    starts.push(Date.now());
    await new Promise(resolve => setTimeout(resolve, 80));
    await route.fallback();
  });
  await login(page);
  starts.length = 0;
  await page.getByTestId('nav-api-monitor').click();
  await expect(page.getByRole('heading', { name: 'API monitor', exact: true })).toBeVisible();
  await expect(page.getByText('Latency p50')).toBeVisible();
  await expect(page.getByText('Latency p95')).toBeVisible();
  expect(starts.length).toBeGreaterThan(10);
  // Chromium limits parallel connections per origin. The first transport wave
  // proves that independent probes are dispatched concurrently. Keep the bound
  // below the 5 * 80 ms a sequential first wave would require, while allowing
  // for loaded CI runners to schedule intercepted requests with modest jitter.
  expect(Math.max(...starts.slice(0, 6)) - Math.min(...starts.slice(0, 6))).toBeLessThan(300);
});

test('Endpoint Configuration tests and saves a safe ClickHouse field-mapped profile', async ({ page }) => {
  const mock = await mockController(page);
  await login(page);
  await page.getByTestId('nav-outputs').click();
  await expect(page.getByRole('heading', { name: 'Endpoint Configuration', level: 1 })).toBeVisible();
  await expect(page.getByRole('tablist', { name: 'Endpoint configuration sections' }).getByRole('tab')).toHaveText(['Destinations', 'Security & credentials', 'Validation', 'Saved profiles', 'Live & rollback']);
  const form = page.locator('#output-profile-form');
  await expect(form.locator('[name=normalised_enabled]')).toBeVisible();
  await page.getByRole('tab', { name: 'Security & credentials' }).click();
  await expect(form.locator('[name=clickhouse_password_env]')).toBeVisible();
  await page.getByRole('tab', { name: 'Validation' }).click();
  await expect(form.locator('[name=field_columns]')).toBeVisible();
  await expect(form.getByRole('button', { name: 'Validate configuration' })).toBeVisible();
  await page.getByRole('tab', { name: 'Saved profiles' }).click();
  await expect(page.getByRole('heading', { name: 'Saved output profiles' })).toBeVisible();
  await page.getByRole('tab', { name: 'Live & rollback' }).click();
  await expect(page.getByRole('heading', { name: 'Roll out endpoint profile' })).toBeVisible();
  await expect(page.locator('#output-rollout-form')).toBeVisible();
  await page.getByRole('tab', { name: 'Destinations' }).click();
  await form.locator('[name=normalised_enabled]').uncheck();
  await form.locator('[name=destinations][value=clickhouse]').check();
  await form.locator('[name=clickhouse_url]').fill('https://clickhouse.example.com:8443');
  await page.getByRole('tab', { name: 'Validation' }).click();
  await form.locator('[name=field_columns]').fill('{"fromhost":"client_ip"}');
  await form.getByRole('button', { name: 'Validate configuration' }).click();
  await expect(page.locator('#output-config-test-result')).toContainText('Local profile validation passed');
  await expect(page.locator('#output-config-test-result')).toContainText('Network connectivity was not tested');
  await expect(page.locator('#output-profile-preview')).toContainText('normal');
  await form.getByRole('button', { name: 'Validate and save draft' }).click();
  await expect(page.locator('#output-profile-preview')).toContainText('client_ip');
  expect(mock.outputSaves).toHaveLength(1);
  const outputs = mock.outputSaves[0].config.outputs;
  expect(outputs.find(item => item.type === 'clickhouse')).toMatchObject({ url: 'https://clickhouse.example.com:8443/', field_columns: { fromhost: 'client_ip' }, fallback_path: '/var/log/nli/output/delivery-fallback.jsonl', routes: ['normal', 'warning'] });
  expect(outputs.some(item => item.type === 'file' && item.routes.includes('quarantine'))).toBe(true);
  expect(outputs.some(item => item.type === 'file' && item.routes.includes('normal'))).toBe(false);
});

test('published endpoint profile creates an outputs-only approval rollout', async ({ page }) => {
  const mock = await mockController(page);
  const profile = { id: 'remote-delivery', revision: 4, state: 'active', updated_at: '2026-10-14T00:00:00Z', payload: { name: 'Remote delivery', config: { outputs: [
    { type: 'loki', url: 'https://loki.example/loki/api/v1/push', routes: ['normal', 'warning'], fallback_path: '/var/lib/nli/spool/loki.jsonl' },
    { type: 'file', path_template: '/var/log/nli/output/auto_quarantine_{hour_utc}.jsonl', routes: ['quarantine', 'unknown'], durable_raw: true },
  ] } } };
  mock.outputProfiles.push(profile);
  await login(page);
  await page.getByTestId('nav-outputs').click();
  await page.getByRole('tab', { name: 'Live & rollback' }).click();
  const form = page.locator('#output-rollout-form');
  await form.locator('[name=profile_id]').selectOption('remote-delivery');
  await form.locator('[name=targets]').fill('node-a, node-b');
  await form.getByRole('button', { name: 'Request independent approval' }).click();
  await expect.poll(() => mock.rolloutSaves.length).toBe(1);
  expect(mock.rolloutSaves[0]).toMatchObject({ type: 'config_push', target_nodes: ['node-a', 'node-b'], payload: { config_patch: { outputs: profile.payload.config.outputs } } });
});

test('Parser Fleet Sync shows live revision, drift, rollout and approval-gated verify/retry actions', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-parser-sync').click();
  await expect(page.locator('#page')).toContainText('Nginx access pattern');
  await expect(page.locator('#page')).toContainText('published');
  await expect(page.locator('#page')).toContainText('node-1');
  await expect(page.locator('#page')).toContainText('1 / 1 complete');
  await page.getByRole('button', { name: 'Verify' }).first().click();
  await expect.poll(() => observed.includes('POST /api/v1/parser-sync/verify')).toBe(true);
  await page.getByRole('button', { name: 'Propose retry' }).last().click();
  await expect.poll(() => observed.includes('POST /api/v1/parser-sync/retry')).toBe(true);
  await expect(page.locator('#toast-region')).toContainText('independent approval is required');
});

test('Quarantine queue exposes exact raw access, exports and auditable replay controls', async ({ page }) => {
  const { observed } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-unknowns').click();
  await expect(page.getByRole('heading', { name: 'Quarantine and reprocessing' })).toBeVisible();
  await expect(page.getByText('exact raw source line')).toHaveCount(0);
  const download = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Download exact raw' }).click();
  expect((await download).suggestedFilename()).toBe('nli-quarantine-q-1.log');
  await page.getByRole('button', { name: 'Open in Resolution Wizard' }).click();
  await expect(page.locator('#page h1').filter({hasText:'Quarantine Resolution Wizard'})).toBeVisible();
  await expect(page.locator('#gprs-preview-form [name=raw_evidence]')).toHaveValue('exact raw source line');
  await page.getByTestId('nav-unknowns').click();
  await page.getByLabel('Record IDs').fill('q-1');
  await page.getByLabel('Target client ID').fill('node-1');
  await page.getByRole('button', { name: 'Request replay' }).click();
  await expect.poll(() => observed.includes('POST /api/v1/quarantine/reprocess')).toBe(true);
  await page.getByTestId('nav-api-monitor').click();
  await expect(page.getByRole('heading', { name: 'API monitor', exact: true })).toBeVisible();
  await expect(page.getByText('/system', { exact: true })).toBeVisible();
});

test('expected replay acknowledgement gates stay inline and never poison notifications or live sync', async ({ page }) => {
  const gate='target client has not acknowledged the current published parser bundle; replay remains gated';
  await mockController(page,{
    replayFailure:{status:409,message:gate},
    controllerNotifications:[{id:'audit-replay',sequence:91,title:'System issue: POST',resource:'/api/v1/quarantine/reprocess',severity:'critical',timestamp:'2026-11-02T00:00:00Z',read:false,status:'unread'}]
  });
  await login(page);
  await expect(page.locator('#notification-count')).toBeHidden();
  await page.getByTestId('nav-unknowns').click();
  await page.getByLabel('Record IDs').fill('q-1');
  await page.getByLabel('Target client ID').fill('node-1');
  await page.getByRole('button', { name: 'Request replay' }).click();
  await expect(page.locator('#quarantine-reprocess-form [data-replay-status]')).toContainText(/Waiting for the origin client/i);
  await expect(page.locator('#toast-region')).not.toContainText('System issue');
  await expect(page.getByText(/Live synchronization is unavailable/i)).toHaveCount(0);
  await expect(page.locator('#notification-count')).toBeHidden();
});

test('Assurance enables clear and download immediately and preserves results across navigation', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  const raw = '<img src=x onerror="window.injected=true"> service=test user=operator';
  await page.getByTestId('assurance-raw').fill(raw);
  await page.getByRole('button', { name: 'Run assurance', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Clear manually' })).toBeEnabled();
  await expect(page.getByRole('button', { name: 'Download result JSON' })).toBeEnabled();
  await expect(page.locator('#assurance-result')).toContainText('service=test user=operator');
  expect(await page.evaluate(() => window.injected)).toBeUndefined();
  await page.locator('#assurance-save-form [name=name]').fill('browser-retained-case');
  await page.getByRole('button', { name: 'Encrypt and retain in Supabase' }).click();
  await expect(page.getByText('browser-retained-case', { exact: true })).toBeVisible();
  await page.getByTestId('nav-fleet').click();
  await page.getByTestId('nav-assurance').click();
  await expect(page.getByTestId('assurance-raw')).toHaveValue(raw);
  await expect(page.locator('#assurance-result')).toContainText('service=test user=operator');
  const downloadEvent = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Download result JSON' }).click();
  expect((await downloadEvent).suggestedFilename()).toBe('nli-v21-assurance-result.json');
  await page.getByRole('button', { name: 'Clear manually' }).click();
  await expect(page.getByText('Run a case to inspect complete output', { exact: false })).toBeVisible();
  await expect(page.getByTestId('assurance-raw')).toHaveValue('');
});

test('Assurance parser and family catalogs drive single and corpus requests', async ({ page }) => {
  const { assuranceRequests, assuranceCorpora } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await expect(page.getByText('Live runtime parsers', { exact: true })).toBeVisible();
  const single = page.locator('#assurance-form');
  await single.locator('[name=service]').selectOption('smtp-in');
  await single.locator('[name=parser_id]').selectOption('smtp');
  await single.locator('[name=raw]').fill('level=INFO user=alice');
  await page.getByRole('button', { name: 'Run assurance', exact: true }).click();
  await expect.poll(() => assuranceRequests.length).toBe(1);
  expect(assuranceRequests[0]).toMatchObject({ service: 'smtp-in', parser_id: 'smtp' });

  await page.getByRole('tab', {name:'Corpus reports'}).click();
  const corpus = page.locator('#assurance-corpus-form');
  await corpus.locator('[name=service]').selectOption('imap');
  await corpus.locator('[name=parser_id]').selectOption('smtp');
  await corpus.locator('[name=cases]').fill('[{"raw":"level=INFO user=alice","desired":{}}]');
  await page.getByRole('button', { name: 'Run corpus', exact: true }).click();
  await expect.poll(() => assuranceCorpora.length).toBe(1);
  expect(assuranceCorpora[0].cases[0]).toMatchObject({ service: 'imap', parser_id: 'smtp' });
  await expect(page.locator('#assurance-corpus-result')).toContainText('Corpus result');
});

test('Assurance upload defaults to transient and shows protected retention metadata/download', async ({ page }) => {
  const { uploads } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  const retain = page.locator('#assurance-upload-form [name=retain]');
  await expect(retain).not.toBeChecked();
  await page.locator('#assurance-upload-form [name=service]').selectOption('smtp-in');
  await page.locator('#assurance-upload-form [name=parser_id]').selectOption('smtp');
  await retain.check();
  await page.locator('#assurance-upload-form [name=file]').setInputFiles({ name: 'sample.log', mimeType: 'text/plain', buffer: Buffer.from('service=example user=operator') });
  await page.getByRole('button', { name: 'Analyze uploaded sample' }).click();
  await expect(page.locator('#assurance-upload-status')).toContainText('encrypted and retained');
  await expect(page.locator('#assurance-upload-status')).toContainText('sample.log');
  await expect(page.locator('#assurance-upload-status')).toContainText('smtp@18.0.0');
  await expect(page.locator('#assurance-upload-status')).toContainText('encrypted_supabase');
  await expect(page.locator('#assurance-upload-status')).toContainText('controller encrypted the fixture');
  expect(uploads).toHaveLength(1);
  expect(uploads[0]).toContain('name="retain"\r\n\r\ntrue');
  expect(uploads[0]).toContain('name="parser_id"\r\n\r\nsmtp');
  const download = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Download retained sample through controller' }).click();
  expect((await download).suggestedFilename()).toBe('sample.log');
});

test('live recovery avoids dashboard fan-out and leaves Assurance inputs and results intact', async ({ page }) => {
  await page.clock.install();
  const { observed } = await mockController(page);
  await login(page);
  const canonicalSnapshots = observed.filter(item => item === 'GET /api/v1/sync/snapshot').length;
  await page.clock.fastForward(31_000);
  expect(observed.filter(item => item === 'GET /api/v1/dashboard/snapshot')).toHaveLength(0);
  await expect.poll(() => observed.filter(item => item === 'GET /api/v1/sync/snapshot').length).toBeGreaterThan(canonicalSnapshots);
  await page.getByTestId('nav-assurance').click();
  await page.getByTestId('assurance-raw').fill('message=keep-me');
  await page.getByRole('button', { name: 'Run assurance', exact: true }).click();
  await expect(page.locator('#assurance-result')).toContainText('keep-me');
  await page.clock.fastForward(61_000);
  await expect(page.getByTestId('assurance-raw')).toHaveValue('message=keep-me');
  await expect(page.locator('#assurance-result')).toContainText('keep-me');
});

test('viewer permissions apply to newly rendered Assurance actions', async ({ page }) => {
  await mockController(page, { role: 'viewer' });
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByTestId('assurance-raw').fill('message=viewer-test');
  await page.getByRole('button', { name: 'Run assurance', exact: true }).click();
  await expect(page.getByText('Complete actual output')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Encrypt and retain in Supabase' })).toBeHidden();
});

test('sign-out clears the last operator input and results', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await page.getByTestId('assurance-raw').fill('user=private-operator');
  await page.getByRole('button', { name: 'Run assurance', exact: true }).click();
  await expect(page.locator('#assurance-result')).toContainText('private-operator');
  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(page.getByLabel('Password')).toHaveValue('');
  await login(page);
  await page.getByTestId('nav-assurance').click();
  await expect(page.getByTestId('assurance-raw')).toHaveValue('');
});

test('incompatible controller keeps credentials disabled', async ({ page }) => {
  const { observed } = await mockController(page, { healthIdentity: 'v5' });
  await page.goto(origin);
  await expect(page.getByText('Controller connection required')).toBeVisible();
  await expect(page.getByLabel('Password')).toBeDisabled();
  expect(observed.filter(item => item === 'POST /api/v1/auth/login')).toEqual([]);
});

test('input paths workspace saves a revisioned family profile using the dedicated collection', async ({ page }) => {
  const { inputSaves } = await mockController(page);
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  await expect(page.getByRole('heading', { name: 'Input paths and family profiles' })).toBeVisible();
  await page.locator('#input-profile-form [name=quick_path]').fill('/var/log/nli/input/pop-extra.log');
  await page.locator('#input-profile-form [name=quick_family]').selectOption('pop');
  await page.getByRole('button', { name: 'Add path with selected family' }).click();
  await expect(page.locator('#input-profile-form [name=config]')).toHaveValue(/pop-extra\.log/);
  await page.locator('#input-profile-form [name=id]').fill('mail-family-paths');
  await page.locator('#input-profile-form [name=name]').fill('Mail family paths');
  await page.locator('#input-profile-form [name=config]').fill(JSON.stringify({ inputs: [{ path: '/var/log/imap.log', service: 'imap' }, { path: '/var/log/smtp.log', service: 'smtp-in' }] }));
  await page.getByRole('button', { name: 'Save profile revision' }).click();
  await expect.poll(() => inputSaves.length).toBe(1);
  expect(inputSaves[0]).toMatchObject({ id: 'mail-family-paths', config: { inputs: [{ path: '/var/log/imap.log', service: 'imap' }, { path: '/var/log/smtp.log', service: 'smtp-in' }] } });
});

test('input path editor adds, edits, removes and validates without replacing existing entries', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  const form=page.locator('#input-profile-form');
  await form.locator('[name=quick_path]').fill('/var/log/custom/one.log');
  await form.locator('[name=quick_family]').selectOption('smtp-in');
  await form.getByRole('button',{name:'Add path with selected family'}).click();
  const config=async()=>JSON.parse(await form.locator('[name=config]').inputValue());
  await expect.poll(async()=> (await config()).inputs.length).toBe(11);
  expect((await config()).inputs[0].path).toBe('/var/log/nli/input/auto.log');
  expect((await config()).inputs[0].multiline.rules).not.toContain('generic_continuation');
  expect((await config()).inputs.at(-1)).toMatchObject({path:'/var/log/custom/one.log',service:'smtp-in'});
  expect((await config()).inputs.at(-1).multiline.rules).toEqual(['generic_continuation']);
  await form.locator('[data-input-family="10"]').selectOption('imap');
  expect((await config()).inputs.at(-1).service).toBe('imap');
  await form.locator('[data-input-family="10"]').selectOption('auto');
  expect((await config()).inputs.at(-1).multiline.rules).not.toContain('generic_continuation');
  await form.getByRole('button',{name:'Edit /var/log/custom/one.log'}).click();
  await expect(form.locator('[data-input-path="10"]')).toBeFocused();
  await form.getByRole('button',{name:'Disable /var/log/custom/one.log'}).click();
  expect((await config()).inputs.at(-1).disabled).toBe(true);
  expect((await config()).inputs[0].disabled).toBeUndefined();
  await form.getByRole('button',{name:'Enable /var/log/custom/one.log'}).click();
  expect((await config()).inputs.at(-1).disabled).toBeUndefined();
  await form.getByRole('button',{name:'Disable all draft paths'}).click();
  expect((await config()).inputs.every(item=>item.disabled)).toBe(true);
  await form.getByRole('button',{name:'Enable all draft paths'}).click();
  expect((await config()).inputs.every(item=>!item.disabled)).toBe(true);
  await form.getByRole('button',{name:'Remove /var/log/custom/one.log'}).click();
  await expect.poll(async()=> (await config()).inputs.length).toBe(10);
  await form.getByRole('button',{name:'Validate paths'}).click();
  await expect(page.locator('#toast-region').getByText('10 input paths valid')).toBeVisible();
});

test('selecting input family leaves the open profile editor mounted for further selections', async ({ page }) => {
  await mockController(page);
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  const family=page.locator('#input-profile-form [name=quick_family]');
  await family.selectOption('smtp-in');
  await expect(family).toHaveValue('smtp-in');
  await expect(page.locator('#input-profile-form')).toBeVisible();
  await family.selectOption('pop');
  await expect(family).toHaveValue('pop');
});

test('performance timeline reads persisted metrics and reports bounded exact-count buckets', async ({ page }) => {
  const mock=await mockController(page);
  await login(page);
  await page.getByTestId('nav-performance').click();
  await expect(page.getByRole('heading',{name:'Overall Timeline History'})).toBeVisible();
  await expect(page.getByText(/persisted measurements/)).toBeVisible();
  expect(mock.observed).toContain('GET /api/v1/metrics/timeseries');
});

test('input path rollout uses the dedicated inputs-only contract and explicit node IDs', async ({ page }) => {
  const mock = await mockController(page);
  const profile = { id: 'mail-inputs', revision: 2, state: 'active', updated_at: '2026-10-14T00:00:00Z', payload: { name: 'Mail inputs', config: { inputs: [{ path: '/var/log/nli/input/imap.log', service: 'imap' }] } } };
  mock.inputProfiles.push(profile);
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  const form = page.locator('#input-rollout-form');
  await form.locator('[name=profile_id]').selectOption('mail-inputs');
  await form.locator('[name=targets]').fill('node-mail-1');
  await form.getByRole('button', { name: 'Request independent approval' }).click();
  await expect.poll(() => mock.inputRolloutSaves.length).toBe(1);
  expect(mock.inputRolloutSaves[0]).toEqual({ path: '/api/v1/config/input-paths/mail-inputs/rollout', body: { node_ids: ['node-mail-1'] } });
});

test('input path rollout supports select-all clients and multiple profile approvals', async ({ page }) => {
  const mock=await mockController(page);
  const base={revision:1,state:'active',payload:{config:{inputs:[{path:'/var/log/one.log',service:'imap'}]}}};
  mock.inputProfiles.push({...base,id:'profile-one',payload:{...base.payload,name:'Profile one'}});
  mock.inputProfiles.push({...base,id:'profile-two',payload:{...base.payload,name:'Profile two'}});
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  await page.getByLabel('Select all 1 clients').check();
  const form=page.locator('#input-rollout-form');
  await expect(form.locator('[name=targets]')).toHaveValue('node-1');
  await form.locator('[name=profile_id]').selectOption(['profile-one','profile-two']);
  await form.getByRole('button',{name:'Request independent approval'}).click();
  await expect.poll(()=>mock.inputRolloutSaves.length).toBe(2);
  expect(mock.inputRolloutSaves.map(item=>item.path)).toEqual(['/api/v1/config/input-paths/profile-one/rollout','/api/v1/config/input-paths/profile-two/rollout']);
  expect(mock.inputRolloutSaves.every(item=>item.body.node_ids[0]==='node-1')).toBe(true);
});

test('input path rollout supports every eligible client without enumerating node IDs', async ({ page }) => {
  const mock=await mockController(page);
  mock.inputProfiles.push({id:'global-inputs',revision:3,state:'active',payload:{name:'Global inputs',config:{inputs:[{path:'/var/log/nli/input/auto.log',service:'auto'}]}}});
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  const form=page.locator('#input-rollout-form');
  await form.locator('[name=profile_id]').selectOption('global-inputs');
  await form.locator('[name=all_eligible]').check();
  await expect(form.locator('[name=targets]')).toBeDisabled();
  await form.getByRole('button',{name:'Request independent approval'}).click();
  await expect.poll(()=>mock.inputRolloutSaves.length).toBe(1);
  expect(mock.inputRolloutSaves[0].body.node_ids).toEqual([]);
});

test('published input profile can be loaded and extended as a new draft revision', async ({ page }) => {
  const mock=await mockController(page);
  mock.inputProfiles.push({id:'published-inputs',revision:4,state:'active',payload:{name:'Published inputs',config:{inputs:[{path:'/var/log/existing.log',service:'auto'}]}}});
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  await page.getByRole('row').filter({hasText:'Published inputs'}).getByRole('button',{name:'Edit',exact:true}).click();
  const form=page.locator('#input-profile-form');
  await expect(form.locator('[name=id]')).toHaveValue('published-inputs');
  await form.locator('[name=quick_path]').fill('/var/log/added.log');
  await form.getByRole('button',{name:'Add path with selected family'}).click();
  expect(JSON.parse(await form.locator('[name=config]').inputValue()).inputs.map(item=>item.path)).toEqual(['/var/log/existing.log','/var/log/added.log']);
});

test('Command Center supports long presets, custom timestamps, exact count hover, JSON view and download', async ({ page }) => {
  const mock = await mockController(page);
  await login(page);
  const range=page.locator('#dashboard-range');
  await expect(range.locator('option')).toContainText(['1 hour','6 hours','12 hours','24 hours','2 days','1 week','2 weeks','4 weeks','30 days','90 days','3 months','6 months','9 months','12 months','1 year','2 years','5 years','10 years','All retained history','Custom dates']);
  await range.selectOption('all');
  await expect.poll(()=>mock.historyQueries.some(query=>query.get('range')==='all'&&query.get('step')==='auto')).toBe(true);
  await range.selectOption('custom');
  await page.locator('#dashboard-from').fill('2026-09-01T08:00');
  await page.locator('#dashboard-to').fill('2026-10-25T08:00');
  await page.getByRole('button',{name:'Apply dates'}).click();
  await expect.poll(()=>mock.historyQueries.some(query=>query.has('from')&&query.has('to')&&query.get('step')==='auto')).toBe(true);
  await expect(page.locator('[data-chart-category="Input"]').first()).toHaveAttribute('data-chart-value','9007199254740993');
  const point=page.locator('[data-chart-category="Input"]').first();
  await point.focus();
  await expect(page.locator('.chart-hover-readout').first()).toContainText('9,007,199,254,740,993');
  await page.getByText('View returned JSON').click();
  await expect(page.locator('.history-json pre')).toContainText('9007199254740993');
  const download=page.waitForEvent('download');
  await page.getByRole('button',{name:'Download JSON'}).click();
  expect((await download).suggestedFilename()).toBe('nli-command-center-custom.json');
});

test('Command Center refreshes retained counts in place while keeping the selected range mounted', async ({ page }) => {
  await page.clock.install();
  const mock=await mockController(page);
  await login(page);
  const range=page.locator('#dashboard-range');
  await expect(page.locator('[data-chart-category="Input"]').first()).toHaveAttribute('data-chart-value','9007199254740993');
  await page.clock.fastForward(5000);
  await expect(page.locator('[data-chart-category="Input"]').first()).toHaveAttribute('data-chart-value','42');
  await expect(range).toHaveValue('24h');
  expect(mock.historyQueries.filter(query=>query.get('range')==='24h')).toHaveLength(2);
});

test('pending approval nav badge follows controller state and clears after an action', async ({ page }) => {
  const mock=await mockController(page);
  mock.rolloutProposals.push({id:'approval-1',state:'pending_approval',created_by:'author',payload:{name:'Input rollout',type:'config_push'}});
  await login(page);
  await expect(page.locator('#approval-badge')).toHaveText('1');
  await page.getByTestId('nav-rollouts').click();
  await page.locator('[data-rollout-reason="approval-1"]').fill('Independent configuration review complete');
  await page.getByRole('button',{name:'Approve reviewed proposal'}).click();
  await expect(page.locator('#approval-badge')).toBeHidden();
  await expect(page.locator('#page').getByText('approved', { exact: false }).first()).toBeVisible();
});

test('canonical navigation counts expose current parser, quarantine and configuration work without notification history', async ({ page }) => {
  await mockController(page, { navigationCounts: { parsers:{total:3,states:{draft:2,in_review:1}},'parser-workspace':{total:3,states:{draft:2,in_review:1}},'input-paths':{total:2,states:{draft:1,approved:1}},notifications:{total:88} } });
  await login(page);
  await expect(page.locator('#nav-count-parsers')).toHaveText('3');
  await expect(page.locator('#nav-count-parser-workspace')).toHaveText('3');
  await expect(page.locator('#nav-count-input-paths')).toHaveText('2');
  await expect(page.locator('#nav-count-quarantine-wizard')).toHaveText('1');
  await expect(page.locator('[data-nav-count="notifications"]')).toHaveCount(0);
  await expect(page.locator('#nav-count-kv')).toBeHidden();
});

test('Parser Catalog isolates a blocked legacy parser and offers an explicit repair draft transition', async ({ page }) => {
  const mock=await mockController(page,{managedParsers:[{id:'legacy-broken',revision:9,state:'published',created_by:'legacy-import',payload:{name:'Legacy broken',version:'0.9.0',priority:10,rules:[]},runtime_sync:{status:'blocked',reason:'legacy parser rules are incompatible with the current runtime'}}]});
  await login(page);
  await page.getByTestId('nav-parsers').click();
  await expect(page.locator('#page')).toContainText('legacy parser rules are incompatible');
  await expect(page.getByRole('button',{name:'Repair as draft'})).toBeVisible();
  await page.getByRole('button',{name:'Repair as draft'}).click();
  await expect.poll(()=>mock.observed).toContain('POST /api/v1/parser-packs/legacy-broken/repair');
  await expect(page.locator('#page')).toContainText('not_published');
});

test('Parser Catalog shows a runtime-synchronized publication as Published / Active', async ({ page }) => {
  await mockController(page,{managedParsers:[{id:'smtp-managed',revision:3,state:'published',created_by:'operator',payload:{name:'SMTP managed',version:'1.2.0',priority:20,rules:[]},runtime_sync:{status:'active',desired_revision:3,applied_revision:3,desired_hash:'a'.repeat(64),applied_hash:'a'.repeat(64),drifted:false}}]});
  await login(page);
  await page.getByTestId('nav-parsers').click();
  const row=page.locator('#page tr').filter({hasText:'SMTP managed'});
  await expect(row.locator('td').nth(4)).toContainText('Published / Active');
  await expect(row.locator('td').nth(5)).toContainText('active');
});

test('draft input profiles expose lifecycle actions and JSON without a dead rollout button', async ({ page }) => {
  const mock=await mockController(page);
  mock.inputProfiles.push({id:'draft-inputs',revision:1,state:'draft',payload:{name:'Draft inputs',config:{inputs:[{path:'/var/log/draft.log',service:'imap'}]}}});
  await login(page);
  await page.getByTestId('nav-input-paths').click();
  const rollout=page.locator('#input-rollout-form');
  await expect(rollout.getByRole('button',{name:'Request independent approval'})).toBeEnabled();
  await expect(page.getByRole('button',{name:'Submit for review'})).toBeVisible();
  await page.getByRole('button',{name:'View / download JSON'}).click();
  await expect(page.locator('#input-profile-detail')).toContainText('/var/log/draft.log');
  const download=page.waitForEvent('download');
  await page.locator('#input-profile-detail').getByRole('button',{name:'Download JSON',exact:true}).click();
  expect((await download).suggestedFilename()).toBe('draft-inputs-input-profile.json');
  await page.getByRole('button',{name:'Submit for review'}).click();
  await expect(page.getByRole('button',{name:'Approve',exact:true})).toBeVisible();
  await page.getByRole('button',{name:'Approve',exact:true}).click();
  await expect(page.getByRole('button',{name:'Publish',exact:true})).toBeVisible();
  await page.getByRole('button',{name:'Publish',exact:true}).click();
  await expect(rollout.locator('[name=profile_id]')).toContainText('Draft inputs · r1');
});

test('mobile navigation exposes all workspaces without horizontal page overflow', async ({ page }) => {
  test.setTimeout(60_000);
  await page.setViewportSize({ width: 390, height: 844 });
  await mockController(page);
  await login(page);
  for (const tab of tabs) {
    await page.getByLabel('Toggle navigation').click();
    await page.getByTestId(`nav-${tab}`).click();
    await expect(page.locator('.loading-state')).toHaveCount(0);
    await expect(page.locator('.error-state')).toHaveCount(0);
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

test('AI profile removal confirms exact ID, disables cleared credentials and preserves sibling profiles',async({page})=>{
  await mockController(page);
  const profiles=[{id:'removable',name:'Removable model',provider:'openai',model:'test-model',endpoint:'https://api.openai.com/v1',configured:true,enabled:true,credential_source:'env_ref',credential_ref:'TEST_AI_KEY',priority:0},{id:'sibling',name:'Sibling model',provider:'openai',model:'other-model',endpoint:'https://api.openai.com/v1',configured:true,enabled:true,credential_source:'env_ref',credential_ref:'TEST_AI_KEY',priority:1}];
  const mutations=[];
  await page.route('**/api/v1/admin/ai-provider?**',async route=>{
    const request=route.request(),url=new URL(request.url()),id=url.searchParams.get('id'),index=profiles.findIndex(p=>p.id===id);
    if(request.method()==='POST'){mutations.push(['clear',id]);Object.assign(profiles[index],{configured:false,enabled:false,credential_removed:true,credential_ref:''});return route.fulfill({json:{id,profile_retained:true}});}
    if(request.method()==='DELETE'){mutations.push(['delete',id]);profiles.splice(index,1);return route.fulfill({json:{id,profile_retained:false}});}
    return route.fulfill({json:{provider:profiles[index]||profiles[0],profiles,credential_modes:['encrypted_inline','env_ref','file_ref','none'],capabilities:{available:profiles.some(p=>p.enabled)}}});
  });
  await page.route('**/api/v1/admin/ai-provider',route=>route.fulfill({json:{provider:profiles[0],profiles,credential_modes:['encrypted_inline','env_ref','file_ref','none']}}));
  await login(page);await page.getByTestId('nav-ai').click();
  await expect(page.locator('#ai-provider-form [name=id]')).toHaveValue('removable');
  page.once('dialog',dialog=>{expect(dialog.message()).toContain('removable');return dialog.dismiss();});
  await page.getByRole('button',{name:'Remove credential',exact:true}).click();expect(mutations).toEqual([]);
  page.once('dialog',dialog=>dialog.accept());await page.getByRole('button',{name:'Remove credential',exact:true}).click();
  await expect(page.locator('#ai-provider-form [name=enabled]')).not.toBeChecked();
  await expect(page.locator('#ai-provider-form [name=api_key]')).toHaveValue('');
  await expect(page.locator('#ai-profile-choice')).toContainText('Sibling model');
  page.once('dialog',dialog=>{expect(dialog.message()).toContain('removable');return dialog.accept();});await page.getByRole('button',{name:'Delete profile',exact:true}).click();
  await expect(page.locator('#ai-provider-form [name=id]')).toHaveValue('sibling');
  expect(mutations).toEqual([['clear','removable'],['delete','removable']]);
});

test('Suggest with AI preserves manual mappings, checks readiness and imports only a reviewed current proposal',async({page})=>{
 await mockController(page);let ready=false;const requests=[];
 await page.route('**/api/v1/ai/capabilities',route=>route.fulfill({json:{available:ready,status:ready?'configured':'not_configured'}}));
 const plan={version:1,fields:[{name:'ai_status',method:'key_value',source:'status',type:'string',policy:'required',operations:[]}]};
 await page.route('**/api/v1/field-mapping/ai-suggest',route=>{requests.push(route.request().postDataJSON());return route.fulfill({json:{proposal:{plan},validation:{syntax_valid:true,extraction_valid:true,desired_match:false,publication_validated:false},raw_sha256:'synthetic-exact-source-hash',review_only:true,persisted:false,activated:false}});});
 await login(page);await page.getByTestId('nav-field-mapping').click();await page.locator('#nfi-raw').fill('status=FAILED');await page.getByRole('button',{name:'Add Custom Field',exact:true}).click();await page.locator('[data-qrw-key=name]').fill('manual_preserved');
 const host=page.locator('[data-ai-mapping-host]');await host.getByRole('button',{name:'Suggest with AI',exact:true}).click();await expect(page.locator('#toast-region')).toContainText('unconfigured');expect(requests).toHaveLength(0);
 ready=true;await host.getByRole('button',{name:'Suggest with AI',exact:true}).click();await expect(host.locator('[data-ai-mapping-proposal]')).toBeVisible();await expect(page.locator('[data-qrw-key=name]')).toHaveValue('manual_preserved');expect(requests[0].plan.fields[0].name).toBe('manual_preserved');await expect(host.locator('[data-ai-mapping-json]')).toContainText('"desired_match": false');
 page.once('dialog',dialog=>dialog.dismiss());await host.getByRole('button',{name:'Import proposed mapping',exact:true}).click();await expect(page.locator('[data-qrw-key=name]')).toHaveValue('manual_preserved');
 await page.locator('#nfi-raw').fill('status=NEW');await expect(host.locator('[data-ai-mapping-proposal]')).toBeHidden();await page.evaluate(()=>document.querySelector('[data-action=mapping-ai-import]').click());await expect(page.locator('#toast-region')).toContainText('Raw evidence changed');await expect(page.locator('[data-qrw-key=name]')).toHaveValue('manual_preserved');
 await host.getByRole('button',{name:'Suggest with AI',exact:true}).click();await expect(host.locator('[data-ai-mapping-proposal]')).toBeVisible();page.once('dialog',dialog=>dialog.accept());await host.getByRole('button',{name:'Import proposed mapping',exact:true}).click();await expect(page.locator('[data-qrw-key=name]')).toHaveValue('ai_status');
 expect(requests).toHaveLength(2);expect(requests[1].raw).toBe('status=NEW');
});

test('AI mapping controls are shared by evidence authoring editors and discard late stale responses',async({page})=>{
 await mockController(page);await page.route('**/api/v1/ai/capabilities',route=>route.fulfill({json:{available:true}}));let pending;await page.route('**/api/v1/field-mapping/ai-suggest',route=>{pending=route;});await login(page);await page.getByTestId('nav-parser-workspace').click();const form=page.locator('#parser-workspace-form');await expect(form.getByRole('button',{name:'Suggest with AI',exact:true})).toBeVisible();await form.locator('[name=samples]').fill('first sample');await form.getByRole('button',{name:'Suggest with AI',exact:true}).click();await expect.poll(()=>!!pending).toBe(true);await form.locator('[name=samples]').fill('second sample');await pending.fulfill({json:{proposal:{plan:{version:1,fields:[{name:'forbidden_stale',method:'raw',type:'string'}]}},validation:{syntax_valid:true,extraction_valid:true}}});await expect(form.locator('[data-ai-mapping-status]')).toContainText('stale');await expect(form.locator('[data-ai-mapping-proposal]')).toBeHidden();
 pending=null;await form.getByRole('button',{name:'Suggest with AI',exact:true}).click();await expect.poll(()=>!!pending).toBe(true);await pending.fulfill({json:{proposal:{plan:{version:1,fields:[{name:'reviewed_message',method:'raw',source:'',type:'string',policy:'required'}]},parser:{service_scope:'custom-service'}},validation:{syntax_valid:true,extraction_valid:true}}});await expect(form.locator('[data-ai-mapping-proposal]')).toBeVisible();await expect(form.locator('[name=samples]')).toHaveValue('second sample');page.once('dialog',dialog=>dialog.accept());await form.getByRole('button',{name:'Open proposed mapping in FieldSense',exact:true}).click();await expect(page.locator('#nfi-raw')).toHaveValue('second sample');await expect(page.locator('[data-qrw-key=name]')).toHaveValue('reviewed_message');await expect(page.locator('#nfi-service')).toHaveValue('custom-service');
 for(const route of ['studio','regex']){await page.getByTestId(`nav-${route}`).click();await expect(page.locator('[data-ai-mapping-raw]')).toBeVisible();await expect(page.getByRole('button',{name:'Suggest with AI',exact:true})).toBeVisible();}
 await page.getByTestId('nav-assurance').click();await page.getByRole('tab',{name:'Parser Creation Helper',exact:true}).click();await expect(page.locator('#parser-helper-form').getByRole('button',{name:'Suggest with AI',exact:true})).toBeVisible();
});

test('AI remediation proposals can be rejected or removed without publishing a parser',async({page})=>{
 await mockController(page);let proposals=[{id:'review-draft',state:'draft',payload:{name:'Review draft'}},{id:'review-approved',state:'approved',payload:{name:'Approved proposal'}}];const writes=[];
 await page.route('**/api/v1/assurance/ai-proposals',route=>route.fulfill({json:{proposals}}));
 await page.route('**/api/v1/assurance/ai-proposals/**',route=>{const request=route.request(),path=new URL(request.url()).pathname;writes.push([request.method(),path,request.postDataJSON()]);if(path.endsWith('/reject'))proposals=proposals.map(item=>({...item,state:item.id==='review-draft'?'rejected':item.state}));else proposals=proposals.filter(item=>item.id!=='review-approved');return route.fulfill({json:{state:'rejected',removed:request.method()==='DELETE'}});});
 await login(page);await page.getByTestId('nav-ai').click();await page.getByRole('tab',{name:'Results',exact:true}).click();
 page.on('dialog',dialog=>dialog.type()==='prompt'?dialog.accept('Operator chose native mapping'):dialog.accept());
 await page.locator('[data-action=ai-proposal-reject]').click();await page.getByRole('tab',{name:'Results',exact:true}).click();await expect(page.locator('#ai-panel-results')).toContainText('rejected');
 await page.locator('[data-action=ai-proposal-remove][data-id=review-approved]').click();await page.getByRole('tab',{name:'Results',exact:true}).click();await expect(page.locator('#ai-panel-results')).not.toContainText('Approved proposal');
 expect(writes).toEqual([['POST','/api/v1/assurance/ai-proposals/review-draft/reject',{reason:'Operator chose native mapping'}],['DELETE','/api/v1/assurance/ai-proposals/review-approved',{}]]);
});

test('An acknowledged but exhausted quarantine record has an honest Step 9 outcome', async ({ page }) => {
  const hash = 'a'.repeat(64);
  await mockController(page, {
    qrwRecords: [{ id: 'q-1', node_id: 'node-1', status: 'manual_attention', raw_available: true, attempt_count: 5 }],
    qrwParsers: [{ id: 'origin-parser', revision: 2, state: 'published', payload: { source_quarantine_record_id: 'q-1', artifact_checksum: hash, validation_passed: true, activated: true } }],
    qrwSyncAssets: [{ asset_class: 'parser', asset_id: 'origin-parser', desired: { sha256: hash }, clients: [{ node_id: 'node-1', desired_hash: hash, applied_hash: hash, drifted: false, status: 'online' }] }]
  });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button', { name: 'Inspect', exact: true }).click();
  await page.locator('[data-wizard-step="8"] button').click();
  await expect(page.getByRole('button', { name: /Request targeted replay/ })).toBeDisabled();
  await expect(page.locator('#gprs-replay-form')).toContainText(/retry limit|attempts exhausted/i);
  await page.getByRole('button', { name: '9 · Verify replay result', exact: true }).click();
  await expect(page.locator('#qrw-verification')).toContainText('manual_attention');
  await expect(page.locator('[data-wizard-step="9"]')).toHaveAttribute('data-step-complete', 'false');
});

test('Five parser strategies do not consume a retained record replay budget', async ({ page }) => {
  const hash = 'a'.repeat(64);
  const mock = await mockController(page, {
    qrwRecords: [{ id: 'q-1', node_id: 'node-1', status: 'waiting', raw_available: true, attempt_count: 5, replay_attempt_count: 0 }],
    qrwParsers: [{ id: 'origin-parser', revision: 2, state: 'published', payload: { source_quarantine_record_id: 'q-1', artifact_checksum: hash, validation_passed: true, activated: true } }],
    qrwSyncAssets: [{ asset_class: 'parser', asset_id: 'origin-parser', desired: { sha256: hash }, clients: [{ node_id: 'node-1', desired_hash: hash, applied_hash: hash, drifted: false, status: 'online' }] }]
  });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button', { name: 'Inspect', exact: true }).click();
  await page.locator('[data-wizard-step="8"] button').click();
  await page.getByRole('button', { name: /Request targeted replay/ }).click();
  await expect.poll(() => mock.qrwReplays).toEqual([{ record_ids: ['q-1'], node_id: 'node-1', expected_bundle_hash: hash }]);
  await expect(page.locator('#qrw-verification')).toBeVisible();
  await expect(page.locator('[data-wizard-step="9"]')).toHaveAttribute('data-step-complete', 'true');
});

test('Guided Parser Repair Studio onboards a pasted sample without quarantine replay', async ({ page }) => {
  const raw='login accepted alice';
  const desired={event:'login',result:'accepted',username:'alice'};
  const hash='a'.repeat(64);
  const mock=await mockController(page,{
    qrwRecords:[],
    qrwSyncAssets:[{asset_class:'parser_pack',asset_id:'draft-1',desired:{sha256:hash},clients:[{node_id:'node-1',desired_hash:hash,applied_hash:hash,drifted:false}]}]
  });
  let draft;
  await page.route('**/api/v1/assurance/parsers',route=>{
    if(route.request().method()==='POST')draft=route.request().postDataJSON();
    return route.fallback();
  });
  await page.route('**/api/v1/assurance/mapping-preview',route=>route.fulfill({json:{passed:true,actual:desired,errors:[]}}));
  let verification,exactAvailable=false;
  await page.route('**/api/v1/assurance/run',route=>{
    verification=route.request().postDataJSON();
    return route.fulfill({json:{passed:true,actual:desired,diff:[],parser:{name:exactAvailable&&verification.parser_id?verification.parser_id:'another-parser'}}});
  });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByLabel('Manual raw sample').fill(raw);
  await page.getByLabel('Service scope',{exact:true}).first().fill('login');
  await page.getByRole('button',{name:'1 · Use manual sample'}).click();
  await expect(page.locator('#gprs-preview-form [name=raw_evidence]')).toHaveValue(raw);
  await page.getByTestId('nav-assurance').click();
  await page.getByRole('tab',{name:'Guided Parser Repair Studio'}).click();
  await expect(page.locator('#gprs-preview-form [name=raw_evidence]')).toHaveValue(raw);
  await page.getByLabel('Desired JSON field names').fill('event, result, username');
  await page.getByRole('button',{name:/Create mappings from my field names/}).click();
  await page.getByRole('button',{name:'Generate + validate complete parser pack',exact:true}).click();
  const form=page.locator('#gprs-parser-helper-form');
  await expect(form.locator('[data-parser-helper-status]')).toContainText('Complete parser validation passed');
  await form.getByRole('button',{name:'Save review-gated draft',exact:true}).click();
  await expect.poll(()=>draft?.cases?.[0]).toMatchObject({raw,desired,source:'manual',service:'login'});
  expect(draft.source_quarantine_record_id).toBe('');
  const studio=page.locator('#assurance-panel-gprs');
  await studio.getByRole('button',{name:/7 · Review & publish/}).click();
  const lifecycle=studio.locator('[data-qrw-panel="7"]');
  await lifecycle.getByRole('button',{name:'Submit',exact:true}).click();
  await lifecycle.getByRole('button',{name:'Approve',exact:true}).click();
  await lifecycle.getByRole('button',{name:'Publish',exact:true}).click();
  await studio.getByRole('button',{name:/8 · Deploy/}).first().click();
  await studio.getByLabel('Verification client').selectOption('node-1');
  await studio.getByRole('button',{name:'9 · Verify manual sample'}).click();
  await studio.getByRole('button',{name:'9.1 · Run production verification'}).click();
  await expect(studio.getByText(/selected a different parser/)).toBeVisible();
  exactAvailable=true;
  await studio.getByRole('button',{name:/9.1 .*Run production verification/}).click();
  await expect(studio.getByText(/Complete · Published parser and automatic output matched/)).toBeVisible();
  expect(verification).toMatchObject({raw,desired,service:'login',preserve_source:true,parser_id:'draft-1'});
  expect(mock.qrwReplays).toEqual([]);
  expect(mock.observed.some(item=>item.endsWith('/analyze')&&item.includes('/quarantine/records/'))).toBe(false);
});

test('Ver22 replay requires the controller acknowledgement even when the hashes match', async ({ page }) => {
  const hash = 'a'.repeat(64);
  await mockController(page, {
    qrwRecords: [{ id: 'q-1', node_id: 'node-1', status: 'waiting', raw_available: true, replay_attempt_count: 0 }],
    qrwParsers: [{ id: 'origin-parser', revision: 2, state: 'published', payload: { source_quarantine_record_id: 'q-1', artifact_checksum: hash, validation_passed: true, activated: true } }],
    qrwSyncAssets: [{ asset_class: 'parser', asset_id: 'origin-parser', desired: { sha256: hash }, clients: [{ node_id: 'node-1', desired_hash: hash, applied_hash: hash, drifted: false, status: 'pending', exact_acknowledgement: false }] }]
  });
  await login(page);
  await page.getByTestId('nav-quarantine-wizard').click();
  await page.getByRole('button', { name: 'Inspect', exact: true }).click();
  await page.locator('[data-wizard-step="8"] button').click();
  await expect(page.getByRole('button', { name: /Request targeted replay/ })).toBeDisabled();
  await page.getByRole('button', { name: '9 · Verify replay result', exact: true }).click();
  await expect(page.locator('#qrw-verification')).toContainText('waiting');
  await expect(page.locator('[data-wizard-step="9"]')).toHaveAttribute('data-step-complete', 'false');
});
