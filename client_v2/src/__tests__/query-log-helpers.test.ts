import { afterEach, describe, expect, test, vi } from 'vitest';

import intl from 'panel/common/intl';
import { FILTERED_STATUS, SPECIAL_FILTER_ID } from '../helpers/constants';
import {
    filterLogsByStatus,
    getQueryReasonLabel,
    getQueryReasonDetails,
    getQueryReasonKey,
    getQueryStatusLabel,
    getQueryStatusKey,
} from '../components/QueryLog/helpers';

afterEach(() => {
    vi.restoreAllMocks();
});

describe('getQueryStatusKey', () => {
    test('maps safe search and rewrites to rewritten status', () => {
        expect(getQueryStatusKey(FILTERED_STATUS.FILTERED_SAFE_SEARCH)).toBe('rewritten');
        expect(getQueryStatusKey(FILTERED_STATUS.REWRITE)).toBe('rewritten');
    });

    test('maps allowlisted and plain answers to allowed/processed', () => {
        expect(getQueryStatusKey(FILTERED_STATUS.NOT_FILTERED_WHITE_LIST)).toBe('allowed');
        expect(getQueryStatusKey(FILTERED_STATUS.NOT_FILTERED_NOT_FOUND)).toBe('processed');
    });

    test('falls back to rewritten when the row has an original response', () => {
        expect(
            getQueryStatusKey('UnexpectedReason', [{ value: '203.0.113.10', type: 'A', ttl: 300 }]),
        ).toBe('rewritten');
    });
});

describe('getQueryReasonKey', () => {
    test('keeps custom rules separate from blocklist rules', () => {
        expect(
            getQueryReasonKey(FILTERED_STATUS.FILTERED_BLACK_LIST, [
                {
                    filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                    text: '||example.org^',
                },
            ]),
        ).toBe('custom_filtering_rules');

        expect(
            getQueryReasonKey(FILTERED_STATUS.FILTERED_BLACK_LIST, [
                { filter_list_id: 42, text: '||example.org^' },
            ]),
        ).toBe('blocked_by_filter');
    });

    test('maps threat, parental, safe-search, allowlist, and rewrite reasons', () => {
        expect(getQueryReasonKey(FILTERED_STATUS.FILTERED_SAFE_BROWSING, [])).toBe(
            'blocked_threats',
        );
        expect(getQueryReasonKey(FILTERED_STATUS.FILTERED_PARENTAL, [])).toBe(
            'blocked_by_parental_control',
        );
        expect(getQueryReasonKey(FILTERED_STATUS.FILTERED_SAFE_SEARCH, [])).toBe('safe_search');
        expect(getQueryReasonKey(FILTERED_STATUS.NOT_FILTERED_WHITE_LIST, [])).toBe('allowlists');
        expect(getQueryReasonKey(FILTERED_STATUS.REWRITE, [])).toBe('dns_rewrites');
    });
});

