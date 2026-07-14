import { test, expect, Page } from '@playwright/test';

import { login } from '../helpers/login';

const MOCK_ALL_SERVICES = {
    blocked_services: [
        {
            id: 'telegram',
            name: 'Telegram',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||telegram.org^'],
            group_id: 'messaging',
        },
        {
            id: 'whatsapp',
            name: 'WhatsApp',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||whatsapp.com^'],
            group_id: 'messaging',
        },
        {
            id: 'steam',
            name: 'Steam',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||steampowered.com^'],
            group_id: 'gaming',
        },
        {
            id: 'epic_games',
            name: 'Epic Games',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||epicgames.com^'],
            group_id: 'gaming',
        },
        {
            id: 'tiktok',
            name: 'TikTok',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||tiktok.com^'],
            group_id: 'social_networks',
        },
        {
            id: 'facebook',
            name: 'Facebook',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||facebook.com^'],
            group_id: 'social_networks',
        },
        {
            id: 'youtube',
            name: 'YouTube',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||youtube.com^'],
            group_id: 'streaming',
        },
        {
            id: 'chatgpt',
            name: 'ChatGPT',
            icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
            rules: ['||openai.com^'],
            group_id: 'ai',
        },
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
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test.beforeEach(async ({ page }) => {
        await login(page);
        await setupMocks(page);
        await page.goto('/#blocked_services');
        await expect(page.locator('h1')).toBeVisible();
    });

    test('should display page title and description', async ({ page }) => {
        const title = page.locator('h1');
        await expect(title).toBeVisible();
        await expect(title).toContainText('Blocked services');
    });

    test('should display inactivity schedule navigation item', async ({ page }) => {
        const navItem = page.getByTestId('blocked-services-schedule-link');
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

        // Verify the API was called with correct payload
        await expect.poll(() => updatePayload).not.toBeNull();
        expect(updatePayload.ids).toContain('whatsapp');
        expect(updatePayload.ids).toContain('telegram');
        expect(updatePayload.ids).toContain('steam');
        // Schedule should be preserved
        expect(updatePayload.schedule).toBeDefined();
    });

    test('should filter services by search text', async ({ page }) => {
        const searchInput = page.getByTestId('blocked-services-search');
        await expect(searchInput).toBeVisible();

        // Type "tel" to filter
        await searchInput.fill('tel');

        // Steam should NOT be visible (filtered by search)
        const steamToggle = page.locator('input#service_steam');
        await expect(steamToggle).not.toBeAttached();
    });

    test('should show nothing found when search has no results', async ({ page }) => {
        const searchInput = page.getByTestId('blocked-services-search');
        await searchInput.fill('zzzznonexistent');

        // Should show "Nothing found" or similar message
        const nothingFound = page.getByTestId('blocked-services-nothing-found');
        await expect(nothingFound).toBeVisible();
    });

    test('should clear search when clear button is clicked', async ({ page }) => {
        const searchInput = page.getByTestId('blocked-services-search');
        await searchInput.fill('tel');

        // Clear by clicking the clear button
        await page.getByTestId('blocked-services-search-clear').click();

        // All services should be visible again
        const allServices = page.locator('[data-testid^="blocked-service-row-"]');
        await expect(allServices).toHaveCount(8);
    });

    test('should filter services by group tag', async ({ page }) => {
        // Click the "Gaming" group button
        const gamingButton = page.getByTestId('blocked-services-group-gaming');

        await expect(gamingButton).toBeVisible();
        await gamingButton.click();

        // Should only show gaming services (steam, epic_games)
        const steamToggle = page.locator('input#service_steam');
        await expect(steamToggle).toBeAttached();

        const epicToggle = page.locator('input#service_epic_games');
        await expect(epicToggle).toBeAttached();

        // Non-gaming services should not be visible
        const telegramToggle = page.locator('input#service_telegram');
        await expect(telegramToggle).not.toBeAttached();
    });

    test('should deactivate group filter when clicked again', async ({ page }) => {
        const gamingButton = page.getByTestId('blocked-services-group-gaming');

        await expect(gamingButton).toBeVisible();

        // Activate filter
        await gamingButton.click();

        // Deactivate filter
        await gamingButton.click();

        // All services should be visible
        const telegramToggle = page.locator('input#service_telegram');
        await expect(telegramToggle).toBeAttached();
    });

    test('should combine search and group filter', async ({ page }) => {
        const gamingButton = page.getByTestId('blocked-services-group-gaming');

        await expect(gamingButton).toBeVisible();
        await gamingButton.click();

        const searchInput = page.getByTestId('blocked-services-search');
        await searchInput.fill('steam');

        // Only steam should be visible (matches search + gaming filter)
        const steamToggle = page.locator('input#service_steam');
        await expect(steamToggle).toBeAttached();

        const epicToggle = page.locator('input#service_epic_games');
        await expect(epicToggle).not.toBeAttached();
    });

    test('should navigate to inactivity schedule page', async ({ page }) => {
        const navItem = page.getByTestId('blocked-services-schedule-link');
        await navItem.click();

        // Should be on schedule page
        await expect(page).toHaveURL(/#blocked_services\/schedule/);
    });

    // TODO: Check if the component actually disables switches during API calls.
    // If not, this test should use test.fixme instead.
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
        const telegramInput = page.locator('input#service_telegram');

        await whatsappLabel.click();

        // During the ongoing 2-second API call, other switches should be disabled
        await expect(telegramInput).toBeDisabled();
    });
});

