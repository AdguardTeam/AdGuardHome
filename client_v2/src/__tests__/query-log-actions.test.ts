import { beforeEach, describe, expect, it, vi } from 'vitest';

import intl from 'panel/common/intl';
import { blockDomain, unblockDomain } from 'panel/actions';

const mocks = vi.hoisted(() => ({
    setRules: vi.fn((rules, options) => ({ type: 'setRules', payload: { rules, options } })),
    getFilteringStatus: vi.fn(() => ({ type: 'getFilteringStatus' })),
    addSuccessToast: vi.fn((payload) => ({ type: 'addSuccessToast', payload })),
}));

vi.mock('panel/actions/filtering', () => ({
    setRules: mocks.setRules,
    getFilteringStatus: mocks.getFilteringStatus,
}));

vi.mock('panel/actions/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: vi.fn(),
    addNoticeToast: vi.fn(),
}));

describe('query-log block and unblock toasts', () => {
    const dispatch = vi.fn(async (action) => action);

    beforeEach(() => {
        dispatch.mockClear();
        mocks.setRules.mockClear();
        mocks.getFilteringStatus.mockClear();
        mocks.addSuccessToast.mockClear();
    });

    it('suppresses the generic custom-rules toast when blocking a domain from query log', async () => {
        const getState = () => ({ filtering: { userRules: '' } });

        await blockDomain('fresh.example')(dispatch, getState as never);

        expect(mocks.setRules).toHaveBeenCalledWith('||fresh.example^$important\n', {
            showToast: false,
        });
        expect(mocks.addSuccessToast).toHaveBeenCalledTimes(1);
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(
            expect.objectContaining({
                code: 'notify_user_rule_added',
                message: intl.getMessage('notify_user_rule_added', {
                    rule: '||fresh.example^$important',
                }),
            }),
        );
    });

    it('suppresses the generic custom-rules toast when unblocking a domain from query log', async () => {
        const getState = () => ({ filtering: { userRules: '||blocked.example^$important\n' } });

        await unblockDomain('blocked.example')(dispatch, getState as never);

        expect(mocks.setRules).toHaveBeenCalledWith('@@||blocked.example^$important\n', {
            showToast: false,
        });
        expect(mocks.addSuccessToast).toHaveBeenCalledTimes(1);
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(
            expect.objectContaining({ code: 'notify_user_rule_added' }),
        );
    });
});