describe('query log label helpers', () => {
    test('maps status keys through static intl keys', () => {
        const getMessageSpy = vi.spyOn(intl, 'getMessage').mockImplementation((key) => key);

        expect(getQueryStatusLabel('blocked')).toBe('query_log_blocked');
        expect(getQueryStatusLabel('allowed')).toBe('query_log_allowed');
        expect(getQueryStatusLabel('processed')).toBe('query_log_processed');
        expect(getQueryStatusLabel('rewritten')).toBe('query_log_rewritten');
        expect(getQueryStatusLabel('error')).toBe('error');

        expect(getMessageSpy).toHaveBeenNthCalledWith(1, 'query_log_blocked');
        expect(getMessageSpy).toHaveBeenNthCalledWith(2, 'query_log_allowed');
        expect(getMessageSpy).toHaveBeenNthCalledWith(3, 'query_log_processed');
        expect(getMessageSpy).toHaveBeenNthCalledWith(4, 'query_log_rewritten');
        expect(getMessageSpy).toHaveBeenNthCalledWith(5, 'error');
    });

    test('maps reason keys through static intl keys and keeps none literal', () => {
        const getMessageSpy = vi.spyOn(intl, 'getMessage').mockImplementation((key) => key);

        expect(getQueryReasonLabel('none')).toBe('-');
        expect(getQueryReasonLabel('blocked_by_filter')).toBe('query_log_blocked_by_filter');
        expect(getQueryReasonLabel('custom_filtering_rules')).toBe(
            'query_log_custom_filtering_rules',
        );
        expect(getQueryReasonLabel('blocked_services')).toBe('query_log_blocked_services');
        expect(getQueryReasonLabel('blocked_threats')).toBe('query_log_blocked_threats');
        expect(getQueryReasonLabel('blocked_by_parental_control')).toBe(
            'query_log_blocked_by_parental_control',
        );
        expect(getQueryReasonLabel('safe_search')).toBe('query_log_safe_search');
        expect(getQueryReasonLabel('dns_rewrites')).toBe('dns_rewrites');
        expect(getQueryReasonLabel('allowlists')).toBe('allowlists');
        expect(getQueryReasonLabel('error')).toBe('error');

        expect(getMessageSpy).toHaveBeenNthCalledWith(1, 'query_log_blocked_by_filter');
        expect(getMessageSpy).toHaveBeenNthCalledWith(2, 'query_log_custom_filtering_rules');
        expect(getMessageSpy).toHaveBeenNthCalledWith(3, 'query_log_blocked_services');
        expect(getMessageSpy).toHaveBeenNthCalledWith(4, 'query_log_blocked_threats');
        expect(getMessageSpy).toHaveBeenNthCalledWith(5, 'query_log_blocked_by_parental_control');
        expect(getMessageSpy).toHaveBeenNthCalledWith(6, 'query_log_safe_search');
        expect(getMessageSpy).toHaveBeenNthCalledWith(7, 'dns_rewrites');
        expect(getMessageSpy).toHaveBeenNthCalledWith(8, 'allowlists');
        expect(getMessageSpy).toHaveBeenNthCalledWith(9, 'error');
    });
});

describe('getQueryReasonDetails', () => {
    const filters = [
        {
            enabled: true,
            id: 42,
            lastUpdated: '',
            name: 'AdGuard DNS filter',
            rulesCount: 0,
            url: '',
        },
    ];

    const whitelistFilters = [
        {
            enabled: true,
            id: 7,
            lastUpdated: '',
            name: "Dan Pollock's List",
            rulesCount: 0,
            url: '',
        },
    ];

    const services = [{ id: 'amazon', name: 'Amazon' }];

    test('returns filter names for blocklist and allowlist reasons', () => {
        expect(
            getQueryReasonDetails({
                elapsedMs: '17',
                filters,
                reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
                rules: [{ filter_list_id: 42, text: '||example.org^' }],
                serviceName: '',
                services,
                whitelistFilters,
            }),
        ).toBe('AdGuard DNS filter');

        expect(
            getQueryReasonDetails({
                elapsedMs: '31',
                filters,
                reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
                rules: [{ filter_list_id: 7, text: '@@||example.org^' }],
                serviceName: '',
                services,
                whitelistFilters,
            }),
        ).toBe("Dan Pollock's List");
    });

    test('returns the service name for blocked-services rows', () => {
        expect(
            getQueryReasonDetails({
                elapsedMs: '37',
                filters,
                reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
                rules: [],
                serviceName: 'amazon',
                services,
                whitelistFilters,
            }),
        ).toBe('Amazon');
    });
});

describe('filterLogsByStatus', () => {
    const logs = [
        { reason: FILTERED_STATUS.NOT_FILTERED_NOT_FOUND, originalResponse: [] },
        { reason: FILTERED_STATUS.FILTERED_BLACK_LIST, originalResponse: [] },
        { reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST, originalResponse: [] },
        { reason: FILTERED_STATUS.REWRITE, originalResponse: [] },
    ] as any[];

    test('filters logs by generic status category', () => {
        expect(filterLogsByStatus(logs, 'processed')).toHaveLength(1);
        expect(filterLogsByStatus(logs, 'blocked')).toHaveLength(1);
        expect(filterLogsByStatus(logs, 'allowed')).toHaveLength(1);
        expect(filterLogsByStatus(logs, 'rewritten')).toHaveLength(1);
        expect(filterLogsByStatus(logs, 'all')).toHaveLength(4);
    });
});
