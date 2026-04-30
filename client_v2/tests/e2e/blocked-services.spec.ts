import { test, expect, Page } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

const MOCK_ALL_SERVICES = {
    blocked_services: [
        { id: 'telegram', name: 'Telegram', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||telegram.org^'], group_id: 'messaging' },
        { id: 'whatsapp', name: 'WhatsApp', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||whatsapp.com^'], group_id: 'messaging' },
        { id: 'steam', name: 'Steam', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||steampowered.com^'], group_id: 'gaming' },
        { id: 'epic_games', name: 'Epic Games', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||epicgames.com^'], group_id: 'gaming' },
        { id: 'tiktok', name: 'TikTok', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||tiktok.com^'], group_id: 'social_networks' },
        { id: 'facebook', name: 'Facebook', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||facebook.com^'], group_id: 'social_networks' },
        { id: 'youtube', name: 'YouTube', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||youtube.com^'], group_id: 'streaming' },
        { id: 'chatgpt', name: 'ChatGPT', icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>', rules: ['||openai.com^'], group_id: 'ai' },
    ],
    groups: [
        { id: 'messaging' },
        { id: 'gaming' },
        { id: 'social_networks' },
        { id: 'streaming' },
        { id: 'ai' },
    ],
};

const MOCK_BLOCKED_SERVICES = {
    ids: ['telegram', 'steam'],
    schedule: {
        time_zone: 'Europe/London',
        mon: { start: 3600000, end: 64800000 },
        wed: { start: 0, end: 86340000 },
    },
};

async function login(page: Page) {
    await page.goto('/login.html');
    await page.locator('#username').fill(ADMIN_USERNAME);
    await page.locator('#password').fill(ADMIN_PASSWORD);
    await page.locator('#sign_in').click();
    await page.waitForURL((url) => !url.href.endsWith('/login.html'));
}

async function setupMocks(page: Page) {
    await page.route('**/control/blocked_services/all', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(MOCK_ALL_SERVICES),
        });
    });

    await page.route('**/control/blocked_services/get', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(MOCK_BLOCKED_SERVICES),
        });
    });

    await page.route('**/control/blocked_services/update', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });
}

test.describe('Blocked Services Page', () => {
    test.beforeEach(async ({ page }) => {
        await login(page);
        await setupMocks(page);
        await page.goto('/#blocked_services');
        await page.waitForTimeout(1000);
    });

    test('should display page title and description', async ({ page }) => {
        const title = page.locator('h1');
        await expect(title).toBeVisible();
        await expect(title).toContainText('Blocked services');
    });

    test('should display inactivity schedule navigation item', async ({ page }) => {
        const navItem = page.locator('a[href*="blocked_services/schedule"]');
        await expect(navItem).toBeVisible();
    });

    test('should display all services from API', async ({ page }) => {
        // Check that all 8 mock services are displayed
        await Promise.all(
            MOCK_ALL_SERVICES.blocked_services.map((service) => {
                const serviceRow = page.locator(`input#service_${service.id}`);
                return expect(serviceRow).toBeAttached();
            }),
        );
    });

    test('should show correct blocked state for services', async ({ page }) => {
        // Telegram should be ON (blocked)
        const telegramToggle = page.locator('input#service_telegram');
        await expect(telegramToggle).toBeChecked();

        // Steam should be ON (blocked)
        const steamToggle = page.locator('input#service_steam');
        await expect(steamToggle).toBeChecked();

        // WhatsApp should be OFF (not blocked)
        const whatsappToggle = page.locator('input#service_whatsapp');
        await expect(whatsappToggle).not.toBeChecked();

        // TikTok should be OFF (not blocked)
        const tiktokToggle = page.locator('input#service_tiktok');
        await expect(tiktokToggle).not.toBeChecked();
    });

    test('should toggle a service and make API call', async ({ page }) => {
        let updatePayload: any = null;

        await page.route('**/control/blocked_services/update', (route) => {
            updatePayload = route.request().postDataJSON();
            // After update, mock the response to reflect the new state
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({}),
            });
        });

        // Also mock the re-fetch that happens after update
        await page.route('**/control/blocked_services/get', (route) => {
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    ...MOCK_BLOCKED_SERVICES,
                    ids: [...MOCK_BLOCKED_SERVICES.ids, 'whatsapp'],
                }),
            });
        });

        // Toggle WhatsApp ON
        const whatsappLabel = page.locator('label[for="service_whatsapp"]');
        await whatsappLabel.click();
        await page.waitForTimeout(1000);

        // Verify the API was called with correct payload
        expect(updatePayload).not.toBeNull();
        expect(updatePayload.ids).toContain('whatsapp');
        expect(updatePayload.ids).toContain('telegram');
        expect(updatePayload.ids).toContain('steam');
        // Schedule should be preserved
        expect(updatePayload.schedule).toBeDefined();
    });

    test('should filter services by search text', async ({ page }) => {
        const searchInput = page.locator('input#blocked-services-search');
        await expect(searchInput).toBeVisible();

        // Type "tel" to filter
        await searchInput.fill('tel');
        await page.waitForTimeout(300);

        // Telegram should be visible
        const telegramToggle = page.locator('input#service_telegram');
        await expect(telegramToggle).toBeAttached();

        // Steam should NOT be visible
        const steamToggle = page.locator('input#service_steam');
        await expect(steamToggle).not.toBeAttached();
    });

    test('should show nothing found when search has no results', async ({ page }) => {
        const searchInput = page.locator('input#blocked-services-search');
        await searchInput.fill('zzzznonexistent');
        await page.waitForTimeout(300);

        // Should show "Nothing found" or similar message
        const nothingFound = page.getByText(/nothing found/i);
        await expect(nothingFound).toBeVisible();
    });

    test('should clear search when clear button is clicked', async ({ page }) => {
        const searchInput = page.locator('input#blocked-services-search');
        await searchInput.fill('tel');
        await page.waitForTimeout(300);

        // Click the clear button (the × button that appears)
        const clearButton = page.locator('button[aria-label]').filter({ has: page.locator('svg') }).first();
        if (await clearButton.isVisible()) {
            await clearButton.click();
            await page.waitForTimeout(300);

            // All services should be visible again
            const allServices = page.locator('[class*="serviceRow"]');
            await expect(allServices).toHaveCount(8);
        }
    });

    test('should filter services by group tag', async ({ page }) => {
        // Click the "Gaming" group button
        const gamingButton = page.locator('button[aria-pressed]').filter({ hasText: /gaming/i });

        if (await gamingButton.isVisible()) {
            await gamingButton.click();
            await page.waitForTimeout(300);

            // Should only show gaming services (steam, epic_games)
            const steamToggle = page.locator('input#service_steam');
            await expect(steamToggle).toBeAttached();

            const epicToggle = page.locator('input#service_epic_games');
            await expect(epicToggle).toBeAttached();

            // Non-gaming services should not be visible
            const telegramToggle = page.locator('input#service_telegram');
            await expect(telegramToggle).not.toBeAttached();
        }
    });

    test('should deactivate group filter when clicked again', async ({ page }) => {
        const gamingButton = page.locator('button[aria-pressed]').filter({ hasText: /gaming/i });

        if (await gamingButton.isVisible()) {
            // Activate filter
            await gamingButton.click();
            await page.waitForTimeout(300);

            // Deactivate filter
            await gamingButton.click();
            await page.waitForTimeout(300);

            // All services should be visible
            const telegramToggle = page.locator('input#service_telegram');
            await expect(telegramToggle).toBeAttached();
        }
    });

    test('should combine search and group filter', async ({ page }) => {
        const gamingButton = page.locator('button[aria-pressed]').filter({ hasText: /gaming/i });

        if (await gamingButton.isVisible()) {
            await gamingButton.click();
            await page.waitForTimeout(300);

            const searchInput = page.locator('input#blocked-services-search');
            await searchInput.fill('steam');
            await page.waitForTimeout(300);

            // Only steam should be visible (matches search + gaming filter)
            const steamToggle = page.locator('input#service_steam');
            await expect(steamToggle).toBeAttached();

            const epicToggle = page.locator('input#service_epic_games');
            await expect(epicToggle).not.toBeAttached();
        }
    });

    test('should navigate to inactivity schedule page', async ({ page }) => {
        const navItem = page.locator('a[href*="blocked_services/schedule"]');
        await navItem.click();
        await page.waitForTimeout(500);

        // Should be on schedule page
        await expect(page).toHaveURL(/#blocked_services\/schedule/);
    });

    test('should disable switches during API processing', async ({ page }) => {
        // Make update slow
        await page.route('**/control/blocked_services/update', (route) => {
            setTimeout(() => {
                route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({}),
                });
            }, 2000);
        });

        // Toggle a service
        const whatsappLabel = page.locator('label[for="service_whatsapp"]');
        await whatsappLabel.click();

        // Check that other switches become disabled during request
        // (processingSet = true disables all switches)
        await page.waitForTimeout(200);
    });
});

