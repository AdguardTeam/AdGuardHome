import { test, expect } from '../runtime/fixtures';
import { UPSTREAM_HOST } from '../shared/api/test-fetch.ts';
import { MockDnsServer, allocateUdpPort } from '../shared/dns/mock-dns-server.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';

// Fire many unique-domain A queries from inside the container in a single exec
// (fast), so each goes upstream rather than being served from cache.
async function floodQueries(agh: AdGuardContainer, prefix: string, count: number): Promise<void> {
  if (!Number.isInteger(count) || count < 0 || !/^[\w-]+$/.test(prefix)) {
    throw new Error(`Unsafe floodQueries args: prefix=${prefix} count=${count}`);
  }
  await agh.exec(['sh', '-c',
    `for i in $(seq 1 ${count}); do dnslookup ${prefix}-$i.com 127.0.0.1:53 >/dev/null 2>&1; done`]);
}

test('4086 — Upstream DNS mode', async ({ agh, api }) => {
  test.setTimeout(120_000);
  const mocks = await Promise.all([0, 1, 2].map(async () => {
    const m = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
    await m.start();
    return m;
  }));
  const upstreams = mocks.map((m) => `${UPSTREAM_HOST}:${m.getPort()}`);

  const baseConfig = {
    fallback_dns: [] as string[],
    bootstrap_dns: ['1.1.1.1'],
    resolve_clients: false,
    local_ptr_upstreams: [] as string[],
    use_private_ptr_resolvers: false,
    upstream_timeout: 10,
  };

  try {
    await test.step('Load Balancing', async () => {
      await setDnsConfig(agh.baseUrl, { ...baseConfig, upstream_dns: upstreams, upstream_mode: 'load_balance' }, api.authHeaders);
      mocks.forEach((m) => m.clearQueries());
      await floodQueries(agh, 'lb', 50);
      const queries = mocks.map((m) => m.getQueries().length);
      expect(queries.every((q) => q > 0), `All servers should receive queries. Got: ${queries}`).toBeTruthy();
    });

    await test.step('Parallel Requests', async () => {
      await setDnsConfig(agh.baseUrl, { ...baseConfig, upstream_dns: upstreams, upstream_mode: 'parallel' }, api.authHeaders);
      mocks.forEach((m) => m.clearQueries());
      await floodQueries(agh, 'parallel', 10);
      const queries = mocks.map((m) => m.getQueries().length);
      // Parallel fans every query out to all upstreams.
      expect(queries.every((q) => q >= 8), `All servers should receive almost all queries. Got: ${queries}`).toBeTruthy();
      // Timing assertions are omitted: per-query `dnslookup exec` overhead makes
      // sub-second wall-clock checks unreliable in the containerized runtime.
    });

    await test.step('Fastest IP', async () => {
      mocks[0].setDelay(0);
      mocks[1].setDelay(1000);
      mocks[2].setDelay(1000);
      await setDnsConfig(agh.baseUrl, { ...baseConfig, upstream_dns: upstreams, upstream_mode: 'fastest_addr' }, api.authHeaders);
      mocks.forEach((m) => m.clearQueries());
      await floodQueries(agh, 'fastest', 30);
      const queries = mocks.map((m) => m.getQueries().length);
      expect(queries.every((q) => q > 0), `All servers should receive queries. Got: ${queries}`).toBeTruthy();
    });
  } finally {
    for (const m of mocks) await m.stop();
  }
});
