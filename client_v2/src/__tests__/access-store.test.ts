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

import { toggleClientBlock, setAccessList, getAccessList, accessState } from 'panel/stores/access';
import { resetStore } from 'panel/__tests__/helpers/resetStore';

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

describe('setAccessList', () => {
    beforeEach(async () => {
        vi.clearAllMocks();
        mocks.accessList.mockResolvedValue({
            allowed_clients: ['10.0.0.1'],
            disallowed_clients: ['10.0.0.2'],
            blocked_hosts: ['blocked.example'],
        });
        await getAccessList();
    });

    it('splits newline-delimited strings into arrays', async () => {
        await setAccessList({
            allowed_clients: '1.1.1.1\n2.2.2.2',
            disallowed_clients: '3.3.3.3',
            blocked_hosts: 'badhost.com\nevil.com',
        });
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: ['1.1.1.1', '2.2.2.2'],
            disallowed_clients: ['3.3.3.3'],
            blocked_hosts: ['badhost.com', 'evil.com'],
        });
    });

    it('keeps the lists that the caller did not pass', async () => {
        await setAccessList({ allowed_clients: '1.1.1.1' });
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: ['1.1.1.1'],
            disallowed_clients: ['10.0.0.2'],
            blocked_hosts: ['blocked.example'],
        });
    });

    it('reads the server lists before saving instead of trusting the store', async () => {
        // The store still holds the lists loaded in `beforeEach`, but the
        // server has moved on; composing the payload from the store would
        // revert the lists the operator did not change.
        mocks.accessList.mockResolvedValue({
            allowed_clients: ['10.0.0.9'],
            disallowed_clients: ['10.0.0.8'],
            blocked_hosts: ['new.example'],
        });

        await setAccessList({ allowed_clients: '1.1.1.1' });

        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: ['1.1.1.1'],
            disallowed_clients: ['10.0.0.8'],
            blocked_hosts: ['new.example'],
        });
    });

    it('treats an empty string as an intentionally emptied list', async () => {
        await setAccessList({ allowed_clients: '' });
        expect(mocks.accessSet).toHaveBeenCalledWith({
            allowed_clients: [],
            disallowed_clients: ['10.0.0.2'],
            blocked_hosts: ['blocked.example'],
        });
        expect(accessState.allowed_clients).toBe('');
    });

    it('does not save when the pre-save read fails', async () => {
        mocks.accessList.mockRejectedValue(new Error('network'));

        const saved = await setAccessList({ allowed_clients: '1.1.1.1' });

        expect(saved).toBe(false);
        expect(mocks.accessSet).not.toHaveBeenCalled();
        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(mocks.addErrorToast).toHaveBeenCalled();
        expect(accessState.processingSet).toBe(false);
        expect(accessState.allowed_clients).toBe('10.0.0.1');
    });

    it('writes the saved lists back to the store', async () => {
        await setAccessList({ blocked_hosts: 'badhost.com\nevil.com' });

        expect(accessState.blocked_hosts).toBe('badhost.com\nevil.com');
        expect(accessState.allowed_clients).toBe('10.0.0.1');
        expect(accessState.disallowed_clients).toBe('10.0.0.2');
    });
});

describe('getAccessList — initialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        resetStore(accessState, { initialized: false });
    });

    it('marks initialized after a successful request', async () => {
        mocks.accessList.mockResolvedValue({});

        const loaded = await getAccessList();

        expect(loaded).toBe(true);
        expect(accessState.initialized).toBe(true);
    });

    it('marks initialized after a failed request so the loader cannot get stuck', async () => {
        mocks.accessList.mockRejectedValue(new Error('network'));

        const loaded = await getAccessList();

        expect(loaded).toBe(false);
        expect(accessState.initialized).toBe(true);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});
