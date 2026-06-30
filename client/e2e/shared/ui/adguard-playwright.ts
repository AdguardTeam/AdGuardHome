import { expect, type Locator, type Page } from '@playwright/test';

export interface AdGuardUiCredentials {
  username: string;
  password: string;
}

const DEFAULT_CREDENTIALS: AdGuardUiCredentials = {
  username: process.env.ADGUARD_USER ?? 'admin',
  password: process.env.ADGUARD_PASSWORD ?? 'password',
};

export async function loginToAdGuardUi(
  page: Page,
  credentials: AdGuardUiCredentials = DEFAULT_CREDENTIALS,
): Promise<void> {
  await page.goto('/');

  if (!page.url().includes('/login.html')) {
    const dashboardLink = page.getByRole('link', { name: 'Dashboard' });

    try {
      await expect(
        dashboardLink,
        `Expected either /login.html or an authenticated dashboard after page.goto('/'), received ${page.url()}`,
      ).toBeVisible({ timeout: 5_000 });
      return;
    } catch (error) {
      throw new Error(
        `Unexpected page after page.goto('/'): ${page.url()}. `
          + `Expected /login.html or an authenticated dashboard. `
          + `Dashboard visibility check failed: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  await page.locator('input[type="text"]').first().fill(credentials.username);
  await page.locator('input[type="password"]').fill(credentials.password);
  await page.getByRole('button', { name: /sign in/i }).click();
  await expect(page).toHaveURL('/');
  await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible();
}

export async function goToAdGuardSection(
  page: Page,
  hash: string,
  heading: string | RegExp,
): Promise<void> {
  const normalizedHash = hash.startsWith('#') ? hash : `#${hash}`;

  await page.goto(`/${normalizedHash}`);
  await expect(page).toHaveURL(new RegExp(`${normalizedHash.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`));
  await expect(page.getByRole('heading', { name: heading })).toBeVisible();
}

export async function scrollToFooter(page: Page): Promise<void> {
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
}

export function queryLogSearchInput(page: Page): Locator {
  return page.getByPlaceholder(/domain or client/i);
}

export function persistentClientRow(page: Page, clientName: string): Locator {
  return page.locator('.rt-tr', {
    has: page.getByText(clientName, { exact: true }),
  }).first();
}

export function queryLogRow(page: Page, domain: string): Locator {
  return page.locator('[data-testid="querylog_cell"], .logs__row[role="row"], .rt-tr', {
    has: page.getByText(domain, { exact: true }),
  }).first();
}

export function clientNameInput(page: Page): Locator {
  return page.locator('input[placeholder="Enter client name"]').first();
}

export function clientIdentifierInput(page: Page): Locator {
  return page.locator('input[placeholder="Enter identifier"]').first();
}

export async function saveClientForm(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Save' }).last().click();
}

export function customRulesTextarea(page: Page): Locator {
  return page.locator('textarea').first();
}
