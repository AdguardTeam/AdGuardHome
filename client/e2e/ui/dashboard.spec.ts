import { type Page } from '@playwright/test';

import { test, expect } from '../runtime/fixtures';
import { resolveAnswers as resolveDnsAnswers } from '../shared/dns/dns-test-helpers.ts';
import {
  clearQueryLog, getQueryLog, getStatus, getStats, resetStats, setCustomRules, setProtection, updateProfile,
} from '../shared/api/adguard-api.ts';
import { loginToAdGuardUi, scrollToFooter } from '../shared/ui/adguard-playwright.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';

test.describe.configure({ mode: 'serial' });

function findNewestAnswerValue(records: Awaited<ReturnType<typeof getQueryLog>>['data'], minTimeMs: number): string | undefined {
  return records
    .filter((r) => typeof r.time === 'string' && Date.parse(r.time) >= minTimeMs)
    .sort((l, r) => Date.parse(r.time ?? '') - Date.parse(l.time ?? ''))[0]?.answer?.[0]?.value;
}

const dashboardDnsQueriesCounter = async (page: Page): Promise<number> =>
  Number.parseInt(((await page.locator('a[href="#logs?response_status=all"]').last().textContent()) ?? '0').replace(/[^\d]/g, ''), 10) || 0;
const dashboardBlockedQueriesCounter = async (page: Page): Promise<number> =>
  Number.parseInt(((await page.locator('a[href="#logs?response_status=blocked"]').last().textContent()) ?? '0').replace(/[^\d]/g, ''), 10) || 0;
const dashboardRefreshButtons = (page: Page) => page.locator('button[title="Refresh"], button:has-text("Refresh statistics")');

function rankedValue(items: Array<Record<string, number>>, key: string): number {
  for (const item of items) if (Object.prototype.hasOwnProperty.call(item, key)) return item[key] ?? 0;
  return 0;
}
function rankedIndex(items: Array<Record<string, number>>, key: string): number {
  return items.findIndex((item) => Object.prototype.hasOwnProperty.call(item, key));
}

async function refreshDashboardStatistics(page: Page): Promise<void> {
  const refreshButtons = dashboardRefreshButtons(page);
  const count = await refreshButtons.count();
  expect(count >= 5, `Expected at least 5 refresh buttons, got ${count}`).toBeTruthy();
  for (let i = 0; i < count; i += 1) {
    await refreshButtons.nth(i).scrollIntoViewIfNeeded();
    await refreshButtons.nth(i).click();
  }
}

const footerLanguageSelect = (page: Page) => page.locator('select').last();
const footerThemeButton = (page: Page, titleKeyword: string) => page.locator(`button.footer__theme-button[title*="${titleKeyword}"]`).first();
async function expectBodyTheme(page: Page, theme: 'light' | 'dark'): Promise<void> {
  await expect.poll(async () => page.locator('body').getAttribute('data-theme')).toBe(theme);
}

test.beforeEach(async ({ api }) => {
  await updateProfile(api, { language: 'en', theme: 'light' });
});

test('4045 — Disable/Enable protection', async ({ page, agh, api }) => {
  const blockedDomain = 'example.com';
  await setCustomRules(api, [`||${blockedDomain}^`]);
  await loginToAdGuardUi(page);

  const disableStartedAt = Date.now();
  await page.getByRole('button', { name: /disable protection/i }).click();
  await expect(page.locator('body')).toContainText('OFF');
  await expect.poll(async () => (await getStatus(api)).protection_enabled).toBe(false);

  const answersWhileDisabled = await resolveDnsAnswers(agh, blockedDomain, 'A');
  expect(answersWhileDisabled.length > 0 && !answersWhileDisabled.includes('0.0.0.0'),
    `Expected ${blockedDomain} to resolve while protection disabled, got ${JSON.stringify(answersWhileDisabled)}`).toBeTruthy();
  const newestAnswer = findNewestAnswerValue((await getQueryLog(api, { search: blockedDomain, limit: 20 })).data, disableStartedAt);
  expect(newestAnswer, `Expected a query-log answer for ${blockedDomain} while protection disabled`).toBeDefined();
  expect(newestAnswer).not.toBe('0.0.0.0');

  const enableStartedAt = Date.now();
  await page.getByRole('button', { name: /enable protection/i }).click();
  await expect(page.locator('body')).toContainText('ON');
  await expect.poll(async () => (await getStatus(api)).protection_enabled).toBe(true);

  const answersWhileEnabled = await resolveDnsAnswers(agh, blockedDomain, 'A');
  expect(answersWhileEnabled.includes('0.0.0.0'),
    `Expected ${blockedDomain} blocked after protection enabled, got ${JSON.stringify(answersWhileEnabled)}`).toBeTruthy();
  expect(findNewestAnswerValue((await getQueryLog(api, { search: blockedDomain, limit: 20 })).data, enableStartedAt)).toBe('0.0.0.0');
});

