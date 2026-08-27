import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mocks, capturedUndo } = vi.hoisted(() => {
    const capturedUndo: { current?: () => Promise<void> } = {};

    const mocks = {
        safebrowsingStatus: vi.fn(),
        safebrowsingEnable: vi.fn(),
        safebrowsingDisable: vi.fn(),
        parentalStatus: vi.fn(),
        parentalEnable: vi.fn(),
        parentalDisable: vi.fn(),
        safesearchStatus: vi.fn(),
        safesearchSettings: vi.fn(),
        testUpstreamDNS: vi.fn(),
        addSuccessToast: vi.fn(),
        addErrorToast: vi.fn(),
        createUndoToast: vi.fn(
            (message: string, actionLabel: string, onUndo: () => Promise<void>) => {
                capturedUndo.current = onUndo;
                return { message, actionLabel, undoId: 'test-undo-id' };
            },
        ),
    };

    return { mocks, capturedUndo };
});

vi.mock('panel/api/generated', () => ({
    safebrowsingStatus: mocks.safebrowsingStatus,
    safebrowsingEnable: mocks.safebrowsingEnable,
    safebrowsingDisable: mocks.safebrowsingDisable,
    parentalStatus: mocks.parentalStatus,
    parentalEnable: mocks.parentalEnable,
    parentalDisable: mocks.parentalDisable,
    safesearchStatus: mocks.safesearchStatus,
    safesearchSettings: mocks.safesearchSettings,
    testUpstreamDNS: mocks.testUpstreamDNS,
}));

vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
    createUndoToast: mocks.createUndoToast,
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
    },
}));

import {
    disableParental,
    disableSafeBrowsing,
    disableSafeSearch,
    initSettings,
    settingsState,
} from 'panel/stores/settings';

const triggerUndo = async () => {
    if (!capturedUndo.current) {
        throw new Error('Undo callback was not registered');
    }
    await capturedUndo.current();
};

const seedSafeSearchConfig = {
    enabled: true,
    google: true,
    bing: false,
    youtube: true,
};

beforeEach(async () => {
    vi.clearAllMocks();
    capturedUndo.current = undefined;

    mocks.safebrowsingStatus.mockResolvedValue({ enabled: true });
    mocks.parentalStatus.mockResolvedValue({ enabled: true });
    mocks.safesearchStatus.mockResolvedValue(seedSafeSearchConfig);
    mocks.safebrowsingDisable.mockResolvedValue(undefined);
    mocks.safebrowsingEnable.mockResolvedValue(undefined);
    mocks.parentalDisable.mockResolvedValue(undefined);
    mocks.parentalEnable.mockResolvedValue(undefined);
    mocks.safesearchSettings.mockResolvedValue(undefined);

    await initSettings();
});

describe('disableSafeBrowsing', () => {
    it('disables safebrowsing, updates state, and offers undo', async () => {
        const result = await disableSafeBrowsing();

        expect(mocks.safebrowsingDisable).toHaveBeenCalledTimes(1);
        expect(settingsState.settingsList.safebrowsing.enabled).toBe(false);
        expect(mocks.createUndoToast).toHaveBeenCalledWith(
            'user_rules_browsing_security_disabled',
            'notify_undo',
            expect.any(Function),
        );
        expect(mocks.addSuccessToast).toHaveBeenCalled();
        expect(result).toBe(true);
    });

    it('undo re-enables safebrowsing and restores state', async () => {
        await disableSafeBrowsing();

        await triggerUndo();

        expect(mocks.safebrowsingEnable).toHaveBeenCalledTimes(1);
        expect(settingsState.settingsList.safebrowsing.enabled).toBe(true);
    });

    it('returns false and shows an error toast on failure', async () => {
        mocks.safebrowsingDisable.mockRejectedValueOnce(new Error('Network error'));

        const result = await disableSafeBrowsing();

        expect(result).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalled();
        expect(settingsState.settingsList.safebrowsing.enabled).toBe(true);
    });
});

describe('disableParental', () => {
    it('disables parental control, updates state, and offers undo', async () => {
        const result = await disableParental();

        expect(mocks.parentalDisable).toHaveBeenCalledTimes(1);
        expect(settingsState.settingsList.parental.enabled).toBe(false);
        expect(mocks.createUndoToast).toHaveBeenCalledWith(
            'user_rules_parental_control_disabled',
            'notify_undo',
            expect.any(Function),
        );
        expect(mocks.addSuccessToast).toHaveBeenCalled();
        expect(result).toBe(true);
    });

    it('undo re-enables parental control and restores state', async () => {
        await disableParental();

        await triggerUndo();

        expect(mocks.parentalEnable).toHaveBeenCalledTimes(1);
        expect(settingsState.settingsList.parental.enabled).toBe(true);
    });

    it('returns false and shows an error toast on failure', async () => {
        mocks.parentalDisable.mockRejectedValueOnce(new Error('Network error'));

        const result = await disableParental();

        expect(result).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalled();
        expect(settingsState.settingsList.parental.enabled).toBe(true);
    });
});

describe('disableSafeSearch', () => {
    it('disables via settings API preserving provider flags', async () => {
        const result = await disableSafeSearch();

        expect(mocks.safesearchSettings).toHaveBeenCalledWith({
            ...seedSafeSearchConfig,
            enabled: false,
        });
        expect(settingsState.settingsList.safesearch).toEqual({
            ...seedSafeSearchConfig,
            enabled: false,
        });
        expect(mocks.createUndoToast).toHaveBeenCalledWith(
            'user_rules_safe_search_disabled',
            'notify_undo',
            expect.any(Function),
        );
        expect(mocks.addSuccessToast).toHaveBeenCalled();
        expect(result).toBe(true);
    });

    it('undo restores the full config including provider flags', async () => {
        await disableSafeSearch();

        await triggerUndo();

        expect(mocks.safesearchSettings).toHaveBeenLastCalledWith(seedSafeSearchConfig);
        expect(settingsState.settingsList.safesearch).toEqual(seedSafeSearchConfig);
    });

    it('returns false and shows an error toast on failure', async () => {
        mocks.safesearchSettings.mockRejectedValueOnce(new Error('Network error'));

        const result = await disableSafeSearch();

        expect(result).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalled();
        expect(settingsState.settingsList.safesearch).toEqual(seedSafeSearchConfig);
    });
});
