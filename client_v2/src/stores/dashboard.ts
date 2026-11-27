import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { STANDARD_DNS_PORT, STANDARD_WEB_PORT, THEMES } from 'panel/helpers/constants';
import { areEqualVersions } from 'panel/helpers/version';
import { getBrowserLanguage, isLang } from 'panel/helpers/helpers';
import {
    getStatusUrl,
    getVersionJson,
    beginUpdate,
    setProtection,
    clientsStatus,
    getProfile,
    updateProfile,
} from 'panel/api/generated';
import { customFetch } from 'panel/api/customFetch';
import type { ServerStatus } from 'panel/api/model/serverStatus';
import type { VersionInfo } from 'panel/api/model/versionInfo';
import type { Clients } from 'panel/api/model/clients';
import type { ProfileInfo } from 'panel/api/model/profileInfo';
import type { Lang } from 'panel/api/model/lang';
import intl from 'panel/common/intl';
import { addErrorToast, addSuccessToast, addNoticeToast } from './toasts';
import { getTlsStatus } from './encryption';
import { getUpdateFailedMessage } from './dashboard/noticeOptions';
import type { Client, AutoClient } from 'panel/initialState';
import type { Client as ClientModel } from 'panel/api/model/client';
import type { ClientAuto } from 'panel/api/model/clientAuto';

type DashboardState = {
    processing: boolean;
    /** The bootstrap status check failed, so the dashboard is unknown. */
    statusFailed: boolean;
    isCoreRunning: boolean;
    processingVersion: boolean;
    processingClients: boolean;
    clientsInitialized: boolean;
    processingUpdate: boolean;
    processingProfile: boolean;
    /** The last profile request failed, so the operator is unknown. */
    profileFailed: boolean;
    protectionEnabled: boolean;
    protectionDisabledDuration: number | null;
    protectionCountdownActive: boolean;
    processingProtection: boolean;
    httpPort: number;
    dnsPort: number;
    dnsAddresses: string[];
    dnsVersion: string;
    clients: Client[];
    autoClients: AutoClient[];
    supportedTags: string[];
    name: string;
    theme: string | undefined;
    checkUpdateFlag: boolean;
    versionCheckFailed: boolean;
    announcementUrl: string;
    newVersion: string;
    canAutoUpdate: boolean;
    language: string;
    isUpdateAvailable: boolean;
};

const initialState: DashboardState = {
    processing: true,
    statusFailed: false,
    isCoreRunning: true,
    processingVersion: true,
    processingClients: true,
    clientsInitialized: false,
    processingUpdate: false,
    processingProfile: true,
    profileFailed: false,
    protectionEnabled: false,
    protectionDisabledDuration: null,
    protectionCountdownActive: false,
    processingProtection: false,
    httpPort: STANDARD_WEB_PORT,
    dnsPort: STANDARD_DNS_PORT,
    dnsAddresses: [],
    dnsVersion: '',
    clients: [],
    autoClients: [],
    supportedTags: [],
    name: '',
    theme: undefined,
    checkUpdateFlag: false,
    versionCheckFailed: false,
    announcementUrl: '',
    newVersion: '',
    canAutoUpdate: false,
    language: '',
    isUpdateAvailable: false,
};

const [state, setState] = createStore<DashboardState>(initialState);

const CHECK_TIMEOUT = 1000;

/** Fetches the server status without redirecting to the login page on 401. */
const fetchStatus = (): Promise<ServerStatus> =>
    customFetch<ServerStatus>(getStatusUrl(), {
        method: 'GET',
        skipAuthRedirect: true,
    });

type StopStatusCheck = () => void;

let stopDnsStatus: StopStatusCheck | null = null;
let stopTimerStatus: StopStatusCheck | null = null;
let stopUpdateStatus: StopStatusCheck | null = null;

