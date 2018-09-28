import { createAction } from 'redux-actions';
import round from 'lodash/round';
import { showLoading, hideLoading } from 'react-redux-loading-bar';

import { normalizeHistory, normalizeFilteringStatus, normalizeLogs } from '../helpers/helpers';
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
        // TODO move setting keys to constants
        switch (settingKey) {
            case 'filtering':
                if (status) {
                    successMessage = 'Disabled filtering';
                    await apiClient.disableFiltering();
                } else {
                    successMessage = 'Enabled filtering';
                    await apiClient.enableFiltering();
                }
                dispatch(toggleSettingStatus({ settingKey }));
                break;
            case 'safebrowsing':
                if (status) {
                    successMessage = 'Disabled safebrowsing';
                    await apiClient.disableSafebrowsing();
                } else {
                    successMessage = 'Enabled safebrowsing';
                    await apiClient.enableSafebrowsing();
                }
                dispatch(toggleSettingStatus({ settingKey }));
                break;
            case 'parental':
                if (status) {
                    successMessage = 'Disabled parental control';
                    await apiClient.disableParentalControl();
                } else {
                    successMessage = 'Enabled parental control';
                    await apiClient.enableParentalControl();
                }
                dispatch(toggleSettingStatus({ settingKey }));
                break;
            case 'safesearch':
                if (status) {
                    successMessage = 'Disabled safe search';
                    await apiClient.disableSafesearch();
                } else {
                    successMessage = 'Enabled safe search';
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

export const toggleFilteringRequest = createAction('TOGGLE_FILTERING_REQUEST');
export const toggleFilteringFailure = createAction('TOGGLE_FILTERING_FAILURE');
export const toggleFilteringSuccess = createAction('TOGGLE_FILTERING_SUCCESS');

export const toggleFiltering = status => async (dispatch) => {
    dispatch(toggleFilteringRequest());
    let successMessage = '';

    try {
        if (status) {
            successMessage = 'Disabled filtering';
            await apiClient.disableFiltering();
        } else {
            successMessage = 'Enabled filtering';
            await apiClient.enableFiltering();
        }

        dispatch(addSuccessToast(successMessage));
        dispatch(toggleFilteringSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleFilteringFailure());
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
        successMessage = 'disabled';
    } else {
        toggleMethod = apiClient.enableQueryLog.bind(apiClient);
        successMessage = 'enabled';
    }
    try {
        await toggleMethod();
        dispatch(addSuccessToast(`Query log ${successMessage}`));
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
        dispatch(addSuccessToast('Custom rules saved'));
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
    dispatch(refreshFiltersRequest);
    dispatch(showLoading());
    try {
        const refreshText = await apiClient.refreshFilters();
        dispatch(refreshFiltersSuccess);

        if (refreshText.includes('OK')) {
            if (refreshText.includes('OK 0')) {
                dispatch(addSuccessToast('All filters are already up-to-date'));
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

export const addFilter = url => async (dispatch) => {
    dispatch(addFilterRequest());
    try {
        await apiClient.addFilter(url);
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

// TODO create some common flasher with all server errors
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

export const setUpstream = url => async (dispatch) => {
    dispatch(setUpstreamRequest());
    try {
        await apiClient.setUpstream(url);
        dispatch(addSuccessToast('Upstream DNS servers saved'));
        dispatch(setUpstreamSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setUpstreamFailure());
    }
};

export const testUpstreamRequest = createAction('TEST_UPSTREAM_REQUEST');
export const testUpstreamFailure = createAction('TEST_UPSTREAM_FAILURE');
export const testUpstreamSuccess = createAction('TEST_UPSTREAM_SUCCESS');

export const testUpstream = servers => async (dispatch) => {
    dispatch(testUpstreamRequest());
    try {
        const upstreamResponse = await apiClient.testUpstream(servers);

        const testMessages = Object.keys(upstreamResponse).map((key) => {
            const message = upstreamResponse[key];
            if (message !== 'OK') {
                dispatch(addErrorToast({ error: `Server "${key}": could not be used, please check that you've written it correctly` }));
            }
            return message;
        });

        if (testMessages.every(message => message === 'OK')) {
            dispatch(addSuccessToast('Specified DNS servers are working correctly'));
        }

        dispatch(testUpstreamSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(testUpstreamFailure());
    }
};
