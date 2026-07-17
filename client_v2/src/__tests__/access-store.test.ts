import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    accessList: vi.fn(),
    accessSet: vi.fn(),
    addSuccessToast: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
        accessList: mocks.accessList,
        accessSet: mocks.accessSet,
}));
vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
}));

import { toggleClientBlock } from 'panel/stores/access';

describe('toggleClientBlock', () => {
    beforeEach(() => vi.clearAllMocks());

    it('not-disallowed + allowlist mode with >1 allowed → removes from allowed', async () => {
        mocks.accessList.mockResolvedValue({
            allowed_clients: ['1.1.1.1', '2.2.2.2'],
            disallowed_clients: [],
            blocked_hosts: [],
        });
        await toggleClientBlock('1.1.1.1', false, '');
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: ['2.2.2.2'],
            disallowed_clients: [],
            blocked_hosts: [],
        });
    });

    it('not-disallowed, no allowlist → adds to disallowed', async () => {
        mocks.accessList.mockResolvedValue({
            allowed_clients: [],
            disallowed_clients: [],
            blocked_hosts: [],
        });
        await toggleClientBlock('3.3.3.3', false, '');
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: [],
            disallowed_clients: ['3.3.3.3'],
            blocked_hosts: [],
        });
    });

    it('disallowed + allowlist mode → adds to allowed', async () => {
        mocks.accessList.mockResolvedValue({
            allowed_clients: ['1.1.1.1'],
            disallowed_clients: ['2.2.2.2'],
            blocked_hosts: [],
        });
        await toggleClientBlock('2.2.2.2', true, '');
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: ['1.1.1.1', '2.2.2.2'],
            disallowed_clients: ['2.2.2.2'],
            blocked_hosts: [],
        });
    });

    it('disallowed, no allowlist → removes from disallowed (uses rule)', async () => {
        mocks.accessList.mockResolvedValue({
            allowed_clients: [],
            disallowed_clients: ['client:X'],
            blocked_hosts: [],
        });
        await toggleClientBlock('1.2.3.4', true, 'client:X');
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: [],
            disallowed_clients: [],
            blocked_hosts: [],
        });
    });
});