// Each call owns its retry timer and can be stopped.  A shared timer would let
// a concurrent chain, e.g. the one started by `getTimerStatus`, cancel another
// chain's pending retry and leave `processing` stuck at true forever.
const checkStatus = (
    handleRequestSuccess: (response: { status: number; data: ServerStatus }) => void,
    handleRequestError: (sawAnswer: boolean) => void,
    attempts = 60,
): StopStatusCheck => {
    let timeout: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    // A chain that only ever got `running: false` answers has learned the core
    // state and must not report a failed check when it runs out of attempts.
    let sawAnswer = false;

    const check = async (remaining: number): Promise<void> => {
        if (stopped) {
            return;
        }
        if (remaining === 0) {
            handleRequestError(sawAnswer);
            return;
        }
        try {
            const data = await fetchStatus();
            // A stopped chain must not touch the state of a newer one.
            if (stopped) {
                return;
            }
            sawAnswer = true;
            handleRequestSuccess({ status: 200, data });
            if (data.running === false) {
                timeout = setTimeout(check, CHECK_TIMEOUT, remaining - 1);
            }
        } catch {
            if (stopped) {
                return;
            }
            timeout = setTimeout(check, CHECK_TIMEOUT, remaining - 1);
        }
    };

    void check(attempts);

    return () => {
        stopped = true;
        if (timeout) {
            clearTimeout(timeout);
            timeout = null;
        }
    };
};

/** Stops every pending status poll, e.g. when the app unmounts. */
export const stopStatusChecks = () => {
    stopDnsStatus?.();
    stopTimerStatus?.();
    stopUpdateStatus?.();
    stopDnsStatus = null;
    stopTimerStatus = null;
    stopUpdateStatus = null;
};

/**
 * Handles a status check that failed all of its attempts: shows the notice and
 * unblocks the UI.  It must not reload the page: the reload would wipe the
 * notice before it is shown and, while the backend keeps failing, restart the
 * attempts and reload forever.
 */
const handleStatusRequestError = () => {
    addErrorToast({ error: intl.getMessage('dns_status_error') });
    setState('processing', false);
};

/**
 * Handles a bootstrap status check that failed all of its attempts.  The store
 * still holds `initialState` defaults, so the route body must not render:
 * `App` shows an error screen with a retry instead.  The screen carries the
 * message, so a toast would only repeat it.
 */
const handleBootstrapStatusError = () => {
    setState({ processing: false, statusFailed: true });
};

export const getDnsStatus = async () => {
    stopDnsStatus?.();
    setState('processing', true);

    const handleRequestSuccess = (response: { status: number; data: ServerStatus }) => {
        setState('statusFailed', false);
        const dnsStatus = response.data;
        if (dnsStatus.protection_disabled_duration === 0) {
            dnsStatus.protection_disabled_duration = null;
        }
        const runningStatus = dnsStatus && dnsStatus.running;
        if (runningStatus === true) {
            setState({
                isCoreRunning: true,
                processing: false,
                dnsVersion: dnsStatus.version,
                dnsPort: dnsStatus.dns_port,
                dnsAddresses: dnsStatus.dns_addresses || [],
                protectionEnabled: dnsStatus.protection_enabled,
                protectionDisabledDuration: dnsStatus.protection_disabled_duration,
                language: dnsStatus.language,
                httpPort: dnsStatus.http_port,
            });
            getVersion();
            getTlsStatus();
        } else {
            // The status is learned even though the core is down, so the retry
            // loop must not keep the page blank while it polls.  The port comes
            // from the config, so the start-error screen can name it; `0` means
            // the config holds no port, and the standard one would be used.
            setState({
                isCoreRunning: runningStatus,
                processing: false,
                dnsPort: dnsStatus.dns_port || STANDARD_DNS_PORT,
            });
        }
    };

    const handleBootstrapError = (sawAnswer: boolean) => {
        if (sawAnswer) {
            // The endpoint answers, it only reports a stopped core: the
            // not-running screen is already up, and the toast would claim a
            // failure that did not happen.
            return;
        }
        handleBootstrapStatusError();
    };

    stopDnsStatus = checkStatus(handleRequestSuccess, handleBootstrapError);
};

