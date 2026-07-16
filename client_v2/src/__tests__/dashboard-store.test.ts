import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    getTlsStatus: vi.fn(),
    getGlobalVersion: vi.fn(),
    getProfile: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    addNoticeToast: vi.fn(),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        baseUrl: 'http://x',
        getGlobalVersion: mocks.getGlobalVersion,
        getProfile: mocks.getProfile,
        getUpdate: vi.fn(),
        getTlsStatus: mocks.getTlsStatus,
    },
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
    addNoticeToast: mocks.addNoticeToast,
}));

import { getDnsStatus } from 'panel/stores/dashboard';

describe('getDnsStatus', () => {
    beforeEach(() => vi.clearAllMocks());

    it('fetches TLS status when the core is running', async () => {
        mocks.getGlobalVersion.mockResolvedValue({
            disabled: true,
            new_version: 'x',
        });
        mocks.getProfile.mockResolvedValue({ name: 'n', theme: 't' });
        mocks.getTlsStatus.mockResolvedValue({ enabled: false });

        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            new Response(
                JSON.stringify({
                    running: true,
                    version: 'v',
                    dns_port: 53,
                    dns_addresses: [],
                    protection_enabled: true,
                    http_port: 80,
                }),
                { status: 200 },
            ),
        );

        await getDnsStatus();
        // allow microtasks for chained getVersion/getTlsStatus/getProfile
        await new Promise((r) => setTimeout(r, 0));
        expect(mocks.getTlsStatus).toHaveBeenCalled();
    });
});
