import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
    safebrowsingStatus: vi.fn(),
    parentalStatus: vi.fn(),
    safesearchStatus: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    safebrowsingStatus: mocks.safebrowsingStatus,
    safebrowsingEnable: vi.fn(),
    safebrowsingDisable: vi.fn(),
    parentalStatus: mocks.parentalStatus,
    parentalEnable: vi.fn(),
    parentalDisable: vi.fn(),
    safesearchStatus: mocks.safesearchStatus,
    safesearchSettings: vi.fn(),
    testUpstreamDNS: vi.fn(),
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/common/intl', () => ({
    default: { getMessage: (key: string) => key },
}));

const loadStore = async () => {
    vi.resetModules();
    return import('panel/stores/settings');
};

const deferred = <T>() => {
    let resolve!: (value: T | PromiseLike<T>) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return { promise, reject, resolve };
};

describe('settings status initialization', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.safebrowsingStatus.mockResolvedValue({ enabled: true });
        mocks.parentalStatus.mockResolvedValue({ enabled: false });
        mocks.safesearchStatus.mockResolvedValue({ enabled: true, google: true });
    });

    it('starts with feature status unknown', async () => {
        const { settingsState } = await loadStore();

        expect(settingsState.loadStatus).toBe('idle');
    });

    it('marks feature status loaded only after all status requests succeed', async () => {
        const { initSettings, settingsState } = await loadStore();

        const request = initSettings();
        expect(settingsState.loadStatus).toBe('loading');

        await request;

        expect(settingsState.loadStatus).toBe('loaded');
        expect(settingsState.processing).toBe(false);
        expect(settingsState.settingsList).toEqual({
            safebrowsing: { enabled: true },
            parental: { enabled: false },
            safesearch: { enabled: true, google: true },
        });
    });

    it('marks feature status failed when any status request fails', async () => {
        const error = new Error('status unavailable');
        mocks.parentalStatus.mockRejectedValue(error);
        const { initSettings, settingsState } = await loadStore();

        await initSettings();

        expect(settingsState.loadStatus).toBe('failed');
        expect(settingsState.processing).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalledWith({ error });
    });

    it('does not let an older successful request overwrite a newer response', async () => {
        const oldSafebrowsing = deferred<{ enabled: boolean }>();
        const oldParental = deferred<{ enabled: boolean }>();
        const oldSafesearch = deferred<{ enabled: boolean; google: boolean }>();
        mocks.safebrowsingStatus
            .mockImplementationOnce(() => oldSafebrowsing.promise)
            .mockResolvedValueOnce({ enabled: false });
        mocks.parentalStatus
            .mockImplementationOnce(() => oldParental.promise)
            .mockResolvedValueOnce({ enabled: true });
        mocks.safesearchStatus
            .mockImplementationOnce(() => oldSafesearch.promise)
            .mockResolvedValueOnce({ enabled: false, google: false });
        const { initSettings, settingsState } = await loadStore();

        const olderRequest = initSettings();
        const newerRequest = initSettings();
        await newerRequest;
        oldSafebrowsing.resolve({ enabled: true });
        oldParental.resolve({ enabled: false });
        oldSafesearch.resolve({ enabled: true, google: true });
        await olderRequest;

        expect(settingsState.loadStatus).toBe('loaded');
        expect(settingsState.settingsList).toEqual({
            safebrowsing: { enabled: false },
            parental: { enabled: true },
            safesearch: { enabled: false, google: false },
        });
    });

    it('does not let an older failure replace a newer successful response', async () => {
        const error = new Error('stale status failure');
        const oldSafebrowsing = deferred<{ enabled: boolean }>();
        const oldParental = deferred<{ enabled: boolean }>();
        const oldSafesearch = deferred<{ enabled: boolean; google: boolean }>();
        mocks.safebrowsingStatus
            .mockImplementationOnce(() => oldSafebrowsing.promise)
            .mockResolvedValueOnce({ enabled: false });
        mocks.parentalStatus
            .mockImplementationOnce(() => oldParental.promise)
            .mockResolvedValueOnce({ enabled: true });
        mocks.safesearchStatus
            .mockImplementationOnce(() => oldSafesearch.promise)
            .mockResolvedValueOnce({ enabled: false, google: false });
        const { initSettings, settingsState } = await loadStore();

        const olderRequest = initSettings();
        const newerRequest = initSettings();
        await newerRequest;
        oldSafebrowsing.resolve({ enabled: true });
        oldParental.reject(error);
        oldSafesearch.resolve({ enabled: true, google: true });
        await olderRequest;

        expect(settingsState.loadStatus).toBe('loaded');
        expect(settingsState.settingsList).toEqual({
            safebrowsing: { enabled: false },
            parental: { enabled: true },
            safesearch: { enabled: false, google: false },
        });
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });
});
