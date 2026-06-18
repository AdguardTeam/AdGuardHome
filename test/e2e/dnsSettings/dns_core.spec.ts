import { test, expect } from '../runtime/fixtures';
import { clearDnsCache, setDnsConfig, getDnsInfo } from './dns_settings.ts';
import { waitFor } from '../shared/polling/retry.ts';

const blockedDomain = `blocked-mode-static.test`;

test.describe('DNS Core Configuration Tests (Cases 4102, 4105, 4106)', () => {
  // Case #4106: Blocking modes
  test('4106 — Blocking mode', async ({ agh, api }) => {
    // Ensure valid upstreams, then block the test domain via a custom rule.
    await setDnsConfig(agh.baseUrl, { upstream_dns: ['8.8.8.8', '1.1.1.1'] }, api.authHeaders);
    await fetch(`${agh.baseUrl}/control/filtering/set_rules`, {
      method: 'POST',
      headers: { ...(api.authHeaders as Record<string, string>), 'Content-Type': 'application/json' },
      body: JSON.stringify({ rules: [`||${blockedDomain}^`] }),
    });

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
  });

  // Case #4102: Rate Limit
  test('4102 — Rate limit', async ({ agh, api }) => {
    await setDnsConfig(agh.baseUrl, { ratelimit: 1, upstream_dns: ['8.8.8.8'] }, api.authHeaders);
    const info = await getDnsInfo(agh.baseUrl, api.authHeaders);
    expect(info.ratelimit).toBe(1);

    try {
      // 20 parallel queries in 1s; with ratelimit=1 most should be dropped.
      const { output } = await agh.exec([
        'godnsbench', '-a', '127.0.0.1:53', '-q', '{random}.org', '-c', '20', '-p', '20', '-t', '1',
      ]);
      const errors = Number(output.match(/Errors count:\s*(\d+)/)?.[1] ?? 0);
      const processed = Number(output.match(/Processed queries:\s*(\d+)/)?.[1] ?? 0);
      expect(errors,
        `Expected >=15 of 20 queries rate-limited, got ${errors} failed (${processed} processed).\n${output}`,
      ).toBeGreaterThanOrEqual(15);
    } finally {
      await setDnsConfig(agh.baseUrl, { ratelimit: 0 }, api.authHeaders);
    }
  });
});
