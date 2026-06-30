import { test, expect } from '../runtime/fixtures';
import { resolveAnswers } from '../shared/dns/dns-test-helpers.ts';
import { authed, UPSTREAM_HOST } from '../shared/api/test-fetch.ts';

import { runCustomRuleTestCase, type CustomRuleTestCase } from './customrules.ts';
import { clearQueryLog } from '../shared/api/adguard-api.ts';
import { waitFor } from '../shared/polling/retry.ts';
import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

function createRuleAwareDnsLookup(agh: AdGuardContainer, predicate: (answers: string[]) => boolean) {
  return async (query: CustomRuleTestCase['query']) => {
    await waitFor(async () => {
      const answers = await resolveAnswers(agh, query.domain, query.type);
      return predicate(answers) ? answers : undefined;
    }, { timeoutMs: 30_000, intervalMs: 500 });
  };
}

test('4072 — Custom rule for specific client', async ({ agh, api }) => {
  test.setTimeout(90_000);
  // Mock upstream returns 1.2.3.4 for all A queries (default), satisfying the precondition.
  const upstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  await upstream.start();

  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
    const addClientRes = await authed(api)(`${agh.baseUrl}/control/clients/add`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'named-client', ids: ['127.0.0.1'], use_global_settings: true }),
    });
    expect(addClientRes.ok, `Failed to add test client: ${addClientRes.status} ${await addClientRes.text()}`).toBe(true);

    await clearQueryLog(api);

    const baselineAnswers = await resolveAnswers(agh, 'example.org', 'A');
    expect(baselineAnswers.length > 0 && !baselineAnswers.includes('0.0.0.0'),
      `Precondition failed: expected real answers for example.org, got ${JSON.stringify(baselineAnswers)}`).toBeTruthy();

    await runCustomRuleTestCase({
      name: 'Block example.org for a persistent client identified by name',
      customRule: "||example.org^$client='named-client'",
      query: { domain: 'example.org', type: 'A' },
      expected: { answerValues: ['0.0.0.0'], rule: "||example.org^$client='named-client'" },
    }, {
      adGuardBaseUrl: agh.baseUrl,
      fetchImpl: authed(api),
      runDnsLookup: createRuleAwareDnsLookup(agh, (answers) => answers.includes('0.0.0.0')),
    });
  } finally { await upstream.stop(); }
});
