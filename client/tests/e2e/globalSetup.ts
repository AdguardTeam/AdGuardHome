import { chromium, type FullConfig } from '@playwright/test';

import { ADMIN_USERNAME, ADMIN_PASSWORD, PORT } from '../constants';

async function globalSetup(config: FullConfig) {
    const browser = await chromium.launch({
        slowMo: 100,
    });
    const page = await browser.newPage({ baseURL: config.webServer?.url });

    try {
        await page.goto('/');
        await page.getByTestId('install_get_started').click();
        await page.getByTestId('install_web_port').fill(PORT.toString());
        await page.getByTestId('install_next').click();
        await page.getByTestId('install_username').fill(ADMIN_USERNAME);
        await page.getByTestId('install_password').fill(ADMIN_PASSWORD);
        await page.getByTestId('install_confirm_password').click();
        await page.getByTestId('install_confirm_password').fill(ADMIN_PASSWORD);
        await page.getByTestId('install_next').click();
        await page.getByTestId('install_next').click();
        await page.getByTestId('install_open_dashboard').click();
        await page.waitForURL((url) => !url.href.endsWith('/install.html'));
    } catch (error) {
        console.error('Error during global setup:', error);
    } finally {
        await browser.close();
    }
}

export default globalSetup;
