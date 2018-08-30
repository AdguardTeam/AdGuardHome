import { combineReducers } from 'redux';
import { handleActions } from 'redux-actions';

import * as actions from '../actions';

const settings = handleActions({
    [actions.initSettingsRequest]: state => ({ ...state, processing: true }),
    [actions.initSettingsFailure]: state => ({ ...state, processing: false }),
    [actions.initSettingsSuccess]: (state, { payload }) => {
        const { settingsList } = payload;
        const newState = { ...state, settingsList, processing: false };
        return newState;
    },
    [actions.toggleSettingStatus]: (state, { payload }) => {
        const { settingsList } = state;
        const { settingKey } = payload;

        const setting = settingsList[settingKey];

        const newSetting = { ...setting, enabled: !setting.enabled };
        const newSettingsList = { ...settingsList, [settingKey]: newSetting };
        return { ...state, settingsList: newSettingsList };
    },
    [actions.setUpstreamRequest]: state => ({ ...state, processingUpstream: true }),
    [actions.setUpstreamFailure]: state => ({ ...state, processingUpstream: false }),
    [actions.setUpstreamSuccess]: state => ({ ...state, processingUpstream: false }),
    [actions.handleUpstreamChange]: (state, { payload }) => {
        const { upstream } = payload;
        return { ...state, upstream };
    },
}, {
    processing: true,
    processingUpstream: true,
    upstream: '',
});

const dashboard = handleActions({
    [actions.dnsStatusRequest]: state => ({ ...state, processing: true }),
    [actions.dnsStatusFailure]: state => ({ ...state, processing: false }),
    [actions.dnsStatusSuccess]: (state, { payload }) => {
        const {
            version,
            running,
            dns_port: dnsPort,
            dns_address: dnsAddress,
            querylog_enabled: queryLogEnabled,
        } = payload;
        const newState = {
            ...state,
            isCoreRunning: running,
            processing: false,
            dnsVersion: version,
            dnsPort,
            dnsAddress,
            queryLogEnabled,
        };
        return newState;
    },

    [actions.enableDnsRequest]: state => ({ ...state, processing: true }),
    [actions.enableDnsFailure]: state => ({ ...state, processing: false }),
    [actions.enableDnsSuccess]: (state) => {
        const newState = { ...state, isCoreRunning: !state.isCoreRunning, processing: false };
        return newState;
    },

    [actions.disableDnsRequest]: state => ({ ...state, processing: true }),
    [actions.disableDnsFailure]: state => ({ ...state, processing: false }),
    [actions.disableDnsSuccess]: (state) => {
        const newState = { ...state, isCoreRunning: !state.isCoreRunning, processing: false };
        return newState;
    },

    [actions.getStatsRequest]: state => ({ ...state, processingStats: true }),
    [actions.getStatsFailure]: state => ({ ...state, processingStats: false }),
    [actions.getStatsSuccess]: (state, { payload }) => {
        const newState = { ...state, stats: payload, processingStats: false };
        return newState;
    },

    [actions.getTopStatsRequest]: state => ({ ...state, processingTopStats: true }),
    [actions.getTopStatsFailure]: state => ({ ...state, processingTopStats: false }),
    [actions.getTopStatsSuccess]: (state, { payload }) => {
        const newState = { ...state, topStats: payload, processingTopStats: false };
        return newState;
    },

    [actions.getStatsHistoryRequest]: state => ({ ...state, processingStatsHistory: true }),
    [actions.getStatsHistoryFailure]: state => ({ ...state, processingStatsHistory: false }),
    [actions.getStatsHistorySuccess]: (state, { payload }) => {
        const newState = { ...state, statsHistory: payload, processingStatsHistory: false };
        return newState;
    },

    [actions.toggleLogStatusRequest]: state => ({ ...state, logStatusProcessing: true }),
    [actions.toggleLogStatusFailure]: state => ({ ...state, logStatusProcessing: false }),
    [actions.toggleLogStatusSuccess]: (state) => {
        const { queryLogEnabled } = state;
        return ({ ...state, queryLogEnabled: !queryLogEnabled, logStatusProcessing: false });
    },
}, {
    processing: true,
    isCoreRunning: false,
    processingTopStats: true,
    processingStats: true,
    logStatusProcessing: false,
});

const queryLogs = handleActions({
    [actions.getLogsRequest]: state => ({ ...state, getLogsProcessing: true }),
    [actions.getLogsFailure]: state => ({ ...state, getLogsProcessing: false }),
    [actions.getLogsSuccess]: (state, { payload }) => {
        const newState = { ...state, logs: payload, getLogsProcessing: false };
        return newState;
    },
    [actions.downloadQueryLogRequest]: state => ({ ...state, logsDownloading: true }),
    [actions.downloadQueryLogFailure]: state => ({ ...state, logsDownloading: false }),
    [actions.downloadQueryLogSuccess]: state => ({ ...state, logsDownloading: false }),
}, { getLogsProcessing: false, logsDownloading: false });

const filtering = handleActions({
    [actions.setRulesRequest]: state => ({ ...state, processingRules: true }),
    [actions.setRulesFailure]: state => ({ ...state, processingRules: false }),
    [actions.setRulesSuccess]: state => ({ ...state, processingRules: false }),

    [actions.handleRulesChange]: (state, { payload }) => {
        const { userRules } = payload;
        return { ...state, userRules };
    },

    [actions.getFilteringStatusRequest]: state => ({ ...state, processingFilters: true }),
    [actions.getFilteringStatusFailure]: state => ({ ...state, processingFilters: false }),
    [actions.getFilteringStatusSuccess]: (state, { payload }) => {
        const { status } = payload;
        const { filters, userRules } = status;
        const newState = {
            ...state, filters, userRules, processingFilters: false,
        };
        return newState;
    },

    [actions.addFilterRequest]: state =>
        ({ ...state, processingAddFilter: true, isFilterAdded: false }),
    [actions.addFilterFailure]: (state) => {
        const newState = { ...state, processingAddFilter: false, isFilterAdded: false };
        return newState;
    },
    [actions.addFilterSuccess]: state =>
        ({ ...state, processingAddFilter: false, isFilterAdded: true }),

    [actions.toggleFilteringModal]: (state) => {
        const newState = {
            ...state,
            isFilteringModalOpen: !state.isFilteringModalOpen,
            isFilterAdded: false,
        };
        return newState;
    },

    [actions.toggleFilterRequest]: state => ({ ...state, processingFilters: true }),
    [actions.toggleFilterFailure]: state => ({ ...state, processingFilters: false }),
    [actions.toggleFilterSuccess]: state => ({ ...state, processingFilters: false }),

    [actions.refreshFiltersRequest]: state => ({ ...state, processingRefreshFilters: true }),
    [actions.refreshFiltersFailure]: state => ({ ...state, processingRefreshFilters: false }),
    [actions.refreshFiltersSuccess]: state => ({ ...state, processingRefreshFilters: false }),
}, {
    isFilteringModalOpen: false,
    processingFilters: false,
    processingRules: false,
    filters: [],
    userRules: '',
});

export default combineReducers({
    settings,
    dashboard,
    queryLogs,
    filtering,
});
