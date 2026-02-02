import { chromium, type FullConfig } from '@playwright/test';

import { ADMIN_USERNAME, ADMIN_PASSWORD, PORT, CONFIG_FILE_PATH } from '../constants';

const BASE_URL = `http://127.0.0.1:${PORT}`;

async function checkServerAvailable(): Promise<boolean> {
    try {
        const response = await fetch(BASE_URL);
        return response.ok || response.status === 302;
    } catch {
        return false;
    }
}

async function globalSetup(config: FullConfig) {
    if (!process.env.CI) {
        const isServerRunning = await checkServerAvailable();
        if (!isServerRunning) {
            console.error(
                `\nAdGuard Home server is not running. Start it first:\n  sudo ./AdGuardHome --local-frontend -v -c ${CONFIG_FILE_PATH}\n`,
            );
            process.exit(1);
        }
    }

    const browser = await chromium.launch({
        slowMo: 100,
    });
    const page = await browser.newPage({ baseURL: config.webServer?.url || BASE_URL });

    await page.goto('/');

    // Check if we're on the install page or already installed
    const isInstallPage = page.url().includes('/install.html');

    if (isInstallPage) {
        await page.getByTestId('install_get_started').click();
        await page.getByTestId('install_web_port').fill(PORT.toString());
        await page.getByTestId('install_next').click();
        await page.getByTestId('install_username').fill(ADMIN_USERNAME);
        await page.getByTestId('install_username').blur();
        await page.getByTestId('install_password').fill(ADMIN_PASSWORD);
        await page.getByTestId('install_password').blur();
        await page.getByTestId('install_confirm_password').fill(ADMIN_PASSWORD);
        await page.getByTestId('install_confirm_password').blur();
        await page.getByTestId('install_next').click();
        await page.getByTestId('install_next').click();
        await page.getByTestId('install_open_dashboard').click();
        await page.waitForURL((url) => !url.href.endsWith('/install.html'));
    }

    await browser.close();
}

export default globalSetup;
