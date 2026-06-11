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
        expect(safeBrowsingMeta.rule).toBe('adguard-malware-shavar');
        expect(blockedServicesMeta.reason).toBe(intl.getMessage('blocked_services'));
        expect(blockedServicesMeta.rule).toBe('||amemv.com^');
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
