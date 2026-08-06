import type { Page } from '@playwright/test';

import { ADMIN_PASSWORD, ADMIN_USERNAME } from '../constants';

/**
 * Logs in to the AdGuard Home dashboard with retry logic.
 * Retries up to 3 times to handle flaky page load timing.
 */
export async function login(page: Page): Promise<void> {
    let lastError: unknown;

    for (let attempt = 0; attempt < 3; attempt += 1) {
        await page.goto('/login.html', { waitUntil: 'domcontentloaded' });

        try {
            await page.locator('#username').waitFor({ state: 'visible', timeout: 5000 });
            await page.locator('#username').fill(ADMIN_USERNAME);
            await page.locator('#password').fill(ADMIN_PASSWORD);
            await page.locator('#sign_in').click();
            await page.waitForURL((url) => !url.href.endsWith('/login.html'));

            return;
        } catch (error) {
            lastError = error;
        }
    }

    throw lastError;
}
