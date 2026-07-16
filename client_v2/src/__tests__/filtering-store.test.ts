import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
    setRules,
    addFiltersBatch,
    getFilteringStatus,
    filteringState,
} from 'panel/stores/filtering';

const mocks = vi.hoisted(() => ({
    apiSetRules: vi.fn(() => Promise.resolve(undefined)),
    apiAddFilter: vi.fn(() => Promise.resolve(undefined)),
    apiGetFilteringStatus: vi.fn(() => Promise.resolve({})),
    addSuccessToast: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        setRules: mocks.apiSetRules,
        addFilter: mocks.apiAddFilter,
        getFilteringStatus: mocks.apiGetFilteringStatus,
    },
}));

vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
}));

describe('getFilteringStatus', () => {
    beforeEach(() => vi.clearAllMocks());

    it('normalizes filters to camelCase and joins user_rules array', async () => {
        mocks.apiGetFilteringStatus.mockResolvedValue({
            enabled: true,
            interval: 24,
            filters: [
                {
                    id: 1,
                    url: 'u',
                    enabled: true,
                    last_updated: 'x',
                    name: 'n',
                    rules_count: 9,
                },
            ],
            whitelist_filters: [],
            user_rules: ['||a^', '||b^'],
        });

        await getFilteringStatus();

        expect(filteringState.userRules).toBe('||a^\n||b^');
        expect(filteringState.filters[0]).toEqual({
            id: 1,
            url: 'u',
            enabled: true,
            lastUpdated: 'x',
            name: 'n',
            rulesCount: 9,
        });
    });
});

describe('setRules', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('sends user_rules as a string array', async () => {
        const result = await setRules('||fresh.example^\n');

        expect(mocks.apiSetRules).toHaveBeenCalledWith({
            rules: ['||fresh.example^'],
        });

        expect(result).toBe(true);
        expect(filteringState.userRules).toBe('||fresh.example^\n');
        expect(filteringState.processingRules).toBe(false);
    });

    it('returns false and shows error toast on failure', async () => {
        mocks.apiSetRules.mockRejectedValueOnce(new Error('Network error'));
        const result = await setRules('||test.example^\n');

        expect(result).toBe(false);
        expect(filteringState.processingRules).toBe(false);
    });
});

describe('addFiltersBatch', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('adds all filters successfully and shows aggregate toast', async () => {
        const filters = [
            { url: 'https://example.com/filter1.txt', name: 'Filter One' },
            { url: 'https://example.com/filter2.txt', name: 'Filter Two' },
            { url: 'https://example.com/filter3.txt', name: 'Filter Three' },
        ];

        await addFiltersBatch(filters);

        expect(mocks.apiAddFilter).toHaveBeenCalledTimes(3);
        expect(mocks.addSuccessToast).toHaveBeenCalled();
        expect(filteringState.processingAddFilter).toBe(false);
    });

    it('handles partial failures', async () => {
        mocks.apiAddFilter
            .mockResolvedValueOnce(undefined)
            .mockRejectedValueOnce(new Error('Failed'));

        const filters = [
            { url: 'https://example.com/filter1.txt', name: 'Filter One' },
            { url: 'https://example.com/filter2.txt', name: 'Filter Two' },
        ];

        await addFiltersBatch(filters);

        expect(mocks.addSuccessToast).toHaveBeenCalled();
        expect(mocks.addErrorToast).toHaveBeenCalled();
        expect(filteringState.processingAddFilter).toBe(false);
    });
});
