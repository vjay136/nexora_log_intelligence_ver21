import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const app = fs.readFileSync(path.join(root, 'web', 'app.js'), 'utf8');
const ui = fs.readFileSync(path.join(root, 'web', 'ui.spec.js'), 'utf8');

const routes = [
  'operations', 'monitoring', 'streams',
  'assurance', 'quarantine-wizard', 'upes', 'rollouts', 'parser-sync',
  'artifacts', 'parser-workspace', 'nupire', 'field-integrity', 'ai',
  'parsers', 'studio', 'regex', 'knownkeys', 'drift', 'notifications',
  'sync-fabric', 'nhle-monitor', 'napi-monitor', 'nal-monitor',
  'input-paths', 'migration', 'outputs', 'metadata', 'unknowns', 'performance',
  'kv', 'static', 'projection', 'semantic',
  'engine-network', 'engine-url', 'engine-email', 'engine-subject',
  'fleet', 'clients', 'controller', 'api-monitor', 'system', 'communications', 'security',
];

const innerTabs = {
  monitoring: [
    'Overview', 'Fleet & Nodes', 'Pipeline', 'Sources', 'Parser Performance',
    'Destinations', 'Queues / Replay / Quarantine', 'Infrastructure',
    'Latency & Synchronization', 'Errors & Incidents', 'Assurance & Release Health',
  ],
  assurance: [
    'Analyse', 'Corpus reports', 'Parser drafts', 'Review Queue', 'History',
    'Fixtures', 'Guided Parser Repair Studio', 'Parser Creation Helper',
    'NAPI Synthesis', 'Test Plans', 'Runs',
  ],
  upes: [
    'Live Failure Observatory', 'Recovery Decision Trace', 'Parser Repair Workshop',
    'Knowledge Base', 'Discovery Jobs', 'Parser Health & Drift',
    'Deployment & Rollback', 'Runtime Health',
  ],
  outputs: ['Destinations', 'Security & credentials', 'Validation', 'Saved profiles', 'Live & rollback'],
  ai: ['Provider', 'Status', 'Results', 'Parser', 'Regex', 'Template', 'Approval'],
  qrw: [
    'Select', 'Inspect', 'Schema', 'Map', 'Generate', 'Test',
    'Review & publish', 'Deploy & replay', 'Verify result',
  ],
  subsystem: ['Overview', 'Live activity', 'Freshness', 'Diagnostics'],
};

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

test('Ver21 R4 declares the complete ordered 44-workspace contract', () => {
  const nav = app.slice(app.indexOf('const NAV = ['), app.indexOf('const TITLES'));
  const actual = [...nav.matchAll(/\[\s*'[^']+'\s*,\s*'([^']+)'/g)].map((match) => match[1]);
  assert.deepEqual(actual, routes);
});

test('every declared workspace has a loader and stateful workspaces have live badges', () => {
  const match = app.match(/const LOADERS = \{([^;]+)\};/s);
  assert.ok(match, 'LOADERS registry must be present');
  for (const route of routes) {
    assert.match(match[1], new RegExp(`(?:'${escapeRegex(route)}'|\\b${escapeRegex(route)}\\b)\\s*:`), `missing loader for ${route}`);
  }

  const badgeMatch = app.match(/const NAV_COUNT_PAGES = new Set\(\[([^\]]+)\]\)/s);
  assert.ok(badgeMatch, 'NAV_COUNT_PAGES must be present');
  for (const route of ['assurance', 'quarantine-wizard', 'upes', 'rollouts', 'parser-workspace', 'parsers', 'unknowns']) {
    assert.ok(badgeMatch[1].includes(`'${route}'`), `${route} must expose actionable live counts`);
  }
});

test('all documented inner-tab workspaces remain implemented and browser-audited', () => {
  const combined = `${app}\n${ui}`.replaceAll('&amp;', '&');
  for (const [workspace, labels] of Object.entries(innerTabs)) {
    for (const label of labels) {
      assert.ok(combined.includes(label), `${workspace} inner tab is missing: ${label}`);
    }
  }
  assert.ok(ui.includes("test('all workspaces render and audit is visible'"));
  assert.ok(ui.includes("test('every visible inner tab activates without browser, request, or render errors'"));
});

test('critical UI actions remain wired to authoritative controller APIs', () => {
  for (const endpoint of [
    '/assurance/parsers', '/rollouts', '/parser-sync', '/config/input-paths',
    '/config/output-destinations', '/quarantine/records/', '/notifications',
  ]) {
    assert.ok(app.includes(endpoint), `missing controller API wiring: ${endpoint}`);
  }
});