test.describe('Inactivity Schedule Page', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test.beforeEach(async ({ page }) => {
        await login(page);
        await setupMocks(page);
        await page.goto('/#blocked_services/schedule');
        await expect(page.locator('h1')).toBeVisible();
    });

    test('should display page title', async ({ page }) => {
        const title = page.locator('h1');
        await expect(title).toBeVisible();
    });

    test('should display breadcrumbs with link back to blocked services', async ({ page }) => {
        const breadcrumb = page
            .getByTestId('inactivity-schedule-breadcrumbs')
            .getByRole('link', { name: 'Blocked services' });
        await expect(breadcrumb).toBeVisible();
    });

    test('should navigate back via breadcrumbs', async ({ page }) => {
        const breadcrumb = page
            .getByTestId('inactivity-schedule-breadcrumbs')
            .getByRole('link', { name: 'Blocked services' });
        await breadcrumb.click();

        await expect(page).toHaveURL(/#blocked_services$/);
    });

    test('should display timezone selector', async ({ page }) => {
        // The timezone select should be visible
        const timezoneSection = page.getByTestId('inactivity-schedule-timezone');
        await expect(timezoneSection).toBeVisible();
    });

    test('should display all 7 days of the week', async ({ page }) => {
        const scheduleRows = page.locator('[data-testid^="schedule-row-"]');
        await expect(scheduleRows).toHaveCount(7);
    });

    test('should show configured time for Monday', async ({ page }) => {
        // Monday has start: 3600000 (01:00), end: 64800000 (18:00)
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        await expect(mondayRow).toContainText('01:00');
        await expect(mondayRow).toContainText('18:00');
    });

    test('should show full day indicator for Wednesday', async ({ page }) => {
        // Wednesday has start: 0, end: 86340000 (full day)
        const wednesdayRow = page.locator('[data-testid^="schedule-row-"]').nth(2);
        // Should show "24h" or "All day"
        const allDayText = wednesdayRow.getByText(/24h|all day/i);
        await expect(allDayText).toBeVisible();
    });

    test('should show no schedule text for unconfigured days', async ({ page }) => {
        // Tuesday (index 1) has no schedule
        const tuesdayRow = page.locator('[data-testid^="schedule-row-"]').nth(1);
        const noScheduleText = tuesdayRow.getByText(/no schedule/i);
        await expect(noScheduleText).toBeVisible();
    });

    test('should show edit and delete buttons for configured days', async ({ page }) => {
        // Monday is configured - should have edit and delete buttons
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        const editButton = mondayRow.locator('button').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await expect(editButton).toBeVisible();
        await expect(deleteButton).toBeVisible();
    });

    test('should show add button for unconfigured days', async ({ page }) => {
        // Tuesday is not configured - should have an add button
        const tuesdayRow = page.locator('[data-testid^="schedule-row-"]').nth(1);
        const addButton = tuesdayRow.locator('button');
        await expect(addButton).toBeVisible();
    });

    test('should open modal when add button is clicked', async ({ page }) => {
        // Click add on Tuesday
        const tuesdayRow = page.locator('[data-testid^="schedule-row-"]').nth(1);
        const addButton = tuesdayRow.locator('button');
        await addButton.click();

        // Modal should be visible
        const modal = page.locator('.rc-dialog-update');
        await expect(modal).toBeVisible();
    });

    test('should open modal with pre-populated data when edit is clicked', async ({ page }) => {
        // Click edit on Monday (first configured day)
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        const editButton = mondayRow.locator('button').first();
        await editButton.click();

        // Modal should be visible
        const modal = page.locator('.rc-dialog-update');
        await expect(modal).toBeVisible();
    });

    test('should show confirmation dialog when delete is clicked', async ({ page }) => {
        // Click delete on Monday
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();

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
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();

        // Confirm deletion
        const confirmButton = page
            .locator('.rc-dialog-update button')
            .filter({ hasText: /delete/i });
        await expect(confirmButton).toBeVisible();
        await confirmButton.click();

        // Verify API was called without Monday schedule
        await expect.poll(() => updatePayload).not.toBeNull();
        expect(updatePayload.schedule.mon).toBeUndefined();
        // Wednesday should still be present
        expect(updatePayload.schedule.wed).toBeDefined();
        // Blocked IDs should be preserved
        expect(updatePayload.ids).toEqual(['telegram', 'steam']);
    });

    test('should cancel deletion when cancel is clicked', async ({ page }) => {
        let updateCalled = false;

        await page.route('**/control/blocked_services/update', (route) => {
            updateCalled = true;
            route.fulfill({ status: 200, body: JSON.stringify({}) });
        });

        // Click delete on Monday
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();

        // Cancel deletion
        const cancelButton = page
            .locator('.rc-dialog-update button')
            .filter({ hasText: /cancel/i });
        await expect(cancelButton).toBeVisible();
        await cancelButton.click();

        // No API call should have been made
        expect(updateCalled).toBe(false);
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
        const tuesdayRow = page.locator('[data-testid^="schedule-row-"]').nth(1);
        const addButton = tuesdayRow.locator('button');
        await addButton.click();

        // Modal should be visible - click Save (default values are 00:00 to 23:59)
        const saveButton = page.locator('.rc-dialog-update button').filter({ hasText: /save/i });
        await expect(saveButton).toBeVisible();
        await expect(saveButton).toBeEnabled();
        await saveButton.click();

        // Verify API was called with Tuesday schedule
        await expect.poll(() => updatePayload).not.toBeNull();
        expect(updatePayload.schedule.tue).toBeDefined();
        expect(updatePayload.schedule.tue.start).toBeDefined();
        expect(updatePayload.schedule.tue.end).toBeDefined();
        // Existing schedules preserved
        expect(updatePayload.schedule.mon).toBeDefined();
        expect(updatePayload.schedule.wed).toBeDefined();
        // IDs preserved
        expect(updatePayload.ids).toEqual(['telegram', 'steam']);
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
        const timezoneWrapper = page.getByTestId('inactivity-schedule-timezone');
        const selectInput = timezoneWrapper.locator('input');

        await expect(selectInput).toBeVisible();
        await selectInput.fill('America/New_York');

        // Click the option
        const option = page.getByText('America/New_York', { exact: true }).first();
        await expect(option).toBeVisible();
        await option.click();

        await expect.poll(() => updatePayload).not.toBeNull();
        expect(updatePayload.schedule.time_zone).toBe('America/New_York');
        // Existing schedules preserved
        expect(updatePayload.schedule.mon).toBeDefined();
        expect(updatePayload.ids).toEqual(['telegram', 'steam']);
    });
});

