import { beforeEach, describe, expect, it, vi } from 'vitest';

import { setRules, addFiltersBatch, filteringState } from 'panel/stores/filtering';

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

describe('setRules', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('updates the stored user rules immediately after save succeeds', async () => {
        const result = await setRules('||fresh.example^\n');

        expect(mocks.apiSetRules).toHaveBeenCalledWith({
            rules: '||fresh.example^\n',
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
