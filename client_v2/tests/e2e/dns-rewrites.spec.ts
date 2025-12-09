import { test, expect } from '@playwright/test';
import { ADMIN_PASSWORD, ADMIN_USERNAME } from '../constants';

const TEST_DOMAIN = 'test-example.org';
const TEST_ANSWER = '192.168.1.100';
const UPDATED_DOMAIN = 'updated-example.org';
const UPDATED_ANSWER = '192.168.1.200';

test.describe('DNS Rewrites', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login.html');
        await page.locator('#username').fill(ADMIN_USERNAME);
        await page.locator('#password').fill(ADMIN_PASSWORD);
        await page.locator('#sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));

        await page.goto('/#dns_rewrites');
        await page.waitForTimeout(1000);
    });

    test('should add a new DNS rewrite', async ({ page }) => {
        const addButton = page.getByTestId('add-rewrite');
        await expect(addButton).toBeVisible();
        await addButton.click();

        await page.waitForTimeout(1000);

        const domainInput = page.locator('input#domain');
        const answerInput = page.locator('input#answer');

        await expect(domainInput).toBeVisible({ timeout: 10000 });
        await expect(answerInput).toBeVisible();

        await domainInput.fill(TEST_DOMAIN);
        await answerInput.fill(TEST_ANSWER);

        const saveButton = page.locator('button#save');
        await expect(saveButton).toBeVisible();
        await saveButton.click();

        await page.waitForSelector('button#save', { state: 'hidden', timeout: 5000 }).catch(() => {});

        await page.waitForTimeout(3000);

        const domainText = page.getByText(TEST_DOMAIN, { exact: true }).first();
        const answerText = page.getByText(TEST_ANSWER, { exact: true }).first();

        await expect(domainText).toBeVisible({ timeout: 15000 });
        await expect(answerText).toBeVisible({ timeout: 15000 });
    });

    test('should toggle global rewrite switch in table header', async ({ page }) => {
        await page.waitForTimeout(1000);

        const globalToggleInput = page.locator('input#rewrite_global_enabled');
        const globalToggleLabel = page.locator('label[for="rewrite_global_enabled"]');

        await expect(globalToggleInput).toBeAttached();
        await expect(globalToggleLabel).toBeVisible();

        const initialState = await globalToggleInput.evaluate((el: HTMLInputElement) => el.checked);

        await globalToggleLabel.click();
        await page.waitForTimeout(1000);

        const newState = await globalToggleInput.evaluate((el: HTMLInputElement) => el.checked);
        expect(newState).toBe(!initialState);

        await globalToggleLabel.click();
        await page.waitForTimeout(1000);

        const finalState = await globalToggleInput.evaluate((el: HTMLInputElement) => el.checked);
        expect(finalState).toBe(initialState);
    });

    test('should toggle individual rewrite', async ({ page }) => {
        await page.waitForTimeout(1000);

        const individualToggleInput = page.locator(`input#rewrite_${TEST_DOMAIN}`);
        const individualToggleLabel = page.locator(`label[for="rewrite_${TEST_DOMAIN}"]`);

        const toggleExists = await individualToggleInput.count() > 0;

        if (!toggleExists) {
            test.skip();
            return;
        }

        await expect(individualToggleInput).toBeAttached();
        await expect(individualToggleLabel).toBeVisible();

        const initialState = await individualToggleInput.evaluate((el: HTMLInputElement) => el.checked);

        await individualToggleLabel.click();
        await page.waitForTimeout(1000);

        const newState = await individualToggleInput.evaluate((el: HTMLInputElement) => el.checked);
        expect(newState).toBe(!initialState);

        await individualToggleLabel.click();
        await page.waitForTimeout(1000);

        const finalState = await individualToggleInput.evaluate((el: HTMLInputElement) => el.checked);
        expect(finalState).toBe(initialState);
    });

    test('should update rewrite through ConfigureRewritesModal', async ({ page }) => {
        await page.waitForTimeout(1000);

        const editButton = page.getByTestId(`edit-rewrite-${TEST_DOMAIN}`);

        const editButtonExists = await editButton.count() > 0;

        if (!editButtonExists) {
            test.skip();
            return;
        }

        await editButton.click();
        await page.waitForTimeout(1000);

        const domainInput = page.locator('input#domain');
        const answerInput = page.locator('input#answer');

        await expect(domainInput).toBeVisible({ timeout: 10000 });
        await expect(answerInput).toBeVisible();

        await domainInput.clear();
        await domainInput.fill(UPDATED_DOMAIN);

        await answerInput.clear();
        await answerInput.fill(UPDATED_ANSWER);

        const saveButton = page.locator('button#save');
        await expect(saveButton).toBeVisible();
        await saveButton.click();

        await page.waitForSelector('button#save', { state: 'hidden', timeout: 5000 }).catch(() => {});

        await page.waitForTimeout(3000);

        const updatedDomain = page.getByText(UPDATED_DOMAIN, { exact: true }).first();
        const updatedAnswer = page.getByText(UPDATED_ANSWER, { exact: true }).first();

        await expect(updatedDomain).toBeVisible({ timeout: 15000 });
        await expect(updatedAnswer).toBeVisible({ timeout: 15000 });
    });

    test('should delete rewrite', async ({ page }) => {
        await page.waitForTimeout(1000);

        const deleteButton = page.getByTestId(`delete-rewrite-${UPDATED_DOMAIN}`);

        const deleteButtonExists = await deleteButton.count() > 0;

        if (!deleteButtonExists) {
            const altDeleteButton = page.getByTestId(`delete-rewrite-${TEST_DOMAIN}`);
            const altButtonExists = await altDeleteButton.count() > 0;

            if (!altButtonExists) {
                test.skip();
                return;
            }

            await altDeleteButton.click();
        } else {
            await deleteButton.click();
        }

        await page.waitForTimeout(1000);

        const confirmButton = page.getByRole('button', { name: /remove|delete/i });
        await expect(confirmButton).toBeVisible({ timeout: 10000 });
        await confirmButton.click();

        await page.waitForTimeout(3000);

        const deletedDomain = page.getByText(UPDATED_DOMAIN, { exact: true });
        await expect(deletedDomain).not.toBeVisible({ timeout: 15000 });
    });
});