test('4046 — Disable protection for interval', async ({ page, api }) => {
  test.setTimeout(90_000);
  await loginToAdGuardUi(page);

  const toggle = page.locator('.dropdown-protection__toggle').first();
  await toggle.click();
  await expect(page.locator('.dropdown-item')).toContainText(['For 30 seconds', 'For 1 minute', 'For 10 minutes', 'For 1 hour']);

  await page.getByText('For 30 seconds', { exact: true }).click();
  await expect(page.locator('body')).toContainText('OFF');
  await expect.poll(async () => (await getStatus(api)).protection_enabled).toBe(false);
  await expect.poll(async () => (await getStatus(api)).protection_disabled_duration, { timeout: 5_000 }).toBeGreaterThan(20_000);

  await expect.poll(async () => (await getStatus(api)).protection_enabled, { timeout: 45_000, intervals: [1_000] }).toBe(true);
  await expect(page.locator('body')).toContainText('ON');

  await toggle.click();
  await page.getByText('For 1 minute', { exact: true }).click();
  await expect.poll(async () => (await getStatus(api)).protection_enabled).toBe(false);
  await expect.poll(async () => (await getStatus(api)).protection_disabled_duration, { timeout: 5_000 }).toBeGreaterThan(45_000);

  await setProtection(api, { enabled: true });
  await expect.poll(async () => (await getStatus(api)).protection_enabled).toBe(true);
});

test('4051 — Refresh statistics', async ({ page, agh, api }) => {
  await loginToAdGuardUi(page);
  const baselineCount = await dashboardDnsQueriesCounter(page);

  await resolveDnsAnswers(agh, 'example.com', 'A');
  await resolveDnsAnswers(agh, 'youtube.com', 'A');
  await resolveDnsAnswers(agh, 'iana.org', 'A');
  await refreshDashboardStatistics(page);

  await expect.poll(async () => dashboardDnsQueriesCounter(page)).toBeGreaterThan(baselineCount);
  await expect.poll(async () => (await getStats(api)).num_dns_queries).toBeGreaterThan(baselineCount);
});

test('4047 — General statistics', async ({ page, agh, api }) => {
  const blockedDomain = 'stats-blocked.example';
  const allowedDomain = 'stats-allowed.example';
  await setCustomRules(api, [`||${blockedDomain}^`]);
  await loginToAdGuardUi(page);

  const baselineTotalUi = await dashboardDnsQueriesCounter(page);
  const baselineBlockedUi = await dashboardBlockedQueriesCounter(page);
  const baselineStats = await getStats(api);

  await resolveDnsAnswers(agh, allowedDomain, 'A');
  await resolveDnsAnswers(agh, allowedDomain, 'A');
  await resolveDnsAnswers(agh, blockedDomain, 'A');
  await refreshDashboardStatistics(page);

  await expect.poll(async () => dashboardDnsQueriesCounter(page)).toBeGreaterThan(baselineTotalUi);
  await expect.poll(async () => dashboardBlockedQueriesCounter(page)).toBeGreaterThan(baselineBlockedUi);
  await expect.poll(async () => (await getStats(api)).num_dns_queries).toBeGreaterThan(baselineStats.num_dns_queries);
  await expect.poll(async () => (await getStats(api)).num_blocked_filtering).toBeGreaterThan(baselineStats.num_blocked_filtering);
});

test('4056 — Version shown in footer', async ({ page, api }) => {
  const status = await getStatus(api);
  await loginToAdGuardUi(page);
  await scrollToFooter(page);
  await expect(page.locator('body')).toContainText('Version:');
  await expect(page.locator('body')).toContainText(status.version);
});

test('4054 — Change interface language', async ({ page }) => {
  await loginToAdGuardUi(page);
  await scrollToFooter(page);

  await footerLanguageSelect(page).selectOption('ru');
  await expect(page.locator('body')).toContainText('Панель управления');
  await expect(page.getByRole('link', { name: 'Журнал' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Инструкция по настройке' })).toBeVisible();

  await page.getByRole('link', { name: 'Журнал' }).click();
  await expect(page).toHaveURL(/#logs/);
  await expect(page.locator('body')).toContainText('Время');
  await expect(page.locator('body')).toContainText('Журнал');

  await page.getByRole('link', { name: 'Инструкция по настройке' }).click();
  await expect(page).toHaveURL(/#guide/);
  await expect(page.locator('body')).toContainText('Инструкция по настройке');

  await scrollToFooter(page);
  await footerLanguageSelect(page).selectOption('en');
  await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Query Log' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Setup Guide' })).toBeVisible();
});

test('4055 — Theme change', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'dark' });
  await loginToAdGuardUi(page);
  await scrollToFooter(page);

  await expect(footerThemeButton(page, 'Auto')).toBeVisible();
  await expect(footerThemeButton(page, 'Dark')).toBeVisible();
  await expect(footerThemeButton(page, 'Light')).toBeVisible();

  await footerThemeButton(page, 'Light').click();
  await expectBodyTheme(page, 'light');
  await page.getByRole('link', { name: 'Query Log' }).click();
  await expect(page).toHaveURL(/#logs/);
  await expectBodyTheme(page, 'light');

  await page.getByRole('link', { name: 'Dashboard' }).click();
  await expect(page).toHaveURL(/\/(#)?$/);
  await scrollToFooter(page);
  await footerThemeButton(page, 'Dark').click();
  await expectBodyTheme(page, 'dark');
  await page.getByRole('link', { name: 'Setup Guide' }).click();
  await expect(page).toHaveURL(/#guide/);
  await expectBodyTheme(page, 'dark');

  await page.getByRole('link', { name: 'Dashboard' }).click();
  await expect(page).toHaveURL(/\/(#)?$/);
  await scrollToFooter(page);
  await footerThemeButton(page, 'Auto').click();
  await expectBodyTheme(page, 'dark');
});
