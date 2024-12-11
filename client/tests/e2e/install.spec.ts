import { test, expect } from '@playwright/test';

const ADMIN_USERNAME = 'admin';
const ADMIN_PASSWORD = 'superpassword';
const PORT = 3000;

test('install', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Get Started' }).click();
    await page.locator('input[name="web\\.port"]').fill(PORT.toString());
    await page.getByRole('button', { name: 'Next' }).click();
    await page.getByPlaceholder('Enter username').fill(ADMIN_USERNAME);
    await page.getByPlaceholder('Enter password').fill(ADMIN_PASSWORD);
    await page.getByPlaceholder('Confirm password').fill(ADMIN_PASSWORD);
    await page.getByRole('button', { name: 'Next' }).click();
    await page.getByRole('button', { name: 'Next' }).click();
});
