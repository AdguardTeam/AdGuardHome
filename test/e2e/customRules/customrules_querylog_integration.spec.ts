import { test, expect } from '../runtime/fixtures';
import { authed } from '../shared/api/test-fetch.ts';

import { setCustomRules } from '../shared/adguard/filtering.ts';
import { waitForQueryLogRecord } from '../shared/querylog/wait-for-querylog.ts';
import { startMockUpstream } from '../shared/dns/mock-upstream.ts';

test('4075 — $dnsrewrite modifier: original_answer in query log', async ({ agh, api }) => {
  const REWRITTEN_IP = '192.0.2.90';
  const domain = 'querylog-rewrite.example';
  const upstream = await startMockUpstream(agh, api, [{ domain, type: 'A', data: '203.0.113.90' }]);
  try {
    await setCustomRules({ baseUrl: agh.baseUrl, fetchImpl: authed(api) }, [`||${domain}^$dnsrewrite=${REWRITTEN_IP}`]);
    const { answers } = await agh.dnslookup(domain, { type: 'A' });
    expect(answers.includes(REWRITTEN_IP), `Expected rewritten IP ${REWRITTEN_IP}, got ${answers}`).toBeTruthy();

    const entry = await waitForQueryLogRecord(agh.baseUrl, domain, { fetchImpl: authed(api), timeoutMs: 15_000 });
    expect((entry.answer ?? []).some((a) => a.value === REWRITTEN_IP),
      `Expected querylog answer to contain ${REWRITTEN_IP}, got ${JSON.stringify(entry.answer)}`).toBeTruthy();
  } finally { await upstream.stop(); }
});
