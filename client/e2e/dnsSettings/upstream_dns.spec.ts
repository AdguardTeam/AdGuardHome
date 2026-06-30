// @egress — every test here verifies real upstream protocols (Plain/DoT/DoH/DoQ)
// against public resolvers (Quad9/Google/Cloudflare); these cannot be mocked.
import { setTimeout } from 'node:timers/promises';
import { test, expect } from '../runtime/fixtures';
import { authed } from '../shared/api/test-fetch.ts';
import { setUpstreamDNS, checkQueryLog, getUpstreamDNS } from './upstream_dns.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

const DEFAULT_UPSTREAM = 'https://dns10.quad9.net/dns-query';

async function runDnsQueryAndCheckUpstream(
  agh: AdGuardContainer,
  api: AdGuardApiClient,
  domain: string,
  expectedUpstream: string,
  options: { minRecordOffsetMs?: number } = {},
): Promise<void> {
  const queryStartedAtMs = Date.now() - (options.minRecordOffsetMs ?? 5_000);
  await agh.dnslookup(domain, { type: 'A' });
  await checkQueryLog(authed(api), agh.baseUrl, domain, expectedUpstream, { minRecordTimeMs: queryStartedAtMs });
}

async function assertSavedUpstreamDns(
  agh: AdGuardContainer,
  api: AdGuardApiClient,
  expectedUpstreams: string[],
  expectedMode?: string,
): Promise<void> {
  const dnsInfo = await getUpstreamDNS(authed(api), agh.baseUrl);
  expect(dnsInfo.upstream_dns,
    `Saved upstream DNS does not match. Expected ${JSON.stringify(expectedUpstreams)}, got ${JSON.stringify(dnsInfo.upstream_dns)}`).toEqual(expectedUpstreams);
  if (expectedMode) {
    expect(dnsInfo.upstream_mode || 'load_balance').toBe(expectedMode);
  }
}

test('4085 — Upstream DNS servers: plain DNS & defaults', async ({ agh, api }) => {
  const initialConfig = await getUpstreamDNS(authed(api), agh.baseUrl);
  expect(initialConfig.upstream_mode || 'load_balance', 'Default upstream mode should be load_balance').toBe('load_balance');
  expect(initialConfig.upstream_dns.includes(DEFAULT_UPSTREAM),
    `Default upstream DNS should include ${DEFAULT_UPSTREAM}, got ${JSON.stringify(initialConfig.upstream_dns)}`).toBeTruthy();

  const upstream = '94.140.14.140';
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: [upstream] });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, [upstream], 'load_balance');
  await runDnsQueryAndCheckUpstream(agh, api, 'example.org', `${upstream}:53`);
});

test('4085 — Upstream DNS servers: DoT', async ({ agh, api }) => {
  const upstream = 'tls://dns-unfiltered.adguard.com';
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: [upstream] });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, [upstream], 'load_balance');
  await runDnsQueryAndCheckUpstream(agh, api, 'google.com', `${upstream}:853`);
});

test('4085 — Upstream DNS servers: DoH', async ({ agh, api }) => {
  const upstream = 'https://dns-unfiltered.adguard.com/dns-query';
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: [upstream] });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, [upstream], 'load_balance');
  await runDnsQueryAndCheckUpstream(agh, api, 'yahoo.com', upstream.replace('.com/', '.com:443/'));
});

test('4085 — Upstream DNS servers: DoQ', async ({ agh, api }) => {
  const upstream = 'quic://dns-unfiltered.adguard.com:784';
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: [upstream] });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, [upstream], 'load_balance');
  await runDnsQueryAndCheckUpstream(agh, api, 'bing.com', upstream);
});

test('4085 — Upstream DNS servers: specific DoH', async ({ agh, api }) => {
  const upstream = 'https://dns.adguard.com/dns-query';
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: [upstream] });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, [upstream], 'load_balance');
  await runDnsQueryAndCheckUpstream(agh, api, 'duckduckgo.com', upstream.replace('.com/', '.com:443/'));
});

test('4085 — Upstream DNS servers: domain-specific', async ({ agh, api }) => {
  const specificUpstream = '94.140.14.15';
  const defaultUpstream = 'tls://dns.adguard.com';
  const configuredUpstreams = [defaultUpstream, '[/ya.ru/gismeteo.ru/]94.140.14.15'];

  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: configuredUpstreams });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, configuredUpstreams, 'load_balance');

  await runDnsQueryAndCheckUpstream(agh, api, 'ya.ru', `${specificUpstream}:53`);
  await runDnsQueryAndCheckUpstream(agh, api, 'gismeteo.ru', `${specificUpstream}:53`);
  await runDnsQueryAndCheckUpstream(agh, api, 'rbc.ru', `${defaultUpstream}:853`);
});

test('4085 — Upstream DNS servers: comments', async ({ agh, api }) => {
  const comment = '# comment';
  const upstream = '8.8.8.8';
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: [comment, upstream] });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, [comment, upstream], 'load_balance');
  await runDnsQueryAndCheckUpstream(agh, api, 'cloudflare.com', `${upstream}:53`);
});

test('4085 — Upstream DNS servers: complex domain-specific', async ({ agh, api }) => {
  const configuredUpstreams = ['8.8.8.8:53', '[/host.com/]1.1.1.1:53', '[/*.host.com/]2.2.2.2:53'];
  await setUpstreamDNS(authed(api), agh.baseUrl, { upstream_dns: configuredUpstreams });
  await setTimeout(2000);
  await assertSavedUpstreamDns(agh, api, configuredUpstreams, 'load_balance');

  await runDnsQueryAndCheckUpstream(agh, api, 'host.com', '1.1.1.1:53');
  await runDnsQueryAndCheckUpstream(agh, api, 'vk.com', '8.8.8.8:53');
});
