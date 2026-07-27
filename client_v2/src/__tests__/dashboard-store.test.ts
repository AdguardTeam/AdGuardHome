import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    tlsStatus: vi.fn(),
    getVersionJson: vi.fn(),
    getProfile: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    addNoticeToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    baseUrl: 'http://x',
    getStatusUrl: () => 'control/status',
    getVersionJson: mocks.getVersionJson,
    getProfile: mocks.getProfile,
    beginUpdate: vi.fn(),
    tlsStatus: mocks.tlsStatus,
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
        mocks.getVersionJson.mockResolvedValue({
            disabled: true,
            new_version: 'x',
        });
        mocks.getProfile.mockResolvedValue({ name: 'n', theme: 't' });
        mocks.tlsStatus.mockResolvedValue({ enabled: false });

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
        expect(mocks.tlsStatus).toHaveBeenCalled();
    });
});
