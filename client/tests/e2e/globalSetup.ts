import { chromium, FullConfig } from '@playwright/test';
import { existsSync, renameSync } from 'fs';

import { ADMIN_USERNAME, ADMIN_PASSWORD, PORT } from '../constants';

export const CONFIG_FILE = 'AdGuardHome.yaml';
export const TEMP_CONFIG_FILE = 'AdGuardHome.yaml.temp';

async function globalSetup(config: FullConfig) {
    // Backup existing config file if it exists
    if (existsSync(CONFIG_FILE)) {
        renameSync(CONFIG_FILE, TEMP_CONFIG_FILE);
    }

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
