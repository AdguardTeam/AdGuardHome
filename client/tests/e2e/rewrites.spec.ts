import { test, expect } from '@playwright/test';
import { ADMIN_PASSWORD, ADMIN_USERNAME } from '../constants';

const EXAMPLE_DOMAIN = `example.org`;
const EXAMPLE_UPDATED_DOMAIN = `updated.org`;
const EXAMPLE_ANSWER = '192.168.1.1';

test.describe('Rewrites', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login.html');
        await page.getByTestId('username').click();
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').click();
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.getByTestId('sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
        await page.goto('/#dns_rewrites');
    });

    test('should add a DNS rewrite', async ({ page }) => {
        await page.getByTestId('add-rewrite').click();
        await page.getByTestId('rewrites_domain').fill(EXAMPLE_DOMAIN);
        await page.getByTestId('rewrites_answer').fill(EXAMPLE_ANSWER);
        await page.getByTestId('rewrites_save').click();

        await expect(page.locator('.logs__text').filter({ hasText: EXAMPLE_DOMAIN }).first()).toBeVisible();
        await expect(page.locator('.logs__text').filter({ hasText: EXAMPLE_ANSWER }).first()).toBeVisible();
    });

    test('should edit a DNS rewrite', async ({ page }) => {
        // Use the first existing rewrite instead of creating a new one
        // Wait for the table to load
        await expect(page.getByTestId('edit-rewrite').first()).toBeVisible({ timeout: 10000 });

        // Get the current domain value before editing
        await page.getByTestId('edit-rewrite').first().click();
        const originalDomain = await page.getByTestId('rewrites_domain').inputValue();

        // Edit the domain - use keyboard to ensure isDirty is triggered
        const domainInput = page.getByTestId('rewrites_domain');
        await domainInput.click();
        await domainInput.press('Control+a');
        await domainInput.pressSequentially(EXAMPLE_UPDATED_DOMAIN);
        await domainInput.blur();
        await page.getByTestId('rewrites_save').click();

        // Verify the update
        await expect(page.locator('.logs__text').filter({ hasText: EXAMPLE_UPDATED_DOMAIN }).first()).toBeVisible({ timeout: 10000 });

        // Restore original value
        await page.getByTestId('edit-rewrite').first().click();
        const restoreInput = page.getByTestId('rewrites_domain');
        await restoreInput.click();
        await restoreInput.press('Control+a');
        await restoreInput.pressSequentially(originalDomain);
        await restoreInput.blur();
        await page.getByTestId('rewrites_save').click();
    });
});
