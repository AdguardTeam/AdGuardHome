import { beforeEach, describe, expect, it, vi } from 'vitest';

import filtering from 'panel/reducers/filtering';
import { setRules } from 'panel/actions/filtering';

const mocks = vi.hoisted(() => ({
    apiSetRules: vi.fn(() => Promise.resolve(undefined)),
    addSuccessToast: vi.fn((payload) => ({ type: 'addSuccessToast', payload })),
    addErrorToast: vi.fn((payload) => ({ type: 'addErrorToast', payload })),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        setRules: mocks.apiSetRules,
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
