import { test, expect } from '../runtime/fixtures';
import { UPSTREAM_HOST } from '../shared/api/test-fetch.ts';
import { MockDnsServer, allocateUdpPort } from '../shared/dns/mock-dns-server.ts';
import { setDnsConfig, getDnsInfo } from '../shared/dns/dns-settings.ts';

test.describe('Case 4099: SOA and NS queries served by private rDNS', () => {
  test('4099 — Private rDNS SOA/NS queries', async ({ agh, api }) => {
    const reverseZone = '1.0.168.192.in-addr.arpa';
    const mock = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
    await mock.start();
    try {
      const upstream = `${UPSTREAM_HOST}:${mock.getPort()}`;
      await setDnsConfig(agh.baseUrl, {
        local_ptr_upstreams: [upstream],
        use_private_ptr_resolvers: true,
      }, api.authHeaders);

      const info = await getDnsInfo(agh.baseUrl, api.authHeaders);
      expect(info.use_private_ptr_resolvers).toBe(true);
      expect(info.local_ptr_upstreams).toEqual([upstream]);

      const soa = await agh.dnslookup(reverseZone, { type: 'SOA' });
      const soaData = soa.records.filter((r) => r.type === 'SOA').map((r) => r.data);
      expect(
        soaData.some((d) => d.includes('ns1.example.com.')),
        `Expected SOA record containing ns1.example.com., got: ${soaData.join(', ')}`,
      ).toBeTruthy();

      const ns = await agh.dnslookup(reverseZone, { type: 'NS' });
      const nsData = ns.records.filter((r) => r.type === 'NS').map((r) => r.data);
      expect(
        nsData.some((d) => d.includes('ns1.example.com.')),
        `Expected NS record ns1.example.com., got: ${nsData.join(', ')}`,
      ).toBeTruthy();
    } finally {
      await mock.stop();
    }
  });
});
