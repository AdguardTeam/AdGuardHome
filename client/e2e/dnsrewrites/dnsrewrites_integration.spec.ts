import { test, expect } from '../runtime/fixtures';
import { authed, UPSTREAM_HOST } from '../shared/api/test-fetch.ts';

import { addRewrite, deleteRewrite, updateRewrite, updateRewriteSettings, type DnsRewrite } from './dnsrewrites.ts';
import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import { waitFor } from '../shared/polling/retry.ts';
import { waitForAnswers } from '../shared/dns/dns-test-helpers.ts';
import { setCustomRules } from '../shared/adguard/filtering.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

async function useMock(agh: AdGuardContainer, api: AdGuardApiClient, answers: Array<{ domain: string; type: 'A' | 'AAAA'; data: string }>): Promise<MockDnsServer> {
  const upstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  for (const { domain, type, data } of answers) upstream.setAnswers(domain, type, [{ type, data }]);
  await upstream.start();
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
  } catch (err) {
    await upstream.stop();
    throw err;
  }
  return upstream;
}

const waitForExactAnswer = (agh: AdGuardContainer, domain: string, type: 'A' | 'AAAA', expected: string) =>
  waitForAnswers(agh, domain, type, (answers) => answers.includes(expected));

const waitForDifferentAnswer = (agh: AdGuardContainer, domain: string, type: 'A' | 'AAAA', unexpected: string) =>
  waitForAnswers(agh, domain, type, (answers) => answers.length > 0 && !answers.includes(unexpected));

test('10023 — Enable/disable a rewrite rule', async ({ agh, api }) => {
  const upstream = await useMock(agh, api, [
    { domain: 'vk.com', type: 'A', data: '93.184.216.34' },
    { domain: 'ya.ru', type: 'AAAA', data: '2001:db8::53' },
  ]);
  try {
    const rewriteV4: DnsRewrite = { domain: 'vk.com', answer: '127.0.0.1', enabled: true };
    const rewriteV6: DnsRewrite = { domain: 'ya.ru', answer: '::1', enabled: true };
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, rewriteV4, authed(api));
    await addRewrite(agh.baseUrl, rewriteV6, authed(api));

    expect((await waitForExactAnswer(agh, 'vk.com', 'A', '127.0.0.1')).includes('127.0.0.1')).toBeTruthy();
    expect((await waitForExactAnswer(agh, 'ya.ru', 'AAAA', '::1')).includes('::1')).toBeTruthy();

    await updateRewrite(agh.baseUrl, rewriteV4, { ...rewriteV4, enabled: false }, authed(api));
    await updateRewrite(agh.baseUrl, rewriteV6, { ...rewriteV6, enabled: false }, authed(api));

    expect((await waitForDifferentAnswer(agh, 'vk.com', 'A', '127.0.0.1')).includes('127.0.0.1')).toBe(false);
    expect((await waitForDifferentAnswer(agh, 'ya.ru', 'AAAA', '::1')).includes('::1')).toBe(false);
  } finally { await upstream.stop(); }
});

