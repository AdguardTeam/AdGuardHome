import { type Page } from '@playwright/test';

import { test, expect } from '../runtime/fixtures';
import { addClient, clearQueryLog, getStats, resetStats, updateProfile } from '../shared/api/adguard-api.ts';
import {
  clientNameInput, clientIdentifierInput, loginToAdGuardUi, persistentClientRow,
  queryLogSearchInput, queryLogRow, saveClientForm,
} from '../shared/ui/adguard-playwright.ts';

// Queries run from inside the container, so AGH attributes them to 127.0.0.1.
// Registering the client by that IP makes per-client counting work.
const CLIENT_IP = '127.0.0.1';

test.describe.configure({ mode: 'serial' });

function stripWrappedQuotes(value: string): string {
  return value.replace(/^"(.*)"$/, '$1');
}

async function openClientsPage(page: Page): Promise<void> {
  await loginToAdGuardUi(page);
  await page.goto('/#clients');
  await expect(page).toHaveURL(/#clients/);
  await expect(page.locator('body')).toContainText('Client settings');
}

async function openQueryLog(page: Page): Promise<void> {
  await loginToAdGuardUi(page);
  await page.goto('/#logs');
  await expect(page).toHaveURL(/#logs/);
  await expect(page.locator('body')).toContainText('Query Log');
}

async function deletePersistentClientFromUi(page: Page, clientName: string): Promise<void> {
  const row = persistentClientRow(page, clientName);
  await expect(row).toBeVisible();
  const dialogPromise = page.waitForEvent('dialog', { timeout: 1_000 }).catch(() => null);
  await row.locator('button[title="Delete"]').click();
  const dialog = await dialogPromise;
  if (dialog) {
    await dialog.accept();
  } else {
    const confirmDeleteButton = page.getByRole('button', { name: /^delete$/i }).last();
    if (await confirmDeleteButton.isVisible({ timeout: 2_000 }).catch(() => false)) await confirmDeleteButton.click();
  }
  await expect(persistentClientRow(page, clientName)).toHaveCount(0);
}

test.beforeEach(async ({ api }) => {
  await updateProfile(api, { language: 'en' });
});

test('4131 — Client request count', async ({ page, agh, api }) => {
  test.setTimeout(60_000);
  const clientName = 'ui-client-request-count';
  const queriedDomains = ['ui-client-requests-a.example', 'ui-client-requests-b.example', 'ui-client-requests-c.example'];

  await addClient(api, { name: clientName, ids: [CLIENT_IP], use_global_settings: true });
  await clearQueryLog(api);
  await resetStats(api);

  for (const domain of queriedDomains) await agh.dnslookup(domain, { type: 'A' });
  await expect.poll(async () => (await getStats(api)).num_dns_queries).toBeGreaterThanOrEqual(queriedDomains.length);

  await openClientsPage(page);
  const row = persistentClientRow(page, clientName);
  await expect(row).toContainText(String(queriedDomains.length));
  await row.getByText(String(queriedDomains.length), { exact: true }).click();

  await expect(page).toHaveURL(/#logs/);
  const searchValue = stripWrappedQuotes(await queryLogSearchInput(page).inputValue());
  expect(searchValue === clientName || searchValue === CLIENT_IP,
    `Expected Query Log filtered by ${clientName} or ${CLIENT_IP}, got ${searchValue}`).toBeTruthy();
  for (const domain of queriedDomains) await expect(page.locator('body')).toContainText(domain);
});
