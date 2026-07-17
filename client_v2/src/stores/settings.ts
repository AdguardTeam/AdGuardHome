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
import { addErrorToast, addSuccessToast } from './toasts';
import { splitByNewLine } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';

type SettingsState = {
    processing: boolean;
    processingTestUpstream: boolean;
    processingDhcpStatus: boolean;
    settingsList: {
        parental: { enabled: boolean };
        safebrowsing: { enabled: boolean };
        safesearch: Record<string, boolean>;
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
                parental: { enabled: parentalStatusData.enable },
                safesearch: { ...safesearchStatusData },
            },
            processing: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const toggleSetting = async (settingKey: string, status: any) => {
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
                await safesearchSettings(status);
                setState('settingsList', 'safesearch', status);
                return true;
            default:
                return false;
        }
    } catch (error) {
        addErrorToast({ error });
        return false;
    }
};

export const settingsState = untrack(() => state);

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

        const config: any = {
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
