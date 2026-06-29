import { test, expect } from '../runtime/fixtures';
import { authed } from '../shared/api/test-fetch.ts';

import { addBlockList, removeBlockList, updateBlockList, type BlockList } from './blocklists.ts';
import { waitFor } from '../shared/polling/retry.ts';
import { waitForAnswers } from '../shared/dns/dns-test-helpers.ts';
import { startMockUpstream } from '../shared/dns/mock-upstream.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

interface FilteringListEntry { url: string; name: string; rules_count?: number; enabled?: boolean; last_updated?: string; }

async function getFilteringStatus(agh: AdGuardContainer, api: AdGuardApiClient): Promise<{ filters?: FilteringListEntry[] }> {
  const response = await authed(api)(`${agh.baseUrl}/control/filtering/status`);
  expect(response.ok, 'Expected filtering status request to succeed').toBe(true);
  return response.json() as Promise<{ filters?: FilteringListEntry[] }>;
}

async function waitForFilter(agh: AdGuardContainer, api: AdGuardApiClient, predicate: (f: FilteringListEntry) => boolean): Promise<FilteringListEntry> {
  return waitFor(async () => (await getFilteringStatus(agh, api)).filters?.find(predicate), { timeoutMs: 10_000, intervalMs: 500 });
}

const waitForAnswer = (agh: AdGuardContainer, domain: string, match: (answers: string[]) => boolean) =>
  waitForAnswers(agh, domain, 'A', match);
const waitForBlockedAnswer = (agh: AdGuardContainer, domain: string) => waitForAnswer(agh, domain, (a) => a.includes('0.0.0.0'));
const waitForResolvedAnswer = (agh: AdGuardContainer, domain: string, ip: string) => waitForAnswer(agh, domain, (a) => a.includes(ip));

async function refreshFilters(agh: AdGuardContainer, api: AdGuardApiClient): Promise<number> {
  const response = await authed(api)(`${agh.baseUrl}/control/filtering/refresh`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}),
  });
  expect(response.ok, 'Expected filter refresh request to succeed').toBe(true);
  const body = await response.json() as { updated?: unknown };
  return (typeof body.updated === 'number' ? body.updated : 0);
}

function isLoadedBlocklistFilter(filter: FilteringListEntry, expectedUrl: string): boolean {
  return filter.url === expectedUrl && filter.enabled === true && (filter.rules_count ?? 0) > 0;
}

function useMock(agh: AdGuardContainer, api: AdGuardApiClient, entries: Array<{ domain: string; data: string }>) {
  return startMockUpstream(agh, api, entries.map(({ domain, data }) => ({ domain, type: 'A', data })));
}

test('4164 — Add custom blocklist', async ({ agh, api }) => {
  const url = await agh.serveRules('custom-blocklist.txt', '# Title: Example Custom Blocklist\n||blocked-custom.example^');
  const blocklist: BlockList = { name: 'Example Custom Blocklist', url, whitelist: false };
  await addBlockList(agh.baseUrl, blocklist, authed(api));

  const storedFilter = await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, blocklist.url));
  expect(storedFilter.name).toBe(blocklist.name);
  expect((storedFilter.rules_count ?? 0) >= 1, 'Expected custom blocklist to load at least one rule').toBeTruthy();

  const answers = await waitForBlockedAnswer(agh, 'blocked-custom.example');
  expect(answers.includes('0.0.0.0'), `Expected blocked-custom.example blocked, got ${JSON.stringify(answers)}`).toBeTruthy();
});

test('4165 — Add filter without a name', async ({ agh, api }) => {
  const url = await agh.serveRules('unnamed-blocklist.txt', '! Title: Auto Named Filter\n||auto-named-blocked.example^');
  const blocklist: BlockList = { name: '', url, whitelist: false };
  await addBlockList(agh.baseUrl, blocklist, authed(api));

  const storedFilter = await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, blocklist.url));
  expect(storedFilter.name).toBe('Auto Named Filter');

  const answers = await waitForBlockedAnswer(agh, 'auto-named-blocked.example');
  expect(answers.includes('0.0.0.0'), `Expected auto-named-blocked.example blocked, got ${JSON.stringify(answers)}`).toBeTruthy();
});

test('4078/4161 — Remove blocklist', async ({ agh, api }) => {
  const REAL_IP = '203.0.113.70';
  const upstream = await useMock(agh, api, [{ domain: 'remove-blocklist.example', data: REAL_IP }]);
  try {
    const url = await agh.serveRules('remove-test-blocklist.txt', '||remove-blocklist.example^\n');
    const blocklist: BlockList = { name: 'Remove Test Blocklist', url, whitelist: false };
    await addBlockList(agh.baseUrl, blocklist, authed(api));
    await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, blocklist.url));
    expect((await waitForBlockedAnswer(agh, 'remove-blocklist.example')).includes('0.0.0.0')).toBeTruthy();

    await removeBlockList(agh.baseUrl, blocklist, authed(api));
    expect((await waitForResolvedAnswer(agh, 'remove-blocklist.example', REAL_IP)).includes(REAL_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4166 — Add blocklist with duplicate URL', async ({ agh, api }) => {
  const upstream = await useMock(agh, api, [{ domain: 'dup-blocklist-test.example', data: '203.0.113.72' }]);
  try {
    const url = await agh.serveRules('dup-blocklist.txt', '||dup-blocklist-test.example^\n');
    const blocklist: BlockList = { name: 'Dup Blocklist', url, whitelist: false };
    await addBlockList(agh.baseUrl, blocklist, authed(api));
    await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, blocklist.url));
    await expect(() => addBlockList(agh.baseUrl, blocklist, authed(api))).rejects.toThrow(/Failed to add blocklist: 400/);
  } finally { await upstream.stop(); }
});

