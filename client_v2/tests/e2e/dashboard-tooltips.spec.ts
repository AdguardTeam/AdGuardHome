import { test, expect } from '@playwright/test';

import { login } from '../helpers/login';

const TRACKER_NAME = 'Google Marketing';

const STATS = {
    dns_queries: [],
    blocked_filtering: [],
    avg_processing_time: 0,
    top_clients: [],
    top_queried_domains: [],
    top_blocked_domains: [{ 'doubleclick.net': 10 }, { 'facebook.com': 5 }],
    top_upstreams_responses: [],
    top_upstreams_avg_time: [],
    stats_period: { interval: 86400000 },
    num_dns_queries: 10,
    num_blocked_filtering: 15,
    num_replaced_safebrowsing: 0,
    num_replaced_safesearch: 0,
    num_replaced_parental: 0,
};

test.describe('Dashboard tracker tooltips', () => {
    test.beforeEach(async ({ page }) => {
        await login(page);

        await page.route(/\/control\/stats\?/, (route) =>
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(STATS),
            }),
        );

        await page.goto('/#dashboard');
        await expect(
            page.getByRole('link', { name: /^doubleclick\.net / }).first(),
        ).toBeVisible({
            timeout: 15000,
        });
    });

    test('keeps the card open while the pointer crosses the gutter', async ({ page }) => {
        // The row link is the whole trigger; its text also appears inside the
        // hidden card, so match the accessible name of the row.
        const firstRow = page.getByRole('link', { name: /^doubleclick\.net / }).first();

        await firstRow.hover();

        const card = page.locator('[data-part="content"]:visible');
        await expect(card).toHaveCount(1);
        await expect(card).toContainText(TRACKER_NAME);

        const rowBox = await firstRow.boundingBox();
        const cardBox = await card.boundingBox();
        expect(rowBox).not.toBeNull();
        expect(cardBox).not.toBeNull();

        if (!rowBox || !cardBox) {
            return;
        }

        // The card sits 4px below the row; cross that strip in small steps so
        // the adjacent row trigger gets a chance to steal the hover.
        for (let step = 1; step <= 4; step += 1) {
            await page.mouse.move(rowBox.x + rowBox.width / 2, rowBox.y + rowBox.height + step);
        }
        for (let step = 1; step <= 4; step += 1) {
            await page.mouse.move(
                cardBox.x + cardBox.width / 2,
                cardBox.y + (cardBox.height * step) / 4,
            );
        }

        await expect(card).toHaveCount(1);
        await expect(card).toContainText(TRACKER_NAME);
        await expect(card).toContainText('doubleclick.net');

        const nameLink = card.getByRole('link', { name: TRACKER_NAME });
        const sourceLink = card.getByRole('link', { name: 'AdGuard' });

        await expect(nameLink).toHaveAttribute(
            'href',
            'https://marketingplatform.google.com/about/enterprise',
        );
        await expect(sourceLink).toHaveAttribute('href', /^https:\/\/github\.com\//);

        const nameIsReachable = await nameLink.evaluate((el) => {
            const rect = el.getBoundingClientRect();
            const atPoint = document.elementFromPoint(
                rect.x + rect.width / 2,
                rect.y + rect.height / 2,
            );

            return atPoint === el || el.contains(atPoint);
        });

        expect(nameIsReachable).toBe(true);

        // The adjacent row still gets its own card once the pointer leaves.
        await page.getByRole('link', { name: /^facebook\.com / }).first().hover();
        await expect(
            page.locator('[data-part="content"]:visible').filter({ hasText: 'Facebook' }).first(),
        ).toBeVisible();
    });
});
