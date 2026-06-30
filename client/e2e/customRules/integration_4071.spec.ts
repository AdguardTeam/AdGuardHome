import { setTimeout } from 'node:timers/promises';
import { test, expect } from '../runtime/fixtures';
import { resolveAnswers as runDnsQuery } from '../shared/dns/dns-test-helpers.ts';
import { authed, UPSTREAM_HOST } from '../shared/api/test-fetch.ts';
import { runCustomRuleTestCase, type CustomRuleTestCase } from './customrules.ts';
import { addClient } from '../shared/api/adguard-api.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

const BLOCKED_IPV4_ANSWERS = ['0.0.0.0'];
const BLOCKED_IPV6_ANSWERS = ['::'];
const BASELINE_ALLOWED_IPV4 = '93.184.216.34';
const BASELINE_ALLOWED_IPV6 = '2001:db8::4071';

function normalizeAnswers(answers: string[]): string[] {
  return [...new Set(answers)].sort((a, b) => a.localeCompare(b));
}

function assertDnsAnswersEqual(actual: string[], expected: string[], description: string): void {
  expect(normalizeAnswers(actual),
    `${description}\nExpected: ${JSON.stringify(normalizeAnswers(expected))}\nActual: ${JSON.stringify(normalizeAnswers(actual))}`).toEqual(normalizeAnswers(expected));
}

async function waitForExpectedDnsAnswers(agh: AdGuardContainer, opts: {
  domain: string; type: string; expectedAnswers: string[]; timeoutMs?: number; retryDelayMs?: number;
}): Promise<void> {
  const timeoutMs = opts.timeoutMs ?? 8_000;
  const retryDelayMs = opts.retryDelayMs ?? 500;
  const startedAt = Date.now();
  let lastAnswers: string[] = [];
  while (Date.now() - startedAt < timeoutMs) {
    lastAnswers = await runDnsQuery(agh, opts.domain, opts.type);
    if (normalizeAnswers(lastAnswers).join('\n') === normalizeAnswers(opts.expectedAnswers).join('\n')) return;
    await setTimeout(retryDelayMs);
  }
  assertDnsAnswersEqual(lastAnswers, opts.expectedAnswers,
    `DNS response mismatch for ${opts.domain} (${opts.type}) after waiting ${timeoutMs}ms`);
}

function createCustomRuleContext(agh: AdGuardContainer, api: AdGuardApiClient, opts: {
  expectedAnswers: string[]; timeoutMs?: number; initialDelayMs?: number;
}) {
  return {
    adGuardBaseUrl: agh.baseUrl,
    fetchImpl: authed(api),
    runDnsLookup: async (query: CustomRuleTestCase['query']) => {
      if ((opts.initialDelayMs ?? 0) > 0) await setTimeout(opts.initialDelayMs);
      await waitForExpectedDnsAnswers(agh, { domain: query.domain, type: query.type, expectedAnswers: opts.expectedAnswers, timeoutMs: opts.timeoutMs });
    },
  };
}

test('4071 — Custom rule for specific IP', async ({ agh, api }) => {
  test.setTimeout(120_000);
  const mockUpstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  mockUpstream.setAnswers('example.org', 'A', [{ type: 'A', data: BASELINE_ALLOWED_IPV4 }]);
  mockUpstream.setAnswers('www.example.org', 'AAAA', [{ type: 'AAAA', data: BASELINE_ALLOWED_IPV6 }]);
  await mockUpstream.start();

  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${mockUpstream.getPort()}`] }, api.authHeaders);
    await addClient(api, {
      name: 'TestClient',
      ids: ['127.0.0.1'],
      use_global_blocked_services: true,
      use_global_settings: true,
    });

    const baselineAllowedIpv6 = await runDnsQuery(agh, 'www.example.org', 'AAAA');
    assertDnsAnswersEqual(baselineAllowedIpv6, [BASELINE_ALLOWED_IPV6],
      'Precondition: expected deterministic AAAA answers for www.example.org before custom rules');
    expect(!baselineAllowedIpv6.includes('::'), `Precondition: expected real AAAA answers, got ${JSON.stringify(baselineAllowedIpv6)}`).toBeTruthy();

    const rule1 = "||example.org^$client='127.0.0.1'";
    const rule2 = "||www.example.org^$client='127.0.0.1'";
    const rule3 = "@@||www.example.org^$client='127.0.0.1'";

    await runCustomRuleTestCase({
      name: 'Step 1: Block example.org', customRule: [rule1],
      query: { domain: 'example.org', type: 'A' }, expected: { answerValues: BLOCKED_IPV4_ANSWERS },
    }, createCustomRuleContext(agh, api, { expectedAnswers: BLOCKED_IPV4_ANSWERS, timeoutMs: 8_000 }));

    await runCustomRuleTestCase({
      name: 'Step 2: Block www.example.org', customRule: [rule1, rule2],
      query: { domain: 'www.example.org', type: 'AAAA' }, expected: { answerValues: BLOCKED_IPV6_ANSWERS },
    }, createCustomRuleContext(agh, api, { expectedAnswers: BLOCKED_IPV6_ANSWERS, timeoutMs: 8_000 }));

    await runCustomRuleTestCase({
      name: 'Step 3: Allow www.example.org', customRule: [rule1, rule2, rule3],
      query: { domain: 'www.example.org', type: 'AAAA' }, expected: { answerValues: baselineAllowedIpv6, notEmptyAnswer: true },
    }, createCustomRuleContext(agh, api, { expectedAnswers: baselineAllowedIpv6, timeoutMs: 20_000, initialDelayMs: 2_000 }));
  } finally {
    await mockUpstream.stop();
  }
});
