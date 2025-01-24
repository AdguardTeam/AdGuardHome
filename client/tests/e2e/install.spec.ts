import { test } from '@playwright/test';

const ADMIN_USERNAME = 'admin';
const ADMIN_PASSWORD = 'superpassword';
const PORT = 3000;

test('install', async ({ page }) => {
    await page.goto('/');
    await page.getByTestId('install_get_started').click();
    await page.getByTestId('install_web_port').fill(PORT.toString());
    await page.getByTestId('install_next').click();
    await page.getByTestId('install_username').fill(ADMIN_USERNAME);
    await page.getByTestId('install_password').fill(ADMIN_PASSWORD);
    await page.getByTestId('install_confirm_password').fill(ADMIN_PASSWORD);
    await page.getByTestId('install_next').click();
    await page.getByTestId('install_next').click();
    await page.getByTestId('install_open_dashboard').click();
});