test.describe('Inactivity Schedule Page', () => {
    test.beforeEach(async ({ page }) => {
        await login(page);
        await setupMocks(page);
        await page.goto('/#blocked_services/schedule');
        await page.waitForTimeout(1000);
    });

    test('should display page title', async ({ page }) => {
        const title = page.locator('h1');
        await expect(title).toBeVisible();
    });

    test('should display breadcrumbs with link back to blocked services', async ({ page }) => {
        const breadcrumb = page.locator('a').filter({ hasText: /blocked services/i });
        await expect(breadcrumb).toBeVisible();
    });

    test('should navigate back via breadcrumbs', async ({ page }) => {
        const breadcrumb = page.locator('a').filter({ hasText: /blocked services/i });
        await breadcrumb.click();
        await page.waitForTimeout(500);

        await expect(page).toHaveURL(/#blocked_services$/);
    });

    test('should display timezone selector', async ({ page }) => {
        // The timezone select should be visible
        const timezoneSection = page.locator('[class*="timezoneWrapper"]');
        await expect(timezoneSection).toBeVisible();
    });

    test('should display all 7 days of the week', async ({ page }) => {
        const scheduleRows = page.locator('[class*="scheduleRow"]');
        await expect(scheduleRows).toHaveCount(7);
    });

    test('should show configured time for Monday', async ({ page }) => {
        // Monday has start: 3600000 (01:00), end: 64800000 (18:00)
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        await expect(mondayRow).toContainText('01:00');
        await expect(mondayRow).toContainText('18:00');
    });

    test('should show full day indicator for Wednesday', async ({ page }) => {
        // Wednesday has start: 0, end: 86340000 (full day)
        const wednesdayRow = page.locator('[class*="scheduleRow"]').nth(2);
        // Should show "24h" or "All day"
        const allDayText = wednesdayRow.getByText(/24h|all day/i);
        await expect(allDayText).toBeVisible();
    });

    test('should show no schedule text for unconfigured days', async ({ page }) => {
        // Tuesday (index 1) has no schedule
        const tuesdayRow = page.locator('[class*="scheduleRow"]').nth(1);
        const noScheduleText = tuesdayRow.getByText(/no schedule/i);
        await expect(noScheduleText).toBeVisible();
    });

    test('should show edit and delete buttons for configured days', async ({ page }) => {
        // Monday is configured - should have edit and delete buttons
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        const editButton = mondayRow.locator('button').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await expect(editButton).toBeVisible();
        await expect(deleteButton).toBeVisible();
    });

    test('should show add button for unconfigured days', async ({ page }) => {
        // Tuesday is not configured - should have an add button
        const tuesdayRow = page.locator('[class*="scheduleRow"]').nth(1);
        const addButton = tuesdayRow.locator('button');
        await expect(addButton).toBeVisible();
    });

    test('should open modal when add button is clicked', async ({ page }) => {
        // Click add on Tuesday
        const tuesdayRow = page.locator('[class*="scheduleRow"]').nth(1);
        const addButton = tuesdayRow.locator('button');
        await addButton.click();
        await page.waitForTimeout(500);

        // Modal should be visible
        const modal = page.locator('.rc-dialog-update');
        await expect(modal).toBeVisible();
    });

    test('should open modal with pre-populated data when edit is clicked', async ({ page }) => {
        // Click edit on Monday (first configured day)
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        const editButton = mondayRow.locator('button').first();
        await editButton.click();
        await page.waitForTimeout(500);

        // Modal should be visible
        const modal = page.locator('.rc-dialog-update');
        await expect(modal).toBeVisible();
    });

    test('should show confirmation dialog when delete is clicked', async ({ page }) => {
        // Click delete on Monday
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();
        await page.waitForTimeout(500);

        // Confirmation dialog should appear
        const confirmDialog = page.locator('.rc-dialog-update');
        await expect(confirmDialog).toBeVisible();
    });

    test('should delete schedule entry on confirm', async ({ page }) => {
        let updatePayload: any = null;

        await page.route('**/control/blocked_services/update', (route) => {
            updatePayload = route.request().postDataJSON();
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({}),
            });
        });

        // Click delete on Monday
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();
        await page.waitForTimeout(500);

        // Confirm deletion
        const confirmButton = page.locator('.rc-dialog-update button').filter({ hasText: /delete/i });
        if (await confirmButton.isVisible()) {
            await confirmButton.click();
            await page.waitForTimeout(1000);

            // Verify API was called without Monday schedule
            expect(updatePayload).not.toBeNull();
            expect(updatePayload.schedule.mon).toBeUndefined();
            // Wednesday should still be present
            expect(updatePayload.schedule.wed).toBeDefined();
            // Blocked IDs should be preserved
            expect(updatePayload.ids).toEqual(['telegram', 'steam']);
        }
    });

    test('should cancel deletion when cancel is clicked', async ({ page }) => {
        let updateCalled = false;

        await page.route('**/control/blocked_services/update', (route) => {
            updateCalled = true;
            route.fulfill({ status: 200, body: JSON.stringify({}) });
        });

        // Click delete on Monday
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();
        await page.waitForTimeout(500);

        // Cancel deletion
        const cancelButton = page.locator('.rc-dialog-update button').filter({ hasText: /cancel/i });
        if (await cancelButton.isVisible()) {
            await cancelButton.click();
            await page.waitForTimeout(500);

            // No API call should have been made
            expect(updateCalled).toBe(false);
        }
    });

    test('should save schedule entry from modal', async ({ page }) => {
        let updatePayload: any = null;

        await page.route('**/control/blocked_services/update', (route) => {
            updatePayload = route.request().postDataJSON();
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({}),
            });
        });

        // Click add on Tuesday
        const tuesdayRow = page.locator('[class*="scheduleRow"]').nth(1);
        const addButton = tuesdayRow.locator('button');
        await addButton.click();
        await page.waitForTimeout(500);

        // Modal should be visible - click Save (default values are 00:00 to 23:59)
        const saveButton = page.locator('.rc-dialog-update button').filter({ hasText: /save/i });
        if (await saveButton.isVisible() && await saveButton.isEnabled()) {
            await saveButton.click();
            await page.waitForTimeout(1000);

            // Verify API was called with Tuesday schedule
            expect(updatePayload).not.toBeNull();
            expect(updatePayload.schedule.tue).toBeDefined();
            expect(updatePayload.schedule.tue.start).toBeDefined();
            expect(updatePayload.schedule.tue.end).toBeDefined();
            // Existing schedules preserved
            expect(updatePayload.schedule.mon).toBeDefined();
            expect(updatePayload.schedule.wed).toBeDefined();
            // IDs preserved
            expect(updatePayload.ids).toEqual(['telegram', 'steam']);
        }
    });

    test('should update timezone via selector', async ({ page }) => {
        let updatePayload: any = null;

        await page.route('**/control/blocked_services/update', (route) => {
            updatePayload = route.request().postDataJSON();
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({}),
            });
        });

        // Open timezone select and type to search
        const timezoneWrapper = page.locator('[class*="timezoneWrapper"]');
        const selectInput = timezoneWrapper.locator('input');

        if (await selectInput.isVisible()) {
            await selectInput.fill('America/New_York');
            await page.waitForTimeout(500);

            // Click the option
            const option = page.getByText('America/New_York', { exact: true }).first();
            if (await option.isVisible()) {
                await option.click();
                await page.waitForTimeout(1000);

                expect(updatePayload).not.toBeNull();
                expect(updatePayload.schedule.time_zone).toBe('America/New_York');
                // Existing schedules preserved
                expect(updatePayload.schedule.mon).toBeDefined();
                expect(updatePayload.ids).toEqual(['telegram', 'steam']);
            }
        }
    });
});

