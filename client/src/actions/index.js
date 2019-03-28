import { createAction } from 'redux-actions';
import round from 'lodash/round';
import { t } from 'i18next';
import { showLoading, hideLoading } from 'react-redux-loading-bar';

import { normalizeHistory, normalizeFilteringStatus, normalizeLogs, normalizeTextarea } from '../helpers/helpers';
import { SETTINGS_NAMES } from '../helpers/constants';
import Api from '../api/Api';

const apiClient = new Api();

export const addErrorToast = createAction('ADD_ERROR_TOAST');
export const addSuccessToast = createAction('ADD_SUCCESS_TOAST');
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

export const getVersion = () => async (dispatch) => {
    dispatch(getVersionRequest());
    try {
        const newVersion = await apiClient.getGlobalVersion();
        dispatch(getVersionSuccess(newVersion));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getVersionFailure());
    }
};

export const getClientsRequest = createAction('GET_CLIENTS_REQUEST');
export const getClientsFailure = createAction('GET_CLIENTS_FAILURE');
export const getClientsSuccess = createAction('GET_CLIENTS_SUCCESS');

export const getClients = () => async (dispatch) => {
    dispatch(getClientsRequest());
    try {
        const clients = await apiClient.getGlobalClients();
        dispatch(getClientsSuccess(clients));
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
        dispatch(getClients());
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

export const getStatsRequest = createAction('GET_STATS_REQUEST');
export const getStatsFailure = createAction('GET_STATS_FAILURE');
export const getStatsSuccess = createAction('GET_STATS_SUCCESS');

export const getStats = () => async (dispatch) => {
    dispatch(getStatsRequest());
    try {
        const stats = await apiClient.getGlobalStats();

        const processedStats = {
            ...stats,
            avg_processing_time: round(stats.avg_processing_time, 2),
        };

        dispatch(getStatsSuccess(processedStats));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getStatsFailure());
    }
};

export const getTopStatsRequest = createAction('GET_TOP_STATS_REQUEST');
export const getTopStatsFailure = createAction('GET_TOP_STATS_FAILURE');
export const getTopStatsSuccess = createAction('GET_TOP_STATS_SUCCESS');

export const getTopStats = () => async (dispatch, getState) => {
    dispatch(getTopStatsRequest());
    const timer = setInterval(async () => {
        const state = getState();
        if (state.dashboard.isCoreRunning) {
            clearInterval(timer);
            try {
                const stats = await apiClient.getGlobalStatsTop();
                dispatch(getTopStatsSuccess(stats));
            } catch (error) {
                dispatch(addErrorToast({ error }));
                dispatch(getTopStatsFailure(error));
            }
        }
    }, 100);
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

export const getStatsHistoryRequest = createAction('GET_STATS_HISTORY_REQUEST');
export const getStatsHistoryFailure = createAction('GET_STATS_HISTORY_FAILURE');
export const getStatsHistorySuccess = createAction('GET_STATS_HISTORY_SUCCESS');

export const getStatsHistory = () => async (dispatch) => {
    dispatch(getStatsHistoryRequest());
    try {
        const statsHistory = await apiClient.getGlobalStatsHistory();
        const normalizedHistory = normalizeHistory(statsHistory);
        dispatch(getStatsHistorySuccess(normalizedHistory));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getStatsHistoryFailure());
    }
};

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

// TODO rewrite findActiveDhcp part
export const setDhcpConfig = values => async (dispatch, getState) => {
    const { config } = getState().dhcp;
    const updatedConfig = { ...config, ...values };
    dispatch(setDhcpConfigRequest());
    if (values.interface_name) {
        dispatch(findActiveDhcpRequest());
        try {
            const activeDhcp = await apiClient.findActiveDhcp(values.interface_name);
            dispatch(findActiveDhcpSuccess(activeDhcp));
            if (!activeDhcp.found) {
                try {
                    await apiClient.setDhcpConfig(updatedConfig);
                    dispatch(setDhcpConfigSuccess(updatedConfig));
                    dispatch(addSuccessToast('dhcp_config_saved'));
                } catch (error) {
                    dispatch(addErrorToast({ error }));
                    dispatch(setDhcpConfigFailure());
                }
            } else {
                dispatch(addErrorToast({ error: 'dhcp_found' }));
            }
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(findActiveDhcpFailure());
        }
    } else {
        try {
            await apiClient.setDhcpConfig(updatedConfig);
            dispatch(setDhcpConfigSuccess(updatedConfig));
            dispatch(addSuccessToast('dhcp_config_saved'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(setDhcpConfigFailure());
        }
    }
};

export const toggleDhcpRequest = createAction('TOGGLE_DHCP_REQUEST');
export const toggleDhcpFailure = createAction('TOGGLE_DHCP_FAILURE');
export const toggleDhcpSuccess = createAction('TOGGLE_DHCP_SUCCESS');

// TODO rewrite findActiveDhcp part
export const toggleDhcp = config => async (dispatch) => {
    dispatch(toggleDhcpRequest());

    if (config.enabled) {
        try {
            await apiClient.setDhcpConfig({ ...config, enabled: false });
            dispatch(toggleDhcpSuccess());
            dispatch(addSuccessToast('disabled_dhcp'));
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(toggleDhcpFailure());
        }
    } else {
        dispatch(findActiveDhcpRequest());
        try {
            const activeDhcp = await apiClient.findActiveDhcp(config.interface_name);
            dispatch(findActiveDhcpSuccess(activeDhcp));

            if (!activeDhcp.found) {
                try {
                    await apiClient.setDhcpConfig({ ...config, enabled: true });
                    dispatch(toggleDhcpSuccess());
                    dispatch(addSuccessToast('enabled_dhcp'));
                } catch (error) {
                    dispatch(addErrorToast({ error }));
                    dispatch(toggleDhcpFailure());
                }
            } else {
                dispatch(addErrorToast({ error: 'dhcp_found' }));
            }
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(findActiveDhcpFailure());
        }
    }
};
