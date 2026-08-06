import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';

import { login } from '../helpers/login';

const MOCK_QUERY_LOG_CONFIG = {
    anonymize_client_ip: false,
    enabled: true,
    ignored: [],
    ignore_enabled: false,
    interval: 24,
};

type QueryLogApiEntry = {
    answer: Array<{ value: string; type: string; ttl: number }>;
    answer_dnssec: boolean;
    cached: boolean;
    client: string;
    client_id: string;
    client_info: {
        name: string;
        whois: {
            city: string;
            country: string;
            orgname: string;
        };
    };
    client_proto: string;
    ecs: string;
    elapsedMs: string;
    filterId: number;
    original_answer: Array<{ value: string; type: string; ttl: number }>;
    question: {
        name: string;
        unicode_name: string;
        type: string;
    };
    reason: string;
    rule: string;
    rules: Array<{ filter_list_id: number; text: string }>;
    service_name: string;
    status: string;
    time: string;
    upstream: string;
};

type QueryLogResponse = {
    data: QueryLogApiEntry[];
    oldest: string;
};

type FilteringStatus = {
    enabled: boolean;
    filters: Array<{
        id: number;
        url: string;
        enabled: boolean;
        last_updated: string;
        name: string;
        rules_count: number;
    }>;
    whitelist_filters: Array<{
        id: number;
        url: string;
        enabled: boolean;
        last_updated: string;
        name: string;
        rules_count: number;
    }>;
    user_rules: string[];
    interval: number;
};

type AccessList = {
    allowed_clients: string[];
    disallowed_clients: string[];
    blocked_hosts: string[];
};

type BlockedServicesList = {
    ids: string[];
    schedule: {
        time_zone: string;
    };
};

type ClientsResponse = {
    clients: Array<{
        name: string;
        ids: string[];
    }>;
    auto_clients: Array<{
        ip: string;
        name: string;
        source: string;
    }>;
    supported_tags: string[];
};

type QueryLogSetupOptions = {
    queryLogResolver?: (requestUrl: URL) => QueryLogResponse;
    filteringStatus?: FilteringStatus;
    accessList?: AccessList;
    blockedServicesList?: BlockedServicesList;
    clientsResponse?: ClientsResponse;
    queryLogConfig?: typeof MOCK_QUERY_LOG_CONFIG;
    allBlockedServices?: Array<{ id: string; name: string }>;
};

type QueryLogSetupResult = {
    queryLogRequests: URL[];
    rulesUpdatePayloads: Array<{ rules?: string[] }>;
    accessSetPayloads: AccessList[];
    blockedServicesUpdatePayloads: BlockedServicesList[];
};

const QUERY_LOGS_PAGE_LIMIT = 20;

