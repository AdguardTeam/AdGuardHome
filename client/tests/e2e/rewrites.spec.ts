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

        await expect(page.locator('.logs__text').filter({ hasText: EXAMPLE_DOMAIN })).toBeVisible();
        await expect(page.locator('.logs__text').filter({ hasText: EXAMPLE_ANSWER })).toBeVisible();
    });

    test('should edit a DNS rewrite', async ({ page }) => {
        await page.getByTestId('edit-rewrite').first().click();
        await expect(page.getByTestId('rewrites_domain')).toHaveValue(EXAMPLE_DOMAIN);

        await page.getByTestId('rewrites_domain').clear();
        await page.getByTestId('rewrites_domain').fill(EXAMPLE_UPDATED_DOMAIN);
        await page.getByTestId('rewrites_save').click();

        await expect(page.locator('.logs__text').filter({ hasText: EXAMPLE_UPDATED_DOMAIN })).toBeVisible();
    });
});
