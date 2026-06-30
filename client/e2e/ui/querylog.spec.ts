import { type Locator, type Page } from '@playwright/test';
import { test, expect } from '../runtime/fixtures';
import { resolveAnswers as resolveDnsAnswers } from '../shared/dns/dns-test-helpers.ts';

import {
  addClient, clearQueryLog, getClients, getQueryLog, setAccessList, setCustomRules, updateProfile,
  type AdGuardApiClient,
} from '../shared/api/adguard-api.ts';
import { getDnsInfo, setDnsConfig } from '../shared/dns/dns-settings.ts';
import {
  clientIdentifierInput, clientNameInput, customRulesTextarea, loginToAdGuardUi,
  persistentClientRow, queryLogRow, queryLogSearchInput, saveClientForm,
} from '../shared/ui/adguard-playwright.ts';
import { waitForQueryLogRecord as waitForSharedQueryLogRecord } from '../shared/querylog/wait-for-querylog.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';

test.describe.configure({ mode: 'serial' });

function escapeRegExp(value: string): string { return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }
function countOccurrences(text: string, value: string): number { return (text.match(new RegExp(escapeRegExp(value), 'g')) ?? []).length; }
async function visibleDomainCount(page: Page, domain: string): Promise<number> { return countOccurrences(await page.locator('body').innerText(), domain); }
function stripWrappedQuotes(value: string): string { return value.replace(/^"(.*)"$/, '$1'); }

async function waitForPaginationResponse(page: Page, matcher: (url: URL) => boolean): Promise<{ data?: Array<{ question?: { name?: string } }>; oldest?: string }> {
  const response = await page.waitForResponse((c) =>
    c.url().includes('/control/querylog') && c.request().method() === 'GET' && matcher(new URL(c.url())));
  return response.json();
}
function waitForQueryLogResponse(page: Page, matcher: (url: URL) => boolean = () => true): Promise<import('@playwright/test').Response> {
  return page.waitForResponse((c) =>
    c.url().includes('/control/querylog') && c.request().method() === 'GET' && matcher(new URL(c.url())));
}
async function waitForQueryLogView(page: Page, predicate: (text: string) => boolean): Promise<void> {
  await expect.poll(async () => predicate(await page.locator('body').innerText())).toBe(true);
}

const sendQuery = (agh: AdGuardContainer, domain: string, type: 'A' | 'AAAA' = 'A') => agh.dnslookup(domain, { type });

async function seedQueries(agh: AdGuardContainer, domains: Array<{ domain: string; count: number }>): Promise<void> {
  for (const entry of domains) for (let i = 0; i < entry.count; i += 1) await sendQuery(agh, entry.domain);
}

async function expectDomainBlocked(agh: AdGuardContainer, domain: string): Promise<void> {
  await expect.poll(async () => (await resolveDnsAnswers(agh, domain, 'A')).includes('0.0.0.0')).toBe(true);
}

async function waitForCategorizedQuery(agh: AdGuardContainer, client: AdGuardApiClient, domain: string, responseStatus: 'blocked' | 'whitelisted'): Promise<void> {
  await expect.poll(async () => {
    await sendQuery(agh, domain);
    return (await getQueryLog(client, { search: domain, response_status: responseStatus, limit: 20 })).data.length;
  }).toBeGreaterThan(0);
}

async function waitForQueryLogRecord(client: AdGuardApiClient, domain: string) {
  return waitForSharedQueryLogRecord(client.baseUrl, domain, {
    fetchImpl: client.fetch, timeoutMs: 10_000, intervalMs: 500,
    path: `/control/querylog?response_status=all&search=${encodeURIComponent(domain)}&older_than=&limit=100`,
  });
}

