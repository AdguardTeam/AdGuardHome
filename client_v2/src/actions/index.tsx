import { createAction } from 'redux-actions';

import React from 'react';
import { compose } from 'redux';
import type { Dispatch } from 'redux';
import intl, { type LocalesType } from 'panel/common/intl';
import type { AppDispatch, AppGetState } from 'panel/store/types';
import { splitByNewLine, sortClients, filterOutComments } from '../helpers/helpers';
import {
    BLOCK_ACTIONS,
    CHECK_TIMEOUT,
    STATUS_RESPONSE,
    SETTINGS_NAMES,
    MANUAL_UPDATE_LINK,
    THEMES,
} from '../helpers/constants';
import { areEqualVersions } from '../helpers/version';
import { getTlsStatus } from './encryption';
import { apiClient } from '../api/Api';
import { addErrorToast, addNoticeToast, addSuccessToast, createUndoToast } from './toasts';
import { getFilteringStatus, setRules } from './filtering';

const getSectionMessage = (sectionKey: string) => {
    switch (sectionKey) {
        case 'upstream_dns':
            return intl.getMessage('upstream_dns');
        case 'bootstrap_dns':
            return intl.getMessage('bootstrap_dns');
        case 'fallback_dns':
            return intl.getMessage('fallback_dns');
        default:
            return sectionKey;
    }
};

type SafeSearchConfig = Record<string, boolean> & { enabled: boolean };
type ToggleSettingKey = keyof typeof SETTINGS_NAMES;
type Theme = (typeof THEMES)[keyof typeof THEMES];
type BlockAction = (typeof BLOCK_ACTIONS)[keyof typeof BLOCK_ACTIONS];

export const toggleSettingStatus = createAction<{
    settingKey: ToggleSettingKey;
    value?: boolean | SafeSearchConfig;
}>('SETTING_STATUS_TOGGLE');
export const showSettingsFailure = createAction('SETTINGS_FAILURE_SHOW');

/**
 *
 * @param {*} settingKey = SETTINGS_NAMES
 * @param {*} status: boolean | SafeSearchConfig
 * @returns
 */
export const toggleSetting =
    (settingKey: ToggleSettingKey, status: boolean | SafeSearchConfig) =>
    async (dispatch: Dispatch) => {
        try {
            switch (settingKey) {
                case SETTINGS_NAMES.safebrowsing:
                    if (status) {
                        await apiClient.disableSafebrowsing();
                    } else {
                        await apiClient.enableSafebrowsing();
                    }
                    dispatch(toggleSettingStatus({ settingKey }));
                    return true;
                case SETTINGS_NAMES.parental:
                    if (status) {
                        await apiClient.disableParentalControl();
                    } else {
                        await apiClient.enableParentalControl();
                    }
                    dispatch(toggleSettingStatus({ settingKey }));
                    return true;
                case SETTINGS_NAMES.safesearch:
                    await apiClient.updateSafesearch(status as SafeSearchConfig);
                    dispatch(toggleSettingStatus({ settingKey, value: status }));
                    return true;
                default:
                    return false;
            }
        } catch (error) {
            dispatch(addErrorToast({ error }));
            return false;
        }
    };

export const initSettingsRequest = createAction('SETTINGS_INIT_REQUEST');
export const initSettingsFailure = createAction('SETTINGS_INIT_FAILURE');
type SettingsSuccessList = {
    safebrowsing: { enabled: boolean };
    parental: { enabled: boolean };
    safesearch: SafeSearchConfig;
};
export const initSettingsSuccess = createAction<{ settingsList: SettingsSuccessList }>(
    'SETTINGS_INIT_SUCCESS',
);

