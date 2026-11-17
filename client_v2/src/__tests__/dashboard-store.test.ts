import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

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

import { resetStore } from 'panel/__tests__/helpers/resetStore';
import { copy } from 'panel/__tests__/helpers/copy';
import {
    getDnsStatus,
    getTimerStatus,
    getClients,
    getVersion,
    refreshDnsStatus,
    stopStatusChecks,
    dashboardState,
} from 'panel/stores/dashboard';

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

    it('loads the profile so the name and theme are available', async () => {
        mocks.getVersionJson.mockResolvedValue({
            disabled: true,
            new_version: 'x',
        });
        mocks.getProfile.mockResolvedValue({ name: 'admin', theme: 'dark' });
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
        await new Promise((r) => setTimeout(r, 0));

        // The profile drives the footer's "logged in" state and the stored
        // theme, so it must land in the store rather than be discarded.
        expect(dashboardState.name).toBe('admin');
        expect(dashboardState.theme).toBe('dark');
        expect(dashboardState.processingProfile).toBe(false);
    });

    it('keeps the server theme out of the store for a userless install', async () => {

        mocks.getVersionJson.mockResolvedValue({
            disabled: true,
            new_version: 'x',
        });
        // A userless install returns the config default for the theme, not a
        // preference of the guest.
        mocks.getProfile.mockResolvedValue({ name: '', theme: 'auto' });
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
        await new Promise((r) => setTimeout(r, 0));

        expect(dashboardState.name).toBe('');
        expect(dashboardState.theme).toBeUndefined();
        expect(dashboardState.processingProfile).toBe(false);
    });
});

describe('refreshDnsStatus', () => {
    beforeEach(() => vi.clearAllMocks());

    /** Serves `body` as the next status response. */
    const respondWithStatus = (body: Record<string, unknown>) =>
        vi
            .spyOn(globalThis, 'fetch')
            .mockResolvedValue(new Response(JSON.stringify(body), { status: 200 }));

    it('updates the DNS addresses and ports', async () => {
        respondWithStatus({
            running: true,
            dns_port: 53,
            dns_addresses: ['https://dns.example.com:443/dns-query', '1.2.3.4'],
            http_port: 80,
        });

        await refreshDnsStatus();

        expect(dashboardState.dnsAddresses).toEqual([
            'https://dns.example.com:443/dns-query',
            '1.2.3.4',
        ]);
        expect(dashboardState.dnsPort).toBe(53);
        expect(dashboardState.httpPort).toBe(80);
    });

    it('keeps the previous addresses and stays quiet when the request fails', async () => {
        respondWithStatus({
            running: true,
            dns_addresses: ['1.2.3.4'],
            dns_port: 53,
            http_port: 80,
        });
        await refreshDnsStatus();

        vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));
        await refreshDnsStatus();

        expect(dashboardState.dnsAddresses).toEqual(['1.2.3.4']);
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });

    it('keeps the previous addresses while the core is not running', async () => {
        respondWithStatus({
            running: true,
            dns_addresses: ['1.2.3.4'],
            dns_port: 53,
            http_port: 80,
        });
        await refreshDnsStatus();

        respondWithStatus({ running: false });
        await refreshDnsStatus();

        expect(dashboardState.dnsAddresses).toEqual(['1.2.3.4']);
    });
});

describe('getClients — clientsInitialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        resetStore(dashboardState, { clientsInitialized: false });
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

describe('getVersion — update check failures', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        resetStore(dashboardState, {
            dnsVersion: 'v1.0.0',
            processingVersion: true,
            isUpdateAvailable: false,
        });
    });

    it('keeps the retry control available and shows a localized error after a failure', async () => {
        mocks.getVersionJson.mockRejectedValue(new Error('network'));

        await getVersion(true);

        expect(mocks.addErrorToast).toHaveBeenCalledWith({
            error: copy('version_request_error'),
        });
        expect(dashboardState.processingVersion).toBe(false);
        expect(dashboardState.versionCheckFailed).toBe(true);
        expect(dashboardState.checkUpdateFlag).toBe(false);
    });

    it('hides the recheck button when updates are disabled', async () => {
        mocks.getVersionJson.mockResolvedValue({ disabled: true });

        await getVersion();

        expect(mocks.addErrorToast).not.toHaveBeenCalled();
        expect(dashboardState.processingVersion).toBe(false);
        expect(dashboardState.checkUpdateFlag).toBe(false);
        expect(dashboardState.versionCheckFailed).toBe(false);
    });

    it('shows no success toast on a recheck when updates are disabled', async () => {
        mocks.getVersionJson.mockResolvedValue({ disabled: true });

        await getVersion(true);

        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(dashboardState.checkUpdateFlag).toBe(false);
        expect(dashboardState.versionCheckFailed).toBe(false);
    });

    it('shows a success toast on a successful recheck', async () => {
        mocks.getVersionJson.mockResolvedValue({
            disabled: false,
            new_version: 'v1.1.0',
        });

        await getVersion(true);

        expect(mocks.addErrorToast).not.toHaveBeenCalled();
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(copy('updates_checked'));
        expect(dashboardState.checkUpdateFlag).toBe(true);
        expect(dashboardState.isUpdateAvailable).toBe(true);
    });
});

