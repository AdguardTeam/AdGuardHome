import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { authed, UPSTREAM_HOST, ctxOf } from '../shared/api/test-fetch.ts';

import { allocateUdpPort, MockDnsServer } from '../dnsSettings/MockDnsServer.ts';
import { waitFor } from '../shared/polling/retry.ts';
import { waitForDnsResult } from '../shared/dns/dns-test-helpers.ts';
import { setCustomRules } from '../shared/adguard/filtering.ts';
import { addBlockList } from '../blocklists/blocklists.ts';
import { addClient, updateClient } from '../shared/adguard/clients.ts';
import { setDnsConfig } from '../dnsSettings/dns_settings.ts';
import { startMockUpstream } from '../shared/dns/mock-upstream.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';

const waitForDnsAnswer = (agh: AdGuardContainer, domain: string, type: string, predicate: (answers: string[], rcode: string) => boolean) =>
  waitForDnsResult(agh, domain, type, predicate);

test('4075 — $dnsrewrite modifier: IPv4/A', async ({ agh, api }) => {
  const domain = 'dnsrewrite-a.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.1' }]);
  try {
    await setCustomRules(ctxOf(agh, api), [`||${domain}^$dnsrewrite=192.0.2.20`]);
    const { answers } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('192.0.2.20'));
    assert.ok(answers.includes('192.0.2.20'), `Expected 192.0.2.20 from $dnsrewrite, got ${answers}`);
  } finally { await upstream.stop(); }
});

test('4075 — $dnsrewrite modifier: hostname/CNAME', async ({ agh, api }) => {
  const CNAME_TARGET = 'cname-target.example';
  const domain = 'dnsrewrite-cname.example';
  const upstream = await startMockUpstream(agh, api, [{ domain: CNAME_TARGET, type: 'A', data: '203.0.113.2' }]);
  try {
    await setCustomRules(ctxOf(agh, api), [`||${domain}^$dnsrewrite=${CNAME_TARGET}`]);
    const { answers } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.some((v) => v.includes(CNAME_TARGET) || v === '203.0.113.2'));
    assert.ok(answers.some((v) => v.includes(CNAME_TARGET) || v === '203.0.113.2'), `Expected CNAME target or resolved IP, got ${answers}`);
  } finally { await upstream.stop(); }
});

test('4075 — $dnsrewrite modifier: REFUSED', async ({ agh, api }) => {
  const domain = 'dnsrewrite-refused.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.3' }]);
  try {
    await setCustomRules(ctxOf(agh, api), [`||${domain}^$dnsrewrite=REFUSED`]);
    const { rcode } = await waitForDnsAnswer(agh, domain, 'A', (_a, rc) => rc === 'REFUSED');
    assert.equal(rcode, 'REFUSED');
  } finally { await upstream.stop(); }
});

test('4075 — $dnsrewrite modifier: IPv6/AAAA', async ({ agh, api }) => {
  const REWRITTEN_AAAA = '::abcd:1234';
  const domain = 'dnsrewrite-aaaa.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'AAAA', data: '2001:db8::1' }]);
  try {
    await setCustomRules(ctxOf(agh, api), [`||${domain}^$dnsrewrite=${REWRITTEN_AAAA}`]);
    const { answers } = await waitForDnsAnswer(agh, domain, 'AAAA',
      (a) => a.some((v) => v.replace(/^0+/g, '') === REWRITTEN_AAAA.replace(/^0+/g, '') || v === REWRITTEN_AAAA));
    assert.ok(answers.length > 0, `Expected AAAA answer from $dnsrewrite, got ${answers}`);
  } finally { await upstream.stop(); }
});

test('4079 — Blocked by CNAME', async ({ agh, api }) => {
  const CNAME_TARGET = 'cdn-blocked.example';
  const domain = 'www-cname-blocked.example';
  const upstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  upstream.setAnswers(domain, 'A', [{ type: 'CNAME', data: CNAME_TARGET }, { type: 'A', data: '203.0.113.5' }]);
  upstream.setAnswers(CNAME_TARGET, 'A', [{ type: 'A', data: '203.0.113.5' }]);
  await upstream.start();
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
    const before = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('203.0.113.5'));
    assert.ok(before.answers.includes('203.0.113.5'), 'Precondition: root domain should resolve before blocking');
    await setCustomRules(ctxOf(agh, api), [`||${CNAME_TARGET}^`]);
    const after = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('0.0.0.0'));
    assert.ok(after.answers.includes('0.0.0.0'), `Expected 0.0.0.0 after blocking CNAME target, got ${after.answers}`);
  } finally { await upstream.stop(); }
});

