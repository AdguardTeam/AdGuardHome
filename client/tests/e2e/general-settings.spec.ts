import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

test.describe('General Settings', () => {
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

    test('should toggle browsing security feature and verify DNS changes', async ({ page }) => {
        await page.goto('/#settings');

        const browsingSecurity = await page.getByTestId('safebrowsing');
        const browsingSecurityLabel = await browsingSecurity.locator('xpath=following-sibling::*[1]');

        const initialState = await browsingSecurity.isChecked();

        if (!initialState) {
            await browsingSecurityLabel.click();
            await expect(browsingSecurity).toBeChecked();
        }

        const resultEnabled = execSync('nslookup totalvirus.com 127.0.0.1').toString();

        await browsingSecurityLabel.click();
        await expect(browsingSecurity).not.toBeChecked();

        const resultDisabled = execSync('nslookup totalvirus.com 127.0.0.1').toString();

        expect(resultEnabled).not.toEqual(resultDisabled);

        if (initialState) {
            await browsingSecurityLabel.click();
            await expect(browsingSecurity).toBeChecked();
        }
    });

    test('should toggle parental control feature and verify DNS changes', async ({ page }) => {
        await page.goto('/#settings');

        const parentalControl = page.getByTestId('parental');
        const parentalControlLabel = await parentalControl.locator('xpath=following-sibling::*[1]');

        const initialState = await parentalControl.isChecked();

        if (!initialState) {
            await parentalControlLabel.click();
            await expect(parentalControl).toBeChecked();
        }

        const resultEnabled = execSync('nslookup pornhub.com 127.0.0.1').toString();

        await parentalControlLabel.click();
        await expect(parentalControl).not.toBeChecked();

        const resultDisabled = execSync('nslookup pornhub.com 127.0.0.1').toString();

        expect(resultEnabled).not.toEqual(resultDisabled);

        if (initialState) {
            await parentalControlLabel.click();
            await expect(parentalControl).toBeChecked();
        }
    });

    test('should toggle safe search feature', async ({ page }) => {
        await page.goto('/#settings');

        const safeSearch = page.getByTestId('safesearch');
        const safeSearchLabel = await safeSearch.locator('xpath=following-sibling::*[1]');

        const initialState = await safeSearch.isChecked();

        await safeSearchLabel.click();

        await expect(safeSearch).not.toBeChecked({ checked: initialState });

        await safeSearchLabel.click();

        await expect(safeSearch).toBeChecked({ checked: initialState });
    });
});
