import { test, expect, type Page } from '@playwright/test';
import { execSync } from 'child_process';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

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

    const runTerminalCommand = (command: string) => {
        try {
            console.info(`Executing command: ${command}`);
        
            const output = execSync(command, { encoding: 'utf-8', stdio: 'pipe' }).trim();
        
            console.info('Command executed successfully.');
            console.debug(`Command output:\n${output}`);
        
            return output;
        } catch (error: any) {
            console.error(`Command execution failed with error:\n${error.message}`);
            throw new Error(`Failed to execute command: ${command}\nError: ${error.message}`);
        }
    }

    const runCustomRuleTest = async (page: Page, domain_to_block: string) => {
        await page.goto('/#custom_rules');

        await page.getByTestId('custom_rule_textarea').fill(domain_to_block);
        await page.getByTestId('apply_custom_rule').click();

        const nslookupBlockedResult = await runTerminalCommand(`nslookup ${domain_to_block} 127.0.0.1`).toString();

        console.info(`nslookup blocked CNAME result: '${nslookupBlockedResult}'`);

        const currentRules = await page.getByTestId('custom_rule_textarea').inputValue();
        console.debug(`Current rules before removal:\n${currentRules}`);

        if (currentRules.includes(domain_to_block)) {
            const updatedRules = currentRules
            .split('\n')
            .filter((line) => line.trim() !== domain_to_block.trim())
            .join('\n');

            await page.getByTestId('custom_rule_textarea').fill(updatedRules);
            console.info(`Rule '${domain_to_block}' removed successfully.`);

            console.info('Applying the updated filtering rules after removal.');
            await page.getByTestId('apply_custom_rule').click();

            await page.waitForLoadState('domcontentloaded');

            console.info(`Filtering rules successfully updated after removing '${domain_to_block}'.`);
        } else {
            console.warn(`Rule '${domain_to_block}' not found. No changes were made.`);
        }

        const nslookupUnblockedResult = await runTerminalCommand(`nslookup ${domain_to_block} 127.0.0.1`).toString();
        console.info(`nslookup unblocked CNAME result: '${nslookupUnblockedResult}'`);
    };

    test('Test blocking rule for apple.com', async ({ page }) => {
        await runCustomRuleTest(page, 'apple.com');
    });
});