export const initSettings = () => async (dispatch: Dispatch) => {
    dispatch(initSettingsRequest());
    try {
        const safebrowsingStatus = await apiClient.getSafebrowsingStatus();
        const parentalStatus = await apiClient.getParentalStatus();
        const safesearchStatus = (await apiClient.getSafesearchStatus()) as SafeSearchConfig;
        const newSettingsList: SettingsSuccessList = {
            safebrowsing: {
                enabled: safebrowsingStatus.enabled,
            },
            parental: {
                enabled: parentalStatus.enabled,
            },
            safesearch: {
                ...safesearchStatus,
            },
        };
        dispatch(initSettingsSuccess({ settingsList: newSettingsList }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(initSettingsFailure());
    }
};

export const toggleProtectionRequest = createAction('TOGGLE_PROTECTION_REQUEST');
export const toggleProtectionFailure = createAction('TOGGLE_PROTECTION_FAILURE');
export const toggleProtectionSuccess = createAction('TOGGLE_PROTECTION_SUCCESS');

export const toggleProtection =
    (status: any, time: number | null = null) =>
    async (dispatch: any) => {
        dispatch(toggleProtectionRequest());
        try {
            await apiClient.setProtection({ enabled: !status, duration: time });
            dispatch(toggleProtectionSuccess({ disabledDuration: time }));
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(toggleProtectionFailure());
        }
    };

export const setDisableDurationTime = createAction('SET_DISABLED_DURATION_TIME');

export const setProtectionTimerTime = (updatedTime: any) => async (dispatch: any) => {
    dispatch(setDisableDurationTime({ timeToEnableProtection: updatedTime }));
};

export const getVersionRequest = createAction('GET_VERSION_REQUEST');
export const getVersionFailure = createAction('GET_VERSION_FAILURE');
export const getVersionSuccess = createAction('GET_VERSION_SUCCESS');

export const getVersion =
    (recheck = false) =>
    async (dispatch: any, getState: any) => {
        dispatch(getVersionRequest());
        try {
            const data = await apiClient.getGlobalVersion({ recheck_now: recheck });
            dispatch(getVersionSuccess(data));

            if (recheck) {
                const { dnsVersion } = getState().dashboard;
                const currentVersion = dnsVersion === 'undefined' ? 0 : dnsVersion;

                if (data && !areEqualVersions(currentVersion, data.new_version)) {
                    dispatch(addSuccessToast(intl.getMessage('updates_checked')));
                } else {
                    dispatch(addSuccessToast(intl.getMessage('updates_version_equal')));
                }
            }
        } catch (_error) {
            dispatch(addErrorToast({ error: 'version_request_error' }));
            dispatch(getVersionFailure());
        }
    };

export const getUpdateRequest = createAction('GET_UPDATE_REQUEST');
export const getUpdateFailure = createAction('GET_UPDATE_FAILURE');
export const getUpdateSuccess = createAction('GET_UPDATE_SUCCESS');

const checkStatus = async (handleRequestSuccess: any, handleRequestError: any, attempts = 60) => {
    let timeout;

    if (attempts === 0) {
        handleRequestError();
    }

    const rmTimeout = (t: any) => t && clearTimeout(t);

    try {
        const response = await fetch(`${apiClient.baseUrl}/status`);
        rmTimeout(timeout);
        if (response.ok) {
            const data = await response.json();
            handleRequestSuccess({ status: response.status, data });
            if (data.running === false) {
                timeout = setTimeout(
                    checkStatus,
                    CHECK_TIMEOUT,
                    handleRequestSuccess,
                    handleRequestError,
                    attempts - 1,
                );
            }
        }
    } catch (_error) {
        rmTimeout(timeout);
        timeout = setTimeout(
            checkStatus,
            CHECK_TIMEOUT,
            handleRequestSuccess,
            handleRequestError,
            attempts - 1,
        );
    }
};

export const getUpdate = () => async (dispatch: any, getState: any) => {
    const { dnsVersion } = getState().dashboard;

    dispatch(getUpdateRequest());
    const handleRequestError = () => {
        const options = {
            components: {
                a: <a href={MANUAL_UPDATE_LINK} target="_blank" rel="noopener noreferrer" />,
            },
        };

        dispatch(addNoticeToast({ error: 'update_failed', options }));
        dispatch(getUpdateFailure());
    };

    const handleRequestSuccess = (response: any) => {
        const responseVersion = response.data?.version;

        if (dnsVersion !== responseVersion) {
            dispatch(getUpdateSuccess());

            window.location.reload();
        }
    };

    try {
        await apiClient.getUpdate();
        checkStatus(handleRequestSuccess, handleRequestError);
    } catch (_error) {
        handleRequestError();
    }
};

export const getClientsRequest = createAction('GET_CLIENTS_REQUEST');
export const getClientsFailure = createAction('GET_CLIENTS_FAILURE');
export const getClientsSuccess = createAction('GET_CLIENTS_SUCCESS');

export const getClients = () => async (dispatch: any) => {
    dispatch(getClientsRequest());
    try {
        const data = await apiClient.getClients();
        const sortedClients = data.clients && sortClients(data.clients);
        const sortedAutoClients = data.auto_clients && sortClients(data.auto_clients);

        dispatch(
            getClientsSuccess({
                clients: sortedClients || [],
                autoClients: sortedAutoClients || [],
                supportedTags: data.supported_tags || [],
            }),
        );
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getClientsFailure());
    }
};

export const getProfileRequest = createAction('GET_PROFILE_REQUEST');
export const getProfileFailure = createAction('GET_PROFILE_FAILURE');
export const getProfileSuccess = createAction('GET_PROFILE_SUCCESS');

export const getProfile = () => async (dispatch: any) => {
    dispatch(getProfileRequest());
    try {
        const profile = await apiClient.getProfile();
        dispatch(getProfileSuccess(profile));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getProfileFailure());
    }
};

