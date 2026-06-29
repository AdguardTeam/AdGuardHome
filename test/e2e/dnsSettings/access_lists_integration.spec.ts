import { test, expect } from '../runtime/fixtures';
import { UPSTREAM_HOST } from '../shared/api/test-fetch.ts';

import { allocateUdpPort, MockDnsServer } from '../shared/dns/mock-dns-server.ts';
import { waitFor } from '../shared/polling/retry.ts';
import { setAccessList } from '../shared/api/adguard-api.ts';
import { setDnsConfig } from '../shared/dns/dns-settings.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

async function startUpstreamWithAnswers(
  entries: Array<{ domain: string; type: string; data: string }>,
): Promise<{ port: number; upstream: MockDnsServer }> {
  const reservedPort = await allocateUdpPort('0.0.0.0');
  const upstream = new MockDnsServer(reservedPort);
  for (const { domain, type, data } of entries) {
    upstream.setAnswers(domain, type, [{ type, data }]);
  }
  await upstream.start();
  return { port: reservedPort.port, upstream };
}

// AGH drops DNS packets for access-list violations (no response) rather than
// returning REFUSED/NXDOMAIN, so a blocked query surfaces as a timeout.
async function waitForPacketDrop(agh: AdGuardContainer, domain: string, type: 'A' | 'AAAA'): Promise<void> {
  await waitFor(async () => {
    const { status } = await agh.dnslookup(domain, { type, timeoutSec: 2 });
    return status === 'NOERROR' ? undefined : true;
  }, { timeoutMs: 15_000, intervalMs: 500 });
}

async function useMockUpstream(agh: AdGuardContainer, api: AdGuardApiClient, port: number): Promise<void> {
  await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${port}`] }, api.authHeaders);
}

test('4115 — Disallowed domains', async ({ agh, api }) => {
  const domain = 'blocked.example';
  const { port, upstream } = await startUpstreamWithAnswers([{ domain, type: 'A', data: '203.0.113.50' }]);
  try {
    await useMockUpstream(agh, api, port);
    const pre = await agh.dnslookup(domain, { type: 'A' });
    expect(pre.status, 'Domain should resolve before blocked_hosts rule').toBe('NOERROR');

    await setAccessList(api, { blocked_hosts: [domain] });
    await waitForPacketDrop(agh, domain, 'A');
  } finally {
    await upstream.stop();
  }
});

test('4113 — Disallowed clients', async ({ agh, api }) => {
  const domain = 'test.example';
  const { port, upstream } = await startUpstreamWithAnswers([{ domain, type: 'A', data: '203.0.113.51' }]);
  try {
    await useMockUpstream(agh, api, port);
    const pre = await agh.dnslookup(domain, { type: 'A' });
    expect(pre.status, 'Domain should resolve before disallowed_clients rule').toBe('NOERROR');

    // Queries come from 127.0.0.1 inside the container.
    await setAccessList(api, { disallowed_clients: ['127.0.0.1'] });
    await waitForPacketDrop(agh, domain, 'A');
  } finally {
    await upstream.stop();
  }
});

test('4112 — Allowed clients: access list drop', async ({ agh, api }) => {
  const domain = 'allowed-test.example';
  const { port, upstream } = await startUpstreamWithAnswers([{ domain, type: 'A', data: '203.0.113.52' }]);
  try {
    await useMockUpstream(agh, api, port);
    const pre = await agh.dnslookup(domain, { type: 'A' });
    expect(pre.status, 'Domain should resolve before allowed_clients rule').toBe('NOERROR');

    await setAccessList(api, { allowed_clients: ['127.0.0.2'] });
    await waitForPacketDrop(agh, domain, 'A');
  } finally {
    await upstream.stop();
  }
});