test('4159/4178 — Add and remove DNS rewrite', async ({ agh, api }) => {
  const REWRITE_IP = '192.0.2.1';
  const REAL_IP = '203.0.113.10';
  const upstream = await useMock(agh, api, [{ domain: 'rewrite-add-del.example', type: 'A', data: REAL_IP }]);
  try {
    const rewrite: DnsRewrite = { domain: 'rewrite-add-del.example', answer: REWRITE_IP, enabled: true };
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, rewrite, authed(api));
    expect((await waitForExactAnswer(agh, 'rewrite-add-del.example', 'A', REWRITE_IP)).includes(REWRITE_IP)).toBeTruthy();
    await deleteRewrite(agh.baseUrl, rewrite, authed(api));
    expect((await waitForExactAnswer(agh, 'rewrite-add-del.example', 'A', REAL_IP)).includes(REAL_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('10021 — Enable/disable DNS rewrites', async ({ agh, api }) => {
  const REWRITE_IP = '192.0.2.2';
  const REAL_IP = '203.0.113.20';
  const upstream = await useMock(agh, api, [{ domain: 'global-toggle.example', type: 'A', data: REAL_IP }]);
  try {
    const rewrite: DnsRewrite = { domain: 'global-toggle.example', answer: REWRITE_IP, enabled: true };
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, rewrite, authed(api));
    expect((await waitForExactAnswer(agh, 'global-toggle.example', 'A', REWRITE_IP)).includes(REWRITE_IP)).toBeTruthy();

    await updateRewriteSettings(agh.baseUrl, { enabled: false }, authed(api));
    const off = await waitForDifferentAnswer(agh, 'global-toggle.example', 'A', REWRITE_IP);
    expect(!off.includes(REWRITE_IP) && off.includes(REAL_IP)).toBeTruthy();

    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    expect((await waitForExactAnswer(agh, 'global-toggle.example', 'A', REWRITE_IP)).includes(REWRITE_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4160/4179 — Advanced DNS rewrites (wildcard): integration spec', async ({ agh, api }) => {
  const REWRITE_IP = '192.0.2.3';
  const upstream = await useMock(agh, api, [
    { domain: 'sub.wildcard.example', type: 'A', data: '203.0.113.30' },
    { domain: 'other.wildcard.example', type: 'A', data: '203.0.113.30' },
  ]);
  try {
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, { domain: '*.wildcard.example', answer: REWRITE_IP, enabled: true }, authed(api));
    expect((await waitForExactAnswer(agh, 'sub.wildcard.example', 'A', REWRITE_IP)).includes(REWRITE_IP)).toBeTruthy();
    expect((await waitForExactAnswer(agh, 'other.wildcard.example', 'A', REWRITE_IP)).includes(REWRITE_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4177 — Edit DNS rewrite: change resolved IP', async ({ agh, api }) => {
  const ORIGINAL_IP = '192.0.2.4';
  const UPDATED_IP = '192.0.2.5';
  const upstream = await useMock(agh, api, [{ domain: 'edit-rewrite.example', type: 'A', data: '203.0.113.40' }]);
  try {
    const original: DnsRewrite = { domain: 'edit-rewrite.example', answer: ORIGINAL_IP, enabled: true };
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, original, authed(api));
    expect((await waitForExactAnswer(agh, 'edit-rewrite.example', 'A', ORIGINAL_IP)).includes(ORIGINAL_IP)).toBeTruthy();
    await updateRewrite(agh.baseUrl, original, { domain: 'edit-rewrite.example', answer: UPDATED_IP, enabled: true }, authed(api));
    expect((await waitForExactAnswer(agh, 'edit-rewrite.example', 'A', UPDATED_IP)).includes(UPDATED_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4181 — IPv4-mapped IPv6 rewrite', async ({ agh, api }) => {
  const MAPPED_ANSWER = '::ffff:192.0.2.6';
  // dnslookup may print the IPv4-mapped address in dotted or canonical hex form.
  const FORMS = ['::ffff:192.0.2.6', '::ffff:c000:206'];
  const upstream = await useMock(agh, api, [{ domain: 'ipv4map.example', type: 'A', data: '203.0.113.50' }]);
  try {
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, { domain: 'ipv4map.example', answer: MAPPED_ANSWER, enabled: true }, authed(api));
    const answers = await waitFor(async () => {
      const { answers: a } = await agh.dnslookup('ipv4map.example', { type: 'AAAA' });
      return a.some((v) => FORMS.includes(v)) ? a : undefined;
    }, { timeoutMs: 15_000, intervalMs: 500 });
    expect(answers.some((v) => FORMS.includes(v)), `Expected an IPv4-mapped AAAA answer, got ${answers}`).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4159 — DNS rewrite override: priority over blocking rule', async ({ agh, api }) => {
  const REWRITE_IP = '192.0.2.7';
  const REAL_IP = '203.0.113.70';
  const upstream = await useMock(agh, api, [{ domain: 'override-filter.example', type: 'A', data: REAL_IP }]);
  try {
    await setCustomRules({ baseUrl: agh.baseUrl, fetchImpl: authed(api) }, ['||override-filter.example^']);
    expect((await waitForExactAnswer(agh, 'override-filter.example', 'A', '0.0.0.0')).includes('0.0.0.0')).toBeTruthy();

    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, { domain: 'override-filter.example', answer: REWRITE_IP, enabled: true }, authed(api));
    const after = await waitForExactAnswer(agh, 'override-filter.example', 'A', REWRITE_IP);
    expect(after.includes(REWRITE_IP) && !after.includes(REAL_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4177/10025 — Edit DNS rewrite to blocking', async ({ agh, api }) => {
  const ORIGINAL_IP = '192.0.2.10';
  const upstream = await useMock(agh, api, [{ domain: 'block-via-edit.example', type: 'A', data: '203.0.113.10' }]);
  try {
    const original: DnsRewrite = { domain: 'block-via-edit.example', answer: ORIGINAL_IP, enabled: true };
    await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
    await addRewrite(agh.baseUrl, original, authed(api));
    expect((await waitForExactAnswer(agh, 'block-via-edit.example', 'A', ORIGINAL_IP)).includes(ORIGINAL_IP)).toBeTruthy();
    await updateRewrite(agh.baseUrl, original, { domain: 'block-via-edit.example', answer: '0.0.0.0', enabled: true }, authed(api));
    expect((await waitForExactAnswer(agh, 'block-via-edit.example', 'A', '0.0.0.0')).includes('0.0.0.0')).toBeTruthy();
  } finally { await upstream.stop(); }
});
