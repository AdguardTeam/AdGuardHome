import { test, expect } from '../runtime/fixtures';
import { authed } from '../shared/api/test-fetch.ts';
import { startMockUpstream } from '../shared/dns/mock-upstream.ts';
import { clearDnsCache, setDnsConfig, getDnsInfo } from '../shared/dns/dns-settings.ts';
import { waitFor } from '../shared/polling/retry.ts';

const blockedDomain = `blocked-mode-static.test`;

test.describe('DNS Core Configuration Tests (Cases 4102, 4105, 4106)', () => {
  // Case #4106: Blocking modes
  test('4106 — Blocking mode', async ({ agh, api }) => {
    // The upstream is incidental here — the test domain is blocked before AGH
    // would ever forward it — so use a local mock instead of a public resolver.
    const upstream = await startMockUpstream(agh, api, []);
    try {
      const setRulesRes = await authed(api)(`${agh.baseUrl}/control/filtering/set_rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rules: [`||${blockedDomain}^`] }),
      });
      expect(setRulesRes.ok).toBeTruthy();

      // 1. Default blocking -> 0.0.0.0
      await setDnsConfig(agh.baseUrl, { blocking_mode: 'default' }, api.authHeaders);
      await clearDnsCache(agh.baseUrl, api.authHeaders);
      const defaultIps = await waitFor(async () => {
        const { answers } = await agh.dnslookup(blockedDomain, { type: 'A' });
        return answers.includes('0.0.0.0') ? answers : undefined;
      }, { timeoutMs: 15_000, intervalMs: 500 });
      expect(defaultIps).toContain('0.0.0.0');

      // 2. REFUSED
      await setDnsConfig(agh.baseUrl, { blocking_mode: 'refused' }, api.authHeaders);
      await clearDnsCache(agh.baseUrl, api.authHeaders);
      const refusedStatus = await waitFor(async () => {
        const { status } = await agh.dnslookup(blockedDomain, { type: 'A' });
        return status === 'REFUSED' || status === 'SERVFAIL' ? status : undefined;
      }, { timeoutMs: 15_000, intervalMs: 500 });
      expect(['REFUSED', 'SERVFAIL']).toContain(refusedStatus);

      // 3. NXDOMAIN
      await setDnsConfig(agh.baseUrl, { blocking_mode: 'nxdomain' }, api.authHeaders);
      await clearDnsCache(agh.baseUrl, api.authHeaders);
      const nxStatus = await waitFor(async () => {
        const { status } = await agh.dnslookup(blockedDomain, { type: 'A' });
        return status === 'NXDOMAIN' ? status : undefined;
      }, { timeoutMs: 15_000, intervalMs: 500 });
      expect(nxStatus).toBe('NXDOMAIN');

      // 4. Custom IP
      await setDnsConfig(agh.baseUrl, {
        blocking_mode: 'custom_ip',
        blocking_ipv4: '1.2.3.4',
        blocking_ipv6: '::1',
      }, api.authHeaders);
      await clearDnsCache(agh.baseUrl, api.authHeaders);
      const customIps = await waitFor(async () => {
        const { answers } = await agh.dnslookup(blockedDomain, { type: 'A' });
        return answers.includes('1.2.3.4') ? answers : undefined;
      }, { timeoutMs: 15_000, intervalMs: 500 });
      expect(customIps).toEqual(['1.2.3.4']);
    } finally {
      await upstream.stop();
    }
  });

  // Case #4102: Rate Limit
  test('4102 — Rate limit', async ({ agh, api }) => {
    // Rate limiting drops by request rate, not by answer, so the upstream is
    // incidental — a local mock keeps the test off the public internet.
    const upstream = await startMockUpstream(agh, api, []);
    try {
      await setDnsConfig(agh.baseUrl, { ratelimit: 1 }, api.authHeaders);
      const info = await getDnsInfo(agh.baseUrl, api.authHeaders);
      expect(info.ratelimit).toBe(1);

      // 20 parallel queries in 1s; with ratelimit=1 most should be dropped.
      const { output } = await agh.exec([
        'godnsbench', '-a', '127.0.0.1:53', '-q', '{random}.org', '-c', '20', '-p', '20', '-t', '1',
      ]);
      const processedMatch = output.match(/Processed queries:\s*(\d+)/);
      // Fail loudly if godnsbench output isn't what we parse, rather than
      // silently coercing a missing count to 0.
      expect(processedMatch, `godnsbench output not parseable:\n${output}`).toBeTruthy();
      const errors = Number(output.match(/Errors count:\s*(\d+)/)?.[1] ?? 0);
      const processed = Number(processedMatch?.[1] ?? 0);
      expect(errors,
        `Expected >=15 of 20 queries rate-limited, got ${errors} failed (${processed} processed).\n${output}`,
      ).toBeGreaterThanOrEqual(15);
    } finally {
      try {
        await setDnsConfig(agh.baseUrl, { ratelimit: 0 }, api.authHeaders);
      } finally {
        await upstream.stop();
      }
    }
  });
});