const QUERY_LOG_ROWS: QueryLogApiEntry[] = [
    {
        answer: [{ value: '93.184.216.34', type: 'A', ttl: 300 }],
        answer_dnssec: true,
        cached: false,
        client: '192.168.0.10',
        client_id: '',
        client_info: {
            name: 'Office Mac',
            whois: { city: 'London', country: 'United Kingdom', orgname: 'Example ISP' },
        },
        client_proto: 'udp',
        ecs: '',
        elapsedMs: '17.2',
        filterId: 1,
        original_answer: [],
        question: { name: 'example.org', unicode_name: 'example.org', type: 'A' },
        reason: 'FilteredBlackList',
        rule: '||example.org^',
        rules: [{ filter_list_id: 1, text: '||example.org^' }],
        service_name: '',
        status: 'NOERROR',
        time: '2026-05-12T10:00:00.000Z',
        upstream: 'tls://1.1.1.1',
    },
    {
        answer: [{ value: '198.51.100.10', type: 'A', ttl: 300 }],
        answer_dnssec: false,
        cached: true,
        client: '192.168.0.20',
        client_id: '',
        client_info: {
            name: 'Living Room TV',
            whois: { city: 'Paris', country: 'France', orgname: 'Mobile ISP' },
        },
        client_proto: 'https',
        ecs: '',
        elapsedMs: '9.4',
        filterId: -5,
        original_answer: [{ value: '203.0.113.30', type: 'A', ttl: 300 }],
        question: { name: 'search.example', unicode_name: 'search.example', type: 'A' },
        reason: 'FilteredSafeSearch',
        rule: '',
        rules: [],
        service_name: '',
        status: 'NOERROR',
        time: '2026-05-12T10:01:00.000Z',
        upstream: 'https://dns.example/dns-query',
    },
    {
        answer: [{ value: '203.0.113.55', type: 'A', ttl: 300 }],
        answer_dnssec: false,
        cached: false,
        client: '192.168.0.30',
        client_id: '',
        client_info: {
            name: 'Family iPad',
            whois: { city: 'Berlin', country: 'Germany', orgname: 'Metro Fiber' },
        },
        client_proto: 'udp',
        ecs: '',
        elapsedMs: '13.0',
        filterId: 7,
        original_answer: [],
        question: { name: 'allowed.example', unicode_name: 'allowed.example', type: 'A' },
        reason: 'NotFilteredWhiteList',
        rule: '@@||allowed.example^',
        rules: [{ filter_list_id: 7, text: '@@||allowed.example^' }],
        service_name: '',
        status: 'NOERROR',
        time: '2026-05-12T10:02:00.000Z',
        upstream: 'tls://9.9.9.9',
    },
    {
        answer: [{ value: '203.0.113.77', type: 'A', ttl: 300 }],
        answer_dnssec: false,
        cached: false,
        client: '192.168.0.40',
        client_id: '',
        client_info: {
            name: 'Kitchen Display',
            whois: { city: 'Rome', country: 'Italy', orgname: 'City Fiber' },
        },
        client_proto: 'udp',
        ecs: '',
        elapsedMs: '21.6',
        filterId: 0,
        original_answer: [],
        question: { name: 'plain.example', unicode_name: 'plain.example', type: 'A' },
        reason: 'NotFilteredNotFound',
        rule: '',
        rules: [],
        service_name: '',
        status: 'NOERROR',
        time: '2026-05-12T10:03:00.000Z',
        upstream: 'tls://8.8.8.8',
    },
    {
        answer: [],
        answer_dnssec: false,
        cached: false,
        client: '192.168.0.50',
        client_id: 'server-rack',
        client_info: {
            name: 'Server Rack',
            whois: { city: 'Madrid', country: 'Spain', orgname: 'Datacenter Net' },
        },
        client_proto: 'tcp',
        ecs: '',
        elapsedMs: '31.1',
        filterId: 0,
        original_answer: [],
        question: { name: 'failed.example', unicode_name: 'failed.example', type: 'AAAA' },
        reason: 'NotFilteredError',
        rule: '',
        rules: [],
        service_name: '',
        status: 'SERVFAIL',
        time: '2026-05-12T10:04:00.000Z',
        upstream: 'tls://8.8.4.4',
    },
];

const DEFAULT_FILTERING_STATUS: FilteringStatus = {
    enabled: true,
    filters: [
        {
            id: 1,
            url: 'https://filters.example/blocklist.txt',
            enabled: true,
            last_updated: '2026-05-12T08:00:00.000Z',
            name: 'Primary blocklist',
            rules_count: 1234,
        },
    ],
    whitelist_filters: [
        {
            id: 7,
            url: 'https://filters.example/allowlist.txt',
            enabled: true,
            last_updated: '2026-05-12T08:05:00.000Z',
            name: 'Primary allowlist',
            rules_count: 32,
        },
    ],
    user_rules: [],
    interval: 24,
};

const DEFAULT_ACCESS_LIST: AccessList = {
    allowed_clients: [],
    disallowed_clients: [],
    blocked_hosts: [],
};

const DEFAULT_BLOCKED_SERVICES_LIST: BlockedServicesList = {
    ids: [],
    schedule: {
        time_zone: 'UTC',
    },
};

const DEFAULT_ALL_BLOCKED_SERVICES = [{ id: 'amazon', name: 'Amazon' }];

const DEFAULT_CLIENTS_RESPONSE: ClientsResponse = {
    clients: [],
    auto_clients: [],
    supported_tags: [],
};

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

