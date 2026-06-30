import { test, expect } from '../runtime/fixtures';
import { authed } from '../shared/api/test-fetch.ts';

import { waitForAnswers } from '../shared/dns/dns-test-helpers.ts';
import { startMockUpstream } from '../shared/dns/mock-upstream.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

const YOUTUBE_TEST_DOMAIN = 'youtube.com';
const YOUTUBE_SERVICE_ID = 'youtube';
const REAL_IP = '203.0.113.42';
const BLOCKED_IP = '0.0.0.0';
const CLIENT_NAME = 'blocked-client';

function useYoutubeUpstream(agh: AdGuardContainer, api: AdGuardApiClient) {
  return startMockUpstream(agh, api, [{ domain: YOUTUBE_TEST_DOMAIN, type: 'A', data: REAL_IP }]);
}

async function setGlobalBlockedServices(agh: AdGuardContainer, api: AdGuardApiClient, ids: string[]): Promise<void> {
  const res = await authed(api)(`${agh.baseUrl}/control/blocked_services/update`, {
    method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ids }),
  });
  expect(res.ok, `Failed to update blocked services: ${res.status}`).toBeTruthy();
}

const waitForDnsAnswer = (agh: AdGuardContainer, domain: string, predicate: (answers: string[]) => boolean) =>
  waitForAnswers(agh, domain, 'A', predicate);

test('4156 — Blocked services', async ({ agh, api }) => {
  const upstream = await useYoutubeUpstream(agh, api);
  try {
    expect((await waitForDnsAnswer(agh, YOUTUBE_TEST_DOMAIN, (a) => a.includes(REAL_IP))).includes(REAL_IP)).toBeTruthy();
    await setGlobalBlockedServices(agh, api, [YOUTUBE_SERVICE_ID]);
    expect((await waitForDnsAnswer(agh, YOUTUBE_TEST_DOMAIN, (a) => a.includes(BLOCKED_IP))).includes(BLOCKED_IP)).toBeTruthy();
    await setGlobalBlockedServices(agh, api, []);
    expect((await waitForDnsAnswer(agh, YOUTUBE_TEST_DOMAIN, (a) => a.includes(REAL_IP))).includes(REAL_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});
