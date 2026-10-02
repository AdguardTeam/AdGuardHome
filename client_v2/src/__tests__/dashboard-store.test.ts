import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    tlsStatus: vi.fn(),
    getVersionJson: vi.fn(),
    getProfile: vi.fn(),
    clientsStatus: vi.fn(),
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
    clientsStatus: mocks.clientsStatus,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
    addNoticeToast: mocks.addNoticeToast,
}));

import { getDnsStatus, getClients, dashboardState } from 'panel/stores/dashboard';

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

describe('getClients — clientsInitialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        dashboardState.clientsInitialized = false;
    });

    it('marks clientsInitialized after a successful request', async () => {
        mocks.clientsStatus.mockResolvedValue({ clients: [], auto_clients: [] });

        await getClients();

        expect(dashboardState.clientsInitialized).toBe(true);
    });

    it('marks clientsInitialized after a failed request so the loader cannot get stuck', async () => {
        mocks.clientsStatus.mockRejectedValue(new Error('network'));

        await getClients();

        expect(dashboardState.clientsInitialized).toBe(true);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});
