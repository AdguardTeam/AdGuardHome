import { createAction } from 'redux-actions';
import i18next from 'i18next';
import axios from 'axios';

import endsWith from 'lodash/endsWith';
import escapeRegExp from 'lodash/escapeRegExp';
import React from 'react';
import { compose } from 'redux';
import {
    splitByNewLine,
    sortClients,
    filterOutComments,
    msToSeconds,
    msToMinutes,
    msToHours,
} from '../helpers/helpers';
import {
    BLOCK_ACTIONS,
    CHECK_TIMEOUT,
    STATUS_RESPONSE,
    SETTINGS_NAMES,
    FORM_NAME,
    MANUAL_UPDATE_LINK,
    DISABLE_PROTECTION_TIMINGS,
} from '../helpers/constants';
import { areEqualVersions } from '../helpers/version';
import { getTlsStatus } from './encryption';
import apiClient from '../api/Api';
import { addErrorToast, addNoticeToast, addSuccessToast } from './toasts';
import { getFilteringStatus, setRules } from './filtering';

export const toggleSettingStatus = createAction('SETTING_STATUS_TOGGLE');
export const showSettingsFailure = createAction('SETTINGS_FAILURE_SHOW');

/**
 *
 * @param {*} settingKey = SETTINGS_NAMES
 * @param {*} status: boolean | SafeSearchConfig
 * @returns
 */
export const toggleSetting = (settingKey, status) => async (dispatch) => {
    let successMessage = '';
    try {
        switch (settingKey) {
            case SETTINGS_NAMES.safebrowsing:
                if (status) {
                    successMessage = 'disabled_safe_browsing_toast';
                    await apiClient.disableSafebrowsing();
                } else {
                    successMessage = 'enabled_safe_browsing_toast';
                    await apiClient.enableSafebrowsing();
                }
                dispatch(toggleSettingStatus({ settingKey }));
                break;
            case SETTINGS_NAMES.parental:
                if (status) {
                    successMessage = 'disabled_parental_toast';
                    await apiClient.disableParentalControl();
                } else {
                    successMessage = 'enabled_parental_toast';
                    await apiClient.enableParentalControl();
                }
                dispatch(toggleSettingStatus({ settingKey }));
                break;
            case SETTINGS_NAMES.safesearch:
                successMessage = 'updated_save_search_toast';
                await apiClient.updateSafesearch(status);
                dispatch(toggleSettingStatus({ settingKey, value: status }));
                break;
            default:
                break;
        }
        dispatch(addSuccessToast(successMessage));
    } catch (error) {
        dispatch(addErrorToast({ error }));
    }
};

export const initSettingsRequest = createAction('SETTINGS_INIT_REQUEST');
export const initSettingsFailure = createAction('SETTINGS_INIT_FAILURE');
export const initSettingsSuccess = createAction('SETTINGS_INIT_SUCCESS');

