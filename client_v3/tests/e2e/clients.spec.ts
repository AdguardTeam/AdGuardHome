import { test, expect, type Page } from '@playwright/test';

import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

// ---- Mock data types ----

type Client = {
    name: string;
    ids: string[];
    tags: string[];
    use_global_settings: boolean;
    filtering_enabled: boolean;
    safebrowsing_enabled: boolean;
    parental_enabled: boolean;
    safe_search: Record<string, boolean>;
    ignore_querylog: boolean;
    ignore_statistics: boolean;
    blocked_services: string[];
    use_global_blocked_services: boolean;
    blocked_services_schedule: { time_zone: string };
    upstreams: string[];
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
};

type AutoClient = {
    ip: string;
    name: string;
    source: string;
    whois_info: Record<string, unknown>;
};

type ClientsResponse = {
    clients: Client[];
    auto_clients: AutoClient[];
    supported_tags: string[];
};

type AddClientPayload = {
    name: string;
    ids: string[];
    tags: string[];
    use_global_settings: boolean;
    filtering_enabled: boolean;
    safebrowsing_enabled: boolean;
    parental_enabled: boolean;
    safe_search: Record<string, boolean>;
    ignore_querylog: boolean;
    ignore_statistics: boolean;
    blocked_services: string[];
    use_global_blocked_services: boolean;
    blocked_services_schedule: { time_zone: string };
    upstreams: string[];
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
};

type UpdateClientPayload = {
    name: string;
    data: AddClientPayload;
};

type DeleteClientPayload = {
    name: string;
};

// ---- Mock data ----

const MOCK_CLIENT_1: Client = {
    name: 'Office Desktop',
    ids: ['192.168.0.100'],
    tags: ['work'],
    use_global_settings: true,
    filtering_enabled: true,
    safebrowsing_enabled: true,
    parental_enabled: false,
    safe_search: {
        enabled: false,
        google: false,
        youtube: false,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
    },
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: [],
    use_global_blocked_services: true,
    blocked_services_schedule: { time_zone: 'UTC' },
    upstreams: [],
    upstreams_cache_enabled: false,
    upstreams_cache_size: 0,
};

const MOCK_CLIENT_2: Client = {
    name: 'Living Room TV',
    ids: ['AA:BB:CC:DD:EE:FF'],
    tags: [],
    use_global_settings: false,
    filtering_enabled: true,
    safebrowsing_enabled: false,
    parental_enabled: true,
    safe_search: {
        enabled: true,
        google: true,
        youtube: true,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
    },
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: ['youtube'],
    use_global_blocked_services: false,
    blocked_services_schedule: { time_zone: 'Europe/London' },
    upstreams: ['tls://1.1.1.1'],
    upstreams_cache_enabled: true,
    upstreams_cache_size: 4194304,
};

const MOCK_AUTO_CLIENT: AutoClient = {
    ip: '192.168.0.200',
    name: 'android-phone.local',
    source: 'rDNS',
    whois_info: {},
};

const DEFAULT_CLIENTS_RESPONSE: ClientsResponse = {
    clients: [MOCK_CLIENT_1, MOCK_CLIENT_2],
    auto_clients: [MOCK_AUTO_CLIENT],
    supported_tags: ['work', 'home', 'guest'],
};

// ---- Helpers ----

async function login(page: Page) {
    let lastError: unknown;

    for (let attempt = 0; attempt < 3; attempt += 1) {
        await page.goto('/login.html', { waitUntil: 'domcontentloaded' });

        try {
            await page.locator('#username').waitFor({ state: 'visible', timeout: 5000 });
            await page.locator('#username').fill(ADMIN_USERNAME);
            await page.locator('#password').fill(ADMIN_PASSWORD);
            await page.locator('#sign_in').click();
            await page.waitForURL((url) => !url.href.endsWith('/login.html'));

            return;
        } catch (error) {
            lastError = error;
        }
    }

    throw lastError;
}

type ClientsMocksResult = {
    addClientPayloads: AddClientPayload[];
    updateClientPayloads: UpdateClientPayload[];
    deleteClientPayloads: DeleteClientPayload[];
};

