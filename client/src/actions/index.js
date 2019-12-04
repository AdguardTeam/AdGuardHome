import { createAction } from 'redux-actions';
import { t } from 'i18next';
import axios from 'axios';

import { normalizeTextarea, sortClients, isVersionGreater } from '../helpers/helpers';
import { SETTINGS_NAMES, CHECK_TIMEOUT } from '../helpers/constants';
import { getTlsStatus } from './encryption';
import apiClient from '../api/Api';

export const addErrorToast = createAction('ADD_ERROR_TOAST');
export const addSuccessToast = createAction('ADD_SUCCESS_TOAST');
export const addNoticeToast = createAction('ADD_NOTICE_TOAST');
export const removeToast = createAction('REMOVE_TOAST');

export const toggleSettingStatus = createAction('SETTING_STATUS_TOGGLE');
export const showSettingsFailure = createAction('SETTINGS_FAILURE_SHOW');

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
                if (status) {
                    successMessage = 'disabled_safe_search_toast';
                    await apiClient.disableSafesearch();
                } else {
                    successMessage = 'enabled_save_search_toast';
                    await apiClient.enableSafesearch();
                }
                dispatch(toggleSettingStatus({ settingKey }));
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

export const initSettings = settingsList => async (dispatch) => {
    dispatch(initSettingsRequest());
    try {
        const safebrowsingStatus = await apiClient.getSafebrowsingStatus();
        const parentalStatus = await apiClient.getParentalStatus();
        const safesearchStatus = await apiClient.getSafesearchStatus();
        const {
            safebrowsing,
            parental,
            safesearch,
        } = settingsList;
        const newSettingsList = {
            safebrowsing: { ...safebrowsing, enabled: safebrowsingStatus.enabled },
            parental: { ...parental, enabled: parentalStatus.enabled },
            safesearch: { ...safesearch, enabled: safesearchStatus.enabled },
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

export const toggleProtection = status => async (dispatch) => {
    dispatch(toggleProtectionRequest());
    try {
        const successMessage = status ? 'disabled_protection' : 'enabled_protection';
        await apiClient.setDnsConfig({ protection_enabled: !status });
        dispatch(addSuccessToast(successMessage));
        dispatch(toggleProtectionSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleProtectionFailure());
    }
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

            if (data && isVersionGreater(currentVersion, data.new_version)) {
                dispatch(addSuccessToast('updates_checked'));
            } else {
                dispatch(addSuccessToast('updates_version_equal'));
            }
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getVersionFailure());
    }
};

export const getUpdateRequest = createAction('GET_UPDATE_REQUEST');
export const getUpdateFailure = createAction('GET_UPDATE_FAILURE');
export const getUpdateSuccess = createAction('GET_UPDATE_SUCCESS');

export const getUpdate = () => async (dispatch, getState) => {
    const { dnsVersion } = getState().dashboard;

    dispatch(getUpdateRequest());
    try {
        await apiClient.getUpdate();

        const checkUpdate = async (attempts) => {
            let count = attempts || 1;
            let timeout;

            if (count > 60) {
                dispatch(addNoticeToast({ error: 'update_failed' }));
                dispatch(getUpdateFailure());
                return false;
            }

            const rmTimeout = t => t && clearTimeout(t);
            const setRecursiveTimeout = (time, ...args) => setTimeout(
                checkUpdate,
                time,
                ...args,
            );

            axios.get('control/status')
                .then((response) => {
                    rmTimeout(timeout);
                    if (response && response.status === 200) {
                        const responseVersion = response.data && response.data.version;

                        if (dnsVersion !== responseVersion) {
                            dispatch(getUpdateSuccess());
                            window.location.reload(true);
                        }
                    }
                    timeout = setRecursiveTimeout(CHECK_TIMEOUT, count += 1);
                })
                .catch(() => {
                    rmTimeout(timeout);
                    timeout = setRecursiveTimeout(CHECK_TIMEOUT, count += 1);
                });

            return false;
        };

        checkUpdate();
    } catch (error) {
        dispatch(addNoticeToast({ error: 'update_failed' }));
        dispatch(getUpdateFailure());
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

export const getDnsStatus = () => async (dispatch) => {
    dispatch(dnsStatusRequest());
    try {
        const dnsStatus = await apiClient.getGlobalStatus();
        dispatch(dnsStatusSuccess(dnsStatus));
        dispatch(getVersion());
        dispatch(getTlsStatus());
        dispatch(getProfile());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(dnsStatusFailure());
    }
};

export const getDnsSettingsRequest = createAction('GET_DNS_SETTINGS_REQUEST');
export const getDnsSettingsFailure = createAction('GET_DNS_SETTINGS_FAILURE');
export const getDnsSettingsSuccess = createAction('GET_DNS_SETTINGS_SUCCESS');

export const getDnsSettings = () => async (dispatch) => {
    dispatch(getDnsSettingsRequest());
    try {
        const dnsStatus = await apiClient.getGlobalStatus();
        dispatch(getDnsSettingsSuccess(dnsStatus));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getDnsSettingsFailure());
    }
};

export const enableDnsRequest = createAction('ENABLE_DNS_REQUEST');
export const enableDnsFailure = createAction('ENABLE_DNS_FAILURE');
export const enableDnsSuccess = createAction('ENABLE_DNS_SUCCESS');

export const enableDns = () => async (dispatch) => {
    dispatch(enableDnsRequest());
    try {
        await apiClient.startGlobalFiltering();
        dispatch(enableDnsSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(enableDnsFailure());
    }
};

export const disableDnsRequest = createAction('DISABLE_DNS_REQUEST');
export const disableDnsFailure = createAction('DISABLE_DNS_FAILURE');
export const disableDnsSuccess = createAction('DISABLE_DNS_SUCCESS');

export const disableDns = () => async (dispatch) => {
    dispatch(disableDnsRequest());
    try {
        await apiClient.stopGlobalFiltering();
        dispatch(disableDnsSuccess());
    } catch (error) {
        dispatch(disableDnsFailure(error));
        dispatch(addErrorToast({ error }));
    }
};

export const handleUpstreamChange = createAction('HANDLE_UPSTREAM_CHANGE');
export const setUpstreamRequest = createAction('SET_UPSTREAM_REQUEST');
export const setUpstreamFailure = createAction('SET_UPSTREAM_FAILURE');
export const setUpstreamSuccess = createAction('SET_UPSTREAM_SUCCESS');

export const setUpstream = config => async (dispatch) => {
    dispatch(setUpstreamRequest());
    try {
        const values = { ...config };
        values.bootstrap_dns = (
            values.bootstrap_dns && normalizeTextarea(values.bootstrap_dns)
        ) || [];
        values.upstream_dns = (
            values.upstream_dns && normalizeTextarea(values.upstream_dns)
        ) || [];

        await apiClient.setUpstream(values);
        dispatch(addSuccessToast('updated_upstream_dns_toast'));
        dispatch(setUpstreamSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setUpstreamFailure());
    }
};

export const testUpstreamRequest = createAction('TEST_UPSTREAM_REQUEST');
export const testUpstreamFailure = createAction('TEST_UPSTREAM_FAILURE');
export const testUpstreamSuccess = createAction('TEST_UPSTREAM_SUCCESS');

export const testUpstream = config => async (dispatch) => {
    dispatch(testUpstreamRequest());
    try {
        const values = { ...config };
        values.bootstrap_dns = (
            values.bootstrap_dns && normalizeTextarea(values.bootstrap_dns)
        ) || [];
        values.upstream_dns = (
            values.upstream_dns && normalizeTextarea(values.upstream_dns)
        ) || [];

        const upstreamResponse = await apiClient.testUpstream(values);
        const testMessages = Object.keys(upstreamResponse).map((key) => {
            const message = upstreamResponse[key];
            if (message !== 'OK') {
                dispatch(addErrorToast({ error: t('dns_test_not_ok_toast', { key }) }));
            }
            return message;
        });

        if (testMessages.every(message => message === 'OK')) {
            dispatch(addSuccessToast('dns_test_ok_toast'));
        }

        dispatch(testUpstreamSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(testUpstreamFailure());
    }
};

export const changeLanguageRequest = createAction('CHANGE_LANGUAGE_REQUEST');
export const changeLanguageFailure = createAction('CHANGE_LANGUAGE_FAILURE');
export const changeLanguageSuccess = createAction('CHANGE_LANGUAGE_SUCCESS');

export const changeLanguage = lang => async (dispatch) => {
    dispatch(changeLanguageRequest());
    try {
        await apiClient.changeLanguage(lang);
        dispatch(changeLanguageSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(changeLanguageFailure());
    }
};

export const getLanguageRequest = createAction('GET_LANGUAGE_REQUEST');
export const getLanguageFailure = createAction('GET_LANGUAGE_FAILURE');
export const getLanguageSuccess = createAction('GET_LANGUAGE_SUCCESS');

export const getLanguage = () => async (dispatch) => {
    dispatch(getLanguageRequest());
    try {
        const language = await apiClient.getCurrentLanguage();
        dispatch(getLanguageSuccess(language));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLanguageFailure());
    }
};

export const getDhcpStatusRequest = createAction('GET_DHCP_STATUS_REQUEST');
export const getDhcpStatusSuccess = createAction('GET_DHCP_STATUS_SUCCESS');
export const getDhcpStatusFailure = createAction('GET_DHCP_STATUS_FAILURE');

export const getDhcpStatus = () => async (dispatch) => {
    dispatch(getDhcpStatusRequest());
    try {
        const status = await apiClient.getDhcpStatus();
        dispatch(getDhcpStatusSuccess(status));
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

export const findActiveDhcp = name => async (dispatch) => {
    dispatch(findActiveDhcpRequest());
    try {
        const activeDhcp = await apiClient.findActiveDhcp(name);
        dispatch(findActiveDhcpSuccess(activeDhcp));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(findActiveDhcpFailure());
    }
};

export const setDhcpConfigRequest = createAction('SET_DHCP_CONFIG_REQUEST');
export const setDhcpConfigSuccess = createAction('SET_DHCP_CONFIG_SUCCESS');
export const setDhcpConfigFailure = createAction('SET_DHCP_CONFIG_FAILURE');

export const setDhcpConfig = values => async (dispatch, getState) => {
    const { config } = getState().dhcp;
    const updatedConfig = { ...config, ...values };
    dispatch(setDhcpConfigRequest());
    dispatch(findActiveDhcp(values.interface_name));
    try {
        await apiClient.setDhcpConfig(updatedConfig);
        dispatch(setDhcpConfigSuccess(updatedConfig));
        dispatch(addSuccessToast('dhcp_config_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setDhcpConfigFailure());
    }
};

export const toggleDhcpRequest = createAction('TOGGLE_DHCP_REQUEST');
export const toggleDhcpFailure = createAction('TOGGLE_DHCP_FAILURE');
export const toggleDhcpSuccess = createAction('TOGGLE_DHCP_SUCCESS');

export const toggleDhcp = values => async (dispatch) => {
    dispatch(toggleDhcpRequest());
    let config = { ...values, enabled: false };
    let successMessage = 'disabled_dhcp';

    if (!values.enabled) {
        config = { ...values, enabled: true };
        successMessage = 'enabled_dhcp';
        dispatch(findActiveDhcp(values.interface_name));
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

export const toggleLeaseModal = createAction('TOGGLE_LEASE_MODAL');

export const addStaticLeaseRequest = createAction('ADD_STATIC_LEASE_REQUEST');
export const addStaticLeaseFailure = createAction('ADD_STATIC_LEASE_FAILURE');
export const addStaticLeaseSuccess = createAction('ADD_STATIC_LEASE_SUCCESS');

export const addStaticLease = config => async (dispatch) => {
    dispatch(addStaticLeaseRequest());
    try {
        const name = config.hostname || config.ip;
        await apiClient.addStaticLease(config);
        dispatch(addStaticLeaseSuccess(config));
        dispatch(addSuccessToast(t('dhcp_lease_added', { key: name })));
        dispatch(toggleLeaseModal());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(addStaticLeaseFailure());
    }
};

export const removeStaticLeaseRequest = createAction('REMOVE_STATIC_LEASE_REQUEST');
export const removeStaticLeaseFailure = createAction('REMOVE_STATIC_LEASE_FAILURE');
export const removeStaticLeaseSuccess = createAction('REMOVE_STATIC_LEASE_SUCCESS');

export const removeStaticLease = config => async (dispatch) => {
    dispatch(removeStaticLeaseRequest());
    try {
        const name = config.hostname || config.ip;
        await apiClient.removeStaticLease(config);
        dispatch(removeStaticLeaseSuccess(config));
        dispatch(addSuccessToast(t('dhcp_lease_deleted', { key: name })));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(removeStaticLeaseFailure());
    }
};
