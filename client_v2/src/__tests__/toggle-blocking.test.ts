import { beforeEach, describe, expect, it, vi } from 'vitest';

import { toggleBlocking, BLOCK_ACTIONS, filteringState } from 'panel/stores/filtering';

let lastSetRules = '';

const mocks = vi.hoisted(() => ({
    apiSetRules: vi.fn((params: any) => {
        lastSetRules = params?.rules || '';
        return Promise.resolve(undefined);
    }),
    apiGetFilteringStatus: vi.fn(() =>
        Promise.resolve({
            user_rules: lastSetRules,
            filters: [],
            whitelist_filters: [],
            interval: 24,
            enabled: true,
        }),
    ),
    addSuccessToast: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        setRules: mocks.apiSetRules,
        getFilteringStatus: mocks.apiGetFilteringStatus,
    },
}));

vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
    createUndoToast: (message: any, actionLabel: string, onUndo: () => void | Promise<void>) => ({
        message,
        actionLabel,
        undoId: 'test-undo-id',
        onUndo,
    }),
}));

describe('toggleBlocking', () => {
    beforeEach(async () => {
        lastSetRules = '';
        mocks.apiSetRules.mockClear();
        mocks.addSuccessToast.mockClear();
        mocks.addErrorToast.mockClear();
        // Reset store state - the default mock impl returns lastSetRules which is ''
        const { setRules } = await import('panel/stores/filtering');
        await setRules('');
    });

    it('adds a blocking rule when no rule exists', async () => {
        const result = await toggleBlocking(BLOCK_ACTIONS.BLOCK, 'example.com');

        expect(result).toBe(true);
        expect(mocks.apiSetRules).toHaveBeenCalled();
        expect(filteringState.userRules).toContain('||example.com^$important');
    });

    it('replaces an allowlist rule with a blocking rule', async () => {
        // Set up initial state with an allowlist rule
        const { setRules } = await import('panel/stores/filtering');
        await setRules('@@||allowed.example^$important\n');
        // After setRules, lastSetRules = '@@||allowed.example^$important\n'
        // and filteringState.userRules should match

        const result = await toggleBlocking(BLOCK_ACTIONS.BLOCK, 'allowed.example');

        expect(result).toBe(true);
        expect(filteringState.userRules).toContain('||allowed.example^$important');
    });

    it('replaces a blocking rule with an allowlist rule', async () => {
        const { setRules } = await import('panel/stores/filtering');
        await setRules('||blocked.example^$important\n');

        const result = await toggleBlocking(BLOCK_ACTIONS.UNBLOCK, 'blocked.example');

        expect(result).toBe(true);
        expect(filteringState.userRules).toContain('@@||blocked.example^$important');
    });
});
