import { test, expect } from '@playwright/test';
import { ADMIN_PASSWORD, ADMIN_USERNAME } from '../constants';

const INTERFACE_NAME = 'en0';
const RANGE_START = '192.168.1.100';
const RANGE_END = '192.168.1.200';
const SUBNET_MASK = '255.255.255.0';
const LEASE_TIME = '86400';

test.describe('DHCP Configuration', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login.html');
        await page.getByTestId('username').click();
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').click();
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.getByTestId('sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
        await page.goto(`/#dhcp`);
    });

    test('should select the correct DHCP interface', async ({ page }) => {
        await page.getByTestId('interface_name').selectOption(INTERFACE_NAME);
        expect(await page.locator('select[name="interface_name"]').inputValue()).toBe(INTERFACE_NAME);
    });

    test('should configure DHCP IPv4 settings correctly', async ({ page }) => {
        await page.getByTestId('interface_name').selectOption(INTERFACE_NAME);
        await page.getByTestId('v4_gateway_ip').click();
        await page.getByTestId('v4_gateway_ip').fill('192.168.1.99');
        await page.getByTestId('v4_subnet_mask').click();
        await page.getByTestId('v4_subnet_mask').fill(SUBNET_MASK);
        await page.getByTestId('v4_range_start').click();
        await page.getByTestId('v4_range_start').fill(RANGE_START);
        await page.getByTestId('v4_range_end').click();
        await page.getByTestId('v4_range_end').fill(RANGE_END);
        await page.getByTestId('v4_lease_duration').click();
        await page.getByTestId('v4_lease_duration').fill(LEASE_TIME);
        await page.getByTestId('v4_save').click();
    });

    test('should show error for invalid DHCP IPv4 range', async ({ page }) => {
        await page.getByTestId('interface_name').selectOption(INTERFACE_NAME);
        await page.getByTestId('v4_range_start').click();
        await page.getByTestId('v4_range_start').fill(RANGE_END);
        await page.getByTestId('v4_range_end').click();
        await page.getByTestId('v4_range_end').fill(RANGE_START);
        await page.keyboard.press('Tab');

        expect(await page.getByText('Must be greater than range').isVisible()).toBe(true);
    });

    test('should show error for invalid DHCP IPv4 address', async ({ page }) => {
        await page.getByTestId('interface_name').selectOption(INTERFACE_NAME);
        await page.getByTestId('v4_gateway_ip').click();
        await page.getByTestId('v4_gateway_ip').fill('192.168.1.200s');
        await page.keyboard.press('Tab');

        expect(await page.getByText('Invalid IPv4 address').isVisible()).toBe(true);
    });
});
