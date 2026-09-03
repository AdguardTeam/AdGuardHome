import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import {
    safebrowsingStatus,
    safebrowsingEnable,
    safebrowsingDisable,
    parentalStatus,
    parentalEnable,
    parentalDisable,
    safesearchStatus,
    safesearchSettings,
    testUpstreamDNS,
} from 'panel/api/generated';
import { addErrorToast, addSuccessToast, createUndoToast } from './toasts';
import { splitByNewLine } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';
import type { SafeSearchConfig } from 'panel/api/model/safeSearchConfig';
import type { UpstreamsConfig } from 'panel/api/model/upstreamsConfig';

type SettingsState = {
    processing: boolean;
    processingTestUpstream: boolean;
    processingDhcpStatus: boolean;
    settingsList: {
        parental: { enabled: boolean };
        safebrowsing: { enabled: boolean };
        safesearch: SafeSearchConfig;
    };
};

const initialState: SettingsState = {
    processing: true,
    processingTestUpstream: false,
    processingDhcpStatus: false,
    settingsList: {
        parental: { enabled: false },
        safebrowsing: { enabled: false },
        safesearch: {},
    },
};

const [state, setState] = createStore<SettingsState>(initialState);

export const initSettings = async () => {
    setState('processing', true);
    try {
        const [safebrowsingStatusData, parentalStatusData, safesearchStatusData] =
            await Promise.all([safebrowsingStatus(), parentalStatus(), safesearchStatus()]);
        setState({
            settingsList: {
                safebrowsing: { enabled: safebrowsingStatusData.enabled },
                parental: { enabled: parentalStatusData.enabled },
                safesearch: { ...safesearchStatusData },
            },
            processing: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export async function toggleSetting(
    settingKey: 'safesearch',
    status: SafeSearchConfig,
): Promise<boolean>;
export async function toggleSetting(
    settingKey: 'safebrowsing' | 'parental',
    status: boolean,
): Promise<boolean>;
export async function toggleSetting(
    settingKey: string,
    status: boolean | SafeSearchConfig,
): Promise<boolean> {
    try {
        switch (settingKey) {
            case 'safebrowsing':
                if (status) {
                    await safebrowsingDisable();
                } else {
                    await safebrowsingEnable();
                }
                setState('settingsList', 'safebrowsing', 'enabled', !status);
                return true;
            case 'parental':
                if (status) {
                    await parentalDisable();
                } else {
                    await parentalEnable();
                }
                setState('settingsList', 'parental', 'enabled', !status);
                return true;
            case 'safesearch':
                await safesearchSettings(status as SafeSearchConfig);
                setState('settingsList', 'safesearch', status as SafeSearchConfig);
                return true;
            default:
                return false;
        }
    } catch (error) {
        addErrorToast({ error });
        return false;
    }
}

export const settingsState = untrack(() => state);

const disableWithUndo = async (
    disable: () => Promise<void>,
    enable: () => Promise<void>,
    setEnabled: (enabled: boolean) => void,
    message: string,
): Promise<boolean> => {
    try {
        await disable();
        setEnabled(false);
        addSuccessToast(
            createUndoToast(message, intl.getMessage('notify_undo'), async () => {
                await enable();
                setEnabled(true);
            }),
        );
        return true;
    } catch (error) {
        addErrorToast({ error });
        return false;
    }
};

export const disableSafeBrowsing = () =>
    disableWithUndo(
        safebrowsingDisable,
        safebrowsingEnable,
        (enabled) => setState('settingsList', 'safebrowsing', 'enabled', enabled),
        intl.getMessage('user_rules_browsing_security_disabled'),
    );

export const disableParental = () =>
    disableWithUndo(
        parentalDisable,
        parentalEnable,
        (enabled) => setState('settingsList', 'parental', 'enabled', enabled),
        intl.getMessage('user_rules_parental_control_disabled'),
    );

export const disableSafeSearch = async (): Promise<boolean> => {
    // Snapshot as a plain object: store reads return live proxies, so keeping
    // a reference would reflect post-disable state instead of the original.
    const previousConfig = { ...state.settingsList.safesearch };
    try {
        await safesearchSettings({ ...previousConfig, enabled: false });
        setState('settingsList', 'safesearch', { ...previousConfig, enabled: false });
        addSuccessToast(
            createUndoToast(
                intl.getMessage('user_rules_safe_search_disabled'),
                intl.getMessage('notify_undo'),
                async () => {
                    await safesearchSettings(previousConfig);
                    setState('settingsList', 'safesearch', previousConfig);
                },
            ),
        );
        return true;
    } catch (error) {
        addErrorToast({ error });
        return false;
    }
};

export const testUpstreamWithFormValues = async (
    formValues: {
        bootstrap_dns: string;
        upstream_dns: string;
        local_ptr_upstreams: string;
        fallback_dns: string;
    },
    upstreamDnsFile?: string,
) => {
    setState('processingTestUpstream', true);
    try {
        const { bootstrap_dns, upstream_dns, local_ptr_upstreams, fallback_dns } = formValues;

        const filterOutComments = (lines: string[]) =>
            lines.filter((line) => !line.startsWith('#') && !line.startsWith('!'));
        const removeComments = (text: string) => filterOutComments(splitByNewLine(text));

        const config: UpstreamsConfig = {
            bootstrap_dns: splitByNewLine(bootstrap_dns),
            private_upstream: splitByNewLine(local_ptr_upstreams),
            fallback_dns: splitByNewLine(fallback_dns),
            ...(upstreamDnsFile ? null : { upstream_dns: removeComments(upstream_dns) }),
        };

        const upstreamResponse = await testUpstreamDNS(config);
        const testMessages = Object.keys(upstreamResponse).map((key) => {
            const message = upstreamResponse[key];
            if (message.startsWith('WARNING:')) {
                addErrorToast({
                    error: intl.getMessage('dns_test_warning_toast', { key }),
                });
            } else if (message.endsWith(': parsing error')) {
                const info = message.substring(0, message.indexOf(':'));
                const [sectionKey, line] = info.split(' ');
                addErrorToast({
                    error: intl.getMessage('dns_test_parsing_error_toast', {
                        section: sectionKey,
                        number: line,
                    }),
                });
            } else if (message !== 'OK') {
                addErrorToast({ error: intl.getMessage('dns_test_not_ok_toast', { key }) });
            }
            return message;
        });

        if (testMessages.every((message) => message === 'OK' || message.startsWith('WARNING:'))) {
            addSuccessToast(intl.getMessage('dns_test_ok_toast'));
        }
    } catch (error) {
        addErrorToast({ error });
    } finally {
        setState('processingTestUpstream', false);
    }
};