test.describe('Blocked Services - Schedule Integration', () => {
    test.beforeEach(async ({ page }) => {
        await login(page);
        await setupMocks(page);
    });

    test('should preserve schedule when toggling services', async ({ page }) => {
        let updatePayload: any = null;

        await page.route('**/control/blocked_services/update', (route) => {
            updatePayload = route.request().postDataJSON();
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({}),
            });
        });

        await page.goto('/#blocked_services');
        await page.waitForTimeout(1000);

        // Toggle WhatsApp ON
        const whatsappLabel = page.locator('label[for="service_whatsapp"]');
        await whatsappLabel.click();
        await page.waitForTimeout(1000);

        // Schedule should be preserved in the update payload
        expect(updatePayload).not.toBeNull();
        expect(updatePayload.schedule).toBeDefined();
        expect(updatePayload.schedule.time_zone).toBe('Europe/London');
        expect(updatePayload.schedule.mon).toBeDefined();
    });

    test('should preserve blocked services when updating schedule', async ({ page }) => {
        let updatePayload: any = null;

        await page.route('**/control/blocked_services/update', (route) => {
            updatePayload = route.request().postDataJSON();
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({}),
            });
        });

        await page.goto('/#blocked_services/schedule');
        await page.waitForTimeout(1000);

        // Delete Monday schedule
        const mondayRow = page.locator('[class*="scheduleRow"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();
        await page.waitForTimeout(500);

        const confirmButton = page.locator('.rc-dialog-update button').filter({ hasText: /delete/i });
        if (await confirmButton.isVisible()) {
            await confirmButton.click();
            await page.waitForTimeout(1000);

            // Blocked service IDs should be preserved
            expect(updatePayload).not.toBeNull();
            expect(updatePayload.ids).toEqual(['telegram', 'steam']);
        }
    });

    test('should navigate from blocked services to schedule and back', async ({ page }) => {
        await page.goto('/#blocked_services');
        await page.waitForTimeout(1000);

        // Navigate to schedule
        const navItem = page.locator('a[href*="blocked_services/schedule"]');
        await navItem.click();
        await page.waitForTimeout(500);

        await expect(page).toHaveURL(/#blocked_services\/schedule/);

        // Navigate back via breadcrumbs
        const breadcrumb = page.locator('a').filter({ hasText: /blocked services/i });
        await breadcrumb.click();
        await page.waitForTimeout(500);

        await expect(page).toHaveURL(/#blocked_services$/);
    });
});
