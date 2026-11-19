import { describe, expect, it } from 'vitest';

import intl from 'panel/common/intl';
import { FILTERED_STATUS, SPECIAL_FILTER_ID } from 'panel/helpers/constants';

import { getCheckResultMeta } from '../components/UserRules/checkResultHelpers';

describe('getCheckResultMeta', () => {
    const filters = [
        {
            id: 101,
            name: 'Example Blocklist',
            url: 'https://filters.example/blocklist.txt',
            enabled: true,
            lastUpdated: '',
            rulesCount: 12,
        },
        {
            id: 1793714189,
            name: 'Multi list',
            url: 'https://filters.example/multi.txt',
            enabled: true,
            lastUpdated: '',
            rulesCount: 2,
        },
    ];
    const whitelistFilters = [
        {
            id: 201,
            name: 'Example Allowlist',
            url: 'https://filters.example/allowlist.txt',
            enabled: true,
            lastUpdated: '',
            rulesCount: 7,
        },
    ];

    it('localizes filter and allowlist reasons through translation keys', () => {
        const blockedMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 101, text: '||filtered.example^' }],
            filters,
            whitelistFilters: [],
        });

        const allowedMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
            rules: [{ filter_list_id: 201, text: '@@||allowed.example^$important' }],
            filters: [],
            whitelistFilters,
        });

        expect(blockedMeta.reason).toBeUndefined();
        expect(blockedMeta.source).toBe('Example Blocklist');
        expect(blockedMeta.sourceListType).toBe('blocklist');
        expect(allowedMeta.reason).toBeUndefined();
        expect(allowedMeta.source).toBe('Example Allowlist');
        expect(allowedMeta.sourceListType).toBe('allowlist');
        expect(allowedMeta.actions).toEqual([
            {
                kind: 'disable-filter',
                label: intl.getMessage('user_rules_disable_filter'),
            },
        ]);
    });

    it('localizes custom filtering and protection reasons through translation keys', () => {
        const customMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [
                {
                    filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                    text: '||blocked.example^',
                },
            ],
            filters: [],
            whitelistFilters: [],
        });

        const safeBrowsingMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_SAFE_BROWSING,
            rules: [
                { filter_list_id: SPECIAL_FILTER_ID.SAFE_BROWSING, text: 'adguard-malware-shavar' },
            ],
            filters: [],
            whitelistFilters: [],
        });

        const blockedServicesMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
            rules: [{ filter_list_id: 0, text: '||amemv.com^' }],
            filters: [],
            whitelistFilters: [],
        });

        expect(customMeta.reason).toBe(
            intl.getMessage('user_rules_reason_blocked_by', {
                source: intl.getMessage('custom_filtering_rules'),
            }),
        );
        expect(safeBrowsingMeta.reason).toBe(intl.getMessage('blocked_threats'));
        expect(safeBrowsingMeta.source).toBe(intl.getMessage('safe_browsing'));
        expect(safeBrowsingMeta.ruleGroups).toEqual([
            {
                source: intl.getMessage('safe_browsing'),
                sourceListType: undefined,
                rules: [
                    {
                        filter_list_id: SPECIAL_FILTER_ID.SAFE_BROWSING,
                        text: 'adguard-malware-shavar',
                    },
                ],
            },
        ]);
        expect(blockedServicesMeta.reason).toBe(intl.getMessage('blocked_services'));
        expect(blockedServicesMeta.rules).toEqual([{ filter_list_id: 0, text: '||amemv.com^' }]);
    });

    it('exposes every rule grouped by filter list and derives source from the first one', () => {
        const rules = [
            { filter_list_id: 101, text: '||filtered.example^' },
            { filter_list_id: 201, text: '@@||allowed.example^$important' },
            { filter_list_id: 101, text: '||filtered.example^' },
        ];

        const meta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules,
            filters,
            whitelistFilters,
        });

        expect(meta.rules).toEqual(rules);
        expect(meta.ruleGroups).toEqual([
            {
                source: 'Example Blocklist',
                sourceListType: 'blocklist',
                rules: [rules[0], rules[2]],
            },
            {
                source: 'Example Allowlist',
                sourceListType: 'allowlist',
                rules: [rules[1]],
            },
        ]);
        expect(meta.source).toBe('Example Blocklist');
        expect(meta.sourceListType).toBe('blocklist');
        expect(meta.actions).toEqual([
            {
                kind: 'allow',
                label: intl.getMessage('user_rules_add_to_allowlist'),
            },
            {
                kind: 'disable-filter',
                label: intl.getMessage('user_rules_disable_filter'),
            },
        ]);
    });

    it('groups the QA multi-list fixture under each rule\'s own source', () => {
        const rules = [
            {
                filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                text: '127.0.0.1 hosts.example',
            },
            { filter_list_id: 1793714189, text: '0.0.0.0 hosts.example' },
        ];

        const meta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules,
            filters,
            whitelistFilters: [],
        });

        expect(meta.ruleGroups).toEqual([
            {
                source: intl.getMessage('custom_rules'),
                sourceListType: undefined,
                rules: [rules[0]],
            },
            {
                source: 'Multi list',
                sourceListType: 'blocklist',
                rules: [rules[1]],
            },
        ]);
        expect(meta.source).toBe(intl.getMessage('custom_rules'));
    });

    it('puts rules without a filter list id in an unlabeled group', () => {
        const rules = [{ text: '||custom.example^' }];

        const meta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules,
            filters,
            whitelistFilters,
        });

        expect(meta.ruleGroups).toEqual([
            {
                source: undefined,
                sourceListType: undefined,
                rules,
            },
        ]);
    });

    it('keeps empty-text rules in ruleGroups for the renderer to skip', () => {
        const rules = [{ filter_list_id: SPECIAL_FILTER_ID.SAFE_SEARCH, text: '' }];

        const meta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_SAFE_SEARCH,
            rules,
            filters: [],
            whitelistFilters: [],
        });

        expect(meta.ruleGroups).toEqual([
            {
                source: intl.getMessage('safe_search'),
                sourceListType: undefined,
                rules,
            },
        ]);
    });

    it('suppresses the generic rules section for parental control responses', () => {
        const rules = [
            {
                filter_list_id: SPECIAL_FILTER_ID.PARENTAL,
                text: 'parental CATEGORY_BLACKLISTED',
            },
        ];

        const meta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_PARENTAL,
            rules,
            filters: [],
            whitelistFilters: [],
        });

        expect(meta.showRulesSection).toBe(false);
        expect(meta.ruleGroups).toEqual([
            {
                source: intl.getMessage('parental_control'),
                sourceListType: undefined,
                rules,
            },
        ]);
    });

    it('keeps the generic rules section enabled for other reasons', () => {
        const blockedMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 101, text: '||filtered.example^' }],
            filters,
            whitelistFilters: [],
        });

        const safeBrowsingMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_SAFE_BROWSING,
            rules: [
                { filter_list_id: SPECIAL_FILTER_ID.SAFE_BROWSING, text: 'adguard-malware-shavar' },
            ],
            filters: [],
            whitelistFilters: [],
        });

        expect(blockedMeta.showRulesSection).not.toBe(false);
        expect(safeBrowsingMeta.showRulesSection).not.toBe(false);
    });

    it('detects a custom rule for actions from the first rule only', () => {
        const meta = getCheckResultMeta({
            reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
            rules: [
                {
                    filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                    text: '@@||allowed.example^$important',
                },
                { filter_list_id: 101, text: '||filtered.example^' },
            ],
            filters,
            whitelistFilters: [],
        });

        expect(meta.actions).toEqual([
            {
                kind: 'block',
                label: intl.getMessage('block'),
            },
        ]);
    });

    it('renders Safe Search as a rewritten result with the Safe search status', () => {
        const safeSearchMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_SAFE_SEARCH,
            rules: [],
            filters: [],
            whitelistFilters: [],
        });

        expect(safeSearchMeta.tone).toBe('rewritten');
        expect(safeSearchMeta.title).toBe(intl.getMessage('user_rules_rewrite_rule_is_applied'));
        expect(safeSearchMeta.reason).toBe(intl.getMessage('settings_safe_search'));
        expect(safeSearchMeta.actions).toEqual([
            {
                kind: 'allow',
                label: intl.getMessage('user_rules_add_to_allowlist'),
            },
            {
                kind: 'disable-safesearch',
                label: intl.getMessage('user_rules_disable_safe_search'),
            },
        ]);
    });

    it('omits filter and allowlist reasons when the source name is unavailable', () => {
        const blockedMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 999, text: '||filtered.example^' }],
            filters: [],
            whitelistFilters: [],
        });

        const allowedMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
            rules: [{ filter_list_id: 999, text: '@@||allowed.example^$important' }],
            filters: [],
            whitelistFilters: [],
        });

        expect(blockedMeta.reason).toBeUndefined();
        expect(allowedMeta.reason).toBeUndefined();
    });

    it('keeps custom allowlist actions limited to block', () => {
        const customAllowedMeta = getCheckResultMeta({
            reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
            rules: [
                {
                    filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                    text: '@@||allowed.example^$important',
                },
            ],
            filters: [],
            whitelistFilters: [],
        });

        expect(customAllowedMeta.actions).toEqual([
            {
                kind: 'block',
                label: intl.getMessage('block'),
            },
        ]);
    });
});