const applySearchFilter = (rows: QueryLogApiEntry[], search: string) => {
    if (!search) {
        return rows;
    }

    const query = search.toLowerCase();

    return rows.filter((row) => {
        const clientName = row.client_info?.name.toLowerCase() ?? '';

        return (
            row.question.name.toLowerCase().includes(query) ||
            row.question.unicode_name.toLowerCase().includes(query) ||
            row.client.toLowerCase().includes(query) ||
            clientName.includes(query)
        );
    });
};

const applyResponseStatusFilter = (rows: QueryLogApiEntry[], responseStatus: string) => {
    switch (responseStatus) {
        case 'all':
            return rows;
        case 'blocked':
            return rows.filter(
                (row) =>
                    row.reason === 'FilteredBlackList' || row.reason === 'FilteredBlockedService',
            );
        case 'blocked_services':
            return rows.filter((row) => row.reason === 'FilteredBlockedService');
        case 'blocked_safebrowsing':
            return rows.filter((row) => row.reason === 'FilteredSafeBrowsing');
        case 'blocked_parental':
            return rows.filter((row) => row.reason === 'FilteredParental');
        case 'safe_search':
            return rows.filter((row) => row.reason === 'FilteredSafeSearch');
        case 'whitelisted':
            return rows.filter((row) => row.reason === 'NotFilteredWhiteList');
        case 'rewritten':
            return rows.filter(
                (row) =>
                    row.reason === 'Rewrite' ||
                    row.reason === 'RewriteEtcHosts' ||
                    row.reason === 'RewriteRule',
            );
        case 'processed':
            return rows.filter(
                (row) =>
                    row.reason !== 'FilteredBlackList' &&
                    row.reason !== 'FilteredBlockedService' &&
                    row.reason !== 'NotFilteredWhiteList',
            );
        default:
            return rows;
    }
};

const buildDefaultQueryLogResponse = (requestUrl: URL): QueryLogResponse => {
    const search = requestUrl.searchParams.get('search') ?? '';
    const responseStatus = requestUrl.searchParams.get('response_status') ?? 'all';
    const filteredRows = applyResponseStatusFilter(
        applySearchFilter(QUERY_LOG_ROWS, search),
        responseStatus,
    );

    return {
        data: filteredRows,
        oldest: '',
    };
};

const createPaginatedRows = (count: number, offset = 0): QueryLogApiEntry[] =>
    Array.from({ length: count }, (_, index) => {
        const item = offset + index + 1;
        const domain = `paged-${item}.example`;
        const client = `10.0.0.${item}`;

        return {
            answer: [{ value: `203.0.113.${(item % 200) + 1}`, type: 'A', ttl: 300 }],
            answer_dnssec: item % 2 === 0,
            cached: false,
            client,
            client_id: '',
            client_info: {
                name: `Paged Device ${item}`,
                whois: { city: 'Tallinn', country: 'Estonia', orgname: 'Pagination ISP' },
            },
            client_proto: 'udp',
            ecs: '',
            elapsedMs: `${10 + item / 10}`,
            filterId: 0,
            original_answer: [],
            question: { name: domain, unicode_name: domain, type: 'A' },
            reason: 'NotFilteredNotFound',
            rule: '',
            rules: [],
            service_name: '',
            status: 'NOERROR',
            time: new Date(Date.UTC(2026, 4, 12, 11, 0, item)).toISOString(),
            upstream: 'tls://8.8.8.8',
        };
    });

