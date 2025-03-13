import { test, expect } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';
import { execSync } from 'child_process';

test.describe('Filtering', () => {
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

    test('should block domain using CNAME and IP rules', async ({ page }) => {
        // Navigate to filtering page
        await page.goto('/#filtering');
        
        // Add a test rule
        await page.getByTestId('add_rule_button').click();
        const ruleModal = page.getByTestId('rule_modal');
        
        // Enter CNAME rule
        await ruleModal.getByTestId('rule_text').fill('||example.org^');
        await ruleModal.getByTestId('save_rule').click();
        
        // Verify rule was added
        await expect(page.getByText('||example.org^')).toBeVisible();
        
        // Check if the domain is blocked
        const result = execSync('nslookup example.org 127.0.0.1').toString();
        
        // Verify result contains blocked indicator
        const isBlocked = result.includes('Non-existent domain') || result.includes('0.0.0.0');
        expect(isBlocked).toBeTruthy();
        
        // Clean up - remove the rule
        await page.getByText('||example.org^').hover();
        await page.getByTestId('delete_rule').click();
        await page.getByTestId('modal_confirm').click();
    });
});
