import { test, expect } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';
import { execSync } from 'child_process';

test.describe('General Settings', () => {
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

    test('should toggle browsing security feature and verify DNS changes', async ({ page }) => {
        // Navigate to general settings
        await page.goto('/#settings/general');
        
        // Find the browsing security toggle
        const browsingSecurity = page.getByTestId('browsing_security_toggle');
        
        // Check initial state
        const initialState = await browsingSecurity.isChecked();
        
        // Enable browsing security if it's not already enabled
        if (!initialState) {
            await browsingSecurity.click();
            await expect(browsingSecurity).toBeChecked();
        }
        
        // Run nslookup with browsing security enabled
        const resultEnabled = execSync('nslookup totalvirus.com 127.0.0.1').toString();
        
        // Disable browsing security
        await browsingSecurity.click();
        await expect(browsingSecurity).not.toBeChecked();
        
        // Run nslookup with browsing security disabled
        const resultDisabled = execSync('nslookup totalvirus.com 127.0.0.1').toString();
        
        // Compare results (using length as a simple way to detect differences)
        expect(resultEnabled).not.toEqual(resultDisabled);
        
        // Restore initial state
        if (initialState) {
            await browsingSecurity.click();
            await expect(browsingSecurity).toBeChecked();
        }
    });

    test('should toggle parental control feature and verify DNS changes', async ({ page }) => {
        // Navigate to general settings
        await page.goto('/#settings/general');
        
        // Find the parental control toggle
        const parentalControl = page.getByTestId('parental_control_toggle');
        
        // Check initial state
        const initialState = await parentalControl.isChecked();
        
        // Enable parental control if it's not already enabled
        if (!initialState) {
            await parentalControl.click();
            await expect(parentalControl).toBeChecked();
        }
        
        // Run nslookup with parental control enabled
        const resultEnabled = execSync('nslookup pornhub.com 127.0.0.1').toString();
        
        // Disable parental control
        await parentalControl.click();
        await expect(parentalControl).not.toBeChecked();
        
        // Run nslookup with parental control disabled
        const resultDisabled = execSync('nslookup pornhub.com 127.0.0.1').toString();
        
        // Compare results
        expect(resultEnabled).not.toEqual(resultDisabled);
        
        // Restore initial state
        if (initialState) {
            await parentalControl.click();
            await expect(parentalControl).toBeChecked();
        }
    });

    test('should toggle safe search feature', async ({ page }) => {
        // Navigate to general settings
        await page.goto('/#settings/general');
        
        // Find the safe search toggle
        const safeSearch = page.getByTestId('safe_search_toggle');
        
        // Check initial state
        const initialState = await safeSearch.isChecked();
        
        // Toggle it
        await safeSearch.click();
        
        // Verify it changed state
        await expect(safeSearch).not.toBeChecked({ checked: initialState });
        
        // Toggle it back
        await safeSearch.click();
        
        // Verify it's back to the initial state
        await expect(safeSearch).toBeChecked({ checked: initialState });
    });
});
