import { expect, test, type Page } from '@playwright/test';

import { login } from '../helpers/login';

type FilterList = {
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

type SettingsStatuses = {
    safebrowsing: { enabled: boolean };
    parental: { enabled: boolean };
    safesearch: {
        enabled: boolean;
        bing: boolean;
        duckduckgo: boolean;
        google: boolean;
        pixabay: boolean;
        yandex: boolean;
        youtube: boolean;
    };
};

type RewriteEntry = {
    domain: string;
    answer: string;
    enabled?: boolean;
};

type BlockedServicesList = {
    ids: string[];
    schedule: {
        time_zone: string;
    };
};

type CheckHostResponse = {
    reason: string;
    rules: Array<{
        filter_list_id?: number;
        text: string;
    }>;
    service_name?: string;
    cname?: string;
    ip_addrs?: string[];
};

type CheckHostContext = {
    name: string;
    qtype: string | null;
    client: string | null;
    filteringStatus: FilterList;
    rewritesList: RewriteEntry[];
    blockedServicesList: BlockedServicesList;
    settingsStatuses: SettingsStatuses;
};

type UserRulesSetupOptions = {
    filteringStatus?: FilterList;
    settingsStatuses?: SettingsStatuses;
    rewritesList?: RewriteEntry[];
    blockedServicesList?: BlockedServicesList;
    allBlockedServices?: Array<{ id: string; name: string }>;
    checkHostResolver?: (context: CheckHostContext) => CheckHostResponse;
};

type UserRulesSetupResult = {
    checkHostRequests: URL[];
    setRulesPayloads: Array<{ rules?: string[] }>;
};

const DEFAULT_FILTERING_STATUS: FilterList = {
    enabled: true,
    filters: [],
    whitelist_filters: [],
    user_rules: [],
    interval: 24,
};

const DEFAULT_SETTINGS_STATUSES: SettingsStatuses = {
    safebrowsing: { enabled: true },
    parental: { enabled: true },
    safesearch: {
        enabled: false,
        bing: true,
        duckduckgo: true,
        google: true,
        pixabay: true,
        yandex: true,
        youtube: true,
    },
};

const DEFAULT_BLOCKED_SERVICES_LIST: BlockedServicesList = {
    ids: [],
    schedule: {
        time_zone: 'Local',
    },
};

const DEFAULT_ALL_BLOCKED_SERVICES = [{ id: 'youtube', name: 'YouTube' }];

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

const buildDefaultCheckHostResponse = ({
    name,
    qtype,
    filteringStatus,
    rewritesList,
    blockedServicesList,
    settingsStatuses,
}: CheckHostContext): CheckHostResponse => {
    const allowRule = filteringStatus.user_rules.find((rule) => rule.startsWith(`@@||${name}^`));
    const matchingRule = filteringStatus.user_rules.find(
        (rule) => !rule.startsWith('@@') && rule.includes(`||${name}^`),
    );
    const matchingRewrite = rewritesList.find((rewrite) => rewrite.domain === name);

    if (allowRule) {
        return {
            reason: 'NotFilteredWhiteList',
            rules: [{ filter_list_id: 0, text: allowRule }],
        };
    }

    if (matchingRewrite) {
        return {
            reason: 'RewriteRule',
            rules: [],
            ...(matchingRewrite.answer.includes('.') &&
            !matchingRewrite.answer.match(/^\d+\.\d+\.\d+\.\d+$/)
                ? { cname: matchingRewrite.answer }
                : { ip_addrs: [matchingRewrite.answer] }),
        };
    }

    if (name === 'service.example' && blockedServicesList.ids.includes('youtube')) {
        return {
            reason: 'FilteredBlockedService',
            rules: [],
            service_name: 'YouTube',
        };
    }

    if (name === 'malware.example' && settingsStatuses.safebrowsing.enabled) {
        return {
            reason: 'FilteredSafeBrowsing',
            rules: [],
        };
    }

    if (name === 'adult.example' && settingsStatuses.parental.enabled) {
        return {
            reason: 'FilteredParental',
            rules: [],
        };
    }

    if (name === 'search.example' && settingsStatuses.safesearch.enabled) {
        return {
            reason: 'FilteredSafeSearch',
            rules: [],
        };
    }

    if (matchingRule) {
        if (matchingRule.includes('$dnstype=CNAME') && qtype !== 'CNAME') {
            return {
                reason: 'NotFilteredNotFound',
                rules: [],
            };
        }

        return {
            reason: 'FilteredBlackList',
            rules: [{ filter_list_id: 0, text: matchingRule }],
        };
    }

    return {
        reason: 'NotFilteredNotFound',
        rules: [],
    };
};

async function setupUserRulesMocks(
    page: Page,
    {
        filteringStatus = DEFAULT_FILTERING_STATUS,
        settingsStatuses = DEFAULT_SETTINGS_STATUSES,
        rewritesList = [],
        blockedServicesList = DEFAULT_BLOCKED_SERVICES_LIST,
        allBlockedServices = DEFAULT_ALL_BLOCKED_SERVICES,
        checkHostResolver = buildDefaultCheckHostResponse,
    }: UserRulesSetupOptions = {},
): Promise<UserRulesSetupResult> {
    const checkHostRequests: URL[] = [];
    const setRulesPayloads: Array<{ rules?: string[] }> = [];
    let filteringStatusState = clone(filteringStatus);
    let settingsState = clone(settingsStatuses);
    const rewritesListState = clone(rewritesList);
    let blockedServicesState = clone(blockedServicesList);

    await page.route('**/control/filtering/status', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(filteringStatusState),
        });
    });

    await page.route('**/control/safebrowsing/status', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(settingsState.safebrowsing),
        });
    });

    await page.route('**/control/parental/status', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(settingsState.parental),
        });
    });

    await page.route('**/control/safesearch/status', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(settingsState.safesearch),
        });
    });

    await page.route('**/control/rewrite/list', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(rewritesListState),
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

    await page.route('**/control/filtering/set_rules', (route) => {
        const payload = route.request().postDataJSON() as { rules?: string[] };
        setRulesPayloads.push(payload);
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

    await page.route('**/control/filtering/check_host*', (route) => {
        const requestUrl = new URL(route.request().url());

        checkHostRequests.push(requestUrl);
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(
                checkHostResolver({
                    name: requestUrl.searchParams.get('name') ?? '',
                    qtype: requestUrl.searchParams.get('qtype'),
                    client: requestUrl.searchParams.get('client'),
                    filteringStatus: clone(filteringStatusState),
                    rewritesList: clone(rewritesListState),
                    blockedServicesList: clone(blockedServicesState),
                    settingsStatuses: clone(settingsState),
                }),
            ),
        });
    });

    await page.route('**/control/safebrowsing/disable', (route) => {
        settingsState = {
            ...settingsState,
            safebrowsing: { enabled: false },
        };

        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    await page.route('**/control/parental/disable', (route) => {
        settingsState = {
            ...settingsState,
            parental: { enabled: false },
        };

        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    await page.route('**/control/safesearch/settings', async (route) => {
        settingsState = {
            ...settingsState,
            safesearch: route.request().postDataJSON() as SettingsStatuses['safesearch'],
        };

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    await page.route('**/control/blocked_services/update', async (route) => {
        blockedServicesState = route.request().postDataJSON() as BlockedServicesList;

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({}),
        });
    });

    return {
        checkHostRequests,
        setRulesPayloads,
    };
}

async function openUserRules(page: Page, options?: UserRulesSetupOptions) {
    const setup = await setupUserRulesMocks(page, options);

    await login(page);
    await page.goto('/#user_rules');
    await expect(page.getByTestId('user-rules-check-submit')).toBeVisible();

    return setup;
}

async function selectDnsRecordType(page: Page, value: string) {
    const control = page.getByTestId('user-rules-check-qtype').locator('[data-part="control"]');

    await control.scrollIntoViewIfNeeded();
    await control.click();
    await expect(page.locator('[data-part="content"]')).toBeVisible();
    await page.locator('[data-part="content"]').getByText(value, { exact: true }).click();
    await page.keyboard.press('Tab');
}

test.describe('User rules desktop', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test('saves custom rules from the editor', async ({ page }) => {
        const { setRulesPayloads } = await openUserRules(page);

        await page.getByTestId('user-rules-editor-textarea').fill('||editor.example^');
        await page.getByTestId('user-rules-editor-save').click();

        await expect.poll(() => setRulesPayloads.length).toBe(1);
        expect(setRulesPayloads[0].rules).toEqual(['||editor.example^']);
        await expect(page.getByTestId('toast').last()).toContainText(
            'Custom rules successfully saved',
        );
    });

    test('checks a qtype-specific rule and refreshes the result after allowlisting', async ({
        page,
    }) => {
        const { checkHostRequests, setRulesPayloads } = await openUserRules(page, {
            filteringStatus: {
                ...DEFAULT_FILTERING_STATUS,
                user_rules: ['||qtype.example^$dnstype=CNAME'],
            },
        });

        await page.getByTestId('user-rules-check-hostname').fill('qtype.example');
        await selectDnsRecordType(page, 'CNAME');
        await page.getByTestId('user-rules-check-submit').click();

        await expect.poll(() => checkHostRequests.length).toBe(1);
        expect(checkHostRequests[0].searchParams.get('qtype')).toBe('CNAME');
        await expect(page.getByTestId('user-rules-result-title')).toHaveText('Domain is blocked');

        await page.getByTestId('user-rules-result-action-allow').click();

        await expect.poll(() => setRulesPayloads.length).toBe(1);
        expect(setRulesPayloads[0].rules?.filter(Boolean)).toEqual([
            '||qtype.example^$dnstype=CNAME',
            '@@||qtype.example^$important',
        ]);
        await expect.poll(() => checkHostRequests.length).toBe(2);
        expect(checkHostRequests[1].searchParams.get('qtype')).toBe('CNAME');
        await expect(page.getByTestId('toast')).toHaveCount(1);
        await expect(page.getByTestId('toast').last()).toContainText('Rule added to allowlist');
        await expect(page.getByTestId('toast-action')).toHaveText('Undo');
        await expect(page.getByTestId('user-rules-result-title')).toHaveText('Domain is allowed');
    });

    test('offers both actions for blocked services and allows the service', async ({ page }) => {
        await openUserRules(page, {
            blockedServicesList: {
                ...DEFAULT_BLOCKED_SERVICES_LIST,
                ids: ['youtube'],
            },
        });

        await page.getByTestId('user-rules-check-hostname').fill('service.example');
        await selectDnsRecordType(page, 'A');
        await page.getByTestId('user-rules-check-submit').click();

        await expect(page.getByTestId('user-rules-result-action-allow')).toHaveText(
            'Add to allowlist',
        );
        await expect(
            page.getByTestId('user-rules-result-action-disable-blocked-service'),
        ).toHaveText('Allow service');

        await page.getByTestId('user-rules-result-action-disable-blocked-service').click();

        await expect(page.getByTestId('toast').last()).toContainText(
            'The YouTube service was allowed',
        );
    });
});

test.describe('User rules mobile', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test('keeps the check and result flow usable on a mobile viewport', async ({ page }) => {
        await page.setViewportSize({ width: 390, height: 844 });

        await openUserRules(page, {
            filteringStatus: {
                ...DEFAULT_FILTERING_STATUS,
                user_rules: ['||mobile.example^'],
            },
        });

        await page.getByTestId('user-rules-check-hostname').fill('mobile.example');
        await selectDnsRecordType(page, 'A');
        await page.getByTestId('user-rules-check-submit').click();

        await expect(page.getByTestId('user-rules-check-submit')).toBeVisible();
        await expect(page.getByTestId('user-rules-result-card')).toBeVisible();
        await expect(page.getByTestId('user-rules-result-action-allow')).toBeVisible();
    });
});
