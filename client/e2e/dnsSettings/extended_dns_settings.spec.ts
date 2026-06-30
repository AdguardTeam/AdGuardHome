import { test, expect } from '../runtime/fixtures';
import { AdGuardContainer } from '../runtime/adguard-container';
import { UPSTREAM_HOST, authed } from '../shared/api/test-fetch.ts';
import { setDnsConfig, getDnsInfo, clearDnsCache, setAccessConfig } from '../shared/dns/dns-settings.ts';
import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import { createMockUpstream } from '../shared/dns/mock-upstream.ts';
import { ADMIN_PASSWORD_HASH } from '../shared/adguard/admin.ts';

test.describe('Extended DNS Settings Tests (Cases 4086-4116)', () => {
  test('4098 — trusted_proxies X-Forwarded-For', async () => {
    test.setTimeout(120_000);
    // The test only checks the logged client (from X-Forwarded-For), but the DoH
    // query is forwarded upstream, so point AGH at a local mock that answers
    // instantly — a dead/non-routable upstream would hang AGH on each lookup.
    const upstream = await createMockUpstream([{ domain: 'example.org', type: 'A', data: '93.184.216.34' }]);
    const inst = await AdGuardContainer.startCustom({
      config: [
        'http:', '  address: 0.0.0.0:3000', '  trusted_proxies: [0.0.0.0/0, ::/0]',
        'tls:', '  enabled: false', '  allow_unencrypted_doh: true',
        'dns:', '  bind_hosts: [0.0.0.0]', '  port: 53', `  upstream_dns: [${UPSTREAM_HOST}:${upstream.getPort()}]`,
        'querylog:', '  enabled: true', '  interval: 24h',
        'users:', '  - name: admin',
        `    password: ${ADMIN_PASSWORD_HASH}`,
        'schema_version: 32', '',
      ].join('\n'),
    });
    try {
      const api = await inst.api();
      const dnsQueryBase64 = 'AAABAAABAAAAAAAAB2V4YW1wbGUDb3JnAAABAAE';
      // Send DoH from inside the container so the source is the trusted loopback
      // and AGH honors X-Forwarded-For (host->mapped-port traffic is rewritten by docker-proxy).
      await inst.exec(['bash', '-c',
        `curl -s 'http://127.0.0.1:3000/dns-query?dns=${dnsQueryBase64}' -H 'X-Forwarded-For: 1.2.3.4' -H 'Accept: application/dns-message' -o /dev/null`]);
      let entry: { client?: string } | undefined;
      const deadline = Date.now() + 10_000;
      while (Date.now() < deadline) {
        const res = await authed(api)(`${inst.baseUrl}/control/querylog?search=example.org`);
        if (!res.ok) { await new Promise((r) => setTimeout(r, 300)); continue; }
        const data = await res.json() as { data?: Array<{ client?: string }> };
        if (data.data?.length) { entry = data.data[0]; break; }
        await new Promise((r) => setTimeout(r, 300));
      }
      expect(entry, 'No query-log entry for example.org within 10s').toBeTruthy();
      expect(entry.client, `Expected client 1.2.3.4 from X-Forwarded-For, got ${entry.client}`).toBe('1.2.3.4');
    } finally {
      await inst.stop().catch(() => {});
      await upstream.stop();
    }
  });

  // @egress — DNSSEC validation needs a real validating resolver (8.8.8.8) and
  // the real dnssec-failed.org test domain; this cannot be mocked.
  test('4104 — Enable DNSSEC', async ({ agh, api }) => {
    const mockUpstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
    try {
      await mockUpstream.start();

      await setDnsConfig(agh.baseUrl, { dnssec_enabled: true, upstream_dns: ['8.8.8.8'] }, api.authHeaders);
      expect((await getDnsInfo(agh.baseUrl, api.authHeaders)).dnssec_enabled).toBe(true);

      await clearDnsCache(agh.baseUrl, api.authHeaders);
      const blocked = await agh.dnslookup('dnssec-failed.org', { type: 'A' });
      expect(blocked.status).toBe('SERVFAIL');

      await setDnsConfig(
        agh.baseUrl,
        { dnssec_enabled: false, upstream_dns: [`${UPSTREAM_HOST}:${mockUpstream.getPort()}`] },
        api.authHeaders,
      );
      expect((await getDnsInfo(agh.baseUrl, api.authHeaders)).dnssec_enabled).toBe(false);

      await clearDnsCache(agh.baseUrl, api.authHeaders);
      const resolved = await agh.dnslookup('dnssec-failed.org', { type: 'A' });
      expect(resolved.status).not.toBe('SERVFAIL');
    } finally {
      await setDnsConfig(agh.baseUrl, { dnssec_enabled: false }, api.authHeaders).catch(() => {});
      await mockUpstream.stop();
    }
  });
});
