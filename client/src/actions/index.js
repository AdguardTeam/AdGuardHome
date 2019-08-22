import { createAction } from 'redux-actions';
import { t } from 'i18next';
import { showLoading, hideLoading } from 'react-redux-loading-bar';
import axios from 'axios';

import versionCompare from '../helpers/versionCompare';
import { normalizeFilteringStatus, normalizeLogs, normalizeTextarea, sortClients } from '../helpers/helpers';
import { SETTINGS_NAMES, CHECK_TIMEOUT } from '../helpers/constants';
import { getTlsStatus } from './encryption';
import Api from '../api/Api';

const apiClient = new Api();

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
            case SETTINGS_NAMES.filtering:
                if (status) {
                    successMessage = 'disabled_filtering_toast';
                    await apiClient.disableFiltering();
                } else {
                    successMessage = 'enabled_filtering_toast';
                    await apiClient.enableFiltering();
                }
                dispatch(toggleSettingStatus({ settingKey }));
                break;
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
        const filteringStatus = await apiClient.getFilteringStatus();
        const safebrowsingStatus = await apiClient.getSafebrowsingStatus();
        const parentalStatus = await apiClient.getParentalStatus();
        const safesearchStatus = await apiClient.getSafesearchStatus();
        const {
            filtering,
            safebrowsing,
            parental,
            safesearch,
        } = settingsList;
        const newSettingsList = {
            filtering: { ...filtering, enabled: filteringStatus.enabled },
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

export const getFilteringRequest = createAction('GET_FILTERING_REQUEST');
export const getFilteringFailure = createAction('GET_FILTERING_FAILURE');
export const getFilteringSuccess = createAction('GET_FILTERING_SUCCESS');

export const getFiltering = () => async (dispatch) => {
    dispatch(getFilteringRequest());
    try {
        const filteringStatus = await apiClient.getFilteringStatus();
        dispatch(getFilteringSuccess(filteringStatus.enabled));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getFilteringFailure());
    }
};

export const toggleProtectionRequest = createAction('TOGGLE_PROTECTION_REQUEST');
export const toggleProtectionFailure = createAction('TOGGLE_PROTECTION_FAILURE');
export const toggleProtectionSuccess = createAction('TOGGLE_PROTECTION_SUCCESS');