async function setupClientsMocks(
    page: Page,
    { clientsResponse = DEFAULT_CLIENTS_RESPONSE }: { clientsResponse?: ClientsResponse } = {},
): Promise<ClientsMocksResult> {
    const addClientPayloads: AddClientPayload[] = [];
    const updateClientPayloads: UpdateClientPayload[] = [];
    const deleteClientPayloads: DeleteClientPayload[] = [];
    let clientsState = JSON.parse(JSON.stringify(clientsResponse)) as ClientsResponse;

    await page.route('**/control/clients', (route) => {
        if (route.request().method() === 'GET') {
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(clientsState),
            });
        } else {
            route.continue();
        }
    });

    await page.route('**/control/clients/add', (route) => {
        const payload = route.request().postDataJSON() as AddClientPayload;
        addClientPayloads.push(payload);
        const newClient: Client = { ...payload, upstreams: payload.upstreams || [] };
        clientsState = {
            ...clientsState,
            clients: [...clientsState.clients, newClient],
        };
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });

    await page.route('**/control/clients/update', (route) => {
        const payload = route.request().postDataJSON() as UpdateClientPayload;
        updateClientPayloads.push(payload);
        clientsState = {
            ...clientsState,
            clients: clientsState.clients.map((c) =>
                c.name === payload.name
                    ? { ...payload.data, name: payload.data.name || payload.name }
                    : c,
            ),
        };
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });

    await page.route('**/control/clients/delete', (route) => {
        const payload = route.request().postDataJSON() as DeleteClientPayload;
        deleteClientPayloads.push(payload);
        clientsState = {
            ...clientsState,
            clients: clientsState.clients.filter((c) => c.name !== payload.name),
        };
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });

    await page.route('**/control/stats', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                dns_queries: [],
                blocked_filtering: [],
                avg_processing_time: 0,
                top_clients: [],
                top_queried_domains: [],
                top_blocked_domains: [],
                stats_period: {},
                num_dns_queries: 0,
                num_blocked_filtering: 0,
                num_replaced_safebrowsing: 0,
                num_replaced_safesearch: 0,
                num_replaced_parental: 0,
            }),
        });
    });

    await page.route('**/control/blocked_services/all', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                blocked_services: [{ id: 'youtube', name: 'YouTube' }],
                groups: [],
            }),
        });
    });

    await page.route('**/control/clients/search', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify([]),
        });
    });

    return { addClientPayloads, updateClientPayloads, deleteClientPayloads };
}

// ---- Tests ----

test.describe('Clients', () => {
    test('renders the clients page with persistent and runtime clients', async ({ page }) => {
        await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients');

        // Page heading
        await expect(page.getByTestId('clients-title')).toBeVisible();
        await expect(page.getByTestId('clients-add-button')).toBeVisible();

        // Persistent clients section heading
        await expect(page.getByText('Persistent clients')).toBeVisible();

        // Mock clients should appear in the table
        await expect(page.getByText('Office Desktop')).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText('192.168.0.100')).toBeVisible();
        await expect(page.getByText('Living Room TV')).toBeVisible();

        // Runtime clients section heading
        await expect(page.getByText('Runtime clients')).toBeVisible();
        await expect(page.getByText('192.168.0.200')).toBeVisible();
    });

    test('adds a new persistent client', async ({ page }) => {
        const { addClientPayloads } = await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients');

        // Click "Add Client" in the page header
        await page.getByTestId('clients-add-button').click();
        await expect(page).toHaveURL(/#clients\/add$/);
        await expect(page.getByTestId('client-form')).toBeVisible();

        // Fill in name and identifier
        await page.getByTestId('client-form-name').fill('Test Client');
        await page.locator('#client-identifier-0').fill('192.168.0.50');

        // Save
        await page.getByTestId('client-form-save').click();
        await expect(page).toHaveURL(/#clients$/);

        // Verify API payload
        await expect.poll(() => addClientPayloads.length).toBe(1);
        expect(addClientPayloads[0].name).toBe('Test Client');
        expect(addClientPayloads[0].ids).toEqual(['192.168.0.50']);
    });

    test('edits an existing persistent client', async ({ page }) => {
        const { updateClientPayloads } = await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients');

        // Click the edit button for "Office Desktop" (second row)
        await page.getByTestId('clients-edit-button').nth(1).click();

        await expect(page).toHaveURL(/#clients\/edit\/Office%20Desktop$/);
        await expect(page.getByTestId('client-form')).toBeVisible();

        // Verify pre-filled data
        await expect(page.getByTestId('client-form-name')).toHaveValue('Office Desktop');
        await expect(page.locator('#client-identifier-0')).toHaveValue('192.168.0.100');

        // Change name and save
        await page.getByTestId('client-form-name').fill('Office Desktop Updated');
        await page.getByTestId('client-form-save').click();

        await expect(page).toHaveURL(/#clients$/);

        // Verify update payload
        await expect.poll(() => updateClientPayloads.length).toBe(1);
        expect(updateClientPayloads[0].name).toBe('Office Desktop');
        expect(updateClientPayloads[0].data.name).toBe('Office Desktop Updated');
    });

    test('deletes a persistent client with confirmation', async ({ page }) => {
        const { deleteClientPayloads } = await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients');

        // Click the delete button for "Living Room TV" (first row)
        await page.getByTestId('clients-delete-button').first().click();

        // ConfirmDialog should appear
        await expect(page.getByText(/Are you sure you want to delete client/)).toBeVisible();

        // Click confirm "Remove"
        await page.getByRole('button', { name: 'Remove' }).click();

        // Verify DELETE API call
        await expect.poll(() => deleteClientPayloads.length).toBe(1);
        expect(deleteClientPayloads[0].name).toBe('Living Room TV');
    });

    test('shows validation error when client name is empty', async ({ page }) => {
        const { addClientPayloads } = await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients/add');

        await expect(page.getByTestId('client-form')).toBeVisible();

        // Fill identifier but leave name empty
        await page.locator('#client-identifier-0').fill('192.168.0.99');

        // Click Save
        await page.getByTestId('client-form-save').click();

        // Should show validation error message
        await expect(page.getByText('Please fill out this field')).toBeVisible();

        // No API call
        expect(addClientPayloads).toHaveLength(0);
    });
});
