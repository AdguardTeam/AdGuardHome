import { test, expect } from '../runtime/fixtures';
import { authed, UPSTREAM_HOST, ctxOf } from '../shared/api/test-fetch.ts';

import { waitFor } from '../shared/polling/retry.ts';
import { waitForDnsResult } from '../shared/dns/dns-test-helpers.ts';
import { setCustomRules } from '../shared/adguard/filtering.ts';
import { addClient, updateClient, deleteClient, listClients } from '../shared/adguard/clients.ts';
import { clearQueryLog, getQueryLog } from '../shared/adguard/querylog.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import { createMockUpstream } from '../shared/dns/mock-upstream.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';

const waitForDnsAnswer = (agh: AdGuardContainer, domain: string, type: string, predicate: (answers: string[], rcode: string) => boolean) =>
  waitForDnsResult(agh, domain, type, predicate);

test('4132 — Client name in query log', async ({ agh, api }) => {
  const domain = 'named-client-query.example';
  const clientName = 'friendly-client';
  const upstream = await createMockUpstream([{ domain, type: 'A', data: '203.0.113.73' }]);
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
    const ctx = ctxOf(agh, api);
    await clearQueryLog(ctx);
    await addClient(ctx, { name: clientName, ids: ['127.0.0.1'], use_global_settings: true });

    await agh.dnslookup(domain, { type: 'A' });
    const entry = await waitFor(async () => {
      const log = await getQueryLog(ctx, { search: domain, limit: 10 });
      return log.data.find((e) => (e.question?.name ?? e.question?.host ?? '') === domain);
    }, { timeoutMs: 10_000, intervalMs: 300 });

    expect(entry !== undefined, `Expected querylog entry for ${domain}`).toBeTruthy();
    expect(entry.client_info?.name, `Expected client_info.name '${clientName}', got '${entry.client_info?.name}'`).toBe(clientName);
  } finally { await upstream.stop(); }
});

test('4141 — Per-client upstream DNS', async ({ agh, api }) => {
  const GLOBAL_IP = '203.0.113.80';
  const CLIENT_IP = '203.0.113.81';
  const domain = 'per-client-upstream.example';
  const globalUpstream = await createMockUpstream([{ domain, type: 'A', data: GLOBAL_IP }]);
  const clientUpstream = await createMockUpstream([{ domain, type: 'A', data: CLIENT_IP }]);
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${globalUpstream.getPort()}`] }, api.authHeaders);
    const ctx = ctxOf(agh, api);
    await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes(GLOBAL_IP));

    await addClient(ctx, { name: 'custom-upstream-client', ids: ['127.0.0.1'], use_global_settings: true, upstreams: [`${UPSTREAM_HOST}:${clientUpstream.getPort()}`] });
    const { answers } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes(CLIENT_IP));
    expect(answers.includes(CLIENT_IP), `Expected per-client upstream answer ${CLIENT_IP}, got ${answers}`).toBeTruthy();
  } finally { await globalUpstream.stop(); await clientUpstream.stop(); }
});
