import { test, expect } from '../runtime/fixtures';
import { authed, UPSTREAM_HOST, ctxOf } from '../shared/api/test-fetch.ts';

import { waitFor, sleep } from '../shared/polling/retry.ts';
import { clearQueryLog, getQueryLog, updateQueryLogConfig, type QueryLogEntry } from '../shared/adguard/querylog.ts';
import { clearStats, getStats, updateStatsConfig } from '../shared/adguard/stats.ts';
import { setCustomRules } from '../shared/adguard/filtering.ts';
import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

const DEFAULT_QUERY_LOG_CONFIG = { enabled: true, interval: 7_776_000_000, ignored: [], ignored_enabled: false, anonymize_client_ip: false };
const DEFAULT_STATS_CONFIG = { enabled: true, interval: 86_400_000, ignored: [], ignored_enabled: false };

function normalizeDnsName(value?: string): string | undefined { return value?.replace(/\.$/, '').toLowerCase(); }
function matchesDomain(record: QueryLogEntry, domain: string): boolean {
  return normalizeDnsName(record.question?.host || record.question?.name) === normalizeDnsName(domain);
}

const queryDomain = (agh: AdGuardContainer, domain: string, type: 'A' | 'AAAA' = 'A') => agh.dnslookup(domain, { type });

async function waitForQueryLogRecord(agh: AdGuardContainer, api: AdGuardApiClient, domain: string): Promise<QueryLogEntry> {
  return waitFor(async () => {
    const response = await getQueryLog(ctxOf(agh, api), { search: domain, limit: 100 });
    return response.data.find((record) => matchesDomain(record, domain));
  }, { timeoutMs: 10_000, intervalMs: 500 });
}

async function assertQueryLogDoesNotContain(agh: AdGuardContainer, api: AdGuardApiClient, domain: string): Promise<void> {
  await sleep(1_000);
  const response = await getQueryLog(ctxOf(agh, api), { search: domain, limit: 100 });
  expect(response.data.some((record) => matchesDomain(record, domain)), `Expected query log to ignore ${domain}`).toBe(false);
}

function rankedKeys(items: Array<Record<string, number>>): string[] { return items.flatMap((item) => Object.keys(item)); }

async function waitForStats(agh: AdGuardContainer, api: AdGuardApiClient) {
  return waitFor(async () => {
    const stats = await getStats(ctxOf(agh, api));
    return stats.num_dns_queries > 0 ? stats : undefined;
  }, { timeoutMs: 10_000, intervalMs: 500 });
}

test('4063/4064/4065/4066 — Query log & statistics enable/ignore', async ({ agh, api }) => {
  test.setTimeout(120_000);
  const mock = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  await mock.start();

  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${mock.getPort()}`] }, api.authHeaders);
    const ctx = ctxOf(agh, api);
    const blockedDomain = `blocked-4063.test`;
    const restoredStatsDomain = `restored-stats.test`;
    const baselineStatsDomain = `baseline-stats.test`;
    const disabledStatsDomain = `disabled-stats.test`;
    await setCustomRules(ctx, [`||${blockedDomain}^`, `||${restoredStatsDomain}^`, `||${baselineStatsDomain}^`, `||${disabledStatsDomain}^`]);
    await updateQueryLogConfig(ctx, DEFAULT_QUERY_LOG_CONFIG);
    await updateStatsConfig(ctx, DEFAULT_STATS_CONFIG);
    await clearQueryLog(ctx);
    await clearStats(ctx);

    await queryDomain(agh, blockedDomain);
    await waitForQueryLogRecord(agh, api, blockedDomain);
    const changedStats = await waitFor(async () => {
      const stats = await getStats(ctx);
      return stats.num_dns_queries > 0 && stats.num_blocked_filtering > 0 ? stats : undefined;
    }, { timeoutMs: 10_000, intervalMs: 500 });
    expect(changedStats.num_dns_queries > 0).toBeTruthy();
    expect(changedStats.num_blocked_filtering > 0).toBeTruthy();

    await clearQueryLog(ctx);
    await updateQueryLogConfig(ctx, { ...DEFAULT_QUERY_LOG_CONFIG, ignored_enabled: true, ignored: ['youtube.com'] });
    await queryDomain(agh, 'youtube.com');
    await queryDomain(agh, 'mail.ru');
    await waitForQueryLogRecord(agh, api, 'mail.ru');
    await assertQueryLogDoesNotContain(agh, api, 'youtube.com');

    await updateQueryLogConfig(ctx, { ...DEFAULT_QUERY_LOG_CONFIG, ignored_enabled: true, ignored: ['93.184.216.34'] });
    await clearQueryLog(ctx);
    await queryDomain(agh, '93.184.216.34');
    await assertQueryLogDoesNotContain(agh, api, '93.184.216.34');

    await updateQueryLogConfig(ctx, DEFAULT_QUERY_LOG_CONFIG);
    await queryDomain(agh, '93.184.216.34');
    await waitForQueryLogRecord(agh, api, '93.184.216.34');

    await clearStats(ctx);
    await updateStatsConfig(ctx, { ...DEFAULT_STATS_CONFIG, ignored_enabled: true, ignored: ['youtube.com'] });
    await queryDomain(agh, 'youtube.com');
    await queryDomain(agh, 'mail.ru');
    const mailOnlyStats = await waitForStats(agh, api);
    const topQueriedDomains = rankedKeys(mailOnlyStats.top_queried_domains);
    expect(topQueriedDomains.includes('mail.ru'), 'Expected mail.ru in statistics').toBeTruthy();
    expect(topQueriedDomains.includes('youtube.com'), 'Expected youtube.com ignored in statistics').toBe(false);

    await clearStats(ctx);
    await updateStatsConfig(ctx, { ...DEFAULT_STATS_CONFIG, ignored_enabled: true, ignored: ['93.184.216.34'] });
    await queryDomain(agh, '93.184.216.34');
    await sleep(1_000);
    expect((await getStats(ctx)).num_dns_queries, 'Expected ignored IP-like queries skipped in statistics').toBe(0);

    await updateStatsConfig(ctx, DEFAULT_STATS_CONFIG);
    await queryDomain(agh, restoredStatsDomain);
    expect((await waitForStats(agh, api)).num_dns_queries > 0, 'Expected statistics to resume').toBeTruthy();

    await clearStats(ctx);
    await queryDomain(agh, baselineStatsDomain);
    expect((await waitForStats(agh, api)).num_dns_queries > 0, 'Expected baseline statistics').toBeTruthy();

    await updateStatsConfig(ctx, { ...DEFAULT_STATS_CONFIG, enabled: false });
    const beforeDisabled = await getStats(ctx);
    await queryDomain(agh, disabledStatsDomain);
    await sleep(1_000);
    expect((await getStats(ctx)).num_dns_queries, 'Expected disabled statistics to stop collecting').toBe(beforeDisabled.num_dns_queries);

    await clearStats(ctx);
    const clearedStats = await getStats(ctx);
    expect(clearedStats.num_dns_queries, 'Expected cleared statistics empty').toBe(0);
    expect(clearedStats.top_queried_domains, 'Expected no top domains after clear').toEqual([]);
  } finally { await mock.stop(); }
});