export const toggleProtection = status => async (dispatch) => {
    dispatch(toggleProtectionRequest());
    let successMessage = '';

    try {
        if (status) {
            successMessage = 'disabled_protection';
            await apiClient.disableGlobalProtection();
        } else {
            successMessage = 'enabled_protection';
            await apiClient.enableGlobalProtection();
        }

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

            if (data && versionCompare(currentVersion, data.new_version) === -1) {
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
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(initSettingsFailure());
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

export const getLogsRequest = createAction('GET_LOGS_REQUEST');
export const getLogsFailure = createAction('GET_LOGS_FAILURE');
export const getLogsSuccess = createAction('GET_LOGS_SUCCESS');

export const getLogs = () => async (dispatch, getState) => {
    dispatch(getLogsRequest());
    const timer = setInterval(async () => {
        const state = getState();
        if (state.dashboard.isCoreRunning) {
            clearInterval(timer);
            try {
                const logs = normalizeLogs(await apiClient.getQueryLog());
                dispatch(getLogsSuccess(logs));
            } catch (error) {
                dispatch(addErrorToast({ error }));
                dispatch(getLogsFailure(error));
            }
        }
    }, 100);
};

export const toggleLogStatusRequest = createAction('TOGGLE_LOGS_REQUEST');
export const toggleLogStatusFailure = createAction('TOGGLE_LOGS_FAILURE');
export const toggleLogStatusSuccess = createAction('TOGGLE_LOGS_SUCCESS');

export const toggleLogStatus = queryLogEnabled => async (dispatch) => {
    dispatch(toggleLogStatusRequest());
    let toggleMethod;
    let successMessage;
    if (queryLogEnabled) {
        toggleMethod = apiClient.disableQueryLog.bind(apiClient);
        successMessage = 'query_log_disabled_toast';
    } else {
        toggleMethod = apiClient.enableQueryLog.bind(apiClient);
        successMessage = 'query_log_enabled_toast';
    }
    try {
        await toggleMethod();
        dispatch(addSuccessToast(successMessage));
        dispatch(toggleLogStatusSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleLogStatusFailure());
    }
};

export const setRulesRequest = createAction('SET_RULES_REQUEST');
export const setRulesFailure = createAction('SET_RULES_FAILURE');
export const setRulesSuccess = createAction('SET_RULES_SUCCESS');

export const setRules = rules => async (dispatch) => {
    dispatch(setRulesRequest());
    try {
        const replacedLineEndings = rules
            .replace(/^\n/g, '')
            .replace(/\n\s*\n/g, '\n');
        await apiClient.setRules(replacedLineEndings);
        dispatch(addSuccessToast('updated_custom_filtering_toast'));
        dispatch(setRulesSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setRulesFailure());
    }
};

export const getFilteringStatusRequest = createAction('GET_FILTERING_STATUS_REQUEST');
export const getFilteringStatusFailure = createAction('GET_FILTERING_STATUS_FAILURE');
export const getFilteringStatusSuccess = createAction('GET_FILTERING_STATUS_SUCCESS');

export const getFilteringStatus = () => async (dispatch) => {
    dispatch(getFilteringStatusRequest());
    try {
        const status = await apiClient.getFilteringStatus();
        dispatch(getFilteringStatusSuccess({ status: normalizeFilteringStatus(status) }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getFilteringStatusFailure());
    }
};

export const toggleFilterRequest = createAction('FILTER_ENABLE_REQUEST');
export const toggleFilterFailure = createAction('FILTER_ENABLE_FAILURE');
export const toggleFilterSuccess = createAction('FILTER_ENABLE_SUCCESS');

export const toggleFilterStatus = url => async (dispatch, getState) => {
    dispatch(toggleFilterRequest());
    const state = getState();
    const { filters } = state.filtering;
    const filter = filters.filter(filter => filter.url === url)[0];
    const { enabled } = filter;
    let toggleStatusMethod;
    if (enabled) {
        toggleStatusMethod = apiClient.disableFilter.bind(apiClient);
    } else {
        toggleStatusMethod = apiClient.enableFilter.bind(apiClient);
    }
    try {
        await toggleStatusMethod(url);
        dispatch(toggleFilterSuccess(url));
        dispatch(getFilteringStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleFilterFailure());
    }
};

export const refreshFiltersRequest = createAction('FILTERING_REFRESH_REQUEST');
export const refreshFiltersFailure = createAction('FILTERING_REFRESH_FAILURE');
export const refreshFiltersSuccess = createAction('FILTERING_REFRESH_SUCCESS');

export const refreshFilters = () => async (dispatch) => {
    dispatch(refreshFiltersRequest());
    dispatch(showLoading());
    try {
        const refreshText = await apiClient.refreshFilters();
        dispatch(refreshFiltersSuccess());

        if (refreshText.includes('OK')) {
            if (refreshText.includes('OK 0')) {
                dispatch(addSuccessToast('all_filters_up_to_date_toast'));
            } else {
                dispatch(addSuccessToast(refreshText.replace(/OK /g, '')));
            }
        } else {
            dispatch(addErrorToast({ error: refreshText }));
        }

        dispatch(getFilteringStatus());
        dispatch(hideLoading());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(refreshFiltersFailure());
        dispatch(hideLoading());
    }
};

export const handleRulesChange = createAction('HANDLE_RULES_CHANGE');

export const addFilterRequest = createAction('ADD_FILTER_REQUEST');
export const addFilterFailure = createAction('ADD_FILTER_FAILURE');
export const addFilterSuccess = createAction('ADD_FILTER_SUCCESS');

export const addFilter = (url, name) => async (dispatch) => {
    dispatch(addFilterRequest());
    try {
        await apiClient.addFilter(url, name);
        dispatch(addFilterSuccess(url));
        dispatch(getFilteringStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(addFilterFailure());
    }
};


export const removeFilterRequest = createAction('ADD_FILTER_REQUEST');
export const removeFilterFailure = createAction('ADD_FILTER_FAILURE');
export const removeFilterSuccess = createAction('ADD_FILTER_SUCCESS');

export const removeFilter = url => async (dispatch) => {
    dispatch(removeFilterRequest());
    try {
        await apiClient.removeFilter(url);
        dispatch(removeFilterSuccess(url));
        dispatch(getFilteringStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(removeFilterFailure());
    }
};

export const toggleFilteringModal = createAction('FILTERING_MODAL_TOGGLE');

export const downloadQueryLogRequest = createAction('DOWNLOAD_QUERY_LOG_REQUEST');
export const downloadQueryLogFailure = createAction('DOWNLOAD_QUERY_LOG_FAILURE');
export const downloadQueryLogSuccess = createAction('DOWNLOAD_QUERY_LOG_SUCCESS');

export const downloadQueryLog = () => async (dispatch) => {
    let data;
    dispatch(downloadQueryLogRequest());
    try {
        data = await apiClient.downloadQueryLog();
        dispatch(downloadQueryLogSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(downloadQueryLogFailure());
    }
    return data;
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
        dispatch(setUpstreamSuccess());
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
