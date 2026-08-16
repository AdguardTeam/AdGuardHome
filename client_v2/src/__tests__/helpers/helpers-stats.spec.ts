import { describe, expect, it } from 'vitest';

import {
    normalizeTopStats,
    addClientInfo,
    normalizeTopClients,
    normalizeFilters,
    normalizeFilteringStatus,
    getParamsForClientsSearch,
    checkFiltered,
    checkBlockedService,
    getPathWithQueryString,
    getSpecialFilterName,
    getServiceName,
    normalizeWhois,
    normalizeLogs,
} from '../../helpers/helpers';
import type { ClientsFindEntry } from '../../api/model/clientsFindEntry';
import type { FilteringReason } from '../../api/model/filteringReason';

describe('normalizeTopStats', () => {
    it('converts {name -> count} objects to {name, count} array', () => {
        expect(normalizeTopStats([{ '192.168.1.1': 42 }, { 'example.com': 5 }])).toStrictEqual([
            { name: '192.168.1.1', count: 42 },
            { name: 'example.com', count: 5 },
        ]);
    });
});

describe('addClientInfo', () => {
    it('resolves client info by param key', () => {
        const data = [{ name: '192.168.1.1', count: 1 }];
        const clients: ClientsFindEntry[] = [{ '192.168.1.1': { name: 'MyPhone' } }];
        expect(addClientInfo(data, clients, 'name')).toStrictEqual([
            { name: '192.168.1.1', count: 1, info: { name: 'MyPhone' } },
        ]);
    });
});

describe('normalizeTopClients', () => {
    it('splits into auto/configured by name and info', () => {
        const r = normalizeTopClients([
            { name: '192.168.1.1', count: 7, info: { name: 'MyPhone' } },
        ]);
        expect(r.auto).toStrictEqual({ '192.168.1.1': 7 });
        expect(r.configured).toStrictEqual({ MyPhone: 7 });
    });
});

describe('normalizeFilters', () => {
    it('maps snake_case API fields to camelCase with defaults', () => {
        expect(
            normalizeFilters([
                {
                    id: 1,
                    url: 'http://example.com/list.txt',
                    enabled: true,
                    last_updated: '2024-01-01',
                    name: 'My List',
                    rules_count: 100,
                },
            ]),
        ).toStrictEqual([
            {
                id: 1,
                url: 'http://example.com/list.txt',
                enabled: true,
                lastUpdated: '2024-01-01',
                name: 'My List',
                rulesCount: 100,
            },
        ]);
    });
    it('returns [] for falsy input', () => {
        expect(normalizeFilters(undefined)).toStrictEqual([]);
    });
});

describe('normalizeFilteringStatus', () => {
    it('normalizes full status with user_rules', () => {
        const r = normalizeFilteringStatus({
            enabled: true,
            filters: [],
            whitelist_filters: [],
            user_rules: ['rule1', 'rule2'],
            interval: 24,
        });
        expect(r.enabled).toBe(true);
        expect(r.interval).toBe(24);
        expect(r.userRules).toBe('rule1\nrule2');
    });
});

describe('getParamsForClientsSearch', () => {
    it('collects unique client ids from TopStat[]', () => {
        expect(
            getParamsForClientsSearch(
                [
                    { name: 'client-a', count: 1 },
                    { name: 'client-b', count: 2 },
                ],
                'name',
            ),
        ).toStrictEqual({ clients: [{ id: 'client-a' }, { id: 'client-b' }] });
    });
    it('includes additional param when provided', () => {
        const r = getParamsForClientsSearch([{ name: 'a', count: 1 }], 'name', 'count');
        expect(r.clients).toStrictEqual([{ id: 'a' }, { id: 1 }]);
    });
});

describe('checkFiltered / checkBlockedService', () => {
    it('checkFiltered returns true for Filtered* reasons', () => {
        expect(checkFiltered('FilteredBlackList' as FilteringReason)).toBe(true);
    });
    it('checkFiltered returns false for NotFiltered* reasons', () => {
        expect(checkFiltered('NotFilteredNotFound' as FilteringReason)).toBe(false);
    });
    it('checkBlockedService returns true for FilteredBlockedService', () => {
        expect(checkBlockedService('FilteredBlockedService' as FilteringReason)).toBe(true);
    });
});

describe('getPathWithQueryString', () => {
    it('serializes params, skips empty/undefined values, repeats arrays', () => {
        const r = getPathWithQueryString('/endpoint', {
            a: '1',
            b: '',
            c: undefined,
            d: ['x', 'y'],
        });
        expect(r).toBe('/endpoint?a=1&d=x&d=y');
    });
    it('handles null params gracefully', () => {
        const r = getPathWithQueryString('/p', { a: '1', b: null });
        expect(r).toBe('/p?a=1');
    });
    it('handles undefined params arg', () => {
        expect(getPathWithQueryString('/p', undefined)).toBe('/p?');
    });
});

describe('getSpecialFilterName', () => {
    it('returns localized name for known special filter IDs', () => {
        expect(typeof getSpecialFilterName(0)).toBe('string');
        expect(typeof getSpecialFilterName(-1)).toBe('string');
        expect(typeof getSpecialFilterName(-5)).toBe('string');
    });
});

describe('getServiceName', () => {
    it('returns name for matching service id', () => {
        expect(getServiceName([{ id: 'svc1', name: 'My Service' }], 'svc1')).toBe('My Service');
    });
    it('returns undefined for unknown id', () => {
        expect(getServiceName([{ id: 'svc1', name: 'My Service' }], 'svc-unknown')).toBeUndefined();
    });
});

describe('normalizeWhois', () => {
    it('derives location from city and country', () => {
        expect(normalizeWhois({ city: 'NY', country: 'US', orgname: 'Example' })).toMatchObject({
            location: 'US, NY',
            orgname: 'Example',
        });
    });
    it('uses only country when city absent', () => {
        expect(normalizeWhois({ country: 'DE', orgname: 'Org' })).toMatchObject({
            location: 'DE',
        });
    });
    it('returns placeholder defaults for empty whois', () => {
        expect(normalizeWhois({})).toMatchObject({
            location: 'New York, US',
            orgname: 'Example Organization',
        });
    });
});

describe('normalizeLogs', () => {
    it('maps query log item to normalized shape', () => {
        const [item] = normalizeLogs([
            {
                time: '2024-01-01T00:00:00Z',
                question: {
                    name: 'example.com',
                    unicode_name: 'example.com',
                    type: 'A',
                },
                answer: [{ value: '1.2.3.4', type: 'A', ttl: 60 }],
                status: 'processed',
            },
        ]);
        expect(item.domain).toBe('example.com');
        expect(item.unicodeName).toBe('example.com');
        expect(item.type).toBe('A');
        expect(item.response).toStrictEqual([{ value: '1.2.3.4', type: 'A', ttl: 60 }]);
    });
});
