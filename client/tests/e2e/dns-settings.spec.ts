import { test, expect, type Page } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

test.describe('DNS Settings', () => {
    test.beforeEach(async ({ page }) => {
        // Login before each test
        await page.goto('/login.html');
        await page.getByTestId('username').click();
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').click();
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.getByTestId('sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
    });

    const runDNSSettingsTest = async (page: Page, address: string) => {
        await page.goto('/#dns');

        const currentDns = await page.getByTestId('upstream_dns').inputValue();

        await page.getByTestId('upstream_dns').fill(address);
        await page.getByTestId('dns_upstream_test').click();

        await page.waitForTimeout(2000);

        await expect(page.getByTestId('upstream_dns')).toHaveValue(address);

        await page.getByTestId('upstream_dns').fill(currentDns);
        await page.getByTestId('dns_upstream_save').click({ force: true });
    };

    test('test for Default DNS', async ({ page }) => {
        await runDNSSettingsTest(page, 'https://dns10.quad9.net/dns-query');
    });

    test('test for Plain DNS', async ({ page }) => {
        await runDNSSettingsTest(page, '94.140.14.140');
    });

    test('test for DNS-over-HTTPS', async ({ page }) => {
        await runDNSSettingsTest(page, 'https://unfiltered.adguard-dns.com/dns-query');
    });

    test('test for DNS-over-TLS', async ({ page }) => {
        await runDNSSettingsTest(page, 'tls://unfiltered.adguard-dns.com');
    });

    test('test for DNS-over-QUIC', async ({ page }) => {
        await runDNSSettingsTest(page, 'quic://unfiltered.adguard-dns.com');
    });
});
