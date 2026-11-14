(() => {
  'use strict';

  const NAV = [
    ['COMMAND CENTER', 'operations', 'Command Center', '◫'],
    ['COMMAND CENTER', 'monitoring', 'Monitoring', '◈'],
    ['COMMAND CENTER', 'streams', 'Live Streams', '⇶'],
    ['ASSURANCE & RELEASE', 'assurance', 'Assurance Lab', '◇'],
    ['ASSURANCE & RELEASE', 'quarantine-wizard', 'Quarantine Resolution Wizard', '⌁'],
    ['ASSURANCE & RELEASE', 'upes', 'UPES · Parser Recovery', '⟳'],
    ['ASSURANCE & RELEASE', 'rollouts', 'Approvals & Rollouts', '↗'],
    ['ASSURANCE & RELEASE', 'parser-sync', 'Parser Fleet Sync', '⇄'],
    ['ASSURANCE & RELEASE', 'artifacts', 'Signed Artifacts & Upgrades', '⬡'],
    ['ASSURANCE & RELEASE', 'parser-workspace', 'Parser Creation Workspace', '⚒'],
    ['ASSURANCE & RELEASE', 'nupire', 'NUPIRE / EGPE', '✧'],
    ['ASSURANCE & RELEASE', 'field-integrity', 'Field Integrity & Corpus Proof', '✓'],
    ['ASSURANCE & RELEASE', 'ai', 'AI Resolution & Providers', '✧'],
    ['ASSURANCE & RELEASE', 'sample-history', 'Test Sample History', '◷'],
    ['PARSER MANAGEMENT', 'header-templates', 'Log Pattern Templates', '⌘'],
    ['PARSER MANAGEMENT', 'kv-templates', 'KV Pair Templates', '⌘'],
    ['PARSER MANAGEMENT', 'parsers', 'Parser Catalog & Priority', '⌘'],
    ['PARSER MANAGEMENT', 'reference-ai', 'Reference based help & Fix', '✧'],
    ['PARSER MANAGEMENT', 'building-blocks', 'Building Blocks', '▦'],
    ['PARSER MANAGEMENT', 'field-mapping', 'NLI FieldSense Studio', '⌗'],
    ['PARSER MANAGEMENT', 'studio', 'Regex, Templates & Studio', '.*'],
    ['PARSER MANAGEMENT', 'regex', 'Regex Pattern Library', '∿'],
    ['PARSER MANAGEMENT', 'knownkeys', 'Known Key Catalog', '⌗'],
    ['PARSER MANAGEMENT', 'drift', 'Template Drift', '≠'],
    ['OPERATIONS', 'notifications', 'Notification Manager', '♢'],
    ['REAL-TIME SUBSYSTEMS', 'sync-fabric', 'Unified Sync Fabric', '⟳'],
    ['REAL-TIME SUBSYSTEMS', 'nhle-monitor', 'NHLE Live', 'ϟ'],
    ['REAL-TIME SUBSYSTEMS', 'napi-monitor', 'NAPI Live', '✧'],
    ['REAL-TIME SUBSYSTEMS', 'nal-monitor', 'NAL Live', '◇'],
    ['INPUT', 'input-paths', 'Input Log Paths', '⇥'],
    ['INPUT', 'migration', 'Migration Center', '⇄'],
    ['OUTPUT', 'outputs', 'Endpoint Configuration', '⇢'],
    ['OUTPUT', 'metadata', 'Live Metadata', '◉'],
    ['OUTPUT', 'unknowns', 'Unknown & Quarantine', '△'],
    ['OUTPUT', 'performance', 'Performance & Backpressure', '⌁'],
    ['PARSING ENGINE', 'kv', 'KV Delimiter Config', '≋'],
    ['PARSING ENGINE', 'static', 'Static Positional Patterns', '▤'],
    ['PARSING ENGINE', 'projection', 'Field Capture & Projection', '⌗'],
    ['PARSING ENGINE', 'semantic', 'Semantic Extraction Policy', '◈'],
    ['RUNTIME ENGINES', 'engine-network', 'Network Identity Engine', '⇧'],
    ['RUNTIME ENGINES', 'engine-url', 'URL Engine', '↗'],
    ['RUNTIME ENGINES', 'engine-email', 'Email Address Engine', '✉'],
    ['RUNTIME ENGINES', 'engine-subject', 'Encoded Subject Engine', 'Aa'],
    ['CLIENT MANAGEMENT', 'fleet', 'Fleet Operations', '◎'],
    ['CLIENT MANAGEMENT', 'clients', 'Client Agent Management', '⌁'],
    ['CONTROLLER MANAGEMENT', 'controller', 'Controller Agent Management', '▣'],
    ['CONTROLLER MANAGEMENT', 'controller-gateway', 'Controller Gateway', '◇'],
    ['CONTROLLER MANAGEMENT', 'ai-privacy', 'AI Privacy and Knowledge', '◇'],
    ['CONTROLLER MANAGEMENT', 'api-monitor', 'API Monitor', '⌁'],
    ['CONTROLLER MANAGEMENT', 'latency', 'Latency', '◷'],
    ['CONTROLLER MANAGEMENT', 'system', 'System Health & Storage', '✦'],
    ['CONTROLLER MANAGEMENT', 'communications', 'Status of All Communication', '⇄'],
    ['CONTROLLER MANAGEMENT', 'security', 'Users, RBAC & Audit', '◆'],
  ];
  const TITLES = Object.fromEntries(NAV.map(([, id, title]) => [id, title]));
  const NAV_COUNT_PAGES = new Set(['assurance','quarantine-wizard','upes','rollouts','parser-workspace','parsers','field-mapping','input-paths','outputs','kv','static','projection','studio','regex','knownkeys','semantic','unknowns']);
  const CONFIG = {
    'input-paths': ['/config/input-paths', 'Input path profiles', { inputs: [] }],
    kv: ['/config/kv', 'KV delimiter policy', { pair_separators: [':', ';', ','], assignment_separators: ['=', ':'], quotes: ['single', 'double'], duplicate_policy: 'last', max_fields: 256 }],
    'header-templates': ['/config/templates', 'Log pattern templates', {
      engine:'re2',
      pattern:'^(?P<date>[0-9]{8})[ \\t]+(?P<timestamp>[^ \\t]+)[ \\t]+(?P<hostname>[^ \\t]+)[ \\t]+(?P<service>[^ \\t]+)[ \\t]+(?P<pid>[0-9]+)[ \\t]+(?P<minor_pid>[0-9]+)[ \\t]+(?P<thread_id>[0-9]+)[ \\t]+(?:(?P<random_number>[0-9]+);(?:(?P<extra_tag>[^; \\t]+);)?)?(?P<event_type>[^; \\t]+);(?P<event>[^;() \\t]+)(?:\\((?P<event_code>[^()\\s]+)\\))?(?:[ \\t]+|$)',
      fields:{date:'date',timestamp:'timestamp',hostname:'hostname',service:'service',pid:'pid',minor_pid:'minor_pid',thread_id:'thread_id',random_number:'random_number',extra_tag:'extra_tag',event_type:'event_type',event:'event',event_code:'event_code'},
      optional_fields:['random_number','extra_tag','event_code'],resource_limits:{max_input_bytes:1048576,timeout_ms:50},
    }],
    'kv-templates': ['/config/kv', 'KV pair templates', {
      pair_separators:[':', ';', '|'],assignment_separators:['='],literal_boundaries:true,
      bare_field:'messages',duplicate_policy:'error',max_fields:512,
    }],
    static: ['/config/static-patterns', 'Static positional pattern', { tokens: [], rest_field: 'message', failure_policy: 'reject_rule' }],
    projection: ['/config/projections', 'Field capture and projection', { mode: 'allowlist', fields: [], types: {}, count_policy: 'projection' }],
    studio: ['/config/templates', 'Regex and template definition', { engine: 're2', pattern: '', fields: {}, resource_limits: { max_input_bytes: 1048576, timeout_ms: 50 } }],
    regex: ['/config/regex', 'Named regular expression pattern', { pattern: '^(?P<timestamp>\\S+)\\s+(?P<level>INFO|WARN|ERROR)\\s+(?P<message>.*)$', fields: { timestamp: 'timestamp', level: 'severity', message: 'message' }, engine: 're2' }],
    knownkeys: ['/config/known-keys', 'Known key catalog', { keys: ['timestamp', 'time', 'level', 'severity', 'message', 'host', 'hostname', 'service', 'pid', 'process', 'user', 'username', 'source_ip', 'destination_ip', 'status', 'duration', 'request_id'], aliases: {}, delimiters: ['=', ':'] }],
    semantic: ['/config/semantic-extraction', 'Semantic extraction policy', { entities: ['ipv4', 'ipv6', 'domain', 'url', 'email'], decode_mime_headers: true, max_input_bytes: 1048576, max_header_bytes: 65536 }],
    outputs: ['/config/output-destinations', 'Output destination profiles', { outputs: [{ type: 'file', path_template: '/var/log/nli/output/auto_normalised_{hour_utc}.jsonl', routes: ['normal', 'warning'] }, { type: 'file', path_template: '/var/log/nli/output/auto_quarantine_{hour_utc}.jsonl', routes: ['quarantine', 'unknown'] }] }],
  };
  const state = { user: null, csrf: '', page: 'operations', loadedAt: 0, socket: null, streamSource: null, streamMode: 'connecting', reconnect: 1000, streamFailures: 0, live: [], streamLastEventId: '', lastAssurance: null, lastNAPI: null, quarantineDiagnosis: null, repairRecords: [], quarantineHistory: [], qrwReplayAcceptedIds: [], assuranceParsers: [], assuranceFixtures: [], assuranceSyncBundle: null, assuranceSyncAssets: [], assuranceSyncClients: [], assuranceSyncTelemetry: null, assuranceFleetNodes: [], parserRolloutProposals: [], parserLiveRollouts: [], parserRolloutLoadError: '', rolloutActionErrors: {}, gprsManual: null, gprsPreview: null, gprsSamples: [], qrwFields: [], qrwStep: 1, upesSnapshot: null, controllerReady: false, controllerCheckInFlight: false, proxy: false, polling: false, loadingPage: false, notifications: [], controllerNotifications: [], notificationRefreshInFlight: false, lastRefreshFailure: '', dashboard: null, dashboardRange: '24h', dashboardClient: 'all', dashboardFrom: '', dashboardTo: '', dashboardHistory: null, dashboardHistoryError: '', dashboardHistoryRefreshInFlight: false, pendingApprovals: 0, navCounts: {}, navCountsInFlight: false, monitoringView: 'overview', monitoringRange: '15m', monitoringNode: 'all', monitoringSource: 'all', monitoringSearch: '', monitoringPaused: false, inputHistoryProfile: '', builtinParsers: [], migrationReports: [], assurancePlans: [], aiProposals: [], uploadReport: null, uploadError: '', apiHistory: [], outputPreview: null, streamPollTimer: null, streamReconnectTimer: null, browserHeartbeatInFlight: false, lastBrowserHeartbeatAt: 0, sync: { snapshot: null, history: [], browserAppliedAt: 0, browserDelayMs: null, lastReceivedAt: 0, pollInFlight: false, pollController: null, pollFailures: 0, rejectedUpdates: 0, connectionStartedAt: 0, resyncing: false, renderedSignature: null, renderScheduled: false, pendingRender: false } };
  const el = id => document.getElementById(id);
  const parserSyncClients = asset => {
    const fleet = Array.isArray(state.assuranceSyncClients) ? state.assuranceSyncClients : [];
    if (fleet.length) return fleet;
    return Array.isArray(asset?.clients) ? asset.clients : [];
  };
  let selectedNupire = null;
  const FORM_PERMISSIONS = {
    'assurance-save-form': 'assurance:save',
    'fleet-desired-form': 'fleet:manage',
    'parser-form': 'parsers:manage',
    'migration-form': 'migration:manage',
    'config-form': 'parsers:manage',
    'input-profile-form': 'parsers:manage',
    'input-rollout-form': 'rollout:trigger',
    'output-profile-form': 'parsers:manage',
    'output-rollout-form': 'rollout:trigger',
    'rollout-form': 'rollout:trigger',
    'client-action-form': 'rollout:trigger',
    'managed-input-file-form': 'rollout:trigger',
    'artifact-form': 'artifacts:upload',
    'upgrade-form': 'rollout:trigger',
    'user-form': 'users:manage',
    'notification-config-form': 'users:manage',
    'ai-provider-form': 'system:health',
    'assurance-parser-form': 'assurance:save',
    'parser-workspace-form': 'assurance:save',
    'parser-helper-form': 'assurance:save',
    'gprs-parser-helper-form': 'assurance:save',
    'assurance-plan-form': 'parsers:manage',
    'assurance-run-form': 'assurance:save',
    'napi-synthesis-form': 'parsers:manage',
    'nupire-discovery-form': 'parsers:manage',
    'nupire-artifact-form': 'parsers:manage',
    'nupire-evaluation-form': 'parsers:manage',
    'assurance-upload-form': 'assurance:run',
    'quarantine-reprocess-form': 'rollout:trigger',
    'gprs-replay-form': 'rollout:trigger',
  };
  const ACTION_PERMISSIONS = {
    'save-assurance': 'assurance:save',
    'rollout-approve': 'rollout:approve',
    'rollout-cancel': 'rollout:cancel',
    'purge-case': 'assurance:save',
    'promote-case': 'assurance:save',
    'user-delete': 'users:manage',
    'download-upload-fixture': 'assurance:save',
    'parser-sync-verify': 'parsers:read',
    'parser-restore': 'parsers:manage',
    'parser-sync-retry': 'rollout:trigger',
    'parser-sync-rollout-propose': 'rollout:trigger',
    'recovery-open-helper': 'assurance:save',
    'recovery-prepare-replay': 'rollout:trigger',
    'nupire-transition': 'parsers:manage',
  };
  const can = permission => !permission || state.user?.role === 'admin' || (state.user?.permissions || []).includes(permission);
  function actionPermission(button) {
    if(['ai-profile-from-preset','ai-profile-set-default','ai-profile-clear-default','ai-order-up','ai-order-down','ai-order-save','ai-order-reset'].includes(button.dataset.action))return 'system:health';
    if(['block-sense','block-preview','block-generate'].includes(button.dataset.action))return 'assurance:run';
    if (['quarantine-raw','quarantine-export','quarantine-diagnose','quarantine-acquire-evidence'].includes(button.dataset.action)) return 'logs:read';
    if (['quarantine-exclude','quarantine-include','quarantine-delete-raw'].includes(button.dataset.action)) return 'logs:write';
    if (['input-profile-edit','input-profile-delete','input-edit-row','input-toggle-row','input-toggle-all','input-remove-row','input-remove-all','input-add-row','input-use-automatic'].includes(button.dataset.action)) return 'parsers:manage';
    if (button.dataset.action==='input-profile-json') return 'parsers:read';
    if (['assurance-plan-edit','assurance-plan-delete','assurance-plan-reset'].includes(button.dataset.action)) return 'parsers:manage';
    if (button.dataset.action === 'assurance-plan-download') return 'parsers:read';
    if (button.dataset.action === 'assurance-run-download') return 'parsers:read';
    if (button.dataset.action === 'assurance-run-cancel') return 'assurance:save';
    if (button.dataset.action === 'assurance-parser-action') return ['approve','reject'].includes(button.dataset.value) ? 'parsers:review' : ['publish','rollback','offboard'].includes(button.dataset.value) ? 'rollout:trigger' : 'parsers:manage';
    if (['assurance-parser-edit','assurance-parser-delete'].includes(button.dataset.action)) return 'parsers:manage';
    if(button.dataset.action?.startsWith('workspace-ai-'))return 'assurance:run';
    if(button.dataset.action==='map-guided-sample')return 'assurance:run';
    if(button.dataset.action==='mapping-ai-readiness')return 'telemetry:read';if(['mapping-ai-suggest','mapping-ai-import'].includes(button.dataset.action))return 'assurance:run';if(button.dataset.action==='ai-proposal-reject')return 'parsers:review';if(button.dataset.action==='ai-proposal-remove')return 'parsers:manage';    if (button.dataset.action==='nfi-ai-help')return 'assurance:run';if(button.dataset.action==='nfi-ai-status')return 'telemetry:read';
    if (['gprs-use-manual','gprs-verify-manual','qrw-field-preview','kv-inspect','kv-apply-profile','qrw-controller-preview','qrw-generate-validate','nfi-learn','nfi-validate-candidate','qrw-suggest-fields'].includes(button.dataset.action)) return 'assurance:run';
    if (['nfi-repair-candidate','nfi-edit-candidate'].includes(button.dataset.action)) return 'parsers:manage';
    if (button.dataset.action === 'nfi-save-candidate') return 'assurance:save';
    if (button.dataset.action === 'gprs-preview') return 'assurance:run';
    if (button.dataset.action === 'gprs-open-editor') return 'assurance:save';
    if (button.dataset.action === 'parser-action') return ['approve', 'reject'].includes(button.dataset.value) ? 'parsers:review' : 'parsers:manage';
    if (button.dataset.action === 'parser-asset') return ['edit','delete'].includes(button.dataset.value) ? 'parsers:manage' : 'parsers:read';
    if (button.dataset.action === 'config-action') {
      if (button.dataset.value === 'approve') return 'parsers:review';
      if (['publish','revoke'].includes(button.dataset.value)) return 'rollout:trigger';
      return ['edit','copy','delete','submit'].includes(button.dataset.value) ? 'parsers:manage' : 'parsers:read';
    }
    if (button.dataset.action === 'engine-test') return 'assurance:run';
    if (button.dataset.action === 'output-config-test') return 'assurance:run';
    if (button.dataset.action === 'parser-sync-verify') return 'parsers:read';
    if (button.dataset.action === 'parser-sync-retry') return 'rollout:trigger';
    if (button.dataset.action === 'nupire-transition' && button.dataset.value === 'approve') return 'parsers:review';
    return ACTION_PERMISSIONS[button.dataset.action];
  }
  function applyPermissions() {
    Object.entries(FORM_PERMISSIONS).forEach(([id, permission]) => {
      document.querySelectorAll(`#${id}`).forEach(form => {
        if (can(permission)) return;
        form.dataset.permissionHidden = 'true';
        form.insertAdjacentHTML('beforebegin', `<div class="callout permission-callout">Read-only for the <b>${esc(state.user?.role || 'current')}</b> role. This action requires <code>${esc(permission)}</code>.</div>`);
      });
    });
    document.querySelectorAll('[data-action]').forEach(button => {
      const permission = actionPermission(button);
      if (permission && !can(permission)) button.dataset.permissionHidden = 'true';
    });
  }
  const page = el('page');
  const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const pretty = value => JSON.stringify(value ?? {}, null, 2);
  const timeFormatter = new Intl.DateTimeFormat(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', timeZoneName: 'short' });
  const time = value => {
    if (!value) return '—';
    const date = new Date(value);
    return Number.isNaN(date.getTime()) || date.getUTCFullYear() < 2000 ? '—' : timeFormatter.format(date);
  };
  const number = value => {
    if (value === undefined || value === null || value === '') return '—';
    const numeric = Number(value);
    return Number.isFinite(numeric) ? numeric.toLocaleString() : '—';
  };
  const countText = value => { if(typeof value==='string'&&/^\d+$/.test(value)){try{return BigInt(value).toLocaleString();}catch{}}return number(value); };
  const badge = value => `<span class="status ${/ok|online|active|approved|passed|validated|success/i.test(value) ? 'good' : /fail|error|offline|tamper|rejected/i.test(value) ? 'bad' : /pending|drift|stale|review|degraded/i.test(value) ? 'warn' : 'info'}"><i></i>${esc(value || 'unknown')}</span>`;
  const rolloutBundleHash = doc => { const payload=doc?.payload||{};return String(payload.approval_bundle_hash||payload.effective_bundle_hash||payload.canonical_bundle_hash||payload.bundle_hash||payload.payload?.bundle_hash||''); };
  const isParserSyncProposal = doc => { const payload=doc?.payload||{},inner=payload.payload||{};return payload.type==='parser_sync'||!!inner.asset_bundle||!!inner.bundle_hash; };
  function rolloutApprovalProblem(doc,currentBundleHash='') {
    const payload=doc?.payload||{},reported=payload.approval_blocked_reason||payload.approval_error||doc?.approval_blocked_reason,integrity=String(payload.approval_integrity||doc?.approval_integrity||'').toLowerCase();
    if(reported)return String(reported);
    if(!isParserSyncProposal(doc))return '';
    if(currentBundleHash===null)return 'The current published parser bundle could not be verified. Refresh before approving this parser rollout.';
    const hash=rolloutBundleHash(doc);
    if(/blocked|invalid|unsafe|stale/.test(integrity))return 'The controller marked this parser proposal as unsafe or stale. Cancel it and create a new proposal from the current published bundle.';
    if(currentBundleHash&&/^[0-9a-f]{64}$/i.test(hash)&&hash!==currentBundleHash)return 'This proposal targets an older parser bundle. Cancel it and create a new proposal for the current published bundle.';
    return '';
  }
  function rolloutApprovalErrorMessage(error) {
    const message=String(error?.message||error||'Approval failed');
    if(/bundle hash is invalid|exact parser bundle snapshot|complete sha-?256/i.test(message))return 'The stored proposal cannot be safely approved because its exact parser bundle snapshot is invalid. Cancel this proposal and create a new one from the current published bundle.';
    if(/stale|older bundle|does not match the current/i.test(message))return 'This proposal no longer matches the current published parser bundle. Cancel it and create a replacement proposal.';
    if(/creator cannot approve|different.*reviewer/i.test(message))return 'Independent approval is required. Sign in as a different authorized reviewer, or use the audited administrator approval with its exact revision, hash, and reason.';
    return message;
  }
  function approvalDetails(doc) {
    const payload=doc?.payload||{};
    const actor=payload.approved_by||payload.emergency_approved_by||((payload.approval_reason||payload.emergency_approval_reason)?payload.reviewed_by:'');
    const reason=payload.approval_reason||payload.emergency_approval_reason;
    if(!actor)return '';
    return `<div class="approval-details"><small>Approved${actor?` by ${esc(actor)}`:''}</small>${reason?`<p style="white-space:pre-wrap;overflow-wrap:anywhere">${esc(reason)}</p>`:''}</div>`;
  }
  function rolloutApprovalControls(doc,currentBundleHash='') {
    if(doc?.state!=='pending_approval')return approvalDetails(doc)||'—';
    const id=String(doc.id||''),isCreator=doc.created_by&&doc.created_by===state.user?.username,emergency=isCreator&&state.user?.role==='admin',problem=rolloutApprovalProblem(doc,currentBundleHash),actionError=state.rolloutActionErrors[id]||'',blocked=!!problem||!!actionError||(isCreator&&!emergency),minimum=emergency?12:3;
    const explanation=problem||actionError||(isCreator&&!emergency?'A different authorized reviewer must approve this proposal.':emergency?(isParserSyncProposal(doc)?'Administrator approval is bound to this exact revision and bundle hash.':'Administrator self-approval records your identity and review reason.'):'Record an independent review reason before approval.');
    return `<div class="rollout-approval-editor"><label>Approval reason<input data-rollout-reason="${esc(id)}" minlength="${minimum}" maxlength="512" placeholder="${emergency?'Administrator approval reason (12+ characters)':'Independent review reason'}"></label><div class="actions"><button class="${emergency?'warning':'primary'} small" data-action="rollout-approve" data-id="${esc(id)}" ${blocked?'disabled':''}>${emergency?(isParserSyncProposal(doc)?'Administrator approve exact bundle':'Administrator approve'):'Approve reviewed proposal'}</button><button class="danger small" data-action="rollout-cancel" data-id="${esc(id)}">Cancel</button></div><small class="${problem||actionError?'validation-error':'hint'}" role="${problem||actionError?'alert':'status'}">${esc(explanation)}</small></div>`;
  }
  const empty = text => `<div class="empty-state"><div><b>No records</b><p>${esc(text)}</p></div></div>`;
  const loading = () => `<div class="loading-state"><div><span></span><b>Loading live controller data</b></div></div>`;
  const errorState = error => `<div class="error-state"><div><b>Request failed</b><p>${esc(error.message || error)}</p><button class="secondary" data-action="reload">Retry</button></div></div>`;
  const card = (label, value, note = '') => `<article class="card metric"><div class="metric-label">${esc(label)}</div><div class="metric-value">${esc(value)}</div><div class="metric-note">${esc(note)}</div></article>`;
  const panel = (title, subtitle, body, actions = '') => `<section class="panel"><div class="section-head"><div><h3>${esc(title)}</h3><p>${esc(subtitle)}</p></div><div class="actions">${actions}</div></div>${body}</section>`;
  function sparkline(labels, values, color) {
    const width=560,height=150,pad=16,max=Math.max(1,...values),points=values.map((value,i)=>`${pad+i*(width-pad*2)/Math.max(values.length-1,1)},${height-pad-(value/max)*(height-pad*2)}`).join(' ');
    const circles=values.map((value,i)=>{const x=pad+i*(width-pad*2)/Math.max(values.length-1,1),y=height-pad-(value/max)*(height-pad*2);return `<circle cx="${x}" cy="${y}" r="3" fill="${color}"><title>${esc(labels[i])}: ${number(value)}</title></circle>`;}).join('');
    return `<div class="chart-wrap"><svg class="chart" viewBox="0 0 ${width} ${height}" role="img" aria-label="Time-series line chart"><path d="M ${pad} ${height-pad} L ${points} L ${width-pad} ${height-pad} Z" fill="${color}" opacity=".10"/><polyline points="${points}" fill="none" stroke="${color}" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>${circles}</svg><div class="chart-labels">${labels.filter((_,i)=>i%3===0).map(label=>`<span>${esc(label)}</span>`).join('')}</div></div>`;
  }
  function multiSeriesChart(labels, series) {
    const width=820,height=260,left=64,right=12,top=16,bottom=34,plotW=width-left-right,plotH=height-top-bottom,max=Math.max(1,...series.flatMap(item=>item.values.map(Number))),tickCount=4;
    const x=index=>left+index*plotW/Math.max(labels.length-1,1),y=value=>top+plotH-(Number(value||0)/max)*plotH;
    const grid=Array.from({length:tickCount+1},(_,index)=>{const value=max*(tickCount-index)/tickCount,yy=top+plotH*index/tickCount;return `<line x1="${left}" y1="${yy}" x2="${width-right}" y2="${yy}" class="chart-grid"/><text x="${left-8}" y="${yy+4}" text-anchor="end" class="chart-tick">${number(Math.round(value))}</text>`;}).join('');
    const paths=series.map(item=>{const unit=item.unit||'records';const points=item.values.map((value,index)=>`${x(index)},${y(value)}`).join(' ');const marks=item.values.map((value,index)=>`<circle class="chart-point" tabindex="0" role="img" cx="${x(index)}" cy="${y(value)}" r="4" fill="${item.color}" data-chart-category="${esc(item.label)}" data-chart-time="${esc(labels[index])}" data-chart-value="${esc(String(value))}" data-chart-unit="${esc(unit)}" aria-label="${esc(item.label)}, ${esc(labels[index])}: ${countText(value)} ${esc(unit)}"><title>${esc(item.label)} · ${esc(labels[index])} · ${countText(value)} ${esc(unit)}</title></circle>`).join('');return `<polyline points="${points}" fill="none" stroke="${item.color}" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"/>${marks}`;}).join('');
    const tickIndexes=labels.map((_,i)=>i).filter(i=>i===0||i===labels.length-1||i%Math.max(1,Math.ceil(labels.length/6))===0);
    return `<div class="multi-chart"><div class="chart-legend">${series.map(item=>`<span><i style="background:${item.color}"></i>${esc(item.label)} <small>(${esc(item.unit||'records')})</small></span>`).join('')}</div><div class="chart-hover-readout" aria-live="polite">Hover or focus a point for category, exact count, unit, and controller time.</div><svg viewBox="0 0 ${width} ${height}" role="group" aria-label="Time series: event count on the vertical axis and controller time on the horizontal axis">${grid}<path d="M${left} ${top}V${top+plotH}H${width-right}" class="chart-axis"/><text transform="translate(14 ${top+plotH/2}) rotate(-90)" class="chart-axis-title" text-anchor="middle">Records</text>${paths}</svg><div class="chart-axis-caption"><b>X axis:</b> controller time <span><b>Y axis:</b> exact records per bucket</span></div><div class="chart-labels">${tickIndexes.map(index=>`<span>${esc(labels[index])}</span>`).join('')}</div></div>`;
  }
  function dashboardKPI(label,value,note,tone='blue') { return `<article class="dashboard-kpi ${esc(tone)}"><span>${esc(label)}</span><strong>${esc(value)}</strong><small>${esc(note)}</small></article>`; }
  function compactTrend(values,color='#53a5ff') {
    const safe=(values||[]).map(value=>Number(value||0)),width=150,height=34,max=Math.max(1,...safe),points=safe.map((value,index)=>`${index*width/Math.max(safe.length-1,1)},${height-(value/max)*(height-3)}`).join(' ');
    return `<svg class="compact-trend" viewBox="0 0 ${width} ${height}" role="img" aria-label="Recent metric trend"><polyline points="${points}" fill="none" stroke="${color}" stroke-width="2" vector-effect="non-scaling-stroke"/></svg>`;
  }
  function stackedVolumeChart(labels,buckets) {
    const max=Math.max(1,...buckets.map(item=>item.received||item.parsed+item.quarantine+item.errors));
    return `<div class="stacked-volume" role="img" aria-label="Parsed, quarantined, and error volumes by time bucket"><div class="chart-hover-readout" aria-live="polite">Hover or focus a bar for exact route counts and controller time.</div><div class="stacked-bars">${buckets.map((item,index)=>{const scale=value=>Math.max(0,Number(value||0))/max*100,summary=`${labels[index]} · Parsed ${countText(item.parsed)} records · Quarantined ${countText(item.quarantine)} records · Errors ${countText(item.errors)} records`;return `<span class="stack-column" tabindex="0" data-chart-summary="${esc(summary)}" aria-label="${esc(summary)}" title="${esc(summary)}"><i class="stack-parsed" style="height:${scale(item.parsed)}%"></i><i class="stack-quarantine" style="height:${scale(item.quarantine)}%"></i><i class="stack-errors" style="height:${scale(item.errors)}%"></i></span>`;}).join('')}</div><div class="chart-axis-caption"><b>X axis:</b> controller time <span><b>Y axis:</b> exact records per bucket</span></div><div class="chart-legend"><span><i style="background:#3dd598"></i>Parsed (records)</span><span><i style="background:#f8c15c"></i>Quarantined (records)</span><span><i style="background:#ff6b7d"></i>Errors (records)</span></div></div>`;
  }
  const table = (headers, rows, emptyMessage = 'No records have been created yet.') => rows.length ? `<div class="table-wrap"><table><thead><tr>${headers.map(h => `<th>${esc(h)}</th>`).join('')}</tr></thead><tbody>${rows.join('')}</tbody></table></div>` : empty(emptyMessage);
  function addInputGuidance(root=page) {
    const guidance={
      raw:'Paste one line to test a single record, or several lines to test a multiline record. Remove credentials first.',
      desired:'Enter the exact expected JSON, for example {"status":"failed"} or {"retry_count":2}.',
      mapping:'Map source keys to output names, for example {"src":"source_ip"} or {"user":"actor"}.',
      priority:'Lower values are tried first. A rule at priority 10 runs before a rule at 20.',
      id:'Use a stable ID such as mail-parser or client-02.',
      targets:'Enter one enrolled client ID, such as client-02, or several separated by commas.',
      config:'Enter valid JSON. For example, use {"enabled":true} when this configuration accepts an enabled flag.',
      file:'Choose a representative log file or a supported archive. A one-line file is useful for checking a single pattern.'
    };
    for(const label of root.querySelectorAll('label.field')){
      const control=label.querySelector('input,textarea,select');
      if(!control||['checkbox','radio','hidden'].includes(control.type)||label.querySelector('.input-help'))continue;
      const name=control.name||control.id||'value';
      const tip=control.title||guidance[name]||(control.placeholder?'Example: '+control.placeholder:'Enter '+name.replaceAll('_',' ')+' and check the validation message before submitting.');
      const icon=document.createElement('button');
      icon.type='button';
      icon.className='input-help';
      icon.textContent='ⓘ';
      icon.title=tip;
      icon.setAttribute('aria-label','Show input help');
      icon.setAttribute('aria-expanded','false');
      const explanation=document.createElement('span');
      explanation.className='input-help-example';
      explanation.hidden=true;
      explanation.textContent=tip;
      const visibleName=label.firstChild?.textContent?.trim();
      if(visibleName&&!control.hasAttribute('aria-label')&&!control.hasAttribute('aria-labelledby'))control.setAttribute('aria-label',visibleName);
      icon.addEventListener('click',event=>{
        event.preventDefault();
        explanation.hidden=!explanation.hidden;
        icon.setAttribute('aria-expanded',String(!explanation.hidden));
      });
      control.before(icon,explanation);
    }
  }
  const QRW_RESERVED=new Set(['raw','nli','meta','source','envelope','parser_id','attempt_count','ordered_attempts','parse_telemetry','__proto__','prototype','constructor']);
  function qrwNormalizeSuggestions(fields){const used=new Set();return fields.map(field=>{let base=String(field.name||'field').replace(/[^A-Za-z0-9_]/g,'_');if(!/^[A-Za-z]/.test(base))base='field'+base;if(QRW_RESERVED.has(base.toLowerCase()))base+='_value';base=base.slice(0,58);let name=base,index=2;while(used.has(name))name=base+'_'+index++;used.add(name);return {...field,name,label:field.label||name.replaceAll('_',' ')};});}
  const NFI_NUMBER_KEYS=['occurrence_index','count','limit','part','start','end','kv_max_pairs'];
  const NFI_JSON_KEYS=['indices','token_separators','assignment_separators','kv_pair_separators','kv_brackets','key_aliases','key_include','key_exclude','key_value_edits'];
  const NFI_ADVANCED_KEYS=['boundary_start','boundary_end','occurrence','occurrence_index','preserve_boundary','direction','count','unit','indices','token_mode','token_separators','token_pattern','assignment_separators','key_aliases','key_include','key_exclude','key_prefix','key_suffix','output_policy','end_of_line','require_complete','empty_policy','custom_type','custom_pattern','kv_profile','kv_start','kv_pair_separators','kv_quotes','kv_escape','kv_brackets','kv_key_pattern','kv_reject_empty','kv_nested_objects','kv_literal_boundaries','kv_bare_field','kv_empty_key_field','kv_header_pattern','kv_max_pairs','key_value_edits'];
  const NFI_PARAMETER_OPTIONS={occurrence:['first','last','nth','all'],direction:['left','right'],unit:['character','byte','word','token','line'],token_mode:['whitespace','delimiter','mixed','regex'],output_policy:['include','flatten','omit','intermediate'],empty_policy:['preserve','missing','null','reject'],condition_operator:['exists','equals','not_empty'],algorithm:['sha256','sha512']};
  function renderFieldSenseParameters(value,operation=false){const strategy=operation?value.op:value.method,groups={before:['boundary_start'],after:['boundary_start'],between:['boundary_start','boundary_end'],first:['count','unit'],last:['count','unit'],range:['unit','indices'],count:['unit','token_mode','token_separators','token_pattern'],word:['indices','token_mode','token_separators','token_pattern'],token:['indices','token_mode','token_separators','token_pattern'],token_match:['occurrence','occurrence_index','token_mode','token_separators','token_pattern'],tokenize:['token_mode','token_separators','token_pattern'],split:['direction','count','limit','preserve_delimiter'],split_regex:['direction','count','limit','preserve_delimiter'],split_characters:['count','limit'],split_words:['count','limit','token_mode','token_separators','token_pattern'],split_tokens:['count','limit','token_mode','token_separators','token_pattern'],conditional_merge:['condition_field','condition_operator','condition_value','else_value','remove_sources'],merge:['remove_sources'],join:['remove_sources'],hash:['algorithm'],regex_all:['occurrence','occurrence_index'],regex_groups:[]},kv=['assignment_separators','key_aliases','key_include','key_exclude','key_prefix','key_suffix'];let keys=[...(groups[strategy]||[])];if(operation&&strategy==='key_value')keys.push('assignment_separators');if(['before','after','between'].includes(strategy))keys.push('occurrence','occurrence_index','preserve_boundary');if(['key_value','kv_all','kv_eol'].includes(strategy))keys.push(...kv.filter(key=>key!=='assignment_separators'));if(strategy==='key_value')keys.push('kv_start','source_end','unit','end_of_line');
    if(!operation&&['key_value','kv_all','kv_eol'].includes(strategy)){
      if(strategy==='key_value')keys.push('kv_pair_separators');
      keys.push('kv_quotes','kv_escape','kv_brackets','kv_key_pattern','kv_reject_empty');
      if(strategy!=='key_value'||value.type==='object')keys.push('kv_nested_objects');
    }if(!operation&&['kv_all','kv_eol'].includes(strategy))keys.push('unit','require_complete','kv_literal_boundaries','kv_bare_field','kv_empty_key_field','kv_header_pattern','kv_max_pairs');if(strategy==='kv_all')keys.push('end_of_line');if(!operation){keys.push('empty_policy');if(value.type==='custom')keys.push('custom_type','custom_pattern');}return keys.length?`<details class="nfi-extended"><summary>Advanced options for ${esc(strategy.replaceAll('_',' '))}</summary><div class="nfi-parameter-grid">${keys.map(key=>{const attr=operation?'data-qrw-op-key':'data-qrw-key',current=value[key],label=key==='empty_policy'&&['kv_all','kv_eol'].includes(strategy)?'Empty object policy (empty pair values are preserved)':key.replaceAll('_',' '),options=NFI_PARAMETER_OPTIONS[key],isBool=['preserve_boundary','preserve_delimiter','remove_sources','require_complete','end_of_line','kv_reject_empty','kv_nested_objects','kv_literal_boundaries'].includes(key);return `<label>${esc(label)}${isBool?`<input type="checkbox" ${attr}="${key}" ${current?'checked':''}>`:options?`<select ${attr}="${key}">${qrwOption('',current||'')}${options.map(option=>qrwOption(option,current)).join('')}</select>`:`<input ${attr}="${key}" ${NFI_NUMBER_KEYS.includes(key)?'type="number" min="0"':''} value="${esc(NFI_JSON_KEYS.includes(key)?current===undefined?'':JSON.stringify(current):current??'')}" placeholder="${NFI_JSON_KEYS.includes(key)?key==='key_aliases'?'JSON object {&quot;src&quot;:&quot;source_ip&quot;}':key==='assignment_separators'?'JSON array, e.g. [&quot;=&quot;,&quot;:&quot;]':'JSON array, e.g. [1,3]':'optional'}">`}</label>`;}).join('')}</div></details>`:'';}
  const QRW_TYPES=['string','integer','number','boolean','timestamp','ip','port','object','array','float','date','datetime','ipv6','email','url','uuid','null','custom'];
  const QRW_POLICIES=['required','optional','nullable','default','derived','unresolved'];
  const QRW_METHODS=['position','delimiter','key_value','kv_eol','character','byte','line','regex','json_path','derived','constant','raw','field','word','token','token_match','before','after','between','count','kv_all','regex_all','regex_groups'];
  const QRW_OPERATIONS=['split','select','substring','character','byte','line','trim','lowercase','uppercase','replace','remove','prefix','suffix','strip_prefix','strip_suffix','regex_extract','regex_replace','key_value','join','merge','add','first','last','range','count','before','after','between','tokenize','token_match','split_regex','split_characters','split_words','split_tokens','conditional_merge','copy','mask','hash','redact','normalize','omit','exclude'];
  let qrwFieldSequence=0;
  let qrwControllerPreview=null;
  let qrwLivePreviewController;
  let kvInspectorState;
  const isKVTail = field => ['kv_all','kv_eol'].includes(field?.method);
  function qrwField(overrides={}){return {id:`field-${Date.now()}-${++qrwFieldSequence}`,name:'field',label:'Field',description:'',example:'',type:'string',policy:'unresolved',method:'position',source:'1',source_end:'',delimiter:';',part:'1',transform:'none',timestamp_format:'',timezone:'preserve',timezone_name:'',default_value:'',null_policy:'omit',validation:'',kv_separator:'=',pair_delimiters:':;,',duplicate_policy:'reject',join_separator:' ',operations:[],preview:'',evidence:'Operator-defined name; value not verified',confidence:'unverified',status:'unresolved',confirmed:false,quote_aware:true,...overrides,...(overrides.method==='kv_all'&&overrides.end_of_line===true?{method:'kv_eol'}:{})};}
  function aiEvidenceSource(raw, marked) {
    const source = state.lastAssurance?.request;
    if (marked === 'production' || (raw && [source,...(state.gprsSamples || [])].some(sample => sample?.raw === raw && /^(quarantine|fixture|live|production):/.test(sample.source || '')))) return 'production';
    return 'manual';
  }
  function qrwRaw(){return page.querySelector('#nfi-raw')?.value??page.querySelector('#gprs-preview-form [name=raw_evidence]')?.value??state.lastAssurance?.request?.raw??state.nfiRaw??'';}
  function qrwTokens(raw){const text=String(raw||''),tokens=[];let chars=0,bytes=0,last=0;const encoder=new TextEncoder();for(const match of text.matchAll(/\S+/gu)){const gap=text.slice(last,match.index);chars+=[...gap].length;bytes+=encoder.encode(gap).length;const charStart=chars+1,byteStart=bytes+1;chars+=[...match[0]].length;bytes+=encoder.encode(match[0]).length;last=match.index+match[0].length;tokens.push({value:match[0],start:match.index,end:last,charStart,charEnd:chars,byteStart,byteEnd:bytes});if(tokens.length>=4096)break;}return tokens;}
  function qrwDelimited(raw,delimiter){if(!delimiter)throw new Error('Delimiter must not be empty.');const values=[];let value='',quote='';for(let index=0;index<String(raw).length;index++){const character=raw[index];if(quote){if(character===quote){if(raw[index+1]===quote){value+=quote;index++;}else quote='';}else if(character==='\\'&&raw[index+1]){value+=raw[++index];}else value+=character;}else if(character==='"'||character==="'"){quote=character;}else if(raw.startsWith(delimiter,index)){values.push(value);value='';index+=delimiter.length-1;if(values.length>4096)throw new Error('Delimiter split exceeds 4096 parts.');}else value+=character;}if(quote)throw new Error('Unterminated quoted delimited value.');values.push(value);return values;}
  // Pair boundaries are separators followed by another key and its assignment.
  // Colons in a URL, IPv6 address, or quoted message are ordinary value bytes.
  function qrwPairs(raw,assignment='=',separators=':;,'){
    if(!assignment||assignment.length>8)throw new Error('Assignment separator must contain 1–8 characters.');
    const text=String(raw),pairs=[],isBoundary=c=>c==null||/\s/.test(c)||separators.includes(c),keyAt=index=>{const m=/^[A-Za-z_][A-Za-z0-9_.-]*/.exec(text.slice(index));if(!m)return null;let end=index+m[0].length;while(/[ \t]/.test(text[end]||'')&&end<text.length)end++;return text.startsWith(assignment,end)?{key:m[0],start:index,valueStart:end+assignment.length}:null;};
    let quote='',escaped=false;
    for(let index=0;index<text.length;index++){
      const c=text[index];if(escaped){escaped=false;continue;}if(c==='\\'){escaped=true;continue;}if(quote){if(c===quote)quote='';continue;}if(c==='"'||c==="'"){quote=c;continue;}
      if(index>0&&!isBoundary(text[index-1]))continue;const key=keyAt(index);if(!key)continue;
      let cursor=key.valueStart,valueQuote='',valueEscape=false,end=text.length,next=text.length;
      for(;cursor<text.length;cursor++){const v=text[cursor];if(valueEscape){valueEscape=false;continue;}if(v==='\\'){valueEscape=true;continue;}if(valueQuote){if(v===valueQuote)valueQuote='';continue;}if(v==='"'||v==="'"){valueQuote=v;continue;}if(isBoundary(v)){let look=cursor;while(look<text.length&&isBoundary(text[look]))look++;if(keyAt(look)){end=cursor;next=look;break;}cursor=look-1;cursor=look-1;}}
      if(valueQuote)throw new Error('Unterminated quoted key/value value.');
      let value=text.slice(key.valueStart,end).trim();if((value.startsWith('"')&&value.endsWith('"'))||(value.startsWith("'")&&value.endsWith("'")))value=value.slice(1,-1);value=value.replace(/\\([\\"':;,= ])/g,'$1');pairs.push({...key,value,end});if(pairs.length>128)throw new Error('A mapping can inspect at most 128 key/value pairs.');index=next-1;
    }
    if(quote)throw new Error('Unterminated quote in raw evidence.');return pairs;
  }
  function qrwSuggestedFields(raw,existingFields=[]){
    const text=String(raw||''),tokens=qrwTokens(text);if(!tokens.length)return [];
    if(/^[\[{]/.test(text.trim())){try{const value=JSON.parse(text),fields=[];const visit=(item,path,parts,depth)=>{if(fields.length>=128||depth>16)return;if(item!==null&&typeof item==='object'&&Object.keys(item).length){for(const key of Object.keys(item)){if(!/^[A-Za-z0-9_-]+$/.test(key)||['__proto__','prototype','constructor'].includes(key))continue;visit(item[key],Array.isArray(item)?path+'['+key+']':path+'.'+key,[...parts,key],depth+1);}}else{const name=(parts.join('_')||'value').replace(/^[^A-Za-z_]/,'_$&');fields.push(qrwField({name,label:parts.join(' / ')||'Value',type:item===null?'string':Array.isArray(item)?'array':typeof item==='object'?'object':typeof item==='boolean'?'boolean':typeof item==='number'?(Number.isSafeInteger(item)?'integer':'number'):'string',policy:item===null?'nullable':'required',method:'json_path',source:path,preview:typeof item==='string'?item:JSON.stringify(item),confidence:'structural_suggestion',status:'unresolved',evidence:'Observed JSON path; original nesting is preserved by extraction'}));}};visit(value,'$',[],0);if(fields.length)return qrwNormalizeSuggestions(fields);}catch{}}
let pairs=[];try{pairs=qrwPairs(text);}catch{}const firstPair=pairs.length?Math.min(...pairs.map(pair=>pair.start)):text.length;
    if(!pairs.length&&!/^\d{8}\s+\d{9}[+-]\d{4}\s/.test(text)){let selected=null;for(const delimiter of ['\t','|',',',';']){try{const parts=qrwDelimited(text,delimiter);if(parts.length>1&&parts.length<=128&&(!selected||parts.length>selected.parts.length))selected={delimiter,parts};}catch{}}if(selected)return qrwNormalizeSuggestions(selected.parts.map((value,index)=>qrwField({name:'column_'+(index+1),label:'Column '+(index+1),policy:'required',method:'delimiter',source:'raw',delimiter:selected.delimiter,part:String(index+1),preview:value,confidence:'proposed',status:'needs_confirmation',evidence:'Quote-aware delimited record; confirm column interpretation'})));}
    const prefix=tokens.filter(token=>token.start<firstPair),mailShape=prefix.length>=8&&/^\d{8}$/.test(prefix[0]?.value)&&/^\d{9}[+-]\d{4}$/.test(prefix[1]?.value)&&/^\d+$/.test(prefix[4]?.value)&&/^\d+$/.test(prefix[5]?.value)&&/^\d+$/.test(prefix[6]?.value),hasPairs=pairs.length>0,colonPairs=pairs.some(pair=>text.slice(pair.end).startsWith(':'));
    const mailNames=hasPairs?['date','timestamp','hostname','service','pid','minor_pid','thread_id']:['date','time_timezone','hostname','service','pid','status','minor_pid'];
    const base=[];
    for(const [index,token] of prefix.entries()){
      if(base.length>=128)break;const compound=/^[^;|,]+([;|,])[^;|,]+(?:\1[^;|,]+)*$/.exec(token.value),parts=compound?token.value.split(compound[1]):[];
      if(compound&&parts.length<=16){for(const [part,value] of parts.entries()){const knownLevel=/^(note|trace|debug|info|notice|warn|warning|error|fatal|critical)$/i.test(parts[0]);let name=`token_${index+1}_part_${part+1}`;if(mailShape&&index===7)name=colonPairs?(part===0?'loglevel':part===1?'event':name):hasPairs?(part===0?'log_level':part===1?'event_type':name):(part===0?'event_type':part===1?'event_name':name);else if(knownLevel)name=part===0?'loglevel':part===1?'event':name;base.push(qrwField({name,label:name.replaceAll('_',' '),policy:'required',method:'delimiter',source:String(index+1),delimiter:compound[1],part:String(part+1),preview:value,evidence:`Observed token ${index+1}; confirm ${JSON.stringify(compound[1])} split`,confidence:'proposed',status:'needs_confirmation'}));}}
      else{const name=mailShape&&index<7?mailNames[index]:`token_${index+1}`,type=mailShape&&([4,5].includes(index)||!hasPairs&&index===6)&&Number.isSafeInteger(Number(token.value))&&!/^0\d/.test(token.value)?'integer':'string';base.push(qrwField({name,label:name.replaceAll('_',' '),type,policy:'required',method:'position',source:String(index+1),preview:token.value,evidence:`Observed token ${index+1} · characters ${token.charStart}–${token.charEnd}`,confidence:mailShape?'structural_suggestion':'suggested',status:'mapped'}));}
    }
    for(const pair of pairs){if(base.length>=128)break;if(base.some(field=>field.method==='key_value'&&field.source===pair.key))continue;const safeName=pair.key.replace(/[^A-Za-z0-9_]/g,'_');base.push(qrwField({name:safeName,label:safeName.replaceAll('_',' '),policy:'required',method:'key_value',source:pair.key,preview:pair.value,evidence:`Observed key ${pair.key}; separators are recognized only before another key=`,confidence:'suggested',status:'mapped'}));}
    const used=new Set();for(const field of base){const old=(existingFields||[]).find(item=>!used.has(item.id)&&item.method===field.method&&String(item.source)===String(field.source)&&(field.method!=='delimiter'||String(item.part)===String(field.part)));if(old){used.add(old.id);if(old.name&&!/^token_\d+(?:_part_\d+)?$/.test(old.name)){field.name=old.name;field.label=old.label;}}}
    for(const old of existingFields||[]){if(!used.has(old.id)&&!/^token_\d+$/.test(old.name||'')&&base.length<128){const {id:discarded,...copy}=old;base.push(qrwField(copy));}}return qrwNormalizeSuggestions(base.slice(0,128));
  }
  function qrwMergeControllerSuggestions(suggestions,existing=[]){const used=new Set(),same=(left,right)=>left.method===right.method&&String(left.source??'')===String(right.source??'')&&String(left.source_end??'')===String(right.source_end??'')&&(left.method!=='delimiter'||left.delimiter===right.delimiter&&String(left.part)===String(right.part));const fields=suggestions.map(suggestion=>{const old=existing.find(item=>!used.has(item.id)&&same(item,suggestion));if(old&&!/^(?:token_\d+(?:_part_\d+)?|column_\d+|field_\d+)$/.test(old.name||'')){used.add(old.id);return qrwField(JSON.parse(JSON.stringify(old)));}if(old)used.add(old.id);return qrwField({...suggestion,status:suggestion.method==='delimiter'&&!suggestion.confirmed?'needs_confirmation':'unresolved'});});for(const old of existing)if(!used.has(old.id)&&fields.length<128&&!/^(?:token_\d+(?:_part_\d+)?|column_\d+|field_\d+)$/.test(old.name||''))fields.push(qrwField(JSON.parse(JSON.stringify(old))));return qrwNormalizeSuggestions(fields.slice(0,128));}
  function qrwOption(value,current){return `<option value="${esc(value)}" ${value===current?'selected':''}>${esc(value==='kv_eol'?'Key value EOL — End of line':value==='kv_all'?'All key values · configured range':({position:'Token position',field:'Earlier mapped field',raw:'Whole raw record',regex_groups:'Regex named groups',intermediate:'Intermediate (hide from output)',include:'Include in output',flatten:'Flatten into top-level keys'})[value]||value.replaceAll('_',' '))}</option>`;}
  function qrwPlan(fields=state.qrwFields){
    const keys=['description','example','name','label','type','policy','method','source','source_end','delimiter','part','confirmed','transform','timestamp_format','timezone','default_value','validation','null_policy','quote_aware','timezone_name','kv_separator','pair_delimiters','duplicate_policy','join_separator','operations'];
    return {version:1,fields:fields.map(original=>{
      const field=original.method==='kv_eol'?{...original,method:'kv_all',end_of_line:true}:original;
      const advanced=NFI_ADVANCED_KEYS.filter(key=>field[key]!==undefined &&
        (key!=='kv_start'||field.method==='key_value'&&field.kv_profile) &&
        (key!=='key_value_edits'||field.method==='kv_all'));
      return Object.fromEntries([
        ...keys.map(key=>[key,field[key]??(key==='operations'?[]:'')]),
        ...advanced.map(key=>[key,field[key]]),
        ...(field.method==='kv_all'&&field.kv_input_edits?.length?[['kv_input_edits',field.kv_input_edits]]:[])
      ]);
    })};
  }
  function qrwRawInspector(raw=qrwRaw()){const tokens=qrwTokens(raw);return `<div class="nfi-inspector"><div class="section-head"><div><h4>Raw evidence position explorer</h4><p>Click a token to map it. Unicode characters, bytes, words, and lines use 1-based inclusive positions.</p></div><button type="button" class="secondary small" data-action="qrw-selection-field">Map selected raw text</button></div><div class="nfi-source-lines"><pre class="nfi-line-numbers" aria-hidden="true">${String(raw||'').split(/\r?\n/).slice(0,8192).map((_,index)=>index+1).join('\n')}</pre><pre id="nfi-raw-selection" tabindex="0">${esc(raw||'Paste or inspect a raw record to reveal its positions.')}</pre></div><p class="hint">${[...String(raw)].length} Unicode characters · ${new TextEncoder().encode(String(raw)).length} UTF-8 bytes · ${tokens.length} whitespace tokens · delimiters: ${esc([...new Set(String(raw).match(/[^\p{L}\p{N}\s]/gu)||[])].slice(0,24).join(' '))}</p><p id="nfi-source-hint" class="hint">Local text-match hints may select the first repeated value. Select Preview here for exact source spans.</p><div class="nfi-token-strip">${tokens.slice(0,128).map((token,index)=>`<button type="button" class="nfi-token" data-action="qrw-token-field" data-token="${index+1}" title="Token ${index+1}; characters ${token.charStart}–${token.charEnd}; bytes ${token.byteStart}–${token.byteEnd}"><b>${index+1}</b><code>${esc(token.value)}</code><small>chars ${token.charStart}–${token.charEnd}</small></button>`).join('')}</div></div>`;}
  function renderQRWOperations(field){return `<details class="nfi-operations"><summary>${(field.operations||[]).length} ordered operation(s) · split, merge, replace and more</summary>${(field.operations||[]).map((op,index)=>`<div class="nfi-operation" data-qrw-op="${index}"><label>Operation<select data-qrw-op-key="op">${QRW_OPERATIONS.map(value=>qrwOption(value,op.op)).join('')}</select></label><label>Delimiter<input data-qrw-op-key="delimiter" value="${esc(op.delimiter||'')}" maxlength="32"></label><label>Respect quoted split values<input type="checkbox" data-qrw-op-key="quote_aware" ${op.quote_aware===false?'':'checked'}></label><label>Part · 0 keeps array<input data-qrw-op-key="part" type="number" min="0" value="${Number(op.part)||0}"></label><label>Start<input data-qrw-op-key="start" type="number" min="1" value="${Number(op.start)||1}"></label><label>End · 0 to end<input data-qrw-op-key="end" type="number" min="0" value="${Number(op.end)||0}"></label><label>Literal / prefix / suffix<input data-qrw-op-key="value" value="${esc(op.value||'')}"></label><label>Find<input data-qrw-op-key="find" value="${esc(op.find||'')}"></label><label>Replace with<input data-qrw-op-key="replace" value="${esc(op.replace||'')}"></label><label>RE2 pattern<input data-qrw-op-key="pattern" value="${esc(op.pattern||'')}" maxlength="4096"></label><label>Capture group<input data-qrw-op-key="group" value="${esc(op.group||'')}"></label><label>Earlier fields · comma separated<input data-qrw-op-key="fields" value="${esc((op.fields||[]).join(','))}"></label><label>Join separator<input data-qrw-op-key="separator" value="${esc(op.separator??' ')}"></label>${renderFieldSenseParameters(op,true)}<div class="actions"><button class="secondary small" type="button" data-action="qrw-operation-up" data-id="${esc(field.id)}" data-index="${index}" ${index===0?'disabled':''}>Move up</button><button class="danger small" type="button" data-action="qrw-operation-delete" data-id="${esc(field.id)}" data-index="${index}">Remove operation</button></div></div>`).join('')}<button class="secondary small" type="button" data-action="qrw-operation-add" data-id="${esc(field.id)}">Add operation</button><p class="hint">Operations run in order. Use Replace with Find <code>failed</code> and Replace with <code>redacted</code>, or leave Replace with empty to remove the matching text. To omit the entire field, choose Output → omit. Merge can reference only earlier fields. Regex runs only in the controller's bounded RE2 engine.</p></details>`;}
  function renderKVEditRows(edits) {
    return (Array.isArray(edits)?edits:[]).map((edit,index)=>`<div class="nfi-parameter-grid" data-kv-edit-row="${index}">
      <label>Output key<input data-kv-edit-field="key" maxlength="128" value="${esc(edit?.key||'')}" placeholder="reason"></label>
      <label>Action<select data-kv-edit-field="action" aria-label="Key edit action">${['omit','blank','replace'].map(action=>qrwOption(action,edit?.action||'omit')).join('')}</select></label>
      ${edit?.action==='replace'?`<label>Replacement text<input data-kv-edit-field="value" maxlength="4096" value="${esc(edit.value??'')}" placeholder="redacted"></label>`:''}
      <button type="button" class="danger small" data-kv-edit-command="remove" aria-label="Remove key edit ${index+1}">Remove rule</button>
    </div>`).join('');
  }
  function renderKVGrammar(field) {
    const profile = field.kv_profile || '';
    const operators = field.assignment_separators?.length ? field.assignment_separators : [field.kv_separator || '='];
    return `<label>KV grammar<select data-qrw-key="kv_profile">
      <option value="" ${profile===''?'selected':''}>Saved grammar (legacy)</option>
      <option value="auto" ${profile==='auto'?'selected':''}>AutoSense</option>
      <option value="configured" ${profile==='configured'?'selected':''}>Configured separators</option>
    </select></label>
    <label>Assignment operators<input data-qrw-key="assignment_separators" value="${esc(JSON.stringify(operators))}" placeholder='[":"]'></label>
    ${isKVTail(field)?renderKVInputTools(field):''}
    <p class="hint">Pipe example <code>|status:failed|reason:notneeded|server:cloud</code> uses assignment <code>:</code> and pair separator <code>|</code>. Choose AutoSense or inspect the sample to compare grammars.</p>
    ${['kv_all','kv_eol'].includes(field.method)?`<details><summary>Edit individual KV output fields</summary>
      <div data-kv-edit-rows>${renderKVEditRows(field.key_value_edits)}</div>
      <button type="button" class="secondary small" data-kv-edit-command="add">Add key edit</button>
      <p class="hint">Use the exact output key after aliases, prefix and suffix. Omit removes it, blank keeps it as an empty string, and replace sets literal text. An edit applies when the key is present.</p>
      <details><summary>Paste or edit rules as JSON</summary><label>Key edits (JSON list)<textarea rows="4" data-qrw-key="key_value_edits" placeholder='[{"key":"reason","action":"blank"}]'>${esc(field.key_value_edits?.length?pretty(field.key_value_edits):'')}</textarea></label></details>
      <p class="hint">Example: choose key <code>reason</code> and action <code>blank</code> to output <code>"reason": ""</code>. Preview the row to verify the result.</p></details>`:''}`;
  }
  function renderKVInputTools(field){
    const edits=field.kv_input_edits||[];
    return `<details class="nfi-extended" ${edits.length?'open':''}><summary>Replace and split KV input</summary>
      <p class="hint">Choose <b>raw</b> in Start position for the whole line, or enter a position to edit only that range. Replace uses literal text, including spaces, Unicode, brackets and symbols such as <code>#</code>, <code>\\</code> and <code>*</code>.</p>
      <div class="actions">
        <button type="button" class="secondary small" data-action="kv-bracket-preset" data-id="${esc(field.id)}">Set up adjacent [key:value] pairs</button>
        <button type="button" class="secondary small" data-action="kv-input-add" data-id="${esc(field.id)}">Add literal replacement</button>
      </div>
      <p class="hint"><code>[status:failed][reason;notneeded][server=cloud]</code> becomes <code>status:failed|reason;notneeded|server=cloud</code>. For other layouts, enter the text to split on below.</p>
      <label>Split into pairs on (literal separators)<input data-qrw-key="kv_pair_separators" value="${esc(field.kv_pair_separators?.length?JSON.stringify(field.kv_pair_separators):'')}" placeholder='["|", "||"]'></label>
      ${edits.map((edit,index)=>`<div class="nfi-parameter-grid" data-kv-input-index="${index}">
        <label>Edit type<select data-kv-input-key="action"><option value="replace" ${edit.action==='replace'?'selected':''}>Replace all occurrences</option><option value="strip_prefix" ${edit.action==='strip_prefix'?'selected':''}>Remove from start</option><option value="strip_suffix" ${edit.action==='strip_suffix'?'selected':''}>Remove from end</option></select></label>
        <label>Find literal text<input aria-label="Find literal text" data-kv-input-key="find" value="${esc(edit.find)}" maxlength="1024"></label>
        ${edit.action==='replace'?`<label>Replace with (blank removes it)<input aria-label="Replace with" data-kv-input-key="replace" value="${esc(edit.replace||'')}" maxlength="1024"></label>`:''}
        <label>Require a match<input type="checkbox" data-kv-input-key="required" ${edit.required?'checked':''}></label>
        <button type="button" class="danger small" data-action="kv-input-remove" data-id="${esc(field.id)}" data-index="${index}">Remove edit</button>
      </div>`).join('')}
      <p class="hint">The controller previews the edited range and validates all pairs. To rename a key in the output, use Key Aliases. To change a key’s value, use Edit individual KV output fields.</p>
    </details>`;
  }
  function kvInspectorIsCurrent(snapshot) {
    return snapshot?.signature === JSON.stringify({raw:qrwRaw(),plan:qrwPlan()})
      && snapshot.host === el('kvie-inspector')
      && snapshot.host.isConnected;
  }
  function renderKVInspector(host, snapshot) {
    const profiles = snapshot.result.profiles || [];
    const profileCards = profiles.map((profile,index) => {
      const values = profile.observations || [];
      const observations = values.slice(0,24).map(pair=>`<li><code>${esc(pair.key)} = ${esc(pair.value === '' ? '(empty)' : pair.value)}</code> <small>bytes ${esc(Number(pair.start)+1)}–${esc(pair.end)}</small></li>`).join('');
      return `<article class="nfi-kv-profile">
        <h5>${esc(profile.family || profile.name || `Grammar ${index+1}`)}</h5>
        <p>Assignments <code>${esc(JSON.stringify(profile.assignment_separators))}</code> · Pair separators <code>${esc(JSON.stringify(profile.pair_delimiters))}</code> · ${values.length} observed pair(s)</p>
        ${profile.kv_input_edits?.length?'<p class="hint">This grammar includes literal edits to remove the observed wrappers. Apply it and check the extracted keys in Preview.</p>':''}
        <ul>${observations}</ul>
        <button type="button" class="secondary small" data-action="kv-apply-profile" data-index="${index}">Apply grammar ${index+1}</button>
      </article>`;
    }).join('');
    host.innerHTML = `<h4>KV AutoSense and Failure Inspector</h4>
      <p role="status">${esc(snapshot.result.diagnostic || (profiles.length?'Compare the observed pairs. Applying a grammar sets its separators and lexical defaults for this draft.':'No complete grammar matched. Review the source range or configure assignments and pair separators.'))}</p>
      ${snapshot.result.fragment? `<details><summary>Selected source fragment</summary><pre>${esc(snapshot.result.fragment)}</pre></details>` : ''}
      ${profileCards}
      <p class="hint">Use quotes around values containing separator-like text. Example <code>reason="retry; still queued"</code>.</p>`;
    nfiDecorate();
    applyPermissions();
  }
  async function inspectKVField(field) {
    if(!isKVTail(field))throw new Error('Select a key/value tail to inspect.');
    if(field.kv_input_edits?.length)return qrwVerifyController();
    qrwValidateInputs();
    const raw=qrwRaw(),plan=qrwPlan(),host=el('kvie-inspector');
    if(!raw.trim()||!host)throw new Error('Paste or inspect a raw record before using KV AutoSense.');
    const selected=plan.fields[state.qrwFields.indexOf(field)];
    const snapshot={host,fieldId:field.id,signature:JSON.stringify({raw,plan})};
    kvInspectorState=snapshot;
    host.hidden=false;
    host.innerHTML='<h4>KV AutoSense and Failure Inspector</h4><p role="status">Checking the selected source range…</p>';
    host.scrollIntoView({block:'nearest'});
    try {
      snapshot.result=await nfiAnalysisRequest('/assurance/kv-inspect',{raw,field:selected});
    } catch(error) {
      if(!kvInspectorIsCurrent(snapshot)||kvInspectorState!==snapshot)return;
      const cancelled=error.name==='AbortError'||/abort|cancel/i.test(error.message);
      snapshot.result={profiles:[],diagnostic:cancelled?'Inspection cancelled. Select Inspect KV grammar to retry.':`KV inspection unavailable. Select Inspect KV grammar to retry. ${error.message}`};
    }
    if(kvInspectorState===snapshot&&kvInspectorIsCurrent(snapshot))renderKVInspector(host,snapshot);
  }
  function applyKVGrammar(index) {
    const snapshot=kvInspectorState,profile=snapshot?.result?.profiles?.[index];
    if(!profile||!kvInspectorIsCurrent(snapshot))throw new Error('Inspect the current sample again before applying a grammar.');
    const field=state.qrwFields.find(item=>item.id===snapshot.fieldId);
    if(!field)throw new Error('The inspected field is unavailable.');
    const observed=new Set((profile.observations||[]).map(pair=>pair.key));
    // Explicit named mappings own their source keys. Keep their renames and types.
    const exclusions=field.output_policy==='flatten'?state.qrwFields.filter(item=>item.method==='key_value'&&observed.has(item.source)).map(item=>item.source):[];
    Object.assign(field,{kv_profile:'configured',assignment_separators:profile.assignment_separators,pair_delimiters:profile.pair_delimiters,
      key_exclude:[...new Set([...(field.key_exclude||[]),...exclusions])]});
    if(profile.kv_input_edits?.length)Object.assign(field,{kv_input_edits:profile.kv_input_edits,
      kv_pair_separators:[profile.pair_delimiters],require_complete:true});
    for(const key of ['kv_pair_separators','kv_quotes','kv_escape','kv_brackets','kv_key_pattern','kv_reject_empty'])delete field[key];
    el('qrw-field-rows').innerHTML=renderQRWFieldRows();
    nfiScheduleLive();
    const status=el('qrw-field-status');
    if(status)status.textContent=exclusions.length?`Grammar applied. Named mappings retain ${exclusions.join(', ')}. Preview is running.`:'Grammar applied. Preview is running.';
  }
  function renderQRWFieldRows(){return (state.qrwFields||[]).map((field,index)=>`<tr data-qrw-row="${esc(field.id)}"><td><span class="drag-handle">${index+1}</span></td><td><input data-qrw-key="name" value="${esc(field.name)}" aria-label="Desired JSON field name ${index+1}"><input data-qrw-key="label" value="${esc(field.label)}" aria-label="Display label ${index+1}"><details><summary>Description and example</summary><input data-qrw-key="description" value="${esc(field.description||'')}" aria-label="Description ${index+1}"><input data-qrw-key="example" value="${esc(field.example||'')}" aria-label="Example ${index+1}"></details></td><td><select data-qrw-key="type" aria-label="Data type ${index+1}">${QRW_TYPES.map(value=>qrwOption(value,field.type)).join('')}</select><select data-qrw-key="policy" aria-label="Presence policy ${index+1}">${QRW_POLICIES.map(value=>qrwOption(value,field.policy)).join('')}</select><select data-qrw-key="null_policy" aria-label="Null and missing policy ${index+1}">${['omit','preserve_null','reject'].map(value=>qrwOption(value,field.null_policy)).join('')}</select></td><td><select data-qrw-key="method" aria-label="Extraction method ${index+1}">${QRW_METHODS.map(value=>qrwOption(value,field.method)).join('')}</select><input data-qrw-key="source" value="${esc(field.source)}" aria-label="Source token key or group ${index+1}" placeholder="${field.method==='derived'?'earlier_field,other_field':field.method==='delimiter'?'token number or raw':field.method==='json_path'?'$.field[0].name':field.method==='regex'?'RE2 pattern':field.method==='key_value'?'key name':'1-based start position'}">${['position','character','byte','line','word','token','kv_all','kv_eol'].includes(field.method)?`<input data-qrw-key="source_end" value="${esc(field.source_end||'')}" aria-label="Inclusive end token ${index+1}" placeholder="inclusive end (optional)"><small>${field.method==='position'?'Positions are 1-based; end token is inclusive.':'Unicode characters, bytes, and lines use 1-based inclusive ranges.'}</small>`:''}${['delimiter','regex','regex_all','regex_groups'].includes(field.method)?`<div class="inline-fields"><input data-qrw-key="delimiter" value="${esc(field.delimiter)}" aria-label="Delimiter ${index+1}" maxlength="32"><input data-qrw-key="part" value="${esc(field.part)}" aria-label="Delimiter part ${index+1}" inputmode="numeric"></div>`:''}${field.method==='delimiter'?`<label class="confirm-mapping"><input type="checkbox" data-qrw-key="quote_aware" ${field.quote_aware===false?'':'checked'}> Respect quoted delimiters</label><label class="confirm-mapping"><input type="checkbox" data-qrw-key="confirmed" ${field.confirmed?'checked':''}> Confirm delimiter interpretation</label>`:''}${['key_value','kv_all','kv_eol'].includes(field.method)?`<small>Quoted and nested values are protected automatically; empty pair values are preserved.</small>${renderKVGrammar(field)}<label>Pair separators<input data-qrw-key="pair_delimiters" value="${esc((field.pair_delimiters??':;,').replaceAll('\t','\\t').replaceAll('\r','\\r').replaceAll('\n','\\n'))}" maxlength="32"></label><label>Duplicate keys<select data-qrw-key="duplicate_policy">${['reject','first','last','array'].map(value=>qrwOption(value,field.duplicate_policy)).join('')}</select></label>`:''}${['kv_all','kv_eol'].includes(field.method)?'<p class="hint" title="A start position of 9 extracts every key/value after the eighth whitespace token through EOL when Key value EOL is selected. Advanced All key values can explicitly use a multiline range. Flatten emits keys at the JSON root; include emits an object under the field name. Complete coverage rejects unparsed text and duplicate output keys.">ⓘ Dynamic KV: start position → EOL (line ending). Set unit, assignment separators and flatten in Advanced parameters. Example: status=failed:reason=notneeded → status, reason.</p>':''}${field.method==='derived'?`<label>Join separator<input data-qrw-key="join_separator" value="${esc(field.join_separator??' ')}"></label>`:''}${renderFieldSenseParameters(field)}${renderQRWOperations(field)}</td><td><select data-qrw-key="transform" aria-label="Transformation ${index+1}">${['none','trim','lowercase','uppercase'].map(value=>qrwOption(value,field.transform)).join('')}</select><label>Output<select data-qrw-key="output_policy" aria-label="Output policy ${index+1}" title="Include the field, omit it from JSON, or use it only in a later mapping. KV rows can also flatten their keys.">${(isKVTail(field)?['include','flatten','omit','intermediate']:['include','omit','intermediate']).map(value=>qrwOption(value,field.output_policy||'include')).join('')}</select></label><details class="nfi-field-options"><summary>Defaults, timestamps and validation</summary><input data-qrw-key="timestamp_format" value="${esc(field.timestamp_format)}" aria-label="Timestamp format ${index+1}" placeholder="Go time layout / epoch format"><select data-qrw-key="timezone" aria-label="Timezone policy ${index+1}">${['preserve','utc','configured'].map(value=>qrwOption(value,field.timezone)).join('')}</select><input data-qrw-key="timezone_name" value="${esc(field.timezone_name||'')}" aria-label="Configured IANA timezone ${index+1}" placeholder="IANA zone, e.g. Asia/Kolkata"><input data-qrw-key="default_value" value="${esc(field.default_value)}" aria-label="Explicit default ${index+1}" placeholder="explicit default or constant only"><input data-qrw-key="validation" value="${esc(field.validation)}" aria-label="Validation rule ${index+1}" placeholder="min=0,max=65535,one_of=a|b"></details></td><td><code data-qrw-preview>${esc(field.preview===''&&field.status==='mapped'?'(empty string)':field.preview||'Unresolved')}</code><small data-qrw-evidence>${esc(field.evidence)}</small><span class="mapping-status" data-qrw-status>${badge(field.status)} ${esc(field.confidence)}</span></td><td>${isKVTail(field)?`<button type="button" class="secondary small" data-action="kv-inspect" data-id="${esc(field.id)}">Inspect KV grammar</button>`:''}<div class="nfi-suggestion-actions"><button type="button" class="secondary small" data-action="nfi-suggestion-accept" data-id="${esc(field.id)}">${field.accepted?'Accepted':'Accept suggestion'}</button><button type="button" class="secondary small" data-action="nfi-suggestion-edit" data-id="${esc(field.id)}">Edit</button><button type="button" class="secondary small" data-action="nfi-suggestion-regenerate" data-id="${esc(field.id)}">Regenerate</button><button type="button" class="secondary small" data-action="qrw-field-delete" data-id="${esc(field.id)}">Reject suggestion</button></div><button type="button" class="secondary small" data-action="qrw-field-preview" data-id="${esc(field.id)}">Preview</button> <button type="button" class="secondary small" data-action="qrw-field-duplicate" data-id="${esc(field.id)}">Duplicate</button> <button type="button" class="secondary small" data-action="qrw-field-up" data-id="${esc(field.id)}" ${index===0?'disabled':''}>↑</button> <button type="button" class="secondary small" data-action="qrw-field-down" data-id="${esc(field.id)}" ${index===state.qrwFields.length-1?'disabled':''}>↓</button> <button type="button" class="danger small" data-action="qrw-field-delete" data-id="${esc(field.id)}">Delete</button></td></tr>`).join('');}
  function renderQRWFieldEditor(){const standalone=state.page==='field-mapping';return `<section class="panel qrw-field-editor" data-qrw-panel="3 4">${renderFieldSenseGuide()}<section class="nfi-live-panel"><h4>Live JSON and source trace</h4><p id="nfi-live-state" role="status">Preview values here as you edit. Generate + validate checks the complete parser before review.</p><pre id="nfi-live-json">{}</pre><div id="nfi-live-trace"></div></section><dialog id="nfi-help-dialog"><div id="nfi-help-content"></div><button type="button" class="secondary" data-action="nfi-help-close">Close help</button></dialog><section id="kvie-inspector" class="nfi-kv-inspector" hidden></section><div class="section-head"><div><h3>${standalone?'FieldSense mapping workbench':'Steps 3–4 · Desired JSON fields and value mapping'}</h3><p>Choose source positions, split compound tokens, extract key/value pairs, and apply ordered operations. Every generated candidate carries the same native mapping plan into production.</p></div><div class="actions"><button type="button" class="secondary" data-action="qrw-suggest-fields">Suggest from raw log</button><button type="button" class="secondary" data-action="workspace-ai-auto" data-artifact="mapping" data-tab="Field mapping" data-suggest-mapping>Suggest Mappings with Nexora AI</button><button type="button" class="secondary" data-action="${standalone?'nfi-generate':'gprs-open-editor'}">${standalone?'Generate candidate':'5 · Auto-populate complete parser'}</button><button type="button" class="secondary" data-action="qrw-add-field">Add Custom Field</button><button type="button" class="secondary" data-action="block-from-mapping">Build with manual field controls</button><button type="button" class="secondary" data-action="workspace-ai-open" data-mode="fix" data-tab="Field mapping">Fix with Nexora AI</button><button type="button" class="secondary" data-action="qrw-add-kv-tail" title="Extract every key/value from a chosen 1-based position to EOL, including future key names. Choose assignment and pair separators, or inspect the existing tail.">Add dynamic KV tail</button><button type="button" class="secondary" data-action="qrw-add-advanced-kv-tail" title="Use explicit pair boundaries and optionally preserve segments that have no assignment">Add advanced dynamic KV tail</button><button type="button" class="primary" data-action="qrw-generate-validate" title="Generate one complete candidate from every mapped static field and configured key/value extractor, run native mapping and full parser validation, and keep saving/review inside this workspace.">Generate + validate complete parser pack</button><button type="button" class="secondary" data-action="qrw-test-fields">Test all fields</button><button type="button" class="primary" data-action="qrw-controller-preview">Preview mapping</button><button type="button" class="secondary" data-action="nfi-cancel-analysis">Cancel analysis</button><button type="button" class="secondary" data-action="qrw-save-draft">Save mapping draft</button><button type="button" class="secondary" data-action="qrw-export-plan">Export mapping</button></div></div><details class="nfi-scope"><summary>Parser service scope and safe matching</summary><p>A reviewed service-scoped NFI parser overrides matching built-in formats only for that service. Wildcard mappings remain fallback parsers unless an operator explicitly approves a global override with a bounded RE2 match condition.</p><label>Service scope · auto or comma-separated services<input id="nfi-service" value="${esc(state.nfiService||state.quarantineDiagnosis?.record?.service||'auto')}" placeholder="auto, smtp, imapserv, or a custom service"></label><label>Mapping priority · 0–1000<input id="nfi-mapping-priority" type="number" min="0" max="1000" value="${Number(state.nfiMappingPriority)||0}"></label><label>Optional RE2 record match condition<input id="nfi-scope-pattern" value="${esc(state.nfiScopePattern||'')}" maxlength="1024" placeholder="e.g. ^[0-9]{8} .* smtp "></label><label><input id="nfi-global-scope" type="checkbox" ${state.nfiGlobalScope?'checked':''}> Allow reviewed global override (requires match pattern)</label></details>
<details id="nfi-runtime-evidence" class="nfi-import" hidden><summary>Automatic parser pool and observed field evidence</summary><p>Approved built-in and published add-on parsers are considered together. Suggestions do not activate or replace a parser. Only one selected parsing result is emitted for each input occurrence.</p><button class="secondary small" type="button" data-action="nfi-use-runtime-fields" hidden>Add observed fields from the existing parser</button><p class="hint">Source-byte suggestions describe this sample. Validate their ranges against representative records before approval.</p><pre id="nfi-runtime-evidence-json" class="json-output"></pre></details><div id="nfi-inspector-host">${qrwRawInspector(state.page==='field-mapping'?(state.nfiRaw||''):(state.lastAssurance?.request?.raw||qrwRaw()))}</div><details class="nfi-import"><summary>Import or edit a mapping plan</summary><label>Mapping plan JSON<textarea id="nfi-plan-import" rows="5" placeholder='{"version":1,"fields":[]}'></textarea></label><button class="secondary" type="button" data-action="qrw-import-plan">Import mapping</button></details><div class="table-wrap"><table class="qrw-field-table"><thead><tr><th>#</th><th>Name / label</th><th>Type / policy</th><th>Extraction and ordered operations</th><th>Transform / validation</th><th>Preview / evidence</th><th>Actions</th></tr></thead><tbody id="qrw-field-rows">${renderQRWFieldRows()||'<tr><td colspan="7"><div class="empty-state">Inspect or paste raw evidence, then suggest mappings or select individual tokens.</div></td></tr>'}</tbody></table></div><div id="qrw-field-status" role="status" class="callout">Local values are provisional until verified against the controller runtime. No parser is activated here.</div><pre id="nfi-controller-result" class="json-output" hidden></pre><details class="nfi-import" id="nfi-ai-help"><summary>Optional configured AI help · review only</summary><p>The controller masks failed log evidence and sensitive expected values before an explicitly requested AI review. Native parsing and mapping remain first. Provider keys never enter this browser or parser bundles.</p><div class="actions"><button type="button" class="secondary" data-action="nfi-ai-status">Check configured Nexora AI readiness</button><button type="button" class="secondary" data-action="nfi-ai-help">Ask configured Nexora AI for review-only help</button></div><div id="nfi-ai-status" class="json-output" role="status"></div><div id="nfi-ai-proposals">${state.aiProposals?.length?renderAIProposalQueue(state.aiProposals,state.lastAssurance?.result):''}</div></details>${standalone?'':`<div class="wizard-navigation"><button type="button" class="secondary qrw-back-button" data-action="qrw-navigate" data-step="2">2 · Back to evidence</button><button type="button" class="primary qrw-next-button" data-action="qrw-navigate" data-step="5">5 · Generate parser candidate</button></div>`}</section>`;}

  function setupQRWStepPanels(root=page){
    const current=String(Math.max(1,Math.min(9,Number(state.qrwStep||1))));
    const validatedCandidate=qrwCandidateValidationPassed(gprsLinkedCandidate());
    root.querySelectorAll('[data-qrw-panel]').forEach(panel=>{panel.hidden=!String(panel.dataset.qrwPanel||'').split(/\s+/).includes(current);});
    const localCompletion=[
      !!(state.gprsManual||state.quarantineDiagnosis),
      (!!state.gprsManual||!!state.quarantineDiagnosis?.sha256||state.lastAssurance?.request?.source===`quarantine:${state.quarantineDiagnosis?.id||''}`)||null,
      state.qrwFields.length?true:null,
      state.qrwFields.length?state.qrwFields.every(field=>!['unresolved','invalid','needs_confirmation','preview_pending'].includes(field.status)):null,
      !!root.querySelector('#gprs-parser-helper-form [name=parser]')?.value?.trim()&&state.gprsSamples.length>0||null,
      validatedCandidate?true:state.gprsPreview?(state.gprsPreview.preview?.passed===true||state.gprsPreview.passed===true):null,
      null,
      null,
      null
    ];
    const items=[...root.querySelectorAll('[data-wizard-step]')];
    const completeAt=index=>localCompletion[index]??items[index]?.dataset.stepComplete==='true';
    const dynamicLocks=[false,!completeAt(0),!completeAt(1),!completeAt(2),!completeAt(3),!(completeAt(4)||state.gprsSamples.length>0),!completeAt(5)];
    items.forEach((item,index)=>{
      if(index<dynamicLocks.length){item.dataset.stepLocked=String(dynamicLocks[index]);item.classList.toggle('locked',dynamicLocks[index]);const button=item.querySelector('button');if(button)button.disabled=dynamicLocks[index];}
    });
    const nextRequired=items.findIndex((item,index)=>!completeAt(index)&&item.dataset.stepLocked!=='true');
    items.forEach((item,index)=>{
      const active=item.dataset.wizardStep===current,complete=localCompletion[index]??item.dataset.stepComplete==='true',button=item.querySelector('button'),label=item.querySelector('.qrw-step-state');
      item.classList.toggle('active',active);item.classList.toggle('done',complete);item.classList.toggle('next-required',!active&&!complete&&index===nextRequired);item.dataset.stepComplete=String(complete);
      button?.setAttribute('aria-current',active?'step':'false');
      if(label)label.textContent=complete?'Complete':active?'Current · do this now':item.dataset.stepLocked==='true'?'Locked':index===nextRequired?'Next required':'Waiting';
    });
    const actions=[
      ['Select a retained record or paste a new log','[data-action=quarantine-diagnose], [data-action=gprs-use-manual]'],
      ['Inspect the source, then define output fields','.qrw-next-button'],
      ['Suggest fields or add your own','[data-action=qrw-suggest-fields]'],
      ['Preview the mapping, then generate the complete parser','[data-action=qrw-controller-preview]'],
      ['Validate the complete parser and save a draft','[data-action=qrw-generate-validate], button[type=submit]'],
      ['Test the candidate against selected evidence','[data-action=gprs-preview]'],
      ['Review, approve and publish the exact candidate','[data-action=assurance-parser-action][data-value=submit], [data-action=assurance-parser-action][data-value=approve], [data-action=assurance-parser-action][data-value=publish]'],
      [state.gprsManual?'Select a client and verify bundle delivery':'Confirm delivery to the origin client, then request replay','[data-action=parser-sync-rollout-propose], button[type=submit]'],
      [state.gprsManual?'Verify the manual sample through the live parser':'Refresh to verify the replay result','[data-action=gprs-verify-manual], [data-action=reload]']
    ];
    const actionHint=actions[Number(current)-1];
    if(items.length&&actionHint){
      const hint=root.querySelector('#qrw-step-hint');
      if(hint)hint.textContent=`Step ${current} · ${actionHint[0]}`;
      root.querySelectorAll('[data-qrw-action-step]').forEach(button=>delete button.dataset.qrwActionStep);
      for(const panel of root.querySelectorAll('[data-qrw-panel]:not([hidden])')){
        const button=[...panel.querySelectorAll(actionHint[1])].find(control=>!control.disabled&&!control.closest('[hidden]'));
        if(button)button.dataset.qrwActionStep=current;
      }
    }
  }
  function qrwSlice(value,start,end,unit='character'){
    const values=unit==='byte'?[...new TextEncoder().encode(String(value))]:unit==='line'?String(value).split(/\r?\n/):unit==='position'?String(value).trim().split(/\s+/u).filter(Boolean):[...String(value)];
    const first=Number(start),last=end===''||end==null||Number(end)===0?values.length:Number(end);if(!Number.isInteger(first)||!Number.isInteger(last)||first<1||last<first||last>values.length)throw new Error(`The ${unit} range must use existing 1-based inclusive positions (1–${values.length}).`);
    const selected=values.slice(first-1,last);if(unit==='byte'){try{return new TextDecoder('utf-8',{fatal:true}).decode(new Uint8Array(selected));}catch{throw new Error('Byte range cuts a UTF-8 character. Use Unicode character positions.');}}return selected.join(unit==='position'?' ':unit==='line'?'\n':'');
  }
  function qrwSelectPairs(raw,key,field){const matches=qrwPairs(raw,field.kv_separator||'=',field.pair_delimiters??':;,').filter(pair=>pair.key===key);if(matches.length>1&&(!field.duplicate_policy||field.duplicate_policy==='reject'))throw new Error(`Duplicate source key ${key}; choose first, last or array explicitly.`);return field.duplicate_policy==='array'?matches.map(pair=>pair.value):field.duplicate_policy==='last'?matches.at(-1)?.value:matches[0]?.value;}
  function qrwJSONPath(raw,path){if(String(path).length>1024||/(?:__proto__|prototype|constructor)/.test(path))throw new Error('Unsafe or excessive JSON path.');const normalized=String(path).replace(/^\$\.?/,'').replace(/\[(\d+)\]/g,'.$1');if(!/^[A-Za-z0-9_.-]*$/.test(normalized))throw new Error('Use a simple dot path and numeric array indexes.');let value=JSON.parse(String(raw));for(const key of normalized.split('.').filter(Boolean)){if(value==null||typeof value!=='object'||!Object.hasOwn(value,key))return undefined;value=value[key];}if(typeof value==='number'&&Number.isInteger(value)&&!Number.isSafeInteger(value))throw new Error('Select Preview here to preserve this large JSON number exactly.');if(typeof value==='number'&&Number.isInteger(value)&&!Number.isSafeInteger(value))throw new Error('Select Preview here to preserve this large JSON number exactly.');return value;}
  const QRW_LOCAL_OPERATIONS=new Set(['trim','lowercase','uppercase','split','select','substring','character','byte','line','replace','remove','prefix','suffix','add','strip_prefix','strip_suffix','join','merge','key_value']);
  function qrwApplyOperations(value,operations,output){if(!Array.isArray(operations)||operations.length>16)throw new Error('A field supports at most 16 ordered operations.');for(const operation of operations){switch(operation.op){case 'trim':value=String(value).trim();break;case 'lowercase':value=String(value).toLowerCase();break;case 'uppercase':value=String(value).toUpperCase();break;case 'split':{if(!operation.delimiter)throw new Error('Split needs a nonempty delimiter.');const parts=operation.quote_aware===false?String(value).split(operation.delimiter):qrwDelimited(String(value),operation.delimiter);value=Number(operation.part)>0?parts[Number(operation.part)-1]:parts;if(value===undefined)throw new Error('Split part is absent.');break;}case 'select':{if(!Array.isArray(value)||!Number.isInteger(Number(operation.part))||Number(operation.part)<1||Number(operation.part)>value.length)throw new Error('Select requires an existing 1-based array part.');value=value[Number(operation.part)-1];break;}case 'substring':case 'character':value=qrwSlice(value,operation.start||1,operation.end||0);break;case 'byte':case 'line':value=qrwSlice(value,operation.start||1,operation.end||0,operation.op);break;case 'replace':case 'remove':{if(!operation.find)throw new Error('Replace/remove requires a literal to find.');value=String(value).split(operation.find).join(operation.op==='remove'?'':operation.replace||'');break;}case 'prefix':value=String(operation.value||'')+String(value);break;case 'suffix':case 'add':value=String(value)+String(operation.value||'');break;case 'strip_prefix':if(String(value).startsWith(operation.value||''))value=String(value).slice(String(operation.value||'').length);break;case 'strip_suffix':if(operation.value&&String(value).endsWith(operation.value))value=String(value).slice(0,-operation.value.length);break;case 'join':case 'merge':{const values=operation.fields?.length?operation.fields.map(name=>{if(!Object.hasOwn(output,name))throw new Error(`Merged field ${name} must appear earlier.`);return output[name];}):Array.isArray(value)?value:[value];value=values.join(operation.separator??' ');break;}case 'key_value':value=qrwSelectPairs(String(value),operation.value,{kv_separator:operation.delimiter||'=',pair_delimiters:operation.separator??':;,',duplicate_policy:'reject'});if(value===undefined)return undefined;break;case 'regex_extract':case 'regex_replace':throw new Error('Select Preview here to run this RE2 operation.');default:throw new Error(QRW_OPERATIONS.includes(operation.op)?'Select Preview here to run this FieldSense operation.':`Unknown mapping operation ${operation.op}.`);}if(typeof value==='string'&&new TextEncoder().encode(value).length>1048576)throw new Error('Mapping output exceeds 1 MiB.');}return value;}
  function qrwExtract(field,raw,output={}){
    try{
      if(field.policy==='unresolved')return {error:'Choose a verified presence policy before testing this field.'};if(new TextEncoder().encode(String(raw)).length>1048576)return {error:'Raw input exceeds the 1 MiB mapping limit.'};
      if(isKVTail(field)||field.method==='key_value'&&(field.kv_profile||field.kv_start))return {pending:true,evidence:'Preview uses the shared KV engine.'};
      if(field.empty_policy&&field.empty_policy!=='preserve'||field.custom_type||field.custom_pattern||field.assignment_separators?.length||field.key_aliases||field.key_include?.length||field.key_exclude?.length||field.key_prefix||field.key_suffix||field.indices?.length||field.token_mode&&field.token_mode!=='whitespace')return {pending:true,evidence:'Preview uses the native FieldSense engine.'};
      const tokens=qrwTokens(raw);let value,evidence='';
      if(field.method==='raw'){value=raw;evidence='complete raw record';}else if(field.method==='field'){if(Object.hasOwn(output,field.source))value=output[field.source];evidence=`earlier field ${field.source}`;}else if(['word','token'].includes(field.method)){const start=Number(field.source),end=Number(field.source_end||field.source);if(!Number.isInteger(start)||start<1||end<start)return {error:'Word/token positions must be 1-based inclusive.'};value=tokens[start-1]&&tokens[end-1]?String(raw).slice(tokens[start-1].start,tokens[end-1].end):undefined;evidence=`${field.method} range ${start}–${end}`;}else if(field.method==='position'){const start=Number(field.source),end=field.source_end===''?start:Number(field.source_end);if(!Number.isInteger(start)||!Number.isInteger(end)||start<1||end<start||end>4096)return {error:'Position range must use 1-based token numbers with an inclusive end between start and 4096.'};if(start<=tokens.length&&end<=tokens.length)value=String(raw).slice(tokens[start-1].start,tokens[end-1].end);evidence=start===end?`token ${start}`:`tokens ${start}–${end} inclusive`;}
      else if(['character','byte','line'].includes(field.method)){const length=field.method==='byte'?new TextEncoder().encode(String(raw)).length:field.method==='line'?String(raw).split(/\r?\n/).length:[...String(raw)].length;const end=field.source_end||field.source;value=Number(field.source)>length||Number(end)>length?undefined:qrwSlice(raw,field.source,end,field.method);evidence=`${field.method} positions ${field.source}–${field.source_end||field.source}`;}
      else if(field.method==='delimiter'){if(!field.confirmed)return {pending:true,evidence:'Preview runs before you confirm this delimiter.'};const source=field.source==='raw'?String(raw):tokens[Number(field.source)-1]?.value;if(!field.delimiter)return {error:'Delimiter must not be empty.'};value=source==null?undefined:(field.quote_aware===false?source.split(field.delimiter):qrwDelimited(source,field.delimiter))[Number(field.part)-1];evidence=`${field.source==='raw'?'raw record':`token ${field.source}`}, delimiter ${JSON.stringify(field.delimiter)}, part ${field.part}`;}
      else if(field.method==='key_value'){value=qrwSelectPairs(raw,field.source,field);evidence=`key ${field.source}`;}
      else if(field.method==='regex')return {pending:true,evidence:'Waiting for the native RE2 preview.'};
      else if(field.method==='json_path'){value=qrwJSONPath(raw,field.source);evidence=`JSON path ${field.source}`;}
      else if(field.method==='constant'){value=field.default_value;evidence='explicit configured constant';}
      else if(field.method==='derived'){value=field.source.split(',').map(name=>{name=name.trim();if(!Object.hasOwn(output,name))throw new Error(`Derived source ${name} must be an earlier mapped field.`);return output[name];}).join(field.join_separator??' ');evidence='deterministic merge of earlier mapped fields';}
      else if(QRW_METHODS.includes(field.method))return {pending:true,evidence:'Waiting for the native FieldSense preview.'};
      else return {error:`Unknown extraction method ${field.method}.`};
      if(value==null){if(field.policy==='default'){value=field.default_value;evidence='explicit configured default';}else if(field.null_policy==='reject')return {error:'Null/missing policy rejects the absent source.'};else if(field.policy==='nullable'||field.null_policy==='preserve_null')return {value:null,evidence:'configured nullable absence'};else if(field.policy==='optional')return {absent:true,evidence:'optional field absent'};else return {error:'Required source value is absent.'};}
      if(field.operations?.some(operation=>QRW_OPERATIONS.includes(operation.op)&&!QRW_LOCAL_OPERATIONS.has(operation.op)))return {pending:true,evidence:'Waiting for the native field operation preview.'};
      value=qrwApplyOperations(value,field.operations||[],output);
      if(value===undefined){if(field.policy==='default'){value=field.default_value;evidence='explicit configured default after absent nested source';}else if(field.null_policy==='reject')return {error:'Null/missing policy rejects the absent nested source.'};else if(field.policy==='nullable'||field.null_policy==='preserve_null')return {value:null,evidence:'configured nullable nested absence'};else if(field.policy==='optional')return {absent:true,evidence:'optional nested field absent'};else return {error:'Required nested source value is absent.'};}
      if(field.transform==='trim')value=String(value).trim();if(field.transform==='lowercase')value=String(value).toLowerCase();if(field.transform==='uppercase')value=String(value).toUpperCase();
      if(['float','date','datetime','ipv6','email','url','uuid','null','custom'].includes(field.type))return {pending:true,evidence:'Waiting for the native type conversion preview.'};
      if(field.type==='integer'||field.type==='port'){if(!/^-?\d+$/.test(String(value)))return {error:`${JSON.stringify(value)} is not an integer.`};value=Number(value);if(!Number.isSafeInteger(value))return {error:'Integer exceeds the exact browser range. Keep identifiers as string.'};if(field.type==='port'&&(value<0||value>65535))return {error:'Port must be between 0 and 65535.'};}
      else if(field.type==='number'){if(!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$/.test(String(value)))return {error:'Value is not a decimal number.'};value=Number(value);if(!Number.isFinite(value))return {error:'Value is not a finite number.'};}
      else if(field.type==='boolean'){if(!/^(true|false)$/.test(String(value)))return {error:'Value is not a boolean.'};value=String(value)==='true';}
      else if(field.type==='timestamp'){if(field.timestamp_format)return {pending:true,evidence:'Waiting for the native timestamp preview.'};if(!/^\d{4}-\d{2}-\d{2}T/.test(String(value))||Number.isNaN(Date.parse(String(value))))return {error:'Timestamp needs an explicit input format or an ISO/RFC3339 value.'};if(field.timezone==='utc')value=new Date(value).toISOString();}
      else if(field.type==='ip'){const candidate=String(value),parts=candidate.split('.');if(!(parts.length===4&&parts.every(part=>/^(0|[1-9]\d{0,2})$/.test(part)&&Number(part)<=255)))return {error:'Verify IPv6 or unusual IP input on the controller.'};}
      else if(field.type==='object'||field.type==='array'){if(typeof value==='string')value=JSON.parse(value);if(field.type==='array'&&!Array.isArray(value))return {error:'Value is not an array.'};if(field.type==='object'&&(Array.isArray(value)||!value||typeof value!=='object'))return {error:'Value is not an object.'};}
      else value=typeof value==='string'?value:JSON.stringify(value);
      for(const rule of String(field.validation||'').split(',').map(item=>item.trim()).filter(Boolean)){const [kind,...rest]=rule.split('='),expected=rest.join('=');if(!expected)return {error:`Validation rule ${JSON.stringify(rule)} requires a value.`};if(['min','max','min_length','max_length'].includes(kind)&&!Number.isFinite(Number(expected)))return {error:`Validation rule ${JSON.stringify(rule)} requires a finite number.`};if(kind==='min'&&(typeof value!=='number'||value<Number(expected)))return {error:`Value is below minimum ${expected}.`};if(kind==='max'&&(typeof value!=='number'||value>Number(expected)))return {error:`Value exceeds maximum ${expected}.`};if(kind==='min_length'&&[...String(value)].length<Number(expected))return {error:`Value is shorter than ${expected}.`};if(kind==='max_length'&&[...String(value)].length>Number(expected))return {error:`Value is longer than ${expected}.`};if(kind==='one_of'&&!expected.split('|').includes(String(value)))return {error:'Value is outside the allowed set.'};if(!['min','max','min_length','max_length','one_of'].includes(kind))return {error:'Use bounded validation: min, max, min_length, max_length, or one_of.'};}
      return {value,evidence};
    }catch(error){return {error:error.message};}
  }
  function qrwConsumedNames(){return new Set(state.qrwFields.flatMap(field=>(field.operations||[]).filter(operation=>operation.remove_sources===true).flatMap(operation=>operation.fields||[])));}
  function qrwConditionalSources(){return new Set(state.qrwFields.flatMap(field=>(field.operations||[]).filter(operation=>operation.op==='conditional_merge'&&operation.remove_sources===true).flatMap(operation=>operation.fields||[])));}
  function qrwOutputFields(){const consumed=qrwConsumedNames(),actual=qrwControllerPreview?.passed===true&&qrwControllerPreview.signature===JSON.stringify({raw:qrwRaw(),plan:qrwPlan()})?qrwControllerPreview.actual:null;return state.qrwFields.filter(field=>!['omit','intermediate','flatten'].includes(field.output_policy)&&(!consumed.has(field.name)||qrwConditionalSources().has(field.name)||!actual||Object.hasOwn(actual,field.name)));}
  function qrwTestFields(preserveFocus=false){const raw=qrwRaw(),signature=JSON.stringify({raw,plan:qrwPlan()}),authoritative=qrwControllerPreview?.signature===signature&&qrwControllerPreview.passed===true,output=Object.assign(Object.create(null),authoritative?qrwControllerPreview.actual||{}:{}),errors=[],pending=[],names=new Set();for(const field of state.qrwFields){let result;if(!/^[A-Za-z_][A-Za-z0-9_]{0,63}$/.test(field.name)||field.name.startsWith('_')||QRW_RESERVED.has(field.name.toLowerCase()))result={error:'Use a non-reserved field name of 1–64 characters, starting with a letter.'};else if(names.has(field.name))result={error:`Duplicate JSON field name: ${field.name}`};else if(authoritative&&['kv_all','kv_eol'].includes(field.method)&&field.output_policy==='flatten')result={value:Object.fromEntries(Object.entries(qrwControllerPreview.actual||{}).filter(([key])=>!state.qrwFields.some(item=>item.name===key&&item.output_policy!=='flatten'))),flatten:true,evidence:'Dynamic keys verified at the JSON root; this container label is not emitted'};else if(authoritative)result=Object.hasOwn(qrwControllerPreview.actual||{},field.name)?{value:qrwControllerPreview.actual[field.name],evidence:'Verified by the native controller runtime'}:field.policy==='required'&&!['omit','intermediate'].includes(field.output_policy)&&!qrwConsumedNames().has(field.name)?{error:'Controller omitted this required output field'}:{absent:true,evidence:'Controller verified optional absence'};else result=qrwExtract(field,raw,output);names.add(field.name);if(result.pending){field.preview='';field.status='preview_pending';field.evidence=result.evidence;pending.push(field.name);}else if(result.error){field.preview='';field.status='invalid';field.evidence=result.error;errors.push(`${field.name}: ${result.error}`);}else{field.preview=result.absent?'(absent)':result.value===null?'null':typeof result.value==='string'?result.value:JSON.stringify(result.value);field.status=field.method==='delimiter'&&!field.confirmed?'needs_confirmation':result.absent&&!['kv_all','kv_eol'].includes(field.method)?'optional_absent':'mapped';field.evidence=result.evidence;if(!result.absent&&!result.flatten)output[field.name]=result.value;}}for(const field of state.qrwFields)if(['omit','intermediate'].includes(field.output_policy))delete output[field.name];const form=page.querySelector('#gprs-preview-form');if(form){form.elements.gprs_desired.value=pretty(output);form.elements.mapping.value=pretty(Object.fromEntries(state.qrwFields.filter(field=>field.status==='mapped').map(field=>[`mapping:${field.name}`,field.name])));}state.nfiDesired=output;const rows=el('qrw-field-rows');if(rows&&!preserveFocus)rows.innerHTML=renderQRWFieldRows();else if(rows)for(const field of state.qrwFields){const row=[...rows.children].find(row=>row.dataset.qrwRow===field.id),preview=row?.querySelector('[data-qrw-preview]');if(preview)preview.textContent=field.preview||field.evidence;
      const evidence=row?.querySelector('[data-qrw-evidence]');
      if(evidence)evidence.textContent=field.evidence;
      const rowStatus=row?.querySelector('[data-qrw-status]');
      if(rowStatus)rowStatus.innerHTML=`${badge(field.status)} ${esc(field.confidence)}`;
    }const status=el('qrw-field-status');if(status){status.classList.toggle('warn',!!errors.length);status.textContent=errors.length?`${errors.length} field mapping error(s): ${errors.join(' ')}`:pending.length?'Native preview pending. Your edits are ready to test.':`${Object.keys(output).length} fields mapped from actual evidence. Review the generated JSON before testing.${authoritative?' Verified by controller.':''}`;}const host=el('nfi-inspector-host');if(host)host.innerHTML=qrwRawInspector();return {output,errors,pending};}
  function qrwScope(){const service_scope=page.querySelector('#nfi-service')?.value||state.nfiService||state.quarantineDiagnosis?.record?.service||'auto',scope_pattern=page.querySelector('#nfi-scope-pattern')?.value||state.nfiScopePattern||'',reviewed_global_scope=page.querySelector('#nfi-global-scope')?.checked??state.nfiGlobalScope??false;if(new TextEncoder().encode(scope_pattern).length>1024)throw new Error('A service scope match condition is limited to 1024 UTF-8 bytes.');if(reviewed_global_scope&&!scope_pattern.trim())throw new Error('A reviewed global override needs a bounded RE2 match condition.');const mapping_priority=Number(page.querySelector('#nfi-mapping-priority')?.value??state.nfiMappingPriority??0);if(!Number.isInteger(mapping_priority)||mapping_priority<0||mapping_priority>1000)throw new Error('Mapping priority must be an integer from 0 to 1000.');return {service_scope,scope_pattern,reviewed_global_scope,mapping_priority};}
  function qrwBuildParser(preserveFocus=false,allowUnverified=false){qrwValidateInputs();const {errors,pending}=qrwTestFields(preserveFocus);if((errors.length||pending.length)&&!allowUnverified)throw new Error('Preview the current mapping and resolve any issues before generating a parser candidate.');if(!allowUnverified&&state.qrwFields.some(field=>field.method==='delimiter'&&!field.confirmed))throw new Error('Confirm the delimiter interpretation for the mapped fields before generating a parser candidate.');if(!state.qrwFields.length||state.qrwFields.length>128)throw new Error('A parser mapping must contain 1–128 verified fields.');const runtimeTypes={integer:'int',number:'float',float:'float',boolean:'bool',timestamp:'time',date:'time',datetime:'time',ipv6:'ip',port:'int',email:'string',url:'string',uuid:'string',null:'string',custom:'string'};return {pattern_type:'mapping',pattern:'.*',...qrwScope(),mapping_plan:qrwPlan(),fields:qrwOutputFields().map(field=>({name:field.name,source:`mapping:${field.name}`,type:runtimeTypes[field.type]||field.type,required:field.policy==='required'&&!qrwConditionalSources().has(field.name),validation:field.validation})),field_mapping:Object.fromEntries(qrwOutputFields().map(field=>[field.name,`mapping:${field.name}`])),output_schema:{version:1,fields:qrwOutputFields().map(({name,label,type,policy,validation,null_policy})=>({name,label,type,policy:qrwConditionalSources().has(name)?'optional':policy,validation,null_policy}))},provenance:{kind:'quarantine_resolution_wizard',subsystem:'NFI',record_id:state.quarantineDiagnosis?.id||'',generated_from_verified_evidence:!allowUnverified||(!errors.length&&!pending.length)},resource_limits:{max_input_bytes:1048576,timeout_ms:50}};}
  function qrwValidateInputs() {
    const invalid=page.querySelector('.qrw-field-editor [data-qrw-key]:invalid, .qrw-field-editor [data-qrw-op-key]:invalid');
    if(invalid)throw new Error(invalid.validationMessage||'Correct the highlighted configuration input.');
  }
  async function qrwPrepareFields() {
    qrwValidateInputs();
    const checked=qrwTestFields(true);
    if(checked.errors.length||checked.pending.length)await qrwVerifyController();
    return qrwTestFields();
  }
  async function qrwVerifyController({background=false}={}) {
    const editor=page.querySelector('.qrw-field-editor');
    qrwValidateInputs();
    const raw=qrwRaw(),plan=qrwPlan(),signature=JSON.stringify({raw,plan});
    if(!raw.trim())throw new Error('Paste or inspect a raw record before previewing the mapping.');
    const current=()=>editor?.isConnected&&editor===page.querySelector('.qrw-field-editor')
      && !editor.querySelector('[data-qrw-key]:invalid,[data-qrw-op-key]:invalid')
      && signature===JSON.stringify({raw:qrwRaw(),plan:qrwPlan()});
    if(!background){
      clearTimeout(nfiLiveTimer);
      qrwLivePreviewController?.abort();
    }
    const previewPlan={...plan,fields:plan.fields.map(field=>field.method==='delimiter'?{...field,confirmed:true}:field)};
    const controller=background?new AbortController():null;
    if(controller){
      qrwLivePreviewController?.abort();
      qrwLivePreviewController=controller;
    }
    const liveState=el('nfi-live-state');
    if(liveState)liveState.textContent='Previewing this sample with the native mapping engine…';
    let result;
    try{
      result=controller?await api('/assurance/mapping-preview',{method:'POST',body:{raw,plan:previewPlan},signal:controller.signal})
        :await nfiAnalysisRequest('/assurance/mapping-preview',{raw,plan:previewPlan});
    }catch(error){
      if(controller?.signal.aborted)return;
      if(!current()){
        if(background)return;
        throw new Error('The sample or mapping changed. Preview the current draft again.');
      }
      if(error.name==='AbortError'||/abort|cancel/i.test(error.message))throw error;
      if(liveState)liveState.textContent=`Preview unavailable. Your draft is still in the editor. Select Preview mapping to retry. ${error.message}`;
      if(background)return;
      throw error;
    }finally{
      if(qrwLivePreviewController===controller)qrwLivePreviewController=null;
    }
    if(controller?.signal.aborted||!current()){
      if(background)return;
      throw new Error('The sample or mapping changed. Preview the current draft again.');
    }
    qrwControllerPreview={...result,signature,actual:result.actual||result.values||{}};
    if(el('nfi-live-json'))el('nfi-live-json').textContent=result.passed===true?pretty(qrwControllerPreview.actual):'No verified JSON for this sample.';
    const diagnostic=el('nfi-controller-result');
    if(diagnostic){diagnostic.hidden=false;diagnostic.textContent=pretty(result);}
    if(result.passed!==true){
      const problems=(result.errors||[result.error||'See the native diagnostic below.']).map(item=>typeof item==='string'?item:JSON.stringify(item));
      const message=`Mapping needs attention. ${problems.join(' ')}${state.qrwFields.some(isKVTail)?' Inspect KV grammar to compare delimiters for this sample.':''}`;
      if(liveState)liveState.textContent=message;
      const mappingStatus=el('qrw-field-status');
      if(mappingStatus){mappingStatus.classList.add('warn');mappingStatus.textContent=message;}
      if(background)return result;
      throw new Error(message);
    }
    qrwTestFields(true);
    state.nfiPreviewed=true;
    if(el('nfi-live-json'))el('nfi-live-json').textContent=pretty(qrwControllerPreview.actual);
    if(liveState){
      const assessment=result.assessment;
      const coverage=typeof assessment?.source_evidence_pct==='number'
        ? ` Source evidence coverage: ${assessment.source_evidence_pct}% (${assessment.observed_fields}/${assessment.output_fields} output fields have a source span).`
        : '';
      const needsConfirmation=plan.fields.some(field=>field.method==='delimiter'&&!field.confirmed);
      liveState.textContent=`Mapping preview passed for this sample.${coverage} ${needsConfirmation?'Confirm the highlighted delimiter settings before generating the candidate.':'Generate + validate checks the complete parser against expected values.'}`;
    }
    if(el('nfi-live-trace'))el('nfi-live-trace').innerHTML=state.qrwFields.map(field=>`<button type="button" class="nfi-trace-field" data-action="nfi-highlight" data-id="${esc(field.id)}"><b>${esc(field.name)}</b> ← verified source</button><details><summary>Native extraction and intermediate operations</summary><pre>${esc(pretty({evidence:(result.evidence||result.fields)?.[field.name],trace:result.trace?.[field.name]}))}</pre></details>`).join('');
    if(page.querySelector('[data-wizard-step]'))setupQRWStepPanels();
    nfiGuideRefresh();
    return result;
  }

  async function qrwValidateCandidate(form,{recordWorkflow=false}={}){
    if(!form)throw new Error('Generated parser candidate is unavailable');
    const parser=parseJSON(form.elements.parser.value,'Parser JSON'),cases=parseJSON(form.elements.cases.value,'Validation cases JSON');
    if(!parser||typeof parser!=='object'||Array.isArray(parser)||!parser.pattern||!Array.isArray(parser.fields)||(!parser.fields.length&&!parser.mapping_plan?.fields?.some(field=>['kv_all','kv_eol'].includes(field.method)&&field.output_policy==='flatten')))throw new Error('Generate a parser with a pattern and at least one field before validation');
    if(!Array.isArray(cases)||!cases.length)throw new Error('Add at least one exact raw-log validation case');
    if(cases.some(item=>!item||typeof item.raw!=='string'||!item.raw.trim()||!item.desired||typeof item.desired!=='object'||Array.isArray(item.desired)||!Object.keys(item.desired).length))throw new Error('Every validation case needs exact raw evidence and a non-empty desired JSON object');
    const result=await api('/assurance/parser-preview',{method:'POST',body:{parser,cases}}),validation=result.validation||{};
    const status=form.querySelector('[data-parser-helper-status]')||el('parser-helper-status'),summary=`Candidate validation: ${number(validation.passed)}/${number(validation.total)} passed · ${number(validation.failed)} failed · ${number(validation.score)}%`;
    if(status){status.textContent=summary+(result.passed===true?'':' · '+pretty(validation.results||result.errors||result));status.classList.toggle('warn',result.passed!==true);}
    if(recordWorkflow){state.gprsPreview={records_tested:cases.length,parser_candidate:true,artifact_checksum:result.artifact_checksum,preview:result};state.lastAssurance={request:cases[0],result};state.qrwStep=6;const output=el('gprs-preview');if(output)output.textContent=pretty(state.gprsPreview);}
    return result;
  }
  function quarantineRecordStatus(record){return String(record?.status||record?.state||'waiting').toLowerCase();}
  function quarantineIsActive(record){return ['waiting','replay_failed'].includes(quarantineRecordStatus(record));}
  function quarantineIsActionable(record){return quarantineIsActive(record)&&record?.raw_available===true;}
  function quarantineCanInspect(record){
    return record?.raw_available===true&&(quarantineIsActive(record)||quarantineNeedsAttention(record));
  }
  function quarantineAttemptLabel(record){
    return record.replay_attempt_count==null?`${number(record.attempt_count)} reported attempts`:`${number(record.replay_attempt_count)} replay attempts · ${number(record.attempt_count)} parser attempts`;
  }
  function quarantineReplayExhausted(record){
    return !!record&&(['dead_letter','manual_attention'].includes(quarantineRecordStatus(record))||
      (quarantineIsActive(record)&&(record.replay_attempt_count==null?Number(record.attempt_count)>=5:Number(record.replay_attempt_count)>=4)));
  }
  function quarantineNeedsAttention(record){return quarantineReplayExhausted(record)||quarantineRecordStatus(record)==='replay_failed';}
  function qrwReplayEligibility(record, bundleHash){
    if(!record?.raw_available||Date.parse(record.raw_expires_at)>0&&Date.parse(record.raw_expires_at)<=Date.now())return {eligible:false,reason:'Exact source evidence is unavailable.'};
    if(!quarantineIsActive(record)&&!quarantineReplayExhausted(record))return {eligible:false,reason:`Record state: ${quarantineRecordStatus(record)}.`};
    if(!quarantineReplayExhausted(record)){
      if(Date.parse(record.retry_after)>Date.now())return {eligible:false,reason:`Retry backoff ends at ${time(record.retry_after)}.`};
      return {eligible:true,reason:''};
    }
    if(!record.recovery_replay_used&&record.last_replay_bundle_hash&&bundleHash&&record.last_replay_bundle_hash!==bundleHash){
      return {eligible:true,recovery:true,reason:'One recovery replay is available for this changed bundle.'};
    }
    return {eligible:false,reason:'The record reached its replay retry limit. Open Step 9 to see its outcome and history.'};
  }
  function quarantineAwaitsEvidence(record){return quarantineIsActive(record)&&record?.raw_available!==true;}
  function quarantineEvidenceHelp(node=''){return `Exact source evidence is not retained${node?` for ${node}`:''}. Enable bounded quarantine evidence retention on the origin client, then reproduce or re-read this event. It becomes inspectable automatically when exact evidence arrives.`;}
  function qrwExactAcknowledgement(asset,nodeId){
    const desiredHash=state.assuranceSyncBundle?.desired_hash||'',client=asset&&parserSyncClients(asset).find(item=>item.node_id===nodeId);
    return !!client&&/^[0-9a-f]{64}$/i.test(desiredHash)&&!client.drifted&&client.desired_hash===desiredHash&&client.applied_hash===desiredHash
      &&client.exact_acknowledgement!==false;
  }
  function qrwCandidateForRecord(recordId,{publishedOnly=false}={}){
    if(!recordId)return null;
    return (state.assuranceParsers||[]).filter(doc=>(doc.payload||{}).source_quarantine_record_id===recordId&&(!publishedOnly||doc.state==='published')).sort((a,b)=>Number(b.revision||0)-Number(a.revision||0)||String(b.updated_at||b.created_at||'').localeCompare(String(a.updated_at||a.created_at||'')))[0]||null;
  }
  function qrwCandidateValidationPassed(candidate){return candidate?.payload?.validation_passed===true||(candidate?.payload?.validation?.failed===0&&Number(candidate?.payload?.validation?.total)>0);}
  function gprsLinkedCandidate(){
    if(state.gprsManual)return state.assuranceParsers.find(doc=>doc.id===state.gprsManual.parserId)||null;
    return qrwCandidateForRecord(state.quarantineDiagnosis?.id);
  }
  function resetGPRSSource(){
    state.gprsManual=null;
    state.quarantineDiagnosis=null;
    state.gprsPreview=null;
    state.qrwReplayAcceptedIds=[];
    state.gprsSamples=[];
    state.qrwFields=[];
    state.nfiService='auto';
    state.nfiScopePattern='';
    state.nfiMappingPriority=0;
    state.nfiGlobalScope=false;
    qrwControllerPreview=null;
    qrwLivePreviewController?.abort();
    nfiAnalysisController?.abort();
    sessionStorage.removeItem('nli.qrw.selected-record');
    sessionStorage.removeItem('nli.qrw.mapping-draft');
  }
  function renderGPRSPanel(){
    const manual=state.gprsManual;
    const records=state.repairRecords||[],fixtures=state.assuranceFixtures||[],parsers=state.assuranceParsers||[];
    const selectable=records.filter(quarantineCanInspect);
    const originRecord=records.find(r=>(r.id||r.record_id)===state.quarantineDiagnosis?.id),originNode=manual?.nodeId||originRecord?.node_id||'';
    const candidateForRecord=qrwCandidateForRecord;
    const linkedCandidate=gprsLinkedCandidate();
    const canEmergencyApprove=state.user?.role==='admin';
    const linkedAsset=linkedCandidate&&(state.assuranceSyncAssets||[]).find(asset=>asset.asset_id===linkedCandidate.id),originClient=parserSyncClients(linkedAsset).find(client=>client.node_id===originNode);
    const originAcknowledged=qrwExactAcknowledgement(linkedAsset,originNode);
    const originResolved=originRecord?originRecord.status==='resolved':state.quarantineHistory.some(item=>(item.record_id||item.id)===state.quarantineDiagnosis?.id&&['resolved','success','succeeded'].includes(String(item.status||item.outcome||'').toLowerCase()));
    const candidateValidationPassed=qrwCandidateValidationPassed(linkedCandidate),tested=!!state.gprsPreview||candidateValidationPassed,previewPassed=state.gprsPreview?.preview?.passed===true||state.gprsPreview?.passed===true||candidateValidationPassed;
    const syncBundleHash=state.assuranceSyncBundle?.desired_hash||'',bundleAvailable=!!syncBundleHash&&(state.assuranceSyncAssets||[]).length>0;
    const eligibleFleetNodes=(state.assuranceFleetNodes||[]).filter(item=>{const node=item.node||item;return node.id&&node.status!=='offline';}),originRolloutAvailable=eligibleFleetNodes.some(item=>(item.node||item).id===originNode);
    const parserProposals=(state.parserRolloutProposals||[]).filter(isParserSyncProposal);
    const currentBundleProposals=parserProposals.filter(doc=>syncBundleHash&&rolloutBundleHash(doc)===syncBundleHash);
    const coversOrigin=doc=>[...(doc.payload?.target_nodes||[]),...(doc.payload?.node_ids||[])].includes(originNode);
    const liveRolloutFor=doc=>(state.parserLiveRollouts||[]).find(item=>item.id===doc.payload?.live_rollout_id);
    const proposalInProgress=doc=>{
      const job=liveRolloutFor(doc),terminal=['completed','complete','failed','cancelled','canceled','expired'];
      return ['pending_approval','approved','active'].includes(doc.state)&&!terminal.includes(job?.status)&&!terminal.includes(job?.node_results?.[originNode]?.status);
    };
    const originProposals=currentBundleProposals.filter(coversOrigin),liveBundleProposals=originProposals.filter(proposalInProgress),pendingBundleProposal=liveBundleProposals.find(doc=>doc.state==='pending_approval'),approvedBundleProposal=liveBundleProposals.find(doc=>doc.state==='approved'||doc.state==='active');
    const verifyOriginButton=(label)=>'<button class="primary" type="button" data-action="parser-sync-verify" data-node="'+esc(originNode)+'" data-bundle="'+esc(syncBundleHash)+'" '+(originNode&&bundleAvailable?'':'disabled')+'>'+label+'</button>';
    const replayObserved=!!originRecord&&(['reprocessing','resolved'].includes(originRecord.status)||state.qrwReplayAcceptedIds.includes(state.quarantineDiagnosis?.id)||state.quarantineHistory.some(item=>(item.record_id||item.id)===state.quarantineDiagnosis?.id&&['reprocess_requested','reprocessing','resolved','success','succeeded'].includes(String(item.status||item.outcome||item.action||'').toLowerCase())));
    const originExhausted=quarantineReplayExhausted(originRecord);
    const originEligibility=qrwReplayEligibility(originRecord,syncBundleHash);
    const manualVerified=!!(manual?.verification?.passed&&originAcknowledged&&linkedCandidate?.state==='published'&&manual.verification.bundleHash===syncBundleHash&&manual.verification.revision===linkedCandidate.revision);
    const verificationAvailable=manual?originAcknowledged&&linkedCandidate?.state==='published':replayObserved||originResolved||originExhausted||!!originRecord&&linkedCandidate?.state==='published';
    const runbook=(label,items,step)=>{const current=items.findIndex(item=>!item[1]);return `<ol class="qrw-runbook" aria-label="${esc(label)}">${items.map(([title,done,detail],index)=>`<li class="${done?'done':index===current?'current':'blocked'}"><span class="qrw-runbook-number">${step}.${index+1}</span><div><b>${esc(title)}</b><small>${esc(detail)}</small></div><span class="qrw-runbook-state">${done?'Done':index===current?'Do this now':'Waiting'}</span></li>`).join('')}</ol>`;};
    const approvedComment=linkedCandidate?.payload?.approval_reason||linkedCandidate?.payload?.emergency_approval_reason;
    const reviewRunbook=runbook('Candidate review and publication checklist',[['Candidate saved',!!linkedCandidate,linkedCandidate?`${linkedCandidate.id} revision ${linkedCandidate.revision||'—'}`:'Save the generated parser candidate'],['Deterministic test recorded',tested,tested?'Candidate preview evidence is available':'Run Step 6 before review'],['Submitted for review',!!linkedCandidate&&['in_review','approved','published'].includes(linkedCandidate.state),linkedCandidate?.state==='draft'?'Use Submit in the candidate row':'Controller lifecycle state'],['Approval recorded',!!linkedCandidate&&['approved','published'].includes(linkedCandidate.state),approvedComment||'Use independent approval, or the audited administrator approval when operationally required'],['Published to desired bundle',linkedCandidate?.state==='published',linkedCandidate?.state==='published'?`Artifact ${String(linkedCandidate.payload?.artifact_checksum||'').slice(0,20)}`:'Publish only after approval']],7);
    const deployRunbook=runbook('Deployment and replay checklist',[['Published bundle available',linkedCandidate?.state==='published'&&bundleAvailable,bundleAvailable?`Bundle ${syncBundleHash.slice(0,20)}`:'Publish and refresh the desired bundle'],['Origin rollout proposed',liveBundleProposals.length>0||originAcknowledged,originAcknowledged?'Exact delivery already acknowledged':liveBundleProposals.length?`${liveBundleProposals.length} active proposal(s) for this exact bundle`:'Propose to the diagnosed origin client'],['Rollout approval recorded',!!approvedBundleProposal||originAcknowledged,approvedBundleProposal?(approvedBundleProposal.payload?.approval_reason||approvedBundleProposal.payload?.emergency_approval_reason||'Approval recorded'):originAcknowledged?'Delivery already acknowledged':'Use independent approval, or audited administrator approval for the exact parser bundle'],['Origin client exact-hash acknowledgement',originAcknowledged,originAcknowledged?`${originNode} is aligned`:'Verify until applied hash equals desired hash'],['Targeted replay requested',replayObserved,replayObserved?(state.qrwReplayAcceptedIds.includes(state.quarantineDiagnosis?.id)?'Controller accepted the request; refresh Step 9 for the final result':'Controller replay state/history is present'):originExhausted?'Automatic attempts exhausted. Check the recovery status below.':'Select acknowledged origin records and request replay']],8);
    const verifyRunbook=runbook('Verification checklist',[['Replay request recorded',replayObserved,replayObserved?'Controller history contains the replay':originExhausted?'Automatic attempts exhausted. Check the controller history below.':'Return to Step 8 and request replay'],['Origin record resolved',originResolved,originResolved?'Quarantine state/history reports resolved':originExhausted?'The retained record is unresolved. A successful fresh ingest does not resolve a retained record.':'Refresh until the controller reports the final outcome'],['Parser remains recoverable',linkedCandidate?.state==='published',linkedCandidate?.state==='published'?'Rollback and offboard controls remain available':'Review the linked parser lifecycle']],9);
    const next=(title,detail,action='',complete=false)=>`<section class="qrw-next-action"><div><p class="eyebrow">${complete?'Completed':'Next required action'}</p><h3>${esc(title)}</h3><p>${esc(detail)}</p></div>${action?`<div class="actions">${action}</div>`:''}</section>`;
    let guide='';
    if(!manual&&!state.quarantineDiagnosis)guide=next('Choose a retained failure or paste a new log','Inspect retained evidence to repair a quarantined record, or paste a representative sample to onboard a new format.');
    else if(!state.qrwFields.length&&!linkedCandidate)guide=next('Define the output fields','Use evidence based suggestions or add fields, then review each extraction and run Test all fields. Suggestions need operator confirmation.');
    else if(!tested)guide=next('Run the deterministic preview','Review the generated JSON and run a preview against the selected retained record or saved fixtures. A preview never activates a parser.',`<button class="primary" type="button" data-action="gprs-preview">Run deterministic preview</button>`);
    else if(!linkedCandidate)guide=next('Create a reviewable parser candidate',previewPassed?'The preview passed. Create a candidate, then retest and submit it for independent review.':'The preview returned findings. Review them and adjust the mapping before you create and validate a candidate.',`<button class="primary" type="button" data-action="gprs-open-editor" ${state.gprsSamples.length?'':'disabled'}>Create parser candidate</button>`);
    else if(['draft','rejected'].includes(linkedCandidate.state))guide=next('Retest, then submit for independent review',`Candidate ${linkedCandidate.id} is ${linkedCandidate.state}. Retest its saved cases, review the result, then submit. A different authorized reviewer must approve it.`, `<button class="secondary" type="button" data-action="assurance-parser-action" data-id="${esc(linkedCandidate.id)}" data-value="validate">Retest candidate</button> <button class="primary" type="button" data-action="assurance-parser-action" data-id="${esc(linkedCandidate.id)}" data-value="submit">Submit for review</button>`);
    else if(linkedCandidate.state==='in_review'&&linkedCandidate.created_by===state.user?.username&&canEmergencyApprove)guide=next('Approve this exact candidate as administrator',`Candidate ${linkedCandidate.id} is in review. An administrator can approve this candidate, including their own work. Approval records a reason and binds to revision ${linkedCandidate.revision} and checksum ${String(linkedCandidate.payload?.artifact_checksum||'').slice(0,20)}.`, `<button class="warning" type="button" data-action="assurance-parser-action" data-id="${esc(linkedCandidate.id)}" data-value="approve">Approve exact revision</button>`);
    else if(linkedCandidate.state==='in_review'&&linkedCandidate.created_by===state.user?.username)guide=next('Wait for an authorized reviewer',`Candidate ${linkedCandidate.id} is in review. You created this candidate and do not hold the administrator approval rights.`);
    else if(linkedCandidate.state==='in_review')guide=next('Review the candidate independently',`Candidate ${linkedCandidate.id} is in review. Approval is recorded by the controller. Administrators can approve their own revision; other authors need an authorized reviewer.`, `<button class="secondary" type="button" data-action="assurance-parser-action" data-id="${esc(linkedCandidate.id)}" data-value="approve">Review and approve</button>`);
    else if(linkedCandidate.state==='approved'&&linkedCandidate.payload?.offboarded_at)guide=next('Review the parser offboard rollout proposal',`Candidate ${linkedCandidate.id} was intentionally removed from the active parser bundle. The associated proposal below must receive authorized approval before clients receive the updated bundle.`);
    else if(linkedCandidate.state==='approved'&&linkedCandidate.payload?.rolled_back_at)guide=next('Review the parser rollback rollout proposal',`Candidate ${linkedCandidate.id} was removed from the active parser bundle as rollback recovery. The associated proposal below must receive authorized approval before clients receive the updated bundle.`);
    else if(linkedCandidate.state==='approved')guide=next('Publish the approved candidate',`Candidate ${linkedCandidate.id} has authorized approval. Publishing makes its exact revision the desired parser bundle and starts the governed rollout.`, `<button class="primary" type="button" data-action="assurance-parser-action" data-id="${esc(linkedCandidate.id)}" data-value="publish">Publish approved revision</button>`);
    else if(linkedCandidate.state==='published'&&state.parserRolloutLoadError)guide=next('Restore rollout status before continuing',`The controller could not return rollout state: ${state.parserRolloutLoadError}. Refresh after the controller recovers. The wizard will not create a duplicate proposal while status is unknown.`,`<button class="primary" type="button" data-action="reload">Refresh rollout status</button>`);
    else if(linkedCandidate.state==='published'&&!bundleAvailable)guide=next('Refresh the published bundle',`Candidate ${linkedCandidate.id} is published, but the controller has not returned its desired bundle hash. Refresh before proposing deployment.`,`<button class="primary" type="button" data-action="reload">Refresh parser bundle</button>`);
    else if(manual&&linkedCandidate.state==='published')guide=next(manualVerified?'Sample and bundle verified':originAcknowledged?'Verify the sample in Step 9':'Choose and confirm a deployment client',manualVerified?'The published parser returned the expected fields and the selected client reports the exact applied bundle.':'Manual samples use the same rollout approval and client acknowledgement checks. Step 9 checks the sample through the controller production parser.','',manualVerified);
    else if(linkedCandidate.state==='published'&&!originAcknowledged&&!liveBundleProposals.length)guide=next('Create the origin-client rollout proposal',`Deploy bundle ${syncBundleHash.slice(0,20)} to ${originNode||'the diagnosed origin client'} first. Proposal creation does not deliver the bundle; approve its exact revision and hash below.`, `<button class="primary" type="button" data-action="parser-sync-rollout-propose" data-target-mode="origin" data-origin-node="${esc(originNode)}" ${originRolloutAvailable?'':'disabled'}>8.1 · Deploy to origin client</button>`);
    else if(linkedCandidate.state==='published'&&!originAcknowledged&&pendingBundleProposal)guide=next(pendingBundleProposal.created_by===state.user?.username?'Complete the audited rollout approval':'Review and approve the exact rollout',`Proposal ${pendingBundleProposal.id} is waiting for bundle ${syncBundleHash.slice(0,20)}. Record the review reason here; administrator self-approval is bound to the exact revision and hash.`,rolloutApprovalControls(pendingBundleProposal,syncBundleHash));
    else if(linkedCandidate.state==='published'&&!originAcknowledged)guide=next('Verify the origin client acknowledgement',originClient?.error||`Wait for ${originNode||'the originating client'} to apply bundle ${syncBundleHash.slice(0,20)}, then check its acknowledgement here.`,verifyOriginButton('8.1 · Verify origin acknowledgement'));
    else if(linkedCandidate.state==='published'&&originAcknowledged&&originExhausted&&!originEligibility.eligible&&!originResolved)guide=next('Review the retained record’s replay status','The origin client has applied the bundle. Check the replay controls and controller history below for this record.',`<button class="secondary" type="button" data-action="qrw-navigate" data-step="9">9 · Open replay verification</button>`);
    else if(linkedCandidate.state==='published'&&originAcknowledged&&!originResolved&&!replayObserved)guide=next('Replay only the acknowledged origin records','The exact desired bundle is acknowledged by the origin client. Select eligible records from that client below and request the audited replay.');
    else if(originRecord?.status==='replay_failed')guide=next('Review the failed replay',originEligibility.reason||'Open Step 9 to see the failure. Return to Step 8 to request another eligible replay.',`<button class="secondary" type="button" data-action="qrw-navigate" data-step="9">9 · Open replay verification</button>`);
    else if(replayObserved&&!originResolved)guide=next('Verify the accepted replay','The controller accepted the targeted replay request. Step 9 is ready; refresh the controller result to confirm whether the record resolved.',`<button class="secondary" type="button" data-action="qrw-navigate" data-step="9">9 · Open replay verification</button>`);
    else if(originResolved)guide=next('Recovery complete','The controller confirmed that replay resolved this record. No further action is required; rollback remains available on the published candidate.',`<button class="secondary" type="button" data-action="reload">Refresh verification</button>`,true);
    else guide=next('Continue from the current governed state',`Candidate state: ${linkedCandidate.state||'unknown'}. Review its lifecycle below and follow the enabled action.`);
    const signatureOf=r=>{const verified=r.failure_signature||r.signature;if(verified)return `verified:${String(verified).toLowerCase()}`;const identity=r.id||r.record_id||`${r.node_id||'unknown'}|${r.first_seen||r.created_at||'unknown'}|${r.raw_sha256||'no-hash'}`;return `unverified-record:${String(identity).toLowerCase()}`;};
    const groups=new Map();records.forEach(r=>{const signature=signatureOf(r);if(!groups.has(signature))groups.set(signature,[]);groups.get(signature).push(r);});
    const rows=[...groups.entries()].flatMap(([signature,group])=>{
      const summary=group[0],sampleId=summary.id||summary.record_id||'',verified=signature.startsWith('verified:');
      const groupHeader=`<tr class="quarantine-signature"><td colspan="7"><b>${esc(summary.service||'unknown family')}</b> · ${esc((summary.error_codes||[]).join(', ')||summary.error||'no_match')} <span class="badge">${verified?`${group.length} signature-matched record${group.length===1?'':'s'}`:'Individual record · shared cause unverified'}</span> <small class="mono">${esc(signature.replace(/^[^:]+:/,''))}</small></td></tr>`;
      const detailRows=group.map(r=>{
      const id=r.id||r.record_id||'',canReplay=qrwReplayEligibility(r,syncBundleHash).eligible;
      const parser=candidateForRecord(id,{publishedOnly:true});
      const asset=parser&&(state.assuranceSyncAssets||[]).find(a=>a.asset_id===parser.id),client=parserSyncClients(asset).find(c=>c.node_id===r.node_id),applied=qrwExactAcknowledgement(asset,r.node_id);
      return '<tr><td><label><input type="checkbox" data-gprs-record="'+esc(id)+'" '+(quarantineCanInspect(r)?'':'disabled')+' aria-label="Select failure '+esc(id)+'"></label> <span class="mono">'+esc(id)+'</span></td><td>'+time(r.first_seen||r.created_at)+'</td><td>'+esc(r.node_id||'—')+'<br>'+esc(r.service||'family unavailable')+'</td><td>'+esc((r.error_codes||[]).join(', ')||r.error||'No parser matched')+'</td><td>'+badge(r.status||'unknown')+'<br>'+quarantineAttemptLabel(r)+'</td><td>'+(r.raw_available?'Ready':'Awaiting exact evidence')+'</td><td>'+ (r.raw_available?'<button class="secondary small" type="button" data-action="quarantine-raw" data-id="'+esc(id)+'">View / download raw</button> ':'')+(quarantineCanInspect(r)?'<button class="secondary small" type="button" data-action="quarantine-diagnose" data-id="'+esc(id)+'" data-service="'+esc(r.service||'auto')+'">Inspect</button> '+mapSampleButton('Quarantine evidence','quarantine-diagnose',r):'')+(quarantineAwaitsEvidence(r)?'<button class="secondary small" type="button" data-action="quarantine-acquire-evidence" data-node="'+esc(r.node_id||'')+'">Acquire evidence</button> ':'')+(!['resolved','reprocessing','excluded','evidence_deleted','dead_letter','manual_attention'].includes(r.status)?'<button class="secondary small" type="button" data-action="quarantine-exclude" data-id="'+esc(id)+'">Ignore / exclude</button>':'')+' '+(r.status==='excluded'?'<button class="secondary small" type="button" data-action="quarantine-include" data-id="'+esc(id)+'">Return to queue</button>':'')+' <button class="danger small" type="button" data-action="quarantine-delete-raw" data-id="'+esc(id)+'" '+(r.raw_available?'':'disabled')+'>Delete raw</button></td></tr><tr><td colspan="7" class="hint">'+(quarantineAwaitsEvidence(r)?quarantineEvidenceHelp(r.node_id||''):(canReplay?(parser?(applied?'Exact published bundle acknowledged by this client; replay is available below':'Waiting for this client to acknowledge the exact published parser bundle'):'Replay blocked until the linked parser is published'): 'Replay unavailable for this state or retry limit'))+'</td></tr>';
      });return [groupHeader,...detailRows];
    });
    const lifecycle=(manual?parsers.filter(doc=>doc.id===linkedCandidate?.id):parsers).map(doc=>{const payload=doc.payload||doc;const actions=[];if(['draft','rejected'].includes(doc.state)){actions.push(['validate','Retest']);actions.push(['submit','Submit']);actions.push(['edit','Edit']);actions.push(['delete','Delete']);}if(doc.state==='in_review'){actions.push(['cancel','Withdraw']);actions.push(['approve','Approve']);actions.push(['reject','Reject']);}if(doc.state==='approved'&&payload.activated!==true)actions.push(['publish','Publish']);if(doc.state==='published'&&payload.activated===true){actions.push(['rollback','Rollback']);actions.push(['offboard','Offboard']);}const sync=(state.assuranceSyncAssets||[]).find(asset=>asset.asset_id===doc.id),clients=parserSyncClients(sync),applied=clients.filter(client=>!client.drifted&&client.applied_hash&&client.applied_hash===client.desired_hash).length,live=clients.length?`${applied}/${clients.length} clients acknowledged`:'No client acknowledgement reported';if(doc.state==='published'&&sync){actions.push(['verify','Verify fleet']);actions.push(['retry','Propose retry']);}const status=payload.activated===false?(payload.offboarded_at?'offboarded':payload.rolled_back_at?'rolled back':'inactive'):doc.state||'unknown',ownReview=doc.created_by===state.user?.username;return '<tr><td><b>'+esc(payload.name||doc.id)+'</b><br><small class="mono">'+esc(doc.id)+' · r'+number(doc.revision)+'</small></td><td>'+badge(status)+'</td><td>'+esc(payload.artifact_checksum||payload.approved_checksum||'checksum unavailable')+'</td><td>'+esc(JSON.stringify(payload.validation||{}))+'</td><td>'+esc(live)+'</td><td>'+actions.map(([action,label])=>action==='edit'?'<button class="secondary small" data-action="assurance-parser-edit" data-id="'+esc(doc.id)+'">Edit</button>':action==='delete'?'<button class="danger small" data-action="assurance-parser-delete" data-id="'+esc(doc.id)+'">Delete</button>':action==='verify'?'<button class="secondary small" data-action="parser-sync-verify" data-class="'+esc(sync.asset_class||'')+'" data-id="'+esc(sync.asset_id)+'">Verify</button>':action==='retry'?'<button class="secondary small" data-action="parser-sync-retry" data-class="'+esc(sync.asset_class||'')+'" data-id="'+esc(sync.asset_id)+'">Propose retry</button>':action==='approve'?'<button class="'+(ownReview&&canEmergencyApprove?'warning':'secondary')+' small" data-action="assurance-parser-action" data-id="'+esc(doc.id)+'" data-value="approve" '+(ownReview&&!canEmergencyApprove?'disabled title="A different reviewer is required"':'')+'>'+(ownReview?'Approve exact revision':'Approve')+'</button>':'<button class="'+(action==='offboard'?'danger':'secondary')+' small" title="'+(action==='rollback'?'Remove this revision from the active bundle as rollback recovery; the new bundle still needs rollout approval.':action==='offboard'?'Intentionally retire this parser from the active bundle; the new bundle still needs rollout approval.':'')+'" data-action="assurance-parser-action" data-id="'+esc(doc.id)+'" data-value="'+action+'">'+label+'</button>').join(' ')+'</td></tr>';});
    const fixtureOptions=fixtures.map(item=>{const v=item.payload||item,id=item.id||v.id;return '<option value="'+esc(id)+'">'+esc(v.name||id)+' · '+esc(v.parser_id||'automatic')+' · '+(v.passed===true?'passed':v.passed===false?'failed':'retained')+'</option>';}).join('');
    const replayRecords=records.filter(r=>qrwReplayEligibility(r,syncBundleHash).eligible);
    const replayOptions=replayRecords.map(r=>{const id=r.id||r.record_id,parser=candidateForRecord(id,{publishedOnly:true}),asset=parser&&(state.assuranceSyncAssets||[]).find(a=>a.asset_id===parser.id),client=parserSyncClients(asset).find(c=>c.node_id===r.node_id),applied=qrwExactAcknowledgement(asset,r.node_id),isOrigin=id===state.quarantineDiagnosis?.id;return `<option value="${esc(id)}" data-node="${esc(r.node_id||'')}" data-ack="${applied?'true':'false'}" ${isOrigin?'':'hidden'} ${applied&&isOrigin?'selected':''} ${applied&&isOrigin?'':'disabled'}>${esc(id)} · ${esc(r.node_id||'unknown origin')} · ${applied?'exact bundle acknowledged':'blocked: exact bundle not acknowledged'}</option>`;}).join('');
    const originReplayReady=replayRecords.some(r=>(r.id||r.record_id)===state.quarantineDiagnosis?.id&&r.node_id===originNode&&(state.assuranceSyncAssets||[]).some(asset=>asset.asset_id===candidateForRecord(r.id||r.record_id,{publishedOnly:true})?.id&&qrwExactAcknowledgement(asset,r.node_id)));
    const verificationRows=state.quarantineHistory.filter(item=>!state.quarantineDiagnosis?.id||(item.record_id||item.id)===state.quarantineDiagnosis.id).slice(0,10).map(item=>`<tr><td>${time(item.timestamp||item.created_at)}</td><td class="mono">${esc(item.record_id||item.id||'—')}</td><td>${esc(item.node_id||'—')}</td><td>${badge(item.status||item.outcome||'pending')}</td><td>${esc(item.message||item.error||item.action||'—')}</td></tr>`);
    const proposalRows=originProposals.map(doc=>{const p=doc.payload||{},hash=rolloutBundleHash(doc),liveId=p.live_rollout_id||'',live=(state.parserLiveRollouts||[]).find(item=>item.id===liveId);return `<tr><td class="mono"><b>${esc(doc.id)}</b><br><small>${esc(p.name||'Parser bundle rollout')}</small></td><td>${badge(doc.state||'unknown')}<br><small>${esc(p.target_mode||'selected')} · ${number((p.target_nodes||[]).length)} client(s)</small></td><td class="mono" title="${esc(hash)}">${esc(hash.slice(0,20)||'hash unavailable')}<br><small>Current exact bundle</small></td><td class="mono">${esc(liveId||live?.id||'—')} ${live?`<br>${badge(live.status||'pending')}`:''}</td><td>${rolloutApprovalControls(doc,syncBundleHash)}</td></tr>`;});
    const parserLiveRows=(state.parserLiveRollouts||[]).filter(item=>item.type==='parser_sync').map(item=>`<tr><td class="mono">${esc(item.id||'—')}</td><td>${badge(item.status||'unknown')}</td><td>${number(item.total_nodes)} targets</td><td>${number(item.done_nodes)} complete · ${number(item.failed_nodes)} failed</td></tr>`);
    const rolloutsAvailable=!state.parserRolloutLoadError;
    const rolloutAvailability=state.parserRolloutLoadError?`<div class="callout warn">Pending approval and live rollout state is unavailable: ${esc(state.parserRolloutLoadError)}. Refresh before creating another proposal.</div>`:!state.assuranceSyncBundle?'<div class="callout warn">Global parser bundle state is unavailable. Refresh the wizard before proposing a rollout.</div>':!bundleAvailable?'<div class="callout warn">No published parser bundle is available for rollout. Publish an approved parser candidate first.</div>':`<div class="callout ${eligibleFleetNodes.length?'':'warn'}"><b>Global rollout availability:</b> bundle ${esc(syncBundleHash.slice(0,20))} · ${eligibleFleetNodes.length} eligible enrolled client(s) reported online. All eligible clients excludes offline clients at proposal creation. <b>${originRolloutAvailable?'Target client is eligible.':originNode?'Target client is currently unavailable.':manual?'Select a client for targeted delivery.':'Inspect an origin record to enable an origin proposal.'}</b></div>`;
    const rolloutProposalPanel=panel('8.2–8.3 · Propose and approve exact bundle delivery','Client delivery starts after authorized rollout approval. Independent review is preferred; administrators can approve their own proposals with an audited reason bound to the exact revision and bundle hash.',rolloutAvailability+(liveBundleProposals.length?'<div class="callout">This client already has a rollout in progress for the exact bundle.</div>':'')+`<div class="actions qrw-rollout-actions"><button class="primary" type="button" data-action="parser-sync-rollout-propose" data-target-mode="${manual?'selected':'origin'}" data-origin-node="${esc(originNode)}" ${rolloutsAvailable&&bundleAvailable&&originRolloutAvailable&&!liveBundleProposals.length?'':'disabled'}>${manual?'Propose to selected client':'Propose to origin client'}</button><button class="secondary" type="button" data-action="parser-sync-rollout-propose" data-target-mode="all_eligible" data-origin-node="${esc(originNode)}" ${rolloutsAvailable&&bundleAvailable&&eligibleFleetNodes.length&&!liveBundleProposals.length?'':'disabled'}>Propose to all eligible clients</button></div><p class="hint">Each proposal snapshots bundle hash ${esc(syncBundleHash||'unavailable')}. Proposal creation does not deliver the bundle; use the Approve action below after independent review, or Cancel to withdraw it.</p>`+table(['Proposal ID','State / targets','Bundle hash','Live rollout','Approval actions'],proposalRows,'No active proposal for the current parser bundle.')+table(['Current live rollout','State','Targets','Results'],parserLiveRows,'No parser bundle rollout is currently live.'));
    const preview=state.gprsPreview?'<pre class="json-output" id="gprs-preview" data-testid="gprs-preview">'+esc(pretty(state.gprsPreview))+'</pre>':'<pre class="json-output" id="gprs-preview" data-testid="gprs-preview">No live preview has run.</pre>';
    const inspected=!!manual||!!state.quarantineDiagnosis?.sha256||!!linkedCandidate,schemaComplete=!!state.qrwFields.length||!!linkedCandidate,mappingComplete=(!!state.qrwFields.length&&state.qrwFields.every(field=>!['unresolved','invalid','needs_confirmation','preview_pending'].includes(field.status)))||!!linkedCandidate,wizardSteps=[['Select','Retained or pasted log',!!(manual||state.quarantineDiagnosis),false],['Inspect','Exact source',inspected,!(manual||state.quarantineDiagnosis)],['Schema','Desired JSON',schemaComplete,!inspected],['Map','Verified values',mappingComplete,!schemaComplete],['Generate','Parser candidate',!!linkedCandidate,!mappingComplete],['Test','Candidate preview',previewPassed,!linkedCandidate&&!state.gprsSamples.length],['Review & publish','Approval and publish',linkedCandidate?.state==='published',!previewPassed&&!candidateValidationPassed],[manual?'Deploy':'Deploy & replay',manual?'Selected client ACK':'Origin bundle ACK',originAcknowledged&&(!!manual||replayObserved),linkedCandidate?.state!=='published'],['Verify result','Controller outcome',manual?manualVerified:!!originResolved,!verificationAvailable]];
    const wizardNav='<ol class="quarantine-wizard nine-step" aria-label="Quarantine resolution workflow">'+wizardSteps.map(([title,sub,complete,locked],index)=>`<li data-wizard-step="${index+1}" data-step-complete="${complete}" data-step-locked="${locked}" class="${state.qrwStep===index+1?'active ':''}${complete?'done ':''}${locked?'locked':''}"><button type="button" data-action="qrw-navigate" data-step="${index+1}" aria-current="${state.qrwStep===index+1?'step':'false'}" ${locked?'disabled':''}><b>${index+1} · ${title}</b><small>${sub}</small><span class="qrw-step-state">${complete?'Complete':state.qrwStep===index+1?'Current · do this now':locked?'Locked':'Next required'}</span></button></li>`).join('')+'</ol><p id="qrw-step-hint" class="qrw-step-hint" role="status"></p>';
    const manualInput=`<section class="panel"><h3>Use a new raw log</h3><div class="form-grid"><label class="field wide">Manual raw sample<textarea id="gprs-manual-raw" rows="4" placeholder="Paste one representative log record">${esc(manual?.raw||'')}</textarea><span class="hint">Paste one record, including continuation lines if needed. Redact secrets before testing or saving a parser validation case.</span></label><label class="field">Service scope<input id="gprs-manual-service" value="${esc(manual?.service||'auto')}" maxlength="64" placeholder="auto"></label><div class="actions"><button class="primary" type="button" data-action="gprs-use-manual">1 · Use manual sample</button>${mapSampleButton('Guided Parser Repair','gprs-use-manual')}</div><label class="field wide"><span><input id="gprs-retain-sample" type="checkbox" ${state.sampleHistoryEnabled?'checked':''} ${can('assurance:save')?'':'disabled'}> Save an encrypted copy to Test Sample History</span><small>Only enable for records you are authorized to retain. Controller retention limits apply.</small></label></div></section>`;
    const manualDeploy=manual?`<section class="qrw-stage-panel" data-qrw-panel="8">${panel('Step 8 · Deploy the published bundle','Select one client to verify. Rollout proposals and approvals use the same Fleet Sync controls.',`<label class="field">Verification client<select id="gprs-manual-node"><option value="">Choose an enrolled client</option>${(state.assuranceFleetNodes||[]).map(item=>item.node||item).filter(node=>node.id).map(node=>`<option value="${esc(node.id)}" ${node.id===originNode?'selected':''}>${esc(node.name||node.id)} · ${esc(node.status||'unknown')}</option>`).join('')}</select></label>`)+rolloutProposalPanel+panel('8.4 · Confirm selected client acknowledgement','The client must report the same full hash as the desired bundle.',`<p>Desired bundle <code>${esc(syncBundleHash||'unavailable')}</code><br>Applied bundle <code>${esc(originClient?.applied_hash||'pending')}</code></p><div class="actions"><button class="secondary" type="button" data-action="parser-sync-verify" data-class="${esc(linkedAsset?.asset_class||'')}" data-id="${esc(linkedAsset?.asset_id||'')}" ${linkedAsset?'':'disabled'}>8.4 · Refresh acknowledgement</button><button class="primary" type="button" data-action="qrw-navigate" data-step="9" ${verificationAvailable?'':'disabled'}>9 · Verify manual sample</button></div>`)}</section>`:'';
    const manualVerify=manual?`<section class="qrw-stage-panel" data-qrw-panel="9">${panel('Step 9 · Verify the manual sample','Run the sample through the controller production parser and check the selected client’s exact bundle acknowledgement. A manual sample has no retained record to replay.',`<p role="status">${manualVerified?'Complete · Published parser and automatic output matched the expected fields. The selected client acknowledged the bundle.':manual.verification?(manual.verification.result.selected_expected_parser?'The output or bundle status needs review.':'The automatic parser selected a different parser. Review the selected parser and the candidate’s service scope or priority.'):'Ready to compare production output with the saved expected fields.'}${manualVerified&&manual.verification.result.selected_expected_parser===false?' Automatic selection used a different parser with equivalent output.':''}</p><div class="actions"><button class="secondary" type="button" data-action="qrw-navigate" data-step="8">8 · Back to deployment</button><button class="primary" type="button" data-action="gprs-verify-manual" ${verificationAvailable?'':'disabled'}>9.1 · Run production verification</button></div>${manual.verification?`<pre>${esc(pretty(manual.verification.result))}</pre>`:''}`)}</section>`:'';
    return panel('Guided Parser Repair Studio','Inspect a retained record or paste a new sample, then map, test, review, publish, deploy and verify the same parser candidate.',
      wizardNav+guide+'<div class="actions"><button type="button" class="secondary" data-action="gprs-reset">Reset Guided Parser Repair</button></div><section class="qrw-stage-panel" data-qrw-panel="1"><div class="callout">This wizard uses the controller’s Assurance, Parser Catalog, Fleet Sync, and quarantine actions. The active parser is unchanged until the existing approval and deployment flow completes. Candidate submission, independent approval, publication, exact client bundle acknowledgement, and replay remain separate audited steps.</div>'+manualInput+table(['Select / ID','First seen','Node / family','Failure','State / attempts','Evidence','Actions'],rows,'No retained parsing failures are reported.')+'</section>'+
      '<form id="gprs-preview-form" class="form-grid qrw-stage-panel" data-qrw-panel="2"><h3 class="wide">Step 2 · Inspect evidence and name the desired fields</h3><label class="field wide">Exact raw log evidence<textarea name="raw_evidence" rows="4" placeholder="Choose a retained record or enter a new sample in Step 1." readonly>'+esc(manual?.raw||'')+'</textarea><span class="hint">'+(manual?'Manual sample supplied in Step 1. Return there to change it.':'Exact source from the authenticated quarantine analysis endpoint.')+'</span></label><label class="field wide">Desired JSON field names<textarea name="manual_field_names" rows="3" placeholder="timestamp, host, service, severity, message"></textarea><span class="hint">Enter names in raw-token order, separated by commas or new lines. NLI creates editable mappings; it never invents a value from a name.</span></label><label class="field">Quick field name<input name="manual_field_name" placeholder="Single-field fallback, e.g. username"></label><label class="field">Starting token index<input name="manual_field_index" type="number" min="1" max="256" value="1"></label><label class="field">Quick delimiter<input name="manual_delimiter" value="=" maxlength="8"></label><label class="field">Output template<input name="manual_template" placeholder="{timestamp} {level} {message}"></label><label class="field wide">Safe regular expression<input name="manual_regex" placeholder="^(?P&lt;level&gt;\\w+)\\s+(?P&lt;message&gt;.*)$"></label><div class="actions wide"><button class="primary" type="button" data-action="qrw-apply-field-names">2.1 · Create mappings from my field names</button><button class="secondary" type="button" data-action="qrw-suggest-fields">2.2 · Suggest names from the raw log</button>'+mapSampleButton('Inspected evidence','map-guided-sample')+'</div><div id="qrw-inspect-status" class="callout wide" role="status">Inspect the raw record, enter the names you want in the output JSON, then create editable mappings.</div><label class="field wide">Desired JSON output instance<textarea name="gprs_desired" required>{}</textarea><span class="hint">Generated from verified field mappings below. Values are never invented from field names.</span></label><label class="field wide">Field mappings<textarea name="mapping">{}</textarea><span class="hint">Evidence-bearing extraction source to output field mapping.</span></label><label class="field wide">Saved fixtures for multi-record validation<select name="fixtures" multiple size="4">'+(fixtureOptions||'')+'</select></label><div class="actions wide"><button class="secondary qrw-back-button" type="button" data-action="qrw-navigate" data-step="1">1 · Back to source selection</button><button class="primary qrw-next-button" type="button" data-action="qrw-navigate" data-step="3">3 · Review desired schema</button></div></form>'+renderQRWFieldEditor()+'<section class="qrw-stage-panel" data-qrw-panel="6"><div class="callout"><b>Step 6 · Test with production Assurance.</b> This deterministic preview cannot activate a parser. AI becomes available only after a failed single-record result.</div><div class="actions"><button class="primary qrw-next-button" type="button" data-action="gprs-preview">6 · Run deterministic preview</button><button class="secondary" type="button" data-action="gprs-open-editor">Edit generated candidate</button></div>'+preview+'<div id="gprs-ai-fallback" class="callout">Deterministic recovery is first. AI appears only after a failed single-record preview.</div><div class="wizard-navigation"><button class="secondary qrw-back-button" type="button" data-action="qrw-navigate" data-step="5">5 · Back to candidate</button><button class="primary qrw-next-button" type="button" data-action="qrw-navigate" data-step="7" '+(tested&&previewPassed?'':'disabled title="Run and pass the deterministic preview first"')+'>7 · Review and approve</button></div></section>'+
      '<div id="qrw-lifecycle"></div><section class="qrw-stage-panel" data-qrw-panel="7">'+panel('Step 7 · Review, approve, then publish','Complete these five checks in order without leaving this workspace. Independent approval remains preferred; administrators also have an explicit, reason-bearing emergency action bound to the exact revision and checksum.',reviewRunbook+table(['Candidate','State','Artifact checksum','Validation','Live acknowledgement','Actions'],lifecycle,'No parser candidates are available.'))+'<div class="wizard-navigation"><button class="secondary qrw-back-button" type="button" data-action="qrw-navigate" data-step="6">6 · Back to test</button><button class="primary qrw-next-button" type="button" data-action="qrw-navigate" data-step="8" '+(linkedCandidate?.state==='published'?'':'disabled title="Publish the approved candidate first"')+'>8 · Continue to deployment</button></div></section>'+
      (manual?manualDeploy+manualVerify:'<section class="qrw-stage-panel" data-qrw-panel="8"><section class="panel qrw-deployment-status" data-testid="qrw-deployment-status"><div class="section-head"><div><h3>Step 8 · Deploy and replay</h3><p>Follow the highlighted action above to deploy or verify. Once the client acknowledges the bundle, request replay below.</p></div></div></section><details class="panel"><summary>Deployment details and fleet controls</summary>'+deployRunbook+rolloutProposalPanel+panel('Origin client acknowledgement','Refresh the controller-reported state until the origin client applies the exact current bundle.',`<p>Origin client: <code>${esc(originNode||'unavailable')}</code><br>Desired bundle: <code>${esc(syncBundleHash||'unavailable')}</code><br>Applied bundle: <code>${esc(originClient?.applied_hash||'not acknowledged')}</code><br>Client acknowledgement: <strong>${originAcknowledged?'Verified':esc(originClient?.status||'pending')}</strong></p>${!originAcknowledged&&originClient?.next_action?`<p class="callout warn">${esc(originClient.error||originClient.next_action)}</p>`:''}<div class="actions">${verifyOriginButton('Refresh origin acknowledgement')} ${linkedAsset&&!originAcknowledged?`<button class="secondary" type="button" data-action="parser-sync-retry" data-class="${esc(linkedAsset.asset_class||'')}" data-id="${esc(linkedAsset.asset_id)}" data-node="${esc(originNode)}">Create retry proposal</button>`:''}</div>` )+'</details><form id="gprs-replay-form" class="form-grid" data-bundle-hash="'+esc(syncBundleHash)+'"><h3 class="wide">8.2 · Targeted replay</h3><label class="field">Safe replay scope<select name="scope"><option value="origin">Origin record only</option><option value="selected">Selected eligible records · one client</option><option value="all">All eligible records · origin client</option></select><span class="hint">The controller accepts one target client per request. “All eligible” is limited to the diagnosed record’s originating client; no cross-client replay is submitted.</span></label><label class="field">Target node ID<input name="node_id" required value="'+esc(originNode)+'" placeholder="Originating enrolled client" readonly></label><label class="field wide">Replay-eligible records for this client<select name="record_ids" multiple size="5">'+replayOptions+'</select><span class="hint wide">Records need retained evidence and an exact applied bundle. A record at its replay limit can be eligible for one recovery replay after a parser bundle change. Select records from one client only.</span></label><div class="callout '+(originReplayReady?'':'warn')+' wide"><b>'+ (originReplayReady?'Ready to replay:':'Replay is safely blocked:')+'</b> '+(originReplayReady?(originEligibility.reason||'The origin client reports the exact desired bundle. Select the record and request replay.'):originAcknowledged?(originEligibility.reason||'No eligible retained record is selected.'):'publish the linked parser and confirm exact bundle acknowledgement from the origin client.')+'</div><button id="gprs-replay-submit" class="primary qrw-next-button" type="submit" '+(originReplayReady?'':'disabled')+'>8.2 · Request targeted replay</button><span class="hint wide">After the controller accepts replay, Step 9 shows the retained record outcome and history.</span><div class="actions wide"><button class="secondary qrw-back-button" type="button" data-action="qrw-navigate" data-step="7">7 · Back to review</button><button class="primary qrw-next-button" type="button" data-action="qrw-navigate" data-step="9" '+(verificationAvailable?'':'disabled title="Request the targeted replay first"')+'>9 · Verify replay result</button></div></form></section><div id="qrw-verification" class="qrw-stage-panel" data-qrw-panel="9">'+panel('Step 9 · Verify the controller-reported result',originRecord?`Origin record ${originRecord.id||originRecord.record_id} · ${originRecord.node_id||'origin unknown'} · ${originRecord.status||'status unavailable'} · ${quarantineAttemptLabel(originRecord)}.${state.qrwReplayAcceptedIds.includes(state.quarantineDiagnosis?.id)&&['waiting','reprocessing'].includes(originRecord.status)&&!originExhausted?' Replay request accepted; the final controller outcome is still pending.':''} Success requires the controller record state or replay history to report resolved.`:'Select and inspect an origin record to show its controller-reported verification state.',verifyRunbook+table(['Time','Record','Client','Result','Details'],verificationRows,'No replay history is reported for the selected record.'))+'<div class="wizard-navigation"><button class="secondary qrw-back-button" type="button" data-action="qrw-navigate" data-step="8">8 · Back to deployment</button><button class="primary qrw-next-button" type="button" data-action="reload">9.1 · Refresh controller result</button></div></div>'));
  }
  function setupAssuranceTabs() {
    const tablist=page.querySelector('.assurance-subnav'); if(!tablist)return;
    tablist.setAttribute('role','tablist');tablist.setAttribute('aria-label','Assurance Lab tools');
    if(!tablist.querySelector('[data-title="Guided Parser Repair Studio"]')){const button=document.createElement('button');button.type='button';button.className='tab';button.dataset.title='Guided Parser Repair Studio';button.textContent='Guided Parser Repair Studio';const helper=[...tablist.querySelectorAll('.tab')].find(item=>item.textContent.includes('Parser Creation Helper'));tablist.insertBefore(button,helper||null);}
    const specs=[['analysis','Analysis'],['corpus','Corpus'],['drafts','Parser drafts'],['review','Review queue'],['history','History'],['fixtures','Fixtures'],['gprs','Guided Parser Repair Studio'],['helper','Parser Creation Helper'],['synthesis','NAPI Synthesis'],['plans','Test Plans'],['runs','Runs']];
    const buttons=[...tablist.querySelectorAll('.tab')];
    buttons.forEach((button,index)=>{button.dataset.assuranceTab=specs[index]?.[0]||'analysis';button.dataset.action='assurance-tab';button.removeAttribute('data-title');button.setAttribute('role','tab');button.id=`assurance-tab-${button.dataset.assuranceTab}`;button.setAttribute('aria-controls',`assurance-panel-${button.dataset.assuranceTab}`);button.setAttribute('aria-selected',index===0?'true':'false');button.classList.toggle('active',index===0);button.tabIndex=index===0?0:-1;});
    const panels=specs.map(([key,label])=>{const section=document.createElement('section');section.className='assurance-tab-panel';section.id=`assurance-panel-${key}`;section.setAttribute('role','tabpanel');section.setAttribute('aria-labelledby',`assurance-tab-${key}`);section.hidden=key!=='analysis';section.dataset.assurancePanel=key;section.setAttribute('tabindex','0');section.innerHTML=`<h2 class="assurance-panel-title">${esc(label)}</h2>`;return section;});
    panels.find(item=>item.dataset.assurancePanel==='gprs').insertAdjacentHTML('beforeend',renderGPRSPanel());
    const classify=node=>{if(node.matches('.assurance-workspace'))return'analysis';const title=node.querySelector('h3')?.textContent?.toLowerCase()||'';if(title.includes('guided parser repair'))return'gprs';if(title.includes('parser creation helper'))return'helper';if(title.includes('napi parser synthesis'))return'synthesis';if(title.includes('test plan'))return'plans';if(title.includes('assurance runs'))return'runs';if(title.includes('upload log')||title.includes('production parser assurance'))return'analysis';if(title.includes('corpus'))return'corpus';if(title.includes('revisioned parser')||title.includes('ai remediation'))return'review';if(title.includes('parser workflow'))return'drafts';if(title.includes('history'))return'history';if(title.includes('fixture'))return'fixtures';return'analysis';};
    let node=tablist.nextElementSibling;while(node){const next=node.nextElementSibling;const section=panels.find(item=>item.dataset.assurancePanel===classify(node));section.append(node);node=next;}
    const helperPanel=panels.find(item=>item.dataset.assurancePanel==='helper'),gprsPanel=panels.find(item=>item.dataset.assurancePanel==='gprs');
    if(helperPanel&&gprsPanel){[...helperPanel.children].filter(child=>child.querySelector?.('#parser-helper-form')).forEach(child=>{const copy=child.cloneNode(true),form=copy.querySelector('#parser-helper-form');copy.querySelectorAll('[id]').forEach(item=>{item.id=`gprs-${item.id}`;});form.id='gprs-parser-helper-form';copy.dataset.qrwPanel='5';copy.classList.add('qrw-stage-panel');copy.insertAdjacentHTML('afterbegin','<div class="callout"><b>Step 5 · Generate a governed candidate.</b> Prepare the editor from the selected evidence, review the exact parser JSON and validation cases, then save a review-gated draft.<div class="actions"><button class="primary qrw-next-button" type="button" data-action="gprs-open-editor">5 · Prepare candidate from evidence</button><button class="primary" type="button" data-action="qrw-generate-validate">Generate + validate complete parser pack</button></div></div>');copy.insertAdjacentHTML('beforeend','<div class="wizard-navigation"><button class="secondary qrw-back-button" type="button" data-action="qrw-navigate" data-step="4">4 · Back to mapping</button><button class="primary qrw-next-button" type="button" data-action="qrw-navigate" data-step="6">6 · Test live preview</button></div>');gprsPanel.append(copy);});const helperTab=buttons.find(button=>button.dataset.assuranceTab==='helper');if(helperTab)helperTab.title='The same candidate editor is also embedded in Guided Parser Repair Studio';}
    let anchor=tablist;panels.forEach(section=>{if(section.children.length===1)section.insertAdjacentHTML('beforeend','<div class="empty-state">No items are available in this section yet.</div>');anchor.after(section);anchor=section;});setupQRWStepPanels(gprsPanel);
  }
  function setupSubsystemTabs() {
    const tablist=page.querySelector('[data-subsystem-tabs]');if(!tablist)return;
    const buttons=[...tablist.querySelectorAll('[data-subsystem-tab]')],panels=[...page.querySelectorAll('[data-subsystem-panel]')];
    const activate=key=>{buttons.forEach(button=>{const active=button.dataset.subsystemTab===key;button.classList.toggle('active',active);button.setAttribute('aria-selected',String(active));button.tabIndex=active?0:-1;});panels.forEach(panel=>panel.hidden=panel.dataset.subsystemPanel!==key);};
    buttons.forEach((button,index)=>{const key=button.dataset.subsystemTab;button.id=`subsystem-tab-${key}`;button.setAttribute('role','tab');button.setAttribute('aria-controls',`subsystem-panel-${key}`);button.setAttribute('aria-selected',String(index===0));button.tabIndex=index===0?0:-1;button.addEventListener('click',()=>activate(key));});
    panels.forEach(panel=>{const key=panel.dataset.subsystemPanel;panel.id=`subsystem-panel-${key}`;panel.setAttribute('role','tabpanel');panel.setAttribute('aria-labelledby',`subsystem-tab-${key}`);panel.tabIndex=0;});
    activate(buttons[0]?.dataset.subsystemTab||'overview');
  }
  function setupEndpointTabs(){
    const panels=[...page.querySelectorAll(':scope > .panel')];if(!panels.length)return;
    const specs=[['destinations','Destinations'],['security','Security & credentials'],['validation','Validation'],['profiles','Saved profiles'],['live','Live & rollback']];
    const nav=document.createElement('div');nav.className='assurance-subnav';nav.setAttribute('role','tablist');nav.setAttribute('aria-label','Endpoint configuration sections');
    const buttons=specs.map(([key,label],i)=>{const b=document.createElement('button');b.type='button';b.className='tab'+(i===0?' active':'');b.textContent=label;b.dataset.endpointTab=key;b.setAttribute('role','tab');b.setAttribute('aria-selected',String(i===0));b.tabIndex=i===0?0:-1;nav.append(b);return b;});
    const getPanel=needle=>panels.find(p=>(p.querySelector('h3')?.textContent||'').toLowerCase().includes(needle));
    const adapters=getPanel('supported output'),editor=getPanel('create output'),saved=getPanel('saved output'),rollout=getPanel('roll out');
    const form=editor?.querySelector('#output-profile-form');let groups=[];
    if(form){const formChildren=[...form.children],classify=node=>{const names=[...node.querySelectorAll('[name]')].map(el=>el.name);if(names.some(n=>['loki_token_env','loki_tenant','clickhouse_username','clickhouse_password_env','controller_token_file','controller_node_id_file'].includes(n)))return'security';if(names.some(n=>['loki_labels','field_columns'].includes(n))||node.id==='output-config-test-result'||node.id==='output-profile-preview'||node.querySelector('[data-action="output-config-test"],[data-action="download-output-config"]'))return'validation';return'destinations';};groups=specs.slice(0,3).map(([key,label],i)=>{const section=document.createElement('section');section.className='endpoint-form-section';section.dataset.endpointFormSection=key;section.id='endpoint-form-'+key;section.setAttribute('role','tabpanel');section.setAttribute('aria-label',label);section.hidden=i!==0;return section;});formChildren.forEach(child=>groups.find(section=>section.dataset.endpointFormSection===classify(child))?.append(child));form.prepend(...groups);}
    const activate=key=>{buttons.forEach(button=>{const active=button.dataset.endpointTab===key;button.classList.toggle('active',active);button.setAttribute('aria-selected',String(active));button.tabIndex=active?0:-1;});if(groups.length)groups.forEach(group=>group.hidden=group.dataset.endpointFormSection!==key);if(adapters)adapters.hidden=key!=='destinations';if(editor)editor.hidden=!['destinations','security','validation'].includes(key);if(saved)saved.hidden=key!=='profiles';if(rollout)rollout.hidden=key!=='live';};
    if(form)form.addEventListener('invalid',event=>{const section=event.target.closest('[data-endpoint-form-section]');if(section)activate(section.dataset.endpointFormSection);},true);
    const intro=page.querySelector('.engine-intro');(intro||page.firstChild).after(nav);
    if(adapters)adapters.setAttribute('aria-label','Destinations');if(editor)editor.setAttribute('aria-label','Output profile editor');if(saved)saved.setAttribute('aria-label','Saved endpoint profiles');if(rollout)rollout.setAttribute('aria-label','Live endpoint rollout and rollback');
    nav.addEventListener('click',event=>{const button=event.target.closest('[data-endpoint-tab]');if(button)activate(button.dataset.endpointTab);});activate('destinations');
  }

  function renderPage(id, html) { if (state.user && state.page === id) { page.innerHTML = html; page.dataset.page = id; page.dataset.dirty = 'false'; addInputGuidance();queueMicrotask(nfiDecorate);if(['assurance','quarantine-wizard'].includes(id))setupAssuranceTabs();if(id==='outputs')setupEndpointTabs();if(id==='ai')setupAITabs();if(id==='monitoring')setupMonitoringTabs();if(['sync-fabric','nhle-monitor','napi-monitor','nal-monitor','upes'].includes(id))setupSubsystemTabs();decorateWorkspaceAI(); } }
  function mappingSuggestionButton(tab='Field mapping'){
    return `<button type="button" class="secondary" data-action="workspace-ai-auto" data-artifact="mapping" data-tab="${esc(tab)}" data-suggest-mapping>Suggest Mappings with Nexora AI</button>`;
  }
  function mapSampleButton(tab,action='workspace-ai-auto',record=null){
    const attrs=record?` data-id="${esc(record.id||record.record_id)}" data-service="${esc(record.service||'auto')}"`:'';
    return `<button type="button" class="secondary" data-action="${esc(action)}" data-tab="${esc(tab)}" data-artifact="mapping" data-map-sample${attrs}>Suggest &amp; map with Nexora AI</button>`;
  }
  async function mapGuidedSample(){
    const editor=page.querySelector('.qrw-field-editor');
    const button=editor?.querySelector('[data-map-sample]');
    if(!button||!qrwRaw().trim())throw new Error('Exact sample evidence is unavailable for mapping.');
    state.qrwStep=4;
    setupQRWStepPanels();
    editor.scrollIntoView({block:'start'});
    button.dataset.aiPending='true';
    button.setAttribute('aria-busy','true');
    try{await action(button);}
    finally{delete button.dataset.aiPending;button.removeAttribute('aria-busy');}
  }
  function workspaceAIActions(tab='') {
    return `<div class="actions workspace-ai-actions"><button type="button" class="secondary" data-action="workspace-ai-open" data-mode="ask" data-tab="${esc(tab)}">Ask Nexora AI</button><button type="button" class="secondary" data-action="workspace-ai-open" data-mode="fix" data-tab="${esc(tab)}">Fix with Nexora AI</button><button type="button" class="secondary" data-action="workspace-ai-auto" data-tab="${esc(tab)}">Auto Fix with Nexora AI</button></div>`;
  }
  function decorateWorkspaceAI() {
    if (!page.dataset.page) return;
    page.insertAdjacentHTML('afterbegin',workspaceAIActions());
    const analysis=page.querySelector('#assurance-form');
    if(analysis)analysis.insertAdjacentHTML('beforeend',mapSampleButton('Assurance analysis')+'<button type="button" class="secondary" data-action="assurance-reset-sample">Reset analysis sample</button>');
    if (!['header-templates','kv-templates'].includes(state.page)) {
      if (state.page === 'knownkeys') {
        page.querySelector('#config-form')?.insertAdjacentHTML('beforebegin', '<label class="field">Raw log sample for template creation<textarea name="raw" rows="3" spellcheck="false"></textarea></label>');
      }
      for (const input of page.querySelectorAll('textarea[name=raw],textarea[name=samples],textarea[name=raw_evidence],textarea[data-ai-mapping-raw],textarea#nfi-raw,textarea#block-raw')) {
        input.insertAdjacentHTML('afterend','<div class="actions" data-sample-template-actions><button type="button" class="secondary" data-action="template-from-sample" data-kind="header-templates">Create log pattern template</button><button type="button" class="secondary" data-action="template-from-sample" data-kind="kv-templates">Create KV pair template</button><button type="button" class="secondary" data-action="reset-source-sample">Clear raw sample</button></div>');
      }
      if (['quarantine-wizard','field-mapping','assurance'].includes(state.page)) {
        for (const editor of page.querySelectorAll('.qrw-field-editor .section-head .actions')) {
          editor.insertAdjacentHTML('beforeend','<button type="button" class="secondary" data-action="template-from-sample" data-kind="header-templates">Create log pattern template</button><button type="button" class="secondary" data-action="template-from-sample" data-kind="kv-templates">Create KV pair template</button>');
        }
      }
    }
    for(const form of page.querySelectorAll('#parser-workspace-form,#parser-helper-form,#gprs-parser-helper-form')){
      const sample=form.querySelector('[name=samples]');
      sample?.closest('label')?.insertAdjacentHTML('afterend',`<div class="actions wide">${mapSampleButton('Parser creation sample')}</div>`);
    }
    const fieldSense=state.page==='field-mapping'&&page.querySelector('.workspace-ai-actions');
    if(fieldSense)fieldSense.insertAdjacentHTML('beforeend','<button type="button" class="secondary" data-action="field-sense-reset">Reset mapping session</button>');
    for(const section of page.querySelectorAll('[role=tabpanel],[data-endpoint-form-section],[data-qrw-panel]')) {
      if(section.matches('.qrw-field-editor')) {
        section.querySelector('.section-head .actions')?.insertAdjacentHTML('beforeend',mapSampleButton('Field mapping')+'<button type="button" class="secondary" data-action="workspace-ai-auto" data-tab="Field mapping">Auto Fix with Nexora AI</button>');
        continue;
      }
      const tab=section.getAttribute('aria-label')||document.getElementById(section.getAttribute('aria-labelledby'))?.textContent||section.dataset.endpointFormSection||`Step ${section.dataset.qrwPanel||''}`;
      section.insertAdjacentHTML('afterbegin',workspaceAIActions(tab));
    }
  }
  const WORKSPACE_ARTIFACTS=[['mapping','Field mapping'],['parser','Full parser pack'],['regex','Regex'],['template','Template'],['pattern','Pattern'],['kv_template','KV pair template'],['key_pack','Key pack'],['keys','Known keys']];
  function workspaceAISample(button) {
    if(button.hasAttribute?.('data-template-sample')){
      return el('template-composition-sample')?.value.split(/\r?\n/).find(line=>line.trim())||'';
    }
    if(button.hasAttribute?.('data-map-sample')){
      const input=button.closest('form')?.querySelector('textarea[name=samples],textarea[name=raw_evidence],textarea[name=raw]');
      if(input)return input.value;
    }
    if(state.page==='building-blocks')return el('block-raw')?.value||'';
    if(state.page==='field-mapping')return qrwRaw();
    if(state.page==='assurance'&&!button.closest('[data-assurance-panel=gprs]')&&button.dataset.tab!=='Field mapping'){
      return page.querySelector('#assurance-form [name=raw]')?.value||page.querySelector('textarea[data-ai-mapping-raw]')?.value||'';
    }
    if(['assurance','quarantine-wizard'].includes(state.page)){
      const last=state.lastAssurance?.request;
      const retained=state.quarantineDiagnosis?.id&&last?.source===`quarantine:${state.quarantineDiagnosis.id}`?last.raw:'';
      return qrwRaw()||state.gprsManual?.raw||retained||'';
    }
    if(state.page==='parser-workspace')return page.querySelector('textarea[name=samples]')?.value||'';
    return button.closest('section')?.querySelector('textarea[name=raw]')?.value||page.querySelector('textarea[name=raw],textarea[data-ai-mapping-raw]')?.value||'';
  }
  function workspaceAIArtifact(button,raw) {
    const assets={regex:'regex',studio:'template','header-templates':'template',static:'pattern',knownkeys:'keys',kv:'kv_template','kv-templates':'kv_template'};
    return button.dataset.artifact||assets[state.page]||(raw?'mapping':'');
  }
  function workspaceAIForm({mode='fix',tab='',raw='',references=[],defaultArtifact='',helpDesk=false}={}) {
    return `<form class="form-grid" data-workspace-ai-form data-mode="${esc(mode)}" data-tab="${esc(tab)}" ${helpDesk?'data-ai-help-desk':''}><label class="field wide">${mode==='ask'?'What do you need help with?':'Describe the desired fix and any fields to keep, rename, omit or transform'}<textarea name="question" maxlength="2000" rows="3"></textarea></label>
      ${helpDesk?`<label class="field">NLI workspace<select name="help_workspace"><option value="reference-ai">General NLI question</option>${NAV.filter(([,id])=>id!=='reference-ai').map(([,id,title])=>`<option value="${esc(id)}">${esc(title)}</option>`).join('')}</select></label><label class="field">Tab or step (optional)<input name="help_tab" maxlength="128" placeholder="Example: Step 4 - Map"></label><label class="field wide">What happened? (optional)<textarea name="help_issue" rows="3" maxlength="1400" placeholder="Paste the exact error, the result you expected, and what you already tried. Do not include credentials."></textarea></label>`:''}
      ${mode==='fix'?'<label class="field">Draft type<select name="artifact"><option value="">Recommend the next action</option>'+WORKSPACE_ARTIFACTS.map(([id,label])=>`<option value="${id}" ${id===defaultArtifact?'selected':''}>${label}</option>`).join('')+'</select></label>':''}
      <label class="field">Sample origin<select name="evidence_source"><option value="manual">Manual test sample</option><option value="production">Production evidence</option></select></label>${references.length?'<label class="field">Existing reference<select name="reference"><option value="">No reference</option>'+references.map(({kind,id,revision,label})=>`<option value="${esc(JSON.stringify({kind,id,revision}))}">${esc(label)}</option>`).join('')+'</select></label>':''}
      <label class="field wide">Sample record · optional for guidance<textarea name="raw" rows="4" maxlength="65536">${esc(raw)}</textarea></label>${mode==='fix'?'<label class="field wide">Expected mapping output (optional JSON object)<textarea name="desired" rows="3"></textarea></label>':''}<p class="hint wide">The controller masks sample evidence before sending it to the configured provider. Review the proposal before applying it to your editor.</p><div class="actions wide"><button type="button" class="primary" data-action="workspace-ai-run">${mode==='ask'?'Get guidance':'Build draft'}</button><button type="reset" class="secondary">Reset form</button><button type="button" class="secondary" data-action="workspace-ai-import" hidden>Apply mapping to editor</button><button type="button" class="secondary" data-action="workspace-ai-download" hidden>Download draft</button></div><pre class="json-output wide" data-workspace-ai-result role="status"></pre></form>`;
  }
  async function loadReferenceAI(){
    const sources=[['parser','/assurance/parsers'],['regex',CONFIG.regex[0]],['template',CONFIG['header-templates'][0]],['pattern',CONFIG.static[0]],['keys',CONFIG.knownkeys[0]],['kv_template',CONFIG['kv-templates'][0]]];
    const lists=await Promise.allSettled(sources.map(([,path])=>api(path)));
    const unavailable=[];
    const references=lists.flatMap((list,index)=>{
      const [kind]=sources[index];
      if(list.status==='rejected'){unavailable.push(kind);return [];}
      const data=list.value;
      return (data.parsers||data.items||data.documents||[]).filter(item=>item.id).map(item=>({kind,id:item.id,revision:item.revision,label:`${kind} · ${item.payload?.name||item.name||item.id}${item.revision?' · revision '+item.revision:''}`}));
    });
    const status=unavailable.length?`<div class="callout warn">Reference catalogs unavailable for ${esc(unavailable.join(', '))}. You can continue with a sample or refresh to retry.</div>`:'';
    renderPage('reference-ai',status+panel('Ask Nexora AI','Ask about an NLI workflow or describe what is blocked. Choose the workspace so the controller can provide steps that fit it.',workspaceAIForm({mode:'ask',references,helpDesk:true}))+panel('Build or fix an artifact','Choose what to draft and include a representative sample. Review the result before applying or saving it.',workspaceAIForm({references,defaultArtifact:'mapping'})));
  }
  function loadBuildingBlocks(){
    const board=state.blockBoard;
    renderPage('building-blocks',panel('Building Blocks','Drag source pieces onto the output board, or select a piece with Enter. The output cards define an editable parser mapping.',`<label class="field">Raw record<textarea id="block-raw" rows="4">${esc(state.blockSeed||board?.raw||'')}</textarea></label>
      <div class="actions"><button type="button" class="secondary" data-action="block-sense">Sense and populate fields</button>${mapSampleButton('Building Blocks')}<button type="button" class="secondary" data-action="block-add">Split at selected text</button><button type="button" class="secondary" data-action="block-whole">Add whole record</button><button type="button" class="secondary" data-action="block-literal">Add literal</button><button type="button" class="secondary" data-action="block-reset">Reset board</button></div>
      <div class="block-board"><section><h4>1 · Source pieces</h4><label class="field">Cut pieces by <select id="block-mode"><option value="token">Words and punctuation</option><option value="character">Characters</option></select></label><div id="block-source" class="block-tray"></div></section>
      <section><h4>2 · Desired output</h4><p class="hint">Drop pieces here. Drag a card to reorder it, or use its move buttons. Open a card to rename it, split, replace, omit, or merge earlier fields.</p><div id="block-fields" class="block-tray"></div></section>
      <section><h4>3 &#183; Final field order</h4><p class="hint">Drag tiles or use their arrow buttons. This order is used to generate the mapping. JSON consumers may display object keys in a different order.</p><ol id="block-order" class="block-order"></ol></section></div>
      <div class="actions"><button type="button" class="secondary" data-action="block-preview">Verify output preview</button>${mappingSuggestionButton('Building Blocks')}<button type="button" class="secondary" data-action="block-open">Open blocks in FieldSense</button><button type="button" class="secondary" data-action="workspace-ai-open" data-mode="fix" data-artifact="template" data-tab="Building Blocks">Draft template with Nexora AI</button><button type="button" class="primary" data-action="block-generate">Generate + validate parser pack</button></div><p id="block-status" role="status"></p><pre id="block-preview" class="json-output">{}</pre>`));
    if(!el('block-raw'))return;
    const raw=el('block-raw'),fields=el('block-fields');
    if(!state.blockSeed&&board?.raw===raw.value)board.fields.forEach(field=>addBuildingBlock(field.method==='constant',null,field));
    state.blockSeed='';
    renderBlockSource();
    renderBlockOrder();
    const order=el('block-order');
    order.addEventListener('dragover',event=>event.preventDefault());
    order.addEventListener('drop',event=>{
      event.preventDefault();
      const cardID=event.dataTransfer.getData('application/x-nli-card');
      const rows=[...fields.children],card=rows.find(row=>row.dataset.blockCard===cardID);
      const index=event.target.closest('[data-order-index]')?.dataset.orderIndex;
      if(card&&index!==undefined&&card!==rows[Number(index)]){
        const target=rows[Number(index)];
        if(rows.indexOf(card)<Number(index))target.after(card);
        else target.before(card);
        rememberBlockBoard();
        el('block-status').textContent='Field order updated.';
      }
    });
    raw.addEventListener('input',()=>{
      if(fields.children.length)el('block-status').textContent='Source changed. Restore the previous sample or reset the board before using its existing cards.';
      renderBlockSource();
      rememberBlockBoard();
    });
    el('block-mode').addEventListener('change',renderBlockSource);
    fields.addEventListener('input',rememberBlockBoard);
    fields.addEventListener('dragover',event=>{event.preventDefault();fields.classList.add('block-drop-ready');});
    fields.addEventListener('dragleave',event=>{if(!fields.contains(event.relatedTarget))fields.classList.remove('block-drop-ready');});
    fields.addEventListener('drop',event=>{
      event.preventDefault();
      fields.classList.remove('block-drop-ready');
      const token=event.dataTransfer.getData('application/x-nli-piece');
      const cardID=event.dataTransfer.getData('application/x-nli-card');
      const target=event.target.closest('[data-block-card]');
      if(token&&el('block-source').pieces[Number(token)-1]){
        try{addSourcePiece(Number(token)-1,target);}
        catch(error){el('block-status').textContent=error.message;}
      }
      else if(cardID){
        const card=[...fields.children].find(row=>row.dataset.blockCard===cardID);
        if(card&&card!==target)fields.insertBefore(card,target);
      }
      rememberBlockBoard();
    });
  }
  function renderBlockSource(){
    const raw=el('block-raw').value;
    const pattern=el('block-mode').value==='character'?/[\s\S]/gu:/[\p{L}\p{N}_]+|[^\p{L}\p{N}_\s]|\s+/gu;
    const pieces=[];
    for(const match of raw.matchAll(pattern)){
      pieces.push(match[0]);
      if(pieces.length>512)break;
    }
    const tray=el('block-source');
    tray.pieces=[];
    let start=0;
    tray.innerHTML=pieces.slice(0,512).map((value,index)=>{
      const end=start+value.length;
      tray.pieces.push({start,end,value});
      start=end;
      const visible=value.replaceAll(' ','␣').replaceAll('\t','⇥').replaceAll('\n','↵').replaceAll('\r','␍');
      return `<button class="block-piece" type="button" draggable="true" data-action="block-piece" data-index="${index}" title="Add source piece ${index+1}">${esc(visible)}</button>`;
    }).join('');
    if(pieces.length>512)tray.insertAdjacentHTML('beforeend','<p class="hint">First 512 pieces shown. Select text in the raw record to capture another range.</p>');
    tray.ondragstart=event=>{
      const piece=event.target.closest('[data-index]');
      if(piece)event.dataTransfer.setData('application/x-nli-piece',String(Number(piece.dataset.index)+1));
    };
  }
  function addSourcePiece(index,before=null){
    const range=el('block-source').pieces[index];
    if(range)addBuildingBlock(false,range,null,before);
  }
  function renderBlockOrder(actual=null){
    const list=el('block-order'),fields=el('block-fields');
    if(!list||!fields)return;
    const rows=[...fields.children];
    list.replaceChildren(...rows.map((row,index)=>{
      const name=row.querySelector('[data-block-name]')?.value||'Unnamed field';
      const policy=row.querySelector('[data-block-output]')?.value;
      const hasOutput=!!actual&&Object.hasOwn(actual,name);
      const sample=policy==='omit'||policy==='intermediate'?policy:hasOutput?JSON.stringify(actual[name],null,2):actual?(policy==='flatten'?'Keys merged into the output object':'No value in preview'):row.querySelector('.block-value')?.textContent||'';
      const item=document.createElement('li');
      item.dataset.orderIndex=index;
      item.draggable=true;
      item.innerHTML=`<code>${esc(name)}</code><small class="block-order-value">${esc(sample.slice(0,160))}${sample.length>160?'...':''}</small><div class="actions"><button class="secondary" type="button" data-direction="-1" aria-label="Move ${esc(name)} up" ${index===0?'disabled':''}>&#8593;</button><button class="secondary" type="button" data-direction="1" aria-label="Move ${esc(name)} down" ${index===rows.length-1?'disabled':''}>&#8595;</button></div>`;
      item.addEventListener('dragstart',event=>event.dataTransfer.setData('application/x-nli-card',row.dataset.blockCard));
      item.addEventListener('click',event=>{
        const button=event.target.closest('[data-direction]');
        if(!button)return;
        const direction=Number(button.dataset.direction),next=index+direction,target=rows[next];
        if(!target)return;
        if(direction<0)target.before(row);else target.after(row);
        rememberBlockBoard();
        const updated=list.children[next];
        (updated.querySelector(`[data-direction="${direction}"]:enabled`)||updated.querySelector('button:enabled'))?.focus();
        el('block-status').textContent=name+' moved to position '+(next+1)+'.';
      });
      return item;
    }));
  }
  function rememberBlockBoard(){
    const input=el('block-raw');
    if(!input)return;
    clearTimeout(input.blockPreviewTimer);
    input.blockPreviewRequest?.abort();
    try{state.blockBoard={raw:input.value,fields:buildingBlockPlan().fields};}
    catch{
      if(!el('block-fields').children.length)state.blockBoard={raw:input.value,fields:[]};
    }
    el('block-preview').textContent='{}';
    for(const row of el('block-fields').children){
      row.querySelector('.block-value').textContent=row.blockSample;
    }
    renderBlockOrder();
    if(can('assurance:run')&&[...el('block-fields').children].some(row=>isKVTail(row.blockSource)||row.blockSource.method==='key_value')){
      input.blockPreviewTimer=setTimeout(()=>{
        if(input.isConnected)void previewBuildingBlocks();
      },350);
    }
  }
  async function previewBuildingBlocks(){
    const input=el('block-raw'),status=el('block-status');
    clearTimeout(input.blockPreviewTimer);
    input.blockPreviewRequest?.abort();
    const controller=new AbortController();
    input.blockPreviewRequest=controller;
    status.textContent='Checking this board with the native mapping engine…';
    try{
      const plan=buildingBlockPlan();
      const result=await api('/assurance/mapping-preview',{method:'POST',body:{raw:input.value,plan},signal:controller.signal});
      if(controller.signal.aborted||!status.isConnected)return;
      const actual=result.actual||{};
      el('block-preview').textContent=pretty(actual);
      renderBlockOrder(result.passed?actual:null);
      if(result.passed){
        for(const row of el('block-fields').children){
          const name=row.querySelector('[data-block-name]').value.trim();
          if(Object.hasOwn(actual,name))row.querySelector('.block-value').textContent=JSON.stringify(actual[name],null,2);
        }
      }
      status.textContent=result.passed?'Native preview passed for this sample.':(result.errors||[]).join(' · ')||'Review the sample and field mappings.';
    }catch(error){
      if(!controller.signal.aborted&&status.isConnected)status.textContent=`Preview unavailable. The board is preserved. ${error.message}`;
    }finally{
      if(input.blockPreviewRequest===controller)input.blockPreviewRequest=null;
    }
  }
  function applyWorkspaceMapping(raw,plan,service='auto'){
    state.nfiRaw=raw;
    state.nfiService=service;
    state.nfiParser=null;
    state.nfiCases=[];
    state.nfiCandidateValidated=null;
    state.qrwFields=plan.fields.map(field=>qrwField(field));
    qrwControllerPreview=null;
  }
  function addBuildingBlock(literal,range=null,field=null,before=null){
    const raw=el('block-raw'), start=range?.start??raw.selectionStart, end=range?.end??raw.selectionEnd;
    if(!literal&&!field&&start===end)throw new Error('Select the exact characters to capture in the raw record.');
    const fields=el('block-fields'),row=document.createElement('div');
    if(fields.children.length>=128)throw new Error('The board supports up to 128 output fields.');
    const extracted=field&&!literal?qrwExtract(field,raw.value):null;
    const value=literal?field?.default_value??'':field?extracted.value??extracted.evidence??'':raw.value.slice(start,end);
    const sequence=++qrwFieldSequence;
    row.className='block-card';
    row.dataset.blockCard=String(sequence);
    row.draggable=true;
    row.addEventListener('dragstart',event=>event.dataTransfer.setData('application/x-nli-card',row.dataset.blockCard));
    row.blockSource=field|| (literal?{method:'constant',default_value:''}:{method:'character',source:String([...raw.value.slice(0,start)].length+1),source_end:String([...raw.value.slice(0,end)].length)});
    row.blockRaw=raw.value;
    row.blockSample=typeof value==='object'?pretty(value):String(value);
    row.innerHTML=`<label class="field">Field name<input data-block-name value="${esc(field?.name||'field_'+sequence)}"></label><code class="block-value">${esc(typeof value==='object'?pretty(value):String(value))}</code>
      <details ${field?'':'open'}><summary>Edit piece</summary><div class="form-grid"><label class="field">${literal?'Literal value':'Source value'}<input data-block-original value="${esc(value)}" ${literal?'':'readonly'}></label><label class="field">Value type <select data-block-type>${QRW_TYPES.map(option=>qrwOption(option,field?.type||'string')).join('')}</select></label><label class="field">Output <select data-block-output>${['include','omit','intermediate','flatten'].map(option=>qrwOption(option,field?.output_policy||'include')).join('')}</select></label><label class="field">Find literal<input data-block-find></label><label class="field">Replace with · leave empty to remove<input data-block-replacement></label><label class="field">Split delimiter<input data-block-delimiter></label><label class="field">Keep part · 1-based<input data-block-part type="number" min="1" value="1"></label><label class="field">Merge earlier fields · comma separated<input data-block-merge></label><label class="field">Merge separator<input data-block-separator value=" "></label></div></details>
      <div class="actions"><button type="button" class="secondary" data-action="block-up">Move up</button><button type="button" class="secondary" data-action="block-down">Move down</button><button type="button" class="danger" data-action="block-remove">Remove</button></div>`;
    fields.insertBefore(row,before);
    rememberBlockBoard();
  }
  function buildingBlockPlan(){
    const raw=el('block-raw').value,rows=[...el('block-fields').children];
    if(!raw.trim()||!rows.length)throw new Error('Paste a record and add at least one block.');
    const used=new Set();
    return qrwPlan(rows.map(row=>{
      const value=key=>row.querySelector(`[data-block-${key}]`)?.value||'';
      const name=value('name').trim(),source={...row.blockSource},operations=[...(source.operations||[])];
      if(!/^[a-zA-Z][a-zA-Z0-9_]{0,63}$/.test(name)||used.has(name))throw new Error('Give each block a unique field name beginning with a letter.');
      if(source.method==='constant')source.default_value=value('original');
      else if(row.blockRaw!==raw)throw new Error('The raw record changed. Recreate the blocks against the new text.');
      if(value('find'))operations.push({op:'replace',find:value('find'),replace:value('replacement')});
      if(value('delimiter'))operations.push({op:'split',delimiter:value('delimiter'),part:Number(value('part'))||1});
      const merge=value('merge').split(',').map(item=>item.trim()).filter(Boolean);
      if(merge.length){
        if(merge.some(field=>!used.has(field)))throw new Error('Merge can reference only earlier blocks.');
        operations.push({op:'merge',fields:merge,separator:value('separator')});
      }
      used.add(name);
      return {policy:'required',confirmed:true,...source,type:value('type'),name,label:name,output_policy:value('output'),operations};
    }));
  }
  function renderAIFieldGuidance(form,result){
    const previous=form.querySelector('[data-ai-field-guidance]');
    previous?.remove();
    const differences=result.validation?.diff||[];
    const fields=result.proposal?.plan?.fields||[];
    if(!fields.length&&!differences.length)return;
    const hints={
      missing:'Check whether this key is present in the raw sample. Add a key/value extractor for a named key, or a source-position selector for an unnamed value.',
      extra:'Check the expected field name and any aliases. Mark the field Omit only if you do not want this observed value in the output.',
      type:'Compare the actual and expected types. Keep IDs and values with leading zeros as strings unless you need numeric conversion.',
      value:'Check the exact source span, delimiter, and ordered operations. For key/value fields, verify the assignment separator, pair separator, and escaping.',
    };
    const section=document.createElement('section');
    section.dataset.aiFieldGuidance='';
    section.className='wide';
    section.innerHTML=`${differences.length?'<h4>Fields that need attention</h4>':''}<ul>${differences.slice(0,128).map(item=>{
      const field=fields.find(candidate=>candidate.name===item.field);
      return `<li><strong>${esc(item.field)}</strong> · ${esc(item.status)}${field?` · ${esc(field.method)}`:''}<br>${esc(hints[item.status]||'Compare this field with the original evidence.')}<br><code>Actual: ${esc(JSON.stringify(item.actual))} · Expected: ${esc(JSON.stringify(item.desired))}</code> <button type="button" class="secondary small" data-action="workspace-ai-field-help" data-field="${esc(item.field)}">Ask Nexora AI about this field</button></li>`;
    }).join('')}</ul>${fields.length?`<label>Get help with an observed field <select data-ai-observed-field>${Object.keys(result.validation?.actual||{}).slice(0,256).map(name=>`<option value="${esc(name)}">${esc(name)}</option>`).join('')}</select></label> <button type="button" class="secondary small" data-action="workspace-ai-field-help">Ask Nexora AI about selected field</button>`:''}`;
    form.querySelector('[data-workspace-ai-result]').after(section);
  }
  function aiActionFailure(error){
    const details=error.details||{};
    const findings=Array.isArray(details.errors)?details.errors.filter(item=>typeof item==='string').slice(0,4):[];
    const pathTip=details.error_code==='timeout'?'Try again or test a faster provider profile.':
      !details.error_code&&error.status===502?'Check Controller connectivity in API Monitor. This is a transport error; the provider reply was not received.':
      error.status===422?'Review the draft diagnostic and adjust the sample or draft type.':'';
    const next=typeof details.next_action==='string'?details.next_action:pathTip;
    return [error.message,...findings,next].filter(Boolean).join('\n');
  }
  async function runAndApplyWorkspaceAI(button){
    const form=button.closest('form');
    await runWorkspaceAI(button);
    const result=form.aiResult?.result;
    const canApply=result?.proposal_origin==='ai'||(form.applyNativeMapping&&result?.proposal_origin==='native_fallback');
    if(form.applyVerifiedMapping&&canApply&&result.validation?.extraction_valid===true&&result.validation?.desired_match!==false&&result.proposal?.plan?.fields?.length){
      await action({dataset:{action:'workspace-ai-import'},closest:selector=>form.closest(selector),applyVerifiedDraft:true});
    }
  }

  async function runWorkspaceAI(button){
    const form=button.closest('form');
    if(form.aiAutoSource)form.elements.raw.value=workspaceAISample(form.aiAutoSource);
    const data=formObject(form), raw=data.raw||'';
    const resultHost=form.querySelector('[data-workspace-ai-result]');
    for(const selector of ['workspace-ai-import','workspace-ai-download'])form.querySelector(`[data-action=${selector}]`).hidden=true;
    delete form.aiResult;
    form.querySelector('[data-ai-field-guidance]')?.remove();
    const requestID=form.aiRequestID=(form.aiRequestID||0)+1;
    button.disabled=true;
    resultHost.textContent='Contacting the configured provider…';
    try{
      if(form.aiAutoSource&&!raw.trim())throw new Error('Paste or inspect a sample before rebuilding the draft.');
      const helpDesk=form.hasAttribute('data-ai-help-desk');
      if(helpDesk&&!data.question?.trim())throw new Error('Enter a question about NLI.');
      const workspace=helpDesk&&Object.hasOwn(TITLES,data.help_workspace)?data.help_workspace:state.page;
      const tab=helpDesk?(data.help_tab||'').trim():form.dataset.tab;
      const question=helpDesk&&data.help_issue?.trim()?data.question+'\n\nObserved issue or result: '+data.help_issue.trim():data.question||'';
      const controls=helpDesk?[]:[...page.querySelectorAll('button[data-action]')].filter(node=>!node.closest('#workspace-ai-dialog,[hidden],.workspace-ai-actions')).map(node=>node.dataset.action).filter(Boolean);
      const mapping=raw&&page.querySelector('#qrw-field-rows')&&qrwRaw()===raw;
      let boardPlan=null,boardIssue='';
      if(state.page==='building-blocks'&&raw===el('block-raw')?.value&&el('block-fields')?.children.length){
        try{boardPlan=buildingBlockPlan();}
        catch(error){boardIssue='Current board was not used as context: '+error.message+' The draft uses the raw record; the board is unchanged.';}
      }
      const currentAssurance=state.page==='assurance'&&state.lastAssurance?.request?.raw===raw?state.lastAssurance.request:null;
      const assuranceForm=el('assurance-form');
      const sourceForm=form.aiAutoSource?.closest('form')||(assuranceForm?.elements.raw?.value===raw?assuranceForm:null);
      const selectedService=sourceForm?.elements.service?.value==='custom'?sourceForm.elements.custom_service?.value:sourceForm?.elements.service?.value;
      const service=mapping?qrwScope().service_scope.split(',')[0].trim():currentAssurance?.service||selectedService||'auto';
      const useAssuranceExpectation=['mapping','parser'].includes(data.artifact);
      const expectedInput=data.desired?.trim()||(useAssuranceExpectation&&sourceForm===assuranceForm?assuranceForm?.elements.desired?.value?.trim():'');
      const desired=expectedInput?parseJSON(expectedInput,'Expected output JSON'):(useAssuranceExpectation?currentAssurance?.desired:null)||{};
      if(!desired||Array.isArray(desired)||typeof desired!=='object')throw new Error('Expected mapping output must be a JSON object.');
      const mappingSignature=mapping?JSON.stringify(qrwPlan()):null;
      const result=await api('/ai/workspace',{method:'POST',body:{mode:form.dataset.mode,workspace,tab,question,artifact:data.artifact||'',raw,desired,evidence_source:aiEvidenceSource(raw,data.evidence_source),service:service==='*'?'auto':service,controls:[...new Set(controls)].slice(0,24),...(form.dataset.mode==='fix'&&data.artifact?{native_fallback:true}:{}),...(boardPlan?{plan:boardPlan}:mapping&&state.qrwFields.length?{plan:qrwPlan()}:{}),...(data.reference?{reference:JSON.parse(data.reference)}:{})}});
      if(!form.isConnected||form.aiRequestID!==requestID||form.elements.raw.value!==raw)return;
      if(helpDesk&&JSON.stringify(formObject(form))!==JSON.stringify(data)){
        resultHost.textContent='The question or context changed. Submit the current question again.';
        return;
      }
      if(form.aiAutoSource&&workspaceAISample(form.aiAutoSource)!==raw){
        resultHost.textContent='The sample changed while the AI draft was building. Rebuild the draft for the current sample.';
        return;
      }
      if(mappingSignature!==null&&(qrwRaw()!==raw||JSON.stringify(qrwPlan())!==mappingSignature)){
        resultHost.textContent='The raw evidence or mapping changed. Build a new draft from the current editor.';
        return;
      }
      form.aiResult={result,raw,mappingSignature};
      resultHost.textContent=result.proposal?.plan
        ?[boardIssue,result.answer,result.next_action, 'Exact sample preview', pretty(result.validation?.actual||{}), ...(result.validation?.errors||[])].filter(Boolean).join('\n\n')
        :result.proposal?[result.proposal_origin==='native_fallback'?'Verified native draft':result.answer,
          result.next_action,pretty(result.validation?.actual??result.proposal.config??result.proposal)].filter(Boolean).join('\n\n')
        :result.answer||pretty(result);
      renderAIFieldGuidance(form,result);
      const apply=form.querySelector('[data-action=workspace-ai-import]');
      apply.hidden=!(result.proposal?.plan?.fields?.length||result.proposal?.config);
      apply.textContent=result.proposal?.config?'Open draft in its editor':'Apply mapping to editor';
      form.querySelector('[data-action=workspace-ai-download]').hidden=!result.proposal;
    }catch(error){if(form.aiRequestID===requestID)resultHost.textContent=aiActionFailure(error);}
    finally{button.disabled=false;}
  }

  function setupAITabs() {
    const existing=[...page.querySelectorAll(':scope > .panel')];if(!existing.length)return;
    const categories=[['provider','Provider'],['status','Status'],['results','Results'],['parser','Parser'],['regex','Regex'],['template','Template'],['approval','Approval']];
    const tablist=document.createElement('div');tablist.className='ai-subnav';tablist.setAttribute('role','tablist');tablist.setAttribute('aria-label','AI resolution categories');
    const sections=categories.map(([key,label],index)=>{const button=document.createElement('button');button.type='button';button.className='tab';button.id=`ai-tab-${key}`;button.dataset.aiTab=key;button.setAttribute('role','tab');button.setAttribute('aria-controls',`ai-panel-${key}`);button.setAttribute('aria-selected',String(index===0));button.tabIndex=index===0?0:-1;button.textContent=label;tablist.append(button);const section=document.createElement('section');section.className='ai-tab-panel';section.id=`ai-panel-${key}`;section.dataset.aiPanel=key;section.setAttribute('role','tabpanel');section.setAttribute('aria-labelledby',button.id);section.tabIndex=0;section.hidden=index!==0;return section;});
    const [provider,status,results,parser,regex,template,approval]=sections;
    for(const panelNode of existing){const title=panelNode.querySelector('h3')?.textContent?.toLowerCase()||'';if(title.includes('ai connection manager')||title.includes('provider template')||title.includes('api key'))provider.append(panelNode);else if(title.includes('connection status'))status.append(panelNode);else if(title.includes('ai remediation'))results.append(panelNode);else approval.append(panelNode);}
    const proposalViews=(key,label,select)=>{const rows=(state.aiProposals||[]).map(doc=>{const p=doc.payload||{},value=select(p);return value==null||value===''?null:`<article class="card ai-category-result"><div class="section-head"><div><h3>${esc(p.name||doc.id)}</h3><p>${esc(doc.state||p.review_state||'draft')} · ${esc(label)}</p></div></div><pre class="json-output">${esc(typeof value==='string'?value:pretty(value))}</pre></article>`;}).filter(Boolean);return `<div class="callout">${esc(label)} extracted from controller proposals. Suggestions remain review-only until validated and approved.</div>${rows.join('')||empty(`No ${label.toLowerCase()} suggestions are available yet.`)}`;};
    parser.innerHTML=proposalViews('parser','Parser proposals',p=>p.proposal||p.parser);regex.innerHTML=proposalViews('regex','Regex suggestions',p=>p.proposal?.pattern||p.proposal?.regex||p.regex);template.innerHTML=proposalViews('template','Template suggestions',p=>p.proposal?.template||p.template);
    page.prepend(tablist);for(const section of sections)page.append(section);
    tablist.addEventListener('click',event=>{const button=event.target.closest('[data-ai-tab]');if(!button)return;sections.forEach(item=>item.hidden=item.dataset.aiPanel!==button.dataset.aiTab);tablist.querySelectorAll('[data-ai-tab]').forEach(item=>{const active=item===button;item.setAttribute('aria-selected',String(active));item.tabIndex=active?0:-1;});});
  }

  function isReplayGateNotification(note) {
    const text=[note?.title,note?.resource,note?.action,note?.message,note?.details].filter(Boolean).join(' ');
    return /(?:system issue:\s*post|post).*\/api\/v1\/quarantine\/reprocess/i.test(text)||(/quarantine\/reprocess/i.test(text)&&/acknowledg|bundle|gated|conflict|409|530/i.test(text));
  }
  function activeControllerNotifications() { return (state.controllerNotifications||[]).filter(note=>!isReplayGateNotification(note)); }
  function renderNotificationBell() {
    const controller=activeControllerNotifications().slice(0,30),local=(state.notifications||[]).slice(0,30),unread=controller.filter(note=>!note.read&&note.status!=='read').length;
    const count=el('notification-count');count.textContent=unread>99?'99+':String(unread);count.hidden=unread===0;
    const serverRows=controller.map(note=>`<article class="notification-item ${note.severity==='critical'?'error':''} ${note.read||note.status==='read'?'read':''}" data-notification-id="${esc(note.id)}"><span class="notification-dot" aria-hidden="true"></span><div class="notification-body"><b>${esc(note.title||'Notification')}</b><p>${esc(note.resource||note.action||'Controller event')}</p><small>${time(note.timestamp)}</small><div class="notification-actions">${note.read||note.status==='read'?`<button class="ghost small" type="button" data-notification-action="unread" data-id="${esc(note.sequence)}">Mark unread</button>`:`<button class="secondary small" type="button" data-notification-action="read" data-id="${esc(note.sequence)}">Mark read</button>`}<button class="ghost small" type="button" data-notification-action="snooze" data-id="${esc(note.sequence)}">Check later</button><button class="danger small" type="button" data-notification-action="dismissed" data-id="${esc(note.sequence)}">Delete</button></div></div></article>`);
    const localRows=local.map(note=>`<article class="notification-item ${esc(note.type)}"><span class="notification-dot" aria-hidden="true"></span><div class="notification-body"><b>${esc(note.type==='progress'?'In progress':note.type==='error'?'Failed':note.type==='warning'?'Warning':'Completed')}</b><p>${esc(note.message)}</p><small>${time(note.at)}</small></div></article>`);
    el('notification-list').innerHTML=[...serverRows,...localRows].join('')||'<p class="muted">No unread or active notifications.</p>';
  }
  async function refreshNotificationBell() {
    if(!state.user||state.notificationRefreshInFlight)return;
    state.notificationRefreshInFlight=true;
    try{const data=await api('/notifications?limit=30');state.controllerNotifications=data.notifications||[];renderNotificationBell();}
    catch{/* Keep the last authoritative controller notification snapshot. */}
    finally{state.notificationRefreshInFlight=false;}
  }
  async function updateNotificationState(sequence,action) {
    const id=encodeURIComponent(sequence),body={action};if(action==='snooze')body.snooze_minutes=60;
    await api(`/notifications/audit-${id}`,{method:'POST',body});await refreshNotificationBell();if(state.page==='notifications')await loadNotifications();
  }
  function notify(message, type = 'success') {
    if(type==='error'&&[...el('toast-region').children].some(node=>node.classList.contains('bad')&&node.textContent===String(message))){
      return {done:text=>notify(text,'success'),fail:text=>notify(text,'error')};
    }
    const entry={ message: String(message), type, at: new Date().toISOString() };
    state.notifications.unshift(entry); state.notifications = state.notifications.slice(0, 30);
    renderNotificationBell();
    const node = document.createElement('div'); node.className = `toast${type === 'error' ? ' bad' : type === 'progress' ? ' progress' : type === 'warning' ? ' warning' : ''}`; node.setAttribute('role', type === 'error' ? 'alert' : 'status'); node.textContent = message;
    el('toast-region').append(node); setTimeout(() => node.remove(), 4500);
    const finish=(text,result)=>{
      if(type==='progress'){
        node.remove();
        state.notifications=state.notifications.filter(note=>note!==entry);
      }
      return notify(text,result);
    };
    return { done: text => finish(text, 'success'), fail: text => finish(text, 'error') };
  }
  const toast = (message, bad = false) => notify(message, bad ? 'error' : 'success');
  function replayGateMessage(error,exactBundleAcknowledged=false) {
    const message=String(error?.message||'');
    if(error?.details?.code==='parser_bundle_not_acknowledged'){
      const {sync_status, sync_error, next_action}=error.details;
      return ['Replay is waiting for the controller to verify the origin client.',sync_status?`Client sync status: ${sync_status}.`:'',sync_error||'',next_action||''].filter(Boolean).join(' ');
    }
    if(/published parser bundle changed/i.test(message))return 'The published bundle changed. Refresh the wizard and check the origin client acknowledgement again.';
    if(/has not acknowledged|has not applied|acknowledgement.*(?:missing|unavailable)/i.test(message))return exactBundleAcknowledged?'The controller has not confirmed the client’s current bundle. Use Step 8.1 to verify the origin client again.':'Waiting for the origin client to acknowledge the exact published parser bundle. Replay remains safely gated; live telemetry is still active.';
    return '';
  }
  function setReplayStatus(form,message,type='warn') {
    if(!form)return;let status=form.querySelector('[data-replay-status]');if(!status){status=document.createElement('div');status.dataset.replayStatus='';status.setAttribute('role','status');status.className='callout wide';form.prepend(status);}
    status.hidden=false;status.className=`callout ${type==='success'?'':type}`.trim();status.textContent=message;
  }
  const CONTROLLER_REQUIRED = 'This is a visual-only WebView copy. Open the same-origin URL served by nli-controller-agent.';
  async function fetchWithTimeout(url, options = {}, timeoutMs = 5000) {
    const controller = new AbortController();
    const externalSignal = options.signal;
    let timedOut = false;
    const forwardAbort = () => controller.abort(externalSignal?.reason);
    if (externalSignal?.aborted) forwardAbort();
    else externalSignal?.addEventListener('abort', forwardAbort, { once: true });
    const timer = setTimeout(() => { timedOut = true; controller.abort(); }, timeoutMs);
    try { return await fetch(url, { ...options, signal: controller.signal }); }
    catch (error) {
      if (timedOut && error?.name === 'AbortError') {
        const timeout = new Error(`Request timed out after ${timeoutMs} ms`);
        timeout.name = 'TimeoutError';
        throw timeout;
      }
      throw error;
    }
    finally { clearTimeout(timer); externalSignal?.removeEventListener('abort', forwardAbort); }
  }
  async function ensureCSRF() {
    if (state.csrf) return state.csrf;
    const response = await fetchWithTimeout('/api/v1/auth/csrf', { credentials: 'same-origin', headers: { Accept: 'application/json' } });
    if (!response.ok) throw new Error('Unable to establish CSRF protection');
    state.csrf = (await response.json()).csrf_token; return state.csrf;
  }
  const inFlightReads = new Map();
  function api(path, options = {}) {
    if ((options.method || 'GET') !== 'GET' || options.signal || options.headers || options.body !== undefined) {
      return requestAPI(path, options);
    }
    const key = JSON.stringify([state.user?.username, path, options.timeoutMs]);
    const existing = inFlightReads.get(key);
    if (existing) return existing;
    const request = requestAPI(path, options).finally(() => {
      if (inFlightReads.get(key) === request) inFlightReads.delete(key);
    });
    inFlightReads.set(key, request);
    return request;
  }
  async function requestAPI(path, options = {}) {
    if (path === '/auth/login' && !state.controllerReady) throw new Error(CONTROLLER_REQUIRED);
    const method = options.method || 'GET'; const headers = { Accept: 'application/json', ...(options.headers || {}) };
    let body = options.body;
    if (body !== undefined && !(body instanceof FormData)) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(body); }
    if (!['GET', 'HEAD', 'OPTIONS'].includes(method) && path !== '/auth/login') headers['X-CSRF-Token'] = await ensureCSRF();
    const started = performance.now();
    let response;
    try { response = await fetchWithTimeout('/api/v1' + path, { method, credentials: 'same-origin', headers, body, signal: options.signal }, options.timeoutMs ?? 55000); }
    catch (error) { recordAPI(path, method, null, performance.now() - started, error.message); throw error; }
    const contentType = response.headers.get('content-type') || '';
    const data = contentType.includes('application/json') ? await response.json().catch(() => ({})) : {};
    const latency=performance.now()-started;
    const degraded=data?.snapshot_stale===true||path.split('?')[0]==='/admin/ai-provider'&&['degraded','unavailable','encryption_unavailable'].includes(data?.store_status);
    const standby=response.ok&&data.provider_routing==='standby'&&typeof data.provider_id==='string'
      ?`Nexora AI switched to standby profile ${data.provider_id}. Review the primary profile in AI Resolution & Providers.`:'';
    if(standby&&!state.notifications.some(note=>note.message===standby&&Date.now()-Date.parse(note.at)<60000)){
      notify(standby,'warning');
    }
    const mappingOutcome=standby||(path==='/assurance/mapping-preview'&&data.passed===false?'Mapping rejected by native validation'
      :data.proposal_origin==='native_fallback'?'Verified native fallback returned':'');
    recordAPI(path, method, response.status, latency, data?.snapshot_stale===true?'Serving last confirmed metadata snapshot':degraded?'Provider storage needs attention':mappingOutcome||(response.ok?'':response.statusText),degraded);
    if ([405, 501].includes(response.status) || (!contentType.includes('application/json') && !response.ok && ![502,504].includes(response.status))) throw new Error(CONTROLLER_REQUIRED);
    if (response.status === 401 && path !== '/auth/login') { signOut(false); throw new Error('Session expired'); }
    if (!response.ok) { const error=new Error(data.error || (response.status===502?'The request gateway could not reach the controller. Check the controller connection and retry.':response.status===504?'The request timed out. Check the controller connection and retry.':`Request failed (HTTP ${response.status}).`));error.status=response.status;error.details=data;throw error; }
    if (!contentType.includes('application/json')) throw new Error('The controller returned an invalid API response.');
    if (data?.snapshot_stale===true && !options.allowStale) {
      const error=new Error(data.snapshot_error || 'Live controller metadata is temporarily unavailable.');
      error.status=503;
      throw error;
    }
    return data;
  }
  function recordAPI(path, method, status, latency, error = '', degraded=false) {
    const safePath = String(path).split('?')[0].replace(/\/[0-9a-f]{8}-[0-9a-f-]{27,}/ig, '/{id}').replace(/\/[A-Za-z0-9_-]{24,}(?=\/|$)/g, '/{id}');
    state.apiHistory.unshift({ path: safePath, method, status, degraded, latency: Math.round(latency), error: String(error || '').slice(0, 180), at: new Date().toISOString() });
    state.apiHistory.length = Math.min(state.apiHistory.length, 500);
  }
  function parseJSON(value, label) { try { return JSON.parse(value); } catch (error) { throw new Error(`${label}: ${error.message}`); } }
  function formObject(form) { return Object.fromEntries(new FormData(form).entries()); }
  function setControllerControls(enabled) {
    el('login-form').querySelectorAll('[data-controller-control]').forEach(control => { control.disabled = !enabled; });
  }
  function setControllerChecking() {
    state.controllerReady = false;
    el('login-workspaces').textContent=String(NAV.length);
    el('login-roles').textContent='—';
    el('login-parsers').textContent='—';
    el('login-form').classList.remove('controller-unavailable');
    el('standalone-hint').hidden = true;
    el('login-error').textContent = '';
    el('login-button-label').textContent = 'Checking controller…';
    setControllerControls(false);
  }
  function enableLogin() {
    state.controllerReady = true;
    el('login-form').classList.remove('controller-unavailable');
    el('standalone-hint').hidden = true;
    el('login-error').textContent = '';
    el('login-button-label').textContent = 'Sign in securely';
    setControllerControls(true);
  }
  function showPreviewOnly(message) {
    state.controllerReady = false;
    const form = el('login-form');
    const hint = el('standalone-hint');
    form.classList.add('controller-unavailable');
    hint.hidden = false;
    hint.querySelector('span').textContent = message || CONTROLLER_REQUIRED;
    el('login-button-label').textContent = 'Controller required';
    el('login-error').textContent = '';
    setControllerControls(false);
  }
  async function requireController() {
    const probeOptions = { credentials: 'same-origin', headers: { Accept: 'application/json' }, cache: 'no-store' };
    const [healthResponse, loginResponse] = await Promise.all([
      fetchWithTimeout('/api/v1/health', probeOptions, 12000),
      fetchWithTimeout('/api/v1/auth/login', { ...probeOptions, method: 'OPTIONS' }, 12000),
    ]);
    const contentType = healthResponse.headers.get('content-type') || '';
    const health = contentType.includes('application/json') ? await healthResponse.json().catch(() => null) : null;
    const identity = healthResponse.headers.get('x-nli-controller');
    if (!healthResponse.ok || identity !== 'v21' || health?.service !== 'nli-controller-agent' || health?.api_version !== 'v1') {
      throw new Error(health?.error || 'This address is not a Nexora Log Intelligence Ver21 controller. Open the Ver21 WebView or configure the controller connection.');
    }
    state.proxy = healthResponse.headers.get('x-nli-upstream') === 'controller';
    const allow = loginResponse.headers.get('allow') || '';
    if (!loginResponse.ok || loginResponse.headers.get('x-nli-controller') !== 'v21' || !allow.split(',').map(value => value.trim().toUpperCase()).includes('POST')) {
      throw new Error('The NLI controller does not advertise a compatible login API. Upgrade the controller RPM.');
    }
    return health;
  }
  async function checkController() {
    if (state.controllerCheckInFlight) return;
    state.controllerCheckInFlight = true;
    setControllerChecking();
    try {
      const health=await requireController();
      for (const [id,key] of [['login-roles','role_count'],['login-parsers','builtin_parser_count']]) {
        const count=health.inventory?.[key];
        el(id).textContent=Number.isSafeInteger(count)&&count>=0?String(count):'—';
      }
      enableLogin();
      // An anonymous session is expected on the login page. Probe it directly
      // so a normal 401 cannot call api()'s session-expiry handler and erase
      // credentials that an operator entered while controller checks finished.
      try {
        const sessionResponse = await fetchWithTimeout('/api/v1/auth/me', {
          credentials: 'same-origin',
          headers: { Accept: 'application/json' },
          cache: 'no-store',
        });
        if (sessionResponse.ok && (sessionResponse.headers.get('content-type') || '').includes('application/json')) {
          await boot(await sessionResponse.json());
        }
      } catch {}
    } catch (error) {
      showPreviewOnly(error.name === 'AbortError' ? 'Controller verification timed out. Open the URL served by nli-controller-agent.' : error.message);
    } finally {
      state.controllerCheckInFlight = false;
    }
  }

  function buildNav() {
    let group = '';
    el('nav').innerHTML = NAV.map(([section, id, title, icon]) => {
      const heading = section !== group ? `<div class="nav-group">${esc(section)}</div>` : ''; group = section;
      const count = NAV_COUNT_PAGES.has(id) ? `<span id="${id==='rollouts'?'approval-badge':`nav-count-${id}`}" class="nav-count" data-nav-count="${id}" aria-label="${esc(title)} pending actions" hidden>0</span>` : '';
      return `${heading}<button class="nav-item" data-page="${id}" data-testid="nav-${id}"><span class="nav-icon">${esc(icon)}</span><span class="nav-label">${esc(title)}</span>${count}</button>`;
    }).join('');
    updateNavigationBadges();
  }
  function updateNavigationBadges(counts=state.navCounts) {
    state.navCounts=counts&&typeof counts==='object'?counts:{};
    for(const pageID of NAV_COUNT_PAGES){const node=document.querySelector(`[data-nav-count="${pageID}"]`);if(!node)continue;const raw=state.navCounts[pageID];if(raw?.available===false){node.classList.add('unavailable');node.title='Pending action count is temporarily unavailable; the last confirmed count is shown when available';continue;}node.classList.remove('unavailable');const value=Math.max(0,Number(raw?.total??raw)||0);node.textContent=value>99?'99+':String(value);node.hidden=value===0;node.title=value?`${value} current operator action${value===1?'':'s'} required`:'';}
    state.pendingApprovals=Math.max(0,Number(state.navCounts.rollouts?.total??state.navCounts.rollouts)||0);
  }
  function updateApprovalBadge(value=state.pendingApprovals) { updateNavigationBadges({...state.navCounts,rollouts:{total:Math.max(0,Number(value)||0)}}); }
  async function refreshNavigationBadges() {
    if(!state.user||state.navCountsInFlight)return;
    state.navCountsInFlight=true;
    try{const data=await api('/navigation/counts');updateNavigationBadges(data.counts||{});}catch{/* Retain the last authoritative badge snapshot during transient recovery. */}
    finally{state.navCountsInFlight=false;}
  }
  const refreshApprovalBadge = refreshNavigationBadges;
  function markActive() { document.querySelectorAll('.nav-item').forEach(node => node.classList.toggle('active', node.dataset.page === state.page)); }
  async function navigate(id) {
    if (!state.user) return;
    state.page = TITLES[id] ? id : 'operations';
    const requestedPage = state.page;
    const navigation = Symbol(requestedPage);
    const started = performance.now();
    state.navigationRequest = navigation;
    state.loadingPage = true;
    const navEntry = NAV.find(([, pageID]) => pageID === state.page);
    el('page-title').textContent = TITLES[state.page];
    el('page-kicker').textContent = `${navEntry?.[0] || 'Control plane'} workspace`;
    markActive(); el('sidebar').classList.remove('open');
    page.removeAttribute('data-page');
    page.innerHTML = loading();
    try {
      await LOADERS[requestedPage]();
      refreshNavigationBadges();
      if (state.page === requestedPage && state.user) { applyPermissions(); state.loadedAt = Date.now(); if(SYNC_PAGES.has(requestedPage))state.sync.renderedSignature=syncRenderSignature(); updateFreshness(); }
    } catch (error) { if (state.page === requestedPage && state.user) { renderPage(requestedPage,errorState(error));applyPermissions();notify(`${TITLES[requestedPage]} could not refresh: ${error.message}`, 'error'); } }
    finally {
      if (state.navigationRequest === navigation) {
        state.loadingPage = false;
        state.navigationHistory ||= [];
        state.navigationHistory.unshift({ page: requestedPage, latency: Math.round(performance.now() - started), at: new Date().toISOString() });
        state.navigationHistory.length = Math.min(state.navigationHistory.length, 100);
      }
    }
  }
  function updateFreshness() {
    const sync=state.sync.snapshot,stamp=sync?.generated_at||sync?.aggregate?.nli_telemetry_last_success_timestamp;
    const canonicalAge=stamp&&Number.isFinite(Date.parse(stamp))?Math.max(0,Math.floor((Date.now()-Date.parse(stamp))/1000)):null,transportAge=state.sync.lastReceivedAt?Math.max(0,Math.floor((Date.now()-state.sync.lastReceivedAt)/1000)):null,age=transportAge??canonicalAge??(state.loadedAt?Math.floor((Date.now()-state.loadedAt)/1000):null),target=Math.max(250,Number(sync?.target_interval_ms||1000));
    // The page transport is authoritative for global freshness. An idle source can
    // have an old event timestamp while the controller connection remains healthy;
    // source-level freshness continues to expose that condition independently.
    const freshness=age==null?'unavailable':age*1000<=Math.max(3000,target*3)?'live':age*1000<=Math.max(15000,target*15)?'reconnecting':'stale';
    el('freshness').textContent=age===null?'Not loaded':`${freshness} · ${age}s old`;
    const stale=freshness==='stale';document.body.classList.toggle('stale',stale);
    const banner=page.querySelector('.stale-banner');
    if(stale&&!banner)page.insertAdjacentHTML('afterbegin',`<div class="stale-banner">Live synchronization is unavailable (${age}s since the controller last answered). The shared background recovery path is retrying automatically.</div>`);
    if(!stale&&banner)banner.remove();
  }

  const SYNC_SCHEMA='nli.sync.v1';
  const SYNC_PAGES=new Set(['operations','monitoring','streams','performance','sync-fabric','nhle-monitor','napi-monitor','nal-monitor']);
  const SYNC_VOLATILE_KEYS=new Set(['revision','generated_at','emitted_at','updated_at','controller_received_at','latest_proposal_at','latest_run_at','last_seen','last_activity_at','nli_telemetry_last_success_timestamp','freshness_age_ms','controller_processing_ms','observed_propagation_delay_ms','sequence','accepted','duplicate','out_of_order','sequence_gaps','reconnects','coalesced','rejected','missed_deadlines','clock_skew']);
  function syncRenderSignature(value){
    const stable=input=>Array.isArray(input)?input.map(stable):input&&typeof input==='object'?Object.fromEntries(Object.entries(input).filter(([key])=>!SYNC_VOLATILE_KEYS.has(key)).sort(([a],[b])=>a.localeCompare(b)).map(([key,item])=>[key,stable(item)])):input;
    return `${state.page}:${JSON.stringify(stable(state.sync.snapshot||{}))}`;
  }
  const present=value=>value!==undefined&&value!==null&&value!=='';
  const metricText=(value,unit='')=>present(value)&&Number.isFinite(Number(value))?`${typeof value==='bigint'?value.toLocaleString():typeof value==='string'&&/^\d+$/.test(value)?BigInt(value).toLocaleString():number(value)}${unit}`:'—';
  const fixedMetric=(value,digits=1,unit='')=>present(value)&&Number.isFinite(Number(value))?`${Number(value).toFixed(digits)}${unit}`:'—';
  const sourceValue=(source,key)=>source?.exact_counts?.[key]??source?.[key]??source?.counters?.[key]??source?.rates?.[key]??source?.queue?.[key];
  function syncStatus(value) { return String(value||'not reported').replaceAll('_',' '); }
  function syncStatusBadge(value) { return badge(syncStatus(value)); }
  function recordSyncHistory(snapshot) {
    const current=snapshot.aggregate||{},previous=state.sync.history.at(-1)?.totals||{},delta=key=>{if(!present(previous[key]))return 0;const value=Number(current[key]||0)-Number(previous[key]||0);return value>=0?value:0;};
    state.sync.history.push({timestamp:snapshot.generated_at||new Date().toISOString(),received:delta('nli_source_records_read_total'),parsed:delta('nli_parser_records_succeeded_total'),quarantine:delta('nli_quarantined_records_total'),errors:delta('nli_parser_records_failed_total'),latency:Number(current.nli_telemetry_propagation_delay_seconds_p95||0)*1000,totals:{...current}});
    state.sync.history=state.sync.history.slice(-120);
  }
  function fetchCanonicalSnapshot(options={}) {
    if(state.sync.snapshotPromise)return state.sync.snapshotPromise;
    const shared=api('/sync/snapshot',options).finally(()=>{if(state.sync.snapshotPromise===shared)state.sync.snapshotPromise=null;});
    state.sync.snapshotPromise=shared;
    return shared;
  }
  function requestSyncResnapshot(reason='revision gap') {
    if(state.sync.resyncing||!state.user)return;state.sync.resyncing=true;state.streamMode='resynchronizing';el('stream-state').innerHTML='<i></i>Resynchronizing';
    fetchCanonicalSnapshot().then(snapshot=>{state.sync.snapshot=null;acceptSyncSnapshot(snapshot,{force:true});}).catch(error=>notify(`Canonical resynchronization failed after ${reason}: ${error.message}`,'error')).finally(()=>{state.sync.resyncing=false;});
  }
  async function pollCanonicalSnapshot(reason='background recovery') {
    if(!state.user||state.sync.pollInFlight)return false;
    state.sync.pollInFlight=true;const controller=new AbortController();state.sync.pollController=controller;
    try{
      const snapshot=await fetchCanonicalSnapshot({signal:controller.signal});
      if(!state.user||controller.signal.aborted)return false;
      acceptSyncSnapshot(snapshot,{force:false,eventType:'sync_poll'});
      state.sync.pollFailures=0;
      if(state.streamMode!=='live')state.streamMode='polling';
      el('stream-state').innerHTML='<i></i>Live sync · recovery';
      return true;
    }catch(error){
      if(controller.signal.aborted)return false;
      state.sync.pollFailures++;
      // Abort is a normal consequence of sign-out, replacement, or browser lifecycle
      // cancellation. Keep retrying silently; only real failures become notifications.
      if(error?.name==='AbortError'||error?.name==='CanceledError')return false;
      if(state.sync.pollFailures===1||state.sync.pollFailures%10===0)notify(`Live synchronization recovery failed (${reason}): ${error.message}`,'error');
      return false;
    }finally{if(state.sync.pollController===controller)state.sync.pollController=null;state.sync.pollInFlight=false;updateFreshness();}
  }
  function startSyncFallback() {
    if(state.streamPollTimer)return;
    state.streamPollTimer=setInterval(()=>{
      if(!state.user||document.hidden)return;
      const target=Math.max(1000,Number(state.sync.snapshot?.target_interval_ms||1000));
      const age=state.sync.lastReceivedAt?Date.now()-state.sync.lastReceivedAt:Number.POSITIVE_INFINITY;
      if(state.streamMode!=='live'||age>Math.max(3000,target*3))pollCanonicalSnapshot(state.streamMode==='live'?'heartbeat overdue':'stream reconnecting');
    },1000);
  }
  function acceptSyncSnapshot(raw,{incremental=false,eventType='sync_snapshot',force=false}={}) {
    const incoming=raw?.snapshot||raw;if(!incoming||typeof incoming!=='object')return false;
    if(incoming.schema_version&&incoming.schema_version!==SYNC_SCHEMA){state.sync.rejectedUpdates++;return false;}
    let heartbeat=false;
    if(incremental){
      const current=state.sync.snapshot||{schema_version:SYNC_SCHEMA,revision:0,sources:[],aggregate:{},diagnostics:{},subsystems:{}};
      if(Number(incoming.revision||0)<=Number(current.revision||0)){state.sync.rejectedUpdates++;return false;}
      if(incoming.resync_required||Number(incoming.revision||0)>Number(current.revision||0)+1){state.sync.rejectedUpdates++;requestSyncResnapshot(incoming.resync_reason||'revision gap');return false;}
      if(!incoming.aggregate||typeof incoming.aggregate!=='object'){state.sync.rejectedUpdates++;requestSyncResnapshot('delta missing authoritative aggregate');return false;}
      const sample=incoming.source||incoming.sample,rows=[...(current.sources||[])],key=`${sample?.node_id||''}\u0000${sample?.source||''}`,index=rows.findIndex(item=>`${item.node_id||''}\u0000${item.source||''}`===key);
      if(sample){if(index<0)rows.push(sample);else rows[index]={...rows[index],...sample};}
      const merged={...current,...incoming,sources:rows,aggregate:incoming.aggregate,diagnostics:{...(current.diagnostics||{}),...(incoming.diagnostics||{})},subsystems:{...(current.subsystems||{}),...(incoming.subsystems||{})}};
      state.sync.snapshot=merged;
    }else{
      const current=state.sync.snapshot;
      if(!force&&state.sync.snapshot&&present(incoming.revision)){
        const nextRevision=Number(incoming.revision),currentRevision=Number(state.sync.snapshot.revision||0);
        if(nextRevision<currentRevision){state.sync.rejectedUpdates++;return false;}
        heartbeat=nextRevision===currentRevision;
      }
      if(!incoming.schema_version)incoming.schema_version=SYNC_SCHEMA;
      state.sync.snapshot=(heartbeat||eventType==='sync_heartbeat')?{
        ...current,...incoming,
        aggregate:incoming.aggregate&&typeof incoming.aggregate==='object'?incoming.aggregate:(current?.aggregate||{}),
        diagnostics:{...(current?.diagnostics||{}),...(incoming.diagnostics||{})},
        sources:Array.isArray(incoming.sources)?incoming.sources:(current?.sources||[]),
        subsystems:{...(current?.subsystems||{}),...(incoming.subsystems||{})},
      }:{...incoming,aggregate:incoming.aggregate&&typeof incoming.aggregate==='object'?incoming.aggregate:{}};
    }
    const emitted=Date.parse(raw?.emitted_at||incoming.emitted_at||incoming.generated_at||'');
    state.sync.browserAppliedAt=Date.now();state.sync.lastReceivedAt=state.sync.browserAppliedAt;state.sync.browserDelayMs=Number.isFinite(emitted)?Math.max(0,Date.now()-emitted):null;state.streamFiles=state.sync.snapshot.sources||[];state.loadedAt=Date.now();
    if(!heartbeat)recordSyncHistory(state.sync.snapshot);updateFreshness();
    if(eventType==='sync_critical'){const item=incoming.source||incoming.sample||incoming;notify(`Critical telemetry: ${item.freshness_reason||(item.diagnostic_codes||[]).join(', ')||'operator review required'}`,'error');}
    renderSyncSubscriber();return true;
  }
  function syncDashboardSnapshot() {
    const sync=state.sync.snapshot||{},aggregate=sync.aggregate||{},history=state.sync.history.slice(-60),sources=sync.sources||[],diagnostics=sync.diagnostics||{};
    const sumKnown=(items,key)=>items.some(item=>present(sourceValue(item,key)))?items.reduce((n,item)=>n+(present(sourceValue(item,key))&&Number.isFinite(Number(sourceValue(item,key)))?Number(sourceValue(item,key)):0),0):null;
    const nodes=[...new Set(sources.map(item=>item.node_id).filter(Boolean))].map(id=>{const nodeSources=sources.filter(item=>item.node_id===id),latest=nodeSources.map(item=>item.controller_received_at||item.updated_at).filter(Boolean).sort().at(-1),parseDurations=nodeSources.map(item=>sourceValue(item,'parse_duration_ns')).filter(present).map(value=>Number(value)/1e6).filter(Number.isFinite),degraded=nodeSources.some(item=>['stale','disconnected','collection_failed'].includes(item.freshness)),live=nodeSources.some(item=>['live','delayed'].includes(item.freshness));return {id,hostname:id,status:degraded?'degraded':live?'online':'unknown',last_seen:latest,issues:nodeSources.flatMap(item=>item.diagnostic_codes||[]),metrics:{logs_received:sumKnown(nodeSources,'nli_source_records_read_total'),logs_parsed:sumKnown(nodeSources,'nli_parser_records_succeeded_total'),logs_quarantined:sumKnown(nodeSources,'nli_quarantined_records_total'),parse_latency_ms:parseDurations.length?Math.max(...parseDurations):null}};});
    const streamFiles=sources.map(item=>({...item,read_lines_per_second:sourceValue(item,'read_lines_per_second'),normalized_per_second:sourceValue(item,'normalized_per_second'),quarantined_per_second:sourceValue(item,'quarantined_per_second'),queue_depth:sourceValue(item,'queue_depth')??sourceValue(item,'nli_queue_depth'),queue_capacity:sourceValue(item,'queue_capacity'),backpressure:sourceValue(item,'backpressure'),read_duration_ns:sourceValue(item,'read_duration_ns'),parse_duration_ns:sourceValue(item,'parse_duration_ns'),output_duration_ns:sourceValue(item,'output_duration_ns')}));
    const controller=sync.controller||{};
    return {canonical_aggregate:aggregate,overview:{fleet:{total:nodes.length,online:nodes.filter(item=>item.status==='online').length}},metadata:{records:[],routes:{}},system:{health:controller.health||'not reported',version:controller.version||'Controller health was not included in the canonical snapshot'},fleet_sync:{nodes:nodes.map(node=>({node}))},timeseries:{from:history[0]?.timestamp,to:history.at(-1)?.timestamp,points:history.map(item=>({timestamp:item.timestamp,logs_received:item.received,logs_parsed:item.parsed,logs_quarantined:item.quarantine,errors:item.errors,parse_latency_ms_avg:item.latency}))},stream_files:streamFiles,sync_diagnostics:diagnostics,replay:sync.subsystems?.replay};
  }
  function issueRows(scope='') {
    const snapshot=state.sync.snapshot||{},issues=[...(snapshot.diagnostics?.issues||[])];
    (snapshot.sources||[]).filter(source=>source.critical||!['live','delayed'].includes(source.freshness)||(source.diagnostic_codes||[]).length).forEach(source=>issues.push({code:(source.diagnostic_codes||[]).join(', ')||source.freshness||'source_state',stage:'source freshness',severity:source.critical?'critical':source.freshness==='live'?'info':'warning',component:'NHLE',node_id:source.node_id,source:source.source,reason:source.freshness_reason||`Source is ${syncStatus(source.freshness)}`,recommended_action:source.recommended_action||source.operator_action,last_seen:source.controller_received_at||source.updated_at,occurrences:1}));
    const filtered=scope?issues.filter(item=>`${item.component||''} ${item.stage||''}`.toLowerCase().includes(scope.toLowerCase())):issues;
    return filtered.map(item=>`<tr><td>${syncStatusBadge(item.severity)}</td><td><b>${esc(item.code||'diagnostic')}</b><br><small>${time(item.last_seen)}</small></td><td>${esc(item.stage||'unknown')}<br><small>${esc(item.component||'—')}</small></td><td>${esc(item.node_id||'—')}<br><small class="mono">${esc(item.source||'—')}</small></td><td>${esc(item.reason||item.evidence||'Reason not supplied')}</td><td>${esc(item.recommended_action||item.operator_action||item.remediation||'Inspect the affected stage and latest evidence.')}</td></tr>`);
  }
  function diagnosticsPanel(scope='') { const rows=issueRows(scope);return `<section class="panel diagnostic-panel"><div class="section-head"><div><h3>Latency and error diagnostics</h3><p>Every delay or error includes the affected stage, component, node, reason, and operator action when supplied.</p></div></div>${table(['Severity','Code / last seen','Stage / component','Node / source','Reason / root cause','Recommended action'],rows,'No active latency or error diagnostics for this scope.')}</section>`; }
  function subsystemNav() { return `<div class="tabs subsystem-tabs" role="tablist" data-subsystem-tabs><button class="tab" data-subsystem-tab="overview">Overview</button><button class="tab" data-subsystem-tab="activity">Live activity</button><button class="tab" data-subsystem-tab="freshness">Freshness</button><button class="tab" data-subsystem-tab="diagnostics">Diagnostics</button></div>`; }
  function subsystemMetric(label,value,note,unit='',tone='blue') { return dashboardKPI(label,metricText(value,unit),note,tone); }
  function subsystemGlossary() {
    const names=[
      ['NHLE','Nexora Hypersonic Log Engine','Per-record source reading, framing, parsing, normalization, quarantine routing, and delivery.'],
      ['NURSF','Nexora Unified Real-Time Synchronization Fabric','One authenticated, revisioned telemetry stream for every live workspace.'],
      ['NAPI','Nexora Adaptive Parser Intelligence','Operator-triggered bounded parser candidate synthesis; never direct production activation.'],
      ['NAL','Nexora Assurance Lab','Deterministic validation, evidence, review preparation, and release assurance outside the ingest hot path.'],
      ['NUPIRE','Nexora Universal Parsing Intelligence & Recovery Engine','Evidence-linked discovery and recovery candidate lifecycle.'],
      ['EGPE','Evidence-Gated Parser Evolution','Held-out evidence, independent review, rollout, drift, and rollback gates.'],
    ];
    return panel('Subsystem name reference','Every short name expands here. Monitoring a subsystem does not mean it runs on every record.',`<div class="subsystem-glossary">${names.map(([short,name,purpose])=>`<article><b>${esc(short)}</b><strong>${esc(name)}</strong><small>${esc(purpose)}</small></article>`).join('')}</div>`);
  }
  function orchestrationPanel(snapshot) {
    const subsystems=snapshot.subsystems||{},aggregate=snapshot.aggregate||{},sources=snapshot.sources||[],nhle=subsystems.nhle||{},napi=subsystems.napi||{},nal=subsystems.nal||{},delivery=subsystems.delivery||{},release=subsystems.release_readiness||{},replay=subsystems.replay||{};
    const stage=(index,short,name,status,detail,tone='')=>`<article class="orchestration-stage ${tone}"><span>${esc(index)}</span><div><b>${esc(short)}</b><strong>${esc(name)}</strong><small>${esc(detail)}</small></div>${syncStatusBadge(status)}</article>`;
    const sourceState=sources.length?(sources.some(item=>['stale','disconnected'].includes(item.freshness))?'degraded':'live'):'idle';
    const live=[
      stage('1','SOURCE','Watch and frame',sourceState,`${number(sources.length)} authenticated source${sources.length===1?'':'s'} in revision ${number(snapshot.revision)}`),
      stage('2','NHLE','Parse and normalize',nhle.status||'unavailable',`${metricText(aggregate.nli_source_records_read_total)} read · ${metricText(aggregate.nli_normalized_records_total)} normalized`),
      stage('3','OUTPUT','Deliver and checkpoint',delivery.status||'unavailable','Customer destinations become durable before the source checkpoint advances.'),
      stage('4','NURSF','Publish canonical telemetry',state.streamMode==='live'?'live':state.streamMode,`Revision ${number(snapshot.revision)} · browser apply ${metricText(state.sync.browserDelayMs,' ms')}`),
    ];
    const recovery=[
      stage('A','QUARANTINE','Retain failure evidence',Number(aggregate.nli_quarantined_records_total)>0?'attention':'idle',`${metricText(aggregate.nli_quarantined_records_total)} records reported by the canonical aggregate`,'recovery'),
      stage('B','NAL','Diagnose and validate',nal.status||'idle',`${metricText(nal.total)} persisted runs · ${metricText(nal.pending_drafts??napi.pending_drafts)} pending drafts`,'recovery'),
      stage('C','NAPI + NUPIRE / EGPE','Propose and gate evolution',napi.status||'idle','Runs only when an operator supplies evidence; candidates cannot activate themselves.','recovery'),
      stage('D','FLEET','Approve, roll out, acknowledge',release.status||'unavailable',`${metricText(release.managed_nodes)} managed clients · exact bundle hash required`,'recovery'),
      stage('E','REPLAY','Replay selected record IDs',replay.status||'idle',`${metricText(replay.queued)} queued · ${metricText(replay.resolved)} resolved; then return to NHLE`,'recovery'),
    ];
    return panel('NLI execution orchestration','Live values come from the current canonical revision. The normal data path and the evidence-gated recovery loop have different execution boundaries.',`<div class="orchestration-block"><div class="orchestration-heading"><b>Ordinary live record</b><span>Per-record hot path</span></div><div class="orchestration-flow">${live.join('')}</div></div><div class="orchestration-block recovery"><div class="orchestration-heading"><b>Evidence-gated recovery</b><span>Starts only after quarantine or an explicit operator request</span></div><div class="orchestration-flow">${recovery.join('')}</div></div><div class="callout">NAPI, NAL, NUPIRE, and EGPE do not run on every record. They create and verify a candidate only through the audited recovery workflow; rollout approval and the origin client’s applied bundle acknowledgement remain mandatory before replay.</div>`);
  }
  function renderSubsystemPage(kind) {
    const snapshot=state.sync.snapshot||{};
    const defs=kind==='nhle-monitor'?{key:'nhle',name:'Nexora Hypersonic Log Engine',short:'NHLE',description:'Parallel source ingestion, framing, normalization, queue and backpressure status.',cards:[['Sources','source_count','Registered source readers'],['Active nodes','active_nodes','Nodes publishing NHLE state'],['Records read','records_read_total','Cumulative source records'],['Normalized','normalized_total','Structured output records'],['Quarantined','quarantined_total','Records awaiting resolution'],['Queue depth','queue_depth','Current bounded queue'],['Read rate','read_lines_per_second','Current aggregate',' lines/s'],['Output rate','normalized_per_second','Current aggregate',' lines/s']]}:kind==='napi-monitor'?{key:'napi',name:'Nexora Adaptive Parser Intelligence',short:'NAPI',description:'Bounded parser synthesis, proposal failures and approval-gated draft activity.',cards:[['Requests','requests_total','Synthesis requests'],['Failures','failures_total','Failed synthesis requests'],['Pending drafts','pending_drafts','Awaiting validation or review'],['Latest proposal','latest_proposal_at','Last proposal timestamp']]}:{key:'nal',name:'Nexora Assurance Lab',short:'NAL',description:'Durable assurance run scheduling, validation outcomes and release evidence.',cards:[['Total runs','total','Persisted assurance runs'],['Queued','queued','Awaiting a bounded worker'],['Running','running','Currently executing'],['Passed','passed','Successful runs'],['Failed','failed','Failed runs'],['Cancelled','cancelled','Operator-cancelled runs'],['Latest run','latest_run_at','Latest run timestamp']]};
    const domain=snapshot.subsystems?.[defs.key]||{},status=domain.status||'unavailable',updated=domain.updated_at||snapshot.generated_at,sources=defs.key==='nhle'?(snapshot.sources||[]):[];
    const cards=defs.cards.map(([label,key,note,unit=''])=>subsystemMetric(label,key.endsWith('_at')&&domain[key]?time(domain[key]):domain[key],note,unit,/fail|quarant|queue/.test(key)&&Number(domain[key])?'amber':'blue')).join('');
    const activity=defs.key==='nhle'?renderStreamRows(sources):`<pre class="json-output subsystem-json">${esc(pretty(domain))}</pre>`;
    const freshnessRows=(defs.key==='nhle'?sources:[]).map(item=>`<tr><td>${esc(item.node_id||'—')}</td><td class="mono">${esc(item.source||defs.short)}</td><td>${syncStatusBadge(item.freshness||status)}</td><td>${metricText(item.freshness_age_ms,' ms')}</td><td>${esc(item.freshness_reason||'Within measured target')}</td><td>${time(item.controller_received_at||item.updated_at)}</td></tr>`);
    renderPage(kind,`<div class="dashboard-intro subsystem-hero"><div><p class="eyebrow">Canonical NURSF subscriber · ${defs.short}</p><h1>${defs.name}</h1><p><b>${defs.short}</b> means <b>${defs.name}</b>. ${defs.description} All measurements come from revision ${number(snapshot.revision)} of the shared controller state.</p></div><div class="subsystem-state">${syncStatusBadge(status)}<small>Updated ${time(updated)}</small></div></div>${subsystemNav()}<section data-subsystem-panel="overview"><div class="dashboard-kpi-strip subsystem-kpis">${cards}</div>${panel('Canonical subsystem state','The same revision is consumed by Command Center, synchronization diagnostics, and every subsystem view.',`<div class="sync-contract"><span>Schema <b>${esc(snapshot.schema_version||'unavailable')}</b></span><span>Revision <b>${number(snapshot.revision)}</b></span><span>Target <b>${metricText(snapshot.target_interval_ms,' ms')}</b></span><span>Browser apply <b>${metricText(state.sync.browserDelayMs,' ms')}</b></span></div>`)}${orchestrationPanel(snapshot)}${subsystemGlossary()}</section><section data-subsystem-panel="activity">${panel(`${defs.short} live activity`,'Current bounded metadata from the canonical synchronization stream.',activity)}</section><section data-subsystem-panel="freshness">${panel('Measured freshness','Missing values remain unavailable; stale last-known values are labelled instead of displayed as healthy zero.',freshnessRows.length?table(['Node','Source / subsystem','State','Age','Reason','Controller received'],freshnessRows):`<div class="grid cols-3">${subsystemMetric('Subsystem state',syncStatus(status),'Controller classification')}${subsystemMetric('Updated',updated?time(updated):null,'Authoritative timestamp')}${subsystemMetric('Browser apply',state.sync.browserDelayMs,'Measured after SSE delivery',' ms')}</div>`)}</section><section data-subsystem-panel="diagnostics">${diagnosticsPanel(defs.short)}</section>`);
  }
  function renderSyncFabric() {
    const snapshot=state.sync.snapshot||{},aggregate=snapshot.aggregate||{},diagnostics=snapshot.diagnostics||{},sources=snapshot.sources||[],states=['live','delayed','stale','disconnected'];
    const freshnessCards=states.map(key=>subsystemMetric(key,diagnostics[`${key}_sources`]??diagnostics[key],`Sources classified ${key}`,'',key==='live'?'green':key==='delayed'?'amber':key==='stale'||key==='disconnected'?'red':'blue')).join('');
    const sourceRows=sources.map(item=>`<tr><td>${esc(item.node_id||'—')}</td><td class="mono">${esc(item.source||'—')}</td><td>${number(item.sequence)}</td><td>${syncStatusBadge(item.freshness)}</td><td>${metricText(item.freshness_age_ms,' ms')}</td><td>${metricText(item.controller_processing_ms,' ms')}</td><td>${metricText(item.observed_propagation_delay_ms,' ms')}</td><td>${esc(item.freshness_reason||'Within target')}</td></tr>`);
    renderPage('sync-fabric',`<div class="dashboard-intro subsystem-hero"><div><p class="eyebrow">NURSF · single source of operational truth</p><h1>Nexora Unified Real-Time Synchronization Fabric</h1><p><b>NURSF</b> means <b>Nexora Unified Real-Time Synchronization Fabric</b>. It collects once, aggregates centrally, distributes incremental revisions, and measures actual freshness against the one-second target.</p></div><div class="subsystem-state">${syncStatusBadge(state.streamMode==='live'?'live':state.streamMode)}<small>Revision ${number(snapshot.revision)}</small></div></div>${subsystemNav()}<section data-subsystem-panel="overview"><div class="dashboard-kpi-strip">${subsystemMetric('Records read',aggregate.nli_source_records_read_total,'Canonical cumulative counter')}${subsystemMetric('Normalized',aggregate.nli_normalized_records_total,'Canonical cumulative counter')}${subsystemMetric('Quarantined',aggregate.nli_quarantined_records_total,'Canonical cumulative counter', '',Number(aggregate.nli_quarantined_records_total)?'amber':'green')}${subsystemMetric('Queue depth',aggregate.nli_queue_depth,'Current gauge')}${subsystemMetric('Propagation p95',diagnostics.propagation_delay_ms_p95??diagnostics.p95_ms??(present(aggregate.nli_telemetry_propagation_delay_seconds_p95)?Number(aggregate.nli_telemetry_propagation_delay_seconds_p95)*1000:null),'Measured controller path',' ms','cyan')}${subsystemMetric('Browser apply',state.sync.browserDelayMs,'SSE event to shared store',' ms','violet')}</div>${panel('Synchronization contract','One snapshot initializes every page; ordered incremental revisions update the shared browser store.',`<div class="sync-contract"><span>Schema <b>${esc(snapshot.schema_version||'unavailable')}</b></span><span>Revision <b>${number(snapshot.revision)}</b></span><span>Target <b>${metricText(snapshot.target_interval_ms,' ms')}</b></span><span>Subscribers <b>${metricText(diagnostics.subscribers)}</b></span><span>Rejected updates <b>${number(state.sync.rejectedUpdates)}</b></span></div>`)}${orchestrationPanel(snapshot)}${subsystemGlossary()}</section><section data-subsystem-panel="activity">${panel('Canonical source revisions','Sequence, processing and propagation measurements per authenticated source.',table(['Node','Source','Sequence','Freshness','Age','Controller processing','Propagation','Reason'],sourceRows,'No authenticated source telemetry has reached the controller.'))}</section><section data-subsystem-panel="freshness"><div class="dashboard-kpi-strip">${freshnessCards}</div>${panel('Transport integrity','Sequence and reconnect signals expose loss, duplication and resynchronization.',`<div class="grid cols-4">${subsystemMetric('Accepted',diagnostics.accepted,'Controller-accepted revisions')}${subsystemMetric('Duplicates',diagnostics.duplicate,'Deduplicated')}${subsystemMetric('Out of order',diagnostics.out_of_order,'Rejected safely')}${subsystemMetric('Sequence gaps',diagnostics.sequence_gaps,'Require resynchronization')}${subsystemMetric('Reconnects',diagnostics.reconnects,'Transport reconnect events')}${subsystemMetric('Coalesced',diagnostics.coalesced,'Bounded visual updates')}${subsystemMetric('Missed target',diagnostics.missed_deadlines??diagnostics.missed,'One-second deadlines')}${subsystemMetric('Clock skew',diagnostics.clock_skew,'Wall-clock uncertainty')}</div>`)}</section><section data-subsystem-panel="diagnostics">${diagnosticsPanel()}</section>`);
  }

  const MONITORING_VIEWS=[
    ['overview','Overview'],['fleet','Fleet & Nodes'],['pipeline','Pipeline'],['sources','Sources'],['parsers','Parser Performance'],['destinations','Destinations'],['queues','Queues / Replay / Quarantine'],['infrastructure','Infrastructure'],['latency','Latency & Synchronization'],['errors','Errors & Incidents'],['assurance','Assurance & Release Health']
  ];
  const unavailable=(label,note='This measurement is not reported in the current canonical telemetry snapshot.')=>dashboardKPI(label,'Not reported',note,'violet');
  const durationMS=value=>present(value)?Number(value)/1e6:null;
  function setupMonitoringTabs(){
    const tabs=[...page.querySelectorAll('[data-monitoring-tab]')],panels=[...page.querySelectorAll('[data-monitoring-panel]')];
    tabs.forEach(tab=>{const key=tab.dataset.monitoringTab,active=key===state.monitoringView;tab.id=`monitoring-tab-${key}`;tab.setAttribute('aria-controls',`monitoring-panel-${key}`);tab.classList.toggle('active',active);tab.setAttribute('aria-selected',String(active));tab.tabIndex=active?0:-1;});
    panels.forEach(panel=>{const key=panel.dataset.monitoringPanel;panel.id=`monitoring-panel-${key}`;panel.setAttribute('aria-labelledby',`monitoring-tab-${key}`);panel.tabIndex=0;panel.hidden=key!==state.monitoringView;});
  }
  function monitoringFilters(sources){
    const nodes=[...new Set(sources.map(item=>item.node_id).filter(Boolean))].sort(),sourceNames=[...new Set(sources.map(item=>item.source).filter(Boolean))].sort();
    const option=(value,label,selected)=>`<option value="${esc(value)}" ${selected===value?'selected':''}>${esc(label)}</option>`;
    return `<div class="monitoring-toolbar"><label>Time range<select data-monitoring-filter="range">${['5m','15m','1h','6h','24h','7d'].map(value=>option(value,value,state.monitoringRange)).join('')}</select></label><label>Node<select data-monitoring-filter="node">${option('all','All nodes',state.monitoringNode)}${nodes.map(value=>option(value,value,state.monitoringNode)).join('')}</select></label><label>Source<select data-monitoring-filter="source">${option('all','All sources',state.monitoringSource)}${sourceNames.map(value=>option(value,value,state.monitoringSource)).join('')}</select></label><label>Search<input type="search" value="${esc(state.monitoringSearch)}" data-monitoring-filter="search" placeholder="Node or source"></label><button class="secondary small" data-action="monitoring-pause">${state.monitoringPaused?'Resume live':'Pause view'}</button><button class="secondary small" data-action="monitoring-reset">Reset filters</button><div class="monitoring-freshness">${syncStatusBadge(state.streamMode==='live'?'live':state.streamMode)}<span>Revision ${number(state.sync.snapshot?.revision)}</span><span>Updated ${time(state.sync.snapshot?.generated_at)}</span></div><p class="filter-contract-note">Filters: time, node, source and search. Component, destination and environment labels are not included in the canonical source contract.</p></div>`;
  }
  function monitoringTabs(){return `<div class="monitoring-tabs" role="tablist" aria-label="Monitoring dashboards">${MONITORING_VIEWS.map(([id,label])=>`<button class="tab ${id===state.monitoringView?'active':''}" role="tab" aria-selected="${id===state.monitoringView}" data-action="monitoring-tab" data-monitoring-tab="${id}">${esc(label)}</button>`).join('')}</div>`;}
  function monitoringPanel(id,body){return `<section class="monitoring-view" role="tabpanel" data-monitoring-panel="${id}" ${id===state.monitoringView?'':'hidden'}>${body}</section>`;}
  function sourceHealth(source){const stateValue=source.freshness||'unavailable';return ['live','delayed'].includes(stateValue)&&!source.backpressure?'healthy':source.backpressure?'degraded':stateValue;}
  function monitorMetric(label,value,unit,note,tone='blue'){return present(value)?dashboardKPI(label,`${number(value)}${unit||''}`,note,tone):unavailable(label,note||undefined);}
  function renderMonitoring(){
    const snapshot=state.sync.snapshot||{},aggregate=snapshot.aggregate||{},diagnostics=snapshot.diagnostics||{},allSources=snapshot.sources||[];
    const search=state.monitoringSearch.trim().toLowerCase(),sources=allSources.filter(item=>(state.monitoringNode==='all'||item.node_id===state.monitoringNode)&&(state.monitoringSource==='all'||item.source===state.monitoringSource)&&(!search||`${item.node_id||''} ${item.source||''}`.toLowerCase().includes(search)));
    const ranges={ '5m':5*60e3,'15m':15*60e3,'1h':60*60e3,'6h':6*60*60e3,'24h':24*60*60e3,'7d':7*24*60*60e3 },allHistory=state.sync.history.slice(-120),latestHistory=Date.parse(allHistory.at(-1)?.timestamp||''),cutoff=Number.isFinite(latestHistory)?latestHistory-(ranges[state.monitoringRange]||ranges['15m']):0,history=allHistory.filter(item=>Date.parse(item.timestamp||'')>=cutoff),labels=history.map(item=>new Date(item.timestamp).toLocaleTimeString()),issues=issueRows(),target=Number(snapshot.target_interval_ms||1000);
    const allSourcesSelected=state.monitoringNode==='all'&&state.monitoringSource==='all',sumSelected=(key,fallback)=>sources.length?(sources.some(item=>present(sourceValue(item,key)))?sources.reduce((sum,item)=>sum+(present(sourceValue(item,key))&&Number.isFinite(Number(sourceValue(item,key)))?Number(sourceValue(item,key)):0),0):null):allSourcesSelected?aggregate[fallback]:null;
    const readRate=sumSelected('read_lines_per_second','read_lines_per_second');
    const normalizedRate=sumSelected('normalized_per_second','normalized_per_second');
    const quarantineRate=sumSelected('quarantined_per_second','quarantined_per_second');
    const validationAttemptsRate=sumSelected('validation_attempts_per_second','validation_attempts_per_second');
    const validationPassedRate=sumSelected('validation_passed_per_second','validation_passed_per_second');
    const validationFailedRate=sumSelected('validation_failed_per_second','validation_failed_per_second');
    const queueValues=sources.map(item=>sourceValue(item,'nli_queue_depth')??sourceValue(item,'queue_depth'));
    const queueDepth=sources.length?(queueValues.some(present)?queueValues.reduce((sum,value)=>sum+(present(value)&&Number.isFinite(Number(value))?Number(value):0),0):null):allSourcesSelected?aggregate.nli_queue_depth:null;
    const propagationP95=diagnostics.observed_delay_p95_ms??diagnostics.propagation_delay_ms_p95??diagnostics.p95_ms??(present(aggregate.nli_telemetry_propagation_delay_seconds_p95)?Number(aggregate.nli_telemetry_propagation_delay_seconds_p95)*1000:null);
    const activeIncidents=(diagnostics.issues||[]).filter(item=>/critical|error/i.test(item.severity||'')).length;
    const ratesChart=history.length>1?multiSeriesChart(labels,[{label:'Input',values:history.map(item=>item.received),color:'#53a5ff',unit:'records'},{label:'Parsed',values:history.map(item=>item.parsed),color:'#3dd598',unit:'records'},{label:'Quarantine',values:history.map(item=>item.quarantine),color:'#f8c15c',unit:'records'},{label:'Errors',values:history.map(item=>item.errors),color:'#ff6b7d',unit:'records'}]):empty('A second canonical revision is required before rates can be graphed.');
    const overview=monitoringPanel('overview',`<div class="dashboard-kpi-strip monitoring-kpis">${monitorMetric('Ingestion',readRate,' rec/s','Canonical current source rate','cyan')}${monitorMetric('Parsing output',normalizedRate,' rec/s','Normalized current rate','green')}${monitorMetric('Quarantine',quarantineRate,' rec/s','Awaiting parser resolution',Number(quarantineRate)?'amber':'green')}${monitorMetric('Output',normalizedRate,' rec/s','Normalized production; destination acknowledgement unavailable','green')}${monitorMetric('Pending queue',queueDepth,' records','Canonical bounded queue',Number(queueDepth)?'amber':'green')}${monitorMetric('Propagation p95',propagationP95,' ms','Measured controller synchronization path','violet')}${dashboardKPI('Critical incidents',number(activeIncidents),'Current bounded diagnostics',activeIncidents?'red':'green')}</div><div class="monitor-grid">${panel('Ingestion and parsing throughput','Exact deltas between canonical revisions. Hover or focus points for category, time and count.',ratesChart)}${panel('Fleet and source state','Current authoritative status; idle, stale and disconnected are kept distinct.',table(['Node','Source','Health','Freshness','Input/s','Output/s','Queue','Reason'],sources.map(item=>`<tr><td>${esc(item.node_id||'—')}</td><td class="mono">${esc(item.source||'—')}</td><td>${syncStatusBadge(sourceHealth(item))}</td><td>${syncStatusBadge(item.freshness)}</td><td>${metricText(sourceValue(item,'read_lines_per_second'))}</td><td>${metricText(sourceValue(item,'normalized_per_second'))}</td><td>${metricText(sourceValue(item,'queue_depth')??sourceValue(item,'nli_queue_depth'))}</td><td>${esc(item.freshness_reason||'Within target')}</td></tr>`),'No source telemetry matches the selected filters.'))}${diagnosticsPanel()}</div>`);
    const nodeIDs=[...new Set(sources.map(item=>item.node_id).filter(Boolean))];
    const fleetRows=nodeIDs.map(id=>{const rows=sources.filter(item=>item.node_id===id),latest=rows.map(item=>item.controller_received_at||item.updated_at).filter(Boolean).sort().at(-1),bad=rows.filter(item=>!['live','delayed'].includes(item.freshness)).length;return `<tr><td><b>${esc(id)}</b><br><small>Stable telemetry identity</small></td><td>${syncStatusBadge(bad?'degraded':'online')}</td><td>${time(latest)}</td><td>${number(rows.length)}</td><td>${metricText(rows.reduce((n,item)=>n+Number(sourceValue(item,'read_lines_per_second')||0),0),' rec/s')}</td><td>${metricText(rows.reduce((n,item)=>n+Number(sourceValue(item,'normalized_per_second')||0),0),' rec/s')}</td><td>${number(rows.reduce((n,item)=>n+Number(sourceValue(item,'queue_depth')??0),0))}</td><td>${bad?badge(`${bad} stale sources`):badge('none')}</td></tr>`;});
    const fleet=monitoringPanel('fleet',`${panel('Fleet snapshot','A single bounded controller snapshot supplies this table; the browser never queries every client row.',table(['Node identity','Status','Last telemetry','Sources','Ingestion','Output','Queue','Alerts'],fleetRows,'No reporting nodes match the selected filters.'))}<div class="instrumentation-note"><b>Telemetry boundary:</b> agent version, CPU, memory and disk are not present in this source snapshot and are therefore never shown as fabricated zeros.</div>`);
    const stage=(name,inRate,outRate,latency,depth,status,note)=>`<tr><td><b>${name}</b></td><td>${metricText(inRate,' rec/s')}</td><td>${metricText(outRate,' rec/s')}</td><td>${metricText(latency,' ms')}</td><td>${metricText(depth)}</td><td>${syncStatusBadge(status)}</td><td>${esc(note)}</td></tr>`;
    const maxDuration=key=>{const values=sources.map(item=>durationMS(item[key])).filter(present).map(Number).filter(Number.isFinite);return values.length?Math.max(...values):null;};
    const readLatency=maxDuration('read_duration_ns'),parseLatency=maxDuration('parse_duration_ns'),outputLatency=maxDuration('output_duration_ns');
    const parseSamples=sources.map(item=>durationMS(item.parse_duration_ns)).filter(present).map(Number).sort((a,b)=>a-b),parsePercentile=p=>parseSamples.length?parseSamples[Math.min(parseSamples.length-1,Math.ceil(p*parseSamples.length)-1)]:null,sourceWorkers=sources.length?sources.reduce((sum,item)=>sum+Number(item.source_workers||0),0):null;
    const pipelineRows=[stage('READ',readRate,readRate,readLatency,queueDepth,sources.some(item=>item.backpressure)?'degraded':present(readRate)?'live':sources.length?'unknown':'idle','Source reader and framing boundary'),stage('PARSE',readRate,present(normalizedRate)&&present(quarantineRate)?Number(normalizedRate)+Number(quarantineRate):null,parseLatency,queueDepth,Number(quarantineRate)>0?'degraded':present(normalizedRate)?'live':sources.length?'unknown':'idle','Success and quarantine split'),stage('NORMALIZE',normalizedRate,normalizedRate,null,null,present(normalizedRate)?'live':sources.length?'unknown':'idle','Rate known; stage latency is included in parser-path duration'),stage('VALIDATE',validationAttemptsRate,validationPassedRate,parseLatency,null,Number(validationFailedRate)>0?'degraded':present(validationAttemptsRate)?'live':sources.length?'awaiting upgrade':'idle',present(validationAttemptsRate)?`${metricText(validationFailedRate,' rec/s')} failed validation; measured inside parser path`:'Upgrade the R7 client to publish validation counters'),stage('PERSIST / ROUTE / DELIVER',normalizedRate,null,outputLatency,null,present(outputLatency)?'measured':'not reported','Output duration may be measured; destination acknowledgements are not published')];
    const pipeline=monitoringPanel('pipeline',`${panel('Live processing pipeline','Rates and timings are only populated where the client publishes authoritative measurements.',table(['Stage','Incoming','Outgoing','Latency','Queue','Health','Evidence'],pipelineRows))}${panel('Likely bottleneck','This is an inference from queue growth, backpressure and latency, never a claimed root cause.',sources.some(item=>item.backpressure)?`<div class="incident-callout bad"><b>Likely queue pressure</b><p>${number(sources.filter(item=>item.backpressure).length)} source(s) report backpressure. Inspect source queue capacity and output latency.</p></div>`:`<div class="incident-callout good"><b>No measured bottleneck</b><p>No selected source currently reports backpressure. Stages outside the current canonical schema can still hide delays.</p></div>`)}`);
    const sourceRows=sources.map(item=>{const processed=present(item.normalized)||present(item.quarantined)?Number(item.normalized||0)+Number(item.quarantined||0):null;return `<tr><td>${esc(item.node_id||'—')}</td><td class="mono">${esc(item.source||'—')}</td><td>${syncStatusBadge(item.phase||'unknown')}</td><td>${metricText(sourceValue(item,'bytes_per_second'),' B/s')}</td><td>${metricText(sourceValue(item,'read_lines_per_second'),' rec/s')}</td><td>${metricText(sourceValue(item,'normalized_per_second'),' rec/s')}</td><td>${metricText(item.read_lines)} / ${metricText(processed)}</td><td>${metricText(item.normalized)} / ${metricText(item.quarantined)}</td><td>${metricText(item.errors)}</td><td>${metricText(item.sequence)}</td><td>${time(item.controller_received_at||item.updated_at)}</td><td>${esc(item.freshness_reason||'Within target')}</td></tr>`;});
    const sourceView=monitoringPanel('sources',`${panel('Source activity','Configured sources appear only after authenticated telemetry reaches the controller. Cumulative counts use the same authoritative source snapshot as Command Center.',table(['Node','Source','Phase','Bytes/s','Read/s','Parser output/s','Read / processed','Parsed / quarantine','Errors','Sequence','Last activity','State reason'],sourceRows,'No configured source telemetry matches the selected filters.'))}<div class="instrumentation-note"><b>Telemetry boundary:</b> filesystem checkpoint offsets, rotation/truncation events and partial-record state are not present in the current canonical schema.</div>`);
    const parserView=monitoringPanel('parsers',`<div class="dashboard-kpi-strip monitoring-kpis">${monitorMetric('Parser successes',aggregate.nli_parser_records_succeeded_total,' records','Canonical cumulative counter','green')}${monitorMetric('Parser failures',aggregate.nli_parser_records_failed_total,' records','Canonical cumulative counter',Number(aggregate.nli_parser_records_failed_total)?'red':'green')}${monitorMetric('Parse p50',parsePercentile(.50),' ms','Current selected source samples','violet')}${monitorMetric('Parse p95',parsePercentile(.95),' ms','Current selected source samples','violet')}${monitorMetric('Parse p99',parsePercentile(.99),' ms','Current selected source samples','violet')}${monitorMetric('Source workers',sourceWorkers,'','Current client-reported workers','cyan')}${monitorMetric('Quarantine',aggregate.nli_quarantined_records_total,' records','Cumulative parser outcome',Number(aggregate.nli_quarantined_records_total)?'amber':'green')}</div>${panel('Parser source comparison','Current output, quarantine, worker count and parse duration by selected source.',table(['Node / source','Workers','Output/s','Quarantine/s','Parse duration','State'],sources.map(item=>`<tr><td>${esc(item.node_id||'—')}<br><small class="mono">${esc(item.source||'—')}</small></td><td>${metricText(item.source_workers)}</td><td>${metricText(sourceValue(item,'normalized_per_second'))}</td><td>${metricText(sourceValue(item,'quarantined_per_second'))}</td><td>${metricText(durationMS(item.parse_duration_ns),' ms')}</td><td>${syncStatusBadge(item.freshness)}</td></tr>`),'No parser source telemetry is available.'))}`);
    const delivery=snapshot.subsystems?.delivery||{},acknowledgements=delivery.acknowledgements_by_destination||{},destinationRows=Object.entries(acknowledgements).map(([name,counts])=>{const total=Object.values(counts||{}).reduce((sum,value)=>sum+Number(value||0),0),failed=Object.entries(counts||{}).filter(([status])=>/fail|error|reject/i.test(status)).reduce((sum,[,value])=>sum+Number(value||0),0);return `<tr><td><b>${esc(name)}</b></td><td>${number(total)}</td><td>${number(failed)}</td><td>${esc(Object.entries(counts||{}).map(([status,value])=>`${status}: ${number(value)}`).join(' · ')||'No acknowledgements')}</td><td>${syncStatusBadge(failed?'degraded':'healthy')}</td></tr>`;});
    const destinations=monitoringPanel('destinations',`${panel('Destination delivery','Client-reported delivery acknowledgements are retained as bounded metadata and share the canonical revision.',table(['Destination','Acknowledgements','Failed','Status counts','Health'],destinationRows,'No destination acknowledgement has been reported yet. Parsing can be idle while delivery telemetry remains ready.'))}<div class="instrumentation-note"><b>Current output:</b> normalized production is ${metricText(normalizedRate,' records/s')}. Destination retry, backlog and latency remain separately labelled by the controller as ${esc(delivery.retry_backlog_latency_status||'not reported')}.</div>`);
    const queueRows=sources.map(item=>`<tr><td>${esc(item.node_id||'—')}</td><td class="mono">${esc(item.source||'—')}</td><td>${metricText(sourceValue(item,'queue_depth')??sourceValue(item,'nli_queue_depth'))}</td><td>${metricText(sourceValue(item,'queue_capacity'))}</td><td>${sourceValue(item,'backpressure')?badge('backpressure'):badge('normal')}</td><td>—</td></tr>`);
    const replay=snapshot.subsystems?.replay||{},replayTotal=Number(replay.queued||0)+Number(replay.reprocessing||0)+Number(replay.resolved||0),replayProgress=replayTotal?100*Number(replay.resolved||0)/replayTotal:0;
    const queues=monitoringPanel('queues',`<div class="dashboard-kpi-strip monitoring-kpis">${monitorMetric('Queue depth',queueDepth,' records','Current selected sources',Number(queueDepth)?'amber':'green')}${monitorMetric('Quarantined total',aggregate.nli_quarantined_records_total,' records','Cumulative initial routing',Number(aggregate.nli_quarantined_records_total)?'amber':'green')}${monitorMetric('Oldest pending',replay.oldest_queued_age_seconds??0,' s',Number(replay.queued)?'Oldest persisted replay candidate':'Replay queue is idle',Number(replay.queued)?'amber':'green')}${monitorMetric('Replay progress',replayProgress,'%',`${number(replay.resolved)} resolved · ${number(replay.reprocessing)} running · ${number(replay.queued)} queued`,'cyan')}</div>${panel('Source queues','Depth, capacity and backpressure are canonical source gauges.',table(['Node','Source','Depth','Capacity','Pressure','Oldest age'],queueRows,'No source queue telemetry is available.'))}`);
    const controller=snapshot.controller||{},clientInfrastructure=(snapshot.infrastructure?.clients||[]),clientCPU=clientInfrastructure.length?clientInfrastructure.reduce((sum,item)=>sum+Number(item.cpu_percent||0),0)/clientInfrastructure.length:null,clientMemory=clientInfrastructure.length?clientInfrastructure.reduce((sum,item)=>sum+Number(item.memory_percent||0),0)/clientInfrastructure.length:null;
    const infrastructure=monitoringPanel('infrastructure',`<div class="dashboard-kpi-strip monitoring-kpis">${monitorMetric('Client CPU avg',clientCPU,'%',clientInfrastructure.length?'Authenticated heartbeat samples':'Awaiting client host samples','cyan')}${monitorMetric('Client memory avg',clientMemory,'%',clientInfrastructure.length?'Authenticated heartbeat samples':'Awaiting client host samples','violet')}${monitorMetric('Controller memory',controller.process_memory_alloc_bytes,' B','Go process allocation','green')}${monitorMetric('Controller memory reserved',controller.process_memory_sys_bytes,' B','Go runtime system memory','blue')}${monitorMetric('CPU cores',controller.cpu_count,'','Controller logical processors','cyan')}${monitorMetric('Goroutines',controller.goroutines,'','Concurrent controller workers','green')}${monitorMetric('File descriptors',controller.open_file_descriptors,'','Linux controller process','blue')}${monitorMetric('Uptime',controller.uptime_seconds,' s','Controller process lifetime','violet')}</div>${panel('Client host samples','CPU, memory and load are collected from Linux kernel counters and sent in authenticated heartbeats.',table(['Node','CPU','Memory','Load 1 / 5 / 15','Updated'],clientInfrastructure.map(item=>`<tr><td>${esc(item.hostname||item.node_id)}</td><td>${metricText(item.cpu_percent,'%')}</td><td>${metricText(item.memory_percent,'%')}</td><td>${metricText(item.load_1m)} / ${metricText(item.load_5)} / ${metricText(item.load_15)}</td><td>${time(item.updated_at)}</td></tr>`),'No client host sample has arrived yet.'))}`);
    const latencyRows=sources.map(item=>`<tr><td>${esc(item.node_id||'—')}</td><td class="mono">${esc(item.source||'—')}</td><td>${metricText(item.controller_processing_ms,' ms')}</td><td>${metricText(item.observed_propagation_delay_ms,' ms')}</td><td>${metricText(item.freshness_age_ms,' ms')}</td><td>${syncStatusBadge(item.freshness)}</td><td>${esc(item.freshness_reason||'Within target')}</td></tr>`);
    const latency=monitoringPanel('latency',`<div class="dashboard-kpi-strip monitoring-kpis">${monitorMetric('Target',target,' ms','Measured freshness objective','cyan')}${monitorMetric('Propagation p50',diagnostics.observed_delay_p50_ms??diagnostics.propagation_delay_ms_p50,' ms','Client publish to controller observe','violet')}${monitorMetric('Propagation p95',propagationP95,' ms','Client publish to controller observe','violet')}${monitorMetric('Propagation p99',diagnostics.observed_delay_p99_ms??diagnostics.propagation_delay_ms_p99,' ms','Client publish to controller observe','violet')}${monitorMetric('Browser apply',state.sync.browserDelayMs,' ms','SSE event to shared store','cyan')}${monitorMetric('Missed targets',diagnostics.missed_freshness_targets??diagnostics.missed_deadlines??diagnostics.missed,'','Controller counter',Number(diagnostics.missed_freshness_targets??0)?'red':'green')}</div>${panel('Per-source synchronization','The reason column explains delayed, stale, disconnected and collection-failed state.',table(['Node','Source','Controller processing','Propagation','Freshness age','State','Reason'],latencyRows,'No synchronization samples match the selected filters.'))}${diagnosticsPanel()}`);
    const errors=monitoringPanel('errors',`${panel('Active errors and incidents','Diagnostics retain safe correlation context, reason, stage and recommended action.',issues.length?table(['Severity','Code / last seen','Stage / component','Node / source','Reason / root cause','Recommended action'],issues):empty('No active canonical incidents.'))}<div class="instrumentation-note"><b>Incident scope:</b> displayed issues are controller-classified metadata. Empty results mean no reported issue; they do not prove every component outside the canonical schema is healthy.</div>`);
    const napi=snapshot.subsystems?.napi||{},nal=snapshot.subsystems?.nal||{},release=snapshot.subsystems?.release_readiness||{};
    const releaseLabel=release.state==='drifted'?`${number(release.release_drifted_nodes)} drifted`:release.state==='no_release_targets'?'Idle':release.state||release.status;
    const assurance=monitoringPanel('assurance',`<div class="dashboard-kpi-strip monitoring-kpis">${monitorMetric('NAL total runs',nal.total,'','Persisted assurance runs','cyan')}${monitorMetric('NAL passed',nal.passed,'','Validated runs','green')}${monitorMetric('NAL failed',nal.failed,'','Failed runs',Number(nal.failed)?'red':'green')}${monitorMetric('NAL queued',nal.queued,'','Awaiting bounded worker',Number(nal.queued)?'amber':'blue')}${monitorMetric('NAPI requests',napi.requests_total,'','Synthesis requests','violet')}${monitorMetric('NAPI failures',napi.failures_total,'','Failed synthesis requests',Number(napi.failures_total)?'red':'green')}${monitorMetric('Pending drafts',napi.pending_drafts,'','Approval-gated proposals',Number(napi.pending_drafts)?'amber':'green')}${dashboardKPI('Release readiness',releaseLabel||'Not reported',`${number(release.managed_nodes)} managed · ${number(release.release_drifted_nodes)} drifted`,release.state==='drifted'?'red':'green')}</div>${panel('Subsystem health','NAL, NAPI and release evidence are part of the same canonical revision.',table(['Subsystem','State','Updated','Latest activity','Interpretation'],[`<tr><td>NAPI</td><td>${syncStatusBadge(napi.status)}</td><td>${time(napi.updated_at)}</td><td>${time(napi.latest_proposal_at)}</td><td>Bounded parser synthesis and draft state</td></tr>`,`<tr><td>NAL</td><td>${syncStatusBadge(nal.status)}</td><td>${time(nal.updated_at)}</td><td>${time(nal.latest_run_at)}</td><td>Durable assurance execution state</td></tr>`,`<tr><td>Release</td><td>${syncStatusBadge(release.status)}</td><td>${time(release.updated_at)}</td><td>${esc(releaseLabel||'Not reported')}</td><td>${esc(release.evidence||'Desired and active release evidence')}</td></tr>`]))}`);
    renderPage('monitoring',`<div class="monitoring-head"><div><p class="eyebrow">NLI observability · canonical live telemetry</p><h1>NLI Monitoring</h1><p>Fleet, pipelines, destinations, replay, infrastructure and assurance consume the same revisioned source of truth.</p></div><div class="monitoring-live">${syncStatusBadge(state.monitoringPaused?'paused':state.streamMode)}<strong>${metricText(snapshot.revision)}</strong><small>canonical revision</small></div></div>${monitoringFilters(allSources)}${monitoringTabs()}${overview}${fleet}${pipeline}${sourceView}${parserView}${destinations}${queues}${infrastructure}${latency}${errors}${assurance}`);
  }
  function renderSyncSubscriber({force=false}={}){
    if(!state.user||state.loadingPage||page.dataset.dirty==='true'||!SYNC_PAGES.has(state.page)||state.page==='monitoring'&&state.monitoringPaused)return;
    if(state.page==='operations'&&page.querySelector('[data-testid="live-dashboard-kpis"]')){scheduleDashboardCounters();return;}
    if(page.contains(document.activeElement)&&document.activeElement.matches?.('select')){state.sync.pendingRender=true;return;}
    if(state.sync.renderScheduled)return;
    const signature=syncRenderSignature();
    if(!force&&signature===state.sync.renderedSignature)return;
    state.sync.renderScheduled=true;
    requestAnimationFrame(()=>{
      state.sync.renderScheduled=false;
      if(!state.user||state.loadingPage||page.dataset.dirty==='true'||!SYNC_PAGES.has(state.page)||state.page==='monitoring'&&state.monitoringPaused)return;
      if(page.contains(document.activeElement)&&document.activeElement.matches?.('select')){state.sync.pendingRender=true;return;}
      const latest=syncRenderSignature();
      if(!force&&latest===state.sync.renderedSignature)return;
      const scrollY=window.scrollY,active=document.activeElement,activeID=active?.id,activeName=active?.getAttribute?.('name'),subsystemTab=page.querySelector('[data-subsystem-tab].active')?.dataset.subsystemTab;
      if(state.page==='operations')renderLiveOperationsDashboard(syncDashboardSnapshot());else if(state.page==='monitoring')renderMonitoring();else if(state.page==='streams')loadStreams();else if(state.page==='performance')loadPerformance();else if(state.page==='sync-fabric')renderSyncFabric();else renderSubsystemPage(state.page);
      if(subsystemTab){const tab=[...page.querySelectorAll('[data-subsystem-tab]')].find(item=>item.dataset.subsystemTab===subsystemTab);tab?.click();}
      const focusTarget=activeID?page.querySelector(`#${CSS.escape(activeID)}`):activeName?[...page.querySelectorAll('[name]')].find(item=>item.getAttribute('name')===activeName):null;
      if(focusTarget){focusTarget.focus({preventScroll:true});if(typeof active?.selectionStart==='number'&&typeof focusTarget.setSelectionRange==='function')focusTarget.setSelectionRange(active.selectionStart,active.selectionEnd);}
      window.scrollTo(0,scrollY);
      applyPermissions();state.sync.renderedSignature=latest;
    });
  }
  function applyServerSubsystemUpdate(name,payload) { if(!state.sync.snapshot||!payload||typeof payload!=='object')return;state.sync.snapshot={...state.sync.snapshot,subsystems:{...(state.sync.snapshot.subsystems||{}),[name]:{...(state.sync.snapshot.subsystems?.[name]||{}),...payload}}};state.loadedAt=Date.now();renderSyncSubscriber(); }

  const replayCardFields=[
    ['queued','Awaiting replay','Queued for recovery','amber'],
    ['reprocessing','Reprocessing','Replay in progress','blue'],
    ['resolved','Reprocessed successfully','Successful replay confirmed by the controller','green'],
    ['manual_attention','Manual attention','Records requiring operator review','red'],
  ];
  function replayCount(snapshot,node,key){
    if(!['ready','partial'].includes(snapshot?.status))return null;
    if(node==='all')return snapshot[key];
    if(!snapshot.by_node)return null;
    return (Object.hasOwn(snapshot.by_node,node)?snapshot.by_node[node]?.[key]:null)??0;
  }
  function replayCountText(snapshot,node,key){
    const value=replayCount(snapshot,node,key);
    return snapshot?.status==='partial'&&present(value)?'≥ '+metricText(value):metricText(value);
  }
  function renderReplayCounts(snapshot,node){
    return `<section class="panel" data-testid="dashboard-replay">
      <div class="section-head"><div><h3>Quarantine and replay</h3>
      <p>Current records retained by the controller for the selected client. The historical time range applies to ingestion and routing above.</p>
      <p class="hint" data-replay-partial ${snapshot?.status==='partial'?'':'hidden'}>The controller checked a bounded set of records. Counts shown with ≥ may increase when the full inventory is available.</p></div></div>
      <div class="dashboard-kpi-strip">${replayCardFields.map(([key,label,note,tone])=>
        `<article class="dashboard-kpi ${tone}"><span>${label}</span>
        <strong data-replay-counter="${key}">${replayCountText(snapshot,node,key)}</strong><small title="${note}">${note}</small></article>`
      ).join('')}</div>
    </section>`;
  }
  function renderLiveOperationsDashboard(snapshot,warnings=[]) {
    const overview=snapshot.overview||{},metadata=snapshot.metadata||overview.metadata||{},timeseries=state.dashboardHistory?.query===dashboardSelectedQuery()&&state.dashboardHistory?.points?{...(snapshot.timeseries||{}),from:state.dashboardHistory.from,to:state.dashboardHistory.to,points:state.dashboardHistory.points}:dashboardSelectedQuery()?snapshot.timeseries||{}:{points:[]},system=snapshot.system||{},ai=snapshot.ai_usage||{};
    const fleetSync=snapshot.fleet_sync||{},fleetEntries=fleetSync.nodes||[],allNodes=fleetEntries.map(item=>item.node||item).filter(Boolean);
    const selectedNodes=state.dashboardClient==='all'?allNodes:allNodes.filter(node=>node.id===state.dashboardClient);
    const streamFiles=(snapshot.stream_files||[]).filter(item=>state.dashboardClient==='all'||item.node_id===state.dashboardClient);
    const records=(metadata.records||[]).map(item=>item.record||item).filter(Boolean);
    const rawPoints=[...(timeseries.points||[])].sort((a,b)=>Date.parse(a.timestamp||'')-Date.parse(b.timestamp||''));
    const buckets=rawPoints.map(point=>{const stamp=Date.parse(point.timestamp||''),counts=point.exact_counts||{};return {timestamp:point.timestamp,label:Number.isFinite(stamp)?new Date(stamp).toLocaleString([],state.dashboardRange==='1h'||state.dashboardRange==='6h'?{hour:'2-digit',minute:'2-digit'}:{month:'short',day:'numeric',hour:'2-digit'}):'Unknown time',received:String(counts.received??point.logs_received??0),parsed:String(counts.parsed??point.logs_parsed??0),quarantine:String(counts.quarantine??point.logs_quarantined??0),errors:String(counts.errors??point.errors??0),latency:Number(point.parse_latency_ms_avg||0),parseRate:Number(point.parse_rate_pct||0),ai:Number(point.ai_requests||0)};});
    const labels=buckets.map(item=>item.label),totals=dashboardSelectedCounts(),received=totals?.received??null,parsed=totals?.parsed??null,quarantined=totals?.quarantine??null,errors=totals?.errors??null;
    const nonzeroLatency=buckets.filter(item=>item.latency>0),latency=nonzeroLatency.length?nonzeroLatency.reduce((total,item)=>total+item.latency,0)/nonzeroLatency.length:null;
    const parseRate=present(received)&&Number(received)>0&&present(parsed)?Number(parsed)/Number(received)*100:present((overview.fleet||{}).parse_rate_pct)?Number(overview.fleet.parse_rate_pct):present(received)&&Number(received)===0&&present(parsed)?0:null,quarantineRate=present(received)&&Number(received)>0&&present(quarantined)?Number(quarantined)/Number(received)*100:present(received)&&Number(received)===0&&present(quarantined)?0:null,errorRate=present(received)&&Number(received)>0&&present(errors)?Number(errors)/Number(received)*100:present(received)&&Number(received)===0&&present(errors)?0:null;
    const fleetStats=overview.fleet||timeseries.fleet?.stats||{},online=allNodes.filter(node=>/online|active|healthy/i.test(node.status||'')).length,degraded=allNodes.filter(node=>/degraded|error/i.test(node.status||'')||(node.issues||[]).length).length;
    const selectedMetrics=selectedNodes.map(node=>node.metrics||{}),sourceTotal=key=>streamFiles.some(item=>present(item[key]))?streamFiles.reduce((total,item)=>total+(present(item[key])&&Number.isFinite(Number(item[key]))?Number(item[key]):0),0):null,liveReadRate=sourceTotal('read_lines_per_second'),liveOutputRate=sourceTotal('normalized_per_second'),liveQuarantineRate=sourceTotal('quarantined_per_second');
    const backpressured=streamFiles.filter(item=>item.backpressure).length,queueDepth=sourceTotal('queue_depth'),queueCapacity=sourceTotal('queue_capacity');
    const nsToMs=value=>present(value)&&Number.isFinite(Number(value))?Number(value)/1000000:null,pipeline={read:sourceTotal('read_duration_ns')===null?null:sourceTotal('read_duration_ns')/1000000,parse:sourceTotal('parse_duration_ns')===null?null:sourceTotal('parse_duration_ns')/1000000,output:sourceTotal('output_duration_ns')===null?null:sourceTotal('output_duration_ns')/1000000};pipeline.total=[pipeline.read,pipeline.parse,pipeline.output].some(present)?[pipeline.read,pipeline.parse,pipeline.output].reduce((n,value)=>n+(value||0),0):null;
    const pipelineBar=(label,value,color)=>`<div class="pipeline-row"><span>${esc(label)}</span><div><i style="width:${present(pipeline.total)&&pipeline.total?Math.max(1,Number(value||0)/pipeline.total*100):0}%;background:${color}"></i></div><b>${present(value)?`${Number(value).toFixed(1)} ms`:'—'}</b></div>`;
    const familyMap=new Map();records.forEach(record=>{const key=String(record.service||record.parser_id||'unclassified').split('/')[0]||'unclassified';if(!familyMap.has(key))familyMap.set(key,{name:key,received:0,parsed:0,quarantine:0,errors:0,latency:0,latencyCount:0,trend:Array(12).fill(0)});const item=familyMap.get(key);item.received++;if(/normal|parsed|success|ok/i.test(`${record.route||''} ${record.result||''}`))item.parsed++;if(/quarantine|unknown/i.test(`${record.route||''} ${record.result||''}`))item.quarantine++;if((record.error_codes||[]).length||/error|fail/i.test(`${record.route||''} ${record.result||''}`))item.errors++;if(Number(record.latency_ms)>0){item.latency+=Number(record.latency_ms);item.latencyCount++;}const stamp=Date.parse(record.timestamp||record.processed_at||record.received_at||''),age=Number.isFinite(stamp)?Math.max(0,Date.now()-stamp):0,windowMinutes={"1h":60,"6h":360,"24h":1440,"7d":10080,"30d":43200}[state.dashboardRange]||60,bucketWidth=Math.max(1,windowMinutes*60000/12),itemIndex=Math.max(0,Math.min(11,11-Math.floor(age/bucketWidth)));item.trend[itemIndex]++;});
    const familyRows=[...familyMap.values()].sort((a,b)=>b.received-a.received).slice(0,20).map(item=>`<tr><td><b>${esc(item.name)}</b></td><td>${number(item.received)}</td><td>${number(item.parsed)}</td><td>${number(item.quarantine)}</td><td>${item.received?(item.parsed/item.received*100).toFixed(2):'0.00'}%</td><td>${item.latencyCount?(item.latency/item.latencyCount).toFixed(2):'—'} ms</td><td>${compactTrend(item.trend,item.quarantine||item.errors?'#f8c15c':'#3dd598')}</td></tr>`);
    const clientRows=selectedNodes.map(node=>{const metrics=node.metrics||{},issues=node.issues||[],percent=key=>present(metrics[key])&&Number.isFinite(Number(metrics[key]))?`${Number(metrics[key]).toFixed(1)}%<div class="health-meter"><i style="width:${Math.max(0,Math.min(100,Number(metrics[key])))}%"></i></div>`:'—';return `<tr><td><b>${esc(node.hostname||node.id||'Unknown')}</b><br><small class="mono">${esc(node.id||'—')}</small></td><td>${badge(node.status||'unknown')}<br><small>${time(node.last_seen)}</small></td><td>${percent('cpu_percent')}</td><td>${percent('memory_percent')}</td><td>${metricText(metrics.parse_rate_pct,'%')}</td><td>${metricText(metrics.parse_latency_ms,' ms')}</td><td>${issues.length?`<span class="bad-text">${esc(issues.join(' · '))}</span>`:'No reported issues'}</td></tr>`;});
    const streamRows=streamFiles.slice(0,256).map(item=>{const processed=present(item.normalized)||present(item.quarantined)?Number(item.normalized||0)+Number(item.quarantined||0):null;return `<tr><td><b>${esc(item.node_id||'—')}</b></td><td class="mono">${esc(item.source||'—')}</td><td>${badge(item.backpressure?'backpressure':item.phase||'waiting')}</td><td>${metricText(item.read_lines_per_second)}</td><td>${metricText(item.normalized_per_second)}</td><td>${metricText(item.quarantined_per_second)}</td><td>${metricText(item.read_lines)} / ${metricText(processed)}</td><td>${metricText(item.normalized)} / ${metricText(item.quarantined)}</td><td>${metricText(item.errors)}</td><td>${metricText(item.queue_depth)} / ${metricText(item.queue_capacity)}</td><td>${time(item.updated_at)}</td></tr>`;});
    const failureRows=records.filter(record=>(record.error_codes||[]).length||/error|fail|quarantine/i.test(`${record.route||''} ${record.result||''}`)).slice(0,50).map(record=>`<tr><td>${time(record.timestamp||record.processed_at||record.received_at)}</td><td>${esc(record.node_id||'—')}</td><td class="mono">${esc(record.parser_id||'—')}</td><td>${badge(record.route||record.result||'error')}</td><td>${esc((record.error_codes||[]).join(', ')||'No error code supplied')}</td></tr>`);
    const clientOptions=allNodes.map(node=>`<option value="${esc(node.id)}" ${state.dashboardClient===node.id?'selected':''}>${esc(node.hostname||node.id)}</option>`).join('');
    const warningHTML=warnings.filter(Boolean).map(message=>`<div class="callout warn">${esc(message)}</div>`).join('');
    const rangeOptions=[['1h','1 hour'],['6h','6 hours'],['12h','12 hours'],['24h','24 hours'],['2d','2 days'],['7d','1 week'],['14d','2 weeks'],['2w','2 weeks'],['4w','4 weeks'],['30d','30 days'],['90d','90 days'],['3mo','3 months'],['6mo','6 months'],['9mo','9 months'],['12mo','12 months'],['1y','1 year'],['2y','2 years'],['5y','5 years'],['10y','10 years'],['all','All retained history'],['custom','Custom dates']].map(([value,label])=>`<option value="${value}" ${state.dashboardRange===value?'selected':''}>${label}</option>`).join('');
    const customControls=state.dashboardRange==='custom'?`<div class="history-custom-range"><label>From<input id="dashboard-from" type="datetime-local" value="${esc(state.dashboardFrom)}"></label><label>To<input id="dashboard-to" type="datetime-local" value="${esc(state.dashboardTo)}"></label><button class="primary small" type="button" data-action="dashboard-history-apply">Apply dates</button></div>`:'';
    const historyActions=state.dashboardHistory?.points?.length?`<button class="secondary small" type="button" data-action="dashboard-history-download">Download JSON</button>`:'';
    const historyJson=state.dashboardHistory?.points?.length?`<details id="dashboard-history-json" class="history-json"><summary>View returned JSON</summary><pre class="json-output">${esc(pretty(state.dashboardHistory))}</pre></details>`:'';
    state.dashboard={...snapshot,overview,metadata,timeseries,system,ai_usage:ai,fleet_sync:fleetSync,nodes:allNodes,stream_files:snapshot.stream_files||[],buckets,refreshed_at:new Date().toISOString()};
    renderPage('operations',`${warningHTML}<div class="dashboard-intro grafana-head"><div><p class="eyebrow">NLI security operations · canonical live telemetry</p><h1>Command Center</h1><p>Live fleet state and retained historical measurements in one view. Customer log bodies are excluded.</p></div><div class="dashboard-controls"><label>Historical range<select id="dashboard-range">${rangeOptions}</select></label>${customControls}<label>Client<select id="dashboard-client"><option value="all">All clients</option>${clientOptions}</select></label><span class="live-pill"><i></i>${state.streamMode==='live'?'Incremental live stream':'Last-known snapshot'} · target 1s</span></div></div><div id="dashboard-history-error" class="callout warn" role="alert" ${state.dashboardHistoryError?'':'hidden'}>${state.dashboardHistoryError?`Historical measurements could not load: ${esc(state.dashboardHistoryError)}`:''}</div><div class="dashboard-kpi-strip" data-testid="live-dashboard-kpis">${dashboardKPI('Input records',metricText(received),`${metricText(liveReadRate,' lines/s')} now`)}${dashboardKPI('Parsed',metricText(parsed),`${fixedMetric(parseRate,2,'%')} success`,'green')}${dashboardKPI('Quarantined',metricText(quarantined),`${fixedMetric(quarantineRate,2,'%')} initially routed`,Number(quarantined)>0?'amber':'green')}${dashboardKPI('Errors',metricText(errors),`${fixedMetric(errorRate,2,'%')} error rate`,Number(errors)>0?'red':'green')}${dashboardKPI('Propagation p95',metricText(snapshot.sync_diagnostics?.propagation_delay_ms_p95??snapshot.sync_diagnostics?.p95_ms,' ms'),'Measured synchronization path','cyan')}${dashboardKPI('Reporting nodes',`${number(online)} / ${metricText(fleetStats.total??(allNodes.length||null))}`,`${number(degraded)} degraded`,degraded?'amber':'green')}</div>${renderReplayCounts(snapshot.replay,state.dashboardClient)}<div class="dashboard-layout"><section class="panel dashboard-wide"><div class="section-head"><div><h3>Ingest, parsed, quarantine and errors</h3><p>Exact record counts per controller-time bucket. Hover or focus a point for its count.</p></div><span id="dashboard-history-bounds" class="metric-note">${esc(timeseries.from?`${time(timeseries.from)} → ${time(timeseries.to)}`:'Awaiting retained history')}</span>${historyActions}</div><div id="dashboard-history-content">${multiSeriesChart(labels,[{label:'Input',values:buckets.map(item=>item.received),color:'#53a5ff',unit:'records / bucket'},{label:'Parsed',values:buckets.map(item=>item.parsed),color:'#3dd598',unit:'records / bucket'},{label:'Quarantine',values:buckets.map(item=>item.quarantine),color:'#f8c15c',unit:'records / bucket'},{label:'Errors',values:buckets.map(item=>item.errors),color:'#ff6b7d',unit:'records / bucket'}])}</div>${historyJson}</section><section class="panel dashboard-side"><div class="section-head"><div><h3>Live source throughput</h3><p>Current canonical source state.</p></div></div><div class="throughput-gauges">${dashboardKPI('Read',`${fixedMetric(liveReadRate,1,'/s')}`,'Input lines','blue')}${dashboardKPI('Output',`${fixedMetric(liveOutputRate,1,'/s')}`,'Normalised lines','green')}${dashboardKPI('Quarantine',`${fixedMetric(liveQuarantineRate,1,'/s')}`,'Waiting review',Number(liveQuarantineRate)>0?'amber':'green')}${dashboardKPI('Queue',metricText(queueDepth),`${number(backpressured)} backpressured`,backpressured?'red':'blue')}</div></section><section class="panel dashboard-half"><div class="section-head"><div><h3>Output route volume</h3><p>Parsed, quarantine and error totals with exact hover details.</p></div></div>${stackedVolumeChart(labels,buckets)}</section><section class="panel dashboard-half"><div class="section-head"><div><h3>Propagation latency</h3><p>Measured end-to-end telemetry propagation latency.</p></div></div>${nonzeroLatency.length?sparkline(labels,buckets.map(item=>item.latency),'#36d7db'):empty('No propagation latency samples are available in this snapshot.')}</section><section class="panel dashboard-half"><div class="section-head"><div><h3>Pipeline time travel</h3><p>Read → parse → output duration from active sources.</p></div></div><div class="pipeline-bars">${pipelineBar('Read',pipeline.read,'#53a5ff')}${pipelineBar('Parse',pipeline.parse,'#8b7cff')}${pipelineBar('Output',pipeline.output,'#3dd598')}</div></section><section class="panel dashboard-half"><div class="section-head"><div><h3>Controller and sync health</h3><p>Authoritative controller and browser delivery state.</p></div></div><div class="controller-health-grid">${dashboardKPI('Status',system.health||system.status||'Not reported',system.version||'Controller health was not included in this snapshot',/ok|healthy/i.test(system.health||system.status||'')?'green':'amber')}${dashboardKPI('Sync revision',number(state.sync.snapshot?.revision),'Canonical sequence','cyan')}${dashboardKPI('Browser apply',metricText(state.sync.browserDelayMs,' ms'),'After event delivery','violet')}${dashboardKPI('Freshness',syncStatus(state.sync.snapshot?.diagnostics?.fleet_freshness||state.streamMode),time(state.sync.snapshot?.generated_at),'blue')}</div></section><section class="panel dashboard-wide"><div class="section-head"><div><h3>Client resource and parser health</h3><p>Latest authenticated state for each selected client.</p></div></div>${table(['Client','Heartbeat','CPU','Memory','Parse rate','Latency','Issues'],clientRows,'No client heartbeat matches the current filter.')}</section><section class="panel dashboard-wide"><div class="section-head"><div><h3>Live sources and backpressure</h3><p>Source, rates, cumulative processing totals, queue, EOF and freshness state without raw event content.</p></div></div>${table(['Node','Input source','Phase','Read/s','Output/s','Quarantine/s','Read / processed','Parsed / quarantine','Errors','Queue','Updated'],streamRows,'No live source telemetry is available yet.')}</section>${diagnosticsPanel()}</div>`);
  }

  function dashboardHistoryQuery() {
    const params=new URLSearchParams();
    if(state.dashboardRange==='custom'){
      if(!state.dashboardFrom||!state.dashboardTo)throw new Error('Choose both start and end dates for a custom range.');
      const from=new Date(state.dashboardFrom),to=new Date(state.dashboardTo);
      if(!Number.isFinite(from.getTime())||!Number.isFinite(to.getTime())||from>=to)throw new Error('The start date must be earlier than the end date.');
      params.set('from',from.toISOString());params.set('to',to.toISOString());
    } else params.set('range',state.dashboardRange);
    params.set('step','auto');
    if(state.dashboardClient!=='all')params.set('node_id',state.dashboardClient);
    return params.toString();
  }
  function dashboardSelectedQuery(){try{return dashboardHistoryQuery();}catch{return null;}}
  function dashboardSelectedCounts() {
    const history=state.dashboardHistory;
    if(!history||history.query!==dashboardSelectedQuery())return null;
    const totals={received:0n,parsed:0n,quarantine:0n,errors:0n};
    const count=value=>{try{return /^\d+$/.test(String(value))?BigInt(value):0n;}catch{return 0n;}};
    for(const point of history.points||[])for(const key of Object.keys(totals))totals[key]+=count(point.exact_counts?.[key]??point[key]??0);
    // Add only increments after the controller's atomic history checkpoint.
    // Periodic rebasing expires rolling-window buckets and repairs reconnects.
    const baseline=history.canonical_history?.counter_baseline,observed=Date.parse(history.canonical_history?.observed_at||'');
    if(baseline&&Number.isFinite(observed))for(const source of state.sync.snapshot?.sources||[]){
      if(state.dashboardClient!=='all'&&source.node_id!==state.dashboardClient)continue;
      const receivedAt=Date.parse(source.controller_received_at||source.updated_at||'');
      if(!Number.isFinite(receivedAt)||receivedAt<=observed)continue;
      const minute=Math.floor(receivedAt/60000)*60000;
      if(state.dashboardRange==='custom'&&(minute<Date.parse(history.from)||minute>=Date.parse(history.to)))continue;
      const old=baseline[`${source.node_id||''}\u0000${source.source||''}`],same=old?.generation===source.process_generation;
      for(const [key,field,previous] of [['received','read_lines','read'],['parsed','normalized','parsed'],['quarantine','quarantined','quarantine'],['errors','errors','errors']]){
        const alias={read_lines:'nli_source_records_read_total',normalized:'nli_normalized_records_total',quarantined:'nli_quarantined_records_total',errors:'nli_parser_records_failed_total'}[field];
        const current=count(sourceValue(source,field)??sourceValue(source,alias)),prior=same?count(old[previous]):0n;
        totals[key]+=current>=prior?current-prior:current;
      }
    }
    return totals;
  }
  function scheduleDashboardCounters(){
    if(state.dashboardCounterFrame)return;
    state.dashboardCounterFrame=requestAnimationFrame(()=>{state.dashboardCounterFrame=null;updateDashboardCounters();});
  }
  function updateDashboardCounters(){
    if(state.page!=='operations')return;
    const totals=dashboardSelectedCounts(),root=page.querySelector('[data-testid="live-dashboard-kpis"]');if(!root)return;
    const rows=[...root.querySelectorAll('.dashboard-kpi')],keys=['received','parsed','quarantine','errors'];
    keys.forEach((key,index)=>{const strong=rows[index]?.querySelector('strong');if(strong)strong.textContent=metricText(totals?.[key]);});
    const replay=state.sync.snapshot?.subsystems?.replay??state.dashboard?.replay;
    for(const counter of page.querySelectorAll('[data-replay-counter]'))counter.textContent=replayCountText(replay,state.dashboardClient,counter.dataset.replayCounter);
    const partial=page.querySelector('[data-replay-partial]');
    if(partial)partial.hidden=replay?.status!=='partial';
    const sources=(state.sync.snapshot?.sources||[]).filter(source=>state.dashboardClient==='all'||source.node_id===state.dashboardClient);
    const rate=key=>sources.length?sources.reduce((total,source)=>total+Number(sourceValue(source,key)||0),0):null;
    const readRate=rate('read_lines_per_second'),readNote=rows[0]?.querySelector('small');if(readNote)readNote.textContent=`${metricText(readRate,' lines/s')} now`;
    const gauges=page.querySelectorAll('.throughput-gauges .dashboard-kpi strong');
    ['read_lines_per_second','normalized_per_second','quarantined_per_second'].forEach((key,index)=>{if(gauges[index])gauges[index].textContent=fixedMetric(rate(key),1,'/s');});
    if(gauges[3])gauges[3].textContent=metricText(sources.length?sources.reduce((total,source)=>total+Number(sourceValue(source,'queue_depth')??sourceValue(source,'nli_queue_depth')??0),0):null);
    const queueNote=gauges[3]?.parentElement.querySelector('small');if(queueNote)queueNote.textContent=`${sources.filter(source=>sourceValue(source,'backpressure')).length} backpressured`;
    for(const [index,key,suffix] of [[1,'parsed','success'],[2,'quarantine','initially routed'],[3,'errors','error rate']]){const note=rows[index]?.querySelector('small');if(note)note.textContent=totals?`${totals.received>0n?(Number(totals[key])*100/Number(totals.received)).toFixed(2):'0.00'}% ${suffix}`:'Awaiting selected-range history';}
  }
  function dashboardHistoryChart(history=state.dashboardHistory) {
    if(history?.query!==dashboardSelectedQuery())history=null;
    const points=[...(history?.points||[])].sort((a,b)=>Date.parse(a.timestamp||'')-Date.parse(b.timestamp||''));
    const buckets=points.map(point=>{const stamp=Date.parse(point.timestamp||''),counts=point.exact_counts||{};return {timestamp:point.timestamp,label:Number.isFinite(stamp)?new Date(stamp).toLocaleString([],state.dashboardRange==='1h'||state.dashboardRange==='6h'||state.dashboardRange==='12h'?{hour:'2-digit',minute:'2-digit'}:{month:'short',day:'numeric',hour:'2-digit'}):'Unknown time',received:String(counts.received??point.logs_received??0),parsed:String(counts.parsed??point.logs_parsed??0),quarantine:String(counts.quarantine??point.logs_quarantined??0),errors:String(counts.errors??point.errors??0)};});
    const labels=buckets.map(item=>item.label);
    return {points,buckets,labels,chart:multiSeriesChart(labels,[{label:'Input',values:buckets.map(item=>item.received),color:'#53a5ff',unit:'records / bucket'},{label:'Parsed',values:buckets.map(item=>item.parsed),color:'#3dd598',unit:'records / bucket'},{label:'Quarantine',values:buckets.map(item=>item.quarantine),color:'#f8c15c',unit:'records / bucket'},{label:'Errors',values:buckets.map(item=>item.errors),color:'#ff6b7d',unit:'records / bucket'}])};
  }
  function renderDashboardHistoryOnly() {
    if(state.page!=='operations')return;
    updateDashboardCounters();
    const root=el('dashboard-history-content'),bounds=el('dashboard-history-bounds'),error=el('dashboard-history-error');
    if(root){const view=dashboardHistoryChart();root.innerHTML=view.points.length?view.chart:empty('No retained measurements are available for this time range.');}
    if(bounds)bounds.textContent=state.dashboardHistory?.from?`${time(state.dashboardHistory.from)} → ${time(state.dashboardHistory.to)}`:'Awaiting retained history';
    if(error){error.hidden=!state.dashboardHistoryError;error.textContent=state.dashboardHistoryError?`Historical measurements could not refresh: ${state.dashboardHistoryError}`:'';}
    const json=page.querySelector('#dashboard-history-json pre');if(json&&state.dashboardHistory)json.textContent=pretty(state.dashboardHistory);
  }
  async function loadDashboardHistory({force=false}={}) {
    const query=dashboardHistoryQuery(),requestID=(state.dashboardHistoryRequestID||0)+1;state.dashboardHistoryRequestID=requestID;
    if(!force&&state.dashboardHistory?.query===query&&Date.now()-state.dashboardHistory.loadedAt<60_000)return state.dashboardHistory;
    try{const result=await api(`/metrics/timeseries?${query}`);if(requestID!==state.dashboardHistoryRequestID||query!==dashboardHistoryQuery())return state.dashboardHistory;const exact=new Map((result.canonical_history?.points||[]).map(point=>[point.timestamp,point]));const points=(result.points||[]).map(point=>{const canonical=exact.get(point.timestamp)||{};return {...point,exact_counts:{received:canonical.read_delta??String(point.received??point.logs_received??0),parsed:canonical.parsed_delta??String(point.parsed??point.logs_parsed??0),quarantine:canonical.quarantine_delta??String(point.quarantine??point.logs_quarantined??0),errors:canonical.error_delta??String(canonical.errors??point.errors??0)}};});state.dashboardHistory={...result,points,query,loadedAt:Date.now()};state.dashboardHistoryError='';return state.dashboardHistory;}
    catch(error){if(requestID===state.dashboardHistoryRequestID&&query===dashboardHistoryQuery())state.dashboardHistoryError=error.message;throw error;}
  }
  async function applyDashboardHistory({force=true}={}) {
    if(!state.dashboard)return;
    try{await loadDashboardHistory({force});}catch(error){state.dashboardHistoryError=error.message;}
    if(state.page==='operations')renderLiveOperationsDashboard(state.sync.snapshot?syncDashboardSnapshot():state.dashboard);
  }
  async function refreshDashboardHistoryInBackground() {
    if(!state.user||state.page!=='operations'||document.hidden||state.loadingPage||page.dataset.dirty==='true'||state.dashboardHistoryRefreshInFlight)return;
    state.dashboardHistoryRefreshInFlight=true;
    try{await loadDashboardHistory({force:true});
      const active=document.activeElement,editing=page.contains(active)&&active.matches?.('input,select,textarea,button');
      if(state.page==='operations'&&state.sync.snapshot&&!editing)renderLiveOperationsDashboard(syncDashboardSnapshot());else renderDashboardHistoryOnly();
    }
    catch{renderDashboardHistoryOnly();}
    finally{state.dashboardHistoryRefreshInFlight=false;}
  }
  async function loadOperations() {
    try{await loadDashboardHistory();}catch{}
    if(state.sync.snapshot){renderLiveOperationsDashboard(syncDashboardSnapshot());return;}
    const nodeQuery=state.dashboardClient==='all'?'':`&node_id=${encodeURIComponent(state.dashboardClient)}`,warnings=[];
    try {
      const snapshot=await api(`/dashboard/snapshot?${dashboardHistoryQuery()}`);
      renderLiveOperationsDashboard(snapshot);
      return;
    } catch(error) { warnings.push(`Combined live snapshot unavailable; using compatible controller APIs. ${error.message}`); }
    const [overview,metadata,system,ai,fleet,timeseries,streams]=await Promise.allSettled([api('/overview'),api('/metadata?limit=500'),api('/system'),api('/admin/ai-usage'),api('/fleet/sync'),api(`/metrics/timeseries?${dashboardHistoryQuery()}`),api('/telemetry/stream/snapshot')]);
    if(overview.status==='rejected'&&timeseries.status==='rejected')throw overview.reason;
    const value=result=>result.status==='fulfilled'?result.value:{};
    [metadata,system,ai,fleet,timeseries,streams].filter(result=>result.status==='rejected').forEach(result=>warnings.push(result.reason.message));
    renderLiveOperationsDashboard({overview:value(overview),metadata:value(metadata),system:value(system),ai_usage:value(ai),fleet_sync:value(fleet),timeseries:value(timeseries),stream_files:value(streams).files||[]},warnings);
  }

  const ASSURANCE_FAMILIES = [
    ['auto','Automatic family detection · all approved ParserPacks'],['imap','IMAP · ImapDisconnected and IMAP events'],['pop','POP · AcctBadPswd and POP events'],
    ['webtop','Webtop / Webmail · auth.loginByToken'],['mta','MTA · MtaRequestingDomainTableUpdate'],['pabd','PSS / PABD · CARDDAV'],
    ['capd','MSS / CAPD'],['smtp-in','SMTP inbound · SieveRuleMsg'],['smtp-out','SMTP outbound'],['custom','Custom family…']
  ];
  function assuranceFamilySelect(name, selected='auto') { return `<select name="${name}" title="Choose a reviewed family, Automatic to evaluate every eligible approved parser, or Custom for a new family.">${ASSURANCE_FAMILIES.map(([value,label])=>`<option value="${esc(value)}" ${value===selected?'selected':''}>${esc(label)}</option>`).join('')}</select>`; }
  function normalizeParserCreationHelper(helper, request) {
    const source=typeof helper.normalized_sample==='string'?helper.normalized_sample:String(request.raw||'');
    const normalized=source.replace(/\r\n?/g,'\n').split('\n').map(line=>line.replace(/[\t ]+$/,'')).join('\n').replace(/\n+$/,'');
    const sampleLines=normalized.split('\n').filter(line=>line.trim()).length;
    const fields=(Array.isArray(helper.fields)?helper.fields:[]).map(field=>typeof field==='string'?{name:field,source:`group:${field}`,type:'string'}:field&&typeof field==='object'?field:null).filter(field=>field&&typeof field.name==='string'&&field.name.trim()).map(field=>({...field,name:field.name.trim()}));
    const fieldNames=fields.map(field=>field.name),projection=Array.isArray(helper.projection)?helper.projection.filter(value=>typeof value==='string'):fieldNames;
    const scope=helper.match_scope&&typeof helper.match_scope==='object'?helper.match_scope:{kind:String(helper.match_scope||'line')};
    const template=helper.template&&typeof helper.template==='object'?helper.template:(typeof helper.template==='string'?helper.template:Object.fromEntries(fieldNames.map(name=>[name,`{${name}}`])));
    return {normalized,sampleLines,fieldNames,projection,scope,template,parser:{pattern_type:'regex',pattern:String(helper.suggested_regex||''),method:String(helper.selected_strategy||''),header_pattern:scope,kv_delimiters:Array.isArray(helper.kv_delimiters)?helper.kv_delimiters:['='],known_fields:fieldNames,projection,template,field_mapping:Object.fromEntries(fields.map(field=>[field.name,field.source||field.name])),fields}};
  }
  function assuranceParserSelect(runtimeParsers, selected='auto', name='parser_id') { const parsers=[...(runtimeParsers||[])].sort((a,b)=>String(a.id).localeCompare(String(b.id)));return `<select name="${esc(name)}" title="Automatic uses the full approved parser pipeline. Exact choices run only that live runtime parser."><option value="auto" ${!selected||selected==='auto'?'selected':''}>Automatic parser detection · evaluate all eligible parsers</option>${parsers.map(item=>`<option value="${esc(item.id)}" ${item.id===selected?'selected':''}>${esc(item.id)} · ${esc(item.description||item.format||'runtime parser')}</option>`).join('')}</select>`; }
  function selectedAssuranceService(data) { const value=String(data.service||'auto').trim();if(value!=='custom')return value;const custom=String(data.custom_service||'').trim();if(!custom)throw new Error('Enter the custom parser family name');if(!/^[A-Za-z0-9._-]{1,64}$/.test(custom))throw new Error('Custom family must use 1..64 letters, digits, dots, underscores, or hyphens');return custom; }

  async function loadAssurance({restoreQRW=false,pageID='assurance'}={}) {
    const read = path => api(path, { timeoutMs: 12000 });
    const [casesResult, parsersResult, historyResult, runtimeResult, packsResult, syncResult, fleetResult, aiProposalsResult, plansResult, runsResult, repairQueueResult, quarantineHistoryResult, rolloutsResult] = await Promise.allSettled([read('/assurance/cases'), read('/assurance/parsers'), read('/assurance/history'), read('/parsers'), read('/parser-packs'), read('/parser-sync'), read('/fleet/sync'), read('/assurance/ai-proposals'), read('/assurance/plans'), read('/assurance/runs'), read('/quarantine/records?limit=100'), read('/quarantine/history?limit=100'), read('/rollouts')]);
    if(state.page!==pageID)return;
    const cases = casesResult.status === 'fulfilled' ? casesResult.value : null;
    const parsers = parsersResult.status === 'fulfilled' ? parsersResult.value.parsers || [] : [];
    const history = historyResult.status === 'fulfilled' ? historyResult.value.events || [] : [];
    const runtimeParsers = runtimeResult.status === 'fulfilled' ? runtimeResult.value.parsers || runtimeResult.value || [] : [];
    const managedPacks = packsResult.status === 'fulfilled' ? packsResult.value.parser_packs || [] : [];
    const syncAssets = syncResult.status === 'fulfilled' ? syncResult.value.assets || [] : [];
    const syncClients = syncResult.status === 'fulfilled' ? syncResult.value.clients || [] : [];
    const syncTelemetry = syncResult.status === 'fulfilled' ? syncResult.value.telemetry || null : null;
    const fleetNodes = fleetResult.status === 'fulfilled' ? fleetResult.value.nodes || [] : [];
    const aiProposals = aiProposalsResult.status === 'fulfilled' ? aiProposalsResult.value.proposals || [] : [];
    const plans = plansResult.status === 'fulfilled' ? plansResult.value.plans || [] : [];
    const fixtures = cases?.cases || [];
    const runs = runsResult.status === 'fulfilled' ? runsResult.value.runs || [] : [];
    const repairRecords = repairQueueResult.status === 'fulfilled' ? repairQueueResult.value.records || [] : [];
    const quarantineHistory = quarantineHistoryResult.status === 'fulfilled' ? quarantineHistoryResult.value.events || quarantineHistoryResult.value.history || [] : [];
    const rolloutProposals = rolloutsResult.status === 'fulfilled' ? rolloutsResult.value.proposals || [] : [];
    const liveRollouts = rolloutsResult.status === 'fulfilled' ? rolloutsResult.value.live || [] : [];
    state.parserRolloutProposals=rolloutProposals;state.parserLiveRollouts=liveRollouts;state.parserRolloutLoadError=rolloutsResult.status==='rejected'?rolloutsResult.reason.message:'';state.assuranceSyncBundle=syncResult.status==='fulfilled'?syncResult.value.bundle||null:null;
    state.repairRecords=repairRecords;state.quarantineHistory=quarantineHistory;state.assuranceParsers=parsers;state.assuranceFixtures=fixtures;state.assuranceSyncAssets=syncAssets;state.assuranceSyncClients=syncClients;state.assuranceSyncTelemetry=syncTelemetry;state.assuranceFleetNodes=fleetNodes;
    if(repairRecords.some(item=>(item.id||item.record_id)===state.quarantineDiagnosis?.id&&['excluded','evidence_deleted'].includes(quarantineRecordStatus(item)))){state.quarantineDiagnosis=null;state.gprsPreview=null;state.gprsSamples=[];state.qrwFields=[];state.qrwStep=1;sessionStorage.removeItem('nli.qrw.selected-record');sessionStorage.removeItem('nli.qrw.mapping-draft');}
    if(restoreQRW){
    if(!state.quarantineDiagnosis){const savedRecordId=sessionStorage.getItem('nli.qrw.selected-record')||'';const savedRecord=state.repairRecords.find(item=>(item.id||item.record_id)===savedRecordId);if(savedRecord&&!['excluded','evidence_deleted'].includes(quarantineRecordStatus(savedRecord))){state.quarantineDiagnosis={id:savedRecordId,record:savedRecord,restored:true};if(state.qrwStep===1){const candidate=state.assuranceParsers.filter(doc=>(doc.payload||{}).source_quarantine_record_id===savedRecordId).sort((a,b)=>Number(b.revision||0)-Number(a.revision||0))[0];state.qrwStep=savedRecord.status==='resolved'?9:candidate?.state==='published'?8:candidate?7:2;}}else if(savedRecordId)sessionStorage.removeItem('nli.qrw.selected-record');}
    }
    state.assurancePlans=plans;
    state.aiProposals=aiProposals;
    const packStates = managedPacks.reduce((out,item)=>(out[item.state||'unknown']=(out[item.state||'unknown']||0)+1,out),{});
    const alignedClients = syncTelemetry ? Number(syncTelemetry.exact_nodes||0) : fleetNodes.filter(item=>!(item.sync||{}).drifted && !(item.sync||{}).asset_drifted).length;
    const selectedFamily = state.lastAssurance?.request?.service || 'auto';
    const selectedParser = state.lastAssurance?.request?.parser_id || 'auto';
    const retentionError = casesResult.status === 'rejected' ? casesResult.reason.message : '';
    const current = state.lastAssurance?.result;
    const defaultPlan={objective:'Certify a parser revision and its bounded runtime contracts',environment:'synthetic-lab',dataset:'retained-fixture-id',target_version:'21.0.0',parser_versions:['imap@1'],required_features:['parser-assurance'],scenarios:['parser-regression','failure-recovery','replay','destination-integration','performance','security'],expected_results:['all local checks pass'],pass_criteria:['zero unexpected quarantine','all expected fields equal','bounded evidence contains no raw fixture'],timeouts:{run_seconds:60},resource_limits:{max_records:1000},cleanup:['purge transient sample'],evidence:['result JSON','parser revision hash','scenario checks'],approvals:['parser-engineer']};
    const plansPanel=panel('Versioned Assurance test plans','Build bounded, declarative plans for repeatable parser certification. Each save creates an immutable revision; execution and production promotion remain separately approval gated.',`<form id="assurance-plan-form" class="form-grid"><input type="hidden" name="edit_id"><label class="field">Plan ID<input name="id" required value="parser-certification"></label><label class="field">Name<input name="name" required value="Parser certification"></label><label class="field">Version<input name="version" required value="1.0.0"></label><label class="field wide">Declarative plan JSON<textarea name="plan" required>${esc(pretty(defaultPlan))}</textarea></label><div class="actions wide"><button class="primary" type="submit">Save test-plan revision</button><button class="secondary" type="button" data-action="assurance-plan-reset">New plan</button></div></form>${plansResult.status==='rejected'?`<div class="callout warn">Test plans unavailable: ${esc(plansResult.reason.message)}</div>`:table(['Plan','Version','Revision','State','Actions'],plans.map(doc=>{const p=doc.payload||{};return `<tr><td><b>${esc(p.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${esc(p.version||'—')}</td><td>${number(doc.revision)}</td><td>${badge(doc.state||'active')}</td><td><button class="secondary small" data-action="assurance-plan-edit" data-id="${esc(doc.id)}">Edit revision</button> <button class="secondary small" data-action="assurance-plan-download" data-id="${esc(doc.id)}">Download JSON</button> <button class="danger small" data-action="assurance-plan-delete" data-id="${esc(doc.id)}">Delete</button></td></tr>`;}),'No Assurance test plans have been created yet.')}`);
    const napiDiscovery=[{raw:'level=INFO user=alice',service:'auto',desired:{level:'INFO',user:'alice'}},{raw:'level=WARN user=bob',service:'auto',desired:{level:'WARN',user:'bob'}}],napiHeldOut=[{raw:'level=ERROR user=carol',service:'auto',desired:{level:'ERROR',user:'carol'}}];
    const napiPanel=panel('NAPI parser synthesis','Generate bounded declarative candidates from separate discovery and held-out samples. The controller performs RE2 safety and negative false-positive checks, stores no raw samples, and saves only a review-gated draft.',`<form id="napi-synthesis-form" class="form-grid" data-testid="napi-synthesis-form"><label class="field">Parser ID<input name="id" value="napi-guided-parser" required></label><label class="field">Name<input name="name" value="NAPI guided parser" required></label><label class="field">Version<input name="version" value="0.1.0" required></label><label class="field">Candidate limit<input name="candidate_limit" type="number" min="1" max="8" value="4" required></label><label class="field wide">Discovery samples JSON<textarea name="discovery" required>${esc(pretty(napiDiscovery))}</textarea></label><label class="field wide">Independent held-out samples JSON<textarea name="held_out" required>${esc(pretty(napiHeldOut))}</textarea></label><label class="field wide">Negative raw samples JSON<textarea name="negative">[]</textarea></label><div class="actions wide"><button class="primary" type="submit">Synthesize and validate draft</button></div></form>${state.lastNAPI?panel('Latest synthesis evidence','Review the selected candidate, safety report, validation metrics, provenance and limitations. Production activation still requires independent approval.',`<pre class="json-output" data-testid="napi-synthesis-result">${esc(pretty(state.lastNAPI))}</pre>`):'<div class="empty-state">No synthesis has run in this browser session.</div>'}`);
    const runScenarioOptions=[['parser-regression','Parser regression'],['failure-recovery','Failure recovery'],['replay','Deterministic replay'],['destination-integration','Destination JSON contract'],['performance','Bounded performance'],['security','Fixture privacy and integrity']];
    const recovery = state.quarantineDiagnosis;
    const recoverySource = recovery && state.lastAssurance?.request?.source === `quarantine:${recovery.id}`;
    const recoveryParser = recovery?.parserId ? parsers.find(doc => doc.id === recovery.parserId) : recovery ? parsers.find(doc => (doc.payload||{}).source_quarantine_record_id === recovery.id) : null;
    if (recovery && recoveryParser && !recovery.parserId) recovery.parserId = recoveryParser.id;
    const recoveryAsset = recovery?.parserId ? syncAssets.find(asset => asset.asset_class === 'parser-packs' && asset.asset_id === recovery.parserId) : null;
    const recoveryNode = recovery?.record?.node_id || '';
    const recoveryApplied = qrwExactAcknowledgement(recoveryAsset,recoveryNode);
    const recoveryStep = (label, done, detail) => `<li class="recovery-step ${done ? 'done' : ''}"><b>${done ? '✓' : '○'} ${esc(label)}</b><small>${esc(detail)}</small></li>`;
    const recoveryPanel = recovery ? panel('Recovery Workspace',`Evidence ${recovery.id} · SHA-256 ${String(recovery.sha256||'').slice(0,16)} · target ${recoveryNode||'unknown node'}. Status below comes from the selected evidence, saved Assurance parser state, and current Fleet Sync snapshot.`, `<ol class="recovery-steps">${recoveryStep('Inspect retained evidence',!!recovery.id,recovery.record?.raw_available?'Retained evidence is available under existing access controls.':'Evidence metadata only; exact input was not retained.')}${recoveryStep('Diagnose and test',!!(recoverySource&&state.lastAssurance?.result),recoverySource?`Last test ${state.lastAssurance?.result?.passed===true?'passed':state.lastAssurance?.result?.passed===false?'completed with findings':'not complete'}.`:'Run the selected evidence through Assurance.')}${recoveryStep('Save and submit parser',!!recoveryParser&&['in_review','approved','published'].includes(recoveryParser.state),recoveryParser?`${recoveryParser.id} · ${recoveryParser.state} · revision ${recoveryParser.revision||'—'}`:'No linked saved parser draft yet.')}${recoveryStep('Independent review',!!recoveryParser&&['approved','published'].includes(recoveryParser.state),recoveryParser?.state==='rejected'?'Rejected; revise the candidate before resubmitting.':recoveryParser?.state==='in_review'?'Awaiting a reviewer other than the author.':'Approval is recorded only after an independent reviewer acts.')}${recoveryStep('Publish and roll out',!!recoveryAsset,recoveryAsset?`Published revision ${recoveryAsset.desired?.revision||'—'} · ${String(recoveryAsset.desired?.sha256||'').slice(0,16)}; rollout approval follows existing policy.`:'No published runtime revision yet.')}${recoveryStep('Verify and replay',recoveryApplied,recoveryApplied?'Origin client acknowledged the exact desired bundle hash; replay can be requested.':`Replay stays blocked until ${recoveryNode||'the origin client'} acknowledges the published bundle.`)}</ol><div class="actions"><button class="secondary" type="button" data-action="recovery-open-helper">Edit or test parser</button>${recoveryParser?`<button class="secondary" type="button" data-action="navigate" data-page="parser-sync">Open Fleet Sync</button>`:''}<button class="secondary" type="button" data-action="navigate" data-page="unknowns">Return to quarantine evidence</button>${recoveryApplied?`<button class="primary" type="button" data-action="recovery-prepare-replay" data-id="${esc(recovery.id)}" data-node="${esc(recoveryNode)}">Prepare replay</button>`:''}</div><p class="hint">Publish, review, rollout, and replay each use their existing audited API and permission gate. This workspace does not mark any step complete optimistically.</p>`) : '';
    const runsPanel=panel('Assurance runs','Execute a bounded local scenario against an exact saved plan revision and retained encrypted fixture. Destination checks never contact production endpoints.',`<form id="assurance-run-form" class="form-grid" data-testid="assurance-run-form"><label class="field">Saved plan revision<select name="plan" required data-testid="assurance-run-plan">${plans.map(doc=>`<option value="${esc(doc.id)}@${number(doc.revision)}">${esc(doc.payload?.name||doc.id)} · ${esc(doc.id)} · revision ${number(doc.revision)} · ${esc(doc.sha256||'hash unavailable')}</option>`).join('')}</select></label><label class="field">Retained fixture<select name="case_id" required data-testid="assurance-run-fixture">${fixtures.map(doc=>`<option value="${esc(doc.id)}">${esc(doc.payload?.name||doc.name||doc.id)} · ${esc(doc.id)}</option>`).join('')}</select></label><label class="field">Scenario<select name="scenario" data-testid="assurance-run-scenario">${runScenarioOptions.map(([value,label])=>`<option value="${value}">${label}</option>`).join('')}</select><span class="hint">The selected scenario must be declared by the exact saved plan revision.</span></label><div class="actions wide"><button class="primary" type="submit" data-testid="assurance-run-start" ${plans.length&&fixtures.length?'':'disabled'}>Start bounded run</button></div>${!plans.length||!fixtures.length?'<p class="hint wide">Save a plan and retain a fixture before starting a run.</p>':''}</form>${runsResult.status==='rejected'?`<div class="callout warn">Runs unavailable: ${esc(runsResult.reason.message)}</div>`:table(['Run','Scenario / state','Plan / revision / hash','Evidence','Timestamps','Actions'],runs.map(doc=>{const r=doc.payload||doc,e=r.evidence||{};return `<tr data-testid="assurance-run-row"><td class="mono">${esc(r.id||doc.id)}</td><td>${esc(r.scenario||'—')}<br>${badge(r.state||doc.state)}</td><td>${esc(r.plan_id)} · r${number(r.plan_revision)}<br><small class="mono">${esc(r.plan_sha256||'')}</small></td><td>${esc(e.raw_sha256||'—')}<br>${e.passed===true?'passed':e.passed===false?'failed':'—'} · score ${esc(e.score??'—')}${e.checks?`<details><summary>Scenario checks</summary><pre class="json-output">${esc(pretty(e.checks))}</pre></details>`:''}</td><td>Queued ${esc(r.queued_at||'—')}<br>Started ${esc(r.started_at||'—')}<br>Finished ${esc(r.finished_at||'—')}</td><td><button class="secondary small" data-action="assurance-run-download" data-id="${esc(r.id||doc.id)}">Download JSON</button>${['queued','running'].includes(r.state||doc.state)?` <button class="danger small" data-action="assurance-run-cancel" data-id="${esc(r.id||doc.id)}">Cancel</button>`:''}</td></tr>`;}),'No persisted Assurance runs yet.')}`);
    renderPage(pageID, `<div class="callout">Workflow: analyze representative raw lines → validate a mixed-format corpus → create a parser draft → submit for independent approval → publish to build the parser pack and start fleet sync. The Assurance corpus is the cases you provide; Field Integrity &amp; Corpus Proof is the separate, fixed 728-record release corpus bundled with the controller.</div><div class="grid cols-4">${card('Live runtime parsers',number(runtimeParsers.length),'Selectable exact parser definitions')}${card('Managed parser packs',number(managedPacks.length),`${number(packStates.active||0)} active · ${number(packStates.approved||0)} approved`)}${card('Assurance drafts',number(parsers.length),`${number(packStates.in_review||0)} managed packs in review`)}${card('Fleet parser sync',`${number(alignedClients)} / ${number(fleetNodes.length)}`,`${number(syncAssets.length)} managed assets · aligned clients`)}</div><div class="callout">Automatic parser detection evaluates every eligible approved ParserPack and runtime strategy. Exact parser selection runs only the chosen live runtime parser. Custom families and new parser definitions remain test-only until review, approval, publish, and client acknowledgement.</div><div class="assurance-subnav" aria-label="Assurance workflows">${[['Analyse','Production parser assurance'],['Corpus reports','Multi-log corpus validation'],['Parser drafts','Assurance parser workflow'],['Review Queue','Revisioned parser proposals'],['History','Parser assurance history'],['Fixtures','Encrypted retained fixtures'],['Parser Creation Helper','Parser helper'],['NAPI Synthesis','NAPI parser synthesis'],['Test Plans','Versioned Assurance test plans'],['Runs','Assurance runs']].map(([label,title])=>`<button class="tab" type="button" data-title="${esc(title)}">${esc(label)}</button>`).join('')}</div>${panel('Parser Creation helper','Generate and edit a review-gated declarative draft from sample lines.',`<form id="parser-helper-form" class="form-grid"><input type="hidden" name="edit_id"><label class="field wide">Sample lines<textarea name="samples" required data-testid="parser-helper-samples">${esc(state.lastAssurance?.request?.raw || '')}</textarea></label><button type="button" class="secondary" data-action="parser-helper-generate">Generate parser draft</button><label class="field">Draft name<input name="name" required value="guided-parser-draft"></label><label class="field">Version<input name="version" required value="0.1.0"></label><div id="parser-helper-overview" data-parser-overview class="wide">Run Assurance or paste sample lines.</div><div id="parser-helper-details" data-parser-helper-details class="parser-helper-details wide" aria-live="polite"></div><label class="field wide">Complete generated parser JSON<textarea name="parser" required>{}</textarea></label><label class="field wide">Validation cases JSON<textarea name="cases" required>[]</textarea></label><div class="actions wide"><button type="button" class="secondary" data-action="parser-helper-validate">Validate draft</button><button type="submit" class="primary">Save review-gated draft</button></div><div id="parser-helper-status" data-parser-helper-status class="wide" role="status"></div></form>`)}${panel('Upload log samples','Upload log/text files or an archive for controller-side inspection. Archive extraction and size limits are enforced by the controller.',`<form id="assurance-upload-form" class="form-grid"><label class="field">Parser family${assuranceFamilySelect('service','auto')}<span class="hint">ⓘ Automatic evaluates all eligible approved families.</span></label><label class="field">Custom family<input name="custom_service" placeholder="used only when Custom family is selected" maxlength="64"></label><label class="field wide">Parser name${assuranceParserSelect(runtimeParsers,'auto')}<span class="hint">ⓘ Choose Automatic for the full pipeline or one exact live runtime parser.</span></label><label class="field">Sample file<input name="file" type="file" accept=".log,.txt,.out,.json,.jsonl,.csv,.zip,.tar,.gz,.tgz,.tar.gz" required title="Supported sample formats depend on controller archive limits. Never upload secrets or production data unless approved."><span class="hint">ⓘ .log, .txt, .json, .jsonl, .csv, .zip, .tar, .gz. Controller applies archive and size limits.</span></label><div class="actions wide"><button class="primary" type="submit">Analyze uploaded sample</button><small class="hint">The controller returns a report; original log content is transient unless retained as a fixture.</small></div></form>`)}<div class="split assurance-workspace"><section class="panel"><div class="section-head"><div><h3>Production parser assurance</h3><p>Runs the production parser pipeline against raw single or multiline logs. Raw payloads are transient unless explicitly retained.</p></div></div><form id="assurance-form" class="form-grid" data-testid="assurance-form"><label class="field wide"><span><input type="checkbox" name="retain_test_sample" value="true" ${state.sampleHistoryEnabled?'checked':''} ${can('assurance:save')?'':'disabled'}> Save an encrypted copy to Test Sample History</span></label><label class="field wide">Raw or multiline input<textarea name="raw" required data-testid="assurance-raw" title="Paste one or more log records. Multiline records are preserved exactly; use a representative sample without secrets.">${esc(state.lastAssurance?.request?.raw || '')}</textarea><span class="hint">ⓘ Paste representative records, including continuations. Secrets and tokens should be redacted.</span></label><label class="field">Parser family${assuranceFamilySelect('service',selectedFamily)}</label><label class="field">Custom family<input name="custom_service" placeholder="used only when Custom family is selected" maxlength="64"></label><label class="field wide">Parser name${assuranceParserSelect(runtimeParsers,selectedParser)}<span class="hint">ⓘ Automatic runs all eligible approved parsers; an exact choice runs only that live parser.</span></label><label class="field">Source<input name="source" value="${esc(state.lastAssurance?.request?.source || '')}" placeholder="optional source family" title="Optional source name or integration. It provides context and does not execute imported parser code."></label><label class="field wide">Exact desired JSON <span class="hint">Leave {} for parse-only validation.</span><textarea name="desired">${esc(pretty(state.lastAssurance?.request?.desired || {}))}</textarea><span class="hint">ⓘ Desired output is the exact field/value contract used for comparison. Extra or missing fields are reported.</span></label><label class="field wide">Field mapping JSON<textarea name="mapping" title="Optional JSON object mapping source keys to canonical output field names.">${esc(pretty(state.lastAssurance?.request?.field_mapping || {}))}</textarea><span class="hint">ⓘ Example: {"src_ip":"source_ip","ts":"timestamp"}. Leave {} to preserve inferred field names.</span></label><div class="actions wide"><button class="primary" type="submit">Run assurance</button><button class="secondary" type="button" data-action="clear-assurance" ${current ? '' : 'disabled'}>Clear manually</button><button class="secondary" type="button" data-action="download-assurance" ${current ? '' : 'disabled'}>Download result JSON</button></div></form></section><section id="assurance-result">${current ? renderAssurance(current) : empty('Run a case to inspect complete output, field/type/count differences, source spans, and every parser stage.')}</section></div>${panel('Multi-log corpus validation','Paste a JSON array of cases to validate mixed log formats in one run. Each case uses the production parser.',`<form id="assurance-corpus-form" class="form-grid"><label class="field">Default parser family${assuranceFamilySelect('service','auto')}</label><label class="field">Custom family<input name="custom_service" placeholder="used only when Custom family is selected" maxlength="64"></label><label class="field wide">Default parser name${assuranceParserSelect(runtimeParsers,'auto')}<span class="hint">ⓘ Defaults apply to cases that do not declare service or parser_id. Individual JSON cases may override either value.</span></label><label class="field wide">Cases JSON<textarea name="cases" required>[{\"raw\":\"service=example user=operator\",\"service\":\"auto\",\"desired\":{}}]</textarea><span class="hint">ⓘ Each item accepts raw, service, source, desired and field_mapping. Keep samples small and redact credentials.</span></label><button class="primary" type="submit">Run corpus</button></form><section id="assurance-corpus-result" aria-live="polite">${empty('Corpus results will appear here.')}</section>`)}${panel('Assurance parser workflow','Create a reviewable declarative parser proposal from the failing sample and validate it against cases.',`<form id="assurance-parser-form" class="form-grid"><label class="field wide">Start from parser${assuranceParserSelect(runtimeParsers,'auto','base_parser_id')}<span class="hint">ⓘ Select an existing live parser to copy its declarative definition, or leave Automatic to create your own.</span></label><label class="field">Parser name<input name="name" required maxlength="120" title="Choose a stable, descriptive family or use case name."></label><label class="field">Version<input name="version" value="0.1.0" required title="Use semantic versioning for changes; each save creates a reviewable revision."></label><label class="field wide">Declarative parser JSON<textarea name="parser" required title="Only declarative, bounded parser definitions are accepted. Executable code is not run.">{"pattern_type":"regex","pattern":"level=(?P&lt;level&gt;\\\\w+) user=(?P&lt;user&gt;\\\\w+)","fields":[{"name":"level","source":"group:level","type":"string"},{"name":"user","source":"group:user","type":"string"}]}</textarea><span class="hint">ⓘ Start with a parser pack from Parser Catalog, or inspect a built-in parser definition. Drafts require corpus validation and reviewer approval.</span></label><label class="field wide">Validation cases JSON<textarea name="cases" required>[{"raw":"level=INFO user=alice","service":"auto","desired":{"level":"INFO","user":"alice"}}]</textarea></label><button class="primary" type="submit">Create and validate draft</button></form>`)}${panel('Revisioned parser proposals','Review lifecycle: validate, submit, approve by a second operator, then publish.',table(['Parser','Revision','State','Validation','Actions'],parsers.map(doc=>{const p=doc.payload||doc;return `<tr><td><b>${esc(p.name||doc.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${doc.revision??p.revision??'—'}</td><td>${badge(doc.state||p.state)}${approvalDetails(doc)}</td><td class="review-validation"><details><summary>${esc((p.validation||doc.validation)?.status||((p.validation||doc.validation)?.passed===undefined?'Details':`${number((p.validation||doc.validation).passed)} passed · ${number((p.validation||doc.validation).failed)} failed`))}</summary><pre>${esc(pretty(p.validation||doc.validation||{}))}</pre></details></td><td class="review-actions">${(['validate',...(['draft','rejected'].includes(doc.state)?['submit']:[]),...(doc.state==='in_review'?['approve','reject']:[]),...(doc.state==='approved'?['publish']:[])].map(action=>`<button class="secondary small" data-action="assurance-parser-action" data-id="${esc(doc.id)}" data-value="${action}">${action}</button>`).join(' ') + (['draft','rejected'].includes(doc.state)?` <button class="secondary small" data-action="assurance-parser-edit" data-id="${esc(doc.id)}">Edit</button> <button class="danger small" data-action="assurance-parser-delete" data-id="${esc(doc.id)}">Delete</button>`:''))}</td></tr>`})))}${panel('Parser assurance history','Recent validation and review events from the controller.',table(['Time','Action','Parser','Actor','Outcome'],history.map(item=>`<tr><td>${time(item.timestamp||item.created_at)}</td><td>${esc(item.action||item.event)}</td><td class="mono">${esc(item.parser_id||item.resource||'—')}</td><td>${esc(item.actor||'—')}</td><td>${badge(item.outcome||item.state||'recorded')}</td></tr>`)))}${panel('Encrypted retained fixtures','Saved cases stay encrypted in Supabase until an authorized operator purges them manually.',retentionError ? `<div class="callout warn">${esc(retentionError)}</div>` : renderCases(cases?.cases || []))}${napiPanel}${runsPanel}`);
    if (recoveryPanel) page.insertAdjacentHTML('afterbegin', recoveryPanel);
    const plansSlot=page.querySelector('#assurance-panel-plans');
    if(plansSlot)plansSlot.innerHTML=`<h2 class="assurance-panel-title">Test Plans</h2>${plansPanel}`;
    page.insertAdjacentHTML('beforeend', renderAIProposalQueue(aiProposals, current, aiProposalsResult.status === 'rejected' ? aiProposalsResult.reason.message : ''));
    const uploadForm=page.querySelector('#assurance-upload-form');
    if(uploadForm){
      uploadForm.querySelector('.actions')?.insertAdjacentHTML('beforebegin',`<label class="field wide retention-toggle"><span><input type="checkbox" name="retain" value="true" ${can('assurance:save')?'':'disabled'}> Retain encrypted fixture in Supabase</span><small>Optional and off by default. The controller encrypts the sample before protected Supabase persistence. The browser never uploads directly to a public bucket. Retention requires controller support and may be subject to configured limits.${can('assurance:save')?'':' Your role does not have assurance:save permission.'}</small></label>`);
      uploadForm.insertAdjacentHTML('afterend',renderUploadStatus());
      addInputGuidance(uploadForm);
    }
    const baseParserSelect=page.querySelector('#assurance-parser-form [name=base_parser_id]');
    baseParserSelect?.addEventListener('change',event=>{const parser=runtimeParsers.find(item=>item.id===event.currentTarget.value);if(!parser)return;const form=event.currentTarget.form;form.elements.name.value=`${parser.id}-custom`;form.elements.version.value=`${Number(parser.version||1)+1}.0.0`;form.elements.parser.value=pretty(parser);});
  }
  function renderCases(items) { return table(['Name','Version','Parser','Result','Retention','Actions'], items.map(item => { const value=item.payload||item,id=item.id||value.id;return `<tr><td>${esc(value.name||id)}</td><td>${esc(item.revision??value.revision??'—')}</td><td class="mono">${esc(value.parser_id||'automatic')}</td><td>${badge(value.passed===true?'passed':value.passed===false?'failed':'retained')}</td><td>${value.manual_retention ? 'Manual purge' : value.expires_at?time(value.expires_at):'Controller policy'}</td><td><button class="secondary small" data-action="download-case" data-id="${esc(id)}">Download .log</button> <button class="secondary small" data-action="promote-case" data-id="${esc(id)}">Promote</button> <button class="danger small" data-action="purge-case" data-id="${esc(id)}">Purge</button></td></tr>`; })); }
  async function requestAIProposal(request) {
    if(!request?.raw?.trim())throw new Error('Run a raw sample through Assurance before requesting AI remediation.');
    return api('/assurance/ai-proposal',{method:'POST',body:request});
  }
  function renderAIProposalQueue(items, current, error='') {
    const currentHash=current?.raw_sha256||'';
    const rows=(items||[]).map(doc=>{const p=doc.payload||{},diagnosis=p.diagnosis||{},sameSample=currentHash&&p.source_hash===currentHash;return `<tr><td><b>${esc(p.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${badge(doc.state||p.review_state||'draft')}${approvalDetails(doc)}<br><small>Created by ${esc(doc.created_by||'—')}</small></td><td><b>${esc(diagnosis.summary||'Deterministic parsing did not meet the requested contract.')}</b><br><small>Score ${esc(diagnosis.score??'—')} · source hash ${esc(String(p.source_hash||'').slice(0,16))}</small><details><summary>Inspect diagnosis</summary><pre class="json-output">${esc(pretty(diagnosis))}</pre></details></td><td>${esc(p.suggested_method||'Validate this declarative parser against the failing case and regression corpus.')}<details><summary>Inspect proposed parser</summary><pre class="json-output">${esc(pretty(p.proposal||{}))}</pre></details></td><td>${doc.state==='draft'?`<button class="secondary small" data-action="ai-proposal-approve" data-id="${esc(doc.id)}">Approve proposal</button><button class="secondary small" data-action="ai-proposal-reject" data-id="${esc(doc.id)}">Reject proposal</button>`:''}${doc.state==='approved'&&sameSample?`<button class="primary small" data-action="ai-proposal-create-draft" data-id="${esc(doc.id)}">Create validated parser draft</button>`:doc.state==='approved'?'<small>Rerun the matching failing sample to create its validated parser draft.</small>':''}<button class="danger small" data-action="ai-proposal-remove" data-id="${esc(doc.id)}">Remove proposal</button></td></tr>`;});
    return panel('AI remediation review','AI explains a deterministic parser failure and proposes a bounded declarative method. It cannot modify, publish, or roll out a parser. An administrator may approve, or an authorized reviewer can review the proposal. The matching failing sample must then validate the parser draft before normal review and publication. Reject or remove a proposal without changing published parsers. Revoke a published pack through Catalog Offboard / Rollback and verify rollout approval and client acknowledgement.',`${error?`<div class="callout warn">${esc(error)}</div>`:''}${table(['Proposal','Review state','Issue','Suggested method','Actions'],rows,'No AI remediation proposals are waiting. Failed Assurance runs can request one after a provider is configured.')}`);
  }
  function renderUploadStatus() {
    if(state.uploadError)return `<div id="assurance-upload-status" class="callout warn"><b>Upload could not be completed</b><br>${esc(state.uploadError)}</div>`;
    if(!state.uploadReport)return '<div id="assurance-upload-status"></div>';
    const response=state.uploadReport.response||{},meta=response.retention||response.fixture||response.file||response,report=response.report||{};
    const fixtureId=meta.fixture_id||meta.case_id||response.fixture_id||response.case_id||meta.id||response.fixture?.id||'';
    const retained=response.retained===true||meta.retained===true||Boolean(fixtureId&&(meta.encrypted===true||response.encrypted===true||meta.retained_at||response.retained_at||meta.storage_status));
    const name=meta.filename||response.filename||state.uploadReport.filename;
    const type=meta.file_type||meta.content_type||meta.mime_type||meta.type||state.uploadReport.type||'unknown';
    const size=meta.size_bytes??meta.size??response.size_bytes??response.file_size_bytes??state.uploadReport.size;
    const hash=meta.sha256||meta.file_sha256||meta.file_hash||response.sha256||response.raw_sha256||response.file_sha256||'Not supplied by controller';
    const parserRevision=meta.parser_revision||meta.parser_revision_id||meta.parser?.version||response.parser_revision||response.parser_revision_id||response.parser?.version||report.parser_revision||'—';
    const storage=retained?(meta.storage_status||meta.storage?.status||response.storage_status||response.storage?.status||'Encrypted in protected Supabase storage'):(meta.storage_status||meta.storage?.status||response.storage_status||response.storage?.status||'Not retained; analysis was transient');
    return `<div id="assurance-upload-status" class="upload-report"><div class="upload-report-head"><div><p class="eyebrow">Controller analysis report</p><h4>${esc(name||'Uploaded sample')}</h4></div>${badge(retained?'encrypted and retained':'transient analysis')}</div><div class="grid cols-3">${card('File type',type,'Detected by controller')}${card('Size',size==null?'—':`${number(size)} bytes`,'Uploaded sample size')}${card('Parser revision',parserRevision,'Selected for analysis')}</div><div class="upload-report-meta"><span><b>SHA-256</b><code>${esc(hash)}</code></span><span><b>Storage</b><span>${esc(storage)}</span></span>${fixtureId?`<span><b>Fixture ID</b><code>${esc(fixtureId)}</code></span>`:''}</div>${retained?`<div class="callout">The controller encrypted the fixture before protected Supabase persistence. No direct or public bucket upload was used.</div><button class="secondary" data-action="download-upload-fixture" data-id="${esc(fixtureId)}" data-filename="${esc(name||'nli-retained-fixture.log')}">Download retained sample through controller</button>`:'<div class="callout">This sample was analyzed in memory and was not retained. Enable retention on a future upload to save an encrypted fixture.</div>'}</div>`;
  }
  function evidenceStart(item) {
    if (!item || !(item.source_backed ?? item.source ?? item.inferred)) return Infinity;
    return item.spans?.length ? Math.min(...item.spans.map(span => span.start ?? 0)) : item.start ?? 0;
  }
  function evidenceOrdered(fields, evidence) {
    if (!fields || Array.isArray(fields) || typeof fields !== 'object') return fields;
    return Object.fromEntries([...new Set([...evidence.map(item => item.field), ...Object.keys(fields)])]
      .filter(name => Object.hasOwn(fields, name)).map(name => [name, fields[name]]));
  }

  function renderAssurance(result) {
    const diffs = result.diff || [], evidence = (result.evidence_spans || []).slice().sort((left,right) => evidenceStart(left)-evidenceStart(right)), trace = result.trace || [], counts = result.counts || {}, errors = result.errors || [];
    const attempts = trace.filter(item => item.stage === 'parser_attempt');
    const traceRows = trace.map((item, index) => {
      const reason = item.failure_reason || item.error_code;
      const detail = reason === 'incomplete_structured_header' ? 'A header component is missing or assigned to the wrong field.' : '';
      return `<tr><td>${index + 1}/${trace.length}</td><td><b>${esc(item.stage_name || item.stage_id || item.stage)}</b><br><small>${esc(item.stage_type || '')}</small></td><td class="mono">${esc(item.parser || item.strategy || '—')}</td><td>${badge(item.outcome || (item.success ? 'success' : 'failed'))}${reason ? `<br><small>${esc(reason)}</small>` : ''}${detail ? `<br><small>${esc(detail)}</small>` : ''}</td><td>${item.fields ?? '—'}</td><td>${item.latency_ms == null ? '—' : `${item.latency_ms} ms`}</td></tr>`;
    });
    const evidenceRows = evidence.map(item => {
      const isSource = item.source_backed ?? item.source ?? false;
      const spans = item.spans?.length ? item.spans : [{start:item.start ?? 0,end:item.end}];
      const ranges = spans.filter(span => span.end != null).map(span => `${span.start ?? 0}–${span.end}`).join(', ');
      const origin = item.derived_rule ? 'derived' : item.inferred ? 'inferred' : isSource ? 'source' : 'unverified';
      return `<tr><td>${esc(item.field)}</td><td><code>${esc(pretty(result.actual?.[item.field]))}</code></td><td>${badge(origin)}</td><td>${esc(isSource || item.inferred ? ranges || '—' : '—')}</td><td>${esc(item.derived_rule || (item.inferred ? 'Value lookup' : isSource ? 'Source span' : 'Unavailable'))}</td></tr>`;
    });
    const comparison = result.comparison_enabled ? `<div><h3>Desired output</h3><pre class="json-output">${esc(pretty(result.desired))}</pre></div>` : '<div class="callout">Parse-only validation: add desired JSON when exact structural comparison is required.</div>';
    const coverage = result.coverage || {};
    const aiRemediation = result.passed ? '' : `<div class="callout warn"><b>Parser remediation is required.</b> The controller can send a masked version of this failed sample to the configured AI provider. The result is saved only as a review draft and cannot change a live parser.<div class="actions"><button class="primary" type="button" data-action="request-ai-proposal">Ask Nexora AI to diagnose and propose a fix</button><button type="button" class="secondary" data-action="workspace-ai-auto" data-tab="Assurance diagnosis">Auto Fix with Nexora AI</button></div></div>`;
    const sourceEvidence = result.raw ? panel('Retained source evidence', 'Exact retained input is permission checked and returned as escaped text; line-ending metadata is supplied by the controller.', `<p>SHA-256: <code>${esc(state.quarantineDiagnosis?.sha256 || result.raw_sha256 || 'not supplied')}</code></p>${result.line_endings?`<p>Line endings: <code>${esc(pretty(result.line_endings))}</code></p>`:''}<pre class="json-output">${esc(result.raw)}</pre>`) : '';
    return `${panel('Assurance verdict','Production parser execution with field coverage, confidence and candidate generation evidence.',`<div class="grid cols-4">${card('Result', result.passed ? 'PASS' : 'FAIL', result.route)}${card('Parser', result.parser?.name || 'unknown', `v${result.parser?.version || '?'} · ${result.parser?.strategy || '—'}`)}${card('Attempts', counts.attempts || attempts.length, `${counts.fallbacks || 0} fallbacks`)}${card('Coverage', Number(coverage.expected)>0?`${coverage.percent ?? 0}%`:'Not measured', Number(coverage.expected)>0?`${coverage.found ?? 0} / ${coverage.expected} expected fields`:'Add expected fields to measure coverage')}</div>${result.score != null ? `<div class="assurance-score"><span>Assurance score</span><b>${esc(result.score)} / 100</b></div>` : ''}${aiRemediation}${coverage.missing?.length ? `<div class="callout warn">Missing fields: ${esc(coverage.missing.join(', '))}</div>` : ''}${result.candidate_parser ? panel('AI-assisted parser candidate','Candidate is review-only and requires validation and independent approval.',`<pre class="json-output">${esc(pretty(result.candidate_parser))}</pre><button class="secondary" type="button" data-action="create-candidate-parser">Create validated draft</button>`) : ''}${errors.length ? `<div class="callout warn">${esc(errors.join(' · '))}</div>` : ''}<div class="split output-split"><div><h3>Complete actual output</h3><pre class="json-output">${esc(pretty(evidenceOrdered(result.actual, evidence)))}</pre></div>${comparison}</div>`)}${sourceEvidence}${panel('Parser traversal dashboard','Actual ordered attempts and field-by-field stage execution returned by the production parser.',`<div class="trace-summary"><progress max="${Math.max(trace.length, 1)}" value="${trace.length}"></progress><span>${trace.length} steps complete · ${attempts.length} parser attempts</span></div>${table(['Progress','Stage','Parser','Outcome','Fields','Latency'],traceRows)}`)}${panel('Field mapping and source evidence','Byte ranges are zero-based and end-exclusive. Inferred ranges come from a value lookup.',table(['Field','Parsed value','Origin','Input bytes','Rule'],evidenceRows))}${result.comparison_enabled ? panel('Field, type and count diff',`${counts.matches || 0} matching · ${counts.differences || 0} differences`,diffs.length ? diffs.map(item => `<div class="diff-row"><b>${esc(item.field)}</b>${badge(item.status)}<code>${esc(pretty(item.actual))}</code><code>${esc(pretty(item.desired))}</code></div>`).join('') : '<div class="callout">Exact structural match.</div>') : ''}<section class="panel"><form id="assurance-save-form" class="form-grid"><label class="field">Fixture name<input name="name" required></label><label class="field">Existing case ID for new version<input name="id"></label><label class="field wide">Description<input name="description"></label><button class="primary" type="button" data-action="save-assurance">Encrypt and retain in Supabase</button></form></section>`;
  }

  function agentMetricsLabel(sync) {
    return ({metrics_reporting:'Metrics reporting',metrics_unavailable:'Metrics unavailable'})[sync.agent_status] || 'Not reported';
  }
  function renderManagedFileForm(items) {
    const clients=items.filter(({node})=>{const [major,minor]=String(node?.version||'').replace(/^v/,'').split('.').map(Number);return major>22||major===22&&minor>=1;});
    const choices=clients.map(({node})=>`<option value="${esc(node.id)}">${esc(node.hostname||node.id)} · ${esc(node.status||'unknown')}</option>`).join('');
    const filenames=[...new Set(clients.flatMap(({sync})=>sync?.input_inventory?.managed_files||[]))].sort().slice(0,256);
    return panel('Copy or delete managed log files','Approved operations apply to files directly inside /var/log/nli/input on the selected clients. Remove the path from collection and verify the rollout before deleting its file.',`<form id="managed-input-file-form" class="form-grid">
      <label class="field">File operation<select name="action"><option value="copy">Copy file</option><option value="delete">Delete file</option></select></label>
      <label class="field">Scope<select name="scope"><option value="selected">Selected clients</option><option value="all">All compatible enrolled clients</option></select></label>
      <label class="field">Source filename<input name="source" list="managed-input-file-names" required pattern="[A-Za-z0-9][A-Za-z0-9._\\-]{0,127}" placeholder="imap.log" maxlength="128"><span class="hint">Enter a filename inside the managed directory.</span></label>
      <datalist id="managed-input-file-names">${filenames.map(name=>`<option value="${esc(name)}"></option>`).join('')}</datalist>
      <label class="field" data-file-destination>Destination filename<input name="destination" required pattern="[A-Za-z0-9][A-Za-z0-9._\\-]{0,127}" placeholder="imap-archive.log" maxlength="128"><span class="hint">Copy keeps file ownership and limits permissions to the original owner and group. Existing destinations are rejected. Limit 64 MiB.</span></label>
      <label class="field wide">Selected clients<select name="targets" multiple size="5">${choices}</select><span class="hint">Requires Ver22.1 or newer. Offline clients receive approved operations when they reconnect.</span></label>
      <label class="field wide"><input type="checkbox" name="confirmed" required> I reviewed the selected clients and filename.</label>
      <button class="primary" type="submit" ${clients.length?'':'disabled'}>Request managed file operation approval</button>
    </form>`);
  }
  function renderReportedInputs(sync) {
    const inventory=sync.input_inventory;
    if(!inventory)return '<small>Input inventory not reported</small>';
    const files=inventory.managed_files;
    const disk=Array.isArray(files)?`<details><summary>${number(files.length)} managed file(s) on disk</summary>${files.length?`<ul>${files.map(name=>`<li><code>${esc(name)}</code></li>`).join('')}</ul>`:'<small>No managed files reported</small>'}</details>`:'';
    if(!inventory.inputs?.length)return '<small>No inputs configured</small>'+disk;
    return `<details><summary>${number(inventory.inputs.length)} reported input(s)${inventory.truncated?' · partial inventory':''}</summary><small class="mono">${esc(inventory.config_path)}</small><ul>${inventory.inputs.map(input=>`<li><code>${esc(input.path)}</code> <small>${esc(input.service||'auto')} · ${input.disabled?'disabled':'enabled'} · ${esc(input.state)}</small></li>`).join('')}</ul></details>${disk}`;
  }
  const FLEET_FOCUS = [['Overview',0],['Heartbeat',1],['Input sources',3],['Parser activity',4],['Resources',5],['Desired state',6],['Issues',7]];
  function applyFleetFocus(column=0) {
    state.fleetFocus=column;
    for(const button of page.querySelectorAll('[data-action=fleet-focus]'))button.setAttribute('aria-pressed',String(Number(button.dataset.value)===column));
    for(const row of page.querySelectorAll('#fleet-focus-table tr')) {
      if(row.cells.length<2)continue;
      [...row.cells].forEach((cell,index)=>{cell.hidden=column!==0&&index!==0&&index!==column;});
    }
    if(el('fleet-desired-form'))el('fleet-desired-form').closest('.panel').hidden=column!==0&&column!==6;
  }
  async function loadFleet() {
    const data = await api('/fleet/sync'); const items = data.nodes || [];
    const onboarding = items.length ? '' : '<div class="callout warn"><b>No client has checked in yet.</b> On the controller run <code>sudo nli-controller-setup --rotate-enrollment</code>, copy the generated bootstrap file to the client, then run <code>sudo nli-client-setup /path/to/nli-client-bootstrap.env</code>.</div>';
    const nodes = items.map(item => item.node || {}), metrics = nodes.map(node => node.metrics || {});
    const online = nodes.filter(node => /online|active|healthy/i.test(node.status || '')).length;
    const drifted = items.filter(item => item.sync?.drifted).length;
    const received = metrics.reduce((total, item) => total + Number(item.logs_received || 0), 0);
    const parsed = metrics.reduce((total, item) => total + Number(item.logs_parsed || 0), 0);
    const quarantined = metrics.reduce((total, item) => total + Number(item.logs_quarantined || 0), 0);
    const summary = `<div class="grid cols-4 fleet-summary">${card('Enrolled clients',number(items.length),`${number(online)} online now`)}${card('Records received',number(received),'Live client counters')}${card('Parsed / quarantined',`${number(parsed)} / ${number(quarantined)}`,received?`${(parsed/received*100).toFixed(2)}% parse rate`:'Awaiting input')}${card('Configuration drift',number(drifted),drifted?'Requires operator review':'All reporting clients aligned')}</div><div class="fleet-focus" aria-label="Client monitoring dimensions">${FLEET_FOCUS.map(([label,column])=>`<button type="button" class="secondary small" data-action="fleet-focus" data-value="${column}" aria-pressed="${(state.fleetFocus||0)===column}">${label}</button>`).join('')}</div>`;
    renderPage('fleet', `${onboarding}${summary}${panel('Fleet desired vs active','Identity, heartbeat, agent health, parse counters and immutable desired/active state from enrolled clients.',`<div id="fleet-focus-table">${table(['Node / IP','Health / heartbeat','Platform','Log sources','In / parsed / quarantine','CPU / RAM / load','Desired vs active','Drift'],items.map(({node={},sync={}})=>{const m=node.metrics||{},tags=node.tags||{},paths=node.log_paths||node.input_paths||tags.log_paths||tags.log_path||tags.inputs;return `<tr><td><b>${esc(node.hostname||node.id)}</b><br><small class="mono">${esc(node.id)} · ${esc(node.ip||'IP unavailable')}</small></td><td>${badge(node.status||'unknown')}<br><small>${time(node.last_seen)}</small></td><td>${esc([node.os,node.arch].filter(Boolean).join(' / ')||'—')}<br><small>v${esc(node.version||'—')}</small></td><td>${sync.input_inventory?renderReportedInputs(sync):esc(Array.isArray(paths)?paths.join(', '):(paths||'Not reported'))}<br><small class="mono">${esc(node.metrics_url||'Metrics endpoint not reported')}</small></td><td>${number(m.logs_received)} / ${number(m.logs_parsed)} / ${number(m.logs_quarantined)}<br><small>parse ${Number(m.parse_rate_pct||0).toFixed(2)}%</small></td><td>${m.cpu_percent==null?'—':`${Number(m.cpu_percent).toFixed(1)}%`} / ${m.memory_percent==null?'—':`${Number(m.memory_percent).toFixed(1)}%`}<br><small>load ${esc(m.load_1m??'—')} / ${esc(m.load_5m??'—')} / ${esc(m.load_15m??'—')}</small></td><td><small>Pack</small><br><code>${esc((sync.desired_pack_hash||'—').slice(0,12))} → ${esc((sync.active_pack_hash||'—').slice(0,12))}</code><br><small>Config</small><br><code>${esc((sync.desired_config_hash||'—').slice(0,12))} → ${esc((sync.active_config_hash||node.config_hash||'—').slice(0,12))}</code></td><td>${badge(sync.drifted?'drifted':'aligned')}<br><small>${esc((node.issues||[]).join(' · ')||'No reported issues')}</small></td></tr>`;}))}</div>`)}${panel('Set desired immutable hashes','Ops-only mutation; clients report active hashes independently when they reconcile.',`<form id="fleet-desired-form" class="form-grid"><label class="field">Node ID<input name="node" required title="Copy the exact enrolled node ID shown in the inventory."></label><label class="field">Parser pack SHA-256<input name="pack_hash" pattern="[a-fA-F0-9]{64}" title="Expected 64-character SHA-256 hash; leave blank to preserve current desired hash."></label><label class="field">Config SHA-256<input name="config_hash" pattern="[a-fA-F0-9]{64}"></label><label class="field">Release SHA-256<input name="release_hash" pattern="[a-fA-F0-9]{64}"></label><button class="primary" type="submit">Set desired state</button></form>`)}`);
    if(state.page==='fleet')applyFleetFocus(state.fleetFocus||0);
  }

  async function loadParserSync() {
    let data;
    try { data = await api('/parser-sync',{allowStale:true}); }
    catch (error) {
      renderPage('parser-sync', `${panel('Parser Fleet Sync unavailable','Controller sync inventory could not be loaded.',`<div class="error-state"><div><b>Live parser state unavailable</b><p>${esc(error.message)}</p><button class="secondary" data-action="reload">Retry snapshot</button></div></div>`)}${panel('How automatic parser rollout works','Web-created assets remain drafts until they pass review. Approved and published revisions become desired state and are eligible for automatic rollout; retry creates a proposal and does not bypass rollout approval.',`<div class="callout">The page reads and refreshes controller state only. It does not activate drafts or change client state.</div>`)}`);
      return;
    }
    const stale = data.snapshot_stale===true;
    const staleWarning = staleSnapshotWarning(data);
    const assets = data.assets || [], clients = data.clients || [], telemetry = data.telemetry || {};
    const rows = assets.map(asset => {
      const desired = asset.desired || {}, controller = asset.controller || {}, rollout = asset.rollout || {}, policy = asset.auto_rollout || {};
      const total = Number(rollout.total_nodes || 0), done = Number(rollout.done_nodes || 0), failed = Number(rollout.failed_nodes || 0);
      const progress = total ? Math.max(0, Math.min(100, Math.round(done / total * 100))) : 0;
      const rolloutErrors = Array.isArray(rollout.errors) ? rollout.errors : rollout.errors ? [rollout.errors] : [];
      return `<tr><td><b>${esc(asset.name || asset.asset_id || 'Unnamed asset')}</b><br><small>${esc(asset.asset_class || 'Unclassified')}</small><br><small class="mono">${esc(asset.asset_id || '—')}</small></td><td>${badge(desired.state || 'unknown')}<br><b>Revision ${esc(desired.revision || '—')}</b><br><code title="${esc(desired.sha256 || '')}">${esc((desired.sha256 || '—').slice(0, 20))}</code></td><td>${badge(controller.status || 'unknown')}<br><b>Revision ${esc(controller.revision || '—')}</b><br><code title="${esc(controller.sha256 || '')}">${esc((controller.sha256 || '—').slice(0, 20))}</code></td><td>${badge(rollout.status || 'not scheduled')}<br><small>${done} / ${total} complete · ${failed} failed</small>${total ? `<progress class="sync-progress" max="100" value="${progress}" aria-label="Rollout ${progress}% complete"></progress>` : ''}${rolloutErrors.length ? `<ul class="sync-errors">${rolloutErrors.map(item => `<li>${esc(typeof item === 'string' ? item : item.message || pretty(item))}</li>`).join('')}</ul>` : ''}</td><td>${badge(policy.enabled ? policy.status || 'enabled' : 'disabled')}<br><small>${policy.enabled ? 'Published revisions roll out automatically' : 'Automatic rollout is off'}</small><br><small>${esc(policy.description || '')}</small></td><td class="sync-actions"><button class="secondary small" data-action="parser-sync-verify" ${stale?'disabled':''} data-class="${esc(asset.asset_class)}" data-id="${esc(asset.asset_id)}">Verify</button><button class="secondary small" data-action="parser-sync-retry" ${stale?'disabled':''} data-class="${esc(asset.asset_class)}" data-id="${esc(asset.asset_id)}">Propose retry</button></td></tr>`;
    });
    const clientRows = clients.map(client => `<tr><td><b>${esc(client.hostname || client.node_id || 'Unknown client')}</b><br><small class="mono">${esc(client.node_id || '—')}</small></td><td>${badge(client.status || 'unknown')}<br><small>Heartbeat ${time(client.last_seen)}</small></td><td>${esc(client.applied_revision || '—')}<br><code title="${esc(client.applied_hash || '')}">${esc((client.applied_hash || '—').slice(0, 16))}</code></td><td>${badge(client.drifted ? 'drifted' : client.status === 'offline' ? 'offline' : 'aligned')}<br><small>Target ${esc(client.desired_revision || '—')} · ${esc((client.desired_hash || '—').slice(0, 12))}</small>${client.error ? `<br><small class="sync-error">${esc(client.error)}</small>` : ''}</td></tr>`);
    const fleetTelemetry = `<div class="grid cols-4 fleet-summary">${card('Fleet nodes',number(telemetry.total_clients ?? telemetry.total_nodes ?? clients.length),'Bounded parser-sync inventory')}${card('Online / offline',`${number(telemetry.online_clients ?? telemetry.online_nodes ?? clients.filter(item=>item.status!=='offline').length)} / ${number(telemetry.offline_clients ?? telemetry.offline_nodes ?? clients.filter(item=>item.status==='offline').length)}`,'Latest authenticated check-ins')}${card('Exact bundle',number(telemetry.exact_acknowledged_clients ?? telemetry.exact_nodes ?? clients.filter(item=>!item.drifted).length),`${Number(telemetry.acknowledgement_pct ?? telemetry.acknowledgement_percent ?? 0).toFixed(2)}% acknowledged`)}${card('Pending / drifted / failed',`${number(telemetry.pending_clients ?? telemetry.pending_nodes ?? 0)} / ${number(telemetry.drifted_clients ?? telemetry.drifted_nodes ?? 0)} / ${number(telemetry.failed_clients ?? telemetry.failed_nodes ?? 0)}`,'Actionable fleet state')}</div>`;
    const automatic = data.policy ? `<div class="callout">${esc(data.policy.description || `Automatic rollout policy: ${data.policy.enabled ? 'enabled' : 'disabled'}.`)}</div>` : '';
    renderPage('parser-sync', `${panel('Parser Fleet Sync','One authoritative desired bundle, one fleet acknowledgement snapshot, and per-asset rollout evidence across every enrolled client.',`<div class="callout">Assets created in the WebView remain drafts until validated and reviewed. Approved and published revisions become part of the signed desired bundle. The client table is rendered once from the canonical fleet snapshot, so every tab evaluates the same acknowledgement state.</div>${staleWarning}${automatic}${fleetTelemetry}<div class="sync-toolbar"><span>${number(data.count ?? assets.length)} managed assets · desired bundle <code>${esc(String(telemetry.desired_hash||data.bundle?.sha256||'—').slice(0,20))}</code></span><button class="secondary" type="button" data-action="reload">Refresh sync state</button></div>${assets.length ? table(['Asset','Approved desired revision / hash','Controller applied revision / hash','Rollout progress / errors','Automatic rollout policy','Actions'],rows,'No parser assets are available.') : empty('No managed parser assets are published yet. Draft assets appear here after they are approved and published.')}`)}${panel('Fleet acknowledgement telemetry','Clients collect and parse locally, acknowledge one exact published bundle hash, and report incremental state to the controller.',table(['Client','State / heartbeat','Applied revision / hash','Alignment / target'],clientRows,'No enrolled client has reported parser bundle state yet.'))}${panel('Asset classes covered','The sync inventory includes regex definitions, patterns, static layouts, parser logic, templates, field mappings, and known-key mappings.',`<div class="sync-class-list">${['regex','patterns','static layouts / patterns','parser logic','templates','field mappings','known-key mappings'].map(name=>`<span>${esc(name)}</span>`).join('')}</div>`)}${panel('Retry and approval boundary','Verification refreshes controller-reported state. Retry only proposes a rollout for online clients or the selected client; an authorized reviewer must approve it before execution.',`<div class="callout warn">This view cannot publish drafts, approve proposals, or force an asset onto clients.</div>`)}`);
  }

  async function loadParsers() {
    const [managedResult, builtinResult, deletedResult, runtimeResult] = await Promise.allSettled([api('/parser-packs',{allowStale:true}), api('/parsers'), api('/parser-packs/deleted'), api('/parser-sync/runtime')]);
    const packs = managedResult.status === 'fulfilled' ? managedResult.value.parser_packs || [] : [];
    const builtin = builtinResult.status === 'fulfilled' ? builtinResult.value || [] : [];
    state.parserPacks=packs;
    state.builtinParsers=builtin;
    if (managedResult.status === 'rejected' && builtinResult.status === 'rejected') {
      renderPage('parsers', `<div class="callout warn">Parser catalogs could not be read: ${esc(managedResult.reason.message)}. Check controller storage readiness and retry.</div>${panel('Managed parser packs','Draft creation remains available while the live catalog recovers.',empty('No live parser-pack snapshot is available.'))}${panel('Create parser draft','Declarative only; executable fields and non-RE2 regex are rejected.',`<form id="parser-form" class="form-grid"><label class="field wide">Start from live parser${assuranceParserSelect(builtin,'auto','base_parser_id')}<span class="hint">ⓘ Select a live parser to copy its declarative definition, or leave Automatic and create your own.</span></label><label class="field">Pack ID<input name="id" required></label><label class="field">Name<input name="name" required></label><label class="field">Semantic version<input name="version" value="0.1.0" required></label><label class="field">Priority<input name="priority" type="number" min="1" max="1000000000" value="100" required></label><label class="field wide">Rules JSON<textarea name="rules">[]</textarea></label><label class="field wide">Reviewed regression cases JSON<textarea name="regression_cases">[]</textarea><span class="hint">ⓘ Include at least one representative input, expected parser ID, route, and exact expected payload. Publication is blocked until the exact runtime parser passes retained review evidence.</span></label><button class="primary" type="submit">Create draft</button></form>`)}`);
      return;
    }
    const managedStale=managedResult.value?.snapshot_stale===true;
    const warnings = staleSnapshotWarning(managedResult.value)+[managedResult, builtinResult].filter(result => result.status === 'rejected').map(result => `<div class="callout warn">${esc(result.reason.message)}</div>`).join('');
    renderPage('parsers', `${warnings}${panel('Managed parser packs','Revisioned priority, runtime compatibility, and independent review/approval state. Blocked historical records stay isolated and can be moved into an editable repair draft.',table(['Pack','Version','Priority','Revision','State','Runtime sync','Author','Definition and actions'],packs.map(doc=>{const p=doc.payload||{},sync=doc.runtime_sync||{status:'not_published'},syncStatus=String(sync.status||'').toLowerCase(),runtimeSynced=!sync.drifted&&(['active','published','synced','synchronized','in_sync','current','applied'].includes(syncStatus)||(sync.applied_revision!=null&&Number(sync.applied_revision)===Number(doc.revision)&&sync.applied_hash&&sync.applied_hash===sync.desired_hash)),displayState=doc.state==='published'&&runtimeSynced?'Published / Active':doc.state;return `<tr><td><b>${esc(p.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${esc(p.version||'—')}</td><td>${esc(p.priority||'—')}</td><td>${doc.revision}</td><td>${badge(displayState)}${approvalDetails(doc)}</td><td>${badge(sync.status||'not_published')}${sync.reason?`<br><small class="sync-error">${esc(sync.reason)}</small>`:''}</td><td>${esc(doc.created_by)}</td><td><details><summary>View full parser JSON</summary><pre class="json-output">${esc(pretty(p))}</pre></details><button class="secondary small" data-action="parser-asset" data-id="${esc(doc.id)}" data-value="edit">Edit draft</button> <button class="secondary small" data-action="parser-asset" data-id="${esc(doc.id)}" data-value="download">Download</button> <button class="danger small" data-action="parser-asset" data-id="${esc(doc.id)}" data-value="delete" ${managedStale?'disabled':''}>Delete</button> ${p.parser&&['published','approved','in_review'].includes(doc.state)?`<button type="button" class="secondary small" data-action="nfi-repair-candidate" data-id="${esc(doc.id)}">Repair as new draft</button>`:''} ${managedStale?'Refresh required for lifecycle actions':parserActions(doc)}</td></tr>`})))}${panel('Create parser draft','Declarative only; executable fields and non-RE2 regex are rejected.',`<form id="parser-form" class="form-grid"><label class="field wide">Start from live parser${assuranceParserSelect(builtin,'auto','base_parser_id')}<span class="hint">ⓘ Select a live parser to copy its declarative definition, or leave Automatic and create your own.</span></label><label class="field">Pack ID<input name="id" required title="Stable identifier; use Edit draft for existing draft packs."></label><label class="field">Name<input name="name" required></label><label class="field">Semantic version<input name="version" value="0.1.0" required></label><label class="field">Priority<input name="priority" type="number" min="1" max="1000000000" value="100" required title="Lower numbers are tried first."></label><label class="field wide">Rules JSON<textarea name="rules">[]</textarea><span class="hint">ⓘ Declarative parser rules only. Add regex, known-key, static-token and field projection stages.</span></label><label class="field wide">Reviewed regression cases JSON<textarea name="regression_cases">[]</textarea><span class="hint">ⓘ Required before submission. Each case must include name, input, expected_parser, expected_route, and expected_payload so the exact candidate can be validated before approval.</span></label><button class="primary" type="submit">Create draft</button></form>`)}${panel('Built-in runtime parsers','Read directly from the active controller registry. Inspect and download the exact registry metadata.',table(['ID','Version','Format','Description','Fields','Source'],builtin.map(item=>`<tr><td class="mono">${esc(item.id)}</td><td>${esc(item.version)}</td><td>${esc(item.format)}</td><td>${esc(item.description)}</td><td>${(item.fields||[]).length}</td><td><details><summary>Inspect JSON</summary><pre class="json-output">${esc(pretty(item))}</pre></details><button class="secondary small" data-action="download-builtin" data-id="${esc(item.id)}">Download JSON</button></td></tr>`)))}`);
    state.deletedParsers=deletedResult.status==='fulfilled'?deletedResult.value.deleted_parsers||[]:[];
    const inventory=runtimeResult.status==='fulfilled'?runtimeResult.value:null;
    const controller=inventory?.controller||{};
    page.insertAdjacentHTML('beforeend',panel('Active parser inventory and client synchronization','Canonical published parsers are materialized for inspection. Paths and hashes below come from the controller/client snapshot; deletion remains pending on a client until its exact acknowledgement.',inventory?`<p style="overflow-wrap:anywhere">Controller: ${badge(controller.status||'unknown')} · bundle <code>${esc(controller.bundle_hash||'unavailable')}</code></p><p style="overflow-wrap:anywhere">Bundle path: <code>${esc(controller.bundle_path||'not reported')}</code><br>Parser directory: <code>${esc(controller.parser_directory||'not reported')}</code><br>Inventory path: <code>${esc(controller.inventory_path||'not reported')}</code></p>${table(['Parser','Revision','Path','SHA-256'],(controller.parsers||[]).map(item=>`<tr><td>${esc(item.id||item.runtime_id)}</td><td>${esc(item.revision)}</td><td><code>${esc(item.path||'not reported')}</code></td><td><code>${esc(item.sha256||'not reported')}</code></td></tr>`))}${table(['Client','State','Desired / applied','Next action'],(inventory.clients||[]).map(client=>`<tr><td>${esc(client.node_id)}</td><td>${badge(client.exact_acknowledgement?'exact acknowledgement':client.status||'pending')}</td><td><code>${esc(client.expected_hash||client.desired_hash||'unavailable')}</code><br><code>${esc(client.applied_hash||'unavailable')}</code></td><td>${esc(client.blocking_code||'')} ${esc(client.next_action||'')}<details><summary>Reported client inventory</summary><pre>${esc(pretty(client.runtime_inventory||{}))}</pre></details></td></tr>`))}<button type="button" class="secondary" data-action="navigate" data-page="parsers">Refresh canonical inventory</button>`:`<div class="callout warn">Inventory could not be read: ${esc(runtimeResult.reason?.message||'unavailable')}. No synchronized status is assumed.</div>`)+panel('Deleted parser archive','Restore creates an inactive draft under a new ID. Validation, approval, publication, and exact client acknowledgement are required before use.',deletedResult.status==='fulfilled'?table(['Parser','Deleted','Original state','Actions'],state.deletedParsers.map(doc=>`<tr><td>${esc(doc.name||doc.id)}<br><code>${esc(doc.id)}</code></td><td>${esc(doc.deleted_at||'—')}<br>${esc(doc.deleted_by||'')}</td><td>${esc(doc.original_state||'unknown')}</td><td><button type="button" class="secondary small" data-action="parser-restore" data-id="${esc(doc.id)}" ${doc.restorable===false?'disabled':''}>Restore as draft</button></td></tr>`)):`<div class="callout warn">Archive could not be read: ${esc(deletedResult.reason?.message||'unavailable')}</div>`));
    const baseParserSelect=page.querySelector('#parser-form [name=base_parser_id]');
    baseParserSelect?.addEventListener('change',event=>{const parser=builtin.find(item=>item.id===event.currentTarget.value);if(!parser)return;const form=event.currentTarget.form;form.elements.id.value=`${parser.id.replace(/[^A-Za-z0-9._-]/g,'-')}-managed`;form.elements.name.value=`${parser.description||parser.id} managed`;form.elements.version.value=`${Number(parser.version||1)+1}.0.0`;form.elements.rules.value=pretty([parser]);});
  }
  function parserActions(doc) { const actions=[]; if(['draft','rejected'].includes(doc.state))actions.push(['submit','Submit']);if(doc.state==='in_review')actions.push(['approve','Approve'],['reject','Reject']);if(doc.state==='approved'&&doc.runtime_sync?.status!=='blocked')actions.push(['activate','Publish']);if(doc.runtime_sync?.status==='blocked')actions.push(['repair','Repair as draft']);return actions.map(([action,label])=>{const selfReview=action==='approve'&&doc.created_by===state.user?.username,blocked=selfReview&&state.user?.role!=='admin';return `<button class="secondary small" data-action="parser-action" data-id="${esc(doc.id)}" data-value="${action}" ${blocked?'disabled title="A different administrator must approve this revision"':''}>${blocked?'Awaiting administrator':selfReview?'Approve exact revision':label}</button>`;}).join(' '); }

  const ENGINE_SPECS = {
    'engine-network': {slug:'network-identity',title:'Network Identity Engine',description:'Recognizes IPv4, IPv6, hostnames and domains as typed, source-backed fields.',sample:'src=192.0.2.10 dst=2001:db8::2 host=mail.example.com',expected:{src_ip:'192.0.2.10',dst_ip:'2001:db8::2',host:'mail.example.com'},family:'network'},
    'engine-url': {slug:'url',title:'URL Engine',description:'Extracts URI scheme, host, port, path, query and fragment while preserving original URL evidence.',sample:'request="GET https://api.example.com:8443/v1/events?limit=10 HTTP/1.1"',expected:{scheme:'https',host:'api.example.com',port:8443,path:'/v1/events',query:'limit=10'},family:'url'},
    'engine-email': {slug:'email-address',title:'Email Address Engine',description:'Finds mailbox identities in common from, to, reply-to and notification fields.',sample:'from=alice@example.org to=ops+alerts@example.net status=queued',expected:{from:'alice@example.org',to:'ops+alerts@example.net',status:'queued'},family:'email'},
    'engine-subject': {slug:'encoded-subject',title:'Encoded Subject Engine',description:'Decodes MIME RFC 2047 encoded-word subjects (Base64 or quoted-printable) into UTF-8. This is content encoding, not encryption.',sample:'subject="=?UTF-8?B?SGVsbG8gZnJvbSBOb3J0aA==?="',expected:{subject:'Hello from North'},family:'email'},
  };
  async function loadDomainEngine(id) {
    const spec=ENGINE_SPECS[id],[capResult,parserResult]=await Promise.allSettled([api('/config/semantic-extraction'),api('/parsers')]);
    const assets=capResult.status==='fulfilled'?(capResult.value.items||[]):[];
    const entity=spec.family==='network'?['ipv4','ipv6','domain']:spec.family==='url'?['url']:spec.family==='email'?['email']:[];
    const asset=assets.find(doc=>(doc.payload?.config?.entities||[]).some(value=>entity.includes(value))||doc.payload?.config?.decode_mime_headers&&id==='engine-subject');
    const capability=asset?{...(asset.payload?.config||{}),id:asset.id,revision:asset.revision,state:asset.state,enabled:asset.state==='active'||asset.payload?.config?.enabled===true,priority:asset.payload?.priority}:null;
    const all=parserResult.status==='fulfilled'?(parserResult.value.parsers||parserResult.value||[]):[];
    const engines=all.filter(item=>`${item.id||''} ${item.name||''} ${item.format||''} ${item.family||''}`.toLowerCase().includes(spec.family));
    const status=capability?`<div class="grid cols-3">${card('Enabled state',capability.enabled?'Enabled':capability.state==='draft'?'Draft · not confirmed active':'Disabled / inactive',capability.state||'Controller catalog')}${card('Priority',capability.priority??'Controller default','Controller policy order')}${card('Revision',capability.revision||'—',asset?.id||spec.slug)}</div>`:'<div class="callout">No semantic extraction policy is configured on this controller yet. The sample and Assurance tester remain available; create the policy from the Semantic Extraction Policy tab.</div>';
    renderPage(id,`${panel(spec.title,spec.description,`${status}<div class="engine-grid"><div><h4>Sample input</h4><pre class="json-output">${esc(spec.sample)}</pre></div><div><h4>Illustrative structured fields</h4><pre class="json-output">${esc(pretty(spec.expected))}</pre><small class="hint">Example only. Actual results come from the active production parser.</small></div></div><div class="actions"><button class="primary" data-action="engine-test" data-page="${id}">Test sample in Assurance</button><a class="secondary download-link" href="#semantic">Semantic policy controls</a><a class="secondary download-link" href="#studio">Regex, Templates &amp; Studio</a><a class="secondary download-link" href="#projection">Field Projections</a></div>`)}${panel('Runtime catalog','The controller reports which built-in parsers are enabled.',engines.length?table(['Engine','Version','Format','Description','Fields'],engines.map(item=>`<tr><td>${esc(item.id)}</td><td>${esc(item.version)}</td><td>${esc(item.format)}</td><td>${esc(item.description)}</td><td>${(item.fields||[]).length}</td></tr>`)):empty('No built-in parser advertises this engine family. Create or enable a declarative parser in the catalog, then validate it in Assurance.'))}${capResult.status==='rejected'?`<div class="callout">Capability API unavailable: ${esc(capResult.reason.message)}. This fallback does not claim the engine is active.</div>`:''}`);
  }

  async function loadMigration() {
    let reports = [], reportWarning = '';
    try { const data = await api('/migration/reports'); reports = data.reports || []; } catch (error) { reportWarning = error.message; }
    state.migrationReports=reports;
    renderPage('migration', `${reportWarning ? `<div class="callout warn">${esc(reportWarning)}</div>` : ''}${panel('Safe parser-source import','Select an existing Alloy/River, Vector/VRL, or NLI source file, or paste source. The importer analyzes and translates supported constructs; it never executes imported code.',`<form id="migration-form" class="form-grid"><label class="field">Source file<input name="source_file" type="file" accept=".alloy,.river,.toml,.vrl,.json,.txt" title="Choose the parser source you want to analyze. File contents are sent to the controller for static analysis."></label><label class="field">Filename<input name="filename" placeholder="pipeline.alloy"></label><label class="field">Source type<select name="source_type"><option value="alloy">Alloy / River</option><option value="vector">Vector TOML / VRL</option><option value="nli">NLI pack JSON</option></select></label><label class="field wide">Parser source<textarea name="content" placeholder="Choose a source file or paste the complete source here" title="Paste the entire source when no file is selected. Import is static analysis and reports supported/unsupported operations."></textarea><span class="hint">ⓘ Import results include source hash, extracted patterns and fields where recognized, compatibility gaps, and optional review draft.</span></label><label class="field"><input type="checkbox" name="create_draft" value="true"> Create reviewable draft</label><button class="primary" type="submit">Analyze import</button></form>`)}${panel('Compatibility reports','Inspect the complete translated logic, extracted fields, pattern expressions, and unsupported constructs for each import.',table(['Source','Type','Hash','Supported','Unsupported','Parity','Draft','Report'],reports.map(doc=>{const p=doc.payload||{};return `<tr><td>${esc(p.filename)}</td><td>${esc(p.source_type)}</td><td class="mono">${esc((p.source_hash||'').slice(0,16))}</td><td>${(p.supported||[]).length}</td><td>${(p.unsupported||[]).length}</td><td>${badge(p.parity_status)}</td><td class="mono">${esc(p.draft_id||'—')}</td><td><details><summary>Inspect extracted logic</summary><pre class="json-output">${esc(pretty(p))}</pre></details><button class="secondary small" data-action="download-migration-report" data-id="${esc(doc.id)}">Download report</button></td></tr>`})))} `);
  }

  async function retainTestSample(request, origin) {
    try{
      const result=await api('/assurance/cases',{method:'POST',body:{name:'Test sample '+origin,description:'Manual sample saved from '+origin,retain:true,case:request}});
      toast('Encrypted sample saved to Test Sample History.');
      return result.id;
    }catch(error){
      notify('Sample was not saved: '+error.message,'error');
      return null;
    }
  }
  async function loadSampleHistory(){
    const data=await api('/assurance/cases');
    const samples=(data.cases||[]).filter(item=>item.id?.startsWith('test-sample-'));
    renderPage('sample-history',panel('Test Sample History','Reopen encrypted manual samples in Guided Parser Repair. The controller applies access and retention rules.',`<label class="field"><span><input id="sample-history-automatic" type="checkbox" ${state.sampleHistoryEnabled?'checked':''} ${can('assurance:save')?'':'disabled'}> Automatically save pasted test samples during this sign-in</span><small>You can uncheck retention for any individual sample. Signing out turns automatic retention off.</small></label>`+table(['Sample','Saved','Actions'],samples.map(item=>`<tr><td>${esc(item.name)} <code>${esc(item.id)}</code></td><td>${time(item.created_at)}</td><td><button class="secondary" data-action="test-sample-open" data-id="${esc(item.id)}">Open in Guided Repair</button> <button class="secondary" data-action="download-case" data-id="${esc(item.id)}">Download</button></td></tr>`))));
    el('sample-history-automatic')?.addEventListener('change',event=>{state.sampleHistoryEnabled=event.currentTarget.checked;});
  }
  async function loadConfig(kind) {
    const [path,title,example]=CONFIG[kind]; let data={items:[]}, warning='';
    try { data=await api(path); } catch(error) { warning=`<div class="callout warn">Live ${esc(title.toLowerCase())} could not be loaded: ${esc(error.message)}. Defaults are shown for reference; retry after controller storage recovers.</div>`; }
    const items=data.items||[], known=kind==='knownkeys';
    const lifecycle = doc => {
      const actions=[];
      if (['draft','rejected'].includes(doc.state)) actions.push(['submit','Submit for review']);
      if (doc.state === 'in_review') actions.push(['approve','Approve']);
      if (doc.state === 'approved') actions.push(['publish',known?'Publish + request fleet rollout':'Publish + auto-rollout']);
      if (known && doc.state === 'active') actions.push(['revoke','Revoke published keys']);
      if (known && doc.state === 'revoked') actions.push(['revoke','Retry fleet removal']);
      return actions.map(([value,label])=>` <button class="secondary small" data-action="config-action" data-kind="${kind}" data-id="${esc(doc.id)}" data-value="${value}">${label}</button>`).join('');
    };
    let templateWorkbench = '';
    if (['header-templates','kv-templates'].includes(kind)) {
      const otherPath = CONFIG[kind === 'header-templates' ? 'kv-templates' : 'header-templates'][0];
      const other = await api(otherPath).catch(() => ({items:[]}));
      const select = (id,label,docs) => '<label class="field">' + label + '<select id="' + id + '">' +
        '<option value="">Select a saved revision</option>' + docs.filter(doc => doc.id && Number(doc.revision)>0).map(doc =>
          '<option value="' + esc(JSON.stringify({id:doc.id,revision:Number(doc.revision)})) + '">' +
          esc((doc.payload?.name || doc.id) + ' · revision ' + doc.revision) + '</option>').join('') + '</select></label>';
      const header = kind === 'header-templates' ? items : other.items || [];
      const kv = kind === 'kv-templates' ? items : other.items || [];
      const artifact = kind === 'header-templates' ? 'template' : 'kv_template';
      templateWorkbench = panel('Build a template from samples','Paste one representative record per line, up to 500. Header drafts capture the prefix. KV drafts accept a recognized mail record or a standalone key/value tail.',
        '<label class="field">Sample log<textarea id="template-composition-sample" rows="6" spellcheck="false"></textarea></label>' +
        '<div class="actions"><button type="button" class="primary" data-action="template-draft-generate" data-artifact="' + artifact + '">Generate draft JSON</button>' +
        '<button type="button" class="secondary" data-action="workspace-ai-open" data-template-sample data-mode="fix" data-tab="' + title + '" data-artifact="' + artifact + '">Draft with Nexora AI</button>' +
        '<button type="button" class="secondary" data-action="template-reset-samples">Reset samples</button>' +
        '<button type="button" class="secondary" data-action="template-draft-verify" data-artifact="' + artifact + '">Verify current JSON</button></div>' +
        '<pre class="json-output" id="template-draft-result" role="status" hidden></pre>') +
        panel('Preview header and KV together','Choose exact saved revisions and verify both templates against the sample logs above. Open the verified mapping in FieldSense to build a parser pack for review.',
        select('template-header','Header template',header) + select('template-kv','KV pair template',kv) +
        '<div class="actions"><button type="button" class="primary" data-action="template-preview-saved">Verify saved templates</button>' +
        '<button type="button" class="secondary" data-action="template-compose-mapping">Open combined mapping in FieldSense</button></div>' +
        '<pre class="json-output" id="template-composition-result" role="status" hidden></pre>');
    }
    renderPage(kind, `${warning}${templateWorkbench}${panel(title,'Revisioned declarative assets stored in Supabase metadata. Drafts require independent review; publishing builds the canonical asset bundle and starts automatic fleet rollout.',table(['Name','ID','Revision','State','Updated','Configuration','Actions'],items.map(doc=>`<tr><td>${esc(doc.payload?.name||doc.id)}</td><td class="mono">${esc(doc.id)}</td><td>${doc.revision}</td><td>${badge(doc.state)}${approvalDetails(doc)}</td><td>${time(doc.updated_at)}</td><td><details><summary>View full content</summary><pre class="json-output">${esc(pretty(doc.payload?.config))}</pre></details></td><td><button class="secondary small" data-action="config-action" data-kind="${kind}" data-id="${esc(doc.id)}" data-value="${known&&!['draft','rejected'].includes(doc.state)?'copy':'edit'}">${known&&!['draft','rejected'].includes(doc.state)?'Copy to new draft':'Edit draft'}</button> <button class="secondary small" data-action="config-action" data-kind="${kind}" data-id="${esc(doc.id)}" data-value="download">Download JSON</button>${!known||['draft','rejected'].includes(doc.state)?` <button class="danger small" data-action="config-action" data-kind="${kind}" data-id="${esc(doc.id)}" data-value="delete">${known?'Delete draft':'Delete'}</button>`:''}${lifecycle(doc)}</td></tr>`)))}${panel('Create or revise definition','Saving creates a draft revision in Supabase. It does not affect live parsing until independent approval and publication.',`<form id="config-form" data-kind="${kind}" class="form-grid"><label class="field">ID<input name="id" required title="Stable identifier for this asset. Reusing an ID creates a new revision."></label><label class="field">Name<input name="name" required></label>${known?`<label class="field wide">Known keys<textarea name="known_keys" rows="8" required spellcheck="false" placeholder="status&#10;reason&#10;source_ip">${esc((example.keys||[]).join('\n'))}</textarea><span class="hint">One key per line. Add, rename or remove lines, then save the draft. Published keys annotate observed FieldSense suggestions. Copy a published definition to a new draft, then revoke the prior definition after reviewing its replacement.</span></label>`:''}<label class="field wide">${known?'Advanced metadata JSON':'Configuration JSON'}<textarea name="config" required>${esc(pretty(known?{}:example))}</textarea><span class="hint">ⓘ Declarative JSON only. Validate the draft in Assurance, submit it, obtain independent approval, then publish to trigger verified fleet synchronization.</span></label><button class="primary" type="submit">Save draft revision</button></form>`)}`);
  }

  const INPUT_FAMILIES = ['auto','imap','pop','webtop','mta','smtp-in','smtp-out','pss','pabd','capd'];
  const multilineForFamily = service => ({enabled:true,max_bytes:2097152,max_lines:10000,flush_timeout_ms:4000,rules:service==='auto'?['java_exception','python_traceback','go_panic']:['generic_continuation']});
  const defaultInputProfile = () => ({inputs:INPUT_FAMILIES.map(service=>({path:`/var/log/nli/input/${service}.log`,service,multiline:multilineForFamily(service),mask:{disable_types:[]}}))});
  function inputPathRows(config) {
    const inputs=Array.isArray(config?.inputs)?config.inputs:[];
    return inputs.map((item,index)=>{
      const path=esc(item.path||'input path');
      const toggle=item.disabled===true?'Enable':'Disable';
      return `<tr><td><input aria-label="Log path ${index+1}" value="${esc(item.path||'')}" data-input-path="${index}"></td>
        <td><select aria-label="Parser family for ${path}" data-input-family="${index}">${INPUT_FAMILIES.map(family=>`<option value="${family}" ${item.service===family?'selected':''}>${family==='webtop'?'webtop-webmail':family}</option>`).join('')}</select></td>
        <td>${badge(item.disabled===true?'disabled':'enabled')} <button class="secondary small" type="button" data-action="input-edit-row" data-index="${index}" aria-label="Edit ${path}">Edit</button>
          <button class="secondary small" type="button" data-action="input-toggle-row" data-index="${index}" aria-label="${toggle} ${path}">${toggle}</button>
          <button class="danger small" type="button" data-action="input-remove-row" data-index="${index}" aria-label="Remove ${path}">Remove</button></td></tr>`;
    });
  }

  function updateInputPathEditor(form,config){
    form.elements.config.value=pretty(config);
    el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config),'No paths configured.');
    page.dataset.dirty='true';
    applyPermissions();
  }
  const defaultFamilyOutputs = () => INPUT_FAMILIES.flatMap(service=>{const prefix=service==='webtop'?'webtop-webmail':service;return [{type:'file',path_template:`/var/log/nli/output/${prefix}_normalised_{hour_utc}.jsonl`,routes:['normal','warning'],families:[service],max_mb:500,max_backups:5},{type:'file',path_template:`/var/log/nli/output/${prefix}_quarantine_{hour_utc}.jsonl`,routes:['quarantine','unknown'],families:[service],max_mb:500,max_backups:5,durable_raw:true}]});
  let inputPathsLoad = 0;
  async function loadInputPaths() {
    const load = ++inputPathsLoad;
    const [profilesResult,fleetResult]=await Promise.allSettled([api('/config/input-paths'),api('/fleet/sync')]);
    if(load!==inputPathsLoad || state.page!=='input-paths')return;
    if(profilesResult.status==='rejected')throw profilesResult.reason;
    const items=profilesResult.value.items||[],fleet=fleetResult.status==='fulfilled'?(fleetResult.value.nodes||[]):[];
    const activeProfiles=items.filter(doc=>doc.state==='active');
    if(!activeProfiles.some(doc=>doc.id===state.inputHistoryProfile))state.inputHistoryProfile=activeProfiles[0]?.id||'';
    const requestedProfile=state.inputHistoryProfile;
    let rolloutStatus=requestedProfile?await api(`/config/input-paths/${encodeURIComponent(requestedProfile)}/rollout`).catch(error=>({error:error.message,history:[],clients:[]})):{history:[],clients:[]};
    if(load!==inputPathsLoad || state.page!=='input-paths' || requestedProfile!==state.inputHistoryProfile)return;
    const requestedRevision=activeProfiles.find(doc=>doc.id===requestedProfile)?.revision;
    if(requestedProfile && !rolloutStatus.error && (rolloutStatus.profile_id!==requestedProfile || (rolloutStatus.profile_revision!=null && Number(rolloutStatus.profile_revision)!==Number(requestedRevision)))){
      rolloutStatus={error:'The selected profile changed during refresh. Refresh to load its current rollout status.'};
    }
    const clients=fleet.map(item=>item.node||item).filter(node=>node.id||node.node_id).map(node=>({id:node.id||node.node_id,hostname:node.hostname||node.name||node.id||node.node_id,status:node.status||'unknown',last_seen:node.last_seen||node.updated_at}));
    const targetRows=clients.map(node=>`<label class="client-select-row"><input type="checkbox" name="client_target" value="${esc(node.id)}"><span><b>${esc(node.hostname)}</b><small class="mono">${esc(node.id)} · ${esc(node.status)}</small></span><time>${time(node.last_seen)}</time></label>`).join('');
    const historyRows=(rolloutStatus.history||[]).map(item=>`<tr><td class="mono">${esc(item.id)}</td><td class="mono">${esc((item.target_nodes||[]).join(', ')||'—')}</td><td>${badge(item.state||item.rollout?.status||'requested')}</td><td>${time(item.created_at)}</td></tr>`);
    const revisionLabel=profile=>{if(!profile)return '—';const name=items.find(doc=>doc.id===profile.id)?.payload?.name;return `${name&&name!==profile.id?`${name} (${profile.id})`:profile.id} · r${number(profile.revision)}`;};
    const syncRows=(rolloutStatus.clients||[]).map(item=>{const sync=item.input_path_sync||{};return `<tr><td><b>${esc(item.hostname||item.node_id)}</b><br><small class="mono">${esc(item.node_id)}</small></td><td>${badge(item.status||'unknown')}</td><td>${badge(sync.input_path_sync_status||sync.status||'not_targeted')}</td><td class="mono">${esc(revisionLabel(sync.desired_input_path_profile))}</td><td class="mono">${esc(revisionLabel(sync.applied_input_path_profile))}</td><td>${time(sync.input_path_sync_updated_at)}</td><td>${esc(sync.input_path_sync_message||'—')}</td></tr>`;});
    const families=INPUT_FAMILIES.map(value=>`<option value="${value}">${value==='webtop'?'webtop-webmail':value}</option>`).join('');
    const profileOptions=activeProfiles.map(doc=>`<option value="${esc(doc.id)}">${esc(doc.payload?.name||doc.id)} · r${doc.revision}</option>`).join('');
    const inputLifecycle=doc=>{const actions=[];if(['draft','rejected'].includes(doc.state))actions.push(['submit','Submit for review']);if(doc.state==='in_review')actions.push(['approve','Approve']);if(doc.state==='approved')actions.push(['publish','Publish']);return actions.map(([value,label])=>` <button class="secondary small" data-action="config-action" data-kind="input-paths" data-id="${esc(doc.id)}" data-value="${value}">${label}</button>`).join('');};
    const historyOptions=activeProfiles.map(doc=>`<option value="${esc(doc.id)}" ${doc.id===state.inputHistoryProfile?'selected':''}>${esc(doc.payload?.name||doc.id)} · r${doc.revision}</option>`).join('');
    renderPage('input-paths', `<div class="dashboard-intro"><div><p class="eyebrow">Client collection · reviewed family routing</p><h1>Input paths and family profiles</h1><p>Manage bounded parallel client tail inputs, then request independent approval before selected or all eligible nodes receive an inputs-only configuration patch.</p></div><span class="live-pill">Up to 256 paths per agent</span></div>
      ${panel('Saved input profiles','Profiles are revisioned controller configuration documents. Drafts need review and publication before rollout; use the row actions to advance their lifecycle.',`${table(['Profile','Revision','State','Inputs','Updated','Actions'],items.map(doc=>{const config=doc.payload?.config||{};return `<tr><td><b>${esc(doc.payload?.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${doc.revision}</td><td>${badge(doc.state)}${approvalDetails(doc)}</td><td>${number(config.inputs?.length||0)}</td><td>${time(doc.updated_at)}</td><td><button class="secondary small" data-action="input-profile-edit" data-id="${esc(doc.id)}">Edit</button> <button class="secondary small" data-action="input-profile-json" data-id="${esc(doc.id)}">View / download JSON</button>${inputLifecycle(doc)} <button class="danger small" data-action="input-profile-delete" data-id="${esc(doc.id)}">Remove</button></td></tr>`;}))}<div id="input-profile-detail" class="input-profile-detail" aria-live="polite"></div>`)}
      ${panel('Live clients','Choose enrolled clients for a targeted rollout. You can include offline clients and queue delivery until they check in.',`<div class="client-selection-toolbar"><label><input type="checkbox" id="input-select-all" ${clients.length?'':'disabled'}> Select all ${clients.length} clients</label><span>${fleetResult.status==='rejected'?`Fleet unavailable: ${esc(fleetResult.reason.message)}`:`${clients.length} live client records`}</span></div><div class="client-selection-list">${targetRows||empty('No enrolled clients are currently available.')}</div>`)}
      <div class="grid cols-2"><section class="panel"><div class="section-head"><div><h3>Profile editor</h3><p>Add one path or manage multiple inputs. Existing entries stay in place as paths are added.</p></div></div><form id="input-profile-form" class="form-grid"><label class="field">Profile ID<input name="id" required pattern="[A-Za-z0-9_-]+" maxlength="64"></label><label class="field">Name<input name="name" required maxlength="128"></label><label class="field">Add log path<input name="quick_path" placeholder="/var/log/nli/input/imap.log" aria-label="Add log path"></label><label class="field">Parser family<select name="quick_family" aria-label="Default parser family">${families}</select></label><div class="actions wide"><button class="secondary" type="button" data-action="input-add-row">Add path with selected family</button><button class="secondary" type="button" data-action="input-use-automatic">Load all standard paths</button><button class="secondary" type="button" data-action="input-validate">Validate paths</button><button class="secondary" type="button" data-action="input-toggle-all" data-value="enable">Enable all draft paths</button><button class="secondary" type="button" data-action="input-toggle-all" data-value="disable">Disable all draft paths</button><button class="danger" type="button" data-action="input-remove-all">Remove all draft paths</button></div><div class="wide" id="input-path-list" aria-live="polite">${table(['Log path','Family','Status and actions'],inputPathRows({inputs:INPUT_FAMILIES.map(service=>({path:`/var/log/nli/input/${service}.log`,service}))}),'No paths configured.')}</div><label class="field wide">Input definitions JSON<textarea name="config" required rows="10" aria-label="Input definitions JSON">${esc(pretty(defaultInputProfile()))}</textarea><span class="hint">You can edit the JSON directly; edit paths and parser families or enable, disable and remove paths here. To apply these changes, save and publish this revision, then select specific clients or the eligible fleet for rollout. Removal stops collection and preserves the source log on disk. <b>auto</b> runs automatic parser detection. Maximum 256 unique absolute paths.</span></label><div class="actions wide"><button class="primary" type="submit">Save profile revision</button><button class="secondary" type="button" data-action="input-profile-reset">New profile</button></div></form></section>
      <section class="panel"><div class="section-head"><div><h3>Roll out to clients</h3><p>The dedicated controller contract replaces only client inputs and preserves outputs, secrets, enrollment and telemetry.</p></div></div><form id="input-rollout-form" class="form-grid"><label class="field wide">Published path profiles<select name="profile_id" multiple size="5" required aria-describedby="input-profile-select-hint">${profileOptions}</select><span id="input-profile-select-hint" class="hint">Select one or more active profiles. Draft, review, and approved profiles can be advanced from Saved input profiles above.</span></label><label class="field wide output-choice"><input type="checkbox" name="all_eligible" value="true"> Target every eligible online client</label><label class="field wide output-choice"><input type="checkbox" name="include_offline" value="true" disabled> Include offline enrolled clients. Apply when they reconnect.</label><label class="field wide">Selected target node IDs<input name="targets" placeholder="Select clients above or enter exact enrolled IDs"></label><button class="primary" type="submit">Request independent approval</button><span class="hint wide">An empty node list is accepted only when “Target every eligible online client” is selected.</span></form></section></div>
      ${panel('Reported client input files','Compare the client-reported config and file presence with the published profile. A removed path no longer appears in the applied collection list.',table(['Client','Last heartbeat','Configured inputs','Input rollout state'],fleet.map(({node={},sync={}})=>`<tr><td>${esc(node.hostname||node.id)}</td><td>${time(node.last_seen)}</td><td>${renderReportedInputs(sync)}</td><td>${badge(sync.input_path_sync_status||'not_targeted')}</td></tr>`),'No clients have reported inputs.'))}
      ${renderManagedFileForm(fleet)}
      ${panel('Input-path rollout sync','Choose an active profile to compare the target revision with the last revision acknowledged by each client.',`<label class="field">Profile<select id="input-history-profile">${historyOptions}</select></label>${rolloutStatus.error?`<div class="callout">${esc(rolloutStatus.error)}</div>`:table(['Client','Fleet','Input sync','Desired profile','Applied profile','Updated','Message'],syncRows,'No client sync state has been reported for this profile.')}`)}
      ${panel('Rollout history','Approval and delivery history for the selected active input profile.',table(['Proposal','Target clients','Status','Created'],historyRows,'No input profile rollouts have been requested yet.'))}`);
  }
  async function loadOutputEngine() {
    const result = await api('/config/output-destinations').catch(error => { throw new Error(`Output profiles could not load: ${error.message}`); });
    const profiles = result.items || [];
    const lifecycle = doc => {
      const actions=[];
      if (['draft','rejected'].includes(doc.state)) actions.push(['submit','Submit for review']);
      if (doc.state==='in_review') actions.push(['approve','Approve']);
      if (doc.state==='approved') actions.push(['publish','Publish profile']);
      return actions.map(([value,label])=>` <button class="secondary small" data-action="config-action" data-kind="outputs" data-id="${esc(doc.id)}" data-value="${value}">${label}</button>`).join('');
    };
    renderPage('outputs', `<div class="engine-intro"><p class="eyebrow">Endpoint Configuration</p><h1>Endpoint Configuration</h1><p>Configure local JSONL, Loki, and ClickHouse destinations for fully parsed events. Quarantine records stay in the local retryable quarantine queue and are never sent to remote destinations. Credentials are referenced by environment variable or protected file only.</p></div>
      <div class="engine-flow" aria-label="Parser data flow"><span>Input files</span><b>→</b><span>Parser selection</span><b>→</b><span>Field transforms</span><b>→</b><span class="active">Output routes</span></div>
      ${renderRecordAccountingHelp()}
      ${panel('Supported output adapters','Each adapter uses the same normalized customer event. Controller metadata continues through the protected telemetry channel.',`<div class="grid cols-4">${card('Local JSONL','Available','Rotated files; parsed and quarantine streams are separate')}${card('Grafana Loki','Available','Batched push API; TLS CA and bearer token supported')}${card('ClickHouse','Available','HTTP JSONEachRow; mapped event fields become table columns')}${card('Controller endpoint','Available','Authenticated controller event endpoint; token file based')}</div><div class="callout">Remote destinations must use HTTPS in production. The profile stores credential variable names or protected file paths only; it never stores secret values. Loki and ClickHouse delivery failures require a local fallback spool.</div>`)}
      ${panel('Create output destination profile','Profiles are saved as revisioned Supabase drafts. Review and approval control publication; the rollout form applies a validated outputs-only patch to selected enrolled clients.',`<form id="output-profile-form" class="form-grid"><label class="field">Profile ID<input name="id" required pattern="[A-Za-z0-9._-]{1,64}" value="customer-output"></label><label class="field">Profile name<input name="name" required maxlength="120" value="Customer output profile"></label>
        <fieldset class="field wide output-choice"><legend>Destinations</legend><label><input type="checkbox" name="normalised_enabled" value="true" checked> Write parsed events to local normalised JSONL</label><label><input type="checkbox" name="destinations" value="loki"> Grafana Loki</label><label><input type="checkbox" name="destinations" value="clickhouse"> ClickHouse</label><label><input type="checkbox" name="destinations" value="controller"> Controller endpoint</label><small class="hint">Local quarantine JSONL is always enabled for raw unparsed records. Loki and ClickHouse continue to receive parsed events when local normalised output is disabled.</small></fieldset>
        <label class="field">Parsed JSONL path or hourly template<input name="normal_path" value="/var/log/nli/output/auto_normalised_{hour_utc}.jsonl"></label><label class="field">Required quarantine path or hourly template<input name="quarantine_path" value="/var/log/nli/output/auto_quarantine_{hour_utc}.jsonl" required></label>
        <label class="field wide">Loki HTTPS endpoint<input name="loki_url" type="url" placeholder="https://loki.example.com/api/v1/push" title="Editable sample only; replace it with your Loki endpoint."></label><label class="field">Loki bearer token environment variable<input name="loki_token_env" placeholder="NLI_LOKI_TOKEN"></label><label class="field">Loki tenant ID<input name="loki_tenant" placeholder="optional"></label><label class="field wide">Loki labels JSON<textarea name="loki_labels">{"job":"nli"}</textarea></label>
        <label class="field wide">ClickHouse HTTPS endpoint<input name="clickhouse_url" type="url" placeholder="https://clickhouse.example.com:8443"></label><label class="field">Database<input name="clickhouse_database" value="default"></label><label class="field">Table<input name="clickhouse_table" value="nli_logs"></label><label class="field">Username<input name="clickhouse_username" autocomplete="off"></label><label class="field">Password environment variable<input name="clickhouse_password_env" placeholder="NLI_CLICKHOUSE_PASSWORD"></label><label class="field wide">Source field → ClickHouse column mappings<textarea name="field_columns">{"fromhost":"fromhost","user":"user"}</textarea><span class="hint">JSON object. When mappings are present, only mapped source fields become table columns; event_json retains the complete typed customer object. Leave mappings empty to create bounded safe columns for every top-level field.</span></label>
        <label class="field wide">Controller HTTPS endpoint<input name="controller_url" type="url" placeholder="https://controller.example.com/api/v1"></label><label class="field">Controller token file<input name="controller_token_file" placeholder="/etc/nli/secrets/controller-token"></label><label class="field">Node ID file<input name="controller_node_id_file" value="/var/lib/nli/node-id"></label><div class="actions wide"><button class="primary" type="submit">Validate and save draft</button><button class="secondary" type="button" data-action="output-config-test">Validate configuration</button><button class="secondary" type="button" data-action="download-output-config" ${state.outputPreview?'':'disabled'}>Download outputs JSON</button></div><p id="output-config-test-result" class="hint wide" role="status">Local profile validation checks structure and safe routing. Network connectivity is not tested here; verify reachability from the enrolled client.</p></form><pre id="output-profile-preview" class="json-output" aria-live="polite">${state.outputPreview?esc(pretty(state.outputPreview)):'Choose destinations and save to preview the validated outputs array.'}</pre>`)}
      <section class="panel"><h3>Native field mapping before delivery</h3><p>Use NFI to split and normalize source fields, then publish that parser before selecting those resulting JSON keys in an endpoint field map. Delivery profiles keep their existing independent review and rollout controls.</p><button type="button" class="secondary" data-action="nfi-open-context" data-target="outputs">Open NFI field workbench</button></section>${panel('Saved output profiles','Profiles remain drafts until reviewed. Approved profiles are applied to selected clients through an independent rollout proposal.',profiles.length?table(['Profile','Revision','State','Destinations','Actions'],profiles.map(doc=>`<tr><td><b>${esc(doc.payload?.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${doc.revision}</td><td>${badge(doc.state)}${approvalDetails(doc)}</td><td>${(doc.payload?.config?.outputs||[]).map(out=>esc(out.type)).join(', ')||'—'}</td><td><button class="secondary small" data-action="config-action" data-kind="outputs" data-id="${esc(doc.id)}" data-value="download">Download JSON</button> <button class="secondary small" data-action="config-action" data-kind="outputs" data-id="${esc(doc.id)}" data-value="edit">Edit draft</button> <button class="danger small" data-action="config-action" data-kind="outputs" data-id="${esc(doc.id)}" data-value="delete">Remove</button>${lifecycle(doc)}</td></tr>`)):empty('No output profiles have been saved yet.'))}${panel('Roll out endpoint profile','The client applies only the outputs section and preserves its input, enrollment, telemetry, parser and enrichment configuration.',`<form id="output-rollout-form" class="form-grid"><label class="field wide">Published endpoint profile<select name="profile_id" required>${profiles.filter(doc=>doc.state==='active').map(doc=>`<option value="${esc(doc.id)}">${esc(doc.payload?.name||doc.id)} · r${doc.revision}</option>`).join('')}</select></label><label class="field wide">Target node IDs<input name="targets" required placeholder="node-id-1, node-id-2"></label><button class="primary" type="submit" ${profiles.some(doc=>doc.state==='active')?'':'disabled'}>Request independent approval</button><span class="hint wide">Only active profiles can be proposed. A different authorized operator must approve the rollout before enrolled clients receive the validated config patch.</span></form>`)}`);
  }

  function buildOutputProfile(form) {
    const data=formObject(form), selected=new FormData(form).getAll('destinations'), normalizedEnabled=new FormData(form).has('normalised_enabled');
    if (!selected.length && !normalizedEnabled) throw new Error('Enable local normalised JSONL or select a remote parsed-event destination');
    const outputs=[];
    if (!data.quarantine_path.startsWith('/')) throw new Error('Quarantine JSONL path must be absolute');
    if (normalizedEnabled) {
      if (!data.normal_path.startsWith('/')) throw new Error('Parsed JSONL path must be absolute');
      outputs.push({type:'file',[data.normal_path.includes('{hour_utc}')?'path_template':'path']:data.normal_path,routes:['normal','warning'],max_mb:500,max_backups:5});
    }
    outputs.push({type:'file',[data.quarantine_path.includes('{hour_utc}')?'path_template':'path']:data.quarantine_path,routes:['quarantine','unknown'],max_mb:500,max_backups:5,durable_raw:true});
    const remoteRoutes=['normal','warning'], fallback='/var/log/nli/output/delivery-fallback.jsonl';
    const endpoint=(value,label)=>{let parsed;try{parsed=new URL(value);}catch{throw new Error(`${label} must be a valid HTTPS URL`);}if(parsed.protocol!=='https:'||parsed.username||parsed.password||parsed.hash)throw new Error(`${label} must use HTTPS and cannot contain credentials or a fragment`);return parsed.href;};
    const env=(value,label)=>{if(value && !/^[A-Z][A-Z0-9_]{0,63}$/.test(value))throw new Error(`${label} must be an environment variable name such as NLI_REMOTE_TOKEN`);return value;};
    if(selected.includes('loki')){const labels=parseJSON(data.loki_labels,'Loki labels');if(!labels||Array.isArray(labels)||typeof labels!=='object'||Object.values(labels).some(value=>typeof value!=='string'))throw new Error('Loki labels must be a JSON object of string values');const item={type:'loki',url:endpoint(data.loki_url,'Loki endpoint'),routes:remoteRoutes,fallback_path:fallback,batch_size:250,flush_sec:5};if(data.loki_token_env)item.bearer_token_env=env(data.loki_token_env,'Loki token variable');if(data.loki_tenant)item.tenant_id=data.loki_tenant;item.labels=labels;outputs.push(item);}
    if(selected.includes('clickhouse')){const fieldColumns=parseJSON(data.field_columns,'ClickHouse mappings');if(!fieldColumns||Array.isArray(fieldColumns)||typeof fieldColumns!=='object'||Object.entries(fieldColumns).some(([source,target])=>!source.trim()||typeof target!=='string'||!/^[A-Za-z_][A-Za-z0-9_]{0,127}$/.test(target)))throw new Error('ClickHouse mappings must be a JSON object with safe column identifiers');for(const [key,value] of [['database',data.clickhouse_database],['table',data.clickhouse_table]])if(!/^[A-Za-z_][A-Za-z0-9_]{0,127}$/.test(value))throw new Error(`ClickHouse ${key} must be an identifier`);const item={type:'clickhouse',url:endpoint(data.clickhouse_url,'ClickHouse endpoint'),database:data.clickhouse_database,table:data.clickhouse_table,routes:remoteRoutes,fallback_path:fallback,batch_size:250,flush_sec:5,field_columns:fieldColumns,max_field_columns:256};if(data.clickhouse_username)item.username=data.clickhouse_username;if(data.clickhouse_password_env)item.password_env=env(data.clickhouse_password_env,'ClickHouse password variable');outputs.push(item);}
    if(selected.includes('controller')){if(!data.controller_token_file.startsWith('/'))throw new Error('Controller token file must be an absolute protected path');outputs.push({type:'controller',url:endpoint(data.controller_url,'Controller endpoint'),token_file:data.controller_token_file,node_id_file:data.controller_node_id_file,routes:remoteRoutes,fallback_path:fallback});}
    return {outputs};
  }

  function normalizeStreamFile(file={}) { const input=Number(sourceValue(file,'read_lines_per_second')||0),normal=Number(sourceValue(file,'normalized_per_second')||0),quarantine=Number(sourceValue(file,'quarantined_per_second')||0),thresholds=file.thresholds||state.sync.snapshot?.diagnostics?.thresholds||{};return {...file,bytes_per_second:sourceValue(file,'bytes_per_second'),queue_depth:sourceValue(file,'nli_queue_depth')??sourceValue(file,'queue_depth'),queue_capacity:sourceValue(file,'queue_capacity'),backpressure:sourceValue(file,'backpressure'),updated_at:file.controller_received_at||file.updated_at,input_rate:input,normal_rate:normal,quarantine_rate:quarantine,output_rate:normal+quarantine,thresholds:{read_lines_per_second:Number(thresholds.read_lines_per_second||0),bytes_per_second:Number(thresholds.bytes_per_second||0),quarantined_per_second:Number(thresholds.quarantined_per_second||0),queue_depth:Number(thresholds.queue_depth||0)}}; }
  function renderStreamRows(files) { return table(['Node / source','Input lines/s','Output lines/s','Normalized / quarantined','Read / processed total','Parsed / quarantine total','Errors','Bytes/s','Queue / capacity','Threshold crossings','Last updated','State'],files.map(raw=>{const file=normalizeStreamFile(raw),processed=present(file.normalized)||present(file.quarantined)?Number(file.normalized||0)+Number(file.quarantined||0):null,crossings=[['read lines/s',file.input_rate,file.thresholds.read_lines_per_second],['bytes/s',Number(file.bytes_per_second||0),file.thresholds.bytes_per_second],['quarantine/s',file.quarantine_rate,file.thresholds.quarantined_per_second],['queue depth',Number(file.queue_depth||0),file.thresholds.queue_depth]].filter(([,value,limit])=>limit>0&&value>=limit);return `<tr><td>${esc(file.node_id||'—')}<br><small>${esc(file.source||'—')}</small></td><td>${file.input_rate.toFixed(1)}</td><td>${file.output_rate.toFixed(1)}</td><td>${file.normal_rate.toFixed(1)} / ${file.quarantine_rate.toFixed(1)}</td><td>${metricText(file.read_lines)} / ${metricText(processed)}</td><td>${metricText(file.normalized)} / ${metricText(file.quarantined)}</td><td>${metricText(file.errors)}</td><td>${number(file.bytes_per_second)}</td><td>${number(file.queue_depth)} / ${number(file.queue_capacity)}</td><td>${file.backpressure?badge('backpressure'):crossings.length?`${badge('high volume')}<br>${crossings.map(([name,value,limit])=>`${esc(name)} ${number(value)} / ${number(limit)}`).join('<br>')}`: '—'}</td><td>${time(file.updated_at)}</td><td>${badge(file.phase||'unknown')}</td></tr>`;}),'No active per-node file streams are reported.'); }
  async function loadStreams() { let files=state.sync.snapshot?.sources||state.streamFiles||[];if(!state.sync.snapshot&&!files.length){const result=await Promise.allSettled([api('/telemetry/stream/snapshot')]).then(([snapshot])=>snapshot);files=result.status==='fulfilled'?(result.value.files||[]):state.dashboard?.stream_files||[];}state.streamFiles=files;renderPage('streams',`<div class="dashboard-intro"><div><p class="eyebrow">NURSF canonical source state</p><h1>Streams</h1><p>Live input, normalized, quarantine, byte-rate, queue, EOF, tailing, and freshness per node and source file.</p></div><span id="stream-transport" class="live-pill">${esc(state.streamMode)} · revision ${number(state.sync.snapshot?.revision)}</span></div>${renderRecordAccountingHelp()}${panel('Live file streams','One authenticated incremental stream updates the shared store used by every dashboard.',`<section id="stream-live-table">${renderStreamRows(files)}</section>`)}${diagnosticsPanel('source')}`); }

  function renderRecordAccountingHelp() {
    return `<details class="panel record-accounting-help"><summary>Record counts, hourly files and delivery acknowledgements</summary><p><b>File size is not a record count.</b> Values such as 48K and 738 from <code>ls -lh</code> are bytes. Hourly JSONL files append earlier records; installing an RPM does not clear that history.</p><p>Streams reports physical input lines and logical parsing outcomes for the reported source/process generation. Blank lines, filters and multiline framing can make physical lines differ from parsed plus quarantined records. A parsing outcome does not itself prove that Loki acknowledged delivery.</p><p>For a controlled 49-record, single-line test after EOF and output drain, compare newly appended normalised and quarantine JSON objects: <b>49 = normalised + quarantined + intentionally filtered</b>. Include replay or multi-destination copies only when they belong to that test scope. Keep each destination separate.</p><p>The release includes the read-only <code>nli-output-audit.py</code> helper. Capture a <code>--snapshot</code> before testing, then use <code>--baseline</code> and <code>--expected-records 49</code> with the same file selection afterward. It reports record counts and detects an edited, truncated or incomplete baseline without deleting logs. See the R8 record-accounting guide for commands.</p></details>`;
  }

  async function loadMetadata() {
    const data=await api('/metadata?limit=200'); const records=data.records||[], stats=data.stats||{};
    renderPage('metadata', `<div class="grid cols-4">${card('Retained metadata',number(stats.retained_metadata),'Bounded controller index')}${card('Last hour',number(stats.events_last_hour),'Authenticated telemetry')}${card('Parser attempts',number(stats.parser_attempts),'Aggregated attempts')}${card('Raw retention',data.payload_retained?'ENABLED':'OFF','Normal payloads never retained')}</div>${panel('Live metadata stream','WebSocket messages contain hashes and parser metadata only.',`<div class="callout">Connection state is shown in the header. Snapshot reconciliation runs every 30 seconds.</div><div id="live-records">${renderMetadataRows([...state.live,...records].slice(0,200))}</div>`)}`);
  }
  function renderMetadataRows(records){return table(['Time','Node','Service','Parser','Route','Attempts','Fields','Hash'],records.map(item=>{const r=item.record||item;return `<tr><td>${time(r.received_at||r.timestamp)}</td><td class="mono">${esc(r.node_id||'—')}</td><td>${esc(r.service||'—')}</td><td class="mono">${esc(r.parser||'—')}</td><td>${badge(r.route)}</td><td>${r.attempt_count??'—'}</td><td>${r.field_count??'—'}</td><td class="mono">${esc((r.event_hash||'').slice(0,16)||'—')}</td></tr>`}))}

  async function loadFieldIntegrity() {
    const report=await api('/verification/sample-corpus'),totals=report.totals||{},keys=report.known_key_inventory||{},families=report.families||[];
    const quarantined=(report.records||[]).filter(item=>item.expected_disposition==='quarantined');
    renderPage('field-integrity', `<div class="callout">This is deterministic evidence for the supplied 728-record corpus. It does not claim that arbitrary unknown logs can be interpreted without ambiguity.</div><div class="grid cols-4">${card('Records accounted',`${number(totals.accounted_records)} / ${number(totals.records)}`,`${number(totals.failing_records)} unaccounted`)}${card('Exact field parity',`${number(totals.exact_parity_records)} / ${number(totals.parsed_records)}`,`${number(totals.parsed_comparable_raw_fields)} raw = ${number(totals.parsed_generated_json_fields)} JSON`)}${card('Known keys covered',`${number(keys.covered_family_keys)} / ${number(keys.declared_family_keys)}`,`${number(keys.unmapped_family_keys)} unmapped`)}${card('Expected quarantine',number(totals.expected_quarantined_records),`${number(totals.unexpected_quarantined_records)} unexpected`)}</div>${panel('Family field-accounting proof','Counts are derived independently from fixed layouts and reviewed top-level key/value syntax, then compared with serialized JSON members.',table(['Family','Records','Passing','Raw fields','JSON fields','Duplicate occurrences'],families.map(item=>`<tr><td><b>${esc(item.family)}</b></td><td>${number(item.records)}</td><td>${number(item.passing_records)}</td><td>${number(item.comparable_raw_fields)}</td><td>${number(item.generated_json_fields)}</td><td>${number(item.duplicate_occurrences)}</td></tr>`)))}${panel('Fail-closed malformed specimens','These immutable records are intentionally not assigned fabricated values. Each is preserved and classified for quarantine.',table(['Corpus','Line','Family','Reason','SHA-256'],quarantined.map(item=>`<tr><td>${esc(item.corpus)}</td><td>${item.source_line}</td><td>${esc(item.family)}</td><td>${esc(item.disposition_reason)}</td><td class="mono">${esc((item.record_sha256||'').slice(0,20))}…</td></tr>`)))}`);
  }

  async function loadUnknowns() {
    const results=await Promise.allSettled([api('/quarantine/summary'),api('/quarantine/records?limit=250'),api('/quarantine/history?limit=100')]);
    const summary=results[0].status==='fulfilled'?results[0].value:{}, records=results[1].status==='fulfilled'?(results[1].value.records||[]):[], history=results[2].status==='fulfilled'?(results[2].value.events||results[2].value.history||[]):[];
    const failures=results.map((result,index)=>result.status==='rejected'?`<div class="callout warn">${['Summary','Record queue','Reprocessing history'][index]} unavailable: ${esc(result.reason.message)}</div>`:'').join('');
    const reasons=new Map();records.forEach(item=>(item.error_codes||[item.error||'no_match']).forEach(code=>reasons.set(code,(reasons.get(code)||0)+1)));
    const diagnosis=reasons.size?[...reasons].sort((a,b)=>b[1]-a[1]).map(([code,count])=>`<tr><td><code>${esc(code)}</code></td><td>${count}</td><td>${code==='no_match'?'No eligible parser matched; open representative raw lines in the Quarantine Resolution Wizard.':code==='malformed_record'?'Input failed structural validation; confirm the source format and required fields in the Quarantine Resolution Wizard.':'Review the exact source line and parser outcome in the Quarantine Resolution Wizard.'}</td></tr>`):[];
    const needsAttention=records.filter(quarantineNeedsAttention).length;
    renderPage('unknowns', `<div class="dashboard-intro"><div><p class="eyebrow">Runtime · retained source queue</p><h1>Quarantine and reprocessing</h1><p>Review unmatched records, preserve exact input, then open the self-contained Quarantine Resolution Wizard to build, validate, approve, deploy, and replay the parser.</p></div><div class="actions"><button class="secondary" data-action="quarantine-export" data-format="raw">Download raw lines</button><button class="secondary" data-action="quarantine-export" data-format="jsonl">Download quarantine JSONL</button><button class="secondary" data-action="navigate" data-page="quarantine-wizard">Open Resolution Wizard</button></div></div><div class="grid cols-4">${card('Waiting',number(summary.waiting??summary.quarantined??summary.total_pending),'Retained records awaiting parser review')}${card('Reprocessing',number(summary.reprocessing),'Replay accepted by controller')}${card('Resolved',number(summary.resolved??summary.reprocessed),'Successfully parsed after replay')}${card('Manual attention',number(needsAttention),'Replay failed or reached its retry limit')}</div>${failures}${panel('Diagnosis by parser outcome','Common failure codes with a suggested next review step.',table(['Reason code','Records','Suggested review'],diagnosis,'No failure codes are currently reported.'))}${panel('Queue and exact source access','Record metadata is safe to list. Exact raw download is separately permission checked, audited, and available only when retention was enabled for that record.',table(['First seen','Client / service','Record','Attempts / parser','State / reason','Source / actions'],records.map(item=>`<tr><td>${time(item.first_seen||item.created_at)}</td><td><b>${esc(item.node_id||'—')}</b><br><small>${esc(item.service||'unknown service')}</small></td><td class="mono">${esc(item.id||item.record_id||'—')}<br><small>${esc((item.hash||item.sha256||'').slice(0,18))}</small></td><td>${quarantineAttemptLabel(item)}<br><small>${esc(item.parser_revision||'no parser revision')}</small></td><td>${badge(quarantineNeedsAttention(item)?'manual_attention':quarantineRecordStatus(item))}<br><small>${esc((item.error_codes||[]).join(', ')||item.error||'No parser matched')}</small></td><td>${item.raw_available?`<button class="secondary small" data-action="quarantine-raw" data-id="${esc(item.id||item.record_id)}">Download exact raw</button> ${quarantineCanInspect(item)?`<button class="secondary small" data-action="quarantine-diagnose" data-id="${esc(item.id||item.record_id)}" data-service="${esc(item.service||'auto')}">Open in Resolution Wizard</button>${mapSampleButton('Quarantine evidence','quarantine-diagnose',item)}`:''} <button class="danger small" data-action="quarantine-delete-raw" data-id="${esc(item.id||item.record_id)}">Delete retained evidence</button>`:'Evidence not retained'} ${item.status==='excluded'?`<button class="secondary small" data-action="quarantine-include" data-id="${esc(item.id||item.record_id)}">Return to queue</button>`:!['resolved','reprocessing','evidence_deleted','dead_letter','manual_attention'].includes(item.status)?`<button class="secondary small" data-action="quarantine-exclude" data-id="${esc(item.id||item.record_id)}">Exclude</button>`:''} ${!['resolved','reprocessing','excluded','evidence_deleted','dead_letter','manual_attention'].includes(item.status)&&Number(item.attempt_count)<5?`<button class="secondary small" data-action="quarantine-queue-replay" data-id="${esc(item.id||item.record_id)}" data-node="${esc(item.node_id||'')}">Prepare replay</button>`:''}</td></tr>`),'No quarantine records are currently available.'))}${panel('Retry records','Each replay is an explicit, auditable request. Replay attempts use bounded backoff and are counted separately from parser strategies.',`<form id="quarantine-reprocess-form" class="form-grid"><label class="field">Record IDs<input name="record_ids" required placeholder="Paste one or more IDs, comma separated"></label><label class="field">Target client ID<input name="node_id" required placeholder="Originating node ID"></label><button class="primary" type="submit">Request replay</button><span class="hint wide">For a quarantined record, the Resolution Wizard contains evidence inspection, editor, deterministic validation, approval, rollout acknowledgement, and replay in one governed workflow. Standalone Assurance remains available for independent fixtures.</span></form>`)}${panel('Reprocessing history','Replay requests and client outcomes, including failed attempts and records still waiting for check-in.',table(['Time','Record','Client','Action','State','Details'],history.map(item=>`<tr><td>${time(item.timestamp||item.created_at)}</td><td class="mono">${esc(item.record_id||item.id||'—')}</td><td>${esc(item.node_id||'—')}</td><td>${esc(item.action||item.event||'reprocess')}</td><td>${badge(item.status||item.outcome||'pending')}</td><td>${esc(item.message||item.error||item.parser_revision||'—')}</td></tr>`)))}<div class="callout">Only records with explicit raw retention can be downloaded or replayed from the original source. Records that were never retained remain metadata-only; the UI cannot reconstruct their exact input. Replay has a bounded budget. Inspect the record in the wizard for its exact replay status.</div>`);
  }
  async function loadAIPrivacy() {
    const snapshot = await api('/ai/privacy');
    const activity = snapshot.activity || {}, knowledge = snapshot.knowledge || {};
    const inventory = Object.entries(knowledge.inventory || {}).map(([name,item]) =>
      card(name.replaceAll('_',' '), item.available ? number(item.count) : 'Unavailable', item.truncated ? 'Bounded count' : 'Controller catalog'));
    const workspaces = (knowledge.workspaces || []).map(({id,guidance}) =>
      '<tr><th scope="row">' + esc(TITLES[id] || id) + '</th><td>' + esc(guidance) + '</td></tr>');
    const events = (activity.events || []).map(({time:at,route,disposition}) =>
      '<tr><td>' + time(at) + '</td><td>' + esc(route) + '</td><td>' + esc(disposition) + '</td></tr>');
    const counters = [
      card('Production evidence withheld', number(activity.production_withheld || 0), 'Native validation stays available'),
      card('Masked authoring', number(activity.masked_authoring || 0), 'Manual sample requests'),
      card('Sample-free requests', number(activity.sample_free || 0), 'Workspace guidance'),
    ];
    renderPage('ai-privacy',
      panel('External AI data policy','Live ingestion and retained production evidence stay on the controller. Mark copied production samples as production evidence.',
        '<div class="grid cols-3">' + counters.join('') + '</div>') +
      panel('Recent request decisions','Payloads are excluded from this activity view.',
        table(['Time','Route','Decision'],events,'No AI privacy decisions recorded.')) +
      panel('NLI workspace knowledge','Controller guidance and current catalog counts.',
        '<div class="grid cols-3">' + inventory.join('') + '</div>' +
        table(['Workspace','Guidance'],workspaces,'No workspace guidance available.')));
  }
  async function loadControllerGateway() {
    const [controllerResult, capabilityResult, usageResult] = await Promise.allSettled([
      api('/health'), api('/ai/capabilities'), api('/admin/ai-usage'),
    ]);
    const probe = result => result.status === 'fulfilled' ? result.value : null;
    const controller = probe(controllerResult), capability = probe(capabilityResult), usage = probe(usageResult);
    const probeState = result => result.reason?.status === 403 ? 'Access denied' : 'Unavailable';
    const value = count => count == null ? '—' : number(count);
    const aiStates = {
      not_configured: 'No AI provider enabled',
      configuration_unavailable: 'Provider configuration unavailable',
      default_unavailable: 'Default profile unavailable',
      credential_unavailable: 'Provider credential unavailable',
      configured_not_connection_verified: 'Ready for a connection test',
    };
    const routes = new Set([
      '/health', '/ai/capabilities', '/admin/ai-usage', '/admin/ai-provider',
      '/admin/ai-provider/test', '/ai/workspace', '/assurance/ai-proposal',
      '/field-mapping/ai-suggest', '/assurance/mapping-preview',
    ]);
    const recent = state.apiHistory.filter(item => routes.has(item.path)).slice(0, 40);
    const rows = recent.map(item => {
      const outcome = item.degraded ? 'Degraded' : item.status == null ? 'Network error'
        : item.status < 400 ? 'HTTP success' : item.status >= 500 ? 'Server error' : 'Rejected';
      return `<tr><td>${time(item.at)}</td><td>${badge(outcome)}</td><td><code>${esc(item.method)} ${esc(item.path)}</code></td><td>${item.status??'—'}</td><td>${value(item.latency)} ms</td><td>${esc(item.error||'—')}</td></tr>`;
    });
    const heading = '<div class="dashboard-intro"><div><p class="eyebrow">Controller management · diagnostics</p><h1>Controller Gateway</h1><p>Check controller reachability, AI configuration, and recent mapping requests independently.</p></div><div class="actions"><button class="secondary" data-action="reload">Refresh gateway status</button></div></div>';
    const controllerCard = card('Controller', controller?.status || probeState(controllerResult), controller ? [controller.service, controller.version].filter(Boolean).join(' · ') : 'Controller health probe failed');
    const storageCard = card('Metadata storage', controller?.storage || '—', controller?.storage_diagnostic || 'Reported by controller health');
    const aiCard = card('AI configuration', capability ? aiStates[capability.status] || 'Unknown' : probeState(capabilityResult), capability ? `${value(capability.credential_ready_profiles)} credential-ready · ${value(capability.enabled_profiles)} enabled` : 'AI capabilities probe failed');
    const routingCard = card('AI routing', capability ? (capability.default_profile_id ? 'Pinned profile' : 'Configured priority') : '—', capability?.default_profile_id || 'Routing follows enabled profile order');
    const diagnostics = `<p>${esc(capability?.connection_check || 'Refresh gateway status to check AI readiness.')}</p>
      <p class="muted">Maximum parallel reviews ${value(capability?.max_parallel_reviews)} · provider attempts per request ${value(capability?.max_provider_attempts)} · controller request deadline ${capability?.request_deadline_seconds == null ? '—' : value(capability.request_deadline_seconds)+' s'}</p>`;
    const usageCards = `<div class="grid cols-4">${card('AI requests',value(usage?.requests),'Completed requests recorded by the controller')}${card('Successful',value(usage?.successes),'Controller-reported AI successes')}${card('Failed',value(usage?.failures),'Controller-reported AI failures')}${card('Mean latency',usage?.latency_ms_avg == null ? '—' : value(usage.latency_ms_avg)+' ms','Controller-reported AI request duration')}</div>`;
    const commands = '<pre>sudo systemctl status nli-controller-agent\nsudo journalctl -u nli-controller-agent -n 100 --no-pager\nsudo systemctl restart nli-controller-agent</pre><p class="muted">The gateway runs inside the controller service. Restarting that service briefly interrupts controller requests.</p>';
    renderPage('controller-gateway', heading +
      `<div class="grid cols-4">${controllerCard}${storageCard}${aiCard}${routingCard}</div>` +
      panel('AI Gateway status', 'Provider configuration and activity reported by the controller.', diagnostics + usageCards,
        '<button class="secondary" data-action="navigate" data-page="ai">Open provider settings</button>') +
      panel('Recent gateway requests', 'Latest matching API requests observed in this browser. The log contains routes, timing and HTTP outcomes.', table(['Time','Outcome','Request','HTTP','Duration','Details'],rows,'No gateway requests have been recorded in this browser.'),
        '<button class="secondary" data-action="navigate" data-page="api-monitor">Open API Monitor</button>') +
      panel('Controller service diagnostics', 'Run these commands on the controller node with a privileged account.', commands));
  }

  function loadLatency() {
    const apiRows = state.apiHistory.slice().sort((left, right) => right.latency - left.latency).slice(0, 25);
    const pageRows = state.navigationHistory || [];
    renderPage('latency',
      panel('Workspace loading','Elapsed time from navigation through the workspace loader, for this browser session.',table(['Workspace','Elapsed','Time'],pageRows.map(item=>`<tr><td>${esc(TITLES[item.page] || item.page)}</td><td>${item.latency} ms</td><td>${esc(item.at)}</td></tr>`))) +
      panel('Slowest observed requests','Includes network time and response decoding. Open API Monitor to run fresh probes.',table(['Method','Route','Elapsed','Result','Time'],apiRows.map(item=>`<tr><td>${esc(item.method)}</td><td>${esc(item.path)}</td><td>${item.latency} ms</td><td>${esc(item.error || item.status || 'Unavailable')}</td><td>${esc(item.at)}</td></tr>`))));
  }

  async function loadAPIMonitor() {
    const probes=['/health','/auth/me','/overview','/telemetry/stats','/telemetry/query?limit=1','/metrics/timeseries?window=1h','/fleet','/fleet/sync','/parser-sync','/metadata?limit=1','/parsers','/parser-packs','/assurance/cases','/assurance/parsers','/assurance/plans','/assurance/runs','/assurance/history','/quarantine/summary','/quarantine/records?limit=1','/quarantine/history?limit=1','/migration/reports','/artifacts','/rollouts','/rbac/roles','/users','/audit?limit=1','/system','/admin/ai-provider'];
    await Promise.allSettled(probes.map(path=>api(path)));
    const purposes={'/sync/snapshot':'Canonical NURSF state and freshness','/health':'Controller liveness','/auth/me':'Current authenticated operator and role','/overview':'Fleet, storage and operating summary','/telemetry/stats':'Ingest and processing counters','/telemetry/query':'Filtered processing records','/metrics/timeseries':'Aggregated processing and latency metrics','/fleet':'Registered nodes','/fleet/sync':'Registered clients, health and usage','/parser-sync':'Desired/applied parser revisions and drift','/metadata':'Bounded activity events','/parsers':'Built-in runtime parser inventory','/parser-packs':'Parser drafts, review and publication','/assurance/cases':'Retained validation fixtures','/assurance/parsers':'Assurance parser drafts','/assurance/plans':'Versioned Assurance test plans','/assurance/runs':'Persisted Assurance run evidence','/assurance/history':'Validation and parser review history','/quarantine/summary':'Waiting, replay and resolution counts','/quarantine/records':'Quarantine queue metadata','/quarantine/history':'Replay request and client outcomes','/migration/reports':'Parser source import reports','/artifacts':'Signed release artifacts','/rollouts':'Approval-gated actions and client delivery','/rbac/roles':'Role permission matrix','/users':'Operator accounts','/audit':'Immutable controller audit events','/system':'Controller version, resources and security','/admin/ai-provider':'Optional AI provider status'};
    const latest=new Map();state.apiHistory.forEach(item=>{if(!latest.has(`${item.method} ${item.path}`))latest.set(`${item.method} ${item.path}`,item);});
    const reachable=status=>status!=null&&(status<400||[401,403,405,422].includes(status));
    const latencies=[...latest.values()].map(item=>item.latency).sort((a,b)=>a-b),percentile=p=>latencies.length?latencies[Math.min(latencies.length-1,Math.ceil(p*latencies.length)-1)]:0,p50=percentile(.5),p95=percentile(.95);
    const rows=[...latest.values()].map(item=>`<tr><td>${badge(item.degraded?'degraded':reachable(item.status)?'online':'offline')}</td><td><code>${esc(item.method)}</code></td><td class="mono">${esc(item.path)}</td><td>${esc(purposes[item.path]||'Authenticated controller API')}</td><td>${item.latency} ms</td><td>${item.status??'network error'}${item.error?`<br><small>${esc(item.error)}</small>`:''}</td><td>${time(item.at)}</td></tr>`);
    const ok=[...latest.values()].filter(item=>reachable(item.status)).length, failed=[...latest.values()].length-ok;
    renderPage('api-monitor',`<div class="dashboard-intro"><div><p class="eyebrow">Controller connectivity · authenticated probes</p><h1>API monitor</h1><p>Each status comes from an actual browser request. A 401/403 means the API is reachable and access was denied; network and server errors show as offline.</p></div><div class="actions"><button class="secondary" data-action="reload">Refresh probes</button></div></div><div class="grid cols-4">${card('Reachable APIs',number(ok),'Successful or permission-gated HTTP responses')}${card('Failed APIs',number(failed),'Network or server errors in latest calls')}${card('Latency p50',`${p50} ms`,'Latest response per route')}${card('Latency p95',`${p95} ms`,'Latest response per route')}</div>${panel('Current controller API status','Method, route, purpose, response latency and result for live API calls observed in this session.',table(['Status','Method','Path','Purpose','Latency','HTTP result','Last checked'],rows,'No API calls observed yet. Refresh probes to check controller connectivity.'))}${panel('Probe coverage','The monitor checks controller health, storage, telemetry, client sync, parser sync, metadata, Assurance history, parser catalog, rollouts and system status.','<p class="muted">The request history is held in this browser session and contains routes and status metadata only. Request bodies, credentials and log payloads are not recorded.</p>')}`);
  }

  async function loadDrift() {
    const [syncResult,historyResult]=await Promise.allSettled([api('/parser-sync'),api('/assurance/history')]);
    const syncData=syncResult.status==='fulfilled'?syncResult.value:{},assets=syncData.assets||[],events=historyResult.status==='fulfilled'?(historyResult.value.events||[]):[];
    const clients=(syncData.clients||[]).map(client=>({...client,asset_name:'Active parser bundle'}));
    const drifted=clients.filter(client=>client.drifted||client.status==='drifted');
    renderPage('drift', `<div class="grid cols-4">${card('Published assets',number(assets.length),'Immutable desired revisions')}${card('Client observations',number(clients.length),'Reported active hashes')}${card('Drifted clients',number(drifted.length),drifted.length?'Review required':'All reported hashes aligned')}${card('Review events',number(events.length),'Assurance history')}</div>${panel('Template and parser drift','Desired controller hashes compared with independently reported active client hashes.',table(['Asset','Client','Status','Desired hash','Applied hash','Last seen'],clients.map(item=>`<tr><td>${esc(item.asset_name)}</td><td class="mono">${esc(item.node_id)}</td><td>${badge(item.drifted?'drifted':item.status||'aligned')}</td><td class="mono">${esc((item.desired_hash||'').slice(0,16)||'—')}</td><td class="mono">${esc((item.applied_hash||'').slice(0,16)||'—')}</td><td>${time(item.last_seen)}</td></tr>`)))}${panel('Recent parser review history','Changes become eligible for rollout only after validation and independent approval.',table(['Time','Action','Parser','Actor','Outcome'],events.map(item=>`<tr><td>${time(item.timestamp||item.created_at)}</td><td>${esc(item.action||item.event)}</td><td class="mono">${esc(item.parser_id||item.resource||'—')}</td><td>${esc(item.actor||'—')}</td><td>${badge(item.outcome||item.state||'recorded')}</td></tr>`)))}`);
  }

  async function loadPerformance() {
    if(state.sync.snapshot){
      const snapshot=state.sync.snapshot,aggregate=snapshot.aggregate||{},files=snapshot.sources||[],historyResult=await Promise.allSettled([api('/metrics/timeseries?range=24h&step=5m')]),persistedPoints=historyResult[0].status==='fulfilled'?(historyResult[0].value.points||[]):[],points=persistedPoints.length?persistedPoints:state.sync.history;
      const sourceRows=files.map(item=>{const file=normalizeStreamFile(item);return `<tr><td>${esc(file.node_id||'—')}</td><td class="mono">${esc(file.source||'—')}</td><td>${badge(file.phase||'unknown')}</td><td>${file.input_rate.toFixed(1)}</td><td>${file.output_rate.toFixed(1)}</td><td>${metricText(file.queue_depth)}</td><td>${syncStatusBadge(file.freshness)}</td><td>${esc(file.freshness_reason||'Within target')}</td></tr>`;});
      const maxBuckets=120,groupSize=Math.max(1,Math.ceil(points.length/maxBuckets)),buckets=[];for(let i=0;i<points.length;i+=groupSize){const group=points.slice(i,i+groupSize),last=group.at(-1);buckets.push({timestamp:last.timestamp||last.time,count:group.length,received:group.reduce((sum,p)=>sum+Number(p.received??p.logs_received??0),0),parsed:group.reduce((sum,p)=>sum+Number(p.parsed??p.logs_parsed??0),0),quarantine:group.reduce((sum,p)=>sum+Number(p.quarantine??p.logs_quarantined??0),0),errors:group.reduce((sum,p)=>sum+Number(p.errors||0),0)});}
      const historyChart=multiSeriesChart(buckets.map(item=>new Date(item.timestamp).toLocaleString()),[{label:'Input',values:buckets.map(item=>item.received),color:'#53a5ff',unit:'records / bucket'},{label:'Parsed',values:buckets.map(item=>item.parsed),color:'#3dd598',unit:'records / bucket'},{label:'Quarantine',values:buckets.map(item=>item.quarantine),color:'#f8c15c',unit:'records / bucket'},{label:'Errors',values:buckets.map(item=>item.errors),color:'#ff6b7d',unit:'records / bucket'}]);
      renderPage('performance',`${panel('Canonical throughput and latency','Current shared counters and measured propagation delay.',`<div class="dashboard-kpi-strip">${subsystemMetric('Read rate',aggregate.read_lines_per_second,'Aggregate input',' lines/s')}${subsystemMetric('Output rate',aggregate.normalized_per_second,'Normalized output',' lines/s','green')}${subsystemMetric('Quarantine rate',aggregate.quarantined_per_second,'Awaiting resolution',' lines/s',Number(aggregate.quarantined_per_second)?'amber':'green')}${subsystemMetric('Queue depth',aggregate.nli_queue_depth,'Current bounded gauge')}${subsystemMetric('Propagation p95',snapshot.diagnostics?.propagation_delay_ms_p95??snapshot.diagnostics?.p95_ms,'Measured sync path',' ms','cyan')}${subsystemMetric('Browser apply',state.sync.browserDelayMs,'Event to shared store',' ms','violet')}</div>`)}${panel('Per-source processing and freshness','Exact source state from the same canonical revision used by Command Center.',table(['Node','Source','Phase','Read/s','Output/s','Queue','Freshness','Reason'],sourceRows,'No source measurements are available.'))}${panel('Overall Timeline History',`Persisted controller time-series measurements, grouped into at most ${maxBuckets} buckets. Hover or focus a point for exact bucket totals; counts are records per bucket and current throughput uses records per second.`,buckets.length?`${historyChart}<p class="hint">${number(points.length)} persisted measurements · ${number(buckets.length)} displayed buckets · up to ${number(groupSize)} measurements per bucket.${historyResult[0].status==='rejected'?' Live canonical revision history is shown because persisted history is unavailable.':''}</p>`:empty('No persisted canonical history is available yet.'))}${diagnosticsPanel()}`);
      return;
    }
    const [statsResult,metricsResult,systemResult,snapshotResult]=await Promise.allSettled([api('/telemetry/stats'),api('/metrics/timeseries?range=1h&step=5m'),api('/system'),api('/telemetry/stream/snapshot')]);
    const stats=statsResult.status==='fulfilled'?statsResult.value:{},metrics=metricsResult.status==='fulfilled'?metricsResult.value:{},system=systemResult.status==='fulfilled'?systemResult.value:{},resources=system.system||system.resources||{},files=snapshotResult.status==='fulfilled'?(snapshotResult.value.files||[]):[],points=metrics.points||[];
    renderPage('performance',`${panel('Per-file processing durations','Read, parse, output and total time from live file stream snapshots.',table(['Node','Source','Read','Parse','Output','Total','Lines read'],files.map(item=>`<tr><td>${esc(item.node_id||'—')}</td><td>${esc(item.source||'—')}</td><td>${(Number(item.read_duration_ns||0)/1e6).toFixed(2)} ms</td><td>${(Number(item.parse_duration_ns||0)/1e6).toFixed(2)} ms</td><td>${(Number(item.output_duration_ns||0)/1e6).toFixed(2)} ms</td><td>${(Number(item.total_duration_ns||0)/1e6).toFixed(2)} ms</td><td>${number(item.read_lines)}</td></tr>`),'No per-file timings are reported.'))}${panel('Throughput, latency and backpressure','Live five-minute buckets from the controller.',table(['Bucket','Received','Parsed','Quarantine','Errors','Latency ms'],points.map(item=>`<tr><td>${time(item.timestamp||item.time)}</td><td>${number(item.received)}</td><td>${number(item.parsed)}</td><td>${number(item.quarantine)}</td><td>${number(item.errors)}</td><td>${Number(item.latency||0).toFixed(2)}</td></tr>`)))}${panel('Effective runtime resources','Controller-reported process state.',`<pre class="json-output">${esc(pretty(resources))}</pre>`)}${[statsResult,metricsResult,systemResult].some(item=>item.status==='rejected')?'<div class="callout warn">One or more live performance sources are unavailable.</div>':''}`);
  }
  function staleSnapshotWarning(data = {}) {
    if (!data?.snapshot_stale) return '';
    const when=data.snapshot_at ? ' from ' + time(data.snapshot_at) : '';
    return '<div class="callout warn" role="status">Showing the last confirmed snapshot' + esc(when) +
      '. Live metadata is temporarily unavailable. Refresh to resume actions.</div>';
  }
  async function loadRollouts() {
    const [rolloutResult,syncResult]=await Promise.allSettled([api('/rollouts',{allowStale:true}),api('/parser-sync')]);
    if(rolloutResult.status==='rejected')throw rolloutResult.reason;
    const data=rolloutResult.value,stale=data.snapshot_stale===true,proposals=data.proposals||[],live=data.live||[],syncBundle=syncResult.status==='fulfilled'?syncResult.value.bundle||null:null,currentBundleHash=syncBundle?.desired_hash||'';
    state.parserRolloutProposals=proposals;state.parserLiveRollouts=live;state.assuranceSyncBundle=stale?null:syncBundle;
    state.parserRolloutLoadError=stale?'Live rollout metadata is unavailable.':'';
    const syncWarning=syncResult.status==='rejected'?`<div class="callout warn">Current parser bundle integrity could not be verified: ${esc(syncResult.reason.message)}. Parser-sync approval remains blocked until refresh; other rollout types remain reviewable.</div>`:'';
    const proposalRows=proposals.map(doc=>{const p=doc.payload||{},hash=rolloutBundleHash(doc),parserSync=isParserSyncProposal(doc),verifiedCurrentHash=parserSync&&syncResult.status==='rejected'?null:parserSync?currentBundleHash:'';return `<tr><td><b>${esc(p.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small>${managedFileReview(p)}</td><td>${esc(p.type||'unknown')}</td><td>${badge(doc.state)}</td><td>${esc(doc.created_by||'—')}</td><td>${esc(p.approved_by||'—')}</td><td class="mono">${esc(p.live_rollout_id||'—')}</td><td>${parserSync?`<span class="mono" title="${esc(hash)}">${esc(hash.slice(0,20)||'unavailable')}</span><br><small>${/^[0-9a-f]{64}$/i.test(hash)?hash===currentBundleHash?'Current exact bundle':'Older exact bundle':'Legacy snapshot · controller verification required'}</small>`:'—'}</td><td>${stale&&doc.state==='pending_approval'?'Refresh required':rolloutApprovalControls(doc,verifiedCurrentHash)}</td></tr>`;});
    renderPage('rollouts', `${panel('Approval-gated proposals','Record a review reason, verify the exact parser bundle snapshot, then approve. Proposal authors need a different reviewer unless an administrator uses the audited administrator approval.',staleSnapshotWarning(data)+syncWarning+table(['Proposal','Type','State','Creator','Approver','Live rollout','Bundle integrity','Actions'],proposalRows))}${panel('Create rollout proposal','No rollout launches without independent approval.',`<form id="rollout-form" class="form-grid"><label class="field">Name<input name="name" required></label><label class="field">Type<select name="type"><option value="config_push">Config</option><option value="parser_sync">Parser pack</option><option value="restart">Restart</option><option value="replay">Replay</option></select></label><label class="field wide">Target node IDs, comma separated<input name="targets"></label><label class="field wide">Payload JSON<textarea name="payload">{}</textarea></label><button class="primary" type="submit" ${stale?'disabled':''}>Request approval</button></form>`)}${panel('Live controller rollouts','Command leases and node results from the active rollout manager.',table(['ID','Name','Type','Status','Targets','Done','Failed'],live.map(item=>`<tr><td class="mono">${esc(item.id)}</td><td>${esc(item.name)}</td><td>${esc(item.type)}</td><td>${badge(item.status)}</td><td>${item.total_nodes||0}</td><td>${item.done_nodes||0}</td><td>${item.failed_nodes||0}</td></tr>`)))}`);
  }

  async function loadArtifacts() {
    const data=await api('/artifacts'); const items=data.artifacts||[];
    renderPage('artifacts', `${panel('Signed immutable artifacts','Ed25519 signature is verified over the SHA-256 digest before persistence. Private file storage accepts approved Rollout RPM and JSON configuration files only.',table(['Name','Type','Size','SHA-256','Signature','State','Uploaded'],items.map(doc=>{const p=doc.payload||{};return `<tr><td>${esc(p.name)}</td><td>${esc(p.type)}</td><td>${number(p.size)}</td><td class="mono">${esc((p.sha256||'').slice(0,20))}</td><td>${badge(p.signature_verified?'verified':'invalid')}</td><td>${badge(doc.state)}</td><td>${time(p.uploaded_at)}</td></tr>`})))}${panel('Upload signed Rollout file',`${data.signing_keys_configured||0} trusted signing keys configured.${state.proxy ? ' Hosted uploads support files up to 4 MB. Use the controller URL for larger RPMs.' : ''}`,`<form id="artifact-form" class="form-grid"><label class="field">Type<select name="type"><option value="rpm">RPM</option><option value="config">Configuration JSON</option></select></label><label class="field">Rollout file<input name="file" type="file" accept=".rpm,.json" required></label><label class="field wide">Ed25519 signature of SHA-256 digest, base64<input name="signature" required></label><button class="primary" type="submit">Verify and upload</button></form>`)}${panel('Upgrade proposal','A validated RPM creates an approval-gated rollout proposal.',`<form id="upgrade-form" class="form-grid"><label class="field">Artifact ID<input name="artifact_id" required></label><label class="field">Proposal name<input name="name"></label><label class="field wide">Target node IDs, comma separated<input name="targets"></label><button class="primary" type="submit">Request upgrade approval</button></form>`)}`);
  }

  async function loadSecurity() {
    const roles=await api('/rbac/roles'); const usersResult=await Promise.allSettled([api('/users'),api('/audit?limit=100')]); const users=usersResult[0].status==='fulfilled'?usersResult[0].value.users||[]:[];const audit=usersResult[1].status==='fulfilled'?usersResult[1].value:null;
    renderPage('security', `${panel('Five-role authorization','Permissions are evaluated explicitly, not by UI hiding.',table(['Role','Permissions'],Object.entries(roles.roles||{}).map(([role,permissions])=>`<tr><td>${badge(role)}</td><td class="mono">${esc((permissions||[]).join(' · '))}</td></tr>`)))}${panel('Users','Administrator-only account lifecycle.',usersResult[0].status==='rejected'?`<div class="callout warn">${esc(usersResult[0].reason.message)}</div>`:table(['Username','Email','Role','Status','Created','Last login','Actions'],users.map(user=>`<tr><td>${esc(user.username)}</td><td>${esc(user.email)}</td><td>${badge(user.role)}</td><td>${badge(user.active?'active':'inactive')}</td><td>${time(user.created_at)}</td><td>${time(user.last_login)}</td><td><button class="danger small" data-action="user-delete" data-id="${esc(user.username)}">Deactivate</button></td></tr>`))+`<form id="user-form" class="form-grid"><label class="field">Username<input name="username" required></label><label class="field">Email<input name="email" type="email" required></label><label class="field">Role<select name="role"><option>viewer</option><option>parser_author</option><option>reviewer</option><option>ops</option><option>admin</option></select></label><label class="field">Initial password<input name="password" type="password" minlength="12" required></label><button class="primary" type="submit">Create user</button></form>`)}${panel('Immutable audit chain','Authenticated changes with tamper-evident chain verification.',audit?`${badge(audit.verification)}${table(['Sequence','Time','Actor','Role','Action','Resource','Outcome'],(audit.events||[]).map(item=>`<tr><td>${item.sequence}</td><td>${time(item.timestamp)}</td><td>${esc(item.actor)}</td><td>${esc(item.role)}</td><td>${esc(item.action)}</td><td>${esc(item.resource)}</td><td>${badge(item.outcome)}</td></tr>`))}`:`<div class="callout warn">${esc(usersResult[1].reason?.message||'Audit access unavailable')}</div>`)}`);
  }

  async function loadCommunicationStatus() {
    const data=await api('/communications/status');
    const hopRows=(data.path||[]).map(hop=>{const label=({client_to_controller:'Client → Controller',controller_internal:'Controller internals',controller_to_webapp:'Controller → Webapp'})[hop.hop]||String(hop.hop||'').replaceAll('_',' ');return `<tr><td><b>${esc(label)}</b></td><td>${badge(hop.state||'unavailable')}</td><td>${time(hop.observed_at||hop.last_seen||data.generated_at)}</td><td>${esc(hop.evidence||'No evidence reported')}</td><td>${hop.clients?number(hop.client_count??hop.clients.length):hop.browser?`${esc(hop.browser.state||'unavailable')} · ${esc(hop.browser.status||'')}`:'—'}</td></tr>`;});
    const canonical=data.canonical||{};
    renderPage('communications',`${panel('End-to-end communication status','Live evidence from client → controller → webapp. Each hop reports its own observed state and timestamp.',table(['Communication hop','State','Observed','Evidence','Details'],hopRows,'No communication-path evidence has been reported yet.'))}${panel('Canonical synchronization source','Revision and freshness are from the same controller snapshot used by dashboards.',`<div class="grid cols-4">${card('Revision',number(canonical.revision),'Authoritative controller revision')}${card('Schema',canonical.schema_version||'—','Canonical contract')}${card('Target interval',present(canonical.target_interval_ms)?`${number(canonical.target_interval_ms)} ms`:'—','Configured cadence')}${card('Generated',time(canonical.generated_at),'Controller timestamp')}</div>`)}<div class="callout">Browser heartbeat is sent every five seconds while signed in. A missing hop stays unavailable until the controller reports fresh evidence.</div>`);
  }
  async function loadNotifications() {
    const [data,config]=await Promise.all([api('/notifications?limit=100'),api('/notifications/config')]),items=data.notifications||[],policy=config.policy||data.policy||{},channels=config.channels||{};
    state.controllerNotifications=items.slice(0,30);renderNotificationBell();
    const configured=policy.channels||{},email=configured.email||{},pagerduty=configured.pagerduty||{},events=['approval','quarantine','sync','system','delivery','security'],watched=new Set(policy.events||[]);
    const configForm=`<form id="notification-config-form" class="form-grid"><label class="field"><input name="email_enabled" type="checkbox" value="true" ${email.enabled?'checked':''}> Enable SMTP email</label><label class="field"><input name="pagerduty_enabled" type="checkbox" value="true" ${pagerduty.enabled?'checked':''}> Enable PagerDuty</label><label class="field wide">Recipient usernames, comma separated<input name="recipients" value="${esc((policy.recipients||[]).join(', '))}" placeholder="admin, oncall-operator"><span class="hint">Users are matched by username. Admin fallback is enabled separately.</span></label><label class="field"><input name="admin_fallback" type="checkbox" value="true" ${policy.admin_fallback!==false?'checked':''}> Send critical notifications to administrators</label><div class="field wide"><b>Watched event categories</b><div class="form-grid cols-3">${events.map(event=>`<label class="field"><input type="checkbox" name="events" value="${event}" ${watched.has(event)?'checked':''}> ${event}</label>`).join('')}</div></div><label class="field">SMTP server environment variable<input name="email_server_env" value="${esc(email.server_env||'NLI_SMTP_SERVER')}" autocomplete="off"><span class="hint">Value must be host:465; TLS is required.</span></label><label class="field">Email sender environment variable<input name="email_sender_env" value="${esc(email.sender_env||'NLI_NOTIFICATION_EMAIL_FROM')}" autocomplete="off"><span class="hint">Name only; set its value on the controller.</span></label><label class="field">Email recipient-list environment variable<input name="email_recipient_env" value="${esc(email.recipient_env||'NLI_NOTIFICATION_EMAIL_RECIPIENTS')}" autocomplete="off"><span class="hint">Comma separated email addresses in the environment value.</span></label><label class="field">SMTP username environment variable<input name="email_username_env" value="${esc(email.username_env||'NLI_SMTP_USERNAME')}" autocomplete="off"></label><label class="field">SMTP password environment variable<input name="email_password_env" value="${esc(email.password_env||'NLI_SMTP_PASSWORD')}" autocomplete="off"></label><label class="field">PagerDuty routing-key environment variable<input name="pagerduty_secret_env" value="${esc(pagerduty.secret_env||'NLI_PAGERDUTY_ROUTING_KEY')}" autocomplete="off"><span class="hint">Only the environment-variable name is stored.</span></label><button class="primary" type="submit">Save notification policy</button></form>`;
    const attempts=data.delivery_history?.attempts||[];
    renderPage('notifications',`${panel('Notification center',`${number(data.unread_count??data.count??0)} unread operator action(s). Read, snooze, and dismissal state is stored per user; immutable audit evidence is never deleted.`,table(['Time','Severity','Category','Event','Resource','Actor','Outcome','State','Actions'],items.map(item=>`<tr><td>${time(item.timestamp)}</td><td>${badge(item.severity)}</td><td>${esc(item.category||'—')}</td><td><b>${esc(item.title||'Notification')}</b><br><small>${esc(item.details||'')}</small></td><td>${esc(item.resource||'—')}</td><td>${esc(item.actor||'—')}</td><td>${badge(item.outcome||'recorded')}</td><td>${badge(item.status||(item.acknowledged?'read':'unread'))}</td><td class="notification-table-actions">${item.read||item.status==='read'?`<button class="ghost small" data-action="notification-state" data-value="unread" data-id="${esc(item.sequence)}">Mark unread</button>`:`<button class="secondary small" data-action="notification-state" data-value="read" data-id="${esc(item.sequence)}">Mark read</button>`} <button class="ghost small" data-action="notification-state" data-value="snooze" data-id="${esc(item.sequence)}">Check later</button> <button class="danger small" data-action="notification-dismiss" data-id="${esc(item.sequence)}">Delete</button></td></tr>`),'No unread or active controller notifications.') )}${panel('Delivery policy','Success is recorded only after SMTP acceptance or a PagerDuty success acknowledgement.',`<div class="grid cols-3">${card('SMTP email',channels.email?.status||channels.email?.state||'unavailable','Latest delivery state')}${card('PagerDuty',channels.pagerduty?.status||channels.pagerduty?.state||'unavailable','Latest delivery state')}${card('In-app',channels.in_app?.status||channels.in_app?.state||'unavailable','Audit-backed feed')}</div>${table(['Time','Audit sequence','Channel','Attempt','Status','Error'],attempts.map(a=>`<tr><td>${time(a.timestamp)}</td><td>${number(a.sequence)}</td><td>${esc(a.channel)}</td><td>${number(a.attempt)}</td><td>${badge(a.status)}</td><td>${esc(a.error||'—')}</td></tr>`),'No external delivery attempts have been recorded.')}${configForm}`)}<div class="callout">SMTP uses implicit TLS on port 465. PagerDuty uses Events API v2. Secret values stay in controller environment variables; the UI stores only their names.</div>`);
  }

  async function loadSystem() {
    const data=await api('/system');
    const metrics=data.infrastructure||data.system||data.resources||{};
    renderPage('system', `<div class="grid cols-4">${card('Controller',data.version,data.health)}${card('Storage',data.storage?.mode,data.storage?.status)}${card('Audit chain',data.security?.audit_chain,'HMAC chained')}${card('Fleet',number(data.fleet?.total),'registered nodes')}</div>${panel('Live controller resources','Process gauges are read from the controller health metrics endpoint.',`<div class="grid cols-4">${card('CPU cores',metrics.cpu_count??'—','Available logical processors')}${card('Memory',metrics.memory_bytes!=null?`${(Number(metrics.memory_bytes)/1048576).toFixed(0)} MB`:'—','Controller process memory')}${card('Goroutines',number(metrics.goroutines),'Concurrent runtime workers')}${card('Uptime',present(metrics.uptime_seconds)?`${Math.floor(Number(metrics.uptime_seconds))} s`:'—','Controller process')}</div><pre class="json-output">${esc(pretty(metrics))}</pre>`)}${panel('Client resource samples','Latest Linux host gauges from authenticated enrolled heartbeats.',table(['Node','CPU','Memory','Load 1 / 5 / 15','Updated'],(data.client_metrics||[]).map(item=>`<tr><td>${esc(item.hostname||item.node_id)}</td><td>${item.cpu_percent==null?'—':`${Number(item.cpu_percent).toFixed(1)}%`}</td><td>${item.memory_percent==null?'—':`${Number(item.memory_percent).toFixed(1)}%`}<br><small>${item.memory_bytes?`${(item.memory_bytes/1048576).toFixed(0)} MB`:''}</small></td><td>${esc(item.load_1m??'—')} / ${esc(item.load_5m??'—')} / ${esc(item.load_15m??'—')}</td><td>${time(item.updated_at)}</td></tr>`),'No authenticated client host sample has arrived yet.'))}${panel('Storage adapter','Credentials stay server-side.',`<pre class="json-output">${esc(pretty(data.storage))}</pre>`)}${panel('Retention boundary','Normal raw/event bodies are never retained. Assurance retention is separate.',`<pre class="json-output">${esc(pretty(data.retention))}</pre>`)}${panel('Security controls','Effective controller controls.',`<pre class="json-output">${esc(pretty(data.security))}</pre>`)}`);
  }

  async function loadParserWorkspace() {
    const [draftsResult,runtimeResult]=await Promise.allSettled([api('/assurance/parsers'),api('/parsers')]);
    const drafts=draftsResult.status==='fulfilled'?(draftsResult.value.parsers||[]):[],runtime=runtimeResult.status==='fulfilled'?(runtimeResult.value.parsers||[]):[];
    const rows=drafts.map(doc=>{const p=doc.payload||doc,editable=['draft','rejected'].includes(doc.state);return `<tr><td><b>${esc(p.name||doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${doc.revision??'—'}</td><td>${badge(doc.state)}</td><td>${esc(p.validation?.score??'—')}%</td><td><button class="secondary small" data-action="parser-workspace-load" data-id="${esc(doc.id)}" ${editable?'':'disabled'}>Edit</button> ${editable?`<button class="danger small" data-action="parser-workspace-delete" data-id="${esc(doc.id)}">Delete</button>`:''} ${editable?`<button class="secondary small" data-action="assurance-parser-action" data-id="${esc(doc.id)}" data-value="submit">Submit</button>`:''}</td></tr>`;});
    renderPage('parser-workspace',`<div class="callout">Build one complete, bounded parser artifact from single or multiline samples. Trailing whitespace is trimmed and the artifact exposes match scope, extraction, known fields, projection and output template. Saving never changes a live parser: independent review, approval, publication and fleet acknowledgement remain mandatory.</div>${panel('Complete parser artifact editor','Generate, edit and validate a declarative parser before saving a draft.',`<form id="parser-workspace-form" class="form-grid"><input type="hidden" name="edit_id"><label class="field wide">Single or multiline samples<textarea name="samples" required></textarea></label><div class="actions wide"><button type="button" class="secondary" data-action="parser-workspace-generate">Auto-populate complete parser</button><button type="button" class="secondary" data-action="parser-workspace-clear">New draft</button><button type="button" class="secondary" data-action="nfi-open-context">Map fields with NFI</button></div><label class="field">Parser name<input name="name" required maxlength="120" value="guided-parser-draft"></label><label class="field">Version<input name="version" required value="0.1.0"></label><div class="wide" data-parser-overview>Paste representative lines, then auto-populate the parser artifact.</div><label class="field wide">Complete parser artifact JSON<textarea name="parser" required>{}</textarea></label><label class="field wide">Validation cases JSON<textarea name="cases" required>[]</textarea></label><div class="actions wide"><button type="button" class="secondary" data-action="parser-workspace-validate">Validate locally</button><button type="submit" class="primary">Save approval-gated draft</button></div><div class="wide" data-parser-status></div></form>`)}${panel('Approval-gated parser drafts',`${runtime.length} live runtime parsers are available as reference. Drafts can be edited or removed only before review.`,table(['Parser','Revision','State','Validation','Actions'],rows,'No Parser Creation Workspace drafts exist yet.'))}${draftsResult.status==='rejected'?`<div class="callout warn">Draft catalog unavailable: ${esc(draftsResult.reason.message)}</div>`:''}`);
  }

  function renderAIProfileOrder(profiles,order,defaultID){
    const ranks=new Map(order.map((id,index)=>[id,index]));
    const eligible=profiles.filter(item=>item.enabled&&item.status!=='unreadable').sort((a,b)=>
      (ranks.get(a.id)??Infinity)-(ranks.get(b.id)??Infinity)||Number(a.priority||0)-Number(b.priority||0)||(a.id<b.id?-1:a.id>b.id?1:0));
    if(!eligible.length)return '<p class="hint">Save and enable at least one model to set its position in the routing order.</p>';
    return `<section class="panel ai-routing-order"><h4>Model priority order</h4><p>${defaultID?'The exclusive default is active. This order is saved for automatic routing.':'NLI starts with the first eligible profile. Fallbacks are tried in order within its configured group, up to three attempts.'}</p>
      <ol id="ai-route-order">${eligible.map(item=>`<li data-profile-id="${esc(item.id)}"><span><strong>${esc(item.name||item.id)}</strong> · ${esc(item.model||'Model unavailable')} · group ${esc(item.fallback_group||'default')}${item.id===defaultID?' · exclusive default':''}${item.credential_available===false?' · credential unavailable':''}</span><button type="button" class="secondary small" data-action="ai-order-up">Move up</button><button type="button" class="secondary small" data-action="ai-order-down">Move down</button></li>`).join('')}</ol>
      <div class="actions"><button type="button" class="primary" data-action="ai-order-save">Save model order</button><button type="button" class="secondary" data-action="ai-order-reset">Reset to numeric priorities</button></div></section>`;
  }

  function aiModelOptions(template){
    if(!template?.fill_model)return '';
    return [...new Set([template.model_hint,...(template.models||[])])].filter(Boolean)
      .map(model=>`<option value="${esc(model)}"></option>`).join('');
  }
  function aiPresetForEndpoint(endpoint){
    const base=String(endpoint||'').replace(/\/+$/,'').replace(/\/accounts\/[a-f0-9]{32}\//i,'/accounts/{account_id}/');
    return state.aiProviderTemplates?.find(item=>item.fill_model&&item.base_url===base);
  }
  async function loadAIProvider() {
    const [providerResult,proposalsResult]=await Promise.allSettled([api('/admin/ai-provider'+(state.aiSelectedProviderId?'?id='+encodeURIComponent(state.aiSelectedProviderId):'')),api('/assurance/ai-proposals')]);
    const data=providerResult.status==='fulfilled'?providerResult.value:{store_status:'unavailable',store_message:providerResult.reason.message};
    const degraded=data.store_status&&data.store_status!=='available';
    const storeNotice=degraded?`<div class="callout" role="status"><strong>Provider storage needs attention.</strong> ${esc(data.store_message||'Saved profiles could not be loaded.')} <button type="button" class="secondary" data-action="ai-profile-retry">Retry loading profiles</button></div>`:'';

    const provider = data.provider || data;
    const defaultProfileID=data.default_profile_id||'';
    const kinds=[['openai','OpenAI'],['nvidia_nim','NVIDIA NIM'],['anthropic','Anthropic / Claude'],['minimax','MiniMax'],['qwen_dashscope','Qwen / DashScope'],['gemini','Gemini / Google'],['openai_compatible','OpenAI-compatible endpoint'],['ollama','Ollama (controller-local)']];
    const modes=[['encrypted_inline','Encrypted inline (write-only)'],['env_ref','Environment variable reference'],['file_ref','Managed secret file reference'],['none','No credential (Ollama only)']];
    const credentialSource=provider.credential_source||'encrypted_inline',templates=data.templates||[];state.aiProviderTemplates=templates;const profiles=data.profiles||[],capabilities=data.capabilities||{};
    const templateRows=templates.map(item=>`<tr><td><b>${esc(item.name)}</b><br><small class="mono">${esc(item.kind)}</small></td><td class="mono">${esc(item.base_url)}</td><td>${esc(item.model_hint)}</td><td>${esc(item.credential_source)}</td></tr>`);
    state.aiProposals=proposalsResult.status==='fulfilled'?(proposalsResult.value.proposals||[]):[];
    renderPage('ai', `${panel('AI connection manager','Configure the controller-side provider used only for optional failure analysis and parser suggestions. Credentials are write-only, redacted from responses and never copied to clients.',`${storeNotice}<div class="callout">AI is outside the live parsing path. Provider unavailability cannot stop collection, deterministic parsing, quarantine, Assurance, or live synchronization. Public provider presets never contain credentials.</div><div class="actions"><label>Saved AI profiles<select id="ai-profile-choice"><option value="">Automatic priority selection</option>${profiles.map(item=>`<option value="${esc(item.id)}" ${item.id===provider.id?'selected':''}>${esc(item.name)} · ${esc(item.model)} · priority ${number(item.priority)}${item.id===defaultProfileID?' (default, exclusive)':''}</option>`).join('')}</select></label><button type="button" class="secondary" data-action="ai-profile-load">Load profile</button><button type="button" class="secondary" data-action="ai-profile-new">New model profile</button><button type="button" class="primary" data-action="ai-profile-set-default" ${profiles.length?'':'disabled'}>Set as Default Nexora AI Profile</button><button type="button" class="secondary" data-action="ai-profile-clear-default" ${defaultProfileID?'':'disabled'}>Use automatic routing</button></div><p role="status">${defaultProfileID?'NLI uses only profile <strong>'+esc(defaultProfileID)+'</strong> and its saved model. If unavailable, AI requests stop with an error.':'Automatic routing is active. NLI uses the saved model order or numeric priorities when no custom order is saved.'}</p>${renderAIProfileOrder(profiles,data.profile_order||[],defaultProfileID)}<form id="ai-provider-form" class="form-grid"><div class="actions wide"><label class="field">Nexora AI provider preset<select id="ai-provider-preset"><option value="">Select a provider</option>${templates.filter(item=>item.fill_model).map(item=>`<option value="${esc(item.id)}">${esc(item.name)}</option>`).join('')}</select></label><label class="field" id="ai-preset-account-field" hidden>Cloudflare account ID<input id="ai-preset-account" autocomplete="off" maxlength="32"></label><button type="button" class="secondary" data-action="ai-profile-from-preset">Start Nexora AI profile from preset</button></div><label class="field">Profile ID<input name="id" value="${esc(provider.id||'default')}" required pattern="[a-z][a-z0-9_-]{0,63}" maxlength="64"></label><label class="field">Provider<select name="provider">${kinds.map(([value,label])=>`<option value="${value}" ${provider.provider===value||provider.kind===value?'selected':''}>${esc(label)}</option>`).join('')}</select></label><label class="field">Display name<input name="name" value="${esc(provider.name||'Default AI provider')}" maxlength="128" required></label><label class="field">Model<input name="model" list="ai-model-options" value="${esc(provider.model||'')}" placeholder="Select a suggested model or type an exact model ID" maxlength="256" required></label><datalist id="ai-model-options">${aiModelOptions(aiPresetForEndpoint(provider.endpoint||provider.base_url))}</datalist><button type="button" class="secondary" data-action="ai-discover-models">Load models for saved profile</button><p id="ai-model-discovery-status" role="status"></p><label class="field">Priority<input name="priority" type="number" min="0" max="100000" value="${Number(provider.priority||0)}"></label><label class="field wide">API endpoint<input name="endpoint" type="url" value="${esc(provider.endpoint||'')}" placeholder="HTTPS base URL; loopback is limited to Ollama"></label><label class="field">Credential mode<select name="credential_source">${modes.map(([value,label])=>`<option value="${value}" ${credentialSource===value?'selected':''}>${esc(label)}</option>`).join('')}</select></label><label class="field">Request timeout (seconds)<input name="timeout_seconds" type="number" min="1" max="120" value="${Number(provider.timeout_seconds)||15}"></label><label class="field">Maximum response bytes<input name="max_response_bytes" type="number" min="1024" max="8388608" value="${Number(provider.max_response_bytes)||1048576}"></label><label class="field">Fallback group<input name="fallback_group" value="${esc(provider.fallback_group||'')}" maxlength="64" placeholder="Optional routing group"></label><label class="field wide">API key (encrypted inline)<input name="api_key" type="password" autocomplete="new-password" spellcheck="false" placeholder="${provider.configured ? `Configured${provider.key_last4?` · ending ${esc(provider.key_last4)}`:''}; leave blank to keep` : 'Paste the actual key only; no Bearer prefix or $ENV placeholder'}"></label><label class="field wide">Credential reference<input name="credential_ref" value="${esc(credentialSource==='file_ref'?'':provider.credential_ref||'')}" autocomplete="off" spellcheck="false" placeholder="Environment variable name, or managed file path; leave blank to retain this profile reference"></label><label class="field"><input name="enabled" type="checkbox" value="true" ${provider.enabled?'checked':''}> Enable optional AI assistance</label><div class="actions wide"><button class="primary" type="submit">Save provider settings</button><button class="secondary" type="button" data-action="ai-profile-preset">Use selected provider endpoint preset</button></div><details class="wide"><summary>Explicit connection-test options</summary><p class="hint">A test sends a small real request only when clicked. JSON is the default. SSE and an optional public image URL are supported for OpenAI-compatible adapters. Log remediation remains non-streaming. No image is fetched automatically by NLI.</p><label class="field">Probe response<select name="test_mode"><option value="json">JSON completion</option><option value="stream">Streaming SSE completion</option></select></label><label class="field">Probe token budget (16–1024)<input name="test_max_tokens" type="number" min="16" max="1024" value="64"></label><label class="field">Optional public HTTPS image URL<input name="test_image_url" type="url" placeholder="Public HTTPS image, no signed query or credentials"></label></details><div class="actions wide"><button class="secondary" type="button" data-action="test-ai-provider">Test connection</button></div><div id="ai-provider-test-status" class="wide json-output" role="status"></div><div class="actions wide"><button type="button" class="danger" data-action="ai-credential-remove" data-id="${esc(provider.id||'')}" ${profiles.some(item=>item.id===provider.id)?'':'disabled'}>Remove credential</button><button type="button" class="danger" data-action="ai-profile-delete" data-id="${esc(provider.id||'')}" ${profiles.some(item=>item.id===provider.id)?'':'disabled'}>Delete profile</button></div><p class="hint wide">Disabling retains credentials. Remove credential purges this profile’s stored credential history and retains disabled settings; Delete profile also removes its settings. External environment variables, files, backups and provider-issued keys require separate removal or revocation. Already outbound requests cannot be recalled.</p></form>`)}${panel('Connection status · AI feature readiness','Credential presence is separate from a successful live connection test. No secret is resolved in the browser.',`<pre class="json-output">${esc(pretty(capabilities))}</pre>`)}${panel('Connection status','Status is controller authoritative. Secret values, encrypted payloads and managed file paths are never returned.',`<div class="grid cols-4">${card('Configured',provider.configured?'Yes':'No',credentialSource)}${card('Credential',provider.key_last4?`•••• ${provider.key_last4}`:(provider.credential_ref||'Redacted'),'No secret value displayed')}${card('Routing',`Priority ${Number(provider.priority||0)}`,provider.fallback_group||'Default group')}${card('Status',provider.status||'Not tested',provider.last_test_at?time(provider.last_test_at):'Run a bounded connection test')}</div>`)}${panel('Provider templates','Public presets supply endpoint and model guidance only. Select a provider above, then confirm the exact model and credential mode for your account.',table(['Provider','Base URL','Model hint','Default credential'],templateRows,'Provider templates are unavailable.'))}${renderAIProposalQueue(state.aiProposals,null,proposalsResult.status==='rejected'?proposalsResult.reason.message:'')}${panel('Human-reviewed parser workflow','AI suggestions remain drafts. Validate them against the Assurance corpus before publication.',`<p class="muted">Provider access is optional and used last, after deterministic recovery. Clients receive only approved parser/config rollouts; provider credentials and raw secret material remain controller-side.</p>`)}`);
  }

  function generateParserHelper(form) {
    const samples=String(form.elements.samples.value||'').replace(/\r\n?/g,'\n').split('\n').map(line=>line.replace(/[\t ]+$/,'')).filter(line=>line.trim());
    if(!samples.length)throw new Error('Paste at least one non-empty sample line');
    const pairs=[...samples[0].matchAll(/([A-Za-z_][\w.-]*)\s*=\s*("[^"]*"|'[^']*'|[^\s,;]+)/g)],fields=pairs.length>1?pairs.map(([,name])=>name):['message'];
    const family=pairs.length>1?'key_value':'custom_log',pattern=pairs.length>1?fields.map(name=>`${name}=(?P<${name}>\\S+)`).join('\\s+'):'(?P<message>.*)';
    const parser={pattern_type:'regex',pattern,method:pairs.length>1?'key_value_regex':'regex',kv_delimiters:['='],fields:fields.map(name=>({name,source:`group:${name}`,type:'string'})),field_mapping:Object.fromEntries(fields.map(name=>[name,name])),known_fields:fields,projection:fields,template:Object.fromEntries(fields.map(name=>[name,'{'+name+'}']))};
    form.elements.samples.value=samples.join('\n');form.elements.name.value=`${family}-draft`;form.elements.parser.value=pretty(parser);form.elements.cases.value=pretty(samples.map(raw=>({raw,service:family,desired:{}})));const overview=form.querySelector('[data-parser-overview]')||el('parser-helper-overview');if(overview)overview.textContent=`${parser.method} · ${fields.join(', ')} · ${samples.length} trimmed samples`;
    renderParserHelperDetails(parser,form);
  }
  function renderParserHelperDetails(parser,container=document) {
    const root=container.querySelector?.('[data-parser-helper-details]')||el('parser-helper-details');if(!root)return;
    const values=[['Generated JSON',parser],['Regex',parser.pattern||''],['Templates',parser.template||{}],['KV rules',parser.kv_delimiters||parser.delimiters||[]],['Field mappings',parser.field_mapping||Object.fromEntries((parser.fields||[]).map(field=>[field.name,field.source||field.name]))],['Field definitions',parser.fields||[]]];
    root.innerHTML=`<div class="parser-helper-output-grid">${values.map(([label,value],index)=>{const content=typeof value==='string'?value:pretty(value);return `<section class="parser-helper-output"><div><h4>${esc(label)}</h4><button class="secondary small" type="button" data-action="copy-helper-value" data-copy-index="${index}" aria-label="Copy ${esc(label)}">Copy</button></div><pre>${esc(content)}</pre><textarea class="copy-source" data-copy-source="${index}" aria-hidden="true" tabindex="-1">${esc(content)}</textarea></section>`;}).join('')}</div>`;
  }
  async function loadControllerManagement() {
    const [systemResult, overviewResult, statsResult] = await Promise.allSettled([api('/system'),api('/overview'),api('/telemetry/stats')]);
    const data=systemResult.status==='fulfilled'?systemResult.value:{},overview=overviewResult.status==='fulfilled'?overviewResult.value:{},stats=statsResult.status==='fulfilled'?statsResult.value:{};
    const services=overview.services||[];
    renderPage('controller',`${panel('Controller agent status','Controller-side health and resource information returned by the secured management API.',`<div class="grid cols-4">${card('Service',data.service||'nli-controller-agent',data.health||data.status||'unknown')}${card('Version',data.version||overview.version||'—','Running controller build')}${card('Storage',data.storage?.mode||overview.storage?.mode||'unknown',data.storage?.status||overview.storage?.status||'')}${card('Registered clients',number(data.fleet?.total??overview.fleet?.total),'Heartbeat registry')}</div>`)}${panel('Controller configuration and security','Secrets are intentionally omitted from management responses. Use the host service manager for process-level start/stop and RPM lifecycle.',`<div class="callout">Manage the controller service on its host. Use the links below for enrolled-client health, delivery, and approvals.</div><div class="actions"><button type="button" class="secondary" data-action="fleet-refresh">Refresh controller status</button><button type="button" class="secondary" data-action="fleet-open" data-value="clients">Client agent operations</button><button type="button" class="secondary" data-action="fleet-open" data-value="fleet">Fleet Operations</button><button type="button" class="secondary" data-action="fleet-open" data-value="rollouts">Approvals &amp; rollouts</button></div><div class="grid cols-2"><div><h4>Effective system status</h4><pre class="json-output">${esc(pretty(data))}</pre></div><div><h4>Telemetry counters</h4><pre class="json-output">${esc(pretty(stats))}</pre></div></div>`)}${panel('Active service families','Services observed from recent metadata events.',table(['Service family','Events'],services.map(x=>`<tr><td>${esc(x.name||x.service||x)}</td><td>${number(x.count||0)}</td></tr>`)))}${systemResult.status==='rejected'?`<div class="callout warn">System status unavailable: ${esc(systemResult.reason.message)}</div>`:''}${statsResult.status==='rejected'?`<div class="callout">Telemetry counters endpoint is unavailable on this controller.</div>`:''}`);
  }
  function managedFileReview(proposal) {
    if(proposal.type!=='managed_input_file')return '';
    const targets=proposal.target_nodes||[];
    return `<details><summary>${number(targets.length)} target client(s)</summary><p class="mono">${esc(targets.join(', '))}</p></details>`;
  }

  function renderFleetJobs(data={}) {
    const live = new Map((data.live || []).map(job => [job.id, job]));
    const rows = (data.proposals || []).filter(doc => ['restart','config_push','parser_sync','managed_input_file'].includes(doc.payload?.type)).slice(0,30).map(doc => {
      const proposal = doc.payload, job = live.get(proposal.live_rollout_id);
      const results = Object.values(job?.node_results || {});
      const recent = results.sort((a,b)=>Date.parse(b.done_at||b.started_at||0)-Date.parse(a.done_at||a.started_at||0)).slice(0,100);
      return `<tr><td><b>${esc(proposal.name || doc.id)}</b><br><small class="mono">${esc(doc.id)}</small></td><td>${badge(job?.status || doc.state)}</td><td>${number(job?.total_nodes ?? proposal.target_nodes?.length ?? 0)}</td><td>${number((job?.done_nodes || 0) - (job?.failed_nodes || 0))} complete · ${number(job?.failed_nodes || 0)} failed</td><td>${results.length?`<details><summary>Per-client results (${recent.length}/${results.length})</summary><ul>${recent.map(result=>`<li><code>${esc(result.node_id)}</code> ${badge(result.status)} ${esc(result.message || result.error || '')}</li>`).join('')}</ul></details>`:'—'}</td></tr>`;
    });
    return panel('Recent client operations','Approval, command delivery, and results reported by enrolled clients.',`<div class="actions"><button type="button" class="secondary" data-action="fleet-refresh">Refresh status</button><button type="button" class="secondary" data-action="fleet-open" data-value="rollouts">Open approvals &amp; rollouts</button></div>${table(['Operation','Status','Targets','Results','Details'],rows,'No client operations have been recorded.')}`);
  }

  async function loadClientManagement() {
    const [data,jobs]=await Promise.all([api('/fleet/sync'),api('/rollouts').catch(error=>({error:error.message}))]);
    const items=data.nodes||[];
    const inventory=table(['Client','Agent service / heartbeat','Log sources','Throughput','Active parsers'],items.map(({node={},sync={}})=>{
      const m=node.metrics||{};
      return `<tr><td><b>${esc(node.hostname||node.id)}</b><br><small class="mono">${esc(node.id)} · ${esc(node.ip||'—')}</small><br><small>${esc([node.os,node.arch,node.version].filter(Boolean).join(' / '))}</small></td><td>${badge(node.status||'unknown')}<br><small>nli-agent: ${esc(agentMetricsLabel(sync))}</small><br><small>${time(node.last_seen)}</small></td><td>${renderReportedInputs(sync)}</td><td>${number(m.logs_received)} in · ${number(m.logs_parsed)} parsed<br><small>${number(m.logs_quarantined)} quarantined · ${Number(m.parse_rate_pct||0).toFixed(2)}%</small></td><td>${(node.parsers||[]).map(parser=>`${esc(parser.id)} v${esc(parser.version)}`).join('<br>')||'—'}<br><small>${badge(sync.asset_drifted?'parser drift':sync.active_asset_bundle_hash?'bundle reported':'bundle unknown')}</small></td></tr>`;
    }),'No client has checked in yet.');
    const options=items.map(({node})=>`<option value="${esc(node.id)}">${esc(node.hostname||node.id)} · ${esc(node.status||'unknown')}</option>`).join('');
    const controls=`<form id="client-action-form" class="form-grid">
      <label class="field">Action<select name="type"><option value="restart">Restart parsing agent</option><option value="parser_sync">Synchronize published parser bundle</option></select></label>
      <label class="field">Scope<select name="scope"><option value="selected">Selected clients</option><option value="all">All eligible clients for this action</option></select></label>
      <label class="field wide">Selected clients<select name="targets" multiple size="6">${options}</select><span class="hint">Select one or more clients. Restarts for offline clients wait for check-in. Parser bundle delivery uses the verified fleet rollout policy.</span></label>
      <button class="primary" type="submit" ${items.length?'':'disabled'}>Request approval</button>
    </form><div class="actions"><button type="button" class="secondary" data-action="fleet-open" data-value="input-paths">Manage input paths</button><button type="button" class="secondary" data-action="fleet-open" data-value="parsers">Manage published parser packs</button><button type="button" class="secondary" data-action="fleet-open" data-value="parser-sync">View parser bundle delivery</button></div>`;
    renderPage('clients',panel('Client agent inventory','Last reported runtime information from enrolled clients.',inventory)+panel('Request a client action','Operations are approval-gated and scoped to enrolled client IDs.',controls)+renderManagedFileForm(items)+(jobs.error?`<div class="callout warn">Client operation status unavailable: ${esc(jobs.error)}</div>`:renderFleetJobs(jobs)));
  }
  let nfiAnalysisController;
  async function nfiAnalysisRequest(path,body){nfiAnalysisController?.abort();const controller=new AbortController();nfiAnalysisController=controller;try{return await api(path,{method:'POST',body,signal:controller.signal});}finally{if(nfiAnalysisController===controller)nfiAnalysisController=null;}}
  const NFI_GUIDE_STEPS=['Select or upload raw logs','Choose Suggest & Sense or Manual Sense','Inspect source fields','Select extraction strategy','Configure extraction parameters','Create or rename output field','Assign data type','Apply split, merge or transformations','Configure inclusion / exclusion','Preview extracted value','Validate mapping on controller','Add another field','Generate complete parser','Validate complete parser','Save as draft','Submit for approval'];
  function renderFieldSenseGuide(){return `<section class="nfi-guide"><div class="section-head"><div><h3>FieldSense Guided Builder</h3><p>Intelligent Field Discovery. Unlimited Extraction Control. Precision Mapping.</p></div><button type="button" class="secondary" data-action="nfi-expert">${state.nfiExpert?'Switch to Guided Builder':'Expert mode'}</button></div><div class="actions"><button type="button" class="secondary" data-action="nfi-mode" data-value="suggest" aria-pressed="${state.nfiSenseMode==='suggest'}">Suggest &amp; Sense Mode</button><button type="button" class="secondary" data-action="nfi-mode" data-value="manual" aria-pressed="${state.nfiSenseMode==='manual'}">Manual Sense Mode</button><button type="button" class="secondary" data-action="nfi-accept-valid">Accept valid high-confidence suggestions</button><button type="button" class="secondary" data-action="nfi-bulk-tokens">Map multiple token ranges</button><button type="button" class="secondary" data-action="nfi-split-outputs">Split field into custom outputs</button></div><p id="nfi-mode-status">${state.nfiSenseMode==='manual'?'Manual Sense: select any raw fragment, add a custom field, and choose your extraction strategy.':'Suggest & Sense: inspect evidence; accept, edit, reject or regenerate each suggestion. Ambiguous delimiter splits need explicit confirmation.'}</p><ol id="nfi-guide-steps" class="nfi-guide-steps" ${state.nfiExpert?'hidden':''}></ol><p id="nfi-guide-next" role="status"></p></section>`;}
  function nfiGuideRefresh(){const host=el('nfi-guide-steps');if(!host)return;const fields=state.qrwFields||[],signature=JSON.stringify({raw:qrwRaw(),plan:qrwPlan()}),verified=qrwControllerPreview?.passed===true&&qrwControllerPreview.signature===signature,valid=fields.length>0&&fields.every(f=>['mapped','optional_absent'].includes(f.status)),doc=(state.assuranceParsers||[]).find(d=>d.id===state.nfiEditId),done=[!!qrwRaw(),!!state.nfiSenseMode,fields.length>0,fields.length>0&&fields.every(f=>QRW_METHODS.includes(f.method)),valid,fields.length>0&&new Set(fields.map(f=>f.name)).size===fields.length,fields.length>0&&fields.every(f=>QRW_TYPES.includes(f.type)),valid,fields.length>0&&fields.every(f=>!f.output_policy||['include','omit','intermediate','flatten'].includes(f.output_policy)),!!state.nfiPreviewed,verified,fields.length>1,!!state.nfiParser,state.nfiCandidateValidated===JSON.stringify({parser:state.nfiParser,cases:state.nfiCases}),!!state.nfiEditId,['in_review','approved','published','active'].includes(doc?.state)],current=Math.max(0,Math.min(15,state.nfiGuideStep??done.findIndex(value=>!value)));host.innerHTML=NFI_GUIDE_STEPS.map((label,index)=>`<li class="${done[index]?'complete':index===current?'current':'pending'}"><button type="button" data-action="nfi-guide-step" data-index="${index}" aria-current="${index===current?'step':'false'}"><b>${index+1}</b> ${esc(label)}<small>${done[index]?'Complete':index===current?'Current':'Pending'}</small></button></li>`).join('');el('nfi-guide-next').textContent=`Step ${current+1}: ${NFI_GUIDE_STEPS[current]}. ${valid?'Mapping values are locally previewable.':'Review field errors in the live trace.'} ${verified?'Controller verified this exact plan.':'Exact controller verification is pending.'}`;}
  const NFI_HELP={
    kv_pair_separators:['Literal pair separators','Enter a JSON array of whole separators. A configured list takes priority over the simple pair separator characters.','["||"] separates status=failed||server=cloud without treating a single | as a boundary.'],
    kv_quotes:['Quote characters','Enter the quote characters that should protect delimiters and spaces inside a value. Blank uses the configured profile defaults.','A double quote protects the semicolon in reason="retry; still queued".'],
    kv_escape:['Escape character','Enter one single-byte escape character. Blank uses the configured profile default.','A backslash protects the pipe in reason=retry\\|queued.'],
    kv_brackets:['Bracket pairs','Enter a JSON object mapping each opening bracket to its closing bracket. The scanner protects separators inside a balanced value.','{"[":"]"} keeps [smtp;imap] together in services=[smtp;imap].'],
    kv_key_pattern:['Key pattern','Enter a Go RE2 pattern describing valid keys in the selected fragment. This rule applies to the whole key.','[A-Za-z][A-Za-z0-9_]* permits status and mail_host.'],
    kv_reject_empty:['Reject empty values','Report a pair whose assignment is followed by no value. Leave this unchecked if empty strings carry meaning.','reason= is an empty string. A missing reason pair is handled separately by presence policy.'],
    kv_profile:['KV grammar','AutoSense compares supported grammars using the selected source. Configured separators apply only your choices. Saved grammar keeps the imported plan behavior.','Select AutoSense for |status:failed|reason:notneeded|server:cloud. Inspect KV grammar to see exact observed pairs.'],
    kv_nested_objects:['Expand nested KV objects','Parse complete bracketed key/value groups into nested objects. Plain bracketed text stays a string. Requires a Ver22 client.','details={region=west;zone=blue} maps to details.region and details.zone inside the JSON object.'],
    kv_start:['Named KV start','Limit this named key search to a source range. Choose a position unit and optionally an inclusive end or EOL.','For status after eight header tokens, enter start 9, unit token, and enable End of line.'],
    end_of_line:['End of line','Stop extraction at the first line ending in the selected range. All key values can also use a multiline range.','Token 9 to EOL inspects only the first line tail.'],
    unit:['Position unit','Choose how the start and inclusive end are counted. Use token for whitespace-delimited positions, character for Unicode positions, byte for UTF-8 bytes, or line for a multiline sample.','Token 9 starts after eight header tokens. Key value EOL stops at the current line ending.'],
    require_complete:['Complete coverage','Require the selected source range to be covered by extracted pairs. Malformed or unhandled text is reported before a candidate can pass.','status=ok;missing with pair separator ; needs a complete second assignment or a corrected source range.'],
    duplicate_policy:['Duplicate source keys','Choose reject, first, last, or array. Duplicate output names from different mapped fields are a separate conflict.','tag=a tag=b with array yields {"tag":["a","b"]}.'],
    key_include:['Include keys','Provide a JSON array of source keys to keep. The scanner still examines the selected source range.','["status","reason"] keeps only those two observed keys.'],
    key_exclude:['Exclude keys','Provide a JSON array of source keys to omit from this tail. Use this when a named mapping already owns that key.','["status"] lets a separate typed status mapping own the output.'],
    key_prefix:['Prefix output keys','Add text before every extracted output key. The final names must follow the parser field naming rules.','mail_ changes status to mail_status.'],
    key_suffix:['Suffix output keys','Add text after every extracted output key. The final names must follow the parser field naming rules.','_raw changes reason to reason_raw.'],
    source:['Source reference','For position/word/token, enter a 1-based index. For field, enter an earlier field name. KV uses a key; regex uses a Go RE2 expression.','mailnode smtp 3692 → token 2 = smtp. End positions are inclusive.'],source_end:['Inclusive end','Use a range for character, byte, line, word or token extraction. UTF-8 byte positions differ from Unicode characters.','α beta → character 1 = α; bytes 1–2 = α. Cutting byte 1 alone is invalid.'],delimiter:['Delimiter split','Choose a literal delimiter; quoted values protect embedded delimiters by default. Empty delimiter is invalid.','Note;DeliveryEvent with delimiter ; and part 2 → DeliveryEvent.'],part:['Part / capture','Split parts are 1-based; operation part 0 retains an array. Regex part selects the capture group.','a|b|c, split |, part 2 → b.'],method:['Extraction strategy','Select a source strategy, configure its parameters, then preview and verify. Advanced regex and native operations run on the bounded controller.','before/after/between use literal boundaries and explicit occurrence settings.'],kv_separator:['KV assignment','The symbol between a key and value; pair delimiters separate assignments only before another key.','status=failed;reason=not needed → status: failed, reason: not needed.'],pair_delimiters:['KV pair boundaries','List separator characters. Whitespace remains a boundary before key assignments. Quoted values preserve embedded separators.','Use | for |status:failed|reason:notneeded|server:cloud. Use ; for status=failed;reason=not needed.'],pattern:['Go RE2 expression','Use supported Go RE2 groups, classes, quantifiers and anchors. Lookbehind and backreferences are unsupported. Verify on controller for actual matches.','(?P<level>Note);(?P<event>[^ ]+) → named groups for Note;DeliveryEvent.'],boundary_start:['Boundary start','Literal boundary for before/after/between. Choose occurrence first, last, nth or all; preserve_boundary controls inclusion.','before = on a=b=c returns a; last occurrence returns a=b.'],boundary_end:['Boundary end','The closing literal boundary for between. Empty unmatched boundaries are reported as missing according to presence policy.','between [ and ] in user=[alice] → alice.'],occurrence:['Repeated occurrences','Choose first, last, nth or all. For nth, set occurrence index starting at 1; all returns an array.','a=1 a=2, last matching boundary selects the later occurrence.'],indices:['Multiple selections','JSON array of positive 1-based word/token indexes. The selected values remain ordered.','[1,3] selects alpha and gamma from alpha beta gamma.'],token_mode:['Tokenization','Whitespace preserves original spans. Delimiter uses explicit separators, mixed accepts several, regex uses bounded Go RE2.','token separators [";","|"] supports a;b|c.'],assignment_separators:['Multiple assignments','JSON string array of literal key/value assignment separators. Combine with pair boundaries and quote protection.','[":"] handles pipe-delimited colon pairs. [":",";"] with whitespace pairs handles status:failed reason;notneeded server;cloud.'],key_aliases:['KV aliases','JSON object maps observed keys to desired names for all-key extraction. Include/exclude and prefix/suffix remain explicit.','{"src":"source_ip"} renames src in the extracted object.'],output_policy:['Output inclusion','Include emits the field. Intermediate keeps it available to later fields and removes it from final JSON. Omit excludes output; it never discards the raw log.','first,last intermediate → full_name included.'],empty_policy:['Empty values','Preserve retains "". Missing applies presence/default/null policy. Null emits null; reject fails validation.','reason= is present and empty; it is different from an absent reason key.'],fields:['Merge references','JSON pipeline references earlier mapped fields. Join separator controls concatenation; conditional_merge has an explicit condition and fallback.','first=Alice last=Lee with separator space → Alice Lee.'],condition_field:['Conditional merge','Reference an earlier field, choose exists/equals/not_empty and condition value. else_value is the explicit fallback.','status equals ok → merge first,last; otherwise use unknown.'],algorithm:['Hash algorithm','Choose sha256 or sha512. Hashing is deterministic; raw evidence remains governed by retention.','hash(email) yields a digest; mask/redact can produce visible placeholders instead.'],remove_sources:['Remove merged inputs','When explicitly enabled, successful merging removes referenced original fields from final output. Source evidence remains retained.','first,last → full_name only after successful merge.'],policy:['Presence policy','Required fails on absence. Optional omits, nullable emits null, default uses an explicit value. Invalid quotes and types still fail.','missing reason with optional → no reason key.'],type:['Output type','Native conversion validates the observed value. Identifiers with leading zeros or large integers should remain strings.','007 remains "007"; invalid integer abc never becomes 0.'],timestamp_format:['Timestamp layout','Use Go time layouts or native compact_mail / epoch formats. Set an IANA timezone for configured local-time handling.','compact_mail parses 20250811 080053105+0100; verify the exact input layout.']};
  Object.assign(NFI_HELP,{name:['Output field name','Choose any unique non-reserved name of up to 64 letters, digits and underscores, starting with a letter. Rename freely; duplicate collisions fail explicitly.','hostname → mail_server_hostname; source/raw/nli are reserved metadata names.'],label:['Display label','A readable label for operators; the JSON key comes from Output field name.','mail_server_hostname may display Mail server hostname.'],description:['Description','Document why this field exists and how consumers should use it. This metadata is preserved with the native plan.','Source SMTP hostname from the third header token.'],example:['Example','Record a representative expected value for reviewers; this does not create a default or fabricate a value.','mailnode.example is an example; extraction still reads actual evidence.'],default_value:['Explicit fallback / constant','Constants always use this value. Default presence policy uses it only when the source is absent. Empty string is an explicit permitted fallback.','missing reason → unknown when default=unknown; invalid integers still fail.'],null_policy:['Null and missing policy','Omit permits optional absence. Preserve null emits null; reject fails unless an explicit default applies.','missing optional client: omit the key or emit null by explicit policy.'],transform:['Final transformation','Apply none, trim, lowercase or uppercase after the ordered operation chain, before native conversion.','  WARN  with trim → WARN.'],timezone:['Timezone policy','Preserve keeps input zone semantics. UTC normalizes. Configured requires an IANA timezone name.','Asia/Kolkata resolves a configured local wall-clock timestamp; verify controller output.'],timezone_name:['IANA timezone','Used only with timezone policy configured. Enter a valid IANA database name; offsets are not timezone names.','Asia/Kolkata; America/New_York.'],validation:['Bounded validation','Comma-separated min/max, min_length/max_length, or one_of. Values never silently become zero or false.','min=0,max=65535 for a port; one_of=ok|failed for status.'],quote_aware:['Quoted delimiter protection','Checked respects matching quotes and escapes during splitting. Uncheck only when quotation marks are literal data.','"west,region",007 with comma split keeps west,region as one value.'],confirmed:['Confirm interpretation','Explicitly review ambiguous compound splits. Confirmation verifies your intended structure; it does not approve publication.','Review Note;DeliveryEvent before mapping its two parts.'],duplicate_policy:['Repeated keys','Reject is safest. First or last chooses the explicit occurrence; array retains every value.','tag=a:tag=b → [a,b] when array is selected.'],join_separator:['Join separator','Literal text inserted between earlier field references. Source fields remain available unless remove_sources is explicitly set.','a=mail,b=node with - → mail-node.'],occurrence_index:['Nth occurrence index','Positive 1-based occurrence used with occurrence=nth. First occurrence is 1.','third : in a:b:c:d selects boundary before d.'],preserve_boundary:['Preserve boundary text','Checked includes matching boundary characters in before/after/between extraction. Default removes them from the selected fragment.','[alice] between [ and ] → alice; preserving boundaries → [alice].'],direction:['Split direction','Left is the normal order. Right applies splitting limits from the final occurrence.','a:b:c split from right with one cut → a:b and c.'],count:['Count / width','For first/last use N items; chunk splits use a positive chunk width. Delimiter split count bounds cuts.','first 3 characters of abcdef → abc; chunk width 2 → ab,cd,ef.'],unit:['Position / count unit','Choose character, byte, word, token or line. Characters are Unicode code points; bytes are UTF-8 and must not cut an encoded character.','α occupies one character and two UTF-8 bytes.'],token_separators:['Token separator list','JSON string array of literal separators for delimiter/mixed tokenization. Empty separators are invalid.','[";","|"] splits a;b|c into a,b,c.'],token_pattern:['Token RE2 pattern','Go RE2 controls regex tokenization/matching. Unsupported PCRE constructs are rejected by the controller.','[A-Za-z]+ matches letter runs; no lookbehind or backreferences.'],key_include:['Include keys','JSON string array selecting keys from all-key extraction. Empty list leaves the default selected object unchanged.','["status","reason"] extracts only those keys.'],key_exclude:['Exclude keys','JSON string array of keys to exclude from all-key extraction. This removes output properties, not the original log evidence.','["password"] prevents password from entering the extracted KV object.'],key_prefix:['Key prefix','Literal prefix applied to emitted all-KV object keys after explicit alias mapping.','net_ + src → net_src.'],key_suffix:['Key suffix','Literal suffix for emitted all-KV object keys. Check collisions after aliases and prefix/suffix changes.','_value + status → status_value.'],custom_type:['Custom type label','A descriptive type label accompanied by custom_pattern for actual bounded validation. It does not execute scripts.','order_code validated by ^[A-Z]{2}-[0-9]{6}$.'],custom_pattern:['Custom type RE2 validator','Go RE2 validation condition for custom string types. Use anchors when the entire value must match.','^[A-Z]{2}-[0-9]{6}$ accepts AB-123456 and rejects arbitrary text.'],limit:['Output limit','Positive bounded maximum outputs for splitting; remaining source is preserved according to native split semantics.','limit 2 on a:b:c retains a and b:c rather than losing the remainder.'],preserve_delimiter:['Preserve delimiters','Checked retains separator text in split output fragments; use when downstream fields need the original boundary.','a:b split on : may retain the boundary explicitly.'],condition_operator:['Condition operator','exists checks presence, not_empty checks observed content, equals compares to condition_value.','status equals ok selects the configured merge; otherwise else_value.'],condition_value:['Condition value','Explicit comparison value for equals. This does not overwrite the referenced source field.','ok matches the observed status=ok.'],else_value:['Conditional fallback','Literal result when the conditional merge does not match. Fallback is explicit and remains subject to type/validation policies.','unknown when status is not ok.'],value:['Literal value','The literal used by prefix/suffix/add, masking/redaction, or nested key_value key selection; operation determines its meaning.','prefix user: plus alice → user:alice; nested key_value reason selects reason.'],find:['Find literal','Exact literal for replace/remove. Empty find is rejected to prevent unintended expansion.','Find ; in Note;Event and replace with a space.'],replace:['Replacement literal','Replacement text for literal replace, or Go-compatible replacement for regex_replace. Regex runs on the controller.','Replace cloud with archive; source evidence remains retained.'],separator:['Operation separator','Join separator for merge; nested key_value uses pair separators here. Assignment separator for nested KV is delimiter.','nested action;keep|geo;US uses delimiter ; and separator |.'],group:['Capture group','Named group or numeric capture selector as supported by the native operation. Group 0 means the complete match.','(?P<id>[0-9]+): group id selects the digits.'],start:['Start position','Positive 1-based start position. End is inclusive; the chosen unit determines character/byte/word/token semantics.','start 2,end 4 on abcdef → bcd.'],end:['Inclusive end','Use a positive inclusive end; operation end 0 means through the end where supported. Do not confuse Unicode and UTF-8 bytes.','start 2,end 0 on abcdef → bcdef.'],'nfi-service':['Service scope','auto infers a reviewed service; comma-separated explicit names scope an override to those services. Wildcard stays fallback unless a guarded global override is authorized.','smtp,imap targets those services and preserves unrelated parser behavior.'],'nfi-mapping-priority':['Mapping priority','Integer 0–1000 orders reviewed native mapping tiers. A cloned repair increments priority while keeping the original parser as fallback.','repair priority 3 precedes the original priority 2; rollback reveals the original.'],'nfi-scope-pattern':['Scope RE2 condition','Optional bounded match condition restricts which records enter the reviewed mapping. Maximum 1024 UTF-8 bytes.','^PREFIX restricts a custom mapping to records beginning PREFIX.'],'nfi-global-scope':['Global override authorization','Explicit global override requires a reviewed RE2 match condition. It remains a draft until server validation and authorized approval.','Use a narrow ^AUDIT pattern with wildcard scope only when deliberately covering several services.']});
  function nfiHelpFor(control){
    const key=control.dataset.qrwKey||control.dataset.qrwOpKey||control.dataset.kvInputKey||control.id;
    const label=control.getAttribute('aria-label')||control.closest('label')?.childNodes[0]?.textContent?.trim()||key;
    const examples={
      find:['Type exactly what you want to change. The text is literal, so punctuation and spaces have their usual meaning.','Use # to find a hash, ][ to find adjacent wrappers, or a single space to find spaces.'],
      replace:['Type the replacement text. An empty value removes every match.','Replace ][ with | to keep a boundary between pairs, or leave it blank to delete a stray #.'],
      action:['Choose whether to edit every match or remove text only at one edge of the selected range.','Use Remove from start for an opening bracket and Remove from end for its closing bracket.'],
    };
    if(Object.hasOwn(examples,key))return {title:label,description:examples[key][0],example:examples[key][1]};
    const entry=NFI_HELP[key]||NFI_HELP[key.includes('pattern')?'pattern':key.includes('separator')?'delimiter':key.includes('token')?'token_mode':key.includes('key_')?'key_aliases':key.includes('boundary')?'boundary_start':'method'];
    return {title:label||entry[0],description:entry[1],example:entry[2]};
  }
  const mappingLabels={name:'Field name',label:'Display label',type:'Field type',policy:'Presence policy',null_policy:'Missing value policy',
    method:'Extraction method',source_end:'Inclusive end (optional)',transform:'Transform',timestamp_format:'Timestamp format',
    timezone:'Timezone policy',timezone_name:'IANA timezone',default_value:'Default value',validation:'Validation rule'};
  const operationInputs={
    split:['delimiter','part','quote_aware'],replace:['find','replace'],remove:['find'],
    strip_prefix:['value'],strip_suffix:['value'],prefix:['value'],suffix:['value'],add:['value'],
    regex_extract:['pattern','group'],regex_replace:['pattern','replace'],split_regex:['pattern','part','quote_aware'],
    key_value:['value','delimiter','separator'],select:['part'],substring:['start','end'],character:['start','end'],
    byte:['start','end'],line:['start','end'],join:['fields','separator'],merge:['fields','separator'],
    conditional_merge:['fields','separator'],copy:['fields'],trim:[],lowercase:[],uppercase:[],omit:[],exclude:[],hash:[]
  };
  function labelMappingControls(editor){
    for(const row of editor.querySelectorAll('[data-qrw-row]')){
      const method=row.querySelector('[data-qrw-key="method"]')?.value;
      const sourceLabel={field:'Earlier field name',raw:'Raw source',regex:'RE2 pattern',regex_groups:'RE2 pattern with named groups',
        regex_all:'RE2 pattern',key_value:'Key to extract',json_path:'JSON path',delimiter:'Source token or raw'}[method]||'Start position (1-based, or raw for KV)';
      for(const control of row.querySelectorAll('[data-qrw-key]')){
        if(control.closest('label')||control.previousElementSibling?.classList.contains('nfi-visible-label'))continue;
        const key=control.dataset.qrwKey;
        const title=key==='source'?sourceLabel:key==='delimiter'?'Split delimiter':key==='part'?(method==='delimiter'?'Split part number':'Regex capture group'):mappingLabels[key];
        if(!title)continue;
        const label=document.createElement('label');
        label.className='nfi-visible-label';
        control.id||=row.dataset.qrwRow+'-'+key;
        label.htmlFor=control.id;
        label.textContent=title;
        control.before(label);
      }
      const operations=row.querySelector('.nfi-operations');
      if(!operations)continue;
      operations.querySelector('summary').textContent='Field operations ('+operations.querySelectorAll('[data-qrw-op]').length+')';
      if(!operations.previousElementSibling?.classList.contains('nfi-operation-shortcuts')){
        const tools=document.createElement('div');
        tools.className='actions nfi-operation-shortcuts';
        for(const [action,label] of [['replace','Replace text'],['split','Split value']]){
          const button=document.createElement('button');
          button.type='button';button.className='secondary small';button.dataset.action='qrw-operation-add';
          button.dataset.id=row.dataset.qrwRow;button.dataset.operation=action;button.textContent=label;
          tools.append(button);
        }
        operations.before(tools);
      }
      for(const operation of operations.querySelectorAll('[data-qrw-op]')){
        const kind=operation.querySelector('[data-qrw-op-key="op"]').value;
        if(!Object.hasOwn(operationInputs,kind))continue;
        for(const input of operation.querySelectorAll(':scope > label > input[data-qrw-op-key]')){
          input.closest('label').hidden=!operationInputs[kind].includes(input.dataset.qrwOpKey);
        }
        if(kind==='key_value'){
          for(const [key,label] of [['value','Key to extract'],['delimiter','Assignment operator'],['separator','Pair separators']]){
            operation.querySelector('[data-qrw-op-key="'+key+'"]').parentElement.firstChild.textContent=label;
          }
        }
      }
    }
  }
  function mappingAIControls(mapping=false){return `<section class="nfi-import wide" data-ai-mapping-host data-ai-mapping-editor="${mapping?'true':'false'}"><h4>Optional AI mapping proposal · separate from native suggestions</h4><p>Suggest with Nexora AI explicitly sends masked evidence and bounded approved parser context to a configured controller provider. Native suggestions and manual edits are preserved. A proposal is ephemeral and cannot save, approve, publish or deploy itself.</p><div class="actions"><button type="button" class="secondary" data-action="mapping-ai-readiness">Check Nexora AI readiness</button><button type="button" class="secondary" data-action="mapping-ai-suggest">Suggest with Nexora AI</button></div><div data-ai-mapping-status role="status"></div><div data-ai-mapping-proposal hidden><pre class="json-output" data-ai-mapping-json></pre><div class="actions"><button type="button" class="secondary" data-action="mapping-ai-import">${mapping?'Import proposed mapping':'Open proposed mapping in FieldSense'}</button><button type="button" class="secondary" data-action="mapping-ai-dismiss">Dismiss proposal</button></div></div></section>`;}
  function mappingAIRaw(host){if(host.dataset.aiMappingEditor==='true')return qrwRaw();return host.closest('form')?.querySelector('[name=samples]')?.value||host.querySelector('[data-ai-mapping-raw]')?.value||'';}
  function setupMappingAIControls(){
    const editor=page.querySelector('.qrw-field-editor');if(editor&&!editor.querySelector('[data-ai-mapping-host]'))editor.insertAdjacentHTML('beforeend',mappingAIControls(true));
    for(const form of page.querySelectorAll('#parser-helper-form,#gprs-parser-helper-form,#parser-workspace-form'))if(!form.querySelector('[data-ai-mapping-host]'))form.insertAdjacentHTML('beforeend',mappingAIControls(false));
    if(['studio','regex'].includes(state.page)&&!page.querySelector('[data-ai-mapping-host]')){page.insertAdjacentHTML('beforeend',mappingAIControls(false));const host=page.querySelector('[data-ai-mapping-host]');host.insertAdjacentHTML('afterbegin','<label class="field">Exact raw log for optional AI mapping proposal<textarea data-ai-mapping-raw rows="3" placeholder="Paste an authorized raw record; existing regex/template settings are preserved."></textarea></label>');}
  }
  function nfiDecorate(){setupMappingAIControls();if(!el('qrw-field-rows'))return;labelMappingControls(page);if(state.page==='field-mapping'){const count=page.querySelectorAll('.grid.cols-4 .metric-value')[1];if(count)count.textContent=number(state.qrwFields.length);}for(const control of page.querySelectorAll('.qrw-field-editor input,.qrw-field-editor select,.nfi-operation input,.nfi-operation select')){if(control.dataset.nfiHelp)continue;control.dataset.nfiHelp='true';const help=nfiHelpFor(control),button=document.createElement('button');button.type='button';button.className='nfi-info';button.dataset.action='nfi-help';button.dataset.helpKey=control.dataset.qrwKey||control.dataset.qrwOpKey||control.id;button.dataset.helpTitle=help.title;button.dataset.helpDescription=help.description;button.dataset.helpExample=help.example;button.title=`${help.title}: ${help.description} Example: ${help.example}`;button.setAttribute('aria-label',`Information: ${help.title}`);button.setAttribute('aria-description',button.title);button.dataset.tooltip=button.title;button.textContent='ⓘ';control.insertAdjacentElement('afterend',button);}nfiGuideRefresh();}
  let nfiLiveTimer;
  function nfiScheduleLive() {
    if(state.page==='quarantine-wizard'||state.page==='assurance')state.gprsPreview=null;
    qrwControllerPreview=null;
    qrwLivePreviewController?.abort();
    state.nfiCandidateValidated=null;
    page.dataset.dirty='true';
    clearTimeout(nfiLiveTimer);
    const inspector=el('kvie-inspector');
    if(inspector&&!kvInspectorIsCurrent(kvInspectorState))inspector.hidden=true;
    const diagnostic=el('nfi-controller-result');
    if(diagnostic)diagnostic.hidden=true;
    nfiLiveTimer=setTimeout(async()=>{
      const liveState=el('nfi-live-state');
      if(!liveState||!el('qrw-field-rows'))return;
      try {
        const result=qrwTestFields(true);
        state.nfiPreviewed=true;
        el('nfi-live-json').textContent=pretty(result.output);
        liveState.textContent=result.errors.length?`Unsaved mapping · ${result.errors.length} issue(s). Review the field messages below.`
          :result.pending.length?'Unsaved mapping · native preview pending.':'Unsaved mapping · local preview passed.';
        el('nfi-live-trace').innerHTML=state.qrwFields.map(field=>`<button type="button" class="nfi-trace-field" data-action="nfi-highlight" data-id="${esc(field.id)}"><b>${esc(field.name)}</b> ← ${esc(field.method)} ${esc(field.source)} → ${esc(field.preview||field.evidence)}</button>${field.operations?.length?`<details><summary>${field.operations.length} intermediate operation(s)</summary><pre>${esc(pretty(nfiOperationTrace(field)))}</pre></details>`:''}`).join('');
        if(state.page==='field-mapping'){
          if(state.qrwFields.length){
            if(result.errors.length||result.pending.length){
              state.nfiParser=qrwBuildParser(true,true);
              el('nfi-parser-json').value=pretty(state.nfiParser);
              el('nfi-candidate-status').textContent='Unsaved plan updated. Preview the current mapping before validation.';
            }else{
              nfiGenerate(true);
              el('nfi-candidate-status').textContent='Unsaved parser updated. Complete parser validation is pending.';
            }
          }else{
            state.nfiParser=null;
            el('nfi-parser-json').value='{}';
          }
        }
        nfiDecorate();
        if(!nfiAnalysisController&&qrwRaw().trim()&&result.pending.length
          && !page.querySelector('.qrw-field-editor [data-qrw-key]:invalid, .qrw-field-editor [data-qrw-op-key]:invalid')){
          await qrwVerifyController({background:true});
        }
      }catch(error){if(liveState.isConnected)liveState.textContent=error.message;}
    },350);
  }
  function nfiOperationTrace(field){const steps=[];for(let index=0;index<(field.operations||[]).length;index++){const result=qrwExtract({...field,operations:field.operations.slice(0,index+1),type:'string',transform:'none',validation:''},qrwRaw(),state.nfiDesired||{});steps.push({step:index+1,operation:field.operations[index].op,value:result.absent?'(absent)':result.error?'Controller verification required: '+result.error:result.value});}return steps;}
  function nfiHighlight(field){const host=el('nfi-raw-selection'),raw=qrwRaw();if(!host)return;const tokens=qrwTokens(raw),native=qrwControllerPreview?.signature===JSON.stringify({raw,plan:qrwPlan()})?(qrwControllerPreview.evidence||qrwControllerPreview.fields)?.[field.name]:null;if(native?.source===false||field.status==='optional_absent'){host.textContent=raw;if(el('nfi-source-hint'))el('nfi-source-hint').textContent='This value has no direct source span (derived, constant, or absent). Inspect its earlier-field references and native trace.';return;}let start=0,end=raw.length,precision='Approximate local text-match hint; repeated values may select the first occurrence. Select Preview here for exact source spans.';if(native&&native.start>=0&&native.end>=native.start){const bytes=new TextEncoder().encode(raw),decoder=new TextDecoder('utf-8',{fatal:true});start=decoder.decode(bytes.slice(0,native.start)).length;end=decoder.decode(bytes.slice(0,native.end)).length;precision='Exact source span from the native controller evidence.';}else if(['kv_all','kv_eol'].includes(field.method)&&['token','word',''].includes(field.unit||'')){start=tokens[Number(field.source||1)-1]?.start??0;end=field.source_end?tokens[Number(field.source_end)-1]?.end??raw.length:raw.length;if(field.method==='kv_eol'||field.end_of_line){const lineBreak=raw.slice(start,end).search(/[\r\n]/);if(lineBreak>=0)end=start+lineBreak;}precision='Selected local key/value range; inspect native per-key evidence for authoritative spans.';}else if(field.method==='position'){precision='Local positional source span; select Preview here for authoritative evidence.';start=tokens[Number(field.source)-1]?.start??0;end=tokens[Number(field.source_end||field.source)-1]?.end??start;}else if(field.method==='character'){precision='Local Unicode character range; select Preview here for authoritative evidence.';start=[...raw].slice(0,Math.max(0,Number(field.source)-1)).join('').length;end=[...raw].slice(0,Number(field.source_end||field.source)).join('').length;}else if(field.preview&&field.preview!=='(absent)'){start=raw.indexOf(field.preview);end=start+field.preview.length;}if(start<0){host.textContent=raw;if(el('nfi-source-hint'))el('nfi-source-hint').textContent='No direct raw span for this derived value. Inspect earlier field references and verify native evidence.';return;}if(el('nfi-source-hint'))el('nfi-source-hint').textContent=precision;host.innerHTML=esc(raw.slice(0,start))+`<mark>${esc(raw.slice(start,end))}</mark>`+esc(raw.slice(end));const charStart=[...raw.slice(0,start)].length+1,charEnd=[...raw.slice(0,end)].length;host.title=`${field.name}: line ${raw.slice(0,start).split('\n').length}; characters ${charStart}–${charEnd}; bytes ${new TextEncoder().encode(raw.slice(0,start)).length+1}–${new TextEncoder().encode(raw.slice(0,end)).length}`;for(const row of page.querySelectorAll('[data-qrw-row]'))row.classList.toggle('nfi-source-active',row.dataset.qrwRow===field.id);}
  async function loadFieldMapping(){
    if(!state.qrwFields.length){try{const saved=JSON.parse(sessionStorage.getItem('nli.qrw.mapping-draft')||'null');if(saved?.version===1&&Array.isArray(saved.fields))state.qrwFields=saved.fields.slice(0,128).map(field=>qrwField(field));}catch{sessionStorage.removeItem('nli.qrw.mapping-draft');}}
    const [status,candidates]=await Promise.allSettled([api('/field-mapping'),api('/assurance/parsers')]);const snapshot=status.status==='fulfilled'?status.value:null;state.assuranceParsers=candidates.status==='fulfilled'?(candidates.value.parsers||[]):[];
    const rows=state.assuranceParsers.filter(doc=>doc.payload?.parser?.pattern_type==='mapping').map(doc=>{const actions=doc.state==='draft'?['validate','submit']:doc.state==='in_review'?['approve','reject']:doc.state==='approved'?['publish']:['published','active'].includes(doc.state)?['rollback','offboard']:[];return `<tr><td>${esc(doc.payload?.name||doc.id)}<br><small>${esc(doc.id)} · r${number(doc.revision)}</small></td><td>${badge(doc.state)}</td><td>${number(doc.payload?.parser?.mapping_plan?.fields?.length)} mapped fields</td><td>${['draft','rejected'].includes(doc.state)?`<button type="button" class="secondary small" data-action="nfi-edit-candidate" data-id="${esc(doc.id)}">Edit mapping</button> `:''}${!['draft','rejected'].includes(doc.state)?`<button type="button" class="secondary small" data-action="nfi-repair-candidate" data-id="${esc(doc.id)}">Repair as new draft</button> `:''}${actions.map(action=>`<button type="button" class="secondary small" data-action="assurance-parser-action" data-id="${esc(doc.id)}" data-value="${action}">${esc(action[0].toUpperCase()+action.slice(1))}</button>`).join(' ')}</td></tr>`;});
    renderPage('field-mapping',`<div class="dashboard-intro"><div><p class="eyebrow">Native parser subsystem</p><h1>NLI FieldSense Studio</h1><p>Build a deterministic field mapping from observed evidence. The native controller and enrolled clients execute the same reviewed mapping plan locally; no AI call runs for each log line.</p></div></div>${status.status==='rejected'?`<div class="callout warn">NFI controller status unavailable: ${esc(status.reason.message)}. Local editing remains available; controller verification and guarded publication require a reachable R7 controller.</div>`:''}<div class="grid cols-4">${card('Subsystem',snapshot?'Available':'Status unavailable',snapshot?.engine||'Native bounded mapping engine')}${card('Mapping fields',number(state.qrwFields.length),'Explicit names and policies')}${card('Saved mapping candidates',number(rows.length),'Authoritative controller lifecycle')}${card('Runtime parity','One plan','Preview → candidate → sealed parser pack → client')}</div><section class="panel"><h3>KV Intelligence Engine</h3><p role="status">${snapshot?.kvie?.engine?`Controller reports ${esc(snapshot.kvie.engine)} · ${number(snapshot.kvie.inspection_total)} inspections · ${number(snapshot.kvie.no_match_total)} without a match`:'Controller KV status is unavailable.'}</p><p>Use Key value EOL in the mapping workbench, then choose Inspect KV grammar on that field to see detected separators, individual keys and source positions. The controller and client use the same native engine for a reviewed parser pack.</p></section><section class="panel"><h3>Sample and corpus evidence</h3><label class="field">Raw record<textarea id="nfi-raw" rows="5" placeholder="Paste one exact log record or select a corpus sample below.">${esc(state.nfiRaw||'')}</textarea></label><div class="actions"><button type="button" class="secondary" data-action="nfi-refresh-inspector">Refresh positions</button><button type="button" class="secondary" data-action="nfi-import-file">Load sample file</button><input id="nfi-sample-file" type="file" accept=".log,.txt,.jsonl,.csv,.tsv" hidden></div><details><summary>Learn structural suggestions from multiple samples</summary><p>Native evidence analysis groups sample structures and reports occurrence frequencies. These are reviewable suggestions, not a trained semantic model or a claim to understand every format.</p><label class="field">One record per line<textarea id="nfi-corpus" rows="6" placeholder="Paste an authorized sample corpus; it is processed on explicit request and not retained here.">${esc(state.nfiCorpus||'')}</textarea></label><label class="field">Manual record delimiter override<input id="nfi-separator" value="${esc(state.nfiSeparator||'')}" placeholder="Literal comma (,), pipe (|), semicolon (;), or \\t for tab"></label><div class="actions"><button type="button" class="primary" data-action="nfi-learn">Learn sample structures</button></div><div id="nfi-learning-status" role="status"></div><div id="nfi-profile-list"></div></details></section><div class="callout">Native mapping plans require an R7 controller and R7 client RPM. Older R6 clients cannot execute this new mapping stage; upgrade before rollout and verify the real client acknowledgement.</div>${renderQRWFieldEditor()}<section class="panel"><h3>Governed native mapping candidate</h3><p>Generate and verify an exact candidate before saving. Saving creates a draft; authorized review and publication are required before live clients receive it.</p><div class="form-grid"><label class="field">Candidate name<input id="nfi-parser-name" value="${esc(state.nfiName||'native-field-mapping')}" maxlength="64"></label><label class="field">Version<input id="nfi-parser-version" value="${esc(state.nfiVersion||'0.1.0')}"></label><label class="field wide">Complete native parser definition<textarea id="nfi-parser-json" rows="8">${esc(pretty(state.nfiParser||{}))}</textarea></label><label class="field wide">Exact expected cases<textarea id="nfi-cases-json" rows="6">${esc(pretty(state.nfiCases||[]))}</textarea></label><div class="actions wide"><button type="button" class="secondary" data-action="nfi-generate">Generate candidate</button><button type="button" class="secondary" data-action="nfi-validate-candidate">Validate complete parser</button><button type="button" class="primary" data-action="nfi-save-candidate">Validate and save review-gated draft</button><button type="button" class="secondary" data-action="nfi-handoff-parser">Use candidate in Parser Creation Workspace</button><button type="button" class="secondary" data-action="nfi-handoff-output">Use mapped names in output delivery</button></div><div id="nfi-candidate-status" class="wide" role="status"></div></div></section><section class="panel"><h3>Production rollout and client acknowledgement</h3><p>Publication and client acknowledgement are distinct. Refresh actual fleet evidence; retry requests still use the existing approval gate. A repair keeps the original parser as fallback.</p><button type="button" class="secondary" data-action="nfi-fleet-status">Refresh rollout / ACK status</button><div id="nfi-fleet-status"></div></section>${panel('Native mapping review and publication','Lifecycle states come from the controller; drafts never activate themselves.',table(['Candidate','State','Mapping','Actions'],rows,'No native field mapping candidates have been saved.'))}`);state.qrwStep=4;
  }
  function nfiLoadProfile(index){const profile=state.nfiProfiles?.[index];if(!profile||!Array.isArray(profile.fields))throw new Error('This structural profile does not contain a reviewable field plan.');state.qrwFields=profile.fields.map(field=>qrwField({...field,status:field.method==='delimiter'&&!field.confirmed?'needs_confirmation':'unresolved'}));if(profile.sample||profile.example){state.nfiRaw=profile.sample||profile.example;const raw=el('nfi-raw');if(raw)raw.value=state.nfiRaw;}el('qrw-field-rows').innerHTML=renderQRWFieldRows();el('nfi-inspector-host').innerHTML=qrwRawInspector();qrwControllerPreview=null;}
  function nfiGenerate(preserveFocus=false){const parser=qrwBuildParser(preserveFocus),service=String(parser.service_scope||'auto'),cases=[{raw:qrwRaw(),service:service.split(',').map(value=>value.trim()).filter(Boolean)[0]==='*'?'auto':service.split(',').map(value=>value.trim()).filter(Boolean)[0]||'auto',source:'nfi:operator-evidence',desired:JSON.parse(JSON.stringify(state.nfiDesired||{}))}];state.nfiParser=parser;state.nfiCases=cases;el('nfi-parser-json').value=pretty(parser);el('nfi-cases-json').value=pretty(cases);el('nfi-candidate-status').textContent='Native plan generated. Validate and save to create an audited review-gated draft.';}

  async function loadQuarantineWizard(){
    if(!state.qrwFields.length){try{const saved=JSON.parse(sessionStorage.getItem('nli.qrw.mapping-draft')||'null');if(saved?.version===1&&Array.isArray(saved.fields))state.qrwFields=saved.fields.slice(0,256);}catch{sessionStorage.removeItem('nli.qrw.mapping-draft');}}
    const requested=state.page;await loadAssurance({restoreQRW:true,pageID:requested});
    if(state.page!==requested)return;
    const gprs=page.querySelector('#assurance-panel-gprs'),tablist=page.querySelector('.assurance-subnav');
    page.querySelectorAll('.assurance-tab-panel').forEach(panel=>panel.hidden=panel!==gprs);
    if(gprs)gprs.hidden=false;if(tablist)tablist.remove();
    page.querySelectorAll(':scope > .callout, :scope > .grid, :scope > .assurance-tab-panel:not(#assurance-panel-gprs)').forEach(node=>node.remove());
    page.insertAdjacentHTML('afterbegin','<div class="dashboard-intro"><div><p class="eyebrow">UPES recovery · governed parser lifecycle</p><h1>Quarantine Resolution Wizard</h1><p>Resolve retained parsing failures in one nine-stage workspace while the current production parser remains authoritative.</p></div></div>');
    page.dataset.page='quarantine-wizard';setupQRWStepPanels(gprs||page);
  }
  const LOADERS = { 'sample-history':loadSampleHistory, 'header-templates':()=>loadConfig('header-templates'), 'kv-templates':()=>loadConfig('kv-templates'), 'reference-ai':loadReferenceAI, 'building-blocks':loadBuildingBlocks, 'field-mapping':loadFieldMapping, operations:loadOperations, monitoring:async()=>renderMonitoring(), 'sync-fabric':async()=>renderSyncFabric(), 'nhle-monitor':async()=>renderSubsystemPage('nhle-monitor'), 'napi-monitor':async()=>renderSubsystemPage('napi-monitor'), 'nal-monitor':async()=>renderSubsystemPage('nal-monitor'), upes:loadUPES, nupire:loadNupire, assurance:loadAssurance, 'quarantine-wizard':loadQuarantineWizard, 'parser-workspace':loadParserWorkspace, 'field-integrity':loadFieldIntegrity, fleet:loadFleet, 'parser-sync':loadParserSync, clients:loadClientManagement, controller:loadControllerManagement, parsers:loadParsers, migration:loadMigration, 'input-paths':loadInputPaths, kv:()=>loadConfig('kv'), static:()=>loadConfig('static'), projection:()=>loadConfig('projection'), studio:()=>loadConfig('studio'), regex:()=>loadConfig('regex'), knownkeys:()=>loadConfig('knownkeys'), semantic:()=>loadConfig('semantic'), 'engine-network':()=>loadDomainEngine('engine-network'), 'engine-url':()=>loadDomainEngine('engine-url'), 'engine-email':()=>loadDomainEngine('engine-email'), 'engine-subject':()=>loadDomainEngine('engine-subject'), outputs:loadOutputEngine, metadata:loadMetadata, streams:loadStreams, unknowns:loadUnknowns, 'controller-gateway':loadControllerGateway, 'ai-privacy':loadAIPrivacy, 'api-monitor':loadAPIMonitor, latency:loadLatency, drift:loadDrift, performance:loadPerformance, rollouts:loadRollouts, artifacts:loadArtifacts, security:loadSecurity, system:loadSystem, communications:loadCommunicationStatus, notifications:loadNotifications, ai:loadAIProvider };

  async function loadUPES(){
    const canonical=state.sync.snapshot||{},upes=canonical.subsystems?.upes;
    let summary=upes||{},records=upes?.failures||upes?.failure_records||upes?.failure_cases||[],candidates=upes?.candidates||upes?.candidate_lifecycle||[],sourceErrors=[];
    const [recordsResult,nupireResult,legacySummaryResult]=await Promise.allSettled([api('/quarantine/records?limit=100'),api('/nupire/snapshot'),...(!upes?[api('/quarantine/summary')]:[])]);
    if(recordsResult.status==='fulfilled')records=recordsResult.value.records||records;else sourceErrors.push('Failure records unavailable: '+recordsResult.reason.message);
    if(nupireResult.status==='fulfilled')candidates=nupireResult.value.candidates||candidates;else sourceErrors.push('NUPIRE candidates unavailable: '+nupireResult.reason.message);
    if(!upes){if(legacySummaryResult.status==='fulfilled')summary=legacySummaryResult.value;else sourceErrors.push('Quarantine summary unavailable: '+legacySummaryResult.reason.message);}
    const failures=records.map(r=>'<tr><td class="mono">'+esc(r.id||r.record_id||r.signature||'—')+'</td><td>'+time(r.first_seen||r.created_at||r.first_seen_at)+'</td><td>'+esc(r.node_id||'—')+'<br>'+esc(r.service||r.family||'family unavailable')+'</td><td>'+esc((r.error_codes||[]).join(', ')||r.error||r.failure_class||'No parser matched')+'</td><td>'+badge(r.status||r.state||'unknown')+'</td><td>'+(quarantineIsActionable(r)?'<button class="secondary small" data-action="quarantine-diagnose" data-id="'+esc(r.id||r.record_id)+'" data-service="'+esc(r.service||r.family||'auto')+'">Inspect in Resolution Wizard</button>':'Raw evidence not retained')+'</td></tr>');
    const candidateRows=candidates.map(c=>'<tr><td>'+esc(c.id||c.candidate_id||'—')+'</td><td>'+badge(c.state||c.status||'unknown')+'</td><td>'+(c.metrics||c.validation?esc(pretty(c.metrics||c.validation)):'Not reported')+'</td><td>'+time(c.updated_at||c.last_seen)+'</td></tr>');
    const jsonPanel=(value,caption)=>value==null?empty(caption+' telemetry unavailable.'):('<pre class="json-output">'+esc(pretty(value))+'</pre>');
    const panelContent={
      failures:panel('Failure inbox','UPES health comes from the shared NURSF snapshot. Detailed retained source records are included only when the controller reports them.', '<div class="grid cols-4">'+card('Waiting',summary.waiting??summary.quarantined??summary.total_pending??'Not reported','UPES/controller snapshot')+card('Observed failures',summary.observed_failures??'Not reported','Canonical UPES counter')+card('Queued discoveries',summary.discovery_queued??summary.queued_discoveries??'Not reported','Canonical UPES counter')+card('Unique signatures',summary.unique_failure_signatures??summary.unique_signatures??'Not reported','Canonical UPES counter')+'</div>'+table(['Failure ID / signature','Observed','Node / family','Failure','State','Evidence'],failures,'No failure records are reported in the canonical UPES snapshot. Use Assurance → Guided Parser Repair Studio for retained quarantine records.')),
      decisions:panel('Recovery decision trace','Only controller-reported strategy attempts and outcomes are shown.',jsonPanel(upes?.recovery_decisions||upes?.decisions||upes?.recovery_trace,'Recovery decision')),
      candidates:panel('Parser repair candidates','Persisted UPES/NUPIRE candidate state and validation evidence.',table(['Candidate','State','Validation evidence','Updated'],candidateRows,'No persisted candidates in the canonical UPES snapshot.')),
      knowledge:panel('Verified knowledge','Verified knowledge entries are displayed only when the canonical subsystem reports them.',jsonPanel(upes?.knowledge||upes?.knowledge_entries,'Knowledge base')),
      jobs:panel('Discovery jobs','Bounded worker counters from the canonical subsystem snapshot.',jsonPanel(upes?.discovery_jobs||upes,'Discovery job')),
      health:panel('Parser health and drift','Controller-reported parser health and drift evidence.',jsonPanel(upes?.parser_health||upes,'Parser health')),
      deployments:panel('Deployment and rollback','Candidate deployment state and rollback evidence as reported by the canonical subsystem.',jsonPanel(upes?.deployments||upes?.deployment_history,'Deployment')),
      runtime:panel('Runtime health','Client runtime counters as reported by the canonical subsystem.',jsonPanel(upes?.runtime_health||upes,'Runtime health'))
    };
    renderPage('upes','<div class="dashboard-intro subsystem-hero"><div><p class="eyebrow">UPES · canonical NURSF subsystem</p><h1>Unstoppable Parsing Engine Support</h1><p>Operational UPES metrics come from the shared controller snapshot revision; retained record evidence continues through the existing Assurance and quarantine APIs.</p></div><div class="subsystem-state">'+syncStatusBadge(upes?.status||'unavailable')+'<small>Snapshot revision '+number(canonical.revision)+'</small></div></div><div class="subsystem-nav" data-subsystem-tabs role="tablist" aria-label="UPES views"><button class="tab active" data-subsystem-tab="failures">Live Failure Observatory</button><button class="tab" data-subsystem-tab="decisions">Recovery Decision Trace</button><button class="tab" data-subsystem-tab="candidates">Parser Repair Workshop</button><button class="tab" data-subsystem-tab="knowledge">Knowledge Base</button><button class="tab" data-subsystem-tab="jobs">Discovery Jobs</button><button class="tab" data-subsystem-tab="health">Parser Health &amp; Drift</button><button class="tab" data-subsystem-tab="deployments">Deployment &amp; Rollback</button><button class="tab" data-subsystem-tab="runtime">Runtime Health</button></div>'+(!upes?'<div class="callout">This controller snapshot predates UPES. Showing only legacy quarantine and NUPIRE API data.</div>':'')+sourceErrors.map(message=>'<div class="callout warn">'+esc(message)+'</div>').join('')+Object.entries(panelContent).map(([key,content],i)=>'<section data-subsystem-panel="'+key+'"'+(i?' hidden':'')+'>'+content+'</section>').join(''));
  }

  async function loadNupire() {
    const snapshot=await api('/nupire/snapshot'),items=snapshot.candidates||[];
    const chosen=selectedNupire?await api(`/nupire/candidates/${encodeURIComponent(selectedNupire)}`).catch(()=>null):null;
    const rows=items.map(item=>{const metrics=item.metrics||{};return `<tr><td><button class="ghost small" data-action="nupire-detail" data-id="${esc(item.id)}">${esc(item.id)}</button><br><small>${esc(item.digest||'digest unavailable')}</small></td><td>${badge(item.state||'unknown')}</td><td>${metrics.samples!=null?`${number(metrics.exact_matches)} exact / ${number(metrics.samples)} samples`:'Held-out metrics unavailable'}</td><td>${time(item.updated_at)}</td></tr>`;});
    const lifecycle=[['Shadow','shadow'],['Canary','canary'],['Promotion','promoted'],['Rollback','rolled_back'],['Drift','drifted']].map(([label,status])=>`<div class="card"><div class="metric-label">${label}</div><div class="metric-value">${(items.length?number(items.filter(item=>item.state===status).length):'No candidates')}</div><div class="metric-note">Persisted candidates in this state</div></div>`).join('');
    let detail=empty('Select a persisted NUPIRE candidate to inspect its validation and evidence metadata.');
    if(chosen){const transitions={evaluated:['shadow'],shadow:['approve'],approval:['canary'],canary:['promote'],promoted:['drift','rollback'],drifted:['rollback']}[chosen.state]||[];const controls=transitions.map(action=>`<button class="secondary small" data-action="nupire-transition" data-id="${esc(chosen.id)}" data-value="${action}">${esc(action)}</button>`).join(' ');detail=`<div class="grid cols-2"><div><p><b>${esc(chosen.id||selectedNupire)}</b> · ${badge(chosen.state||'unknown')}</p><p>Artifact digest: <code>${esc(chosen.digest||'unavailable')}</code> · integrity ${chosen.digest_valid?'valid':'unavailable'}</p><p>Held-out evidence: ${chosen.metrics?esc(pretty(chosen.metrics)):'Unavailable'}</p><div class="actions">${controls||'No transitions are available for this state.'}</div></div><div><p>Evidence provenance and lifecycle history</p><pre class="json-output">${esc(pretty({provenance:chosen.provenance,events:chosen.events,approved_by:chosen.approved_by}))}</pre></div></div>`;}
    const artifactForm=chosen?.state==='discovery'?`<form id="nupire-artifact-form" class="form-grid"><input type="hidden" name="id" value="${esc(chosen.id)}"><label class="field wide">Declarative parser artifact JSON<textarea name="artifact" required placeholder='{"pattern_type":"key_value","fields":[]}'></textarea></label><button class="primary" type="submit">Attach candidate artifact</button></form>`:'';
    const evaluationForm=chosen?.state==='candidate'?`<form id="nupire-evaluation-form" class="form-grid"><input type="hidden" name="id" value="${esc(chosen.id)}"><label class="field wide">Measured held-out metrics JSON<textarea name="metrics" required placeholder="Enter measured sample count, exact_matches, field_precision, field_recall, error_rate, ambiguous"></textarea></label><label class="field wide">Held-out evidence IDs<input name="held_out_ids" required placeholder="Comma-separated immutable evidence IDs"></label><button class="primary" type="submit">Submit measured evaluation</button></form>`:'';
    renderPage('nupire',`<div class="dashboard-intro"><div><p class="eyebrow">Evidence-gated parser evolution</p><h1>NUPIRE / EGPE workspace</h1><p><b>NUPIRE</b> means <b>Nexora Universal Parsing Intelligence &amp; Recovery Engine</b>. <b>EGPE</b> means <b>Evidence-Gated Parser Evolution</b>. This operator-triggered workspace persists candidates and held-out evaluation evidence; it is outside the per-record hot path.</p></div><span>${badge(snapshot.evolution_state||'unknown')}</span></div>${panel('Candidate registry','Candidate records come from the durable NUPIRE lifecycle store. Evidence appears after held-out evaluation.',table(['Candidate','Status','Held-out evidence','Updated'],rows,'No NUPIRE candidates are persisted yet.'))}${panel('Create discovery','Record evidence identifiers and provenance before attaching a parser candidate. Raw log text is not stored by this workflow.',`<form id="nupire-discovery-form" class="form-grid"><label class="field">Discovery ID<input name="id" required maxlength="128"></label><label class="field">Source<input name="source" required maxlength="256"></label><label class="field">Generator<input name="generator" required maxlength="128"></label><label class="field">Training evidence IDs<input name="training_ids" required placeholder="Comma-separated IDs"></label><label class="field wide">Held-out evidence IDs<input name="held_out_ids" required placeholder="Independent comma-separated IDs"></label><button class="primary" type="submit">Create discovery record</button></form>`)}${panel('Candidate detail','Scores and evidence are shown only when present in the persisted record.',detail)}${artifactForm?panel('Submit candidate artifact','The controller accepts declarative JSON objects only; code and raw events are rejected.',artifactForm):''}${evaluationForm?panel('Held-out evaluation','Submit measurements produced by the assurance workflow and the immutable held-out evidence identifiers.',evaluationForm):''}${panel('Evolution lifecycle','Current lifecycle status is read from each persisted candidate.',`<div class="grid cols-3">${lifecycle}</div>`)}`);
  }

  async function submit(form) {
    const data=formObject(form);
    switch(form.getAttribute('id')){
      case 'nupire-discovery-form': { const ids=value=>value.split(',').map(item=>item.trim()).filter(Boolean);const body={id:data.id,source:data.source,generator:data.generator,training_ids:ids(data.training_ids),held_out_ids:ids(data.held_out_ids)};if(!body.training_ids.length||!body.held_out_ids.length)throw new Error('Provide training and independent held-out evidence IDs');await api('/nupire/discoveries',{method:'POST',body});selectedNupire=data.id;notify('NUPIRE discovery record persisted.');await navigate('nupire');break; }
      case 'nupire-artifact-form': { const artifact=parseJSON(data.artifact,'Declarative artifact JSON');if(!artifact||Array.isArray(artifact)||typeof artifact!=='object')throw new Error('Candidate artifact must be a JSON object');await api('/nupire/candidates/'+encodeURIComponent(data.id)+'/artifact',{method:'POST',body:{artifact}});selectedNupire=data.id;notify('Declarative parser candidate persisted.');await navigate('nupire');break; }
      case 'nupire-evaluation-form': { const metrics=parseJSON(data.metrics,'Measured metrics JSON'),held_out_ids=data.held_out_ids.split(',').map(item=>item.trim()).filter(Boolean);if(!held_out_ids.length)throw new Error('Provide the held-out evidence IDs used for these measurements');await api('/nupire/candidates/'+encodeURIComponent(data.id)+'/evaluation',{method:'POST',body:{metrics,held_out_ids}});selectedNupire=data.id;notify('Held-out evaluation recorded by NUPIRE policy.');await navigate('nupire');break; }
      case 'parser-workspace-form': { const parser=parseJSON(data.parser,'Parser artifact JSON'),cases=parseJSON(data.cases,'Validation cases JSON');if(!Array.isArray(cases)||!cases.length)throw new Error('Provide at least one validation case');const body={name:data.name,version:data.version,parser,cases,source_quarantine_record_id:state.quarantineDiagnosis?.id||''};const path=data.edit_id?`/assurance/parsers/${encodeURIComponent(data.edit_id)}`:'/assurance/parsers';const method=data.edit_id?'PUT':'POST';const result=await api(path,{method,body});notify(`Parser draft saved · validation ${result.validation?.score??'pending'}%. Independent approval remains required.`);await navigate('parser-workspace');break; }
      case 'parser-helper-form':
      case 'gprs-parser-helper-form': { const parser=parseJSON(data.parser,'Parser JSON'),cases=parseJSON(data.cases,'Validation cases JSON');if(!Array.isArray(cases)||!cases.length)throw new Error('Provide validation cases');const payload={name:data.name,version:data.version,parser,cases,source_quarantine_record_id:state.quarantineDiagnosis?.id||''},editId=data.edit_id.trim();const result=editId?await api(`/assurance/parsers/${encodeURIComponent(editId)}`,{method:'PUT',body:payload}):await api('/assurance/parsers',{method:'POST',body:payload});if(state.quarantineDiagnosis&&result.id)state.quarantineDiagnosis.parserId=result.id;if(state.gprsManual&&result.id){state.gprsManual.parserId=result.id;state.gprsManual.verification=null;}toast(editId?'Parser revision saved':'Parser draft created');const target=state.page==='quarantine-wizard'?'quarantine-wizard':'assurance';await navigate(target);page.querySelector('[data-assurance-tab=gprs]')?.click();page.querySelector('#assurance-panel-gprs')?.scrollIntoView({behavior:'smooth',block:'start'});break; }    case 'assurance-form': { const progress=notify('Running production parser assurance…','progress');try{const request={raw:data.raw,service:selectedAssuranceService(data),parser_id:data.parser_id==='auto'?'':data.parser_id,source:data.source,desired:parseJSON(data.desired,'Desired JSON'),field_mapping:parseJSON(data.mapping,'Field mapping')};const result=await api('/assurance/run',{method:'POST',body:{...request,preserve_source:true}});state.lastAssurance={request,result};if(data.retain_test_sample==='true')await retainTestSample({...request,preserve_source:true},'Assurance Lab');const helper=result.parser_creation_helper,form=el('parser-helper-form');if(helper&&form){const normalized=normalizeParserCreationHelper(helper,request);form.elements.samples.value=normalized.normalized;form.elements.name.value=`${String(helper.requested_family||request.service||'custom').replace(/[^A-Za-z0-9._-]/g,'-')}-draft`;form.elements.parser.value=pretty(normalized.parser);form.elements.cases.value=pretty(normalized.normalized.split('\n').filter(line=>line.trim()).map(raw=>({raw,service:request.service,desired:{}})));const overview=form.querySelector('[data-parser-overview]')||el('parser-helper-overview');if(overview)overview.innerHTML=`<dl><dt>Parser family</dt><dd>${esc(helper.requested_family||request.service)}</dd><dt>Detected format</dt><dd>${esc(helper.detected_format||'—')} · ${Number.isFinite(Number(helper.sample_lines))?Number(helper.sample_lines):normalized.sampleLines} sample lines</dd><dt>Selected parser / strategy</dt><dd>${esc(helper.selected_parser||'automatic')} / ${esc(helper.selected_strategy||'—')}</dd><dt>Match scope</dt><dd><code>${esc(pretty(normalized.scope))}</code></dd><dt>Regex</dt><dd><code>${esc(helper.suggested_regex||'—')}</code></dd><dt>Fields / projection</dt><dd>${esc(normalized.fieldNames.join(', '))} / ${esc(normalized.projection.join(', '))}</dd><dt>Template</dt><dd><code>${esc(pretty(normalized.template))}</code></dd></dl>`;renderParserHelperDetails(normalized.parser,form);}if(state.page==='assurance'){el('assurance-result').innerHTML=renderAssurance(result);page.querySelector('[data-action=clear-assurance]').disabled=false;page.querySelector('[data-action=download-assurance]').disabled=false;applyPermissions();}progress.done(result.passed?'Assurance passed':'Assurance completed with findings');if(!result.passed){notify('Deterministic failure recorded. Review the result, then explicitly request AI remediation if needed.',true);}}catch(error){progress.fail(error.message);throw error;}break; }
      case 'assurance-upload-form': { const file=form.querySelector('[name=file]').files[0];if(!file)throw new Error('Choose a log, text, or archive sample');if(state.proxy&&file.size>4_000_000){state.uploadError='Hosted upload limit is 4 MB. Use the controller WebView for this sample.';const status=el('assurance-upload-status');if(status)status.outerHTML=renderUploadStatus();throw new Error(state.uploadError);}const service=selectedAssuranceService(data),parser_id=data.parser_id==='auto'?'':data.parser_id,retain=data.retain==='true',body=new FormData();body.append('file',file,file.name);body.append('service',service);body.append('parser_id',parser_id);body.append('retain',String(retain));state.uploadError='';const status=el('assurance-upload-status');if(status)status.innerHTML='<div class="callout">Upload in progress. The controller validates file type, archive contents and configured limits.</div>';const progress=notify(`Uploading ${file.name} for ${retain?'encrypted retention':'transient analysis'}…`,'progress');try{const result=await api('/assurance/upload',{method:'POST',body});state.uploadReport={filename:file.name,type:result.mime_type||result.content_type||file.type||file.name.split('.').pop(),size:file.size,response:result};state.uploadError='';state.lastAssurance={request:{raw:'',service,parser_id,source:file.name,desired:{},field_mapping:{}},result:result.result||result.report?.result||result};notify(result.message||`Sample analyzed · ${result.report?.files_analyzed??result.files_analyzed??1} file(s)`);await navigate('assurance');}catch(error){const archive=/\.(zip|tar|gz|tgz)$/i.test(file.name),routeMissing=/404|not found|unknown route|not implemented/i.test(error.message);if(!retain&&!archive&&routeMissing){try{const raw=await file.text();const request={raw,service,parser_id,source:file.name,desired:{},field_mapping:{}};const result=await api('/assurance/run',{method:'POST',body:{...request,preserve_source:true}});state.uploadReport={filename:file.name,type:file.type||file.name.split('.').pop(),size:file.size,response:{retained:false,storage_status:'Not retained; analysis was transient'}};state.lastAssurance={request,result};notify('File analyzed as a single sample; this controller does not yet support server-side archive analysis.');await navigate('assurance');break;}catch(fallbackError){state.uploadError=fallbackError.message;progress.fail(fallbackError.message);throw fallbackError;}}state.uploadError=retain&&routeMissing?'This controller does not support encrypted Assurance upload retention yet. No fixture was saved. Upgrade the controller or retry with retention turned off.':error.message;const errorStatus=el('assurance-upload-status');if(errorStatus)errorStatus.outerHTML=renderUploadStatus();progress.fail(state.uploadError);throw error;}break; }
      case 'assurance-corpus-form': { const parsedCases=parseJSON(data.cases,'Cases JSON');if(!Array.isArray(parsedCases)||!parsedCases.length)throw new Error('Provide at least one corpus case');const defaultService=selectedAssuranceService(data),defaultParser=data.parser_id==='auto'?'':data.parser_id;const cases=parsedCases.map(item=>({...item,service:item.service||defaultService,parser_id:item.parser_id||defaultParser}));const progress=notify(`Validating ${cases.length} corpus cases…`,'progress');try{const result=await api('/assurance/corpus',{method:'POST',body:{cases:cases.map(item=>({...item,preserve_source:true}))}});notify(`Corpus completed: ${result.passed ?? result.passed_count ?? 'see report'} passed`);el('assurance-corpus-result').innerHTML=panel('Corpus result','Mixed-format corpus execution report.',`<pre class="json-output">${esc(pretty(result))}</pre>`);}catch(error){progress.fail(error.message);throw error;}break; }
      case 'assurance-parser-form': { const cases=parseJSON(data.cases,'Validation cases JSON');const parser=parseJSON(data.parser,'Parser JSON');const progress=notify('Creating and validating parser draft…','progress');try{const result=await api('/assurance/parsers',{method:'POST',body:{name:data.name,version:data.version,parser,cases,source_quarantine_record_id:state.quarantineDiagnosis?.id||''}});notify(`Draft created · validation ${result.validation?.score ?? 'pending'}%`);await navigate('assurance');}catch(error){progress.fail(error.message);throw error;}break; }
      case 'assurance-run-form': { const [plan_id,revision]=data.plan.split('@');if(!plan_id||!Number(revision))throw new Error('Choose an exact saved plan revision');const run=await api('/assurance/runs',{method:'POST',body:{plan_id,plan_revision:Number(revision),case_id:data.case_id,scenario:data.scenario||'parser-regression'}});notify(`${run.scenario||'Assurance'} run ${run.id} accepted: ${run.state}`);await navigate('assurance');page.querySelector('[data-assurance-tab=runs]')?.click();break; }
      case 'napi-synthesis-form': { const discovery=parseJSON(data.discovery,'Discovery samples JSON'),held_out=parseJSON(data.held_out,'Held-out samples JSON'),negative=parseJSON(data.negative,'Negative samples JSON');if(!Array.isArray(discovery)||discovery.length<2)throw new Error('Provide at least two discovery samples');if(!Array.isArray(held_out)||!held_out.length)throw new Error('Provide at least one independent held-out sample');if(!Array.isArray(negative))throw new Error('Negative samples must be a JSON array');const progress=notify('Synthesizing bounded parser candidates…','progress');try{const result=await api('/assurance/napi/synthesize',{method:'POST',body:{id:data.id,name:data.name,version:data.version,candidate_limit:Number(data.candidate_limit||4),discovery,held_out,negative}});state.lastNAPI=result;progress.done(`Draft ${result.id} revision ${result.revision} created; independent approval remains required.`);await navigate('assurance');page.querySelector('[data-assurance-tab=synthesis]')?.click();}catch(error){progress.fail(error.message);throw error;}break; }
      case 'assurance-plan-form': { const plan=parseJSON(data.plan,'Test plan JSON'),editId=data.edit_id.trim(),id=data.id.trim();const result=await api(editId?`/assurance/plans/${encodeURIComponent(editId)}`:'/assurance/plans',{method:editId?'PUT':'POST',body:{id,name:data.name,version:data.version,plan}});notify(`Test plan ${result.id||id} revision ${result.revision||'saved'} stored`);await navigate('assurance');page.querySelector('[data-assurance-tab=plans]')?.click();break; }
      case 'gprs-replay-form': {const selected=[...form.elements.record_ids.selectedOptions],ids=selected.map(option=>option.value),nodes=[...new Set(selected.map(option=>option.dataset.node).filter(Boolean))],exactAck=selected.length>0&&selected.every(option=>option.dataset.ack==='true');if(!ids.length)throw new Error('Select at least one replay-eligible record');if(nodes.length!==1)throw new Error('Selected records must belong to one target node');if(!exactAck)throw new Error('Every selected record must have an exact current bundle acknowledgement');const node=nodes[0];if(data.node_id!==node)throw new Error('Target node must match the selected records');const target=state.page==='quarantine-wizard'?'quarantine-wizard':'assurance';setReplayStatus(form,'Submitting the targeted replay request. Live telemetry continues in the background.','success');try{const result=await api('/quarantine/reprocess',{method:'POST',body:{record_ids:ids,node_id:node,expected_bundle_hash:form.dataset.bundleHash}});state.qrwReplayAcceptedIds=[...new Set([...state.qrwReplayAcceptedIds,...ids])];state.qrwStep=9;notify(result.message||'Replay proposal accepted; per-record parser confirmation is pending.');await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){setReplayStatus(form,replayGateMessage(error,exactAck)||`Replay request was not accepted: ${error.message}. Live telemetry remains active.`,'warn');}break; }
      case 'quarantine-reprocess-form': { const record_ids=data.record_ids.split(',').map(value=>value.trim()).filter(Boolean);if(!record_ids.length)throw new Error('Enter at least one quarantine record ID');setReplayStatus(form,`Requesting replay for ${record_ids.length} record(s). Live telemetry continues in the background.`,'success');try{const result=await api('/quarantine/reprocess',{method:'POST',body:{record_ids,node_id:data.node_id.trim()}});notify(result.message||`Replay request ${result.rollout_id||result.proposal_id||'queued'}; client confirmation is pending.`);await navigate('unknowns');}catch(error){setReplayStatus(form,replayGateMessage(error)||`Replay request was not accepted: ${error.message}. Live telemetry remains active.`,'warn');}break; }
      case 'assurance-save-form': if(!state.lastAssurance)throw new Error('Run assurance first');await api('/assurance/cases',{method:'POST',body:{id:data.id,name:data.name,description:data.description,retain:true,case:state.lastAssurance.request}});toast('Encrypted fixture retained');await navigate('assurance');page.querySelector('[data-assurance-tab=fixtures]')?.click();break;
      case 'fleet-desired-form': await api(`/fleet/${encodeURIComponent(data.node)}/desired`,{method:'PUT',body:{pack_hash:data.pack_hash,config_hash:data.config_hash,release_hash:data.release_hash}});toast('Desired state updated');await navigate('fleet');break;
      case 'parser-form': { const regressionCases=parseJSON(data.regression_cases,'Reviewed regression cases JSON');if(!Array.isArray(regressionCases)||!regressionCases.length)throw new Error('Add at least one reviewed regression case before creating this parser pack');const payload={id:data.id,name:data.name,version:data.version,priority:Number(data.priority),rules:parseJSON(data.rules,'Rules JSON'),regression_cases:regressionCases};if(form.dataset.editId){await api(`/parser-packs/${encodeURIComponent(form.dataset.editId)}`,{method:'PUT',body:payload});toast('Parser draft revision saved');}else{await api('/parser-packs',{method:'POST',body:payload});toast('Parser draft created');}await navigate('parsers');break; }
      case 'migration-form': { const file=form.querySelector('[name=source_file]').files[0];const filename=file?.name||data.filename;const content=file?await file.text():data.content;if(!content.trim())throw new Error('Choose a parser source file or paste its contents');await api('/migration/import',{method:'POST',body:{filename,source_type:data.source_type,content,create_draft:data.create_draft==='true'}});toast('Source analyzed without execution');await navigate('migration');break; }
      case 'input-profile-form': { const config=parseJSON(data.config,'Input definitions JSON');await api('/config/input-paths',{method:'POST',body:{id:data.id,name:data.name,config}});toast('Input profile revision saved');await navigate('input-paths');break; }
      case 'input-rollout-form': { const formData=new FormData(form),profileIds=formData.getAll('profile_id').filter(Boolean),allEligible=formData.has('all_eligible'),targets=(data.targets||'').split(',').map(v=>v.trim()).filter(Boolean);if(!profileIds.length)throw new Error('Select one or more published path profiles');if(!allEligible&&!targets.length)throw new Error('Select clients or choose every eligible online client');for(const profileId of profileIds)await api(`/config/input-paths/${encodeURIComponent(profileId)}/rollout`,{method:'POST',body:{node_ids:allEligible?[]:targets,...(allEligible&&formData.has('include_offline')?{include_offline:true}:{})}});state.inputHistoryProfile=profileIds[0];toast(`${profileIds.length} input-path rollout proposal${profileIds.length===1?'':'s'} await independent approval`);refreshApprovalBadge();await navigate('input-paths');break; }
      case 'config-form': { const [path]=CONFIG[form.dataset.kind], config=parseJSON(data.config,'Configuration JSON');if(form.dataset.kind==='knownkeys'){if(!config||Array.isArray(config)||typeof config!=='object')throw new Error('Advanced metadata must be a JSON object.');config.keys=data.known_keys.split(/\r?\n/).map(key=>key.trim()).filter(Boolean);}await api(path,{method:'POST',body:{id:data.id,name:data.name,config}});toast('Configuration revision saved');await navigate(form.dataset.kind);break; }
      case 'output-profile-form': { const config=buildOutputProfile(form);await api('/config/output-destinations',{method:'POST',body:{id:data.id,name:data.name,config}});state.outputPreview=config;notify('Output profile validated and saved as a Supabase draft. Submit, approve, publish, then create a node rollout.');await navigate('outputs');break; }
      case 'output-rollout-form': { const profile=await api(`/config/output-destinations/${encodeURIComponent(data.profile_id)}`);if(profile.state!=='active')throw new Error('Only a published endpoint profile can be rolled out');const outputs=profile.payload?.config?.outputs||[];await api('/rollouts',{method:'POST',body:{name:`Endpoint profile ${profile.payload?.name||data.profile_id}`,type:'config_push',target_nodes:data.targets.split(',').map(v=>v.trim()).filter(Boolean),payload:{config_patch:{outputs}},profile_id:data.profile_id,profile_revision:profile.revision}});toast('Endpoint rollout awaits independent approval');await navigate('rollouts');break; }
      case 'managed-input-file-form': {
        const all=data.scope==='all',select=form.elements.targets;
        const targets=[...(all?select.options:select.selectedOptions)].map(option=>option.value);
        if(!targets.length)throw new Error('Select at least one compatible client');
        const action=data.action,source=data.source.trim(),destination=action==='copy'?data.destination.trim():'';
        if(action==='copy'&&source===destination)throw new Error('Choose a different destination filename');
        const name=action==='copy'?`Copy managed log file ${source} to ${destination}`:`Delete managed log file ${source}`;
        await api('/rollouts',{method:'POST',body:{name,type:'managed_input_file',target_nodes:targets,payload:{action,source,destination}}});
        toast('Managed file operation submitted for approval');refreshApprovalBadge();await navigate('clients');break;
      }
      case 'client-action-form': {
        const nodeIds=[...form.elements.targets.selectedOptions].map(option=>option.value),all=data.scope==='all';
        if(!all&&!nodeIds.length)throw new Error('Select at least one enrolled client');
        if(data.type==='parser_sync'){
          const bundleHash=(await api('/parser-sync')).bundle?.desired_hash;
          if(!bundleHash)throw new Error('The current published parser bundle is unavailable');
          await api('/parser-sync/rollout',{method:'POST',body:{target_mode:all?'all_eligible':'selected',node_ids:all?[]:nodeIds,bundle_hash:bundleHash}});
        }else{
          const targetNodes=all?[...form.elements.targets.options].map(option=>option.value):nodeIds;
          if(!targetNodes.length)throw new Error('No enrolled client is available');
          await api('/rollouts',{method:'POST',body:{name:'Restart parsing agent',type:'restart',target_nodes:targetNodes,payload:{}}});
        }
        toast('Client action submitted for approval');refreshApprovalBadge();await navigate('clients');break;
      }
      case 'rollout-form': await api('/rollouts',{method:'POST',body:{name:data.name,type:data.type,target_nodes:data.targets.split(',').map(v=>v.trim()).filter(Boolean),payload:parseJSON(data.payload,'Payload JSON')}});toast('Rollout awaiting independent approval');await navigate('rollouts');break;
      case 'artifact-form': { const file=form.querySelector('[name=file]').files[0];if(!file)throw new Error('Choose an artifact');if(state.proxy && file.size>4_000_000)throw new Error('This artifact exceeds the hosted upload limit. Open the controller URL to upload the signed artifact, then manage its rollout here.');const digest=Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256',await file.arrayBuffer()))).map(v=>v.toString(16).padStart(2,'0')).join('');const body=new FormData();body.append('file',file,file.name);await api(`/artifacts/upload?type=${encodeURIComponent(data.type)}`,{method:'POST',body,headers:{'X-Artifact-SHA256':digest,'X-Artifact-Signature':data.signature}});toast('Signed artifact verified and stored');await navigate('artifacts');break; }
      case 'upgrade-form': await api('/upgrades',{method:'POST',body:{artifact_id:data.artifact_id,name:data.name,target_nodes:data.targets.split(',').map(v=>v.trim()).filter(Boolean)}});toast('Upgrade awaiting approval');await navigate('artifacts');break;
      case 'user-form': await api('/users',{method:'POST',body:data});toast('User created');await navigate('security');break;
      case 'notification-config-form': { const events=[...form.querySelectorAll('[name=events]:checked')].map(input=>input.value),recipients=(data.recipients||'').split(',').map(item=>item.trim()).filter(Boolean);const body={channels:{in_app:{enabled:true},email:{enabled:data.email_enabled==='true',server_env:data.email_server_env.trim(),sender_env:data.email_sender_env.trim(),recipient_env:data.email_recipient_env.trim(),username_env:data.email_username_env.trim(),password_env:data.email_password_env.trim()},pagerduty:{enabled:data.pagerduty_enabled==='true',secret_env:data.pagerduty_secret_env.trim()}},admin_fallback:data.admin_fallback==='true',recipients,events};const result=await api('/notifications/config',{method:'PUT',body});notify(`Notification policy saved · SMTP ${result.channels?.email?.status||'updated'} · PagerDuty ${result.channels?.pagerduty?.status||'updated'}`);await loadNotifications();break; }
      case 'ai-provider-form': { const body={id:data.id,name:data.name,provider:data.provider,model:data.model,endpoint:data.endpoint,credential_source:data.credential_source,credential_ref:data.credential_ref,priority:Number(data.priority||0),fallback_group:data.fallback_group,timeout_seconds:Number(data.timeout_seconds||15),max_response_bytes:Number(data.max_response_bytes||1048576),enabled:data.enabled==='true'};if(data.api_key&&data.credential_source==='encrypted_inline')body.api_key=data.api_key;const progress=notify('Saving AI provider settings securely…','progress');try{await api('/admin/ai-provider',{method:'PUT',body});form.querySelector('[name=api_key]').value='';state.aiSelectedProviderId=data.id;progress.done('AI provider settings saved; inline secret input was cleared');await navigate('ai');}catch(error){progress.fail(error.message);throw error;}break; }
    }
  }

  async function action(button) {
    const id=button.dataset.id, value=button.dataset.value;
    if(button.hasAttribute?.('data-map-sample')&&!can('assurance:run'))throw new Error('Mapping requires Assurance run access.');
    if(button.dataset.action==='parser-helper-validate'){
      const progress=notify('Validating the generated parser against its exact evidence…','progress');
      try{const result=await qrwValidateCandidate(button.closest('form'));const validation=result.validation||{};result.passed===true?progress.done(`Candidate validation passed ${number(validation.passed)}/${number(validation.total)} exact case(s).`):progress.fail(`Candidate validation failed ${number(validation.failed)}/${number(validation.total)} case(s). Review the reported diff before saving.`);}
      catch(error){progress.fail(error.message);throw error;}
      return;
    }
    if(button.dataset.action==='gprs-preview'){
      const form=page.querySelector('#gprs-parser-helper-form');
      if(!form?.elements.parser?.value?.trim()||!form?.elements.cases?.value?.trim())throw new Error('Generate the parser candidate in Step 5 before testing. QRW never substitutes the currently live parser for the candidate under review.');
      const progress=notify('Running the generated candidate through production parser validation…','progress');
      try{const result=await qrwValidateCandidate(form,{recordWorkflow:true}),validation=result.validation||{},output=el('gprs-preview'),fallback=el('gprs-ai-fallback');if(output)output.textContent=pretty(state.gprsPreview);if(fallback)fallback.innerHTML=result.passed===false?'<b>The generated candidate did not match the exact expected JSON.</b> Review the case-level diff and adjust the mapping. AI is available only as a final, review-only fallback.<div class="actions"><button class="primary" type="button" data-action="request-ai-proposal">Ask Nexora AI to diagnose and propose a fix</button><button type="button" class="secondary" data-action="workspace-ai-auto" data-tab="Assurance diagnosis">Auto Fix with Nexora AI</button></div>':'<b>Candidate preview passed every exact case.</b> Continue to the embedded review and publication step; no parser was activated by this test.';setupQRWStepPanels();result.passed===true?progress.done(`Candidate preview passed ${number(validation.passed)}/${number(validation.total)} exact case(s).`):progress.fail(`Candidate preview failed ${number(validation.failed)}/${number(validation.total)} case(s).`);}
      catch(error){progress.fail(error.message);throw error;}
      return;
    }
    switch(button.dataset.action){
      case 'nfi-cancel-analysis': {nfiAnalysisController?.abort();qrwLivePreviewController?.abort();clearTimeout(nfiLiveTimer);const status=el('nfi-live-state');if(status)status.textContent='Analysis cancelled locally; no draft was published. Editing remains available.';break;}
      case 'nfi-fleet-status': {const result=await api('/parser-sync'),assets=result.assets||[];el('nfi-fleet-status').innerHTML=assets.length?assets.map(asset=>`<div class="nfi-profile"><b>${esc(asset.asset_id||asset.id)}</b> ${badge(asset.status||asset.runtime_sync?.status||'reported')}<pre>${esc(pretty({desired:asset.desired,clients:asset.clients,acknowledgement:asset.acknowledgement}))}</pre><button type="button" class="secondary small" data-action="parser-sync-verify" data-id="${esc(asset.asset_id||asset.id)}" data-class="${esc(asset.asset_class||'parser_pack')}">Verify reported state</button><button type="button" class="secondary small" data-action="parser-sync-retry" data-id="${esc(asset.asset_id||asset.id)}" data-class="${esc(asset.asset_class||'parser_pack')}">Request approval-gated rollout retry</button></div>`).join(''):`<pre>${esc(pretty(result))}</pre><p>No mapping-specific client acknowledgement was supplied; no deployment success is inferred.</p>`;break;}
      case 'nfi-mode': {state.nfiSenseMode=value;state.nfiGuideStep=2;el('nfi-mode-status').textContent=value==='manual'?'Manual Sense: select raw characters/tokens, Add Custom Field, or choose any native extraction strategy. Existing mappings are preserved.':'Suggest & Sense: use Suggest from raw log, then review each proposed mapping. Existing custom fields are preserved.';nfiGuideRefresh();break;}
      case 'nfi-expert': {state.nfiExpert=!state.nfiExpert;el('nfi-guide-steps').hidden=state.nfiExpert;button.textContent=state.nfiExpert?'Switch to Guided Builder':'Expert mode';break;}
      case 'nfi-guide-step': {state.nfiGuideStep=Number(button.dataset.index);nfiGuideRefresh();const index=state.nfiGuideStep,target=index===0?el('nfi-raw'):index===1?el('nfi-mode-status'):index<10?el('qrw-field-rows'):index===10?el('nfi-controller-result'):index<13?el('nfi-parser-json'):index<15?el('nfi-candidate-status'):page.querySelector('[data-action="assurance-parser-action"]');target?.scrollIntoView({block:'center'});if(target?.matches('input,textarea,button'))target.focus();break;}
      case 'nfi-help': {el('nfi-help-content').innerHTML=`<h3>${esc(button.dataset.helpTitle)}</h3><p>${esc(button.dataset.helpDescription)}</p><ol><li>Select the raw evidence and choose the matching extraction method.</li><li>Configure this option using the example: <code>${esc(button.dataset.helpExample)}</code></li><li>Inspect the live value and source trace. Resolve missing or invalid values explicitly.</li><li>Verify the exact plan on the controller, validate the full parser, then save and submit for authorized review.</li></ol><p>Positions are 1-based and inclusive unless the control explicitly says otherwise. Go RE2 rejects lookbehind and backreferences. Native verification remains authoritative.</p>`;el('nfi-help-dialog').showModal();break;}
      case 'nfi-help-close': {el('nfi-help-dialog').close();break;}
      case 'nfi-suggestion-accept': {const field=state.qrwFields.find(f=>f.id===id);if(!field)throw new Error('Suggestion no longer present');const result=qrwExtract({...field,confirmed:true},qrwRaw(),state.nfiDesired||{});if(result.error)throw new Error(result.error);field.confirmed=true;field.accepted=true;field.confidence='operator_accepted';el('qrw-field-rows').innerHTML=renderQRWFieldRows();nfiScheduleLive();break;}
      case 'nfi-suggestion-edit': {const row=[...page.querySelectorAll('[data-qrw-row]')].find(row=>row.dataset.qrwRow===id);row?.querySelector('[data-qrw-key="name"]')?.focus();break;}
      case 'nfi-suggestion-regenerate': {const index=state.qrwFields.findIndex(f=>f.id===id),old=state.qrwFields[index];if(!old)throw new Error('Suggestion no longer present');const candidates=qrwSuggestedFields(qrwRaw()),found=candidates.find(f=>f.method===old.method&&f.source===old.source&&String(f.part)===String(old.part));if(!found)throw new Error('This source is no longer sensed; edit its extraction parameters manually.');state.qrwFields[index]=qrwField({...found,name:old.name,label:old.label,accepted:false});el('qrw-field-rows').innerHTML=renderQRWFieldRows();nfiScheduleLive();break;}
      case 'nfi-accept-valid': {let count=0;for(const field of state.qrwFields){if(field.method==='delimiter'&&!field.confirmed||!['structural_suggestion','high','high_confidence'].includes(field.confidence))continue;const result=qrwExtract(field,qrwRaw(),state.nfiDesired||{});if(!result.error){field.accepted=true;field.confidence='operator_accepted';count++;}}el('qrw-field-rows').innerHTML=renderQRWFieldRows();toast(`${count} valid structural suggestions accepted. Ambiguous and low-confidence mappings remain for individual review.`);nfiScheduleLive();break;}
      case 'nfi-highlight': {const field=state.qrwFields.find(f=>f.id===id);if(field)nfiHighlight(field);break;}
      case 'nfi-bulk-tokens': {const spec=prompt('Token indexes or inclusive ranges, e.g. 1,3,5-7. Each selected token becomes a separately named custom field.');if(spec===null)break;const indices=[];for(const part of spec.split(',')){const match=/^\s*(\d+)(?:\s*-\s*(\d+))?\s*$/.exec(part);if(!match)throw new Error('Use positive token indexes or ranges such as 1,3,5-7.');const start=Number(match[1]),end=Number(match[2]||match[1]);if(start<1||end<start||end>4096||end-start>127)throw new Error('Use a bounded 1–4096 inclusive token range.');for(let i=start;i<=end;i++)if(!indices.includes(i))indices.push(i);}if(state.qrwFields.length+indices.length>128)throw new Error('A mapping supports up to 128 practical custom fields.');const tokens=qrwTokens(qrwRaw());for(const index of indices){if(!tokens[index-1])throw new Error(`Token ${index} is absent.`);}state.qrwFields.push(...indices.map(index=>qrwField({name:`selected_token_${index}`,label:`Selected token ${index}`,policy:'required',method:'position',source:String(index)})));state.qrwFields=qrwNormalizeSuggestions(state.qrwFields);el('qrw-field-rows').innerHTML=renderQRWFieldRows();nfiScheduleLive();break;}
      case 'nfi-split-outputs': {const sourceName=prompt('Existing field name to split into separately named outputs:');if(sourceName===null)break;const source=state.qrwFields.find(f=>f.name===sourceName);if(!source)throw new Error('Choose an existing mapped field.');const delimiter=prompt('Literal delimiter, e.g. ; or |');if(delimiter===null)break;if(!delimiter)throw new Error('Delimiter must be nonempty.');const names=prompt('Output names in order, comma-separated, e.g. log_level,event_type');if(names===null)break;const outputs=names.split(',').map(v=>v.trim()).filter(Boolean);if(!outputs.length||state.qrwFields.length+outputs.length>128)throw new Error('Choose 1–128 bounded output fields.');if(outputs.some(name=>state.qrwFields.some(f=>f.name===name))||new Set(outputs).size!==outputs.length)throw new Error('Output names must be unique; rename collisions explicitly.');const start=state.qrwFields.indexOf(source)+1;state.qrwFields.splice(start,0,...outputs.map((name,index)=>qrwField({name,label:name,method:'field',source:sourceName,policy:'required',operations:[{op:'split',delimiter,part:index+1}]})));el('qrw-field-rows').innerHTML=renderQRWFieldRows();nfiScheduleLive();break;}
      case 'nfi-validate-candidate': {const parser=parseJSON(el('nfi-parser-json').value,'Complete parser'),cases=parseJSON(el('nfi-cases-json').value,'Exact cases'),result=await nfiAnalysisRequest('/assurance/parser-preview',{parser,cases});el('nfi-candidate-status').textContent=pretty(result);if(result.passed!==true)throw new Error('Complete parser validation failed; inspect the authoritative result.');state.nfiParser=parser;state.nfiCases=cases;state.nfiCandidateValidated=JSON.stringify({parser,cases});nfiGuideRefresh();break;}

      case 'input-profile-json': {const doc=await api(`/config/input-paths/${encodeURIComponent(id)}`);const detail=el('input-profile-detail');if(detail)detail.innerHTML=panel(`${doc.payload?.name||doc.id} JSON`,`Revision ${number(doc.revision)} · ${esc(doc.state||'unknown')}`,`<pre class="json-output">${esc(pretty(doc))}</pre><div class="actions"><button class="secondary small" type="button" data-action="input-profile-download" data-id="${esc(id)}">Download JSON</button></div>`);break;}
      case 'input-profile-download': {const doc=await api(`/config/input-paths/${encodeURIComponent(id)}`);downloadJSON(`${id}-input-profile.json`,doc);break;}
      case 'dashboard-history-apply': {const from=page.querySelector('#dashboard-from')?.value||'',to=page.querySelector('#dashboard-to')?.value||'';state.dashboardFrom=from;state.dashboardTo=to;page.dataset.dirty='false';await applyDashboardHistory({force:true});break;}
      case 'dashboard-history-download': {if(!state.dashboardHistory?.points)throw new Error('Historical measurements are not loaded yet.');downloadJSON(`nli-command-center-${state.dashboardRange}.json`,{range:state.dashboardRange,from:state.dashboardHistory.from,to:state.dashboardHistory.to,node_id:state.dashboardClient==='all'?undefined:state.dashboardClient,points:state.dashboardHistory.points,canonical_history:state.dashboardHistory.canonical_history});break;}
      case 'reload': {const restoreGprs=state.page==='assurance'&&page.querySelector('[data-assurance-panel=gprs]')?.hidden===false;await navigate(state.page);if(restoreGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();break;}
      case 'notification-ack':
      case 'notification-state': await updateNotificationState(id,value||'read');toast(`Notification ${value==='snooze'?'set aside for one hour':value||'marked read'}`);break;
      case 'notification-dismiss': await api(`/notifications/audit-${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast('Notification removed from this user’s active feed');await refreshNotificationBell();await loadNotifications();break;
      case 'nupire-detail': selectedNupire=id; await loadNupire(); break;
      case 'nupire-transition': { const result=await api(`/nupire/candidates/${encodeURIComponent(id)}/actions/${encodeURIComponent(value)}`,{method:'POST',body:{}});notify(`NUPIRE candidate ${id} moved to ${result.state||value}.`);selectedNupire=id;await loadNupire();break; }
      case 'monitoring-tab': state.monitoringView=button.dataset.monitoringTab||'overview';setupMonitoringTabs();break;
      case 'monitoring-pause': state.monitoringPaused=!state.monitoringPaused;renderMonitoring();break;
      case 'monitoring-reset': state.monitoringRange='15m';state.monitoringNode='all';state.monitoringSource='all';state.monitoringSearch='';state.monitoringView='overview';renderMonitoring();break;
      case 'download-output-config': if(state.outputPreview)downloadJSON('nli-output-destinations.json',state.outputPreview); break;
      case 'output-config-test': { const form=page.querySelector('#output-profile-form');if(!form)throw new Error('Endpoint configuration editor is unavailable');const config=buildOutputProfile(form);state.outputPreview=config;const preview=el('output-profile-preview'),result=el('output-config-test-result');if(preview)preview.textContent=pretty(config);if(result)result.textContent='Local profile validation passed. Only normal and warning routes are enabled for remote destinations; quarantine and unknown stay local and retryable. Network connectivity was not tested; verify reachability from the enrolled client.';const download=page.querySelector('[data-action=download-output-config]');if(download)download.disabled=false;toast('Endpoint configuration validated locally; network connectivity was not tested');break; }
      case 'parser-sync-verify': { const target=['quarantine-wizard','assurance','field-mapping'].includes(state.page)?state.page:'parser-sync',progress=notify('Refreshing controller and client reported parser state…','progress');try{const node=button.dataset.node;
        const body=node?{bundle_hash:button.dataset.bundle,node_ids:[node]}:{asset_class:button.dataset.class,asset_id:id};
        const result=await api('/parser-sync/verify',{method:'POST',body});
        const acknowledgement=result.client_acknowledgements?.find(item=>item.node_id===node);
        if(node){
          if(acknowledgement?.exact_acknowledgement===true)progress.done('Origin client acknowledged the exact bundle.');
          else progress.fail([acknowledgement?.error,acknowledgement?.next_action].filter(Boolean).join(' · ')||'The origin client has not acknowledged this bundle yet.');
        }else progress.done([result.message||'Parser sync state verified',result.blocking_code,result.next_action].filter(Boolean).join(' · '));if(target==='quarantine-wizard')state.qrwStep=8;await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){progress.fail(error.message);throw error;}break; }
      case 'parser-sync-retry': { const target=['quarantine-wizard','assurance','field-mapping'].includes(state.page)?state.page:'parser-sync',body={asset_class:button.dataset.class,asset_id:id};if(button.dataset.node)body.node_ids=[button.dataset.node];const progress=notify('Creating an approval-gated parser rollout proposal…','progress');try{const result=await api('/parser-sync/retry',{method:'POST',body});notify(result.message||`Rollout proposal ${result.proposal_id||result.id||'created'}; independent approval is still required.`);if(target==='quarantine-wizard')state.qrwStep=8;await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){progress.fail(error.message);throw error;}break; }
      case 'fleet-focus': applyFleetFocus(Number(button.dataset.value));break;
      case 'fleet-refresh': await navigate(state.page);break;
      case 'fleet-open': await navigate(button.dataset.value);break;
      case 'parser-sync-rollout-propose': {const target=['quarantine-wizard','assurance','field-mapping'].includes(state.page)?state.page:'parser-sync',targetMode=button.dataset.targetMode,originNode=button.dataset.originNode||'',bundleHash=state.assuranceSyncBundle?.desired_hash;if(!bundleHash)throw new Error('The current published parser bundle hash is unavailable. Refresh the wizard before proposing a rollout.');if(!['origin','all_eligible','selected'].includes(targetMode))throw new Error('Choose a supported rollout target scope');if(['origin','selected'].includes(targetMode)&&!originNode)throw new Error('Choose a client before proposing targeted delivery');const body={target_mode:targetMode,origin_node_id:originNode,node_ids:targetMode==='selected'?[originNode]:[],bundle_hash:bundleHash};const progress=notify(`Creating ${targetMode==='origin'?'origin client':'all eligible client'} rollout proposal…`,'progress');try{const result=await api('/parser-sync/rollout',{method:'POST',body}),proposal=result.proposal||{},proposalId=proposal.id||result.proposal_id||'created';state.qrwStep=8;notify(`Parser rollout proposal ${proposalId} is ${proposal.state||'pending_approval'}. No client receives this bundle until an independent reviewer approves it.`);await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){progress.fail(error.message);throw error;}break; }
      case 'recovery-open-helper': { state.qrwStep=5;const helperTab=page.querySelector('[data-assurance-tab=gprs]');if(helperTab)helperTab.click();setupQRWStepPanels();const form=page.querySelector('#gprs-parser-helper-form');form?.scrollIntoView({behavior:'smooth',block:'start'});form?.elements.samples?.focus();break; }
      case 'recovery-prepare-replay': { await navigate('unknowns');const form=page.querySelector('#quarantine-reprocess-form');if(!form)throw new Error('Replay form is unavailable');form.elements.record_ids.value=id;form.elements.node_id.value=button.dataset.node||'';form.scrollIntoView({behavior:'smooth',block:'center'});form.elements.record_ids.focus();toast('Replay form prepared after the origin client acknowledged the published parser bundle');break; }
      case 'assurance-tab': { const key=button.dataset.assuranceTab;page.querySelectorAll('[data-assurance-panel]').forEach(section=>section.hidden=section.dataset.assurancePanel!==key);page.querySelectorAll('[role=tab][data-assurance-tab]').forEach(tab=>{const active=tab===button;tab.classList.toggle('active',active);tab.setAttribute('aria-selected',String(active));tab.tabIndex=active?0:-1;});break; }
      case 'copy-helper-value': { const source=page.querySelector(`[data-copy-source="${button.dataset.copyIndex}"]`);if(!source)throw new Error('Generated value is unavailable');source.select();await navigator.clipboard.writeText(source.value);toast('Generated value copied');break; }
      case 'assurance-parser-edit': { const doc=await api(`/assurance/parsers/${encodeURIComponent(id)}`),p=doc.payload||doc,form=page.querySelector('#gprs-parser-helper-form');if(!['draft','rejected'].includes(doc.state))throw new Error('Only draft or rejected parsers can be edited');form.elements.edit_id.value=id;form.elements.samples.value=(p.cases||[]).map(item=>item.raw||'').filter(Boolean).join('\n');form.elements.name.value=p.name||id;form.elements.version.value=p.version||'0.1.0';form.elements.parser.value=pretty(p.parser||{});form.elements.cases.value=pretty(p.cases||[]);renderParserHelperDetails(p.parser||{},form);form.querySelector('[data-parser-helper-status]').textContent=`Editing ${id} revision ${doc.revision||'—'}. Saving creates a reviewable revision.`;state.qrwStep=5;setupQRWStepPanels();page.querySelector('[data-assurance-tab=gprs]')?.click();form.scrollIntoView({behavior:'smooth',block:'start'});break; }
      case 'assurance-parser-delete': { const target=state.page==='quarantine-wizard'?'quarantine-wizard':state.page==='ai'?'ai':state.page==='field-mapping'?'field-mapping':'assurance',inGprs=!!button.closest('[data-assurance-panel=gprs]'); if(confirm(`Delete draft parser ${id}?`)){await api(`/assurance/parsers/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast('Parser draft deleted');await navigate(target);if(target==='assurance'&&inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();}break; }
      case 'assurance-plan-edit': { const doc=await api(`/assurance/plans/${encodeURIComponent(id)}`),p=doc.payload||{},form=page.querySelector('#assurance-plan-form');form.elements.edit_id.value=id;form.elements.id.value=id;form.elements.id.readOnly=true;form.elements.name.value=p.name||id;form.elements.version.value=p.version||'1.0.0';form.elements.plan.value=pretty(p.plan||{});form.scrollIntoView({behavior:'smooth',block:'start'});toast(`Editing test-plan revision ${doc.revision||'—'}`);break; }
      case 'assurance-plan-reset': { const form=page.querySelector('#assurance-plan-form');form.reset();form.elements.edit_id.value='';form.elements.id.readOnly=false;break; }
      case 'assurance-plan-download': { const doc=await api(`/assurance/plans/${encodeURIComponent(id)}`);downloadJSON(`${id}-assurance-plan.json`,doc);break; }
      case 'assurance-run-cancel': { const result=await api(`/assurance/runs/${encodeURIComponent(id)}/cancel`,{method:'POST',body:{}});notify(`Assurance run ${id} is ${result.state}`);await navigate('assurance');page.querySelector('[data-assurance-tab=runs]')?.click();break; }
      case 'assurance-run-download': { const doc=await api(`/assurance/runs/${encodeURIComponent(id)}`);downloadJSON(`${id}-assurance-run.json`,doc);break; }
      case 'assurance-plan-delete': { if(confirm(`Delete Assurance test plan ${id}?`)){await api(`/assurance/plans/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast('Assurance test plan deleted');await navigate('assurance');page.querySelector('[data-assurance-tab=plans]')?.click();}break; }
      case 'parser-workspace-generate': { const form=button.closest('form');generateParserHelper(form);const status=form.querySelector('[data-parser-status]');if(status)status.textContent='Complete draft generated locally. Review every field and validation case before saving.';break; }
      case 'parser-workspace-validate': { const form=button.closest('form'),parser=parseJSON(form.elements.parser.value,'Parser artifact JSON'),cases=parseJSON(form.elements.cases.value,'Validation cases JSON');if(!parser.pattern_type||!parser.pattern||!Array.isArray(parser.fields)||(!parser.fields.length&&!parser.mapping_plan?.fields?.some(field=>['kv_all','kv_eol'].includes(field.method)&&field.output_policy==='flatten')))throw new Error('Parser requires pattern_type, pattern and at least one field');if(!Array.isArray(cases)||!cases.length)throw new Error('Provide validation cases');const status=form.querySelector('[data-parser-status]');if(status)status.textContent=`Local structure valid: ${parser.fields.length} fields and ${cases.length} cases. Save to run authoritative controller validation.`;break; }
      case 'parser-workspace-clear': { const form=button.closest('form');form.reset();form.elements.edit_id.value='';form.elements.parser.value='{}';form.elements.cases.value='[]';form.querySelector('[data-parser-overview]').textContent='Paste representative lines, then auto-populate the parser artifact.';form.querySelector('[data-parser-status]').textContent='';break; }
      case 'parser-workspace-load': { const doc=await api(`/assurance/parsers/${encodeURIComponent(id)}`),p=doc.payload||doc,form=page.querySelector('#parser-workspace-form');form.elements.edit_id.value=id;form.elements.samples.value=(p.cases||[]).map(item=>item.raw||'').filter(Boolean).join('\n');form.elements.name.value=p.name||id;form.elements.version.value=p.version||'0.1.0';form.elements.parser.value=pretty(p.parser||{});form.elements.cases.value=pretty(p.cases||[]);form.querySelector('[data-parser-overview]').textContent=`Editing ${id} revision ${doc.revision||'—'}. Saving creates a new draft revision.`;form.scrollIntoView({behavior:'smooth',block:'start'});break; }
      case 'parser-workspace-delete': { if(confirm(`Delete draft parser ${id}?`)){await api(`/assurance/parsers/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});notify('Draft parser deleted');await navigate('parser-workspace');}break; }
      case 'assurance-open-parsers': location.hash='parsers';break;
      case 'engine-test': { const spec=ENGINE_SPECS[button.dataset.page];state.lastAssurance={request:{raw:spec.sample,service:spec.family,source:spec.slug,desired:{},field_mapping:{}},result:null};location.hash='assurance';break; }
      case 'input-profile-edit': { const doc=await api(`/config/input-paths/${encodeURIComponent(id)}`);const form=page.querySelector('#input-profile-form'),config=doc.payload?.config||{};form.elements.id.value=id;form.elements.id.readOnly=true;form.elements.name.value=doc.payload?.name||id;form.elements.config.value=pretty(config);el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config),'No paths configured.');form.scrollIntoView({behavior:'smooth',block:'center'});toast(`Profile revision ${doc.revision||'—'} loaded; saving creates a new draft revision`);break; }
      case 'input-profile-delete': { if(confirm(`Remove input profile ${id}?`)){await api(`/config/input-paths/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast('Input profile removed');await navigate('input-paths');}break; }
      case 'input-add-row': { const form=button.closest('form'),path=form.elements.quick_path.value.trim(),service=form.elements.quick_family.value;if(!path.startsWith('/'))throw new Error('Enter an absolute Linux log path');const config=parseJSON(form.elements.config.value,'Input definitions JSON');config.inputs=Array.isArray(config.inputs)?config.inputs:[];if(config.inputs.length>=256)throw new Error('An agent supports at most 256 paths');if(config.inputs.some(item=>item.path===path))throw new Error('That input path is already listed');config.inputs.push({path,service,multiline:multilineForFamily(service),mask:{disable_types:[]}});form.elements.config.value=pretty(config);form.elements.quick_path.value='';el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config));toast('Input path added; existing paths were preserved');break; }
      case 'input-remove-row': { const form=button.closest('form'),config=parseJSON(form.elements.config.value,'Input definitions JSON'),index=Number(button.dataset.index);if(!config.inputs?.[index])throw new Error('Input path is no longer available');config.inputs.splice(index,1);updateInputPathEditor(form,config);break; }
      case 'input-edit-row': {
        const input=button.closest('tr')?.querySelector('[data-input-path]');
        input?.focus();
        input?.select();
        break;
      }
      case 'input-toggle-row':
      case 'input-toggle-all':
      case 'input-remove-all': {
        const operation=button.dataset.action;
        const form=button.closest('form');
        const config=parseJSON(form.elements.config.value,'Input definitions JSON');
        if(!Array.isArray(config.inputs))throw new Error('Input definitions must contain an inputs array');
        if(operation==='input-remove-all'){
          if(!confirm('Remove all paths from this collection profile?'))break;
          config.inputs=[];
        }else{
          const paths=operation==='input-toggle-all'?config.inputs:[config.inputs[Number(button.dataset.index)]];
          if(paths.some(item=>!item))throw new Error('Input path is no longer available');
          for(const item of paths){
            const disable=operation==='input-toggle-all'?button.dataset.value==='disable':item.disabled!==true;
            if(disable)item.disabled=true;
            else delete item.disabled;
          }
        }
        updateInputPathEditor(form,config);
        break;
      }
      case 'input-validate': {
        const inputs=parseJSON(button.closest('form').elements.config.value,'Input definitions JSON').inputs;
        if(!Array.isArray(inputs)||inputs.length>256)throw new Error('Provide an inputs array with no more than 256 input paths');
        if(inputs.some(item=>!item||typeof item.path!=='string'||!item.path.startsWith('/')||item.path.length>4096||/[\r\n\x00]/.test(item.path)||![...INPUT_FAMILIES,'webtop-webmail'].includes(item.service)))throw new Error('Each path must be absolute and use a supported parser family');
        if(inputs.some(item=>Object.hasOwn(item,'disabled')&&typeof item.disabled!=='boolean'))throw new Error('The disabled field must be true or false');
        if(new Set(inputs.map(item=>item.path.toLowerCase())).size!==inputs.length)throw new Error('Input paths must be unique');
        toast(`${inputs.length} input path${inputs.length===1?'':'s'} valid`);
        break;
      }
      case 'input-use-automatic': { const form=button.closest('form'),config=defaultInputProfile();form.elements.config.value=pretty(config);el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config));toast('All standard paths loaded');break; }
      case 'input-profile-reset': { const form=page.querySelector('#input-profile-form'),config=defaultInputProfile();form.reset();form.elements.id.readOnly=false;form.elements.config.value=pretty(config);el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config));break; }
      case 'config-action': { const kind=button.dataset.kind,[collection]=CONFIG[kind],assetPath=`${collection}/${encodeURIComponent(id)}`;if(value==='revoke'&&kind==='knownkeys'){if(confirm(`Remove ${id} from the published Known Key Catalog and request fleet synchronization?`)){await api(`${assetPath}/revoke`,{method:'POST',body:{}});toast('Known keys revoked. Review the removal rollout in Approvals & Rollouts.');await navigate(kind);}}else if(value==='edit'||value==='copy'&&kind==='knownkeys'){const doc=await api(assetPath);if(value==='edit'&&!['draft','rejected'].includes(doc.state))throw new Error('Only draft or rejected assets can be edited.');if(kind==='outputs'){const form=page.querySelector('#output-profile-form'),config=doc.payload?.config||{},outputs=config.outputs||[],byType=type=>outputs.find(item=>item.type===type);if(!form)throw new Error('Output profile editor is unavailable');form.elements.id.value=id;form.elements.name.value=doc.payload?.name||id;form.elements.id.readOnly=true;form.querySelectorAll('[name=destinations]').forEach(input=>input.checked=outputs.some(item=>item.type===input.value));const local=outputs.filter(item=>item.type==='file'),normalised=form.querySelector('[name=normalised_enabled]');if(normalised)normalised.checked=local.some(item=>(item.routes||[]).includes('normal')||(item.routes||[]).includes('warning'));const normalFile=local.find(item=>(item.routes||[]).includes('normal')||(item.routes||[]).includes('warning'));const quarantineFile=local.find(item=>(item.routes||[]).includes('quarantine')||(item.routes||[]).includes('unknown'));if(normalFile)form.elements.normal_path.value=normalFile.path_template||normalFile.path;if(quarantineFile)form.elements.quarantine_path.value=quarantineFile.path_template||quarantineFile.path;const loki=byType('loki')||{};form.elements.loki_url.value=loki.url||'';form.elements.loki_token_env.value=loki.bearer_token_env||'';form.elements.loki_tenant.value=loki.tenant_id||'';form.elements.loki_labels.value=pretty(loki.labels||{job:'nli'});const clickhouse=byType('clickhouse')||{};form.elements.clickhouse_url.value=clickhouse.url||'';form.elements.clickhouse_database.value=clickhouse.database||'default';form.elements.clickhouse_table.value=clickhouse.table||'nli_logs';form.elements.clickhouse_username.value=clickhouse.username||'';form.elements.clickhouse_password_env.value=clickhouse.password_env||'';form.elements.field_columns.value=pretty(clickhouse.field_columns||{});const controller=byType('controller')||{};form.elements.controller_url.value=controller.url||'';form.elements.controller_token_file.value=controller.token_file||'';form.elements.controller_node_id_file.value=controller.node_id_file||'/var/lib/nli/node-id';state.outputPreview=config;const preview=el('output-profile-preview');if(preview)preview.textContent=pretty(config);const download=page.querySelector('[data-action=download-output-config]');if(download)download.disabled=false;form.scrollIntoView({behavior:'smooth',block:'center'});form.querySelector('button[type=submit]').textContent='Validate and save revised draft';toast('Output profile loaded for editing');}else{const form=page.querySelector('#config-form');const config={...(doc.payload?.config||{})};form.elements.id.value=value==='copy'?'':id;form.elements.name.value=doc.payload?.name||id;if(kind==='knownkeys'){form.elements.known_keys.value=(config.keys||[]).join('\n');delete config.keys;}form.elements.config.value=pretty(config);form.scrollIntoView({behavior:'smooth',block:'center'});form.querySelector('[name=id]').readOnly=value!=='copy';form.querySelector('button[type=submit]').textContent='Save new draft revision';toast('Draft loaded in the editor');}}else if(value==='delete'){if(confirm(`Remove ${kind==='outputs'?'endpoint profile':'parser asset'} ${id}?`)){await api(assetPath,{method:'DELETE',body:{}});toast(kind==='outputs'?'Endpoint profile removed':'Asset deleted');await navigate(kind);}}else if(value==='download'){const response=await fetchWithTimeout(`/api/v1${assetPath}/download`,{credentials:'same-origin',headers:{Accept:'application/json'}},15000);if(!response.ok)throw new Error(`Asset download failed (${response.status})`);downloadBlob(`${id}.json`,await response.blob());}else if(['submit','approve','publish'].includes(value)){const noun=kind==='outputs'?'endpoint profile':kind==='input-paths'?'input-path profile':'parser asset',body={};if(value==='approve'){const reason=prompt('Approval comment (optional, 3–512 characters):');if(reason===null)break;const comment=reason.trim();if(comment&&comment.length<3)throw new Error('Approval comment must contain at least 3 characters');if(comment)body.reason=comment;}const progress=notify(value==='publish'?`Publishing ${noun}…`:`${value==='approve'?'Approving':'Submitting'} ${noun}…`,'progress');try{const result=await api(`${assetPath}/${value}`,{method:'POST',body});notify(value==='publish'?(kind==='outputs'?'Endpoint profile published. Select target nodes in the rollout form.':kind==='input-paths'?'Input-path profile published and available for rollout.':`Published. Fleet bundle ${String(result.bundle_hash||'').slice(0,12)} is synchronizing.`):`${noun} ${value} complete.`);await navigate(kind);}catch(error){progress.fail(error.message);throw error;}}break; }
      case 'parser-restore': {const newID=prompt('New parser ID for the restored inactive draft:',`${id}-restored`);if(newID===null)break;if(!newID.trim()||newID.trim()===id)throw new Error('Use a distinct, nonempty parser ID for the restored draft.');const result=await api(`/parser-packs/deleted/${encodeURIComponent(id)}/restore`,{method:'POST',body:{id:newID.trim(),reason:'Administrator restored archived parser as inactive draft'}});notify(`Restored ${result.id||newID} as an inactive draft. Validate representative logs, submit, approve and publish before rollout.`);await navigate('parsers');break;}
      case 'parser-asset': { if(value==='edit'){const doc=await api(`/parser-packs/${encodeURIComponent(id)}`);const p=doc.payload||doc;if(!['draft','rejected'].includes(doc.state))throw new Error('Only draft or rejected parser packs can be edited.');const form=page.querySelector('#parser-form');form.dataset.editId=id;form.elements.id.value=id;form.elements.id.readOnly=true;form.elements.name.value=p.name||id;form.elements.version.value=p.version||'0.1.0';form.elements.priority.value=p.priority||100;form.elements.rules.value=pretty(p.rules||[]);form.elements.regression_cases.value=pretty(p.regression_cases||[]);form.querySelector('button[type=submit]').textContent='Save parser revision';form.scrollIntoView({behavior:'smooth',block:'center'});}else if(value==='delete'){if(confirm(`Archive and remove parser pack ${id} from the desired runtime bundle? Active clients must acknowledge the removal bundle before it stops running there. You can restore an inactive draft later.`)){const deleted=await api(`/parser-packs/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast(deleted.rollout_proposal?`Parser archived. Removal rollout ${deleted.rollout_proposal.id||''} requires approval; client removal is pending exact acknowledgement.`:'Parser archived; refresh inventory to verify controller and client status.');await navigate('parsers');}}else if(value==='download'){const doc=await api(`/parser-packs/${encodeURIComponent(id)}`);downloadJSON(`${id}.json`,doc.payload||doc);}break; }
      case 'download-builtin': { const item=state.builtinParsers.find(parser=>String(parser.id)===id);if(!item)throw new Error('Built-in parser details are no longer available; refresh the catalog.');downloadJSON(`${id}-builtin.json`,item);break; }
      case 'download-migration-report': { const report=state.migrationReports.find(item=>String(item.id)===id);if(!report)throw new Error('Migration report is no longer in the current snapshot.');downloadJSON(`${id}-migration-report.json`,report.payload||report);break; }
      case 'save-assurance': { const form=button.closest('form');if(!form?.reportValidity())break;if(!state.lastAssurance)throw new Error('Run assurance first');const data=formObject(form);await api('/assurance/cases',{method:'POST',body:{id:data.id,name:data.name,description:data.description,retain:true,case:state.lastAssurance.request}});toast('Encrypted fixture retained');await navigate('assurance');page.querySelector('[data-assurance-tab=fixtures]')?.click();break; }
      case 'clear-assurance': state.lastAssurance = null; await navigate('assurance'); break;
      case 'download-assurance': if(state.lastAssurance){const result=state.lastAssurance.result;downloadJSON('nli-v22-assurance-result.json',{...state.lastAssurance,result:{...result,actual:evidenceOrdered(result.actual,(result.evidence_spans||[]).slice().sort((left,right)=>evidenceStart(left)-evidenceStart(right)))}});} break;
      case 'show-corpus': { const text=prompt('Paste a JSON array of assurance cases');if(!text)return;const result=await api('/assurance/corpus',{method:'POST',body:{cases:parseJSON(text,'Corpus JSON')}});el('assurance-result').innerHTML=panel('Corpus result','Corpus result is transient; run and retain individual cases for persistence.',`<pre class="json-output">${esc(pretty(result))}</pre>`);break; }
      case 'assurance-parser-action': { const target=['quarantine-wizard','field-mapping'].includes(state.page)?state.page:'assurance',body={},doc=(state.assuranceParsers||[]).find(item=>item.id===id);if(value==='reject'){const reason=prompt('Why is this parser proposal being rejected?');if(reason===null)break;if(!reason.trim())throw new Error('A rejection reason is required');body.reason=reason.trim();}if(value==='approve'&&doc?.created_by===state.user?.username){if(state.user?.role!=='admin')throw new Error('A different authorized reviewer must approve this candidate');const reason=prompt(`Administrator approval for ${id} revision ${doc.revision}. Enter the operational reason (minimum 12 characters):`);if(reason===null)break;if(reason.trim().length<12)throw new Error('Approval reason must contain at least 12 characters');const checksum=doc.payload?.artifact_checksum;if(!checksum)throw new Error('Candidate artifact checksum is unavailable; refresh before approval');Object.assign(body,{emergency_self_approval:true,reason:reason.trim(),expected_revision:Number(doc.revision),expected_checksum:checksum});}else if(value==='approve'&&state.user?.role==='admin'){const reason=prompt('Approval comment (optional, 3–512 characters):');if(reason===null)break;if(reason.trim())body.reason=reason.trim();}const progress=notify(`${value} parser proposal…`,'progress');try{const result=await api(`/assurance/parsers/${encodeURIComponent(id)}/${value}`,{method:'POST',body}),rolloutProposal=result.rollout_proposal;notify(rolloutProposal?`Parser ${value} saved. Rollout proposal ${rolloutProposal.id||'created'} is ${rolloutProposal.state||result.approval_gate||'pending approval'} for bundle ${String(result.bundle_hash||'').slice(0,16)}.`:`Parser ${value} completed${result.validation?.score!=null?` · score ${result.validation.score}`:''}`);if(target==='quarantine-wizard')state.qrwStep=['publish','rollback','offboard'].includes(value)?8:7;await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){progress.fail(error.message);throw error;}break; }
      case 'create-candidate-parser': { const target=state.page==='quarantine-wizard'?'quarantine-wizard':state.page==='ai'?'ai':state.page==='field-mapping'?'field-mapping':'assurance',inGprs=!!button.closest('[data-assurance-panel=gprs]'); const candidate=state.lastAssurance?.result?.candidate_parser;if(!candidate)throw new Error('No parser candidate is available');const parser=candidate.parser||candidate;const result=await api('/assurance/parsers',{method:'POST',body:{name:candidate.name||'Assurance candidate',version:candidate.version||'0.1.0',parser,cases:[state.lastAssurance.request],source_quarantine_record_id:state.quarantineDiagnosis?.id||''}});notify(`Candidate draft created · validation ${result.validation?.score??'pending'}%`);await navigate(target);if(target==='assurance'&&inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();break; }
      case 'request-ai-proposal': { const target=state.page==='quarantine-wizard'?'quarantine-wizard':state.page==='ai'?'ai':state.page==='field-mapping'?'field-mapping':'assurance',inGprs=!!button.closest('[data-assurance-panel=gprs]'); if(!state.lastAssurance||state.lastAssurance.result?.passed)throw new Error('Run a failing Assurance case first');button.disabled=true;const progress=notify('Masking the failed sample and requesting an AI remediation draft…','progress');try{const result=await requestAIProposal(state.lastAssurance.request);progress.done(`AI diagnosis ${result.id||'draft'} created and ready for review.`);await navigate(target);if(target==='assurance'&&inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){progress.fail(error.message);}finally{button.disabled=false;}break; }
      case 'ai-proposal-reject': case 'ai-proposal-remove': {const reject=button.dataset.action==='ai-proposal-reject';let reason='';if(reject){reason=prompt(`Reason for rejecting AI proposal ${id} (1–512 characters):`)||'';if(!reason.trim())break;if(reason.length>512)throw new Error('Rejection reason must be 1–512 characters.');}if(!confirm(`${reject?'Reject':'Remove'} AI proposal ${id}? Published parsers remain unchanged; active pack revocation uses Catalog Offboard / Rollback.`))break;await api(`/assurance/ai-proposals/${encodeURIComponent(id)}${reject?'/reject':''}`,{method:reject?'POST':'DELETE',body:reject?{reason}:{}});notify(`AI proposal ${id} ${reject?'rejected':'removed'}; published parsers unchanged.`);state.aiProposals=(await api('/assurance/ai-proposals')).proposals||[];await navigate(state.page);break;}
      case 'ai-proposal-approve': { const target=state.page==='quarantine-wizard'?'quarantine-wizard':state.page==='ai'?'ai':state.page==='field-mapping'?'field-mapping':'assurance',inGprs=!!button.closest('[data-assurance-panel=gprs]'); const reason=prompt('Approval comment (optional, 3–512 characters):');if(reason===null)break;const progress=notify('Approving AI remediation proposal…','progress');try{await api(`/assurance/ai-proposals/${encodeURIComponent(id)}/approve`,{method:'POST',body:{reason:reason.trim()}});notify('AI remediation approved for parser-draft validation. No live parser was changed.');if(target==='field-mapping')state.aiProposals=(await api('/assurance/ai-proposals')).proposals||[];await navigate(target);if(target==='assurance'&&inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();}catch(error){progress.fail(error.message);throw error;}break; }
      case 'ai-proposal-create-draft': { const target=state.page==='quarantine-wizard'?'quarantine-wizard':state.page==='ai'?'ai':state.page==='field-mapping'?'field-mapping':'assurance',inGprs=!!button.closest('[data-assurance-panel=gprs]'); const doc=state.aiProposals.find(item=>String(item.id)===String(id)),p=doc?.payload||{};if(!doc||doc.state!=='approved')throw new Error('The AI remediation must be approved first');if(!state.lastAssurance?.request?.raw||state.lastAssurance.result?.raw_sha256!==p.source_hash)throw new Error('Rerun the matching failing sample before creating this parser draft');const result=await api('/assurance/parsers',{method:'POST',body:{name:p.name||`AI remediation ${id}`,version:'0.1.0',parser:p.proposal,cases:[state.lastAssurance.request],source_quarantine_record_id:state.quarantineDiagnosis?.id||''}});notify(`Validated parser draft created · score ${result.validation?.score??'pending'}%. Submit it for independent parser review when ready.`);await navigate(target);if(target==='assurance'&&inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();break; }
      case 'ai-credential-remove': case 'ai-profile-delete': {const form=page.querySelector('#ai-provider-form'),profileID=button.dataset.id;if(!profileID||form?.elements.id.value!==profileID)throw new Error('Load the exact saved profile before removing it.');const removeCredential=button.dataset.action==='ai-credential-remove';if(!confirm(`${removeCredential?'Remove stored credentials and their metadata history for':'Delete profile and its stored metadata history for'} ${profileID}? This cannot be undone. External keys are not revoked; already outbound requests may finish.`))break;const result=await api(`/admin/ai-provider?${removeCredential?'action=clear-credential&':''}id=${encodeURIComponent(profileID)}`,{method:removeCredential?'POST':'DELETE',body:{}});state.aiSelectedProviderId=removeCredential&&result.profile_retained?profileID:'';notify(`${profileID}: ${removeCredential?'stored credentials removed; profile disabled':'profile deleted'}. External keys were not revoked.`);await navigate('ai');break;}
      case 'ai-profile-set-default':
      case 'ai-profile-clear-default': {
        const pin=button.dataset.action==='ai-profile-set-default',id=el('ai-profile-choice').value;
        if(pin&&!id)throw new Error('Choose a saved AI profile first.');
        await api('/admin/ai-provider?action='+(pin?'set-default&id='+encodeURIComponent(id):'clear-default'),{method:'POST',body:{}});
        state.aiSelectedProviderId=pin?id:'';
        await navigate('ai');
        notify(pin?'Default AI profile saved. NLI will use only '+id+'.':'Automatic AI routing restored.');
        break;
      }
      case 'ai-order-up':
      case 'ai-order-down': {
        const row=button.closest('[data-profile-id]');
        if(button.dataset.action==='ai-order-up')row.previousElementSibling?.before(row);
        else row.nextElementSibling?.after(row);
        page.dataset.dirty='true';
        button.focus();
        break;
      }
      case 'ai-order-save':
      case 'ai-order-reset': {
        const order=button.dataset.action==='ai-order-reset'?[]:[...el('ai-route-order').children].map(row=>row.dataset.profileId);
        await api('/admin/ai-provider?action=set-order',{method:'POST',body:{profile_order:order}});
        await navigate('ai');
        notify(order.length?'Model priority order saved.':'Numeric model priorities restored.');
        break;
      }
      case 'ai-profile-load': {state.aiSelectedProviderId=el('ai-profile-choice').value;await navigate('ai');break;}
      case 'ai-profile-retry': await loadAIProvider();applyPermissions();break;
      case 'ai-profile-new': {page.dataset.dirty='true';const form=page.querySelector('#ai-provider-form');for(const key of ['id','name','model','endpoint','api_key','credential_ref'])form.elements[key].value='';form.elements.enabled.checked=true;form.elements.priority.value='10';form.elements.fallback_group.value='review';form.elements.credential_source.value='env_ref';form.querySelectorAll('[data-action=ai-credential-remove],[data-action=ai-profile-delete]').forEach(item=>item.disabled=true);form.elements.id.focus();toast('Enter a unique profile ID and exact model. Profiles may share a protected controller environment reference.');break;}
      case 'ai-profile-from-preset': {
        const form=el('ai-provider-form');
        const template=state.aiProviderTemplates?.find(item=>item.id===el('ai-provider-preset').value&&item.fill_model);
        if(!template)throw new Error('Select a Nexora AI provider preset first.');
        let endpoint=template.base_url;
        if(endpoint.includes('{account_id}')){
          const account=el('ai-preset-account').value.trim().toLowerCase();
          if(!/^[a-f0-9]{32}$/.test(account))throw new Error('Enter the 32-character Cloudflare account ID for this API token.');
          endpoint=endpoint.replace('{account_id}',account);
        }
        const used=new Set([...el('ai-profile-choice').options].map(option=>option.value));
        let id=template.id;
        for(let suffix=2;used.has(id);suffix++)id=template.id+'-'+suffix;
        await action({dataset:{action:'ai-profile-new'}});
        form.elements.id.value=id;
        form.elements.name.value=template.name;
        form.elements.provider.value=template.kind;
        form.elements.endpoint.value=endpoint;
        form.elements.model.value=template.model_hint;
        el('ai-model-options').innerHTML=aiModelOptions(template);
        form.elements.credential_source.value=template.credential_source;
        form.elements.api_key.focus();
        break;
      }
      case 'ai-profile-preset': {page.dataset.dirty='true';const form=page.querySelector('#ai-provider-form'),template=state.aiProviderTemplates?.find(item=>item.kind===form.elements.provider.value);if(!template)throw new Error('Provider endpoint preset is unavailable; enter the base URL explicitly.');form.elements.endpoint.value=template.base_url;form.elements.model.placeholder=template.model_hint||'Exact model identifier';toast('Endpoint preset copied. The configured model and credentials were preserved; confirm them before testing.');break;}
      case 'mapping-ai-readiness': {const host=button.closest('[data-ai-mapping-host]');host.querySelector('[data-ai-mapping-status]').textContent=pretty(await api('/ai/capabilities'));break;}
      case 'mapping-ai-dismiss': {const host=button.closest('[data-ai-mapping-host]');delete host.aiProposal;host.querySelector('[data-ai-mapping-proposal]').hidden=true;host.querySelector('[data-ai-mapping-status]').textContent='Proposal dismissed. Manual/native mappings remain unchanged.';break;}
      case 'mapping-ai-suggest': {const host=button.closest('[data-ai-mapping-host]'),raw=mappingAIRaw(host),status=host.querySelector('[data-ai-mapping-status]');if(!raw.trim())throw new Error('Paste or inspect exact raw log evidence before requesting AI suggestions.');const capability=await api('/ai/capabilities');status.textContent=pretty(capability);if(!capability.available&&aiEvidenceSource(raw)!=='production')throw new Error('AI is unconfigured or unavailable. Configure a controller provider and credential first; no AI suggestion request was sent.');const requestID=(host.aiRequestID||0)+1;host.aiRequestID=requestID;delete host.aiProposal;host.querySelector('[data-ai-mapping-proposal]').hidden=true;const mapping=host.dataset.aiMappingEditor==='true',signature=mapping?JSON.stringify(qrwPlan()):'',service=mapping?qrwScope().service_scope.split(',')[0].trim()||'auto':'auto';status.textContent='Requesting a separate AI proposal. Current manual/native mappings are preserved.';try{const result=await api('/field-mapping/ai-suggest',{method:'POST',body:{raw,evidence_source:aiEvidenceSource(raw),service:service==='*'?'auto':service,native_fallback:true,...(mapping&&state.qrwFields.length?{plan:qrwPlan()}:{})}});if(!host.isConnected||host.aiRequestID!==requestID)return;if(mappingAIRaw(host)!==raw){status.textContent='Raw evidence changed; stale AI proposal discarded.';return;}const plan=result.proposal?.plan;if(!plan||!Array.isArray(plan.fields)||result.validation?.syntax_valid!==true||result.validation?.extraction_valid!==true)throw new Error('The controller did not return a syntax-compatible, extractable native mapping proposal.');host.aiProposal={result,raw,signature};host.querySelector('[data-ai-mapping-json]').textContent=pretty(result);host.querySelector('[data-ai-mapping-proposal]').hidden=false;status.textContent=(result.proposal_origin==='native_fallback'?result.answer+' ':'')+'Review-only mapping proposal received. Syntax/extraction were checked; expected-output matching and publication validation remain distinct. Nothing was imported or saved. Review and confirm delimiter and explicit constant interpretations before native verification or saving.';}catch(error){if(host.isConnected&&host.aiRequestID===requestID)status.textContent=aiActionFailure(error);}break;}
      case 'mapping-ai-import': {const host=button.closest('[data-ai-mapping-host]'),proposal=host.aiProposal;if(!proposal)throw new Error('Request a current AI mapping proposal first.');if(mappingAIRaw(host)!==proposal.raw)throw new Error('Raw evidence changed. Request a new AI proposal; stale mappings cannot be imported.');const mapping=host.dataset.aiMappingEditor==='true';if(mapping&&proposal.signature!==JSON.stringify(qrwPlan()))throw new Error('The editable mapping changed. Request a new AI proposal.');if(!confirm(mapping?'Replace the editable mapping fields with this reviewed AI proposal? Existing fields will be replaced only in this unsaved editor. Validation and authorized publication are still required.':'Open this proposed mapping in FieldSense? Current authoring form edits are not saved automatically. No active parser will change.'))break;const plan=proposal.result.proposal.plan;if(mapping){el('nfi-plan-import').value=pretty(plan);await action({dataset:{action:'qrw-import-plan'}});nfiScheduleLive();}else{const proposedParser=proposal.result.proposal.parser||{};state.nfiService=Array.isArray(proposedParser.service_scope)?proposedParser.service_scope.join(','):proposedParser.service_scope||'auto';state.nfiScopePattern=proposedParser.scope_pattern||'';state.nfiGlobalScope=proposedParser.reviewed_global_scope===true;state.nfiMappingPriority=Number(proposedParser.mapping_priority)||0;state.nfiRaw=proposal.raw;state.qrwFields=plan.fields.map(field=>qrwField({...field,status:'unresolved',confidence:'ai_proposed',evidence:'AI proposal requires full native validation'}));qrwControllerPreview=null;await navigate('field-mapping');nfiScheduleLive();}delete host.aiProposal;if(host.isConnected){host.querySelector('[data-ai-mapping-proposal]').hidden=true;host.querySelector('[data-ai-mapping-status]').textContent='Proposed fields imported into the unsaved editor. Review delimiter confirmation controls and explicit constants, then verify and validate the complete candidate before review.';}break;}
      case 'workspace-ai-field-help': {
        const draft=button.closest('form')?.aiResult;
        const field=button.dataset.field||button.closest('[data-ai-field-guidance]')?.querySelector('[data-ai-observed-field]')?.value;
        if(!field||!draft)throw new Error('Build a current draft and select a field first.');
        const finding=draft.result.validation?.diff?.find(item=>item.field===field);
        const candidate=draft.result.proposal?.plan?.fields?.find(item=>item.name===field);
        const question='Explain how to fix only field '+field+
          (finding?' ('+finding.status+'; expected type '+(finding.desired_type||'not specified')+')':'')+
          (candidate?'. Current extraction method: '+candidate.method:'')+
          '. Use the attached current record. Show the exact NLI selector, separators or type settings that fit this field. Preserve all other fields.';
        const originalDialog=button.closest('dialog');
        if(originalDialog){
          const form=originalDialog.querySelector('form');
          form.elements.question.value=question;
          form.elements.question.focus();
          break;
        }
        el('workspace-ai-dialog')?.remove();
        const dialog=document.createElement('dialog');
        dialog.id='workspace-ai-dialog';dialog.className='panel';
        dialog.innerHTML=`<h3>Ask Nexora AI · ${esc(field)}</h3>${workspaceAIForm({mode:'ask',tab:'Field mapping',raw:draft.raw})}<button type="button" class="secondary" data-action="workspace-ai-close">Close</button>`;
        page.append(dialog);
        dialog.querySelector('[name=question]').value=question;
        dialog.showModal();
        break;
      }
      case 'workspace-ai-open': {
        const existing=el('workspace-ai-dialog');existing?.remove();
        const dialog=document.createElement('dialog'),tab=button.dataset.tab||page.querySelector('[role=tab][aria-selected=true]')?.textContent||'';
        dialog.id='workspace-ai-dialog';
        dialog.className='panel';
        const raw=workspaceAISample(button);
        dialog.innerHTML=`<h3>${button.dataset.mode==='fix'?'Fix with Nexora AI':'Ask Nexora AI'} · ${esc(tab||TITLES[state.page])}</h3>${workspaceAIForm({mode:button.dataset.mode,tab,raw,defaultArtifact:workspaceAIArtifact(button,raw)})}<button type="button" class="secondary" data-action="workspace-ai-close">Close</button>`;
        page.append(dialog);
        dialog.showModal();
        break;
      }
      case 'workspace-ai-auto': {
        const raw=workspaceAISample(button);
        const tab=button.dataset.tab||page.querySelector('[role=tab][aria-selected=true]')?.textContent||'';
        const artifact=workspaceAIArtifact(button,raw);
        const bar=button.closest('.actions');
        if(!bar)break;
        const anchor=bar.closest('form')||bar.closest('.section-head')||bar;
        if(anchor.nextElementSibling?.hasAttribute('data-workspace-ai-auto-panel'))anchor.nextElementSibling.remove();
        const section=document.createElement('section');
        section.className='panel';
        section.dataset.workspaceAiAutoPanel='';
        if(anchor.dataset.qrwPanel){
          section.dataset.qrwPanel=anchor.dataset.qrwPanel;
          section.classList.add('qrw-stage-panel');
        }
        anchor.after(section);
        section.innerHTML=`<h4>${button.hasAttribute('data-map-sample')?'Suggest &amp; map with Nexora AI':button.hasAttribute('data-suggest-mapping')?'Suggest Mappings with Nexora AI':'Auto Fix with Nexora AI'} · editable draft</h4>${workspaceAIForm({mode:'fix',tab,raw,defaultArtifact:artifact||'mapping'})}`;
        const form=section.querySelector('form');
        if(raw.trim())form.aiAutoSource=button;
        form.applyVerifiedMapping=!button.hasAttribute('data-suggest-mapping');
        form.applyNativeMapping=button.hasAttribute('data-map-sample');
        for(const name of ['question','desired','artifact'])form.elements[name].closest('label').hidden=true;
        const rawLabel=form.elements.raw.closest('label');
        rawLabel.firstChild.nodeValue=artifact==='kv_template'?'Paste only the key/value portion':'Paste a representative sample record';
        rawLabel.hidden=!!form.aiAutoSource;
        form.elements.raw.required=true;
        form.querySelector('[type=reset]').hidden=true;
        const build=form.querySelector('[data-action=workspace-ai-run]');
        if(!form.aiAutoSource){form.elements.raw.focus();break;}
        build.hidden=true;
        button.disabled=true;
        try{
          await runAndApplyWorkspaceAI(build);
        }
        finally{button.disabled=false;build.hidden=false;build.textContent='Rebuild draft';}
        break;
      }
      case 'map-guided-sample': await mapGuidedSample();break;
      case 'workspace-ai-close': el('workspace-ai-dialog')?.remove();break;
      case 'workspace-ai-run': await runAndApplyWorkspaceAI(button);break;
      case 'workspace-ai-download': {
        const proposal=button.closest('form').aiResult?.result?.proposal;
        if(proposal)downloadJSON('nli-ai-draft.json',proposal);
        break;
      }
      case 'workspace-ai-import': {
        const form=button.closest('form'),draft=form.aiResult;
        if(!draft||draft.raw!==form.elements.raw.value||(form.aiAutoSource&&draft.raw!==workspaceAISample(form.aiAutoSource)))throw new Error('The sample changed. Build a new draft.');
        if(draft.mappingSignature!=null&&(qrwRaw()!==draft.raw||JSON.stringify(qrwPlan())!==draft.mappingSignature))throw new Error('The editable mapping changed. Build a new proposal before applying.');
        const {parser,plan,config,kind}=draft.result.proposal;
        if(config){
          const route={regex:'regex',template:state.page==='studio'?'studio':'header-templates',pattern:'static',kv_template:state.page==='kv'?'kv':'kv-templates',key_pack:'knownkeys',keys:'knownkeys'}[kind];
          if(!route)throw new Error('Download this draft to review its configuration.');
          if(state.page!==route)await navigate(route);
          const editor=el('config-form');
          if(!editor)throw new Error('The configuration editor is unavailable.');
          if(kind==='keys'||kind==='key_pack'){
            const {keys,...metadata}=config;
            editor.elements.known_keys.value=(keys||[]).join('\n');
            editor.elements.config.value=pretty(metadata);
          }else editor.elements.config.value=pretty(config);
          page.dataset.dirty='true';
          form.closest('dialog')?.close();
          editor.scrollIntoView({block:'start'});
          break;
        }
        if(!button.applyVerifiedDraft&&!confirm('Apply this proposal to the editable FieldSense draft?'))break;
        const service=Array.isArray(parser?.service_scope)?parser.service_scope.join(','):parser?.service_scope;
        if(state.page==='building-blocks'&&el('block-raw').value===draft.raw){
          el('block-fields').replaceChildren();
          plan.fields.forEach(field=>addBuildingBlock(field.method==='constant',null,field));
          rememberBlockBoard();
          el('workspace-ai-dialog')?.remove();
          form.closest('[data-workspace-ai-auto-panel]')?.remove();
          el('block-status').textContent=draft.result.proposal_origin==='native_fallback'?'Verified native draft loaded. Previewing the exact sample…':'Nexora AI draft loaded. Previewing the exact sample…';
          await previewBuildingBlocks();
          break;
        }
        const analysisRaw=el('assurance-form')?.elements.raw?.value;
        const analysisDraft=state.page==='assurance'&&analysisRaw===draft.raw&&
          !form.closest('[data-assurance-panel=gprs]')&&aiEvidenceSource(draft.raw,form.elements.evidence_source?.value)==='manual';
        if(analysisDraft&&state.gprsManual?.raw!==draft.raw){
          resetGPRSSource();
          state.gprsManual={raw:draft.raw,service:service||'auto',parserId:'',nodeId:'',verification:null};
        }
        const insideMapping=analysisDraft||page.querySelector('#qrw-field-rows')&&qrwRaw()===draft.raw;
        applyWorkspaceMapping(draft.raw,plan,insideMapping&&!analysisDraft?qrwScope().service_scope:service);
        if(insideMapping){
          state.qrwStep=4;
          if(analysisDraft){
            page.querySelector('#gprs-preview-form [name=raw_evidence]').value=draft.raw;
            el('gprs-manual-raw').value=draft.raw;
            el('gprs-manual-service').value=state.gprsManual.service;
            page.querySelector('[data-assurance-tab=gprs]')?.click();
          }
          el('qrw-field-rows').innerHTML=renderQRWFieldRows();
          qrwTestFields();
          nfiDecorate();
          setupQRWStepPanels();
          page.dataset.dirty='true';
          el('workspace-ai-dialog')?.remove();
          form.closest('[data-workspace-ai-auto-panel]')?.remove();
        }
        else await navigate('field-mapping');
        nfiScheduleLive();
        if(el('qrw-field-rows')&&qrwRaw()===draft.raw){
          await qrwVerifyController();
          if(button.applyVerifiedDraft){
            const origin=draft.result.proposal_origin==='native_fallback'?'Verified native fallback draft':'Nexora AI draft';
            const liveState=el('nfi-live-state');
            if(liveState)liveState.textContent+=` ${origin} applied. Review fields, then use Generate + validate complete parser pack.`;
          }
        }
        break;
      }
      case 'block-add': addBuildingBlock(false);break;
      case 'block-whole': {
        if(!el('block-raw').value.trim())throw new Error('Paste a raw record first.');
        addBuildingBlock(false,null,{method:'raw',name:'raw_record',type:'string'});
        el('block-fields').lastElementChild.querySelector('details').open=true;
        break;
      }
      case 'block-piece': addSourcePiece(Number(button.dataset.index));break;
      case 'block-from-mapping': {
        state.blockSeed=qrwRaw();
        await navigate('building-blocks');
        break;
      }
      case 'block-literal': addBuildingBlock(true);break;
      case 'block-up': {
        const row=button.closest('[data-block-card]');
        if(row.previousElementSibling)row.previousElementSibling.before(row);
        rememberBlockBoard();
        break;
      }
      case 'block-down': {
        const row=button.closest('[data-block-card]');
        if(row.nextElementSibling)row.nextElementSibling.after(row);
        rememberBlockBoard();
        break;
      }
      case 'block-remove': button.closest('[data-block-card]').remove();rememberBlockBoard();break;
      case 'block-reset': el('block-fields').replaceChildren();el('block-raw').value='';renderBlockSource();rememberBlockBoard();el('block-status').textContent='';break;
      case 'block-sense': {
        const input=el('block-raw'),raw=input.value,fields=el('block-fields');
        if(!raw.trim())throw new Error('Paste a raw record to sense its fields.');
        const original=fields.children.length?buildingBlockPlan().fields:[];
        const status=el('block-status');
        status.textContent='Looking for fields in this sample…';
        let sensed, fallback=false;
        try{
          const result=await nfiAnalysisRequest('/field-mapping/suggest',{samples:[raw],service:'auto'});
          sensed=result.fields||result.profiles?.[0]?.fields;
        }catch(error){
          if(error.name==='AbortError')return;
          fallback=true;
        }
        if(!input.isConnected||input.value!==raw||JSON.stringify(original)!==JSON.stringify(fields.children.length?buildingBlockPlan().fields:[]))return;
        const existing=new Set(original.map(field=>field.name));
        const suggestions=(sensed?.length?sensed:qrwSuggestedFields(raw,[])).filter(field=>!existing.has(field.name)).slice(0,128-original.length);
        suggestions.forEach(field=>addBuildingBlock(field.method==='constant',null,field));
        status.textContent=`${suggestions.length} editable fields added. ${fallback?'Local fallback used. ':''}Verify output before generating the parser.`;
        break;
      }
      case 'block-preview': {
        await previewBuildingBlocks();
        break;
      }
      case 'block-generate':
      case 'block-open': {
        const plan=buildingBlockPlan();
        applyWorkspaceMapping(el('block-raw').value,plan);
        state.nfiDesired=null;
        await navigate('field-mapping');
        if(state.page!=='field-mapping'||!el('qrw-field-rows'))break;
        if(button.dataset.action==='block-generate')await action({dataset:{action:'qrw-generate-validate'}});
        else nfiScheduleLive();
        break;
      }
      case 'gprs-reset': {
        if(!confirm('Reset the current Guided Parser Repair session?'))break;
        resetGPRSSource();
        state.lastAssurance=null;
        state.nfiRaw='';
        state.nfiDesired=null;
        state.qrwStep=1;
        await navigate(state.page);
        if(state.page==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();
        break;
      }
      case 'assurance-reset-sample': {
        const form=button.closest('form');
        form.reset();
        const raw=form.querySelector('#assurance-raw,[name=raw]');
        if(raw)raw.value='';
        state.lastAssurance=null;
        el('assurance-result')?.replaceChildren();
        page.dataset.dirty='true';
        break;
      }
      case 'field-sense-reset': {
        if(!confirm('Reset the current FieldSense mapping session?'))break;
        resetGPRSSource();
        Object.assign(state,{nfiRaw:'',nfiCorpus:'',nfiDesired:null,nfiParser:null,nfiCases:[],nfiEditId:'',nfiName:'',nfiVersion:'',nfiCandidateValidated:null,nfiGuideStep:undefined});
        state.lastAssurance=null;
        await navigate('field-mapping');
        break;
      }
      case 'nfi-ai-status': {const result=await api('/ai/capabilities');el('nfi-ai-status').textContent=pretty(result);break;}
      case 'nfi-ai-help': {await action({dataset:{action:'workspace-ai-open',mode:'ask',tab:'Field mapping'}});break;}
      case 'ai-discover-models': {
        const id = page.querySelector('#ai-provider-form [name=id]')?.value;
        if (!id) throw new Error('Load or save a provider profile first.');
        const status = el('ai-model-discovery-status');
        status.textContent='Loading provider model catalog…';
        try {
          const result=await api('/admin/ai-provider/models',{method:'POST',body:{id}});
          if (!status.isConnected || page.querySelector('#ai-provider-form [name=id]')?.value!==id) break;
          const list=el('ai-model-options'), existing=new Set([...list.options].map(option=>option.value));
          for (const model of result.models || []) {
            if (!existing.has(model.id)) list.append(new Option(model.label || model.id,model.id));
          }
          status.textContent=(result.models?.length || 0)+' model IDs loaded'+(result.truncated?' (catalog limit reached)':'')+'. Test the selected model to verify access.';
        } catch (error) {
          if (status.isConnected) status.textContent=error.message;
          throw error;
        }
        break;
      }
      case 'test-ai-provider': {page.dataset.dirty='true'; const form=button.closest('section')?.querySelector('#ai-provider-form');if(form&&!form.reportValidity())throw new Error('Enter the profile ID, display name, exact model and valid endpoint first.');const data=form?formObject(form):{};const body={id:data.id,name:data.name,provider:data.provider,model:data.model,endpoint:data.endpoint,credential_source:data.credential_source,credential_ref:data.credential_ref,stream:data.test_mode==='stream',test_max_tokens:Number(data.test_max_tokens||64),image_url:data.test_image_url||'',timeout_seconds:Number(data.timeout_seconds||15),max_response_bytes:Number(data.max_response_bytes||1048576)};if(data.api_key&&data.credential_source==='encrypted_inline')body.api_key=data.api_key;const progress=notify('Testing AI provider connection…','progress');try{const result=await api('/admin/ai-provider/test',{method:'POST',body});const tested=result.tested_config||{},description=[tested.configuration||'draft test',tested.profile_id||data.id,tested.model||data.model,tested.endpoint_host||''].filter(Boolean).join(' · ');const message=`${result.message||'Provider test completed'} (${description})`;const status=el('ai-provider-test-status');if(status)status.textContent=pretty({...result,tested_config:tested});if(result.connected)progress.done(message);else if(result.acknowledged&&result.error_code==='incomplete_output')notify(message,'warning');else progress.fail(message);}catch(error){progress.fail(error.message);throw error;}break; }
      case 'parser-action': { const doc=(state.parserPacks||[]).find(item=>String(item.id)===String(id))||await api(`/parser-packs/${encodeURIComponent(id)}`),body={};if(value==='reject'){const reason=prompt('Why is this parser pack being rejected?');if(reason===null)break;if(reason.trim().length<3)throw new Error('A rejection reason of at least 3 characters is required');body.reason=reason.trim();}if(value==='approve'&&doc.created_by===state.user?.username){if(state.user?.role!=='admin')throw new Error('A different administrator must approve this parser pack');const reason=prompt(`Administrator exact-revision approval for ${id} revision ${doc.revision}. Enter the operational reason (minimum 12 characters):`);if(reason===null)break;if(reason.trim().length<12)throw new Error('Approval reason must contain at least 12 characters');const checksum=doc.approval_checksum||doc.payload?.artifact_checksum||doc.sha256;if(!checksum)throw new Error('Parser checksum is unavailable; refresh before approving');Object.assign(body,{emergency_self_approval:true,reason:reason.trim(),expected_revision:Number(doc.revision),expected_checksum:checksum});}else if(value==='approve'&&state.user?.role==='admin'){const reason=prompt('Approval comment (optional, 3–512 characters):');if(reason===null)break;if(reason.trim())body.reason=reason.trim();}await api(`/parser-packs/${encodeURIComponent(id)}/${value}`,{method:'POST',body});toast(`Parser ${value} complete`);await navigate('parsers');break; }
      case 'rollout-approve': {
        const target=['quarantine-wizard','assurance'].includes(state.page)?state.page:'rollouts',proposal=(state.parserRolloutProposals||[]).find(item=>item.id===id);
        if(!proposal)throw new Error('This rollout proposal is no longer in the current controller snapshot. Refresh and review it again.');
        const reasonInput=button.closest('.rollout-approval-editor')?.querySelector('[data-rollout-reason]'),reason=String(reasonInput?.value||'').trim(),isCreator=proposal.created_by===state.user?.username,emergency=isCreator&&state.user?.role==='admin',payload=proposal.payload||{},bundleHash=rolloutBundleHash(proposal),currentBundleHash=state.assuranceSyncBundle?.desired_hash||'';
        if(isCreator&&!emergency)throw new Error('A different authorized reviewer must approve this rollout');
        if(reason.length<(emergency?12:3))throw new Error(emergency?'Approval reason must contain at least 12 characters':'Enter an independent review reason before approval');
        const problem=rolloutApprovalProblem(proposal,isParserSyncProposal(proposal)?currentBundleHash:'');
        if(problem)throw new Error(problem);
        const body={reason};
        if(emergency&&isParserSyncProposal(proposal)){const exactApprovalHash=/^[0-9a-f]{64}$/i.test(bundleHash)?bundleHash:currentBundleHash;if(!/^[0-9a-f]{64}$/i.test(exactApprovalHash))throw new Error('The exact current parser bundle hash is unavailable. Refresh before administrator approval.');Object.assign(body,{emergency_self_approval:true,expected_revision:Number(proposal.revision),expected_bundle_hash:exactApprovalHash});}
        try{
          const result=await api(`/rollouts/${encodeURIComponent(id)}/approve`,{method:'POST',body});
          delete state.rolloutActionErrors[id];state.qrwStep=8;toast(`Rollout proposal ${id} approved${result.id?` · live rollout ${result.id}`:''}`);await refreshApprovalBadge();await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();
        }catch(error){
          if(![409,422].includes(error.status))throw error;
          state.rolloutActionErrors[id]=rolloutApprovalErrorMessage(error);state.qrwStep=8;await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();
        }
        break;
      }
      case 'rollout-cancel': {const target=['quarantine-wizard','assurance'].includes(state.page)?state.page:'rollouts';await api(`/rollouts/${encodeURIComponent(id)}/cancel`,{method:'POST',body:{}});delete state.rolloutActionErrors[id];state.qrwStep=8;toast(`Rollout proposal ${id} cancelled`);await refreshApprovalBadge();await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();break;}
      case 'purge-case': if(confirm('Permanently purge this encrypted fixture and every revision?')){await api(`/assurance/cases/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast('Fixture purged');await navigate('assurance');}break;
      case 'download-case': { const progress=notify('Downloading retained fixture…','progress');try{const response=await fetchWithTimeout(`/api/v1/assurance/cases/${encodeURIComponent(id)}/download`,{credentials:'same-origin',headers:{Accept:'text/plain, application/octet-stream'}},55000);if(!response.ok){const data=await response.json().catch(()=>({}));throw new Error(data.error||`Download failed (${response.status})`);}const blob=await response.blob();downloadBlob(`nli-v21-case-${id}.log`,blob);notify('Retained fixture downloaded');}catch(error){progress.fail(error.message);throw error;}break; }
      case 'download-upload-fixture': { const progress=notify('Downloading retained sample from the authenticated controller…','progress');try{if(!id)throw new Error('The controller did not return a retained fixture ID.');const response=await fetchWithTimeout(`/api/v1/assurance/cases/${encodeURIComponent(id)}/download`,{credentials:'same-origin',headers:{Accept:'text/plain, application/octet-stream'}},55000);if(!response.ok){const result=await response.json().catch(()=>({}));throw new Error(result.error||`Authenticated fixture download failed (${response.status})`);}downloadBlob(button.dataset.filename||`nli-v21-fixture-${id}.log`,await response.blob());notify('Retained fixture downloaded through the controller');}catch(error){progress.fail(error.message);throw error;}break; }
      case 'navigate': await navigate(button.dataset.page);break;
      case 'qrw-controller-preview': {await qrwVerifyController();break;}
      case 'qrw-token-field': {if(state.qrwFields.length>=128)throw new Error('A mapping can contain at most 128 fields');const token=Number(button.dataset.token),item=qrwTokens(qrwRaw())[token-1];if(!item)throw new Error('Token is no longer present');state.qrwFields.push(qrwField({name:`token_${token}_custom`,label:`Token ${token}`,method:'position',source:String(token),policy:'required',preview:item.value,status:'unresolved'}));el('qrw-field-rows').innerHTML=renderQRWFieldRows();break;}
      case 'qrw-selection-field': {const selection=window.getSelection(),anchor=selection?.anchorNode?.parentElement,focus=selection?.focusNode?.parentElement,host=el('nfi-raw-selection');if(!selection||!selection.toString()||!host||!host.contains(anchor)||!host.contains(focus))throw new Error('Select text inside the raw evidence explorer first.');const range=selection.getRangeAt(0),prefix=range.cloneRange();prefix.selectNodeContents(host);prefix.setEnd(range.startContainer,range.startOffset);const start=[...prefix.toString()].length+1,end=start+[...range.toString()].length-1;if(state.qrwFields.length>=128)throw new Error('A mapping can contain at most 128 fields');state.qrwFields.push(qrwField({name:`characters_${start}_${end}`,label:'Selected raw characters',method:'character',source:String(start),source_end:String(end),policy:'required',preview:range.toString(),status:'unresolved'}));el('qrw-field-rows').innerHTML=renderQRWFieldRows();break;}
      case 'kv-bracket-preset':
      case 'kv-input-add':
      case 'kv-input-remove': {
        const field=state.qrwFields.find(item=>item.id===id);
        if(!field||!isKVTail(field))throw new Error('Choose a dynamic KV row first');
        const edits=field.kv_input_edits||[];
        if(button.dataset.action==='kv-bracket-preset'){
          Object.assign(field,{kv_input_edits:[{action:'replace',find:'][',replace:'|'},{action:'strip_prefix',find:'[',required:true},{action:'strip_suffix',find:']',required:true}],
            kv_pair_separators:['|'],pair_delimiters:'|',assignment_separators:[':', ';', '=']});
        }else if(button.dataset.action==='kv-input-add'){
          if(edits.length>=16)throw new Error('A KV row supports up to 16 input edits');
          field.kv_input_edits=[...edits,{action:'replace',find:'',replace:''}];
        }else{
          field.kv_input_edits=edits.filter((_,index)=>index!==Number(button.dataset.index));
        }
        Object.assign(field,{kv_profile:'configured',require_complete:true,status:'unresolved'});
        qrwControllerPreview=null;
        el('qrw-field-rows').innerHTML=renderQRWFieldRows();
        nfiScheduleLive();
        nfiDecorate();
        if(button.dataset.action==='kv-input-add')page.querySelector(`[data-qrw-row="${CSS.escape(id)}"] [data-kv-input-index]:last-of-type [data-kv-input-key=find]`)?.focus();
        break;
      }
      case 'qrw-operation-add': {const field=state.qrwFields.find(item=>item.id===id);if(!field)throw new Error('Field no longer present');field.operations=field.operations||[];if(field.operations.length>=16)throw new Error('A field supports at most 16 operations');field.operations.push({op:button.dataset.operation||'split',delimiter:';',part:1,start:1,end:0,value:'',find:'',replace:'',pattern:'',group:'',fields:[],separator:' '});field.status='unresolved';qrwControllerPreview=null;el('qrw-field-rows').innerHTML=renderQRWFieldRows();page.querySelector(`[data-qrw-row="${CSS.escape(id)}"] .nfi-operations`)?.setAttribute('open','');nfiScheduleLive();break;}
      case 'qrw-operation-delete': case 'qrw-operation-up': {const field=state.qrwFields.find(item=>item.id===id),index=Number(button.dataset.index);if(!field?.operations?.[index])throw new Error('Operation no longer present');if(button.dataset.action==='qrw-operation-delete')field.operations.splice(index,1);else if(index>0)[field.operations[index-1],field.operations[index]]=[field.operations[index],field.operations[index-1]];field.status='unresolved';el('qrw-field-rows').innerHTML=renderQRWFieldRows();page.querySelector(`[data-qrw-row="${CSS.escape(id)}"] .nfi-operations`)?.setAttribute('open','');break;}
      case 'qrw-export-plan': {downloadJSON('nli-v21-native-field-mapping.json',{...qrwPlan(),...qrwScope()});break;}
      case 'qrw-import-plan': {const envelope=parseJSON(el('nfi-plan-import').value,'Native mapping plan'),plan=envelope.mapping_plan||envelope;state.nfiService=Array.isArray(envelope.service_scope)?envelope.service_scope.join(','):envelope.service_scope||state.nfiService||'auto';state.nfiMappingPriority=Number(envelope.mapping_priority)||0;const priorityControl=el('nfi-mapping-priority');if(priorityControl)priorityControl.value=state.nfiMappingPriority;state.nfiScopePattern=envelope.scope_pattern||'';state.nfiGlobalScope=envelope.reviewed_global_scope===true;const serviceControl=el('nfi-service'),patternControl=el('nfi-scope-pattern'),globalControl=el('nfi-global-scope');if(serviceControl)serviceControl.value=state.nfiService;if(patternControl)patternControl.value=state.nfiScopePattern;if(globalControl)globalControl.checked=state.nfiGlobalScope;if(plan?.version!==1||!Array.isArray(plan.fields)||!plan.fields.length||plan.fields.length>128)throw new Error('Import a version 1 mapping plan with 1–128 fields.');if(plan.fields.some(field=>!field||typeof field!=='object'||Array.isArray(field)||!QRW_METHODS.includes(field.method)||!Array.isArray(field.operations||[])||(field.operations||[]).length>16))throw new Error('Mapping plan contains an unsupported field or operation count');state.qrwFields=plan.fields.map(field=>qrwField({...field,preview:'',status:'unresolved',confidence:'imported',evidence:'Imported mapping needs verification'}));qrwControllerPreview=null;el('qrw-field-rows').innerHTML=renderQRWFieldRows();toast('Mapping imported for review; no parser activated');break;}
      case 'nfi-refresh-inspector': {state.nfiRaw=qrwRaw();el('nfi-inspector-host').innerHTML=qrwRawInspector();break;}
      case 'nfi-import-file': {el('nfi-sample-file')?.click();break;}
      case 'nfi-learn': {const corpus=el('nfi-corpus').value||qrwRaw(),samples=corpus.split(/\r?\n/).filter(line=>line.trim());if(!samples.length||samples.length>100||new TextEncoder().encode(corpus).length>1048576)throw new Error('Use 1–100 sample records within 1 MiB.');state.nfiCorpus=corpus;state.nfiSeparator=el('nfi-separator').value;const result=await nfiAnalysisRequest('/field-mapping/suggest',{samples,separator:state.nfiSeparator==='\\t'?'\t':state.nfiSeparator});state.nfiProfiles=result.profiles||[];const fields=result.fields||state.nfiProfiles[0]?.fields;if(Array.isArray(fields)&&fields.length){state.qrwFields=fields.map(field=>qrwField({...field,status:field.method==='delimiter'&&!field.confirmed?'needs_confirmation':'unresolved'}));el('qrw-field-rows').innerHTML=renderQRWFieldRows();}el('nfi-learning-status').textContent=`${result.sample_count??samples.length} samples inspected · ${state.nfiProfiles.length} structural profile(s). Suggestions require review; optional keys and identifiers retain their observed evidence.`;el('nfi-profile-list').innerHTML=state.nfiProfiles.map((profile,index)=>`<div class="nfi-profile"><b>${esc(profile.name||profile.signature||`Profile ${index+1}`)}</b><span>${number(profile.sample_count??profile.count)} samples</span><button class="secondary small" type="button" data-action="nfi-use-profile" data-index="${index}">Review this profile</button><pre>${esc(pretty(profile.fields?.map(field=>({name:field.name,type:field.type,policy:field.policy,confidence:field.confidence,evidence:field.evidence}))||profile))}</pre></div>`).join('');qrwControllerPreview=null;break;}
      case 'nfi-use-profile': {nfiLoadProfile(Number(button.dataset.index));break;}
      case 'nfi-repair-candidate': {const result=await api('/assurance/parsers/'+encodeURIComponent(id)+'/clone-draft',{method:'POST',body:{reason:'Operator requested editable FieldSense repair revision'}});if(!result.id)throw new Error('Controller did not return a repair draft identifier');await action({dataset:{action:'nfi-edit-candidate',id:result.id}});toast('Repair draft created. The original production revision is preserved; validate and review this draft before any deployment.');break;}
      case 'nfi-edit-candidate': {const doc=await api('/assurance/parsers/'+encodeURIComponent(id)),payload=doc.payload||doc,parser=payload.parser;if(!['draft','rejected'].includes(doc.state))throw new Error('Only draft/rejected candidates can be edited');if(parser?.pattern_type!=='mapping'){await navigate('parser-workspace');await action({dataset:{action:'parser-workspace-load',id}});toast('Canonical repair draft loaded in Parser Creation Workspace; original production revision is preserved.');break;}state.nfiEditId=id;state.nfiName=payload.name||id;state.nfiVersion=payload.version||'0.1.0';state.nfiParser=parser;state.nfiCases=payload.cases||[];state.nfiRaw=state.nfiCases[0]?.raw||'';state.nfiService=Array.isArray(parser.service_scope)?parser.service_scope.join(','):parser.service_scope||'auto';state.nfiMappingPriority=Number(parser.mapping_priority)||0;state.nfiScopePattern=parser.scope_pattern||'';state.nfiGlobalScope=parser.reviewed_global_scope===true;state.qrwFields=(parser.mapping_plan?.fields||[]).map(field=>qrwField({...field,status:'unresolved'}));qrwControllerPreview=null;await navigate('field-mapping');toast('Native plan loaded for an editable revision; service scope and ordered operations preserved');break;}
      case 'reset-source-sample': {
        const input = button.closest('[data-sample-template-actions]')?.previousElementSibling;
        if (!(input instanceof HTMLTextAreaElement)) break;
        input.value = '';
        input.dispatchEvent(new Event('input',{bubbles:true}));
        input.focus();
        break;
      }
      case 'template-from-sample': {
        const source = button.closest('[data-sample-template-actions]')?.previousElementSibling;
        const raw = source?.value ?? workspaceAISample(button);
        if (!raw.trim()) throw new Error('Load or paste a raw log sample first.');
        await navigate(button.dataset.kind);
        const sample = el('template-composition-sample');
        if (sample && state.page === button.dataset.kind) {
          sample.value = raw;
          sample.dispatchEvent(new Event('input',{bubbles:true}));
          sample.focus();
        }
        break;
      }
      case 'template-reset-samples': {
        const sample = el('template-composition-sample');
        sample.value = '';
        for (const id of ['template-draft-result','template-composition-result']) {
          const output = el(id);
          output.textContent = '';
          output.hidden = true;
          output.verifiedComposition = null;
          output.parentElement.querySelector('[data-action=template-open-verified]')?.remove();
        }
        sample.focus();
        break;
      }
      case 'template-draft-generate':
      case 'template-draft-verify': {
        const input = el('template-composition-sample');
        const host = el('template-draft-result');
        const editor = el('config-form').elements.config;
        const raw = input.value, original = editor.value;
        const samples = raw.split(/\r?\n/).filter(line => line.trim());
        if (!samples.length || samples.length > 500) throw new Error('Paste 1 to 500 sample records.');
        const generate = button.dataset.action === 'template-draft-generate';
        const body = {kind:button.dataset.artifact,samples};
        if (!generate) body.config=parseJSON(original,'Configuration JSON');
        if (!generate && (!body.config || Array.isArray(body.config) || typeof body.config!=='object')) throw new Error('Enter a configuration JSON object.');
        page.dataset.dirty='true';
        host.hidden=false;
        host.textContent='Checking the supplied samples…';
        button.disabled=true;
        try {
          const result=await api('/field-mapping/template-draft',{method:'POST',body});
          if (!host.isConnected) break;
          if (raw!==input.value || original!==editor.value) {
            host.textContent='The sample or JSON changed. Check the current values again.';
            break;
          }
          if (generate && result.passed && result.config) {
            editor.value=pretty(result.config);
            page.dataset.dirty='true';
          }
          host.textContent=(result.records_requiring_kv_template > 0
            ? 'Header verified. Verify the remaining key/value data with a KV template for a complete parse.\n\n' : '')+pretty(result);
        } catch(error) {
          if (host.isConnected) host.textContent=error.message;
        } finally {button.disabled=false;}
        break;
      }
      case 'template-preview-saved': {
        const header = el('template-header').value, kv = el('template-kv').value;
        const raw = el('template-composition-sample').value;
        const samples = raw.split(/\r?\n/).filter(line => line.trim());
        if (!header || !kv || !samples.length) throw new Error('Select both saved revisions and enter sample logs.');
        if (samples.length>500) throw new Error('Preview up to 500 lines at a time.');
        const resultHost = el('template-composition-result');
        resultHost.hidden=false;
        resultHost.verifiedComposition=null;
        resultHost.parentElement.querySelector('[data-action=template-open-verified]')?.remove();
        resultHost.textContent='Checking exact samples…';
        try {
          const result = await api('/field-mapping/template-compose',{method:'POST',body:{header:JSON.parse(header),kv:JSON.parse(kv),samples}});
          if (!resultHost.isConnected) break;
          if (el('template-header').value!==header || el('template-kv').value!==kv || el('template-composition-sample').value!==raw) {
            resultHost.textContent='The templates or samples changed. Verify the current selection again.';
            break;
          }
          resultHost.textContent=pretty({passed:result.passed,total:result.total,failed:result.failed,examples:result.examples,failures:result.failures});
          if (result.failed===0 && result.plan?.fields?.length) {
            resultHost.insertAdjacentHTML('afterend','<button type="button" class="secondary" data-action="template-open-verified">Open verified mapping in FieldSense</button>');
            resultHost.verifiedComposition={plan:result.plan,raw:samples[0],scope:result.scope_pattern,header,kv,source:raw};
          }
        } catch (error) {
          if (resultHost.isConnected) resultHost.textContent=error.message;
          throw error;
        }
        break;
      }
      case 'template-open-verified': {
        const verified=el('template-composition-result')?.verifiedComposition;
        if (!verified || el('template-header').value!==verified.header || el('template-kv').value!==verified.kv ||
            el('template-composition-sample').value!==verified.source) throw new Error('Verify the current template and sample selection first.');
        state.nfiRaw=verified.raw;
        state.nfiScopePattern=new TextEncoder().encode(verified.scope || '').length <= 1024 ? verified.scope || '' : '';
        state.nfiService='auto';
        state.nfiGlobalScope=false;
        await navigate('field-mapping');
        state.qrwFields=verified.plan.fields.map(field=>qrwField(field.kv_header_pattern?{...field,source:'raw'}:field));
        qrwControllerPreview=null;
        el('qrw-field-rows').innerHTML=renderQRWFieldRows();
        nfiScheduleLive();
        break;
      }
      case 'template-compose-mapping': {
        const raw = el('template-composition-sample')?.value.split(/\r?\n/).find(line=>line.trim());
        if (!raw) throw new Error('Paste one representative log record.');
        state.nfiRaw = raw;
        state.nfiScopePattern = /^[0-9]{8}\s+\S+\s+\S+\s+\S+\s+[0-9]+\s+[0-9]+\s+[0-9]+\s+(?:[0-9]+;(?:[^;\s]+;)?)?[^;\s]+;[^;\s]+(\s|$)/.source;
        state.nfiService='auto';
        state.nfiGlobalScope=false;
        await navigate('field-mapping');
        const header = ['date','timestamp','hostname','service','pid','minor_pid','thread_id'];
        state.qrwFields = header.map((name,index) => qrwField({
          name,label:name,method:'position',source:String(index+1),policy:'required',
        }));
        const eventHeader = CONFIG['header-templates'][2].pattern;
        for (const name of ['random_number','extra_tag','event_type','event','event_code']) {
          state.qrwFields.push(qrwField({name,label:name,method:'regex',source:eventHeader,part:name,
            policy:CONFIG['header-templates'][2].optional_fields.includes(name)?'optional':'required'}));
        }
        state.qrwFields.push(qrwField({name:'kv_pair',label:'Dynamic KV tail',method:'kv_eol',end_of_line:true,
          type:'object',policy:'optional',source:'9',unit:'token',output_policy:'flatten',kv_profile:'configured',
          assignment_separators:['='],pair_delimiters:':;|',require_complete:true,kv_literal_boundaries:true,kv_max_pairs:512}));
        qrwControllerPreview=null;
        el('qrw-field-rows').innerHTML=renderQRWFieldRows();
        nfiScheduleLive();
        break;
      }
      case 'nfi-open-context': {const form=button.closest('form'),sample=form?.elements.samples?.value||qrwRaw();if(sample)state.nfiRaw=sample.split(/\r?\n/).find(line=>line.trim())||sample;await navigate('field-mapping');break;}
      case 'nfi-handoff-output': {const parser=parseJSON(el('nfi-parser-json').value,'Native parser definition');if(parser.pattern_type!=='mapping'||!Array.isArray(parser.fields)||(!parser.fields.length&&!parser.mapping_plan?.fields?.some(field=>['kv_all','kv_eol'].includes(field.method)&&field.output_policy==='flatten')))throw new Error('Generate a native mapping candidate first');const columns=Object.fromEntries([...new Set([...parser.fields.map(field=>field.name),...Object.keys(state.nfiDesired||{})])].map(name=>[name,name]));await navigate('outputs');const input=page.querySelector('#output-profile-form [name=field_columns]');if(input)input.value=pretty(columns);page.querySelector('[data-endpoint-tab=validation]')?.click();page.dataset.dirty='true';toast('Mapped JSON field names copied into the output draft. Publish and deliver the parser before using those fields in production.');break;}
      case 'nfi-handoff-parser': {const parser=parseJSON(el('nfi-parser-json').value,'Native parser definition'),cases=parseJSON(el('nfi-cases-json').value,'Exact cases');if(parser.pattern_type!=='mapping'||!Array.isArray(cases)||!cases.length)throw new Error('Generate and verify the native mapping candidate first');const name=el('nfi-parser-name').value,version=el('nfi-parser-version').value;await navigate('parser-workspace');const form=page.querySelector('#parser-workspace-form');form.elements.parser.value=pretty(parser);form.elements.cases.value=pretty(cases);form.elements.samples.value=cases.map(item=>item.raw).join('\n');form.elements.name.value=name;form.elements.version.value=version;renderParserHelperDetails(parser,form);page.dataset.dirty='true';toast('The same native mapping plan is ready for validation and guarded saving');break;}
      case 'qrw-generate-validate': {if(!state.qrwFields.length)throw new Error('Suggest or manually map the raw fields first. Add Key value EOL when the record has a dynamic key/value tail.');const progress=notify('Generating and validating the complete static + key/value candidate…','progress');try{if(!(qrwControllerPreview?.passed===true&&qrwControllerPreview.signature===JSON.stringify({raw:qrwRaw(),plan:qrwPlan()})))await qrwVerifyController();if(state.page==='field-mapping'){nfiGenerate();await action({dataset:{action:'nfi-validate-candidate'}});}else{await action({dataset:{action:'gprs-open-editor'}});const form=page.querySelector('#gprs-parser-helper-form'),result=await qrwValidateCandidate(form,{recordWorkflow:true});state.qrwStep=5;setupQRWStepPanels();if(result.passed!==true)throw new Error('Complete parser validation failed. Review the exact result below, revise mappings and retry; no candidate was published.');form.querySelector('[data-parser-helper-status]').textContent='Complete parser validation passed. Save the review-gated draft here, then submit, approve and publish inside this wizard.';}progress.done('Complete candidate generated and validated. Save the review-gated draft to continue; no live parser changed.');}catch(error){progress.fail(error.message);throw error;}break;}
      case 'nfi-generate': {await qrwPrepareFields();nfiGenerate();break;}
      case 'nfi-save-candidate': {const parser=parseJSON(el('nfi-parser-json').value,'Native parser definition'),cases=parseJSON(el('nfi-cases-json').value,'Exact validation cases');if(parser.pattern_type!=='mapping'||!parser.mapping_plan||!Array.isArray(cases)||!cases.length)throw new Error('Generate a native mapping parser with exact expected cases first');const preview=await nfiAnalysisRequest('/assurance/parser-preview',{parser,cases});if(preview.passed!==true)throw new Error('Native candidate failed exact runtime validation; review the mapping and cases');const result=await api(state.nfiEditId?'/assurance/parsers/'+encodeURIComponent(state.nfiEditId):'/assurance/parsers',{method:state.nfiEditId?'PUT':'POST',body:{name:el('nfi-parser-name').value,version:el('nfi-parser-version').value,parser,cases,source_quarantine_record_id:''}});toast(`Native mapping draft ${result.id||'created'} saved; authorized review is still required`);state.nfiParser=parser;state.nfiCases=cases;state.nfiEditId=result.id||state.nfiEditId||'';state.nfiName=el('nfi-parser-name').value;state.nfiVersion=el('nfi-parser-version').value;await navigate('field-mapping');break;}

      case 'qrw-apply-field-names': {const form=page.querySelector('#gprs-preview-form'),raw=form?.elements.raw_evidence.value||state.lastAssurance?.request?.raw||'';if(!raw)throw new Error('Inspect a retained raw record before defining output fields');const supplied=String(form.elements.manual_field_names.value||form.elements.manual_field_name.value||'').split(/[\n,]+/).map(value=>value.trim()).filter(Boolean);if(!supplied.length)throw new Error('Enter at least one desired JSON field name');if(supplied.length>128)throw new Error('A mapping can contain at most 128 fields');const invalid=supplied.find(name=>!/^[A-Za-z_][A-Za-z0-9_]{0,127}$/.test(name));if(invalid)throw new Error(`Invalid field name ${JSON.stringify(invalid)}. Use letters, digits, and underscores; the first character must be a letter or underscore.`);if(new Set(supplied).size!==supplied.length)throw new Error('Desired field names must be unique');const start=Number(form.elements.manual_field_index.value||1);if(!Number.isInteger(start)||start<1||start>128||start+supplied.length-1>128)throw new Error('The selected token range must stay between 1 and 128');const tokens=String(raw).trim().split(/\s+/).filter(Boolean);state.qrwFields=supplied.map((name,offset)=>{const position=start+offset,value=tokens[position-1];return qrwField({name,label:name.replaceAll('_',' '),policy:'required',method:'position',source:String(position),preview:value||'',evidence:value?`Observed token ${position}: ${value}`:`Token ${position} is absent in this sample`,confidence:value?'operator_named':'unverified',status:value?'mapped':'unresolved'});});state.qrwStep=3;const rows=el('qrw-field-rows');if(rows)rows.innerHTML=renderQRWFieldRows();const tested=qrwTestFields(),inspectStatus=el('qrw-inspect-status');if(inspectStatus)inspectStatus.textContent=tested.errors.length?`${tested.errors.length} field name mapping(s) need correction before continuing.`:`${state.qrwFields.length} operator-named fields mapped to observed tokens. Review types and extraction rules in Steps 3–4.`;setupQRWStepPanels();break;}
      case 'qrw-suggest-fields': {const raw=qrwRaw();if(!raw)throw new Error('Inspect a retained raw record before requesting field suggestions');const signature=JSON.stringify(qrwPlan());let result=null,notice='';try{result=await nfiAnalysisRequest('/field-mapping/suggest',{samples:[raw],service:state.nfiService||state.quarantineDiagnosis?.record?.service||'auto'});}catch(error){if(error.name==='AbortError'||/abort|cancel/i.test(error.message))throw error;notice='Controller parser-pool analysis is unavailable; local structural suggestions remain unverified. ';}if(qrwRaw()!==raw||JSON.stringify(qrwPlan())!==signature)throw new Error('Evidence or mapping changed during analysis; request suggestions for the current draft.');const fields=result?.fields||result?.profiles?.[0]?.fields;state.qrwFields=Array.isArray(fields)&&fields.length?qrwMergeControllerSuggestions(fields,state.qrwFields):qrwSuggestedFields(raw,state.qrwFields);state.qrwStep=4;qrwControllerPreview=null;el('qrw-field-rows').innerHTML=renderQRWFieldRows();const evidence=el('nfi-runtime-evidence'),json=el('nfi-runtime-evidence-json');if(evidence&&json){state.nfiRuntimeRaw=raw;state.nfiRuntimeFieldSuggestions=result?.runtime_field_suggestions||[];const useFields=evidence.querySelector('[data-action=nfi-use-runtime-fields]');if(useFields)useFields.hidden=!state.nfiRuntimeFieldSuggestions.length;evidence.hidden=false;json.textContent=pretty(result?.runtime||{available:false,diagnostics:[notice||'The controller returned structural suggestions without runtime-pool evidence.']});}const status=el('qrw-field-status');if(status)status.textContent=`${notice}${state.qrwFields.length} evidence-based suggestions created. Existing operator mappings are preserved; review proposed labels and delimiter splits.`;setupQRWStepPanels();break;}
      case 'nfi-use-runtime-fields': {if(state.nfiRuntimeRaw!==qrwRaw())throw new Error('Inspect the current sample with Suggest from raw log before adding observed fields');const names=new Set(state.qrwFields.map(field=>field.name)),fields=(state.nfiRuntimeFieldSuggestions||[]).filter(field=>field&&QRW_METHODS.includes(field.method)&&!names.has(field.name)).slice(0,128-state.qrwFields.length);state.qrwFields=qrwNormalizeSuggestions([...state.qrwFields,...fields.map(field=>qrwField({...field,status:'unresolved'}))]);el('qrw-field-rows').innerHTML=renderQRWFieldRows();qrwControllerPreview=null;nfiScheduleLive();toast(`${fields.length} observed fields added for review; existing mappings were preserved`);break;}
      case 'qrw-add-field': {if(state.qrwFields.length>=128)throw new Error('A mapping can contain at most 128 fields');state.qrwFields.push(qrwField());el('qrw-field-rows').innerHTML=renderQRWFieldRows();break;}
      case 'qrw-field-duplicate': {if(state.qrwFields.length>=128)throw new Error('A mapping can contain at most 128 fields');const index=state.qrwFields.findIndex(field=>field.id===id);if(index<0)throw new Error('Field mapping is no longer present');const {id:discarded,...source}=state.qrwFields[index],copy=qrwField({...JSON.parse(JSON.stringify(source)),name:`${source.name}_copy`,label:`${source.label} copy`,status:'unresolved',confidence:'unverified',evidence:'Duplicated mapping requires review'});state.qrwFields.splice(index+1,0,copy);el('qrw-field-rows').innerHTML=renderQRWFieldRows();break;}
      case 'qrw-field-delete': {state.qrwFields=state.qrwFields.filter(field=>field.id!==id);el('qrw-field-rows').innerHTML=renderQRWFieldRows()||'<tr><td colspan="7"><div class="empty-state">No mapped fields.</div></td></tr>';break;}
      case 'qrw-field-up':
      case 'qrw-field-down': {const index=state.qrwFields.findIndex(field=>field.id===id),offset=button.dataset.action.endsWith('up')?-1:1,target=index+offset;if(index<0||target<0||target>=state.qrwFields.length)break;[state.qrwFields[index],state.qrwFields[target]]=[state.qrwFields[target],state.qrwFields[index]];el('qrw-field-rows').innerHTML=renderQRWFieldRows();break;}
      case 'qrw-field-preview': {
        const field=state.qrwFields.find(item=>item.id===id);
        if(!field)throw new Error('The selected field is unavailable.');
        await qrwVerifyController();
        nfiHighlight(field);
        break;
      }
      case 'kv-inspect': {
        await inspectKVField(state.qrwFields.find(field=>field.id===id));
        break;
      }
      case 'kv-apply-profile': {
        applyKVGrammar(Number(button.dataset.index));
        break;
      }
      case 'qrw-add-advanced-kv-tail':
      case 'qrw-add-kv-tail': {
        const advanced=button.dataset.action==='qrw-add-advanced-kv-tail';
        const raw=qrwRaw();
        if(!raw.trim())throw new Error('Paste or inspect a raw record before adding a KV tail.');
        const existing=state.qrwFields.find(isKVTail);
        if(existing){
          if(advanced){Object.assign(existing,{kv_profile:'configured',kv_literal_boundaries:true,kv_max_pairs:existing.kv_max_pairs||512,require_complete:true});qrwControllerPreview=null;el('qrw-field-rows').innerHTML=renderQRWFieldRows();nfiScheduleLive();}
          const row=page.querySelector(`[data-qrw-row="${CSS.escape(existing.id)}"]`);
          row?.querySelector('[data-qrw-key=source]')?.focus({preventScroll:true});
          await inspectKVField(existing);
          break;
        }
        if(state.qrwFields.length>=128)throw new Error('A mapping can contain at most 128 fields.');
        const guessed=qrwTokens(raw).findIndex(({value})=>
          /^[|;]*[\p{L}_][\p{L}\p{N}_.-]*[=:]/u.test(value)||
          /^[\[\{(<][\p{L}_][\p{L}\p{N}_.-]*[=:;]/u.test(value));
        const names=new Set(state.qrwFields.map(field=>field.name));
        let name='tail_fields';
        for(let suffix=2;names.has(name);suffix++)name=`tail_fields_${suffix}`;
        const field=qrwField({name,label:'Dynamic key/value tail',method:'kv_eol',end_of_line:true,type:'object',
          policy:advanced?'optional':'required',source:String(guessed>=0?guessed+1:advanced?9:1),source_end:'',unit:'token',output_policy:'flatten',
          kv_profile:advanced?'configured':'auto',assignment_separators:['='],pair_delimiters:advanced?':;|':':;| \t\r\n',require_complete:true,
          ...(advanced?{kv_literal_boundaries:true,kv_max_pairs:512}:{}),
          key_exclude:[...new Set(state.qrwFields.filter(item=>item.method==='key_value').map(item=>item.source))],
          evidence:'Choose a source position and preview the observed pairs.'});
        state.qrwFields.push(field);
        state.qrwStep=4;
        el('qrw-field-rows').innerHTML=renderQRWFieldRows();
        setupQRWStepPanels();
        await inspectKVField(field);
        nfiScheduleLive();
        break;
      }
      case 'qrw-test-fields': {await qrwPrepareFields();qrwTestFields();state.qrwStep=4;setupQRWStepPanels();break;}
      case 'qrw-save-draft': {const payload={version:1,record_id:state.quarantineDiagnosis?.id||'',updated_at:new Date().toISOString(),fields:state.qrwFields};sessionStorage.setItem('nli.qrw.mapping-draft',JSON.stringify(payload));const status=el('qrw-field-status');if(status)status.textContent=`Mapping draft saved in this authenticated browser session at ${time(payload.updated_at)}. Saving a parser candidate below persists the governed server-side revision.`;break;}
      case 'qrw-navigate': {const step=Math.max(1,Math.min(9,Number(button.dataset.step||1)));if(step===5&&state.qrwFields.length){await qrwPrepareFields();const checked=qrwTestFields();if(checked.errors.length)throw new Error('Resolve field mapping errors before generating a parser candidate');}state.qrwStep=step;setupQRWStepPanels();const target=page.querySelector(`[data-qrw-panel~="${step}"]`);target?.scrollIntoView({behavior:'smooth',block:'start'});break;}
      case 'gprs-use-manual': {
        const raw=el('gprs-manual-raw').value,service=el('gprs-manual-service').value.trim()||'auto';
        if(!raw.trim()||new TextEncoder().encode(raw).length>1048576)throw new Error('Paste one non-empty raw record up to 1 MiB');
        if(!/^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(service))throw new Error('Enter auto or a service name of up to 64 letters, digits, dots, underscores or hyphens');
        if(state.gprsManual?.raw!==raw||state.gprsManual?.service!==service){
          resetGPRSSource();
          state.gprsManual={raw,service,parserId:'',nodeId:'',verification:null};
          state.nfiService=service;
          state.lastAssurance=null;
        }
        if(el('gprs-retain-sample')?.checked)await retainTestSample({raw,service,source:'manual',preserve_source:true},'Guided Repair');
        state.qrwStep=2;
        await navigate(state.page);
        if(state.page==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();
        if(button.hasAttribute('data-map-sample'))await mapGuidedSample();
        else page.querySelector('#gprs-preview-form')?.scrollIntoView({block:'start'});
        break;
      }
      case 'test-sample-open': {
        const result=await api('/assurance/cases/'+encodeURIComponent(id));
        const fixture=result.fixture;
        if(!fixture?.raw)throw new Error('The controller did not return retained evidence.');
        resetGPRSSource();
        state.gprsManual={raw:fixture.raw,service:fixture.service||'auto',parserId:'',nodeId:'',verification:null};
        state.nfiService=state.gprsManual.service;
        state.qrwStep=2;
        await navigate('assurance');
        page.querySelector('[data-assurance-tab=gprs]')?.click();
        break;
      }
      case 'gprs-verify-manual': {
        const manual=state.gprsManual,candidate=gprsLinkedCandidate(),bundleHash=state.assuranceSyncBundle?.desired_hash;
        const asset=state.assuranceSyncAssets.find(item=>item.asset_id===candidate?.id);
        if(!manual||candidate?.state!=='published'||!qrwExactAcknowledgement(asset,manual.nodeId))throw new Error('Refresh delivery and confirm the selected client has applied the published bundle first');
        const sample=candidate.payload?.cases?.find(item=>item.raw===manual.raw&&item.service===manual.service);
        if(!sample||!Object.keys(sample.desired||{}).length)throw new Error('The published revision does not contain an exact expected output for this sample');
        const nodeId=manual.nodeId;
        const response=await api('/assurance/run',{method:'POST',body:{raw:sample.raw,service:sample.service,source:'manual',desired:sample.desired,preserve_source:true}});
        if(state.gprsManual!==manual||manual.nodeId!==nodeId)return;
        const expectedParser=candidate.runtime_sync?.parser_id||candidate.id;
        const parserMatches=response.parser?.name===expectedParser;
        const exact=parserMatches?response:await api('/assurance/run',{method:'POST',body:{raw:sample.raw,service:sample.service,source:'manual',desired:sample.desired,preserve_source:true,parser_id:expectedParser}});
        if(state.gprsManual!==manual||manual.nodeId!==nodeId)return;
        const exactPassed=exact.passed===true&&exact.parser?.name===expectedParser;
        manual.verification={
          passed:response.passed===true&&exactPassed,
          bundleHash,
          revision:candidate.revision,
          result:{expected_parser:expectedParser,selected_expected_parser:parserMatches,output_matches:response.passed===true,published_parser_verified:exactPassed,published_parser_errors:exact.errors,actual:response.actual,diff:response.diff,errors:response.errors,parser:response.parser}
        };
        state.qrwStep=9;
        await navigate(state.page);
        if(state.page==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();
        if(manual.verification.passed&&qrwExactAcknowledgement(asset,nodeId))toast('Step 9 complete: published parser, automatic output and client acknowledgement verified.');
        break;
      }
      case 'gprs-preview': {if(state.qrwFields.length)await qrwPrepareFields();const form=page.querySelector('#gprs-preview-form');if(!form)throw new Error('Guided Parser Repair Studio form is unavailable');const checkedRecords=[...page.querySelectorAll('[data-gprs-record]:checked')].map(input=>input.dataset.gprsRecord),diagnosedId=state.quarantineDiagnosis?.id||'',diagnosedRecord=state.repairRecords.find(item=>(item.id||item.record_id)===diagnosedId),selectedRecords=checkedRecords.length?checkedRecords:(quarantineCanInspect(diagnosedRecord)?[diagnosedId]:[]),selectedFixtures=[...form.elements.fixtures.selectedOptions].map(option=>option.value);if(!state.gprsManual&&!selectedRecords.length&&!selectedFixtures.length)throw new Error(diagnosedId?'The selected record is no longer actionable or its raw evidence is unavailable. Return to Step 1 and select an active retained failure.':'Select and inspect a retained failure, or choose a saved fixture, before testing');const checked=state.qrwFields.length?qrwTestFields():{errors:[]};if(checked.errors.length)throw new Error('Resolve every field mapping error before running the deterministic preview');const desired=parseJSON(form.elements.gprs_desired.value,'Desired JSON output instance'),mapping=parseJSON(form.elements.mapping.value,'Field mappings');if(!desired||Array.isArray(desired)||typeof desired!=='object')throw new Error('Desired JSON must be an object');if(!mapping||Array.isArray(mapping)||typeof mapping!=='object'||Object.values(mapping).some(value=>typeof value!=='string'))throw new Error('Field mappings must be a JSON object with string targets');const cases=state.gprsManual?[{raw:state.gprsManual.raw,service:state.gprsManual.service,desired,field_mapping:mapping,source:'manual'}]:[];state.gprsSamples=[...cases];for(const recordId of state.gprsManual?[]:selectedRecords){const record=state.repairRecords.find(item=>(item.id||item.record_id)===recordId);if(!record?.raw_available)throw new Error('Exact raw evidence is not retained for '+recordId);const result=await api('/quarantine/records/'+encodeURIComponent(recordId)+'/analyze',{method:'POST',body:{service:record.service||'auto',desired,field_mapping:mapping,source:'quarantine:'+recordId}});const raw=result.result?.raw;if(typeof raw!=='string'||!raw)throw new Error('Controller did not return retained raw evidence for '+recordId);form.elements.raw_evidence.value=raw;state.gprsSamples.push({raw,service:record.service||'auto',desired,field_mapping:mapping,source:'quarantine:'+recordId});cases.push(state.gprsSamples.at(-1));state.quarantineDiagnosis={id:recordId,sha256:result.raw_sha256,record:result.record};}for(const fixtureId of selectedFixtures){const response=await fetchWithTimeout('/api/v1/assurance/cases/'+encodeURIComponent(fixtureId)+'/download',{credentials:'same-origin',headers:{Accept:'text/plain, application/octet-stream'}},55000);if(!response.ok)throw new Error('Could not retrieve retained fixture '+fixtureId+' ('+response.status+')');const raw=await response.text();if(!raw)throw new Error('Retained fixture '+fixtureId+' is empty');const fixture=state.assuranceFixtures.find(item=>(item.id||item.payload?.id)===fixtureId),payload=fixture?.payload||fixture||{},request=payload.case||payload.request||{};state.gprsSamples.push({raw,service:request.service||payload.service||'auto',desired,field_mapping:mapping,source:'fixture:'+fixtureId});cases.push(state.gprsSamples.at(-1));}const progress=notify('Running selected records through the production Assurance pipeline…','progress');try{const parserCandidate=state.qrwFields.length?qrwBuildParser():null;const preview=parserCandidate?await api('/assurance/parser-preview',{method:'POST',body:{parser:parserCandidate,cases}}):cases.length===1?await api('/assurance/run',{method:'POST',body:{...cases[0],preserve_source:true}}):await api('/assurance/corpus',{method:'POST',body:{cases:cases.map(item=>({...item,preserve_source:true}))}});state.gprsPreview={records_tested:cases.length,parser_candidate:!!parserCandidate,artifact_checksum:preview.artifact_checksum,preview};state.lastAssurance={request:cases[0],result:preview};state.qrwStep=6;const output=el('gprs-preview');if(output)output.textContent=pretty(state.gprsPreview);const aiFallback=el('gprs-ai-fallback');if(aiFallback)aiFallback.innerHTML=cases.length===1&&preview.passed===false?'<b>Deterministic recovery preview failed.</b> AI is now available as the final fallback. It creates a review-only proposal and cannot change a live parser.<div class="actions"><button class="primary" type="button" data-action="request-ai-proposal">Ask Nexora AI to diagnose and propose a fix</button><button type="button" class="secondary" data-action="workspace-ai-auto" data-tab="Assurance diagnosis">Auto Fix with Nexora AI</button></div>':'Deterministic preview completed. Continue with the embedded candidate editor; AI remains reserved for a failed single-record preview.';setupQRWStepPanels();progress.done('Live parser preview returned by the controller. No candidate was activated.');}catch(error){progress.fail(error.message);throw error;}break; }
      case 'gprs-open-editor': {if(state.qrwFields.length)await qrwPrepareFields();const helper=page.querySelector('#gprs-parser-helper-form'),wizard=page.querySelector('#gprs-preview-form');if(!helper||!wizard)throw new Error('Parser candidate editor is unavailable');const raw=wizard.elements.raw_evidence.value||state.lastAssurance?.request?.raw||'';if(!raw)throw new Error('Inspect retained raw evidence before preparing a candidate');const manualName=wizard.elements.manual_field_name.value.trim(),index=Number(wizard.elements.manual_field_index.value||1),delimiter=wizard.elements.manual_delimiter.value||'=',template=wizard.elements.manual_template.value,regex=wizard.elements.manual_regex.value,safeName=(manualName||'field').replace(/[^A-Za-z0-9_]/g,'_'),escapedDelimiter=delimiter.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'),parser=state.qrwFields.length?qrwBuildParser():{pattern_type:'regex',pattern:regex||`(?:^|\\s)(?P<${safeName}>[^\\s${escapedDelimiter}]+)`,fields:manualName?[{name:manualName,source:regex?'group:'+safeName:`token:${index}`,type:'string'}]:[],delimiter,template};const desired=parseJSON(wizard.elements.gprs_desired.value,'Desired JSON output instance'),fieldMapping=parseJSON(wizard.elements.mapping.value,'Field mappings'),originId=state.quarantineDiagnosis?.id||'selected',service=state.gprsManual?.service||state.repairRecords.find(item=>(item.id||item.record_id)===originId)?.service||'auto',source=state.gprsManual?'manual':`quarantine:${originId}`;state.gprsSamples=state.gprsSamples.length?state.gprsSamples.map(item=>({...item,desired,field_mapping:fieldMapping})): [{raw,service,desired,field_mapping:fieldMapping,source}];helper.elements.samples.value=state.gprsSamples.map(item=>item.raw).join('\n');helper.elements.cases.value=pretty(state.gprsSamples);helper.elements.edit_id.value='';helper.elements.parser.value=pretty(parser);state.qrwStep=5;setupQRWStepPanels();helper.scrollIntoView({behavior:'smooth',block:'start'});helper.elements.samples.focus();if(state.page==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();break;}
      case 'quarantine-acquire-evidence': {toast(quarantineEvidenceHelp(button.dataset.node||''));break;}
      case 'quarantine-diagnose': {const target=state.page==='assurance'&&button.closest('[data-assurance-panel=gprs]')?'assurance':'quarantine-wizard',analysis=await api(`/quarantine/records/${encodeURIComponent(id)}/analyze`,{method:'POST',body:{service:button.dataset.service||'auto'}});const result=analysis.result||{};if(state.gprsManual||state.quarantineDiagnosis?.id!==id)resetGPRSSource();state.lastAssurance={request:{raw:result.raw||'',service:button.dataset.service||'auto',parser_id:'',source:`quarantine:${id}`,desired:{},field_mapping:{}},result};state.quarantineDiagnosis={id,sha256:analysis.raw_sha256,record:analysis.record};sessionStorage.setItem('nli.qrw.selected-record',id);state.qrwStep=2;await navigate(target);if(target==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();const rawField=page.querySelector('#gprs-preview-form [name=raw_evidence]');if(rawField)rawField.value=result.raw||'';const checkbox=page.querySelector(`[data-gprs-record="${CSS.escape(id)}"]`);if(checkbox)checkbox.checked=true;page.querySelectorAll('[data-wizard-step]').forEach(item=>item.classList.toggle('done',Number(item.dataset.wizardStep)<2));setupQRWStepPanels();page.querySelector('#gprs-preview-form')?.scrollIntoView({behavior:'smooth',block:'center'});toast(`Evidence ${String(analysis.raw_sha256||'').slice(0,12)} analyzed · opened in the Resolution Wizard; no parser state changed`);if(button.hasAttribute('data-map-sample'))await mapGuidedSample();break;}
      case 'quarantine-exclude': {const inGprs=!!button.closest('[data-assurance-panel="gprs"]'),target=inGprs?(state.page==='quarantine-wizard'?'quarantine-wizard':'assurance'):'unknowns',reason=prompt('Why should this record be excluded from the active quarantine queue?');if(reason===null)break;if(!reason.trim())throw new Error('An exclusion reason is required');await api(`/quarantine/records/${encodeURIComponent(id)}/exclude`,{method:'POST',body:{reason:reason.trim()}});if(state.quarantineDiagnosis?.id===id){state.quarantineDiagnosis=null;state.gprsPreview=null;state.gprsSamples=[];state.qrwFields=[];state.qrwStep=1;sessionStorage.removeItem('nli.qrw.selected-record');sessionStorage.removeItem('nli.qrw.mapping-draft');}await refreshNavigationBadges();toast('Record excluded from active work; retained evidence and audit history remain available');await navigate(target);if(inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();break;}
      case 'quarantine-include': {const inGprs=!!button.closest('[data-assurance-panel="gprs"]'),target=inGprs?(state.page==='quarantine-wizard'?'quarantine-wizard':'assurance'):'unknowns';await api(`/quarantine/records/${encodeURIComponent(id)}/include`,{method:'POST',body:{}});toast('Record returned to the quarantine queue');await navigate(target);if(inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();break;}
      case 'quarantine-delete-raw': {const inGprs=!!button.closest('[data-assurance-panel="gprs"]'),target=inGprs?(state.page==='quarantine-wizard'?'quarantine-wizard':'assurance'):'unknowns';if(confirm(`Permanently delete retained raw evidence for ${id}? The metadata and audit history will remain.`)){await api(`/quarantine/records/${encodeURIComponent(id)}/raw`,{method:'DELETE',body:{}});if(state.quarantineDiagnosis?.id===id){state.quarantineDiagnosis=null;state.gprsPreview=null;state.gprsSamples=[];state.qrwFields=[];state.qrwStep=1;sessionStorage.removeItem('nli.qrw.selected-record');sessionStorage.removeItem('nli.qrw.mapping-draft');}await refreshNavigationBadges();toast('Retained raw evidence permanently deleted; metadata no longer appears as actionable recovery work');await navigate(target);if(inGprs)page.querySelector('[data-assurance-tab=gprs]')?.click();}break;}
      case 'quarantine-queue-replay': {const form=page.querySelector('#quarantine-reprocess-form');if(!form)throw new Error('Replay form is unavailable');const ids=form.elements.record_ids.value.split(',').map(value=>value.trim()).filter(Boolean);if(!ids.includes(id))form.elements.record_ids.value=[...ids,id].join(', ');form.elements.node_id.value=button.dataset.node||'';form.scrollIntoView({behavior:'smooth',block:'center'});toast('Record added to the targeted replay request');break;}
      case 'quarantine-raw': { const response=await fetchWithTimeout(`/api/v1/quarantine/records/${encodeURIComponent(id)}/raw`,{credentials:'same-origin',headers:{Accept:'text/plain, application/octet-stream'}},55000);if(!response.ok){const result=await response.json().catch(()=>({}));throw new Error(result.error||`Raw record download failed (${response.status})`);}downloadBlob(`nli-quarantine-${id}.log`,await response.blob());toast('Exact retained source line downloaded');break; }
      case 'quarantine-export': { const format=button.dataset.format==='jsonl'?'jsonl':'raw';const response=await fetchWithTimeout(`/api/v1/quarantine/export?format=${format}`,{credentials:'same-origin',headers:{Accept:format==='jsonl'?'application/x-ndjson, application/json':'text/plain, application/octet-stream'}},55000);if(!response.ok){const result=await response.json().catch(()=>({}));throw new Error(result.error||`Quarantine export failed (${response.status})`);}downloadBlob(format==='jsonl'?'nli-quarantine.jsonl':'nli-quarantine-raw.log',await response.blob());toast('Quarantine export downloaded');break; }
      case 'promote-case': {const name=prompt('Draft parser pack name');if(name){await api(`/assurance/cases/${encodeURIComponent(id)}/promote`,{method:'POST',body:{name,parser:{}}});toast('Draft parser pack created');await navigate('parsers');}break;}
      case 'user-delete': if(confirm(`Deactivate ${id}?`)){await api(`/users/${encodeURIComponent(id)}`,{method:'DELETE',body:{}});toast('User deactivated');await navigate('security');}break;
    }
  }

  function downloadJSON(name, value) {
    downloadBlob(name,new Blob([pretty(value)+'\n'],{type:'application/json'}));
  }
  function downloadBlob(name, blob) {
    const url=URL.createObjectURL(blob);
    const link=document.createElement('a');link.href=url;link.download=name;document.body.append(link);link.click();link.remove();
    setTimeout(()=>URL.revokeObjectURL(url),1000);
  }

  async function connectStream() {
    if(!state.user)return;
    startSyncFallback();
    const protocol=location.protocol==='https:'?'wss':'ws';
    if(!state.socket||state.socket.readyState>1){const socket=new WebSocket(`${protocol}://${location.host}/api/v1/stream`);state.socket=socket;socket.onmessage=event=>{try{const data=JSON.parse(event.data);if(data.type==='ai_provider_changed'){notify(`AI provider status: ${data.status||'updated'}`);if(data.napi||data.subsystem)applyServerSubsystemUpdate('napi',data.napi||data.subsystem);if(state.page==='ai'&&page.dataset.dirty!=='true')navigate('ai');return;}if(data.type==='ai_proposal_changed'){notify(`AI remediation ${data.id||''}: ${data.status||'updated'}`);if(data.napi||data.subsystem)applyServerSubsystemUpdate('napi',data.napi||data.subsystem);if(state.page==='assurance')navigate('assurance');return;}if(data.type==='assurance_run_changed'){if(data.nal||data.subsystem)applyServerSubsystemUpdate('nal',data.nal||data.subsystem);if(state.page==='assurance')navigate('assurance');return;}state.live.unshift(data);state.live=state.live.slice(0,200);if(state.page==='metadata'){const target=el('live-records');if(target)target.innerHTML=renderMetadataRows(state.live);}}catch{}};}
    if(!state.sync.snapshot){try{acceptSyncSnapshot(await fetchCanonicalSnapshot());}catch(error){state.streamMode='snapshot unavailable';notify(`Canonical synchronization snapshot failed: ${error.message}`,'error');}}
    if(typeof EventSource!=='function'){state.streamMode='polling';el('stream-state').textContent='Live sync · recovery';await pollCanonicalSnapshot('streaming unsupported');return;}
    if(state.streamSource)return;
    const url=`/api/v1/sync/stream?last_revision=${encodeURIComponent(state.sync.snapshot?.revision||0)}`,source=new EventSource(url,{withCredentials:true});state.streamSource=source;state.sync.connectionStartedAt=Date.now();
    const receive=(event,eventType)=>{try{const data=JSON.parse(event.data);if(event.lastEventId)state.streamLastEventId=event.lastEventId;acceptSyncSnapshot(data,{incremental:eventType!=='sync_snapshot'&&eventType!=='sync_heartbeat',eventType});}catch(error){state.sync.rejectedUpdates++;notify(`Synchronization update rejected: ${error.message}`,'error');}};
    source.addEventListener('sync_snapshot',event=>receive(event,'sync_snapshot'));
    source.addEventListener('sync_heartbeat',event=>receive(event,'sync_heartbeat'));
    source.addEventListener('sync_delta',event=>receive(event,'sync_delta'));
    source.addEventListener('sync_critical',event=>receive(event,'sync_critical'));
    source.onopen=()=>{state.streamFailures=0;state.streamMode='live';el('stream-state').innerHTML='<i></i>Live sync';renderSyncSubscriber({force:true});};
    source.onerror=()=>{if(!state.user)return;state.streamMode='reconnecting';el('stream-state').innerHTML='<i></i>Reconnecting · recovery active';state.streamFailures++;source.close();state.streamSource=null;pollCanonicalSnapshot('event stream interrupted');if(state.streamReconnectTimer)clearTimeout(state.streamReconnectTimer);const delay=Math.min(1000*(2**Math.min(state.streamFailures-1,5)),30000);state.streamReconnectTimer=setTimeout(()=>{state.streamReconnectTimer=null;if(state.user)connectStream();},delay);renderSyncSubscriber({force:true});};
  }

  async function signOut(notify=true){state.sampleHistoryEnabled=false;inFlightReads.clear();state.navigationHistory=[];if(state.socket){state.socket.onclose=null;state.socket.close();state.socket=null;}if(state.streamSource){state.streamSource.close();state.streamSource=null;}if(state.streamPollTimer){clearInterval(state.streamPollTimer);state.streamPollTimer=null;}if(state.streamReconnectTimer){clearTimeout(state.streamReconnectTimer);state.streamReconnectTimer=null;}state.sync.pollController?.abort();state.user=null;if(notify){try{await api('/auth/logout',{method:'POST',body:{}});}catch{}}state.csrf='';state.lastAssurance=null;state.quarantineDiagnosis=null;state.gprsManual=null;state.blockBoard=null;state.blockSeed='';state.qrwReplayAcceptedIds=[];state.gprsSamples=[];state.gprsPreview=null;state.qrwFields=[];state.nfiRaw='';state.nfiCorpus='';state.nfiService='auto';state.nfiScopePattern='';state.nfiGlobalScope=false;state.nfiEditId='';state.nfiName='';state.nfiVersion='';state.nfiParser=null;state.nfiCases=[];state.nfiProfiles=[];state.nfiDesired=null;state.aiSelectedProviderId='';state.aiProviderTemplates=[];state.nfiMappingPriority=0;state.nfiSenseMode='';state.nfiExpert=false;state.nfiGuideStep=undefined;state.nfiCandidateValidated=null;state.nfiPreviewed=false;clearTimeout(nfiLiveTimer);nfiAnalysisController?.abort();qrwControllerPreview=null;state.controllerNotifications=[];state.notifications=[];renderNotificationBell();sessionStorage.removeItem('nli.qrw.mapping-draft');sessionStorage.removeItem('nli.qrw.selected-record');state.live=[];state.polling=false;state.sync={snapshot:null,history:[],browserAppliedAt:0,browserDelayMs:null,lastReceivedAt:0,pollInFlight:false,pollController:null,pollFailures:0,rejectedUpdates:0,connectionStartedAt:0,resyncing:false,renderedSignature:null,renderScheduled:false};el('login-form').querySelector('[name=password]').value='';el('app-view').hidden=true;el('auth-view').hidden=false;}
  let browserHeartbeatRetryAt = 0;
  let browserHeartbeatDelay = 5000;
  async function sendBrowserHeartbeat(){
    if(!state.user||state.browserHeartbeatInFlight||document.hidden||Date.now()<browserHeartbeatRetryAt)return;
    state.browserHeartbeatInFlight=true;
    try{
      await api('/communications/browser-heartbeat',{method:'POST',body:{revision:Number(state.sync.snapshot?.revision||0)}});
      state.lastBrowserHeartbeatAt=Date.now();
      browserHeartbeatDelay=5000;
      browserHeartbeatRetryAt=0;
    }catch(error){
      if(error.status===403 && /csrf/i.test(error.message))state.csrf='';
      browserHeartbeatRetryAt=Date.now()+browserHeartbeatDelay;
      browserHeartbeatDelay=Math.min(60000,browserHeartbeatDelay*2);
    }finally{
      state.browserHeartbeatInFlight=false;
    }
  }
  async function boot(user){state.user=user;buildNav();el('identity-name').textContent=user.username;el('identity-role').textContent=user.role;el('auth-view').hidden=true;el('app-view').hidden=false;await ensureCSRF();refreshApprovalBadge();await refreshNotificationBell();sendBrowserHeartbeat();await connectStream();await navigate(location.hash.slice(1)||'operations');}

  el('login-form').addEventListener('submit',async event=>{event.preventDefault();if(!state.controllerReady){showPreviewOnly(CONTROLLER_REQUIRED);return;}const data=formObject(event.currentTarget);el('login-error').textContent='';try{await api('/auth/login',{method:'POST',body:data});const principal=await api('/auth/me');await boot(principal);}catch(error){el('login-error').textContent=error.message;}});
  el('retry-controller').addEventListener('click',checkController);
  page.addEventListener('input',event=>{page.dataset.dirty='true';if(event.target.matches('#input-profile-form [name=config]')){try{const config=parseJSON(event.target.value,'Input definitions JSON');el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config),'No paths configured.');}catch{}}});
  page.addEventListener('focusout',event=>{if(!event.target.matches('select'))return;setTimeout(()=>{if(state.sync.pendingRender&&!(page.contains(document.activeElement)&&document.activeElement.matches?.('select'))){state.sync.pendingRender=false;renderSyncSubscriber({force:true});}},0);});
  page.addEventListener('change',event=>{
    if(event.target.matches('#ai-provider-form [name=endpoint]')){
      el('ai-model-options').innerHTML=aiModelOptions(aiPresetForEndpoint(event.target.value));
    }
    if(event.target.id==='ai-provider-preset'){
      const template=state.aiProviderTemplates?.find(item=>item.id===event.target.value);
      el('ai-preset-account-field').hidden=!template?.base_url.includes('{account_id}');
    }
  });
  page.addEventListener('change',event=>{
    const target=event.target;
    if(target.id==='gprs-manual-node'&&state.gprsManual){
      state.gprsManual.nodeId=target.value;
      state.gprsManual.verification=null;
      state.qrwStep=8;
      const targetPage=state.page;
      navigate(targetPage).then(()=>{if(targetPage==='assurance')page.querySelector('[data-assurance-tab=gprs]')?.click();}).catch(error=>toast(error.message,true));
      return;
    }
    if(target.matches('#gprs-replay-form [name=scope]')){const form=target.form,recordsSelect=form.elements.record_ids,originId=state.quarantineDiagnosis?.id||'',origin=state.repairRecords.find(item=>(item.id||item.record_id)===originId),originNode=origin?.node_id||'';for(const option of recordsSelect.options){const sameOrigin=option.dataset.node===originNode,allowed=option.dataset.ack==='true';option.hidden=target.value==='origin'?option.value!==originId:target.value==='all'?!sameOrigin:false;option.disabled=!allowed||(target.value==='origin'&&option.value!==originId)||(target.value==='all'&&!sameOrigin);option.selected=target.value==='origin'?option.value===originId&&allowed:target.value==='all'&&sameOrigin&&allowed?true:target.value==='selected'?option.selected:false;}const selected=[...recordsSelect.selectedOptions],nodes=[...new Set(selected.map(option=>option.dataset.node).filter(Boolean))];form.elements.node_id.value=nodes.length===1?nodes[0]:'';const submit=form.querySelector('#gprs-replay-submit');if(submit)submit.disabled=!selected.length||nodes.length!==1;return;}
    if(target.matches('#gprs-replay-form [name=record_ids]')){const form=target.form,selected=[...target.selectedOptions],nodes=[...new Set(selected.map(option=>option.dataset.node).filter(Boolean))];if(nodes.length>1){const keep=nodes[0];selected.filter(option=>option.dataset.node!==keep).forEach(option=>option.selected=false);toast('A replay request can target one client. Records from other clients were deselected.',true);}const remaining=[...target.selectedOptions],nodeField=form.elements.node_id;nodeField.value=remaining[0]?.dataset.node||'';const submit=form.querySelector('#gprs-replay-submit');if(submit)submit.disabled=!remaining.length||new Set(remaining.map(option=>option.dataset.node)).size!==1;return;}
    if(target.matches('[data-input-family],[data-input-path]')){
      const form=target.closest('form'),config=parseJSON(form.elements.config.value,'Input definitions JSON'),index=Number(target.dataset.inputFamily??target.dataset.inputPath);
      if(config.inputs?.[index]){
        if(target.matches('[data-input-family]')){config.inputs[index].service=target.value;config.inputs[index].multiline=multilineForFamily(target.value);}
        else {const path=target.value.trim();if(!path.startsWith('/')||config.inputs.some((item,i)=>i!==index&&item.path===path)){toast('Enter a unique absolute Linux path',true);el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config));return;}config.inputs[index].path=path;}
        form.elements.config.value=pretty(config);el('input-path-list').innerHTML=table(['Log path','Family','Status and actions'],inputPathRows(config));
      }
      page.dataset.dirty='true';return;
    }
    if(target.matches('[data-monitoring-filter]')){const kind=target.dataset.monitoringFilter;if(kind==='range')state.monitoringRange=target.value;if(kind==='node')state.monitoringNode=target.value;if(kind==='source')state.monitoringSource=target.value;if(kind==='search')state.monitoringSearch=target.value;page.dataset.dirty='false';renderMonitoring();return;}
    if(target.id==='dashboard-range'||target.id==='dashboard-client'){if(target.id==='dashboard-range')state.dashboardRange=target.value;else state.dashboardClient=target.value;page.dataset.dirty='false';if(state.dashboardRange==='custom'){renderLiveOperationsDashboard(state.dashboard||syncDashboardSnapshot());}else applyDashboardHistory({force:true});return;}
    if(target.id==='dashboard-from'||target.id==='dashboard-to'){state[target.id==='dashboard-from'?'dashboardFrom':'dashboardTo']=target.value;page.dataset.dirty='true';return;}
    if(target.id==='input-history-profile'){state.inputHistoryProfile=target.value;page.dataset.dirty='false';navigate('input-paths');return;}
    page.dataset.dirty='true';
    if(target.id==='input-select-all')page.querySelectorAll('[name=client_target]').forEach(input=>input.checked=target.checked);
    if(target.form?.id==='managed-input-file-form'&&target.name==='action'){
      const destination=target.form.elements.destination;
      destination.disabled=target.value==='delete';
      destination.required=!destination.disabled;
      destination.closest('[data-file-destination]').hidden=destination.disabled;
    }
    if(target.name==='all_eligible'){const form=target.form,targets=form.elements.targets,offline=form.elements.include_offline;targets.disabled=target.checked;offline.disabled=!target.checked;if(!target.checked)offline.checked=false;if(target.checked)targets.value='';}
    if(target.name==='client_target'||target.id==='input-select-all'){const selected=[...page.querySelectorAll('[name=client_target]:checked')].map(input=>input.value),targets=page.querySelector('#input-rollout-form [name=targets]'),all=page.querySelector('#input-select-all'),allEligible=page.querySelector('#input-rollout-form [name=all_eligible]');if(allEligible?.checked){allEligible.checked=false;allEligible.dispatchEvent(new Event('change',{bubbles:true}));}if(targets){targets.disabled=false;targets.value=selected.join(', ');}if(all){const boxes=[...page.querySelectorAll('[name=client_target]')];all.checked=boxes.length>0&&boxes.every(input=>input.checked);all.indeterminate=boxes.some(input=>input.checked)&&!all.checked;}}
  });
  el('nav').addEventListener('click',event=>{const button=event.target.closest('[data-page]');if(button){if(state.page===button.dataset.page||location.hash.slice(1)===button.dataset.page)navigate(button.dataset.page);else location.hash=button.dataset.page;}});
  page.addEventListener('submit',async event=>{event.preventDefault();const button=event.submitter,inlineReplay=['gprs-replay-form','quarantine-reprocess-form'].includes(event.target.id);if(button)button.disabled=true;const progress=inlineReplay?null:notify('Processing request…','progress');try{await submit(event.target);}catch(error){if(inlineReplay)setReplayStatus(event.target,replayGateMessage(error)||`Replay request was not accepted: ${error.message}. Live telemetry remains active.`,'warn');else{progress.fail(error.message);toast(error.message,true);}}finally{if(button)button.disabled=false;}});
  page.addEventListener('input',()=>{for(const host of page.querySelectorAll('[data-ai-mapping-host]')){if(host.aiProposal&&mappingAIRaw(host)!==host.aiProposal.raw){host.querySelector('[data-ai-mapping-proposal]').hidden=true;host.querySelector('[data-ai-mapping-status]').textContent='Raw evidence changed; request a fresh AI proposal before import.';}}});
  page.addEventListener('reset',event=>{
    const form=event.target;
    if(!form.matches('[data-workspace-ai-form]'))return;
    form.aiRequestID=(form.aiRequestID||0)+1;
    delete form.aiResult;
    form.querySelector('[data-workspace-ai-result]').textContent='';
    for(const button of form.querySelectorAll('[data-action=workspace-ai-import],[data-action=workspace-ai-download]'))button.hidden=true;
  });
  const AI_REQUEST_ACTIONS=new Set(['workspace-ai-run','workspace-ai-auto','mapping-ai-suggest','mapping-ai-readiness','nfi-ai-status','request-ai-proposal','test-ai-provider']);
  page.addEventListener('click',async event=>{const button=event.target.closest('[data-action]');if(!button)return;
    const aiPending=button.hasAttribute('data-map-sample')||AI_REQUEST_ACTIONS.has(button.dataset.action);
    if(aiPending){button.dataset.aiPending='true';button.setAttribute('aria-busy','true');}
    button.disabled=true;
    try{await action(button);}catch(error){toast(error.message,true);}finally{
    button.disabled=false;
    if(aiPending){delete button.dataset.aiPending;button.removeAttribute('aria-busy');}
    if(['qrw-token-field','qrw-selection-field','qrw-add-field','qrw-field-duplicate','qrw-field-delete','qrw-field-up','qrw-field-down','qrw-operation-add','qrw-operation-delete','qrw-operation-up','qrw-suggest-fields','qrw-import-plan'].includes(button.dataset.action))nfiScheduleLive();queueMicrotask(nfiDecorate);}});
  const updateChartReadout=event=>{const point=event.target.closest?.('[data-chart-category],[data-chart-summary]');if(!point)return;const chart=point.closest('.multi-chart,.stacked-volume'),target=chart?.querySelector('.chart-hover-readout');if(!target)return;target.textContent=point.dataset.chartSummary||`${point.dataset.chartCategory} · ${countText(point.dataset.chartValue)} ${point.dataset.chartUnit} · ${point.dataset.chartTime}`;};
  page.addEventListener('focusin',event=>{const row=event.target.closest('[data-qrw-row]');if(row){const field=state.qrwFields.find(f=>f.id===row.dataset.qrwRow);if(field)nfiHighlight(field);}});
  page.addEventListener('pointerover',updateChartReadout);page.addEventListener('focusin',updateChartReadout);
  page.addEventListener('input',event=>{
    if(event.target.closest('#gprs-parser-helper-form')&&['parser','cases'].includes(event.target.name)){state.gprsPreview=null;const status=event.target.closest('form').querySelector('[data-parser-helper-status]');if(status)status.textContent='Candidate JSON changed. Run complete parser validation again before review.';}
    if(event.target.id==='nfi-raw'||event.target.matches('#gprs-preview-form [name=raw_evidence]')){if(el('nfi-runtime-evidence'))el('nfi-runtime-evidence').hidden=true;nfiScheduleLive();if(event.target.id==='nfi-raw')state.nfiRaw=event.target.value;qrwControllerPreview=null;const host=el('nfi-inspector-host');if(host&&new TextEncoder().encode(event.target.value).length<=1048576)host.innerHTML=qrwRawInspector();const status=el('qrw-field-status');if(status)status.textContent='Raw sample changed. The preview will use the current sample.';return;}
    if(['nfi-service','nfi-scope-pattern','nfi-global-scope','nfi-mapping-priority'].includes(event.target.id)){state.nfiMappingPriority=Number(el('nfi-mapping-priority')?.value)||0;state.nfiService=el('nfi-service')?.value||'auto';state.nfiScopePattern=el('nfi-scope-pattern')?.value||'';state.nfiGlobalScope=el('nfi-global-scope')?.checked===true;nfiScheduleLive();return;}
    if(['nfi-parser-json','nfi-cases-json'].includes(event.target.id)){state.nfiCandidateValidated=null;page.dataset.dirty='true';const status=el('nfi-candidate-status');if(status)status.textContent='Expert JSON edited; exact parser validation is pending. Import a changed mapping plan to update the visual editor.';nfiGuideRefresh();return;}
    if(event.target.id==='nfi-corpus'){state.nfiCorpus=event.target.value;return;}
    const kvInput=event.target.closest('[data-kv-input-key]'),opControl=event.target.closest('[data-qrw-op-key]'),control=event.target.closest('[data-qrw-key]'),row=(kvInput||opControl||control)?.closest('[data-qrw-row]');if(!row)return;const field=state.qrwFields.find(item=>item.id===row.dataset.qrwRow);if(!field)return;
    if(kvInput){
      const edit=field.kv_input_edits?.[Number(kvInput.closest('[data-kv-input-index]').dataset.kvInputIndex)];
      if(!edit)return;
      edit[kvInput.dataset.kvInputKey]=kvInput.type==='checkbox'?kvInput.checked:kvInput.value;
      field.status='unresolved';
      qrwControllerPreview=null;
      if(kvInput.dataset.kvInputKey==='action'){
        if(edit.action!=='replace')delete edit.replace;
        el('qrw-field-rows').innerHTML=renderQRWFieldRows();
        nfiDecorate();
      }
      nfiScheduleLive();
      return;
    }
    if(opControl){const operation=field.operations?.[Number(opControl.closest('[data-qrw-op]').dataset.qrwOp)];if(!operation)return;const key=opControl.dataset.qrwOpKey;try{operation[key]=opControl.type==='checkbox'?opControl.checked:key==='fields'?opControl.value.split(',').map(value=>value.trim()).filter(Boolean):NFI_NUMBER_KEYS.includes(key)?Number(opControl.value):NFI_JSON_KEYS.includes(key)?JSON.parse(opControl.value||'null'):opControl.value;opControl.setCustomValidity('');}catch{opControl.setCustomValidity('Enter valid JSON for this list or alias map.');return;}}
    else {const key=control.dataset.qrwKey,previousMethod=field.method;try{field[key]=control.type==='checkbox'?control.checked:NFI_NUMBER_KEYS.includes(key)?Number(control.value):NFI_JSON_KEYS.includes(key)?JSON.parse(control.value||'null'):key==='pair_delimiters'?control.value.replaceAll('\\t','\t').replaceAll('\\r','\r').replaceAll('\\n','\n'):control.value;}catch{control.setCustomValidity('Enter valid JSON for this list or alias map.');return;}control.setCustomValidity('');
      if(['assignment_separators','pair_delimiters','kv_pair_separators','kv_quotes','kv_escape','kv_brackets','kv_key_pattern','kv_reject_empty'].includes(key)&&(isKVTail(field)||field.method==='key_value')){
        if(['assignment_separators','kv_pair_separators'].includes(key)&&field[key]!=null&&(!Array.isArray(field[key])||field[key].some(value=>typeof value!=='string'||!value))){
          control.setCustomValidity('Enter a JSON list of nonempty operators, such as [":"]');
          return;
        }
        field.kv_profile='configured';
        const mode=row.querySelector('[data-qrw-key=kv_profile]');
        if(mode)mode.value='configured';
      }
      if(key==='type'&&field.method==='key_value'&&field.type!=='object')delete field.kv_nested_objects;
      if(key==='method'){
        const namedStart=field.kv_start;
        if(field.method!=='key_value')delete field.kv_start;
        if(field.method==='kv_eol'){
          const start=previousMethod==='key_value'?namedStart:field.source;
          const unit=previousMethod==='key_value'?field.unit:previousMethod;
          Object.assign(field,{type:'object',output_policy:'flatten',end_of_line:true,
            unit:['character','byte','line','word','token'].includes(unit)?unit:'token',
            source:/^[1-9][0-9]*$/.test(start)?start:'9',source_end:'',
            kv_profile:field.kv_profile||'auto',require_complete:true});
        }else if(field.method==='kv_all'&&previousMethod==='key_value'){
          Object.assign(field,{type:'object',source:namedStart||'raw'});
        }else if(!isKVTail(field)){
          if(field.output_policy==='flatten')field.output_policy='include';
          if(field.method!=='key_value'){
            for(const name of ['kv_profile','end_of_line','kv_pair_separators','kv_quotes','kv_escape','kv_brackets','kv_key_pattern','kv_reject_empty','kv_nested_objects'])delete field[name];
          }
        }
      }
    }
    if(control?.dataset.qrwKey==='key_value_edits') {
      const edits=field.key_value_edits;
      if(edits!=null&&(!Array.isArray(edits)||edits.some(edit=>!edit||typeof edit.key!=='string'||!['omit','blank','replace'].includes(edit.action)||edit.value!=null&&typeof edit.value!=='string'))){
        control.setCustomValidity('Enter an array of key edit objects with key, action and optional replacement value.');
        control.reportValidity();
        return;
      }
      if(!event.detail?.preserveKVEditor)row.querySelector('[data-kv-edit-rows]').innerHTML=renderKVEditRows(edits);
    }
    qrwControllerPreview=null;field.status=field.method==='delimiter'&&!field.confirmed?'needs_confirmation':'unresolved';field.preview='';field.evidence='Mapping changed; test or verify the current draft';const preview=row.querySelector('[data-qrw-preview]');if(preview)preview.textContent='Unverified change';
    nfiScheduleLive();
    if(['method','type'].includes(control?.dataset.qrwKey)||opControl?.dataset.qrwOpKey==='op'){const open=opControl!=null;el('qrw-field-rows').innerHTML=renderQRWFieldRows();if(open)page.querySelector('[data-qrw-row="'+CSS.escape(field.id)+'"] .nfi-operations')?.setAttribute('open','');nfiDecorate();}
  });
  function updateKVEditRows(row, mutate, {focusNew=false,preserveEditor=false}={}) {
    const field=state.qrwFields.find(item=>item.id===row.dataset.qrwRow);
    const input=row.querySelector('[data-qrw-key="key_value_edits"]');
    if(!field||!input)return;
    if(!input.reportValidity())return;
    const edits=(field.key_value_edits||[]).map(edit=>({...edit}));
    mutate(edits);
    input.value=edits.length?pretty(edits):'';
    input.dispatchEvent(new CustomEvent('input',{bubbles:true,detail:{preserveKVEditor:preserveEditor}}));
    if(focusNew)row.querySelector('[data-kv-edit-row]:last-child [data-kv-edit-field="key"]')?.focus();
  }
  page.addEventListener('click',event=>{
    const button=event.target.closest('[data-kv-edit-command]');
    if(!button)return;
    const row=button.closest('[data-qrw-row]'),add=button.dataset.kvEditCommand==='add';
    updateKVEditRows(row,edits=>{
      if(add){if(edits.length<256)edits.push({key:'',action:'omit'});}
      else edits.splice(Number(button.closest('[data-kv-edit-row]').dataset.kvEditRow),1);
    },{focusNew:add});
  });
  page.addEventListener('change',event=>{
    const control=event.target.closest('[data-kv-edit-field]');
    if(!control)return;
    const index=Number(control.closest('[data-kv-edit-row]').dataset.kvEditRow);
    updateKVEditRows(control.closest('[data-qrw-row]'),edits=>{
      const edit=edits[index];
      if(!edit)return;
      edit[control.dataset.kvEditField]=control.value;
      if(edit.action!=='replace')delete edit.value;
    },{preserveEditor:control.dataset.kvEditField!=='action'});
  });
  page.addEventListener('change',async event=>{if(event.target.id!=='nfi-sample-file')return;const file=event.target.files?.[0];if(!file)return;try{if(file.size>1048576)throw new Error('Sample file exceeds the 1 MiB mapping limit.');const text=await file.text();state.nfiCorpus=text;state.nfiRaw=text.split(/\r?\n/).find(line=>line.trim())||'';el('nfi-raw').value=state.nfiRaw;el('nfi-corpus').value=text;el('nfi-inspector-host').innerHTML=qrwRawInspector();toast('Sample loaded locally. Learn structures or map the selected raw record.');}catch(error){toast(error.message,true);}});
  page.addEventListener('click',event=>{const button=event.target.closest('[data-action]');if(button?.dataset.action==='parser-helper-generate'){try{generateParserHelper(button.closest('form'));}catch(error){toast(error.message,true);}}});
  page.addEventListener('keydown',event=>{const tab=event.target.closest('[role=tab][data-assurance-tab],[role=tab][data-ai-tab],[role=tab][data-monitoring-tab]');if(!tab||!['ArrowRight','ArrowLeft','Home','End'].includes(event.key))return;event.preventDefault();const selector=tab.hasAttribute('data-ai-tab')?'[role=tab][data-ai-tab]':tab.hasAttribute('data-monitoring-tab')?'[role=tab][data-monitoring-tab]':'[role=tab][data-assurance-tab]',tabs=[...page.querySelectorAll(selector)],index=tabs.indexOf(tab),next=event.key==='Home'?0:event.key==='End'?tabs.length-1:(index+(event.key==='ArrowRight'?1:tabs.length-1))%tabs.length;tabs[next].focus();tabs[next].click();});
  let refreshAnimations=0;
  async function refreshCurrentWorkspace(){
    const button=el('refresh');
    refreshAnimations++;
    button.setAttribute('aria-busy','true');
    button.dataset.refreshing='true';
    try{await navigate(state.page);}
    finally{if(--refreshAnimations===0){button.removeAttribute('aria-busy');delete button.dataset.refreshing;}}
  }
  el('refresh').addEventListener('click',refreshCurrentWorkspace); el('logout').addEventListener('click',()=>signOut(true)); el('menu-toggle').addEventListener('click',()=>el('sidebar').classList.toggle('open'));
  el('notification-bell').addEventListener('click',()=>{const panel=el('notification-panel');panel.hidden=!panel.hidden;el('notification-bell').setAttribute('aria-expanded',String(!panel.hidden));if(!panel.hidden)refreshNotificationBell();});
  el('notification-list').addEventListener('click',async event=>{const button=event.target.closest('[data-notification-action]');if(!button)return;button.disabled=true;try{const action=button.dataset.notificationAction;if(action==='dismissed')await api(`/notifications/audit-${encodeURIComponent(button.dataset.id)}`,{method:'DELETE',body:{}});else await updateNotificationState(button.dataset.id,action);await refreshNotificationBell();}catch(error){toast(error.message,true);}finally{button.disabled=false;}});
  el('clear-notifications').addEventListener('click',async()=>{const button=el('clear-notifications');button.disabled=true;try{const unread=(state.controllerNotifications||[]).filter(note=>!note.read&&note.status!=='read');await Promise.all(unread.map(note=>api(`/notifications/audit-${encodeURIComponent(note.sequence)}`,{method:'POST',body:{action:'read'}})));state.notifications=[];await refreshNotificationBell();}catch(error){toast(error.message,true);}finally{button.disabled=false;}});
  document.addEventListener('click',event=>{if(!event.target.closest('.notification-wrap')){el('notification-panel').hidden=true;el('notification-bell').setAttribute('aria-expanded','false');}});
  window.addEventListener('hashchange',()=>navigate(location.hash.slice(1)||'operations'));
  buildNav();
  setControllerChecking();
  if (location.protocol === 'file:') {
    showPreviewOnly('Direct files are visual assets only. Run Open-NLI-WebView.ps1 or open the URL served by nli-controller-agent.');
    return;
  }
  setInterval(updateFreshness,5000);
  setInterval(()=>{if(state.user&&Date.now()-state.lastBrowserHeartbeatAt>=5000)sendBrowserHeartbeat();},1000);
  setInterval(()=>refreshApprovalBadge(),30000);
  setInterval(()=>refreshNotificationBell(),5000);
  setInterval(()=>refreshDashboardHistoryInBackground(),5000);
  // Reconcile the current workspace while preserving any form the operator has started editing.
  setInterval(async()=>{
    if(!state.user || state.loadingPage || document.hidden || page.dataset.dirty==='true' || !LOADERS[state.page] || SYNC_PAGES.has(state.page) || ['header-templates','kv-templates'].includes(state.page))return;
    const refreshAge=state.page==='ai'?5000:(state.page==='operations'?15000:30000);
    if(state.loadedAt && Date.now()-state.loadedAt<refreshAge)return;
    const current=state.page;
    state.loadingPage=true;
    try{await LOADERS[current]();state.lastRefreshFailure='';if(state.page===current && state.user){state.loadedAt=Date.now();updateFreshness();}}
    catch(error){if(state.lastRefreshFailure!==error.message){state.lastRefreshFailure=error.message;notify(`Live ${TITLES[current]} refresh failed: ${error.message}`,'error');}}
    finally{if(state.page===current)state.loadingPage=false;}
  },5000);
  checkController();
})();