test('4077 — $important modifier: block over allowlist', async ({ agh, api }) => {
  const domain = 'important-block.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.6' }]);
  try {
    await setCustomRules(ctxOf(agh, api), [`@@||${domain}^`, `||${domain}^$important`]);
    const { answers } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('0.0.0.0'));
    assert.ok(answers.includes('0.0.0.0'), `Expected $important to override allowlist, got ${answers}`);
  } finally { await upstream.stop(); }
});

test('4077 — $important modifier: allowlist over block', async ({ agh, api }) => {
  const domain = 'important-allow.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.7' }]);
  try {
    await setCustomRules(ctxOf(agh, api), [`||${domain}^$important`, `@@||${domain}^$important`]);
    const { answers } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('203.0.113.7'));
    assert.ok(answers.includes('203.0.113.7'), `Expected @@$important to win, got ${answers}`);
  } finally { await upstream.stop(); }
});

test('4074 — $denyallow modifier', async ({ agh, api }) => {
  const comDomain = 'denyallow-test.com';
  const netDomain = 'denyallow-test.net';
  const upstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  upstream.setAnswers(comDomain, 'A', [{ type: 'A', data: '203.0.113.9' }]);
  upstream.setAnswers(netDomain, 'A', [{ type: 'A', data: '203.0.113.10' }]);
  await upstream.start();
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
    await setCustomRules(ctxOf(agh, api), [`||${comDomain}^`, `||${netDomain}^`, `@@*$denyallow=net`]);
    const { answers: comAnswers } = await waitForDnsAnswer(agh, comDomain, 'A', (a) => a.includes('203.0.113.9'));
    assert.ok(comAnswers.includes('203.0.113.9'), `Expected ${comDomain} allowed, got ${comAnswers}`);
    const { answers: netAnswers } = await waitForDnsAnswer(agh, netDomain, 'A', (a) => a.includes('0.0.0.0'));
    assert.ok(netAnswers.includes('0.0.0.0'), `Expected ${netDomain} blocked, got ${netAnswers}`);
  } finally { await upstream.stop(); }
});

test('4073 — $badfilter modifier', async ({ agh, api }) => {
  const domain = 'badfilter-test.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.20' }]);
  try {
    const listUrl = await agh.serveRules('badfilter-list.txt', `! Title: Badfilter Test List\n||${domain}^\n`);
    await addBlockList(agh.baseUrl, {
      name: 'Badfilter Test List', url: listUrl, whitelist: false,
    }, authed(api));

    await waitFor(async () => {
      const res = await authed(api)(`${agh.baseUrl}/control/filtering/status`);
      const status = await res.json() as { filters?: Array<{ url: string; rules_count?: number; enabled?: boolean }> };
      return status.filters?.find((f) => f.url.includes('badfilter-list.txt') && (f.rules_count ?? 0) > 0 && f.enabled) ? true : undefined;
    }, { timeoutMs: 15_000, intervalMs: 500 });

    const { answers: blocked } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('0.0.0.0'));
    assert.ok(blocked.includes('0.0.0.0'), `Precondition: expected ${domain} blocked by blocklist`);

    await setCustomRules(ctxOf(agh, api), [`||${domain}^$badfilter`]);
    const { answers: unblocked } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('203.0.113.20'));
    assert.ok(unblocked.includes('203.0.113.20'), `Expected $badfilter to lift block, got ${unblocked}`);
  } finally { await upstream.stop(); }
});