/**
 * Refreshes the DNS addresses and ports without touching `processing`, the
 * version, TLS status, or the profile, and keeps the previous values on
 * failure.  Used after settings that change the addresses, e.g. encryption.
 */
export const refreshDnsStatus = async (): Promise<void> => {
    try {
        const dnsStatus = await fetchStatus();
        if (!dnsStatus || dnsStatus.running !== true) {
            return;
        }
        setState({
            dnsPort: dnsStatus.dns_port,
            dnsAddresses: dnsStatus.dns_addresses || [],
            httpPort: dnsStatus.http_port,
        });
    } catch {
        // A background refresh must not interrupt the UI: keep the old values.
    }
};

export const getTimerStatus = async () => {
    stopTimerStatus?.();

    const handleRequestSuccess = (response: { status: number; data: ServerStatus }) => {
        const dnsStatus = response.data;
        const runningStatus = dnsStatus && dnsStatus.running;

        // A background check cannot rebuild the page: after a failed bootstrap
        // or while the not-running screen is up, only the full check restores
        // the status values the route body renders.
        const needsFullCheck = untrack(
            () => state.statusFailed || (runningStatus === true && !state.isCoreRunning),
        );
        if (needsFullCheck) {
            getDnsStatus();
            return;
        }

        if (dnsStatus.protection_disabled_duration === 0) {
            dnsStatus.protection_disabled_duration = null;
        }
        if (runningStatus === true) {
            setState({
                protectionEnabled: dnsStatus.protection_enabled,
                protectionDisabledDuration: dnsStatus.protection_disabled_duration,
            });
        } else {
            setState('isCoreRunning', runningStatus);
        }
    };

    const handleTimerError = (sawAnswer: boolean) => {
        if (sawAnswer) {
            // Same as the bootstrap check: a chain that received answers has
            // already learned the core state and must not claim a failure.
            return;
        }
        // The full check may have raised a screen while this chain was still
        // polling, and the screen carries the message already.
        const screenIsUp = untrack(() => state.statusFailed || !state.isCoreRunning);
        if (screenIsUp) {
            return;
        }
        handleStatusRequestError();
    };

    stopTimerStatus = checkStatus(handleRequestSuccess, handleTimerError);
};

export const getVersion = async (recheck = false) => {
    setState('processingVersion', true);
    try {
        const data: VersionInfo = await getVersionJson({ recheck_now: recheck });
        const currentVersion =
            untrack(() => state.dnsVersion) === 'undefined' ? '' : untrack(() => state.dnsVersion);
        if (data && !data.disabled && !areEqualVersions(currentVersion, data.new_version)) {
            setState({
                announcementUrl: data.announcement_url,
                newVersion: data.new_version,
                canAutoUpdate: data.can_autoupdate,
                isUpdateAvailable: true,
                processingVersion: false,
                checkUpdateFlag: !data.disabled,
                versionCheckFailed: false,
            });
        } else {
            setState({
                processingVersion: false,
                checkUpdateFlag: data ? !data.disabled : false,
                versionCheckFailed: false,
            });
        }
        if (recheck && data && !data.disabled) {
            if (!areEqualVersions(currentVersion, data.new_version)) {
                addSuccessToast(intl.getMessage('updates_checked'));
            } else {
                addSuccessToast(intl.getMessage('updates_version_equal'));
            }
        }
    } catch {
        addErrorToast({ error: intl.getMessage('version_request_error') });
        // A failed check does not tell us whether updates are disabled, so keep
        // the retry control available and leave `checkUpdateFlag` unchanged.
        setState({ processingVersion: false, versionCheckFailed: true });
    }
};

