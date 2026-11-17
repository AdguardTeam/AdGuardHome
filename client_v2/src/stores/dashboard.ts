import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { STANDARD_DNS_PORT, STANDARD_WEB_PORT } from 'panel/helpers/constants';
import { areEqualVersions } from 'panel/helpers/version';
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
    handleRequestError: () => void,
    attempts = 60,
): StopStatusCheck => {
    let timeout: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const check = async (remaining: number): Promise<void> => {
        if (stopped) {
            return;
        }
        if (remaining === 0) {
            handleRequestError();
            return;
        }
        try {
            const data = await fetchStatus();
            // A stopped chain must not touch the state of a newer one.
            if (stopped) {
                return;
            }
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
    addErrorToast({ error: intl.getMessage('dns_status_error_reload') });
    setState('processing', false);
};

/**
 * Handles a bootstrap status check that failed all of its attempts.  The store
 * still holds `initialState` defaults, so the route body must not render:
 * `App` shows an error screen with a retry instead.
 */
const handleBootstrapStatusError = () => {
    handleStatusRequestError();
    setState('statusFailed', true);
};

export const getDnsStatus = async () => {
    stopDnsStatus?.();
    setState('processing', true);
    // A core that keeps answering `running: false` is a known state, not a
    // failed check: `App` shows the not-running screen instead of the error.
    let sawAnswer = false;

    const handleRequestSuccess = (response: { status: number; data: ServerStatus }) => {
        sawAnswer = true;
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
            // The store wrapper, not the raw API call: it saves name and theme.
            getProfileData();
        } else {
            // The status is learned even though the core is down, so the retry
            // loop must not keep the page blank while it polls.
            setState({ isCoreRunning: runningStatus, processing: false });
        }
    };

    const handleBootstrapError = () => {
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
        // A background check cannot rebuild the page after a failed bootstrap:
        // escalate to the full check instead of leaving the retry screen up.
        if (untrack(() => state.statusFailed)) {
            getDnsStatus();
            return;
        }
        const dnsStatus = response.data;
        if (dnsStatus.protection_disabled_duration === 0) {
            dnsStatus.protection_disabled_duration = null;
        }
        const runningStatus = dnsStatus && dnsStatus.running;
        if (runningStatus === true) {
            setState({
                protectionEnabled: dnsStatus.protection_enabled,
                protectionDisabledDuration: dnsStatus.protection_disabled_duration,
            });
        } else {
            setState('isCoreRunning', runningStatus);
        }
    };

    stopTimerStatus = checkStatus(handleRequestSuccess, handleStatusRequestError);
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

export const getProfileData = async () => {
    setState('processingProfile', true);
    try {
        const profile: ProfileInfo = await getProfile();
        setState({
            name: profile.name,
            // A userless install (auth disabled or GL-iNet) has no operator
            // theme: the profile only echoes the server config default, which
            // would overwrite the guest's choice stored in `localStorage` on
            // every load.  Keep the store theme unset for guests so that `App`
            // falls back to `getTheme()`.
            theme: profile.name ? profile.theme : undefined,
            processingProfile: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingProfile', false);
    }
};

export const changeLanguage = async (lang: ProfileInfo['language']) => {
    try {
        const profile = await getProfile();
        profile.language = lang;
        await updateProfile(profile);
        setState('language', lang);
    } catch (error) {
        addErrorToast({ error });
    }
};

export const changeTheme = async (theme: ProfileInfo['theme']) => {
    try {
        const profile = await getProfile();
        profile.theme = theme;
        await updateProfile(profile);
        setState('theme', theme);
    } catch (error) {
        addErrorToast({ error });
    }
};

export const dashboardState = untrack(() => state);