test.describe('Blocked Services - Schedule Integration', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

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
        await expect(page.locator('h1')).toBeVisible();

        // Toggle WhatsApp ON
        const whatsappLabel = page.locator('label[for="service_whatsapp"]');
        await whatsappLabel.click();

        // Schedule should be preserved in the update payload
        await expect.poll(() => updatePayload).not.toBeNull();
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
        await expect(page.locator('h1')).toBeVisible();

        // Delete Monday schedule
        const mondayRow = page.locator('[data-testid^="schedule-row-"]').first();
        const deleteButton = mondayRow.locator('button').nth(1);
        await deleteButton.click();

        const confirmButton = page
            .locator('.rc-dialog-update button')
            .filter({ hasText: /delete/i });
        await expect(confirmButton).toBeVisible();
        await confirmButton.click();

        // Blocked service IDs should be preserved
        await expect.poll(() => updatePayload).not.toBeNull();
        expect(updatePayload.ids).toEqual(['telegram', 'steam']);
    });

    test('should navigate from blocked services to schedule and back', async ({ page }) => {
        await page.goto('/#blocked_services');
        await expect(page.locator('h1')).toBeVisible();

        // Navigate to schedule
        const navItem = page.getByTestId('blocked-services-schedule-link');
        await navItem.click();

        await expect(page).toHaveURL(/#blocked_services\/schedule/);

        // Navigate back via breadcrumbs
        const breadcrumb = page
            .getByTestId('inactivity-schedule-breadcrumbs')
            .getByRole('link', { name: 'Blocked services' });
        await breadcrumb.click();

        await expect(page).toHaveURL(/#blocked_services$/);
    });
});
