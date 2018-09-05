import { createAction } from 'redux-actions';
import round from 'lodash/round';

import { normalizeHistory, normalizeFilteringStatus, normalizeLogs } from '../helpers/helpers';
import Api from '../api/Api';

const apiClient = new Api();

export const toggleSettingStatus = createAction('SETTING_STATUS_TOGGLE');
export const showSettingsFailure = createAction('SETTINGS_FAILURE_SHOW');

export const toggleSetting = (settingKey, status) => async (dispatch) => {
    switch (settingKey) {
        case 'filtering':
            if (status) {
                await apiClient.disableFiltering();
            } else {
                await apiClient.enableFiltering();
            }
            dispatch(toggleSettingStatus({ settingKey }));
            break;
        case 'safebrowsing':
            if (status) {
                await apiClient.disableSafebrowsing();
            } else {
                await apiClient.enableSafebrowsing();
            }
            dispatch(toggleSettingStatus({ settingKey }));
            break;
        case 'parental':
            if (status) {
                await apiClient.disableParentalControl();
            } else {
                await apiClient.enableParentalControl();
            }
            dispatch(toggleSettingStatus({ settingKey }));
            break;
        case 'safesearch':
            if (status) {
                await apiClient.disableSafesearch();
            } else {
                await apiClient.enableSafesearch();
            }
            dispatch(toggleSettingStatus({ settingKey }));
            break;
        default:
            break;
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
        console.error(error);
        dispatch(initSettingsFailure());
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
        console.error(error);
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
        console.error(error);
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
        console.error(error);
        dispatch(disableDnsFailure());
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
        console.error(error);
        dispatch(getStatsFailure());
    }
};

export const getTopStatsRequest = createAction('GET_TOP_STATS_REQUEST');
export const getTopStatsFailure = createAction('GET_TOP_STATS_FAILURE');
export const getTopStatsSuccess = createAction('GET_TOP_STATS_SUCCESS');

export const getTopStats = () => async (dispatch, getState) => {
    dispatch(getTopStatsRequest());
    try {
        const state = getState();
        const timer = setInterval(async () => {
            if (state.dashboard.isCoreRunning) {
                const stats = await apiClient.getGlobalStatsTop();
                dispatch(getTopStatsSuccess(stats));
                clearInterval(timer);
            }
        }, 100);
    } catch (error) {
        console.error(error);
        dispatch(getTopStatsFailure());
    }
};

export const getLogsRequest = createAction('GET_LOGS_REQUEST');
export const getLogsFailure = createAction('GET_LOGS_FAILURE');
export const getLogsSuccess = createAction('GET_LOGS_SUCCESS');

export const getLogs = () => async (dispatch, getState) => {
    dispatch(getLogsRequest());
    try {
        const state = getState();
        const timer = setInterval(async () => {
            if (state.dashboard.isCoreRunning) {
                const logs = normalizeLogs(await apiClient.getQueryLog());
                dispatch(getLogsSuccess(logs));
                clearInterval(timer);
            }
        }, 100);
    } catch (error) {
        console.error(error);
        dispatch(getLogsFailure());
    }
};

export const toggleLogStatusRequest = createAction('TOGGLE_LOGS_REQUEST');
export const toggleLogStatusFailure = createAction('TOGGLE_LOGS_FAILURE');
export const toggleLogStatusSuccess = createAction('TOGGLE_LOGS_SUCCESS');

export const toggleLogStatus = queryLogEnabled => async (dispatch) => {
    dispatch(toggleLogStatusRequest());
    let toggleMethod;
    if (queryLogEnabled) {
        toggleMethod = apiClient.disableQueryLog.bind(apiClient);
    } else {
        toggleMethod = apiClient.enableQueryLog.bind(apiClient);
    }
    try {
        await toggleMethod();
        dispatch(toggleLogStatusSuccess());
    } catch (error) {
        console.error(error);
        dispatch(toggleLogStatusFailure());
    }
};

export const setRulesRequest = createAction('SET_RULES_REQUEST');
export const setRulesFailure = createAction('SET_RULES_FAILURE');
export const setRulesSuccess = createAction('SET_RULES_SUCCESS');

export const setRules = rules => async (dispatch) => {
    dispatch(setRulesRequest());
    try {
        await apiClient.setRules(rules);
        dispatch(setRulesSuccess());
    } catch (error) {
        console.error(error);
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
        console.error(error);
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
        console.error(error);
        dispatch(toggleFilterFailure());
    }
};

export const refreshFiltersRequest = createAction('FILTERING_REFRESH_REQUEST');
export const refreshFiltersFailure = createAction('FILTERING_REFRESH_FAILURE');
export const refreshFiltersSuccess = createAction('FILTERING_REFRESH_SUCCESS');

export const refreshFilters = () => async (dispatch) => {
    dispatch(refreshFiltersRequest);
    try {
        await apiClient.refreshFilters();
        dispatch(refreshFiltersSuccess);
        dispatch(getFilteringStatus());
    } catch (error) {
        console.error(error);
        dispatch(refreshFiltersFailure());
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
        console.error(error);
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
        console.error(error);
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
        console.error(error);
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
        console.error(error);
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
        dispatch(setUpstreamSuccess());
    } catch (error) {
        console.error(error);
        dispatch(setUpstreamFailure());
    }
};
