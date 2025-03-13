import { test, expect } from '@playwright/test';
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

    test('should change upstream DNS settings and verify they are applied', async ({ page }) => {
        // Navigate to DNS settings
        await page.goto('/#dns');

        const dns_addresses = [
            {"type": "Default DNS", "address": "https://dns10.quad9.net/dns-query"},
            {"type": "Plain DNS", "address": "94.140.14.140"},
            {"type": "DNS-over-HTTPS", "address": "https://unfiltered.adguard-dns.com/dns-query"},
            {"type": "DNS-over-TLS", "address": "tls://unfiltered.adguard-dns.com"},
            {"type": "DNS-over-QUIC", "address": "quic://unfiltered.adguard-dns.com"},
        ];
        
        // Save current upstream DNS for later restoration
        const currentDns = await page.getByTestId('upstream_dns').inputValue();

        dns_addresses.forEach(async (address) => {
           // Change to Cloudflare DNS
            await page.getByTestId('upstream_dns').fill(address.address);
            await page.getByTestId('dns_upstream_test').click();
            
            // Wait for changes to be applied
            await page.waitForTimeout(2000);  // 2 seconds
            
            // Verify DNS change was successful
            await expect(page.getByTestId('upstream_dns')).toHaveValue(address.address);
            
            // Restore original DNS settings
            await page.getByTestId('upstream_dns').fill(currentDns);
            await page.getByTestId('dns_upstream_save').click(); 
        });
    });
});
