import { test, expect } from '@playwright/test';
import { ADMIN_PASSWORD, ADMIN_USERNAME } from '../constants';

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
        const domain = `add.org`;
        const answer = '127.0.0.1';
        
        await page.getByTestId('add-rewrite').click();
        await page.getByTestId('rewrites_domain').fill(domain);
        await page.getByTestId('rewrites_answer').fill(answer);
        await page.getByTestId('rewrites_save').click();

        await expect(page.locator('.logs__text').filter({ hasText: domain })).toBeVisible();
        await expect(page.locator('.logs__text').filter({ hasText: answer })).toBeVisible();
    });

    test('should edit a DNS rewrite', async ({ page }) => {
        const originalDomain = `edit.org`;
        const updatedDomain = `updated.org`;
        const answer = '192.168.1.1';

        await page.getByTestId('add-rewrite').click();
        await page.getByTestId('rewrites_domain').fill(originalDomain);
        await page.getByTestId('rewrites_answer').fill(answer);
        await page.getByTestId('rewrites_save').click();

        await page.getByTestId('edit-rewrite').first().click();
        await expect(page.getByTestId('rewrites_domain')).toHaveValue(originalDomain);
        
        await page.getByTestId('rewrites_domain').clear();
        await page.getByTestId('rewrites_domain').fill(updatedDomain);
        await page.getByTestId('rewrites_save').click();

        await expect(page.locator('.logs__text').filter({ hasText: updatedDomain })).toBeVisible();
        await expect(page.locator('.logs__text').filter({ hasText: answer })).toBeVisible();
    });
});
