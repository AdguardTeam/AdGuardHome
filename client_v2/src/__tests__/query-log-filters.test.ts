import { describe, expect, test } from 'vitest';

import {
    DEFAULT_LOGS_FILTER,
    QUERY_LOG_REASON_FILTER,
    QUERY_LOG_STATUS_FILTER,
} from '../helpers/constants';
import { getLogsUrlParams } from '../helpers/helpers';

describe('Query Log filter model', () => {
    test('stores search, status, and reason separately', () => {
        expect(DEFAULT_LOGS_FILTER).toEqual({
            search: '',
            status: 'all',
            reason: 'all',
        });
    });

    test('serializes all three filter fields into the URL', () => {
        expect(getLogsUrlParams('example.org', 'blocked', 'FilteredBlockedService')).toBe(
            '?search=example.org&status=blocked&reason=FilteredBlockedService',
        );
    });

    test('keeps status and reason option sets separate', () => {
        expect(QUERY_LOG_STATUS_FILTER.REWRITTEN.QUERY).toBe('rewritten');
        expect(Object.keys(QUERY_LOG_STATUS_FILTER)).not.toContain('ERROR');
        expect(QUERY_LOG_REASON_FILTER.BLOCKED_BY_FILTER.QUERY).toBe('FilteredBlackList');
        expect(QUERY_LOG_REASON_FILTER.SAFE_SEARCH.QUERY).toBe('FilteredSafeSearch');
    });
});
