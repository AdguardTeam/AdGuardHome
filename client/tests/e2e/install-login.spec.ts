import { test, expect } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

test.describe('Install and Login', () => {
    test('should complete the onboarding process successfully', async ({ page }) => {
        // Navigate to setup page
        await page.goto('/install.html');
        
        // Complete onboarding steps
        await page.getByTestId('install_get_started').getByText('Get Started').click();
        await page.getByTestId('install_next').getByText('Next').click();
        
        // Fill onboarding form
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.getByTestId('confirm_password').fill(ADMIN_PASSWORD);
        
        await page.getByTestId('install_next').getByText('Next').click();
        await page.getByTestId('install_next').getByText('Next').click();
        await page.getByTestId('install_open_dashboard').getByText('Open Dashboard').click();
        
        // Verify 'Sign in' button is disabled
        const signInButton = page.locator('button[type="submit"][class*="btn-success"]:has-text("Sign in")');
        await expect(signInButton).toBeDisabled();
    });
    
    test('should login successfully and show the dashboard', async ({ page }) => {
        // Navigate to dashboard
        await page.goto('/');
        
        // Login
        await page.getByTestId('username').click();
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').click();
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.getByTestId('sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
        
        // Verify dashboard is displayed
        const dashboardHeader = page.locator('h1.page-title.pr-2');
        await expect(dashboardHeader).toHaveText('Dashboard');
    });
});
