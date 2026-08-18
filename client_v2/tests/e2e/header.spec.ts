import { test, expect, type Page } from '@playwright/test';

import { login } from '../helpers/login';

const VIEWPORTS = [
    { name: 'mobile', width: 390, height: 844 },
    { name: 'tablet', width: 1024, height: 600 },
] as const;

async function ensureScrollablePage(page: Page) {
    await page.evaluate(() => {
        const existing = document.getElementById('e2e-header-scroll-spacer');
        if (existing) {
            return;
        }

        const spacer = document.createElement('div');
        spacer.id = 'e2e-header-scroll-spacer';
        spacer.style.height = '2000px';
        document.body.appendChild(spacer);
    });
}

async function expectHeaderStickyAtTop(page: Page) {
    const header = page.locator('#header');

    await expect(header).toBeVisible();

    await ensureScrollablePage(page);
    await page.evaluate(() => window.scrollTo(0, 800));

    await expect
        .poll(async () => {
            const box = await header.boundingBox();
            return box?.y ?? -1;
        })
        .toBe(0);
}

test.describe('Header sticky behavior', () => {
    for (const viewport of VIEWPORTS) {
        test(`stays at the top while scrolling on ${viewport.name}`, async ({ page }) => {
            await page.setViewportSize({
                width: viewport.width,
                height: viewport.height,
            });
            await login(page);
            await page.goto('/#');
            await expectHeaderStickyAtTop(page);
        });
    }
});
