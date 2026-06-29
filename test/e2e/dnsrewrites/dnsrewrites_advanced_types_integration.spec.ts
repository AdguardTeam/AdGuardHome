import { test, expect } from '../runtime/fixtures';
import { authed, UPSTREAM_HOST } from '../shared/api/test-fetch.ts';

import { addRewrite, deleteRewrite, updateRewriteSettings, type DnsRewrite } from './dnsrewrites.ts';
import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import { waitFor } from '../shared/polling/retry.ts';
import { waitForDnsStatus } from '../shared/dns/dns-test-helpers.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

const waitForDnsRcode = (agh: AdGuardContainer, domain: string, expectedRcode: string) =>
  waitForDnsStatus(agh, domain, expectedRcode);

test('4160 — Advanced DNS rewrites (wildcard): advanced-types spec', async ({ agh, api }) => {
  await updateRewriteSettings(agh.baseUrl, { enabled: true }, authed(api));
  const wildcard: DnsRewrite = { domain: '*.wildcard-test.internal', answer: '10.99.0.1' };
  await addRewrite(agh.baseUrl, wildcard, authed(api));

  for (const sub of ['alpha', 'beta', 'gamma']) {
    const name = `${sub}.wildcard-test.internal`;
    const { records } = await waitFor(async () => {
      const r = await agh.dnslookup(name, { type: 'A' });
      return r.status === 'NOERROR' && r.answers.length > 0 ? r : undefined;
    }, { timeoutMs: 10_000, intervalMs: 500 });
    const aAnswer = records.find((a) => a.type === 'A');
    expect(aAnswer, `Expected A answer for ${name}`).toBeTruthy();
    expect(aAnswer.data, `Expected 10.99.0.1 for ${name}, got ${aAnswer.data}`).toBe('10.99.0.1');
  }

  await deleteRewrite(agh.baseUrl, wildcard, authed(api));
});
