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
import intl, { LocalesType } from 'panel/common/intl';
import { addErrorToast, addSuccessToast, addNoticeToast } from './toasts';
import { getTlsStatus } from './encryption';
import { getUpdateFailedMessage } from './dashboard/noticeOptions';
import type { Client, AutoClient } from 'panel/initialState';

type DashboardState = {
    processing: boolean;
    isCoreRunning: boolean;
    processingVersion: boolean;
    processingClients: boolean;
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
    announcementUrl: string;
    newVersion: string;
    canAutoUpdate: boolean;
    language: string;
    isUpdateAvailable: boolean;
};

const initialState: DashboardState = {
    processing: true,
    isCoreRunning: true,
    processingVersion: true,
    processingClients: true,
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
    announcementUrl: '',
    newVersion: '',
    canAutoUpdate: false,
    language: '',
    isUpdateAvailable: false,
};

const [state, setState] = createStore<DashboardState>(initialState);

const CHECK_TIMEOUT = 1000;

let statusTimeout: ReturnType<typeof setTimeout> | null = null;
const rmTimeout = (t: ReturnType<typeof setTimeout> | null): null => {
    if (t) clearTimeout(t);
    return null;
};

const checkStatus = async (
    handleRequestSuccess: (response: { status: number; data: ServerStatus }) => void,
    handleRequestError: () => void,
    attempts = 60,
): Promise<void> => {
    if (attempts === 0) {
        handleRequestError();
        return;
    }
    try {
        const data = await customFetch<ServerStatus>(getStatusUrl(), {
            method: 'GET',
            skipAuthRedirect: true,
        });
        statusTimeout = rmTimeout(statusTimeout);
        handleRequestSuccess({ status: 200, data });
        if (data.running === false) {
            statusTimeout = setTimeout(
                checkStatus,
                CHECK_TIMEOUT,
                handleRequestSuccess,
                handleRequestError,
                attempts - 1,
            );
        }
    } catch {
        statusTimeout = rmTimeout(statusTimeout);
        statusTimeout = setTimeout(
            checkStatus,
            CHECK_TIMEOUT,
            handleRequestSuccess,
            handleRequestError,
            attempts - 1,
        );
    }
};

export const getDnsStatus = async () => {
    setState('processing', true);

    const handleRequestError = () => {
        addErrorToast({ error: 'dns_status_error' });
        setState('processing', false);
        window.location.reload();
    };

    const handleRequestSuccess = (response: any) => {
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
            getProfile();
        } else {
            setState('isCoreRunning', runningStatus);
        }
    };

    try {
        checkStatus(handleRequestSuccess, handleRequestError);
    } catch {
        handleRequestError();
    }
};

export const getTimerStatus = async () => {
    const handleRequestError = () => {
        addErrorToast({ error: 'dns_status_error' });
        setState('processing', false);
        window.location.reload();
    };

    const handleRequestSuccess = (response: any) => {
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

    try {
        checkStatus(handleRequestSuccess, handleRequestError);
    } catch {
        handleRequestError();
    }
};

export const getVersion = async (recheck = false) => {
    setState('processingVersion', true);
    try {
        const data: VersionInfo = await getVersionJson({ recheck_now: recheck });
        const currentVersion =
            untrack(() => state.dnsVersion) === 'undefined' ? 0 : untrack(() => state.dnsVersion);
        if (data && !data.disabled && !areEqualVersions(currentVersion, data.new_version)) {
            setState({
                announcementUrl: data.announcement_url,
                newVersion: data.new_version,
                canAutoUpdate: data.can_autoupdate,
                isUpdateAvailable: true,
                processingVersion: false,
                checkUpdateFlag: !data.disabled,
            });
        } else {
            setState({ processingVersion: false, checkUpdateFlag: data ? !data.disabled : false });
        }
        if (recheck) {
            if (data && !areEqualVersions(currentVersion, data.new_version)) {
                addSuccessToast(intl.getMessage('updates_checked'));
            } else {
                addSuccessToast(intl.getMessage('updates_version_equal'));
            }
        }
    } catch {
        addErrorToast({ error: 'version_request_error' });
        setState('processingVersion', false);
    }
};

export const getUpdate = async () => {
    setState('processingUpdate', true);
    const handleRequestError = () => {
        addNoticeToast(getUpdateFailedMessage());
        setState('processingUpdate', false);
    };
    const handleRequestSuccess = (response: any) => {
        const responseVersion = response.data?.version;
        if (untrack(() => state.dnsVersion) !== responseVersion) {
            setState('processingUpdate', false);
            window.location.reload();
        }
    };
    try {
        await beginUpdate();
        checkStatus(handleRequestSuccess, handleRequestError);
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

const sortClients = (clients: any[]) => {
    if (!Array.isArray(clients)) return [];
    return [...clients].sort((a, b) => {
        const nameA = (a.name || a.ip || '').toString().toLowerCase();
        const nameB = (b.name || b.ip || '').toString().toLowerCase();
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
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingClients', false);
    }
};

export const getProfileData = async () => {
    setState('processingProfile', true);
    try {
        const profile: ProfileInfo = await getProfile();
        setState({
            name: profile.name,
            theme: profile.theme,
            processingProfile: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingProfile', false);
    }
};

export const changeLanguage = async (lang: LocalesType) => {
    try {
        const profile = await getProfile();
        profile.language = lang as ProfileInfo['language'];
        await updateProfile(profile);
        setState('language', lang);
    } catch (error) {
        addErrorToast({ error });
    }
};

export const changeTheme = async (theme: string) => {
    try {
        const profile = await getProfile();
        profile.theme = theme as ProfileInfo['theme'];
        await updateProfile(profile);
        setState('theme', theme);
    } catch (error) {
        addErrorToast({ error });
    }
};

export const dashboardState = untrack(() => state);