async function openQueryLog(page: Page): Promise<void> {
  await loginToAdGuardUi(page);
  await page.getByRole('link', { name: 'Query Log' }).click();
  await expect(page).toHaveURL(/#logs/);
  await expect(page.locator('body')).toContainText('Query Log');
}
const queryLogRefreshButton = (page: Page) => page.locator('button[title="Refresh"]').first();

async function readTooltipText(page: Page, trigger: Locator, expectedPattern?: RegExp): Promise<string> {
  await page.mouse.move(0, 0);
  await page.waitForTimeout(250);
  const previousVisibleTexts = (await page.locator('.tooltip-custom__container:visible').allTextContents()).map((t) => t.trim()).filter((t) => t.length > 0);
  await trigger.hover();
  const tooltipId = await trigger.getAttribute('aria-describedby') ?? await trigger.getAttribute('aria-controls') ?? await trigger.getAttribute('data-tooltip-id');
  if (tooltipId) {
    const tooltip = page.locator(`.tooltip-custom__container#${tooltipId}, #${tooltipId}.tooltip-custom__container`).first();
    await tooltip.waitFor({ state: 'visible' });
    const text = ((await tooltip.textContent()) ?? '').trim();
    if (expectedPattern) expect(text).toMatch(expectedPattern);
    return text;
  }
  const matcher = expectedPattern ? new RegExp(expectedPattern.source, expectedPattern.flags.replace(/g/g, '')) : null;
  let tooltipText = '';
  await expect.poll(async () => {
    const visibleTexts = (await page.locator('.tooltip-custom__container:visible').allTextContents()).map((t) => t.trim()).filter((t) => t.length > 0);
    if (matcher) { tooltipText = visibleTexts.find((t) => matcher.test(t)) ?? ''; return tooltipText; }
    tooltipText = visibleTexts.find((t) => !previousVisibleTexts.includes(t)) ?? visibleTexts.at(-1) ?? '';
    return tooltipText;
  }).not.toEqual('');
  return tooltipText;
}

async function clickQueryLogAction(page: Page, domain: string, actionLabel: 'Add as persistent client' | 'Block' | 'Block for this client only' | 'Disallow this client'): Promise<void> {
  const row = queryLogRow(page, domain);
  await expect(row).toBeVisible();
  await row.locator('.button-action__container button').click();
  const action = page.locator('.button-action--arrow-option').filter({ hasText: new RegExp(`^\\s*${escapeRegExp(actionLabel)}\\s*$`) }).first();
  await expect(action).toBeVisible();
  await action.click();
}

async function openCustomRules(page: Page): Promise<void> {
  await page.goto('/#custom_rules');
  await expect(page).toHaveURL(/#custom_rules/);
  await expect(page.locator('body')).toContainText('Custom filtering rules');
}

test.beforeEach(async ({ api }) => {
  await updateProfile(api, { language: 'en' });
});

test('4182 — Query log is shown', async ({ page, agh }) => {
  await seedQueries(agh, [{ domain: 'example.com', count: 3 }, { domain: 'youtube.com', count: 3 }]);
  await openQueryLog(page);
  await expect(page.locator('body')).toContainText('example.com');
  await expect(page.locator('body')).toContainText('youtube.com');
});

test('4183 — Query log filtered by domain', async ({ page, agh }) => {
  await seedQueries(agh, [{ domain: 'example.com', count: 3 }, { domain: 'youtube.com', count: 3 }]);
  await openQueryLog(page);
  const filteredResponse = waitForQueryLogResponse(page, (url) => url.searchParams.get('search') === 'example.com');
  await queryLogSearchInput(page).fill('example.com');
  await filteredResponse;
  await waitForQueryLogView(page, (text) => text.includes('example.com') && !text.includes('youtube.com'));
});

test('4184 — Query log pagination', async ({ page, agh, api }) => {
  test.setTimeout(90_000);
  const olderDomain = `querylog-older.example`;
  const newerDomain = `querylog-newer.example`;
  await seedQueries(agh, [{ domain: olderDomain, count: 40 }]);
  await waitForQueryLogRecord(api, olderDomain);
  await seedQueries(agh, [{ domain: newerDomain, count: 30 }]);

  const firstPage = await getQueryLog(api, { limit: 20 });
  expect(firstPage.oldest, 'Expected first query-log page to expose an oldest cursor').toBeTruthy();

  await openQueryLog(page);
  await expect(page.locator('body')).toContainText(newerDomain);
  await expect(page.locator('body')).not.toContainText(olderDomain);

  const beforeScrollCount = countOccurrences(await page.locator('body').innerText(), newerDomain);
  const paginationResponse = waitForPaginationResponse(page, (url) => url.searchParams.get('older_than') === firstPage.oldest && url.searchParams.get('search') === '');
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  const nextPage = await paginationResponse;
  expect(nextPage.data?.some((r) => r.question?.name?.includes(olderDomain)), `Expected paginated response to contain older records for ${olderDomain}`).toBeTruthy();

  await expect.poll(async () => visibleDomainCount(page, olderDomain)).toBeGreaterThan(0);
  const afterScrollText = await page.locator('body').innerText();
  const afterScrollCount = countOccurrences(afterScrollText, newerDomain) + countOccurrences(afterScrollText, olderDomain);
  expect(afterScrollCount > beforeScrollCount, `Expected more rows after pagination, before=${beforeScrollCount}, after=${afterScrollCount}`).toBeTruthy();
});

test('4185 — Query log pagination with filter', async ({ page, agh, api }) => {
  test.setTimeout(90_000);
  const filteredDomain = `querylog-filtered.example`;
  const ignoredDomain = `querylog-unfiltered.example`;
  await seedQueries(agh, [{ domain: filteredDomain, count: 35 }]);
  await waitForQueryLogRecord(api, filteredDomain);
  await seedQueries(agh, [{ domain: filteredDomain, count: 35 }, { domain: ignoredDomain, count: 45 }]);

  await openQueryLog(page);
  const filteredResponse = waitForQueryLogResponse(page, (url) => url.searchParams.get('search') === filteredDomain);
  await queryLogSearchInput(page).fill(filteredDomain);
  await filteredResponse;
  await waitForQueryLogView(page, (text) => text.includes(filteredDomain) && !text.includes(ignoredDomain));

  const firstFilteredPage = await getQueryLog(api, { search: filteredDomain, limit: 20 });
  expect(firstFilteredPage.oldest, 'Expected filtered page to expose an oldest cursor').toBeTruthy();

  const beforeScrollCount = countOccurrences(await page.locator('body').innerText(), filteredDomain);
  const paginationResponse = waitForPaginationResponse(page, (url) => url.searchParams.get('older_than') === firstFilteredPage.oldest && url.searchParams.get('search') === filteredDomain);
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  const nextFilteredPage = await paginationResponse;
  expect(nextFilteredPage.data?.length, 'Expected filtered pagination to return a non-empty older page').toBeTruthy();

  await expect.poll(async () => visibleDomainCount(page, filteredDomain)).toBeGreaterThan(beforeScrollCount);
  const afterScrollText = await page.locator('body').innerText();
  expect(countOccurrences(afterScrollText, filteredDomain) > beforeScrollCount, 'Expected more filtered rows after pagination').toBeTruthy();
  expect(afterScrollText.includes(ignoredDomain)).toBe(false);
});

test('4189 — Query log response filter', async ({ page, agh, api }) => {
  const allowedDomain = `querylog-allowed.example`;
  const blockedDomain = `querylog-blocked.example`;
  const processedDomain = 'iana.org';
  await setCustomRules(api, [`@@||${allowedDomain}^$important`, `||${allowedDomain}^`, `||${blockedDomain}^`]);
  await clearQueryLog(api);
  await waitForCategorizedQuery(agh, api, allowedDomain, 'whitelisted');
  await waitForCategorizedQuery(agh, api, blockedDomain, 'blocked');
  await clearQueryLog(api);

  await sendQuery(agh, allowedDomain);
  await sendQuery(agh, blockedDomain);
  await sendQuery(agh, processedDomain);
  await openQueryLog(page);

  const responseSelector = page.locator('select').first();
  // The response-filter <select> is populated by a React render after the log
  // view mounts; wait for its options to exist before selecting, otherwise
  // selectOption flakes with "did not find some options" under load.
  await expect(responseSelector.locator('option', { hasText: 'Filtered' })).toBeAttached({ timeout: 15_000 });
  let r = waitForQueryLogResponse(page);
  await responseSelector.selectOption({ label: 'Filtered' });
  await r;
  await waitForQueryLogView(page, (t) => t.includes(allowedDomain) && t.includes(blockedDomain) && !t.includes(processedDomain));

  r = waitForQueryLogResponse(page);
  await responseSelector.selectOption({ label: 'Processed' });
  await r;
  await waitForQueryLogView(page, (t) => t.includes(processedDomain) && !t.includes(allowedDomain) && !t.includes(blockedDomain));

  r = waitForQueryLogResponse(page);
  await responseSelector.selectOption({ label: 'Blocked' });
  await r;
  await waitForQueryLogView(page, (t) => t.includes(blockedDomain) && !t.includes(allowedDomain) && !t.includes(processedDomain));
});

test('4190 — Query log refresh', async ({ page, agh, api }) => {
  const firstDomain = 'example.com';
  const secondDomain = 'iana.org';
  await sendQuery(agh, firstDomain);
  await openQueryLog(page);
  await expect(page.locator('body')).toContainText(firstDomain);
  await expect(page.locator('body')).not.toContainText(secondDomain);

  const refreshButton = queryLogRefreshButton(page);
  await refreshButton.click();
  await expect(page.locator('body')).toContainText(firstDomain);

  await sendQuery(agh, secondDomain);
  await waitForQueryLogRecord(api, secondDomain);
  await refreshButton.click();
  await expect(page.locator('body')).toContainText(secondDomain);
});

test('4194 — Query log response details', async ({ page, agh, api }) => {
  const processedDomain = 'iana.org';
  const blockedDomain = 'example.com';
  await sendQuery(agh, processedDomain);
  await waitForQueryLogRecord(api, processedDomain);

  await openQueryLog(page);
  let row = queryLogRow(page, processedDomain);
  let tip = await readTooltipText(page, row.locator('.logs__cell--response .tooltip-custom__trigger').first());
  expect(tip).toMatch(/Response details/i);
  expect(tip).toMatch(/Status\s*Processed/i);
  expect(tip).toMatch(/Response code\s*NOERROR/i);

  await setCustomRules(api, [`||${blockedDomain}^`]);
  await clearQueryLog(api);
  await waitForCategorizedQuery(agh, api, blockedDomain, 'blocked');

  await openQueryLog(page);
  await expect(page.locator('body')).toContainText(blockedDomain);
  row = queryLogRow(page, blockedDomain);
  tip = await readTooltipText(page, row.locator('.logs__cell--response .tooltip-custom__trigger').first());
  expect(tip).toMatch(/Response details/i);
  expect(tip).toMatch(/Status\s*Blocked/i);
  expect(tip).toMatch(/Rule\(s\)/i);
  expect(tip).toMatch(/Response\s*A:\s*0\.0\.0\.0/i);
});

test('4144 — Add persistent client from query log', async ({ page, agh, api }) => {
  const requestDomain = `querylog-persistent.example`;
  const savedClientName = 'ui-querylog-persistent-client';
  const updatedClientName = 'ui-querylog-persistent-client-edited';
  await sendQuery(agh, requestDomain);
  const currentClientIp = (await waitForQueryLogRecord(api, requestDomain)).client ?? '127.0.0.1';

  await openQueryLog(page);
  await expect(page.locator('body')).toContainText(requestDomain);
  await clickQueryLogAction(page, requestDomain, 'Add as persistent client');
  await expect(page).toHaveURL(new RegExp(`#clients\\?clientId=${escapeRegExp(currentClientIp)}`));
  await expect(clientNameInput(page)).toHaveValue(`Client ${currentClientIp}`);
  await expect(clientIdentifierInput(page)).toHaveValue(currentClientIp);

  await clientNameInput(page).fill(savedClientName);
  await saveClientForm(page);
  await expect(page.locator('body')).toContainText(`Client "${savedClientName}" successfully added`);
  await expect.poll(async () => (await getClients(api)).clients?.some((c) => c.name === savedClientName && c.ids.includes(currentClientIp)) ?? false).toBe(true);

  const savedClientRow = persistentClientRow(page, savedClientName);
  await expect(savedClientRow).toBeVisible();
  await savedClientRow.getByTitle('Edit').click();
  await expect(clientNameInput(page)).toHaveValue(savedClientName);
  await clientNameInput(page).fill(updatedClientName);
  await saveClientForm(page);
  await expect(page.locator('body')).toContainText('successfully updated');
  await expect.poll(async () => (await getClients(api)).clients?.some((c) => c.name === updatedClientName && c.ids.includes(currentClientIp)) ?? false).toBe(true);
});

test('4186 — Block query from query log', async ({ page, agh, api }) => {
  const globalBlockDomain = 'example.com';
  const clientOnlyBlockDomain = 'iana.org';
  await sendQuery(agh, globalBlockDomain);
  const currentClientIp = (await waitForQueryLogRecord(api, globalBlockDomain)).client ?? '127.0.0.1';

  await openQueryLog(page);
  await clickQueryLogAction(page, globalBlockDomain, 'Block');
  await expect(page.locator('body')).toContainText('Rule added to the custom filtering rules');

  await openCustomRules(page);
  await expect(customRulesTextarea(page)).toHaveValue(new RegExp(escapeRegExp(globalBlockDomain)));
  await expectDomainBlocked(agh, globalBlockDomain);

  await setCustomRules(api, []);
  await clearQueryLog(api);
  await sendQuery(agh, clientOnlyBlockDomain);
  await waitForQueryLogRecord(api, clientOnlyBlockDomain);

  await openQueryLog(page);
  await clickQueryLogAction(page, clientOnlyBlockDomain, 'Block for this client only');
  await expect(page.locator('body')).toContainText('Rule added to the custom filtering rules');

  await openCustomRules(page);
  const rulesValue = await customRulesTextarea(page).inputValue();
  expect(rulesValue).toMatch(new RegExp(escapeRegExp(clientOnlyBlockDomain)));
  expect(rulesValue).toMatch(new RegExp(escapeRegExp(currentClientIp)));
  await expectDomainBlocked(agh, clientOnlyBlockDomain);
});

// Case 4196 (filter by client) moved to tests/ui/querylog_byclient.spec.ts, which
// uses a companion non-loopback DNS client via the Docker-network cluster.