test('4167 — DNS blocklists general', async ({ agh, api }) => {
  test.setTimeout(90_000);
  const REAL_IP = '203.0.113.73';
  const firstDomain = 'general-blocklist-one.example';
  const secondDomain = 'general-blocklist-two.example';
  const upstream = await useMock(agh, api, [{ domain: firstDomain, data: REAL_IP }, { domain: secondDomain, data: REAL_IP }]);
  try {
    const firstUrl = await agh.serveRules('general-blocklist-one.txt', `||${firstDomain}^\n`);
    const secondUrl = await agh.serveRules('general-blocklist-two.txt', `||${secondDomain}^\n`);
    const first: BlockList = { name: 'General Blocklist One', url: firstUrl, whitelist: false };
    const second: BlockList = { name: 'General Blocklist Two', url: secondUrl, whitelist: false };
    await addBlockList(agh.baseUrl, first, authed(api));
    await addBlockList(agh.baseUrl, second, authed(api));
    await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, first.url));
    await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, second.url));

    expect((await waitForBlockedAnswer(agh, firstDomain)).includes('0.0.0.0')).toBeTruthy();
    expect((await waitForBlockedAnswer(agh, secondDomain)).includes('0.0.0.0')).toBeTruthy();

    // refreshFilters asserts the request succeeded; verify blocking survives it.
    await refreshFilters(agh, api);
    expect((await waitForBlockedAnswer(agh, firstDomain)).includes('0.0.0.0')).toBeTruthy();
    expect((await waitForBlockedAnswer(agh, secondDomain)).includes('0.0.0.0')).toBeTruthy();

    await updateBlockList(agh.baseUrl, first, { ...first, enabled: false }, authed(api));
    await updateBlockList(agh.baseUrl, second, { ...second, enabled: false }, authed(api));
    expect((await waitForResolvedAnswer(agh, firstDomain, REAL_IP)).includes(REAL_IP)).toBeTruthy();
    expect((await waitForResolvedAnswer(agh, secondDomain, REAL_IP)).includes(REAL_IP)).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4168 — Blocklist updates: no-op refresh timestamp', async ({ agh, api }) => {
  test.setTimeout(90_000);
  const upstream = await useMock(agh, api, []);
  try {
    const url = await agh.serveRules('stable-blocklist.txt', '||stable-blocked.example^\n');
    const blocklist: BlockList = { name: 'Stable Blocklist', url, whitelist: false };
    await addBlockList(agh.baseUrl, blocklist, authed(api));
    const initialFilter = await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, blocklist.url) && typeof f.last_updated === 'string');
    expect(initialFilter.last_updated).toBeTruthy();

    await new Promise((r) => setTimeout(r, 1_100));
    expect(await refreshFilters(agh, api), 'Expected no updated filters when remote rules are unchanged').toBe(0);

    const refreshedFilter = await waitForFilter(agh, api, (f) => f.url === blocklist.url && typeof f.last_updated === 'string');
    expect(refreshedFilter.last_updated).toBeTruthy();
    expect(Date.parse(refreshedFilter.last_updated!) > Date.parse(initialFilter.last_updated!),
      `Expected last_updated to move forward, got ${initialFilter.last_updated} -> ${refreshedFilter.last_updated}`).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4168 — Blocklist updates: apply updated rules', async ({ agh, api }) => {
  test.setTimeout(90_000);
  const REAL_IP = '203.0.113.74';
  const oldDomain = 'before-refresh.example';
  const newDomain = 'after-refresh.example';
  const upstream = await useMock(agh, api, [{ domain: oldDomain, data: REAL_IP }, { domain: newDomain, data: REAL_IP }]);
  try {
    const url = await agh.serveRules('refreshable-blocklist.txt', `||${oldDomain}^\n`);
    const blocklist: BlockList = { name: 'Refreshable Blocklist', url, whitelist: false };
    await addBlockList(agh.baseUrl, blocklist, authed(api));
    await waitForFilter(agh, api, (f) => isLoadedBlocklistFilter(f, blocklist.url));

    expect((await waitForBlockedAnswer(agh, oldDomain)).includes('0.0.0.0')).toBeTruthy();
    expect((await waitForResolvedAnswer(agh, newDomain, REAL_IP)).includes(REAL_IP)).toBeTruthy();

    // Update the served content, then refresh.
    await agh.serveRules('refreshable-blocklist.txt', `||${newDomain}^\n`);
    expect((await refreshFilters(agh, api)) >= 1, 'Expected at least one updated filter').toBeTruthy();

    expect((await waitForResolvedAnswer(agh, oldDomain, REAL_IP)).includes(REAL_IP)).toBeTruthy();
    expect((await waitForBlockedAnswer(agh, newDomain)).includes('0.0.0.0')).toBeTruthy();
  } finally { await upstream.stop(); }
});