export const initSettings = (settingsList = {
    safebrowsing: {}, parental: {},
}) => async (dispatch) => {
    dispatch(initSettingsRequest());
    try {
        const safebrowsingStatus = await apiClient.getSafebrowsingStatus();
        const parentalStatus = await apiClient.getParentalStatus();
        const safesearchStatus = await apiClient.getSafesearchStatus();
        const {
            safebrowsing,
            parental,
        } = settingsList;
        const newSettingsList = {
            safebrowsing: {
                ...safebrowsing,
                enabled: safebrowsingStatus.enabled,
            },
            parental: {
                ...parental,
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

const getDisabledMessage = (time) => {
    switch (time) {
        case DISABLE_PROTECTION_TIMINGS.HALF_MINUTE:
            return i18next.t(
                'disable_notify_for_seconds',
                { count: msToSeconds(DISABLE_PROTECTION_TIMINGS.HALF_MINUTE) },
            );
        case DISABLE_PROTECTION_TIMINGS.MINUTE:
            return i18next.t(
                'disable_notify_for_minutes',
                { count: msToMinutes(DISABLE_PROTECTION_TIMINGS.MINUTE) },
            );
        case DISABLE_PROTECTION_TIMINGS.TEN_MINUTES:
            return i18next.t(
                'disable_notify_for_minutes',
                { count: msToMinutes(DISABLE_PROTECTION_TIMINGS.TEN_MINUTES) },
            );
        case DISABLE_PROTECTION_TIMINGS.HOUR:
            return i18next.t(
                'disable_notify_for_hours',
                { count: msToHours(DISABLE_PROTECTION_TIMINGS.HOUR) },
            );
        case DISABLE_PROTECTION_TIMINGS.TOMORROW:
            return i18next.t('disable_notify_until_tomorrow');
        default:
            return 'disabled_protection';
    }
};

export const toggleProtection = (status, time = null) => async (dispatch) => {
    dispatch(toggleProtectionRequest());
    try {
        const successMessage = status ? getDisabledMessage(time) : 'enabled_protection';
        await apiClient.setProtection({ enabled: !status, duration: time });
        dispatch(addSuccessToast(successMessage));
        dispatch(toggleProtectionSuccess({ disabledDuration: time }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleProtectionFailure());
    }
};

export const setDisableDurationTime = createAction('SET_DISABLED_DURATION_TIME');

export const setProtectionTimerTime = (updatedTime) => async (dispatch) => {
    dispatch(setDisableDurationTime({ timeToEnableProtection: updatedTime }));
};

export const getVersionRequest = createAction('GET_VERSION_REQUEST');
export const getVersionFailure = createAction('GET_VERSION_FAILURE');
export const getVersionSuccess = createAction('GET_VERSION_SUCCESS');

export const getVersion = (recheck = false) => async (dispatch, getState) => {
    dispatch(getVersionRequest());
    try {
        const data = await apiClient.getGlobalVersion({ recheck_now: recheck });
        dispatch(getVersionSuccess(data));

        if (recheck) {
            const { dnsVersion } = getState().dashboard;
            const currentVersion = dnsVersion === 'undefined' ? 0 : dnsVersion;

            if (data && !areEqualVersions(currentVersion, data.new_version)) {
                dispatch(addSuccessToast('updates_checked'));
            } else {
                dispatch(addSuccessToast('updates_version_equal'));
            }
        }
    } catch (error) {
        dispatch(addErrorToast({ error: 'version_request_error' }));
        dispatch(getVersionFailure());
    }
};

export const getUpdateRequest = createAction('GET_UPDATE_REQUEST');
export const getUpdateFailure = createAction('GET_UPDATE_FAILURE');
export const getUpdateSuccess = createAction('GET_UPDATE_SUCCESS');

const checkStatus = async (handleRequestSuccess, handleRequestError, attempts = 60) => {
    let timeout;

    if (attempts === 0) {
        handleRequestError();
    }

    const rmTimeout = (t) => t && clearTimeout(t);

    try {
        const response = await axios.get(`${apiClient.baseUrl}/status`);
        rmTimeout(timeout);
        if (response?.status === 200) {
            handleRequestSuccess(response);
            if (response.data.running === false) {
                timeout = setTimeout(
                    checkStatus,
                    CHECK_TIMEOUT,
                    handleRequestSuccess,
                    handleRequestError,
                    attempts - 1,
                );
            }
        }
    } catch (error) {
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

export const getUpdate = () => async (dispatch, getState) => {
    const { dnsVersion } = getState().dashboard;

    dispatch(getUpdateRequest());
    const handleRequestError = () => {
        const options = {
            components: {
                a: <a href={MANUAL_UPDATE_LINK} target="_blank"
                      rel="noopener noreferrer" />,
            },
        };

        dispatch(addNoticeToast({ error: 'update_failed', options }));
        dispatch(getUpdateFailure());
    };

    const handleRequestSuccess = (response) => {
        const responseVersion = response.data?.version;

        if (dnsVersion !== responseVersion) {
            dispatch(getUpdateSuccess());
            window.location.reload(true);
        }
    };

    try {
        await apiClient.getUpdate();
        checkStatus(handleRequestSuccess, handleRequestError);
    } catch (error) {
        handleRequestError();
    }
};

export const getClientsRequest = createAction('GET_CLIENTS_REQUEST');
export const getClientsFailure = createAction('GET_CLIENTS_FAILURE');
export const getClientsSuccess = createAction('GET_CLIENTS_SUCCESS');

export const getClients = () => async (dispatch) => {
    dispatch(getClientsRequest());
    try {
        const data = await apiClient.getClients();
        const sortedClients = data.clients && sortClients(data.clients);
        const sortedAutoClients = data.auto_clients && sortClients(data.auto_clients);

        dispatch(getClientsSuccess({
            clients: sortedClients || [],
            autoClients: sortedAutoClients || [],
            supportedTags: data.supported_tags || [],
        }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getClientsFailure());
    }
};

export const getProfileRequest = createAction('GET_PROFILE_REQUEST');
export const getProfileFailure = createAction('GET_PROFILE_FAILURE');
export const getProfileSuccess = createAction('GET_PROFILE_SUCCESS');

export const getProfile = () => async (dispatch) => {
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

export const getDnsStatus = () => async (dispatch) => {
    dispatch(dnsStatusRequest());

    const handleRequestError = () => {
        dispatch(addErrorToast({ error: 'dns_status_error' }));
        dispatch(dnsStatusFailure());
        window.location.reload(true);
    };

    const handleRequestSuccess = (response) => {
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
    } catch (error) {
        handleRequestError();
    }
};

export const timerStatusRequest = createAction('TIMER_STATUS_REQUEST');
export const timerStatusFailure = createAction('TIMER_STATUS_FAILURE');
export const timerStatusSuccess = createAction('TIMER_STATUS_SUCCESS');

export const getTimerStatus = () => async (dispatch) => {
    dispatch(timerStatusRequest());

    const handleRequestError = () => {
        dispatch(addErrorToast({ error: 'dns_status_error' }));
        dispatch(dnsStatusFailure());
        window.location.reload(true);
    };

    const handleRequestSuccess = (response) => {
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
    } catch (error) {
        handleRequestError();
    }
};

export const testUpstreamRequest = createAction('TEST_UPSTREAM_REQUEST');
export const testUpstreamFailure = createAction('TEST_UPSTREAM_FAILURE');
export const testUpstreamSuccess = createAction('TEST_UPSTREAM_SUCCESS');

export const testUpstream = (
    {
        bootstrap_dns,
        upstream_dns,
        local_ptr_upstreams,
        fallback_dns,
    }, upstream_dns_file,
) => async (dispatch) => {
    dispatch(testUpstreamRequest());
    try {
        const removeComments = compose(filterOutComments, splitByNewLine);

        const config = {
            bootstrap_dns: splitByNewLine(bootstrap_dns),
            private_upstream: splitByNewLine(local_ptr_upstreams),
            fallback_dns: splitByNewLine(fallback_dns),
            ...(upstream_dns_file ? null : {
                upstream_dns: removeComments(upstream_dns),
            }),
        };

        const upstreamResponse = await apiClient.testUpstream(config);
        const testMessages = Object.keys(upstreamResponse)
            .map((key) => {
                const message = upstreamResponse[key];
                if (message.startsWith('WARNING:')) {
                    dispatch(addErrorToast({ error: i18next.t('dns_test_warning_toast', { key }) }));
                } else if (message !== 'OK') {
                    dispatch(addErrorToast({ error: i18next.t('dns_test_not_ok_toast', { key }) }));
                }
                return message;
            });

        if (testMessages.every((message) => message === 'OK' || message.startsWith('WARNING:'))) {
            dispatch(addSuccessToast('dns_test_ok_toast'));
        }

        dispatch(testUpstreamSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(testUpstreamFailure());
    }
};

export const testUpstreamWithFormValues = () => async (dispatch, getState) => {
    const { upstream_dns_file } = getState().dnsConfig;
    const {
        bootstrap_dns,
        upstream_dns,
        local_ptr_upstreams,
        fallback_dns,
    } = getState().form[FORM_NAME.UPSTREAM].values;

    return dispatch(testUpstream({
        bootstrap_dns,
        upstream_dns,
        local_ptr_upstreams,
        fallback_dns,
    }, upstream_dns_file));
};

export const changeLanguageRequest = createAction('CHANGE_LANGUAGE_REQUEST');
export const changeLanguageFailure = createAction('CHANGE_LANGUAGE_FAILURE');
export const changeLanguageSuccess = createAction('CHANGE_LANGUAGE_SUCCESS');

export const changeLanguage = (lang) => async (dispatch) => {
    dispatch(changeLanguageRequest());
    try {
        await apiClient.changeLanguage({ language: lang });
        dispatch(changeLanguageSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(changeLanguageFailure());
    }
};

export const changeThemeRequest = createAction('CHANGE_THEME_REQUEST');
export const changeThemeFailure = createAction('CHANGE_THEME_FAILURE');
export const changeThemeSuccess = createAction('CHANGE_THEME_SUCCESS');

export const changeTheme = (theme) => async (dispatch) => {
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

export const getDhcpStatus = () => async (dispatch) => {
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

export const getDhcpInterfaces = () => async (dispatch) => {
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

export const findActiveDhcp = (name) => async (dispatch, getState) => {
    dispatch(findActiveDhcpRequest());
    try {
        const req = {
            interface: name,
        };
        const activeDhcp = await apiClient.findActiveDhcp(req);
        dispatch(findActiveDhcpSuccess(activeDhcp));
        const { check, interface_name, interfaces } = getState().dhcp;
        const selectedInterface = getState().form[FORM_NAME.DHCP_INTERFACES].values.interface_name;
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
            dispatch(addErrorToast({ error: 'dhcp_static_ip_error' }));
        }

        if (isError) {
            dispatch(addErrorToast({ error: 'dhcp_error' }));
        }

        if (isStaticIPError || isError) {
            // No need to proceed if there was an error discovering DHCP server
            return;
        }

        if ((hasV4Interface && v4.other_server.found === STATUS_RESPONSE.YES)
                || (hasV6Interface && v6.other_server.found === STATUS_RESPONSE.YES)) {
            dispatch(addErrorToast({ error: 'dhcp_found' }));
        } else if (hasV4Interface && v4.static_ip.static === STATUS_RESPONSE.NO
                && v4.static_ip.ip
                && interface_name) {
            const warning = i18next.t('dhcp_dynamic_ip_found', {
                interfaceName: interface_name,
                ipAddress: v4.static_ip.ip,
                interpolation: {
                    prefix: '<0>{{',
                    suffix: '}}</0>',
                },
            });
            dispatch(addErrorToast({ error: warning }));
        } else {
            dispatch(addSuccessToast('dhcp_not_found'));
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(findActiveDhcpFailure());
    }
};

export const setDhcpConfigRequest = createAction('SET_DHCP_CONFIG_REQUEST');
export const setDhcpConfigSuccess = createAction('SET_DHCP_CONFIG_SUCCESS');
export const setDhcpConfigFailure = createAction('SET_DHCP_CONFIG_FAILURE');

export const setDhcpConfig = (values) => async (dispatch) => {
    dispatch(setDhcpConfigRequest());
    try {
        await apiClient.setDhcpConfig(values);
        dispatch(setDhcpConfigSuccess(values));
        dispatch(addSuccessToast('dhcp_config_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setDhcpConfigFailure());
    }
};

export const toggleDhcpRequest = createAction('TOGGLE_DHCP_REQUEST');
export const toggleDhcpFailure = createAction('TOGGLE_DHCP_FAILURE');
export const toggleDhcpSuccess = createAction('TOGGLE_DHCP_SUCCESS');

export const toggleDhcp = (values) => async (dispatch) => {
    dispatch(toggleDhcpRequest());
    let config = {
        ...values,
        enabled: false,
    };
    let successMessage = 'disabled_dhcp';

    if (!values.enabled) {
        config = {
            ...values,
            enabled: true,
        };
        successMessage = 'enabled_dhcp';
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

export const resetDhcp = () => async (dispatch) => {
    dispatch(resetDhcpRequest());
    try {
        const status = await apiClient.resetDhcp();
        dispatch(resetDhcpSuccess(status));
        dispatch(addSuccessToast('dhcp_config_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(resetDhcpFailure());
    }
};

export const resetDhcpLeasesRequest = createAction('RESET_DHCP_LEASES_REQUEST');
export const resetDhcpLeasesSuccess = createAction('RESET_DHCP_LEASES_SUCCESS');
export const resetDhcpLeasesFailure = createAction('RESET_DHCP_LEASES_FAILURE');

export const resetDhcpLeases = () => async (dispatch) => {
    dispatch(resetDhcpLeasesRequest());
    try {
        const status = await apiClient.resetDhcpLeases();
        dispatch(resetDhcpLeasesSuccess(status));
        dispatch(addSuccessToast('dhcp_reset_leases_success'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(resetDhcpLeasesFailure());
    }
};

export const toggleLeaseModal = createAction('TOGGLE_LEASE_MODAL');

export const addStaticLeaseRequest = createAction('ADD_STATIC_LEASE_REQUEST');
export const addStaticLeaseFailure = createAction('ADD_STATIC_LEASE_FAILURE');
export const addStaticLeaseSuccess = createAction('ADD_STATIC_LEASE_SUCCESS');

export const addStaticLease = (config) => async (dispatch) => {
    dispatch(addStaticLeaseRequest());
    try {
        const name = config.hostname || config.ip;
        await apiClient.addStaticLease(config);
        dispatch(addStaticLeaseSuccess(config));
        dispatch(addSuccessToast(i18next.t('dhcp_lease_added', { key: name })));
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

export const removeStaticLease = (config) => async (dispatch) => {
    dispatch(removeStaticLeaseRequest());
    try {
        const name = config.hostname || config.ip;
        await apiClient.removeStaticLease(config);
        dispatch(removeStaticLeaseSuccess(config));
        dispatch(addSuccessToast(i18next.t('dhcp_lease_deleted', { key: name })));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(removeStaticLeaseFailure());
    }
};

export const updateStaticLeaseRequest = createAction('UPDATE_STATIC_LEASE_REQUEST');
export const updateStaticLeaseFailure = createAction('UPDATE_STATIC_LEASE_FAILURE');
export const updateStaticLeaseSuccess = createAction('UPDATE_STATIC_LEASE_SUCCESS');

export const updateStaticLease = (config) => async (dispatch) => {
    dispatch(updateStaticLeaseRequest());
    try {
        await apiClient.updateStaticLease(config);
        dispatch(updateStaticLeaseSuccess(config));
        dispatch(addSuccessToast(i18next.t('dhcp_lease_updated', { key: config.hostname || config.ip })));
        dispatch(toggleLeaseModal());
        dispatch(getDhcpStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(updateStaticLeaseFailure());
    }
};

export const removeToast = createAction('REMOVE_TOAST');

export const toggleBlocking = (
    type, domain, baseRule, baseUnblocking,
) => async (dispatch, getState) => {
    const baseBlockingRule = baseRule || `||${domain}^$important`;
    const baseUnblockingRule = baseUnblocking || `@@${baseBlockingRule}`;
    const { userRules } = getState().filtering;

    const lineEnding = !endsWith(userRules, '\n') ? '\n' : '';

    const blockingRule = type === BLOCK_ACTIONS.BLOCK ? baseUnblockingRule : baseBlockingRule;
    const unblockingRule = type === BLOCK_ACTIONS.BLOCK ? baseBlockingRule : baseUnblockingRule;
    const preparedBlockingRule = new RegExp(`(^|\n)${escapeRegExp(blockingRule)}($|\n)`);
    const preparedUnblockingRule = new RegExp(`(^|\n)${escapeRegExp(unblockingRule)}($|\n)`);

    const matchPreparedBlockingRule = userRules.match(preparedBlockingRule);
    const matchPreparedUnblockingRule = userRules.match(preparedUnblockingRule);

    if (matchPreparedBlockingRule) {
        await dispatch(setRules(userRules.replace(`${blockingRule}`, '')));
        dispatch(addSuccessToast(i18next.t('rule_removed_from_custom_filtering_toast', { rule: blockingRule })));
    } else if (!matchPreparedUnblockingRule) {
        await dispatch(setRules(`${userRules}${lineEnding}${unblockingRule}\n`));
        dispatch(addSuccessToast(i18next.t('rule_added_to_custom_filtering_toast', { rule: unblockingRule })));
    } else if (matchPreparedUnblockingRule) {
        dispatch(addSuccessToast(i18next.t('rule_added_to_custom_filtering_toast', { rule: unblockingRule })));
        return;
    } else if (!matchPreparedBlockingRule) {
        dispatch(addSuccessToast(i18next.t('rule_removed_from_custom_filtering_toast', { rule: blockingRule })));
        return;
    }

    dispatch(getFilteringStatus());
};

export const toggleBlockingForClient = (type, domain, client) => {
    const escapedClientName = client.replace(/'/g, '\\\'')
        .replace(/"/g, '\\"')
        .replace(/,/g, '\\,')
        .replace(/\|/g, '\\|');
    const baseRule = `||${domain}^$client='${escapedClientName}'`;
    const baseUnblocking = `@@${baseRule}`;

    return toggleBlocking(type, domain, baseRule, baseUnblocking);
};