export const dnsStatusRequest = createAction('DNS_STATUS_REQUEST');
export const dnsStatusFailure = createAction('DNS_STATUS_FAILURE');
export const dnsStatusSuccess = createAction('DNS_STATUS_SUCCESS');
export const setDnsRunningStatus = createAction('SET_DNS_RUNNING_STATUS');

export const getDnsStatus = () => async (dispatch: any) => {
    dispatch(dnsStatusRequest());

    const handleRequestError = () => {
        dispatch(addErrorToast({ error: 'dns_status_error' }));
        dispatch(dnsStatusFailure());

        window.location.reload();
    };

    const handleRequestSuccess = (response: any) => {
        const dnsStatus = response.data;
        if (dnsStatus.protection_disabled_duration === 0) {
            dnsStatus.protection_disabled_duration = null;
        }
        const { running } = dnsStatus;
        const runningStatus = dnsStatus && running;
        if (runningStatus === true) {
            dispatch(dnsStatusSuccess(dnsStatus));
            dispatch(getVersion());
            dispatch(getTlsStatus());
            dispatch(getProfile());
        } else {
            dispatch(setDnsRunningStatus(running));
        }
    };

    try {
        checkStatus(handleRequestSuccess, handleRequestError);
    } catch (_error) {
        handleRequestError();
    }
};

export const timerStatusRequest = createAction('TIMER_STATUS_REQUEST');
export const timerStatusFailure = createAction('TIMER_STATUS_FAILURE');
export const timerStatusSuccess = createAction('TIMER_STATUS_SUCCESS');

export const getTimerStatus = () => async (dispatch: any) => {
    dispatch(timerStatusRequest());

    const handleRequestError = () => {
        dispatch(addErrorToast({ error: 'dns_status_error' }));
        dispatch(dnsStatusFailure());

        window.location.reload();
    };

    const handleRequestSuccess = (response: any) => {
        const dnsStatus = response.data;
        if (dnsStatus.protection_disabled_duration === 0) {
            dnsStatus.protection_disabled_duration = null;
        }
        const { running } = dnsStatus;
        const runningStatus = dnsStatus && running;
        if (runningStatus === true) {
            dispatch(timerStatusSuccess(dnsStatus));
        } else {
            dispatch(setDnsRunningStatus(running));
        }
    };

    try {
        checkStatus(handleRequestSuccess, handleRequestError);
    } catch (_error) {
        handleRequestError();
    }
};