async function setupQueryLogMocks(
    page: Page,
    {
        queryLogResolver = buildDefaultQueryLogResponse,
        filteringStatus = DEFAULT_FILTERING_STATUS,
        accessList = DEFAULT_ACCESS_LIST,
        blockedServicesList = DEFAULT_BLOCKED_SERVICES_LIST,
        clientsResponse = DEFAULT_CLIENTS_RESPONSE,
        queryLogConfig = MOCK_QUERY_LOG_CONFIG,
        allBlockedServices = DEFAULT_ALL_BLOCKED_SERVICES,
    }: QueryLogSetupOptions = {},
): Promise<QueryLogSetupResult> {
    const queryLogRequests: URL[] = [];
    const rulesUpdatePayloads: Array<{ rules?: string[] }> = [];
    const accessSetPayloads: AccessList[] = [];
    const blockedServicesUpdatePayloads: BlockedServicesList[] = [];
    let filteringStatusState = clone(filteringStatus);
    let accessListState = clone(accessList);
    let blockedServicesState = clone(blockedServicesList);

    await page.route('**/control/querylog/config', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(queryLogConfig),
        });
    });

    await page.route('**/control/clients', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(clientsResponse),
        });
    });

    await page.route('**/control/querylog*', (route) => {
        const requestUrl = new URL(route.request().url());

        queryLogRequests.push(requestUrl);
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(queryLogResolver(requestUrl)),
        });
    });

    await page.route('**/control/filtering/status', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(filteringStatusState),
        });
    });

    await page.route('**/control/filtering/set_rules', (route) => {
        const payload = route.request().postDataJSON() as { rules?: string[] };
        rulesUpdatePayloads.push(payload);
        filteringStatusState = {
            ...filteringStatusState,
            user_rules: payload.rules ?? [],
        };

        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    await page.route('**/control/access/list', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(accessListState),
        });
    });

    await page.route('**/control/access/set', (route) => {
        const payload = route.request().postDataJSON() as AccessList;
        accessSetPayloads.push(payload);
        accessListState = payload;

        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    await page.route('**/control/blocked_services/get', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(blockedServicesState),
        });
    });

    await page.route('**/control/blocked_services/all', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                blocked_services: allBlockedServices,
                groups: [],
            }),
        });
    });

    await page.route('**/control/blocked_services/update', (route) => {
        const payload = route.request().postDataJSON() as BlockedServicesList;
        blockedServicesUpdatePayloads.push(payload);
        blockedServicesState = payload;

        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    return {
        queryLogRequests,
        rulesUpdatePayloads,
        accessSetPayloads,
        blockedServicesUpdatePayloads,
    };
}

async function openQueryLog(page: Page, options?: QueryLogSetupOptions) {
    const setup = await setupQueryLogMocks(page, options);

    await login(page);
    await page.goto('/#logs');
    await expect(page.locator('[data-testid="query-log-request-cell"]').first()).toBeAttached();

    return setup;
}

async function selectFilterOption(
    page: Page,
    filterTestId: string,
    optionValue: string,
    optionPrefix: string,
) {
    const filter = page.getByTestId(filterTestId);

    await filter.locator('[data-part="control"]').click();
    await expect(page.getByTestId(`${optionPrefix}-${optionValue}`)).toBeVisible();
    await page.getByTestId(`${optionPrefix}-${optionValue}`).click();
}

const getPageLocation = (page: Page) => {
    const currentUrl = new URL(page.url());
    const hash = currentUrl.hash.startsWith('#') ? currentUrl.hash.slice(1) : currentUrl.hash;
    const [rawRoute = '', hashQuery = ''] = hash.split('?');
    const route = rawRoute.replace(/^\/?/, '');
    const params =
        currentUrl.searchParams.size > 0 ? currentUrl.searchParams : new URLSearchParams(hashQuery);

    return {
        currentUrl,
        route,
        params,
    };
};

const getRequestCellByDomain = (page: Page, domain: string) =>
    page
        .getByTestId('query-log-request-cell')
        .filter({ has: page.locator(`[title="${domain}"]`) })
        .first();

const getClientCellByIp = (page: Page, clientIp: string) =>
    page.getByTestId('query-log-client-cell').filter({ hasText: clientIp }).first();

const getDesktopRowByDomain = (page: Page, domain: string) =>
    getRequestCellByDomain(page, domain).locator('xpath=../..');

const getVisibleActionsMenu = (page: Page) =>
    page.locator('[data-testid="query-log-row-actions-menu"]:visible').last();

const getVisibleInfiniteScrollTrigger = (page: Page) =>
    page.locator('[data-testid="query-log-infinite-scroll-trigger"]:visible');

async function closeOpenActionMenus(page: Page) {
    await page.keyboard.press('Escape');
}

async function expectQueryLogRequestCount(queryLogRequests: URL[], expectedCount: number) {
    await expect.poll(() => queryLogRequests.length).toBe(expectedCount);
    return queryLogRequests.at(-1);
}

function expectPageFilters(
    page: Page,
    {
        search,
        status,
        reason,
    }: {
        search: string | null;
        status: string;
        reason: string;
    },
) {
    const { route, params } = getPageLocation(page);

    expect(route).toBe('logs');
    expect(params.get('search')).toBe(search);
    expect(params.get('status')).toBe(status);
    expect(params.get('reason')).toBe(reason);
}

async function scrollToLoadMoreTrigger(page: Page) {
    await getVisibleInfiniteScrollTrigger(page).scrollIntoViewIfNeeded();
}

test.describe('Query log desktop', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test('supports semantic row assertions, quick filters, search, refresh, and URL sync', async ({
        page,
    }) => {
        const { queryLogRequests } = await openQueryLog(page);
        const searchInput = page.getByTestId('query-log-search-input');

        const blockedRequestCell = getRequestCellByDomain(page, 'example.org');
        await expect(blockedRequestCell).toContainText('example.org');

        await getClientCellByIp(page, '192.168.0.40')
            .getByRole('button', { name: '192.168.0.40' })
            .click();
        const ipFilterRequest = await expectQueryLogRequestCount(queryLogRequests, 2);

        expect(ipFilterRequest?.searchParams.get('search')).toBe('192.168.0.40');
        await expect(searchInput).toHaveValue('192.168.0.40');
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(1);
        await expect(getRequestCellByDomain(page, 'plain.example')).toHaveCount(1);
        expectPageFilters(page, { search: '192.168.0.40', status: 'all', reason: 'all' });

        await searchInput.fill('');
        const clearedSearchRequest = await expectQueryLogRequestCount(queryLogRequests, 3);

        expect(clearedSearchRequest?.searchParams.get('search')).toBeNull();
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(QUERY_LOG_ROWS.length);
        expectPageFilters(page, { search: null, status: 'all', reason: 'all' });

        const officeClientCell = getClientCellByIp(page, '192.168.0.10');

        await officeClientCell.getByRole('button', { name: 'Office Mac' }).click();
        const clientNameRequest = await expectQueryLogRequestCount(queryLogRequests, 4);

        expect(clientNameRequest?.searchParams.get('search')).toBe('Office Mac');
        await expect(searchInput).toHaveValue('Office Mac');
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(1);
        await expect(getRequestCellByDomain(page, 'example.org')).toHaveCount(1);
        expectPageFilters(page, { search: 'Office Mac', status: 'all', reason: 'all' });

        await searchInput.fill('plain.example');
        const manualSearchRequest = await expectQueryLogRequestCount(queryLogRequests, 5);

        expect(manualSearchRequest?.searchParams.get('search')).toBe('plain.example');
        await expect(searchInput).toHaveValue('plain.example');
        await expect(page.getByTestId('query-log-search-clear-button')).toBeEnabled();
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(1);
        await expect(getRequestCellByDomain(page, 'plain.example')).toHaveCount(1);
        expectPageFilters(page, { search: 'plain.example', status: 'all', reason: 'all' });

        await page.getByTestId('query-log-refresh-button-desktop').click();
        const refreshRequest = await expectQueryLogRequestCount(queryLogRequests, 6);

        expect(refreshRequest?.searchParams.get('search')).toBe('plain.example');
        expect(refreshRequest?.searchParams.get('response_status')).toBe('all');
        await expect(
            page.locator('[data-testid="toast"][data-toast-code="notify_updated"]'),
        ).toBeVisible();

        await page.getByTestId('query-log-search-clear-button').click();
        const resetSearchRequest = await expectQueryLogRequestCount(queryLogRequests, 7);

        expect(resetSearchRequest?.searchParams.get('search')).toBeNull();
        await expect(searchInput).toHaveValue('');
        await expect(page.getByTestId('query-log-search-clear-button')).toHaveCount(0);
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(QUERY_LOG_ROWS.length);
        expectPageFilters(page, { search: null, status: 'all', reason: 'all' });
    });

    test('supports client-side status filtering, server-backed reason filtering, and filtered empty states', async ({
        page,
    }) => {
        const { queryLogRequests } = await openQueryLog(page);

        await selectFilterOption(
            page,
            'query-log-status-filter',
            'blocked',
            'query-log-status-option',
        );
        const blockedStatusRequest = await expectQueryLogRequestCount(queryLogRequests, 2);

        expect(blockedStatusRequest?.searchParams.get('response_status')).toBe('blocked');
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(1);
        await expect(getRequestCellByDomain(page, 'example.org')).toHaveCount(1);
        expectPageFilters(page, { search: null, status: 'blocked', reason: 'all' });

        await selectFilterOption(page, 'query-log-status-filter', 'all', 'query-log-status-option');
        await expectQueryLogRequestCount(queryLogRequests, 3);
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(QUERY_LOG_ROWS.length);

        await selectFilterOption(
            page,
            'query-log-reason-filter',
            'FilteredSafeSearch',
            'query-log-reason-option',
        );
        const safeSearchRequest = await expectQueryLogRequestCount(queryLogRequests, 4);

        expect(safeSearchRequest?.searchParams.get('response_status')).toBe('safe_search');
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(1);
        await expect(getRequestCellByDomain(page, 'search.example')).toHaveCount(1);
        expectPageFilters(page, { search: null, status: 'all', reason: 'FilteredSafeSearch' });

        await selectFilterOption(
            page,
            'query-log-status-filter',
            'blocked',
            'query-log-status-option',
        );
        const emptyStateRequest = await expectQueryLogRequestCount(queryLogRequests, 5);

        expect(emptyStateRequest?.searchParams.get('response_status')).toBe('blocked');
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(1);
        await expect(getRequestCellByDomain(page, 'example.org')).toHaveCount(1);
        expectPageFilters(page, { search: null, status: 'blocked', reason: 'all' });
    });

    test('shows the log-rotation empty state with a settings link', async ({ page }) => {
        await setupQueryLogMocks(page, {
            queryLogResolver: () => ({
                data: [],
                oldest: '',
            }),
            queryLogConfig: {
                ...MOCK_QUERY_LOG_CONFIG,
                interval: 0,
            },
        });

        await login(page);
        await page.goto('/#logs');

        const emptyState = page.locator('[data-testid="query-log-empty-state"]:visible');
        const settingsLink = emptyState.locator('a');

        await expect(emptyState).toBeVisible();
        await expect(settingsLink).toHaveCount(1);
        await expect(settingsLink).toHaveAttribute('href', /#\/?settings$/);
    });

    test('shows the disabled-log empty state with a settings link', async ({ page }) => {
        await setupQueryLogMocks(page, {
            queryLogResolver: () => ({
                data: [],
                oldest: '',
            }),
            queryLogConfig: {
                ...MOCK_QUERY_LOG_CONFIG,
                enabled: false,
            },
        });

        await login(page);
        await page.goto('/#logs');

        const emptyState = page.locator('[data-testid="query-log-empty-state"]:visible');
        const settingsLink = emptyState.locator('a');

        await expect(emptyState).toBeVisible();
        await expect(settingsLink).toHaveCount(1);
        await expect(settingsLink).toHaveAttribute('href', /#\/?settings$/);
    });

    // TODO(AGDNS-4057): Re-enable when row action menus and detail modal
    // action buttons are fully implemented.
    test.skip('supports detail-modal actions and row actions', async ({ page }) => {
        const { rulesUpdatePayloads, accessSetPayloads } = await openQueryLog(page);

        // plain.example has reason NotFilteredNotFound, so showBlock=true (block button visible)
        await getDesktopRowByDomain(page, 'plain.example').click();

        const detailModal = page.getByTestId('query-log-detail-modal');
        await expect(detailModal).toBeVisible();
        await expect(detailModal).toContainText('plain.example');

        await page.getByTestId('query-log-detail-action-block').click();
        await expect.poll(() => rulesUpdatePayloads.length).toBe(1);
        expect(rulesUpdatePayloads[0].rules).toContain('||plain.example^$important');
        await expect(
            page.locator('[data-testid="toast"][data-toast-code="notify_user_rule_added"]'),
        ).toBeVisible();
        await expect(detailModal).toHaveCount(0);

        const blockedRow = getDesktopRowByDomain(page, 'example.org');
        await blockedRow.getByTestId('query-log-row-actions-trigger').click();
        await getVisibleActionsMenu(page).getByTestId('query-log-row-action-toggle-block').click();
        await expect.poll(() => rulesUpdatePayloads.length).toBe(2);
        expect(rulesUpdatePayloads[1].rules).toContain('@@||example.org^$important');
        await closeOpenActionMenus(page);

        const processedRow = getDesktopRowByDomain(page, 'plain.example');
        await processedRow.getByTestId('query-log-row-actions-trigger').click();
        await getVisibleActionsMenu(page).getByTestId('query-log-row-action-block-client').click();
        await expect.poll(() => rulesUpdatePayloads.length).toBe(3);
        expect(rulesUpdatePayloads[2].rules).toContain("||plain.example^$client='192.168.0.40'");
        await closeOpenActionMenus(page);

        await processedRow.getByTestId('query-log-row-actions-trigger').click();
        await getVisibleActionsMenu(page)
            .getByTestId('query-log-row-action-disallow-client')
            .click();
        await page.getByTestId('query-log-disallow-cancel').click();
        expect(accessSetPayloads).toHaveLength(0);

        await processedRow.getByTestId('query-log-row-actions-trigger').click();
        await getVisibleActionsMenu(page)
            .getByTestId('query-log-row-action-disallow-client')
            .click();
        await page.getByTestId('query-log-disallow-confirm').click();
        await expect.poll(() => accessSetPayloads.length).toBe(1);
        expect(accessSetPayloads[0]).toEqual({
            allowed_clients: [],
            disallowed_clients: ['192.168.0.40'],
            blocked_hosts: [],
        });
        await expect(
            page.locator('[data-testid="toast"][data-toast-code="settings_notify_changes_saved"]'),
        ).toBeVisible();
    });

    // TODO(AGDNS-4057): Re-enable when row action menus with
    // add-persistent-client action are implemented.
    test.skip('shows the add persistent client action only for clients missing from persistent clients and navigates to the clients page', async ({
        page,
    }) => {
        await openQueryLog(page, {
            clientsResponse: {
                clients: [
                    {
                        name: 'Office Mac',
                        ids: ['192.168.0.10'],
                    },
                    {
                        name: 'Server Rack',
                        ids: ['server-rack'],
                    },
                ],
                auto_clients: [],
                supported_tags: [],
            },
        });

        const persistentRow = getDesktopRowByDomain(page, 'example.org');
        await persistentRow.getByTestId('query-log-row-actions-trigger').click();
        await expect(
            getVisibleActionsMenu(page).getByTestId('query-log-row-action-add-persistent-client'),
        ).toHaveCount(0);
        await closeOpenActionMenus(page);

        const clientIdRow = getDesktopRowByDomain(page, 'failed.example');
        await clientIdRow.getByTestId('query-log-row-actions-trigger').click();
        await expect(
            getVisibleActionsMenu(page).getByTestId('query-log-row-action-add-persistent-client'),
        ).toHaveCount(0);
        await closeOpenActionMenus(page);

        const unmanagedRow = getDesktopRowByDomain(page, 'plain.example');
        await unmanagedRow.getByTestId('query-log-row-actions-trigger').click();

        const unmanagedMenu = getVisibleActionsMenu(page);
        await expect(
            unmanagedMenu.getByTestId('query-log-row-action-add-persistent-client'),
        ).toBeVisible();

        await unmanagedMenu.getByTestId('query-log-row-action-add-persistent-client').click();
        await expect(page).toHaveURL(/#clients\?clientId=192\.168\.0\.40$/);
    });

    // TODO(AGDNS-4057): Re-enable when detail modal action buttons
    // (allowlist, allow-service) and row action menus are implemented.
    test.skip('allows a blocked service from the detail modal', async ({ page }) => {
        const blockedServiceRow: QueryLogApiEntry = {
            answer: [{ value: '203.0.113.120', type: 'A', ttl: 300 }],
            answer_dnssec: false,
            cached: false,
            client: '192.168.0.60',
            client_id: '',
            client_info: {
                name: 'Media Console',
                whois: { city: 'Prague', country: 'Czechia', orgname: 'Fiber ISP' },
            },
            client_proto: 'udp',
            ecs: '',
            elapsedMs: '12.7',
            filterId: -2,
            original_answer: [],
            question: { name: 'streaming.example', unicode_name: 'streaming.example', type: 'A' },
            reason: 'FilteredBlockedService',
            rule: '||streaming.example^',
            rules: [],
            service_name: 'amazon',
            status: 'NOERROR',
            time: '2026-05-12T10:05:00.000Z',
            upstream: 'tls://1.0.0.1',
        };
        const { rulesUpdatePayloads, blockedServicesUpdatePayloads } = await openQueryLog(page, {
            queryLogResolver: () => ({
                data: [blockedServiceRow],
                oldest: '',
            }),
            blockedServicesList: {
                ids: ['amazon'],
                schedule: {
                    time_zone: 'UTC',
                },
            },
        });

        await getDesktopRowByDomain(page, 'streaming.example').click();

        const detailModal = page.getByTestId('query-log-detail-modal');

        await expect(detailModal).toBeVisible();

        await page.getByTestId('query-log-detail-action-allowlist').click();
        await expect.poll(() => rulesUpdatePayloads.length).toBe(1);
        expect(rulesUpdatePayloads[0].rules).toContain('@@||streaming.example^$important');
        await expect(
            page.locator('[data-testid="toast"][data-toast-code="notify_user_rule_added"]'),
        ).toBeVisible();
        await expect(detailModal).toHaveCount(0);

        await getDesktopRowByDomain(page, 'streaming.example').click();
        await expect(detailModal).toBeVisible();

        await page.getByTestId('query-log-detail-action-allow-service').click();
        await expect.poll(() => blockedServicesUpdatePayloads.length).toBe(1);
        expect(blockedServicesUpdatePayloads[0]).toEqual({
            ids: [],
            schedule: {
                time_zone: 'UTC',
            },
        });
        await expect(
            page.locator('[data-testid="toast"][data-toast-code="settings_notify_changes_saved"]'),
        ).toBeVisible();
        await expect(detailModal).toHaveCount(0);
    });

    test('loads additional log pages through the infinite scroll trigger', async ({ page }) => {
        const firstPageRows = createPaginatedRows(QUERY_LOGS_PAGE_LIMIT);
        const secondPageRows = createPaginatedRows(5, QUERY_LOGS_PAGE_LIMIT);
        const servedCursors = new Map<string, number>();
        const { queryLogRequests } = await openQueryLog(page, {
            queryLogResolver: (requestUrl) => {
                const olderThan = requestUrl.searchParams.get('older_than') ?? '';
                const requestCount = servedCursors.get(olderThan) ?? 0;

                servedCursors.set(olderThan, requestCount + 1);

                switch (olderThan) {
                    case '':
                        return requestCount === 0
                            ? { data: firstPageRows, oldest: 'page-1' }
                            : { data: [], oldest: 'page-1' };
                    case 'page-1':
                        return requestCount === 0
                            ? { data: secondPageRows, oldest: '' }
                            : { data: [], oldest: '' };
                    default:
                        return { data: [], oldest: '' };
                }
            },
        });

        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(QUERY_LOGS_PAGE_LIMIT);
        await expect(getVisibleInfiniteScrollTrigger(page)).toBeVisible();

        await scrollToLoadMoreTrigger(page);
        await expect
            .poll(() =>
                queryLogRequests.some(
                    (requestUrl) => requestUrl.searchParams.get('older_than') === 'page-1',
                ),
            )
            .toBe(true);
        await expect(page.getByTestId('query-log-request-cell')).toHaveCount(
            QUERY_LOGS_PAGE_LIMIT + secondPageRows.length,
        );
        await expect(getVisibleInfiniteScrollTrigger(page)).toHaveCount(0);
    });
});
