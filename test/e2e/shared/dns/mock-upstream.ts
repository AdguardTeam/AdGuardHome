import { allocateUdpPort, MockDnsServer } from './mock-dns-server.ts';
import { setDnsConfig } from './dns-settings.ts';
import { UPSTREAM_HOST } from '../api/test-fetch.ts';
import type { AdGuardContainer } from '../../runtime/adguard-container';
import type { AdGuardApiClient } from '../api/adguard-api';

/** One canned answer the mock upstream should return for `domain`/`type`. */
export interface MockUpstreamEntry {
  domain: string;
  type: string;
  data: unknown;
}

/**
 * Start a MockDnsServer seeded with `entries`. Callers that want AGH to forward
 * to it should use `startMockUpstream` (which also points `upstream_dns` at it).
 */
export async function createMockUpstream(entries: MockUpstreamEntry[]): Promise<MockDnsServer> {
  const upstream = new MockDnsServer(await allocateUdpPort('0.0.0.0'));
  for (const { domain, type, data } of entries) upstream.setAnswers(domain, type, [{ type, data }]);
  await upstream.start();
  return upstream;
}

/**
 * Start a mock upstream seeded with `entries` and point AGH's `upstream_dns` at
 * it. Returns the server so the test can stop it (and tweak answers) afterwards.
 */
export async function startMockUpstream(
  agh: AdGuardContainer,
  api: AdGuardApiClient,
  entries: MockUpstreamEntry[],
): Promise<MockDnsServer> {
  const upstream = await createMockUpstream(entries);
  try {
    await setDnsConfig(agh.baseUrl, { upstream_dns: [`${UPSTREAM_HOST}:${upstream.getPort()}`] }, api.authHeaders);
  } catch (err) {
    await upstream.stop();
    throw err;
  }
  return upstream;
}