export const testUpstreamRequest = createAction('TEST_UPSTREAM_REQUEST');
export const testUpstreamFailure = createAction('TEST_UPSTREAM_FAILURE');
export const testUpstreamSuccess = createAction('TEST_UPSTREAM_SUCCESS');

export const testUpstream =
    (
        { bootstrap_dns, upstream_dns, local_ptr_upstreams, fallback_dns }: any,
        upstream_dns_file: any,
    ) =>
    async (dispatch: any) => {
        dispatch(testUpstreamRequest());
        try {
            const removeComments = compose(filterOutComments, splitByNewLine);

            const config = {
                bootstrap_dns: splitByNewLine(bootstrap_dns),
                private_upstream: splitByNewLine(local_ptr_upstreams),
                fallback_dns: splitByNewLine(fallback_dns),
                ...(upstream_dns_file
                    ? null
                    : {
                          upstream_dns: removeComments(upstream_dns),
                      }),
            };

            const upstreamResponse = await apiClient.testUpstream(config);
            const testMessages = Object.keys(upstreamResponse).map((key) => {
                const message = upstreamResponse[key];
                if (message.startsWith('WARNING:')) {
                    dispatch(
                        addErrorToast({
                            error: intl.getMessage('dns_test_warning_toast', { key }),
                        }),
                    );
                } else if (message.endsWith(': parsing error')) {
                    const info = message.substring(0, message.indexOf(':'));
                    const [sectionKey, line] = info.split(' ');
                    const section = getSectionMessage(sectionKey);
                    dispatch(
                        addErrorToast({
                            error: intl.getMessage('dns_test_parsing_error_toast', {
                                section,
                                line,
                            }),
                        }),
                    );
                } else if (message !== 'OK') {
                    dispatch(
                        addErrorToast({ error: intl.getMessage('dns_test_not_ok_toast', { key }) }),
                    );
                }
                return message;
            });

            if (
                testMessages.every((message) => message === 'OK' || message.startsWith('WARNING:'))
            ) {
                dispatch(addSuccessToast(intl.getMessage('dns_test_ok_toast')));
            }

            dispatch(testUpstreamSuccess());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(testUpstreamFailure());
        }
    };

export const testUpstreamWithFormValues =
    (formValues: any) => async (dispatch: any, getState: any) => {
        const { upstream_dns_file } = getState().dnsConfig;
        const { bootstrap_dns, upstream_dns, local_ptr_upstreams, fallback_dns } = formValues;

        return dispatch(
            testUpstream(
                {
                    bootstrap_dns,
                    upstream_dns,
                    local_ptr_upstreams,
                    fallback_dns,
                },
                upstream_dns_file,
            ),
        );
    };

export const changeLanguageRequest = createAction('CHANGE_LANGUAGE_REQUEST');
export const changeLanguageFailure = createAction('CHANGE_LANGUAGE_FAILURE');
export const changeLanguageSuccess = createAction<{ language: LocalesType }>(
    'CHANGE_LANGUAGE_SUCCESS',
);

