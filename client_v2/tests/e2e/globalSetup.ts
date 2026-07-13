import { chromium, type FullConfig } from '@playwright/test';

import { ADMIN_USERNAME, ADMIN_PASSWORD, PORT } from '../constants';

async function globalSetup(config: FullConfig) {
    const browser = await chromium.launch({
        slowMo: 100,
    });
    const page = await browser.newPage({ baseURL: config.webServer?.url });

    try {
        await page.goto('/');

        const { pathname } = new URL(page.url());

        if (pathname === '/login.html' || pathname === '/') {
            return;
        }

        if (pathname !== '/install.html') {
            throw new Error(`Unexpected initial page during global setup: ${pathname}`);
        }

        // Step 1: Greeting
        await page.locator('#install_get_started').click();

        // Step 2: Auth
        await page.locator('#install_username').fill(ADMIN_USERNAME);
        await page.locator('#install_password').fill(ADMIN_PASSWORD);
        await page.locator('#install_confirm_password').fill(ADMIN_PASSWORD);
        await page.locator('#install_next').click();

        // Step 3: Interface settings
        await page.locator('#install_web_port').fill(PORT.toString());
        await page.locator('#install_next').click();

        // Step 4: DNS settings
        await page.locator('#install_next').click();

        // Step 5: Setup guide
        await page.locator('#install_next').click();

        // Step 6: Submit — open dashboard
        await page.locator('#open_dashboard').click();
        await page.waitForURL((url) => !url.href.endsWith('/install.html'));
    } catch (error) {
        console.error('Error during global setup:', error);
    } finally {
        await browser.close();
    }
}

export default globalSetup;
