import { test, expect } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

test.describe('Control Panel', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login.html');
        await page.getByTestId('username').click();
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').click();
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.getByTestId('sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
    });

    test('should sign out successfully', async ({ page }) => {
        // Click sign out
        await page.getByTestId('sign_out').click();
        
        // Verify redirect to login page
        await page.waitForURL((url) => url.href.endsWith('/login.html'));
        
        // Verify login form is visible
        await expect(page.getByTestId('sign_in')).toBeVisible();
    });

    test('should change theme to dark and then light', async ({ page }) => {
        // Select dark theme
        await page.getByTestId('theme_dark').click();
        
        // Verify dark theme is applied
        await expect(page.locator('body[data-theme="dark"]')).toBeVisible();

        // Select light theme
        await page.getByTestId('theme_light').click();
        
        // Verify light theme is applied
        await expect(page.locator('body:not([data-theme="dark"])')).toBeVisible();
    });
});