export const changeLanguage = (lang: LocalesType) => async (dispatch: Dispatch) => {
    dispatch(changeLanguageRequest());
    try {
        await apiClient.changeLanguage({ language: lang });
        dispatch(changeLanguageSuccess({ language: lang }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(changeLanguageFailure());
    }
};

export const changeThemeRequest = createAction('CHANGE_THEME_REQUEST');
export const changeThemeFailure = createAction('CHANGE_THEME_FAILURE');
export const changeThemeSuccess = createAction<{ theme: Theme }>('CHANGE_THEME_SUCCESS');

export const changeTheme = (theme: Theme) => async (dispatch: Dispatch) => {
    dispatch(changeThemeRequest());
    try {
        await apiClient.changeTheme({ theme });
        dispatch(changeThemeSuccess({ theme }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(changeThemeFailure());
    }
};

export const getDhcpStatusRequest = createAction('GET_DHCP_STATUS_REQUEST');
export const getDhcpStatusSuccess = createAction('GET_DHCP_STATUS_SUCCESS');
export const getDhcpStatusFailure = createAction('GET_DHCP_STATUS_FAILURE');

export const getDhcpStatus = () => async (dispatch: any) => {
    dispatch(getDhcpStatusRequest());
    try {
        const globalStatus = await apiClient.getGlobalStatus();
        if (globalStatus.dhcp_available) {
            const status = await apiClient.getDhcpStatus();
            status.dhcp_available = globalStatus.dhcp_available;
            dispatch(getDhcpStatusSuccess(status));
        } else {
            dispatch(getDhcpStatusFailure());
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getDhcpStatusFailure());
    }
};

export const getDhcpInterfacesRequest = createAction('GET_DHCP_INTERFACES_REQUEST');
export const getDhcpInterfacesSuccess = createAction('GET_DHCP_INTERFACES_SUCCESS');
export const getDhcpInterfacesFailure = createAction('GET_DHCP_INTERFACES_FAILURE');

export const getDhcpInterfaces = () => async (dispatch: any) => {
    dispatch(getDhcpInterfacesRequest());
    try {
        const interfaces = await apiClient.getDhcpInterfaces();
        dispatch(getDhcpInterfacesSuccess(interfaces));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getDhcpInterfacesFailure());
    }
};

export const findActiveDhcpRequest = createAction('FIND_ACTIVE_DHCP_REQUEST');
export const findActiveDhcpSuccess = createAction('FIND_ACTIVE_DHCP_SUCCESS');
export const findActiveDhcpFailure = createAction('FIND_ACTIVE_DHCP_FAILURE');

export const toggleLeaseModal = createAction('TOGGLE_LEASE_MODAL');

export const findActiveDhcp = (selectedInterface: any) => async (dispatch: any, getState: any) => {
    dispatch(findActiveDhcpRequest());
    try {
        const req = {
            interface: selectedInterface,
        };
        const activeDhcp = await apiClient.findActiveDhcp(req);
        dispatch(findActiveDhcpSuccess(activeDhcp));
        const { check, interface_name, interfaces } = getState().dhcp;
        const v4 = check?.v4 ?? { static_ip: {}, other_server: {} };
        const v6 = check?.v6 ?? { other_server: {} };

        let isError = false;
        let isStaticIPError = false;

        const hasV4Interface = !!interfaces[selectedInterface]?.ipv4_addresses;
        const hasV6Interface = !!interfaces[selectedInterface]?.ipv6_addresses;

        if (hasV4Interface && v4.other_server.found === STATUS_RESPONSE.ERROR) {
            isError = true;
            if (v4.other_server.error) {
                dispatch(addErrorToast({ error: v4.other_server.error }));
            }
        }

        if (hasV6Interface && v6.other_server.found === STATUS_RESPONSE.ERROR) {
            isError = true;
            if (v6.other_server.error) {
                dispatch(addErrorToast({ error: v6.other_server.error }));
            }
        }

        if (hasV4Interface && v4.static_ip.static === STATUS_RESPONSE.ERROR) {
            isStaticIPError = true;
            dispatch(
                addErrorToast({
                    error: intl.getMessage('dhcp_static_ip_error'),
                    action: {
                        text: intl.getMessage('set_static_ip_manually'),
                        actionType: toggleLeaseModal.toString(),
                        actionPayload: { type: 'ADD_LEASE' },
                    },
                }),
            );
        }

        if (isError) {
            dispatch(
                addErrorToast({
                    error: intl.getMessage('dhcp_error'),
                    action: {
                        text: intl.getMessage('try_again'),
                        callback: () => dispatch(findActiveDhcp(selectedInterface)),
                    },
                }),
            );
        }

        if (isStaticIPError || isError) {
            // No need to proceed if there was an error discovering DHCP server
            return;
        }

        if (
            (hasV4Interface && v4.other_server.found === STATUS_RESPONSE.YES) ||
            (hasV6Interface && v6.other_server.found === STATUS_RESPONSE.YES)
        ) {
            dispatch(
                addErrorToast({
                    error: intl.getMessage('dhcp_found'),
                    action: {
                        text: intl.getMessage('try_again'),
                        callback: () => dispatch(findActiveDhcp(selectedInterface)),
                    },
                }),
            );
        } else if (
            hasV4Interface &&
            v4.static_ip.static === STATUS_RESPONSE.NO &&
            v4.static_ip.ip &&
            interface_name
        ) {
            dispatch(
                addErrorToast({
                    error: intl.getMessage('dhcp_dynamic_ip_found'),
                    action: {
                        text: intl.getMessage('try_again'),
                        callback: () => dispatch(findActiveDhcp(selectedInterface)),
                    },
                }),
            );
        } else {
            dispatch(addSuccessToast(intl.getMessage('dhcp_not_found')));
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(findActiveDhcpFailure());
    }
};

export const setDhcpConfigRequest = createAction('SET_DHCP_CONFIG_REQUEST');
export const setDhcpConfigSuccess = createAction('SET_DHCP_CONFIG_SUCCESS');
export const setDhcpConfigFailure = createAction('SET_DHCP_CONFIG_FAILURE');

export const setDhcpConfig = (values: any) => async (dispatch: any) => {
    dispatch(setDhcpConfigRequest());
    try {
        await apiClient.setDhcpConfig(values);
        dispatch(setDhcpConfigSuccess(values));
        dispatch(addSuccessToast(intl.getMessage('dhcp_config_saved')));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setDhcpConfigFailure());
    }
};

export const toggleDhcpRequest = createAction('TOGGLE_DHCP_REQUEST');
export const toggleDhcpFailure = createAction('TOGGLE_DHCP_FAILURE');
export const toggleDhcpSuccess = createAction('TOGGLE_DHCP_SUCCESS');

export const toggleDhcp = (values: any) => async (dispatch: any) => {
    dispatch(toggleDhcpRequest());
    let config = {
        ...values,
        enabled: false,
    };
    let successMessage = intl.getMessage('disabled_dhcp');

    if (!values.enabled) {
        config = {
            ...values,
            enabled: true,
        };
        successMessage = intl.getMessage('enabled_dhcp');
    }

    try {
        await apiClient.setDhcpConfig(config);
        dispatch(toggleDhcpSuccess());
        dispatch(addSuccessToast(successMessage));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleDhcpFailure());
    }
};

export const resetDhcpRequest = createAction('RESET_DHCP_REQUEST');
export const resetDhcpSuccess = createAction('RESET_DHCP_SUCCESS');
export const resetDhcpFailure = createAction('RESET_DHCP_FAILURE');

export const resetDhcp = () => async (dispatch: any) => {
    dispatch(resetDhcpRequest());
    try {
        const status = await apiClient.resetDhcp();
        dispatch(resetDhcpSuccess(status));
        dispatch(addSuccessToast(intl.getMessage('dhcp_config_saved')));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(resetDhcpFailure());
    }
};

export const resetDhcpLeasesRequest = createAction('RESET_DHCP_LEASES_REQUEST');
export const resetDhcpLeasesSuccess = createAction('RESET_DHCP_LEASES_SUCCESS');
export const resetDhcpLeasesFailure = createAction('RESET_DHCP_LEASES_FAILURE');

export const resetDhcpLeases = () => async (dispatch: any) => {
    dispatch(resetDhcpLeasesRequest());
    try {
        const status = await apiClient.resetDhcpLeases();
        dispatch(resetDhcpLeasesSuccess(status));
        dispatch(addSuccessToast(intl.getMessage('dhcp_reset_leases_success')));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(resetDhcpLeasesFailure());
    }
};

export const addStaticLeaseRequest = createAction('ADD_STATIC_LEASE_REQUEST');
export const addStaticLeaseFailure = createAction('ADD_STATIC_LEASE_FAILURE');
export const addStaticLeaseSuccess = createAction('ADD_STATIC_LEASE_SUCCESS');

export const addStaticLease = (config: any) => async (dispatch: any) => {
    dispatch(addStaticLeaseRequest());
    try {
        const name = config.hostname || config.ip;
        await apiClient.addStaticLease(config);
        dispatch(addStaticLeaseSuccess(config));
        dispatch(addSuccessToast(intl.getMessage('dhcp_lease_added', { key: name })));
        dispatch(toggleLeaseModal());
        dispatch(getDhcpStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(addStaticLeaseFailure());
    }
};

export const removeStaticLeaseRequest = createAction('REMOVE_STATIC_LEASE_REQUEST');
export const removeStaticLeaseFailure = createAction('REMOVE_STATIC_LEASE_FAILURE');
export const removeStaticLeaseSuccess = createAction('REMOVE_STATIC_LEASE_SUCCESS');

export const removeStaticLease = (config: any) => async (dispatch: any) => {
    dispatch(removeStaticLeaseRequest());
    try {
        const name = config.hostname || config.ip;
        await apiClient.removeStaticLease(config);
        dispatch(removeStaticLeaseSuccess(config));
        dispatch(addSuccessToast(intl.getMessage('dhcp_lease_deleted', { key: name })));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(removeStaticLeaseFailure());
    }
};

export const updateStaticLeaseRequest = createAction('UPDATE_STATIC_LEASE_REQUEST');
export const updateStaticLeaseFailure = createAction('UPDATE_STATIC_LEASE_FAILURE');
export const updateStaticLeaseSuccess = createAction('UPDATE_STATIC_LEASE_SUCCESS');

export const updateStaticLease = (config: any) => async (dispatch: any) => {
    dispatch(updateStaticLeaseRequest());
    try {
        await apiClient.updateStaticLease(config);
        dispatch(updateStaticLeaseSuccess(config));
        dispatch(
            addSuccessToast(
                intl.getMessage('dhcp_lease_updated', { key: config.hostname || config.ip }),
            ),
        );
        dispatch(toggleLeaseModal());
        dispatch(getDhcpStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(updateStaticLeaseFailure());
    }
};

export const removeToast = createAction('REMOVE_TOAST');

export const toggleBlocking =
    (
        type: BlockAction,
        domain: string,
        baseRule?: string,
        baseUnblocking?: string,
        matchedRuleToReplace?: string,
    ) =>
    async (dispatch: AppDispatch, getState: AppGetState): Promise<boolean> => {
        const baseBlockingRule = baseRule || `||${domain}^$important`;
        const baseUnblockingRule = baseUnblocking || `@@${baseBlockingRule}`;
        const previousRules = getState().filtering?.userRules || '';
        const desiredRule = type === BLOCK_ACTIONS.BLOCK ? baseBlockingRule : baseUnblockingRule;
        const oppositeRule = type === BLOCK_ACTIONS.BLOCK ? baseUnblockingRule : baseBlockingRule;
        const currentRules = splitByNewLine(previousRules);
        const hasDesiredRule = currentRules.includes(desiredRule);
        const rulesToReplace = [oppositeRule, matchedRuleToReplace].filter(
            (rule): rule is string => Boolean(rule) && rule !== desiredRule,
        );
        const hasRuleToReplace = rulesToReplace.some((rule) => currentRules.includes(rule));
        const addedRuleMessageKey = intl.getMessage(
            'user_rules_rule_added_to_custom_filtering_rules',
        );
        const undoToastPayload = createUndoToast(
            addedRuleMessageKey,
            intl.getMessage('notify_undo'),
            async () => {
                const didUndo = (await dispatch(setRules(previousRules))) as boolean;

                if (didUndo) {
                    await dispatch(getFilteringStatus());
                }
            },
        );

        if (hasDesiredRule && !hasRuleToReplace) {
            return true;
        }

        const rulesToRemove = new Set([desiredRule, ...rulesToReplace]);
        const updatedRules = currentRules.filter((rule: string) => !rulesToRemove.has(rule));
        updatedRules.push(desiredRule);

        const didSave = (await dispatch(setRules(`${updatedRules.join('\n')}\n`))) as boolean;

        if (!didSave) {
            return false;
        }

        dispatch(addSuccessToast(undoToastPayload));

        await dispatch(getFilteringStatus());

        return true;
    };

export const toggleBlockingForClient = (type: BlockAction, domain: string, client: string) => {
    const escapedClientName = client
        .replace(/'/g, "\\'")
        .replace(/"/g, '\\"')
        .replace(/,/g, '\\,')
        .replace(/\|/g, '\\|');
    const baseRule = `||${domain}^$client='${escapedClientName}'`;
    const baseUnblocking = `@@${baseRule}`;

    return toggleBlocking(type, domain, baseRule, baseUnblocking);
};

export const blockDomainForClient = (domain: string, client: string) =>
    toggleBlockingForClient(BLOCK_ACTIONS.BLOCK, domain, client);

export const blockDomain =
    (domain: string) => async (dispatch: AppDispatch, getState: AppGetState) => {
        const previousRules = getState().filtering?.userRules || '';
        const rule = `||${domain}^$important`;
        const desiredRule = rule;
        const currentRules = splitByNewLine(previousRules);

        if (currentRules.includes(desiredRule)) {
            return true;
        }

        const updatedRules = [
            ...currentRules.filter((r: string) => r !== `@@${rule}`),
            desiredRule,
        ];
        const didSave = (await dispatch(setRules(`${updatedRules.join('\n')}\n`))) as boolean;

        if (!didSave) {
            return false;
        }

        dispatch(
            addSuccessToast(
                createUndoToast(
                    intl.getMessage('user_rules_rule_added_to_custom_filtering_rules'),
                    intl.getMessage('notify_undo'),
                    async () => {
                        const didUndo = (await dispatch(setRules(previousRules))) as boolean;

                        if (didUndo) {
                            await dispatch(getFilteringStatus());
                        }
                    },
                ),
            ),
        );

        await dispatch(getFilteringStatus());

        return true;
    };

export const unblockDomain =
    (domain: string) => async (dispatch: AppDispatch, getState: AppGetState) => {
        const previousRules = getState().filtering?.userRules || '';
        const rule = `||${domain}^$important`;
        const desiredRule = `@@${rule}`;
        const currentRules = splitByNewLine(previousRules);

        if (currentRules.includes(desiredRule)) {
            return true;
        }

        const updatedRules = [...currentRules.filter((r: string) => r !== rule), desiredRule];
        const didSave = (await dispatch(setRules(`${updatedRules.join('\n')}\n`))) as boolean;

        if (!didSave) {
            return false;
        }

        dispatch(
            addSuccessToast(
                createUndoToast(
                    intl.getMessage('user_rules_rule_added_to_custom_filtering_rules'),
                    intl.getMessage('notify_undo'),
                    async () => {
                        const didUndo = (await dispatch(setRules(previousRules))) as boolean;

                        if (didUndo) {
                            await dispatch(getFilteringStatus());
                        }
                    },
                ),
            ),
        );

        await dispatch(getFilteringStatus());

        return true;
    };
