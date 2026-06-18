import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { UPSTREAM_HOST } from '../shared/api/test-fetch.ts';
import { MockDnsServer, allocateUdpPort } from './MockDnsServer.ts';
import { setDnsConfig, getDnsInfo } from './dns_settings.ts';

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
      assert.equal(info.use_private_ptr_resolvers, true);
      assert.deepEqual(info.local_ptr_upstreams, [upstream]);

      const soa = await agh.dnslookup(reverseZone, { type: 'SOA' });
      const soaData = soa.records.filter((r) => r.type === 'SOA').map((r) => r.data);
      assert.ok(
        soaData.some((d) => d.includes('ns1.example.com.')),
        `Expected SOA record containing ns1.example.com., got: ${soaData.join(', ')}`,
      );

      const ns = await agh.dnslookup(reverseZone, { type: 'NS' });
      const nsData = ns.records.filter((r) => r.type === 'NS').map((r) => r.data);
      assert.ok(
        nsData.some((d) => d.includes('ns1.example.com.')),
        `Expected NS record ns1.example.com., got: ${nsData.join(', ')}`,
      );
    } finally {
      await mock.stop();
    }
  });
});