test('4082 — CNAME advanced blocking', async ({ agh, api }) => {
  const cnameTarget = 'target.cname82.example';
  const rootDomain = 'mail.cname82.example';
  const upstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  upstream.setAnswers(rootDomain, 'A', [{ type: 'CNAME', data: cnameTarget }, { type: 'A', data: '203.0.113.21' }]);
  upstream.setAnswers(cnameTarget, 'A', [{ type: 'A', data: '203.0.113.21' }]);
  await upstream.start();
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
    const { answers: beforeRoot } = await waitForDnsAnswer(agh, rootDomain, 'A', (a) => a.includes('203.0.113.21'));
    assert.ok(beforeRoot.includes('203.0.113.21'), `Precondition: ${rootDomain} should resolve`);
    await setCustomRules(ctxOf(agh, api), [`||${cnameTarget}^`]);
    const { answers: rootBlocked } = await waitForDnsAnswer(agh, rootDomain, 'A', (a) => a.includes('0.0.0.0'));
    assert.ok(rootBlocked.includes('0.0.0.0'), `Expected ${rootDomain} blocked via CNAME, got ${rootBlocked}`);
    const { answers: targetBlocked } = await waitForDnsAnswer(agh, cnameTarget, 'A', (a) => a.includes('0.0.0.0'));
    assert.ok(targetBlocked.includes('0.0.0.0'), `Expected ${cnameTarget} direct query blocked, got ${targetBlocked}`);
  } finally { await upstream.stop(); }
});

test('4084 — check_host filtering API', async ({ agh, api }) => {
  const domain = 'checkhost.example';
  const ctx = ctxOf(agh, api);
  async function checkHost(name: string, client?: string): Promise<{ reason: string; ip_addrs?: string[] | null }> {
    const url = new URL(`${agh.baseUrl}/control/filtering/check_host`);
    url.searchParams.set('name', name);
    if (client) url.searchParams.set('client', client);
    const res = await authed(api)(url.toString());
    assert.ok(res.ok, `check_host failed: ${res.status}`);
    return res.json() as Promise<{ reason: string; ip_addrs?: string[] | null }>;
  }
  await addClient(ctx, { name: 'test-4084', ids: ['127.0.0.1'], use_global_settings: true, use_global_blocked_services: true, filtering_enabled: true });
  await setCustomRules(ctx, [`||${domain}^$client=test-4084`]);
  assert.equal((await checkHost(domain, '127.0.0.1')).reason, 'FilteredBlackList');
  assert.equal((await checkHost(domain, '10.10.10.10')).reason, 'NotFilteredNotFound');
  await setCustomRules(ctx, [`||${domain}^$client=127.0.0.1/16,dnsrewrite=NOERROR;A;10.0.0.250`]);
  assert.equal((await checkHost(domain, '127.0.0.1')).reason, 'RewriteRule');
});

test('4076 — $ctag modifier', async ({ agh, api }) => {
  const domain = 'ctag-blocked.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.90' }]);
  const ctx = ctxOf(agh, api);
  try {
    await setCustomRules(ctx, [`||${domain}^$ctag=user_child`]);
    const { answers: before } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('203.0.113.90'));
    assert.ok(before.includes('203.0.113.90'), `Expected resolve before tagging, got ${before}`);
    await addClient(ctx, { name: 'child-client', ids: ['127.0.0.1'], tags: ['user_child'], use_global_settings: true });
    const { answers: after } = await waitForDnsAnswer(agh, domain, 'A', (a, rc) => a.includes('0.0.0.0') || rc === 'NXDOMAIN');
    assert.ok(after.includes('0.0.0.0') || after.length === 0, `Expected blocked for user_child, got ${after}`);
    await updateClient(ctx, 'child-client', { name: 'child-client', ids: ['127.0.0.1'], tags: ['device_laptop'], use_global_settings: true });
    const { answers: afterRetag } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('203.0.113.90'));
    assert.ok(afterRetag.includes('203.0.113.90'), `Expected resolve after retag, got ${afterRetag}`);
  } finally { await upstream.stop(); }
});

test('4204 — IPv4-only rewrite returns NODATA for AAAA', async ({ agh, api }) => {
  const domain = 'type-a-only.example';
  await setCustomRules(ctxOf(agh, api), [`||${domain}^$dnsrewrite=NOERROR;A;5.5.5.5`]);
  const { answers } = await waitForDnsAnswer(agh, domain, 'A', (a) => a.includes('5.5.5.5'));
  assert.ok(answers.includes('5.5.5.5'), `Expected A answer 5.5.5.5, got ${answers}`);
  const aaaa = await agh.dnslookup(domain, { type: 'AAAA' });
  assert.equal(aaaa.status, 'NOERROR', `Expected NOERROR (NODATA) for AAAA, got ${aaaa.status}`);
  assert.equal(aaaa.records.filter((r) => r.type === 'AAAA').length, 0, `Expected no AAAA answers, got ${JSON.stringify(aaaa.records)}`);
});
