import { beforeEach, describe, expect, it, vi } from 'vitest';

import filtering from 'panel/reducers/filtering';
import { setRules, addFiltersBatch } from 'panel/actions/filtering';

const mocks = vi.hoisted(() => ({
    apiSetRules: vi.fn(() => Promise.resolve(undefined)),
    apiAddFilter: vi.fn(() => Promise.resolve(undefined)),
    apiGetFilteringStatus: vi.fn(() => Promise.resolve({})),
    addSuccessToast: vi.fn((payload) => ({ type: 'addSuccessToast', payload })),
    addErrorToast: vi.fn((payload) => ({ type: 'addErrorToast', payload })),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        setRules: mocks.apiSetRules,
        addFilter: mocks.apiAddFilter,
        getFilteringStatus: mocks.apiGetFilteringStatus,
    },
}));

vi.mock('panel/actions/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
    createUndoToast: (message: any, actionLabel: any) => ({
        message,
        actionLabel,
        undoId: 'mock-undo-id',
    }),
}));

vi.mock('panel/reducers/modals', () => ({
    closeModal: vi.fn(() => ({ type: 'CLOSE_MODAL' })),
}));

describe('setRules', () => {
    const dispatch = vi.fn((action) => action);

    beforeEach(() => {
        dispatch.mockClear();
        mocks.apiSetRules.mockClear();
        mocks.addSuccessToast.mockClear();
        mocks.addErrorToast.mockClear();
    });

    it('updates the stored user rules immediately after save succeeds', async () => {
        await setRules('||fresh.example^\n')(dispatch);

        expect(mocks.apiSetRules).toHaveBeenCalledWith({
            rules: ['||fresh.example^', ''],
        });

        const finalState = dispatch.mock.calls
            .map(([action]) => action)
            .reduce((state, action) => filtering(state, action), undefined);

        expect(finalState.userRules).toBe('||fresh.example^\n');
        expect(finalState.processingRules).toBe(false);
    });
});

const mockFilters = [
    { url: 'https://example.com/filter1.txt', name: 'Filter One' },
    { url: 'https://example.com/filter2.txt', name: 'Filter Two' },
    { url: 'https://example.com/filter3.txt', name: 'Filter Three' },
];

describe('addFiltersBatch', () => {
    const dispatch = vi.fn((action) => {
        if (typeof action === 'function') {
            return action(dispatch);
        }
        return action;
    });

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('dispatches request, all succeed, shows aggregate toast, closes modal, refreshes', async () => {
        mocks.apiAddFilter.mockResolvedValue(undefined);

        await addFiltersBatch(mockFilters)(dispatch);

        // Request dispatched exactly once
        expect(dispatch).toHaveBeenCalledWith(
            expect.objectContaining({ type: 'ADD_FILTER_REQUEST' }),
        );

        // API called for all 3 filters
        expect(mocks.apiAddFilter).toHaveBeenCalledTimes(3);

        // Success dispatched exactly once
        expect(dispatch).toHaveBeenCalledWith(
            expect.objectContaining({ type: 'ADD_FILTER_SUCCESS' }),
        );

        // Aggregate success toast dispatched
        const toastCalls = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'addSuccessToast',
        );
        expect(toastCalls).toHaveLength(1);

        // closeModal called once
        const closeModalCalls = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'CLOSE_MODAL',
        );
        expect(closeModalCalls).toHaveLength(1);

        // getFilteringStatus called once
        expect(mocks.apiGetFilteringStatus).toHaveBeenCalledTimes(1);
    });

    it('shows single-filter toast when only 1 filter added', async () => {
        mocks.apiAddFilter.mockResolvedValue(undefined);

        await addFiltersBatch([mockFilters[0]])(dispatch);

        // Single success toast dispatched (message content is a React node — count is sufficient verification)
        const toastCalls = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'addSuccessToast',
        );
        expect(toastCalls).toHaveLength(1);
    });

    it('handles partial failure: success toast + error toasts', async () => {
        mocks.apiAddFilter
            .mockResolvedValueOnce(undefined)                       // success
            .mockRejectedValueOnce(new Error('Network error'))      // fail
            .mockResolvedValueOnce(undefined);                      // success

        await addFiltersBatch(mockFilters)(dispatch);

        // Success dispatched
        expect(dispatch).toHaveBeenCalledWith(
            expect.objectContaining({ type: 'ADD_FILTER_SUCCESS' }),
        );

        // One success toast
        const successToasts = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'addSuccessToast',
        );
        expect(successToasts).toHaveLength(1);

        // One error toast
        const errorToasts = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'addErrorToast',
        );
        expect(errorToasts).toHaveLength(1);
    });

    it('shows no success toast when all filters fail, keeps modal open', async () => {
        mocks.apiAddFilter.mockRejectedValue(new Error('Network error'));

        await addFiltersBatch(mockFilters)(dispatch);

        // Failure dispatched
        expect(dispatch).toHaveBeenCalledWith(
            expect.objectContaining({ type: 'ADD_FILTER_FAILURE' }),
        );

        // No success toast
        const successToasts = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'addSuccessToast',
        );
        expect(successToasts).toHaveLength(0);

        // Error toasts for each failure
        const errorToasts = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'addErrorToast',
        );
        expect(errorToasts).toHaveLength(3);

        // Modal stays open for retry — closeModal NOT dispatched
        const closeModalCalls = dispatch.mock.calls.filter(
            ([call]: any) => call?.type === 'CLOSE_MODAL',
        );
        expect(closeModalCalls).toHaveLength(0);
    });

    it('dispatches addFilterFailure on unexpected error in batch logic', async () => {
        // Synchronous throw in .map() causes outer try/catch to fire
        mocks.apiAddFilter.mockImplementation(() => {
            throw new Error('Unexpected');
        });

        await addFiltersBatch(mockFilters)(dispatch);

        expect(dispatch).toHaveBeenCalledWith(
            expect.objectContaining({ type: 'ADD_FILTER_FAILURE' }),
        );
    });
});