const runningStatus = {
    running: true,
    version: 'v',
    dns_port: 53,
    dns_addresses: [] as string[],
    protection_enabled: true,
    http_port: 80,
};

/** jsdom's own `location.reload` cannot be spied on, so swap the object. */
const stubLocation = () => {
    const reload = vi.fn();
    vi.spyOn(window, 'location', 'get').mockReturnValue({ reload } as unknown as Location);
    return reload;
};

describe('status checks — ownership of retries', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
        resetStore(dashboardState, {
            processing: true,
            statusFailed: false,
            isCoreRunning: true,
            dnsVersion: '',
        });
        mocks.getVersionJson.mockResolvedValue({ disabled: true, new_version: 'v' });
        mocks.getProfile.mockResolvedValue({ name: 'admin', theme: 'dark' });
        mocks.tlsStatus.mockResolvedValue({ enabled: false });
    });

    afterEach(() => {
        stopStatusChecks();
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    it('does not let a timer check cancel the DNS status retry', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch');
        fetchMock.mockRejectedValueOnce(new Error('network'));
        fetchMock.mockResolvedValue(new Response(JSON.stringify(runningStatus), { status: 200 }));

        getDnsStatus();
        await vi.advanceTimersByTimeAsync(0);

        // The tab becomes visible while the DNS check is waiting to retry, so
        // a second chain starts and must leave the first one's timer alone.
        await getTimerStatus();
        await vi.advanceTimersByTimeAsync(1000);

        expect(fetchMock).toHaveBeenCalledTimes(3);
        expect(dashboardState.processing).toBe(false);
        expect(dashboardState.dnsVersion).toBe('v');
    });

    it('starts one chain per entry point instead of stacking them', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

        getTimerStatus();
        await vi.advanceTimersByTimeAsync(0);
        getTimerStatus();
        await vi.advanceTimersByTimeAsync(0);

        // The second visibility event replaced the pending retry of the first.
        await vi.advanceTimersByTimeAsync(1000);

        expect(fetchMock).toHaveBeenCalledTimes(3);
    });

    it('stops every pending retry on request', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

        getDnsStatus();
        getTimerStatus();
        await vi.advanceTimersByTimeAsync(0);

        stopStatusChecks();
        await vi.advanceTimersByTimeAsync(5000);

        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });
});

describe('status checks — all attempts failed', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
        resetStore(dashboardState, {
            processing: true,
            isCoreRunning: true,
            dnsVersion: '',
        });
        mocks.getVersionJson.mockResolvedValue({ disabled: true, new_version: 'v' });
        mocks.getProfile.mockResolvedValue({ name: 'admin', theme: 'dark' });
        mocks.tlsStatus.mockResolvedValue({ enabled: false });
    });

    afterEach(() => {
        stopStatusChecks();
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    it('keeps the notice, flags only a failed bootstrap, and does not reload', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));
        const reload = stubLocation();

        // A failing timer chain must not replace a loaded dashboard.
        getTimerStatus();
        await vi.advanceTimersByTimeAsync(61_000);

        expect(fetchMock).toHaveBeenCalledTimes(60);
        expect(mocks.addErrorToast).toHaveBeenCalledWith({ error: copy('dns_status_error') });
        expect(dashboardState.statusFailed).toBe(false);
        expect(reload).not.toHaveBeenCalled();

        fetchMock.mockClear();
        mocks.addErrorToast.mockClear();

        // A failing bootstrap leaves the dashboard unknown: the retry surface
        // must show up instead of the route body.
        getDnsStatus();
        await vi.advanceTimersByTimeAsync(61_000);

        expect(fetchMock).toHaveBeenCalledTimes(60);
        expect(mocks.addErrorToast).toHaveBeenCalledWith({ error: copy('dns_status_error') });
        expect(dashboardState.processing).toBe(false);
        expect(dashboardState.statusFailed).toBe(true);
        expect(reload).not.toHaveBeenCalled();
    });

    it('clears the failed flag when a retry succeeds', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

        getDnsStatus();
        await vi.advanceTimersByTimeAsync(61_000);
        expect(dashboardState.statusFailed).toBe(true);

        fetchMock.mockResolvedValue(new Response(JSON.stringify(runningStatus), { status: 200 }));
        getDnsStatus();
        await vi.advanceTimersByTimeAsync(0);

        expect(dashboardState.statusFailed).toBe(false);
        expect(dashboardState.processing).toBe(false);
        expect(dashboardState.dnsVersion).toBe('v');
    });
});
