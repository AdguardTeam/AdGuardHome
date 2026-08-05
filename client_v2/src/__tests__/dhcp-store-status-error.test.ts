import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
    status: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    status: mocks.status,
    dhcpStatus: vi.fn(),
    dhcpInterfaces: vi.fn(),
    checkActiveDhcp: vi.fn(),
    dhcpSetConfig: vi.fn(),
    dhcpReset: vi.fn(),
    dhcpResetLeases: vi.fn(),
    dhcpAddStaticLease: vi.fn(),
    dhcpRemoveStaticLease: vi.fn(),
    dhcpUpdateStaticLease: vi.fn(),
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: vi.fn(),
}));

import { dhcpState, getDhcpStatus } from 'panel/stores/dhcp';

describe('getDhcpStatus errors', () => {
    beforeEach(() => vi.clearAllMocks());

    it('distinguishes a request failure from an unavailable DHCP server', async () => {
        const requestError = new Error('network unavailable');
        mocks.status.mockRejectedValueOnce(requestError);

        await getDhcpStatus();

        expect(dhcpState.statusError).toBe(true);
        expect(dhcpState.processing).toBe(false);
        expect(dhcpState.processingStatus).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalledWith({ error: requestError });

        let resolveStatus!: (value: { dhcp_available: boolean }) => void;
        mocks.status.mockReturnValueOnce(
            new Promise((resolve) => {
                resolveStatus = resolve;
            }),
        );

        const retry = getDhcpStatus();
        expect(dhcpState.statusError).toBe(false);
        expect(dhcpState.processingStatus).toBe(true);

        resolveStatus({ dhcp_available: false });
        await retry;

        expect(dhcpState.statusError).toBe(false);
        expect(dhcpState.dhcp_available).toBe(false);
        expect(dhcpState.processing).toBe(false);
        expect(dhcpState.processingStatus).toBe(false);
    });
});
