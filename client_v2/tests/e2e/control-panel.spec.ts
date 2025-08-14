import { test } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

test.describe('Control Panel', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login.html');
        await page.locator('#username').click();
        await page.locator('#username').fill(ADMIN_USERNAME);
        await page.locator('#password').click();
        await page.locator('#password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.locator('#sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
    });
});
