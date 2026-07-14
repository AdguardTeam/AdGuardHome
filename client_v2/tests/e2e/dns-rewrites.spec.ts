import { test, expect, type Page } from '@playwright/test';

import { login } from '../helpers/login';

type RewriteEntry = {
    domain: string;
    answer: string;
    enabled?: boolean;
};

const ADD_REWRITE: RewriteEntry = {
    domain: 'e2e-add-example.org',
    answer: '192.168.1.100',
};
const TOGGLE_REWRITE: RewriteEntry = {
    domain: 'e2e-toggle-example.org',
    answer: '192.168.1.101',
    enabled: true,
};
const SOURCE_REWRITE: RewriteEntry = {
    domain: 'e2e-update-source-example.org',
    answer: '192.168.1.102',
    enabled: true,
};
const UPDATED_REWRITE: RewriteEntry = {
    domain: 'e2e-update-target-example.org',
    answer: '192.168.1.200',
    enabled: true,
};
const DELETE_REWRITE: RewriteEntry = {
    domain: 'e2e-delete-example.org',
    answer: '192.168.1.103',
    enabled: true,
};

const openDnsRewritesPage = async (page: Page) => {
    await page.goto('/#dns_rewrites');
    await expect(page.getByText('DNS rewrites', { exact: true })).toBeVisible();
};

const listRewrites = async (page: Page): Promise<RewriteEntry[]> => {
    const response = await page.request.get('/control/rewrite/list');
    expect(response.ok()).toBeTruthy();

    return response.json();
};

const removeRewriteIfPresent = async (page: Page, domain: string) => {
    const rewrites = await listRewrites(page);

    for (const rewrite of rewrites.filter((item) => item.domain === domain)) {
        const response = await page.request.post('/control/rewrite/delete', {
            data: rewrite,
        });

        expect(response.ok()).toBeTruthy();
    }
};

const addRewriteViaApi = async (page: Page, rewrite: RewriteEntry) => {
    await removeRewriteIfPresent(page, rewrite.domain);

    const response = await page.request.post('/control/rewrite/add', {
        data: rewrite,
    });

    expect(response.ok()).toBeTruthy();
};

const expectRewriteVisible = async (page: Page, rewrite: RewriteEntry) => {
    await expect(page.getByText(rewrite.domain, { exact: true }).first()).toBeVisible({
        timeout: 15_000,
    });
    await expect(page.getByText(rewrite.answer, { exact: true }).first()).toBeVisible({
        timeout: 15_000,
    });
};

const toggleGlobalRewrite = async (page: Page, targetEnabled: boolean) => {
    const globalToggleInput = page.locator('input#rewrite_global_enabled');
    const globalToggleLabel = page.locator('label[for="rewrite_global_enabled"]');

    await expect(globalToggleInput).toBeAttached();
    await expect(globalToggleLabel).toBeVisible();

    if ((await globalToggleInput.isChecked()) === targetEnabled) {
        return;
    }

    await globalToggleLabel.click();

    const confirmButton = page.getByTestId(
        targetEnabled ? 'confirm-enable-rewrites' : 'confirm-disable-rewrites',
    );
    await expect(confirmButton).toBeVisible({ timeout: 10_000 });
    await confirmButton.click();

    if (targetEnabled) {
        await expect(globalToggleInput).toBeChecked();
    } else {
        await expect(globalToggleInput).not.toBeChecked();
    }
};

test.describe('DNS Rewrites', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test.beforeEach(async ({ page }) => {
        await login(page);
    });

    test('should add a new DNS rewrite', async ({ page }) => {
        await removeRewriteIfPresent(page, ADD_REWRITE.domain);
        await openDnsRewritesPage(page);

        const addButton = page.getByTestId('add-rewrite');
        await expect(addButton).toBeVisible();
        await addButton.click();

        const domainInput = page.locator('input#domain');
        const answerInput = page.locator('input#answer');

        await expect(domainInput).toBeVisible({ timeout: 10000 });
        await expect(answerInput).toBeVisible();

        await domainInput.fill(ADD_REWRITE.domain);
        await answerInput.fill(ADD_REWRITE.answer);

        const saveButton = page.locator('button#save');
        await expect(saveButton).toBeVisible();
        await saveButton.click();

        await page
            .waitForSelector('button#save', { state: 'hidden', timeout: 5000 })
            .catch(() => {});

        await expectRewriteVisible(page, ADD_REWRITE);
    });

    test('should toggle global rewrite switch in table header', async ({ page }) => {
        await openDnsRewritesPage(page);

        const globalToggleInput = page.locator('input#rewrite_global_enabled');
        await expect(globalToggleInput).toBeAttached();

        const initialState = await globalToggleInput.isChecked();

        await toggleGlobalRewrite(page, !initialState);
        await toggleGlobalRewrite(page, initialState);
    });

    test('should toggle individual rewrite', async ({ page }) => {
        await addRewriteViaApi(page, TOGGLE_REWRITE);
        await openDnsRewritesPage(page);

        const toggleId = `rewrite_${TOGGLE_REWRITE.domain}`;
        const individualToggleInput = page.locator(`input[id="${toggleId}"]`);
        const individualToggleLabel = page.locator(`label[for="${toggleId}"]`);

        await expect(individualToggleInput).toBeAttached();
        await expect(individualToggleLabel).toBeVisible();

        const initialState = await individualToggleInput.isChecked();

        await individualToggleLabel.click();
        await expect(individualToggleInput).not.toBeChecked();

        await individualToggleLabel.click();
        if (initialState) {
            await expect(individualToggleInput).toBeChecked();
        } else {
            await expect(individualToggleInput).not.toBeChecked();
        }
    });

    test('should update rewrite through ConfigureRewritesModal', async ({ page }) => {
        await removeRewriteIfPresent(page, UPDATED_REWRITE.domain);
        await addRewriteViaApi(page, SOURCE_REWRITE);
        await openDnsRewritesPage(page);

        const editButton = page.getByTestId(`edit-rewrite-${SOURCE_REWRITE.domain}`);

        await expect(editButton).toBeVisible({ timeout: 10_000 });
        await editButton.click();

        const domainInput = page.locator('input#domain');
        const answerInput = page.locator('input#answer');

        await expect(domainInput).toBeVisible({ timeout: 10000 });
        await expect(answerInput).toBeVisible();

        await domainInput.clear();
        await domainInput.fill(UPDATED_REWRITE.domain);

        await answerInput.clear();
        await answerInput.fill(UPDATED_REWRITE.answer);

        const saveButton = page.locator('button#save');
        await expect(saveButton).toBeVisible();
        await saveButton.click();

        await page
            .waitForSelector('button#save', { state: 'hidden', timeout: 5000 })
            .catch(() => {});

        await expectRewriteVisible(page, UPDATED_REWRITE);
    });

    test('should delete rewrite', async ({ page }) => {
        await addRewriteViaApi(page, DELETE_REWRITE);
        await openDnsRewritesPage(page);

        const deleteButton = page.getByTestId(`delete-rewrite-${DELETE_REWRITE.domain}`);
        await expect(deleteButton).toBeVisible({ timeout: 10_000 });
        await deleteButton.click();

        const confirmButton = page.getByRole('button', { name: /remove|delete/i });
        await expect(confirmButton).toBeVisible({ timeout: 10000 });
        await confirmButton.click();

        await expect(page.getByText(DELETE_REWRITE.domain, { exact: true })).not.toBeVisible({
            timeout: 15_000,
        });
    });
});