export const getUpdate = async () => {
    setState('processingUpdate', true);
    const handleRequestError = () => {
        addNoticeToast(getUpdateFailedMessage());
        setState('processingUpdate', false);
    };
    const handleRequestSuccess = (response: { status: number; data: ServerStatus }) => {
        const responseVersion = response.data?.version;
        if (untrack(() => state.dnsVersion) !== responseVersion) {
            setState('processingUpdate', false);
            window.location.reload();
        }
    };
    try {
        await beginUpdate();
        stopUpdateStatus?.();
        stopUpdateStatus = checkStatus(handleRequestSuccess, handleRequestError);
    } catch {
        handleRequestError();
    }
};

export const toggleProtection = async (time: number | null = null) => {
    setState('processingProtection', true);
    try {
        await setProtection({
            enabled: !untrack(() => state.protectionEnabled),
            duration: time ?? undefined,
        });
        setState({
            protectionEnabled: !untrack(() => state.protectionEnabled),
            processingProtection: false,
            protectionDisabledDuration: time,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingProtection', false);
    }
};

export const setDisableDurationTime = (timeToEnableProtection: number) => {
    setState('protectionDisabledDuration', timeToEnableProtection);
};

const sortClients = (clients: (ClientModel | ClientAuto)[]) => {
    if (!Array.isArray(clients)) return [];
    return [...clients].sort((a, b) => {
        const nameA = (a.name || ('ip' in a ? a.ip : '') || '').toString().toLowerCase();
        const nameB = (b.name || ('ip' in b ? b.ip : '') || '').toString().toLowerCase();
        return nameA.localeCompare(nameB);
    });
};

export const getClients = async () => {
    setState('processingClients', true);
    try {
        const data: Clients = await clientsStatus();
        setState({
            clients: sortClients(data.clients || []),
            autoClients: sortClients(data.auto_clients || []),
            supportedTags: data.supported_tags || [],
            processingClients: false,
            clientsInitialized: true,
        });
    } catch (error) {
        addErrorToast({ error });
        setState({ processingClients: false, clientsInitialized: true });
    }
};

/**
 * An installation that never selected a language returns an empty one in the
 * profile, but the API accepts only supported codes.  Fall back to the
 * language the UI is showing.
 */
const getSupportedLanguage = (lang: string): Lang => (isLang(lang) ? lang : getBrowserLanguage());

/**
 * The update API requires one of {@link THEMES}, but a server may return an
 * empty or null theme, which is sent back as is if spread into the payload.
 */
const getSupportedTheme = (theme: string | null | undefined): ProfileInfo['theme'] =>
    theme === THEMES.auto || theme === THEMES.dark || theme === THEMES.light
        ? theme
        : THEMES.auto;

export const getProfileData = async () => {
    setState({ processingProfile: true, profileFailed: false });
    try {
        const profile: ProfileInfo = await getProfile();
        setState({
            name: profile.name,
            // A userless install (auth disabled or GL-iNet) has no operator
            // theme: the profile only echoes the server config default, which
            // would overwrite the guest's choice stored in `localStorage` on
            // every load.  Keep the store theme unset for guests so that `App`
            // falls back to `getTheme()`.
            theme: profile.name ? getSupportedTheme(profile.theme) : undefined,
            processingProfile: false,
        });
    } catch (error) {
        addErrorToast({ error });
        // Without the profile the operator is unknown, not signed out.
        setState({ processingProfile: false, profileFailed: true });
    }
};

/**
 * Sends the profile with only the fields the update API requires and accepts.
 */
const saveProfile = async (profile: ProfileInfo): Promise<void> => {
    await updateProfile({
        name: profile.name,
        language: getSupportedLanguage(profile.language),
        theme: getSupportedTheme(profile.theme),
    });
};

export const changeLanguage = async (lang: ProfileInfo['language']) => {
    try {
        const profile = await getProfile();
        await saveProfile({ ...profile, language: lang });
        setState('language', lang);
    } catch (error) {
        addErrorToast({ error });
    }
};

export const changeTheme = async (theme: ProfileInfo['theme']) => {
    try {
        const profile = await getProfile();
        await saveProfile({ ...profile, theme });
        setState('theme', theme);
    } catch (error) {
        addErrorToast({ error });
    }
};

export const dashboardState = untrack(() => state);
