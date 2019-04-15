import { combineReducers } from 'redux';
import { handleActions } from 'redux-actions';
import { loadingBarReducer } from 'react-redux-loading-bar';
import { reducer as formReducer } from 'redux-form';
import versionCompare from '../helpers/versionCompare';

import * as actions from '../actions';
import toasts from './toasts';
import encryption from './encryption';

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

    [actions.testUpstreamRequest]: state => ({ ...state, processingTestUpstream: true }),
    [actions.testUpstreamFailure]: state => ({ ...state, processingTestUpstream: false }),
    [actions.testUpstreamSuccess]: state => ({ ...state, processingTestUpstream: false }),
}, {
    processing: true,
    processingTestUpstream: false,
    processingSetUpstream: false,
    processingDhcpStatus: false,
});

const dashboard = handleActions({
    [actions.dnsStatusRequest]: state => ({ ...state, processing: true }),
    [actions.dnsStatusFailure]: state => ({ ...state, processing: false }),
    [actions.dnsStatusSuccess]: (state, { payload }) => {
        const {
            version,
            running,
            dns_port: dnsPort,
            dns_addresses: dnsAddresses,
            querylog_enabled: queryLogEnabled,
            upstream_dns: upstreamDns,
            bootstrap_dns: bootstrapDns,
            all_servers: allServers,
            protection_enabled: protectionEnabled,
            language,
            http_port: httpPort,
        } = payload;
        const newState = {
            ...state,
            isCoreRunning: running,
            processing: false,
            dnsVersion: version,
            dnsPort,
            dnsAddresses,
            queryLogEnabled,
            upstreamDns: upstreamDns.join('\n'),
            bootstrapDns: bootstrapDns.join('\n'),
            allServers,
            protectionEnabled,
            language,
            httpPort,
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

    [actions.getVersionRequest]: state => ({ ...state, processingVersion: true }),
    [actions.getVersionFailure]: state => ({ ...state, processingVersion: false }),
    [actions.getVersionSuccess]: (state, { payload }) => {
        const currentVersion = state.dnsVersion === 'undefined' ? 0 : state.dnsVersion;

        if (versionCompare(currentVersion, payload.version) === -1) {
            const {
                version,
                announcement_url: announcementUrl,
            } = payload;

            const newState = {
                ...state,
                version,
                announcementUrl,
                isUpdateAvailable: true,
            };
            return newState;
        }

        return state;
    },

    [actions.getFilteringRequest]: state => ({ ...state, processingFiltering: true }),
    [actions.getFilteringFailure]: state => ({ ...state, processingFiltering: false }),
    [actions.getFilteringSuccess]: (state, { payload }) => {
        const newState = { ...state, isFilteringEnabled: payload, processingFiltering: false };
        return newState;
    },

    [actions.toggleProtectionRequest]: state => ({ ...state, processingProtection: true }),
    [actions.toggleProtectionFailure]: state => ({ ...state, processingProtection: false }),
    [actions.toggleProtectionSuccess]: (state) => {
        const newState = {
            ...state,
            protectionEnabled: !state.protectionEnabled,
            processingProtection: false,
        };
        return newState;
    },

    [actions.handleUpstreamChange]: (state, { payload }) => {
        const { upstreamDns } = payload;
        return { ...state, upstreamDns };
    },

    [actions.getLanguageSuccess]: (state, { payload }) => {
        const newState = { ...state, language: payload };
        return newState;
    },

    [actions.getClientsRequest]: state => ({ ...state, processingClients: true }),
    [actions.getClientsFailure]: state => ({ ...state, processingClients: false }),
    [actions.getClientsSuccess]: (state, { payload }) => {
        const newState = {
            ...state,
            clients: payload,
            processingClients: false,
        };
        return newState;
    },
}, {
    processing: true,
    isCoreRunning: false,
    processingTopStats: true,
    processingStats: true,
    logStatusProcessing: false,
    processingVersion: true,
    processingFiltering: true,
    processingClients: true,
    upstreamDns: '',
    bootstrapDns: '',
    allServers: false,
    protectionEnabled: false,
    processingProtection: false,
    httpPort: 80,
    dnsPort: 53,
    dnsAddresses: [],
    dnsVersion: '',
    clients: [],
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
    processingAddFilter: false,
    processingRefreshFilters: false,
    filters: [],
    userRules: '',
});

const dhcp = handleActions({
    [actions.getDhcpStatusRequest]: state => ({ ...state, processing: true }),
    [actions.getDhcpStatusFailure]: state => ({ ...state, processing: false }),
    [actions.getDhcpStatusSuccess]: (state, { payload }) => {
        const newState = {
            ...state,
            ...payload,
            processing: false,
        };
        return newState;
    },

    [actions.getDhcpInterfacesRequest]: state => ({ ...state, processingInterfaces: true }),
    [actions.getDhcpInterfacesFailure]: state => ({ ...state, processingInterfaces: false }),
    [actions.getDhcpInterfacesSuccess]: (state, { payload }) => {
        const newState = {
            ...state,
            interfaces: payload,
            processingInterfaces: false,
        };
        return newState;
    },

    [actions.findActiveDhcpRequest]: state => ({ ...state, processingStatus: true }),
    [actions.findActiveDhcpFailure]: state => ({ ...state, processingStatus: false }),
    [actions.findActiveDhcpSuccess]: (state, { payload }) => {
        const {
            other_server: otherServer,
            static_ip: staticIP,
        } = payload;

        const newState = {
            ...state,
            check: {
                otherServer,
                staticIP,
            },
            processingStatus: false,
        };
        return newState;
    },

    [actions.toggleDhcpRequest]: state => ({ ...state, processingDhcp: true }),
    [actions.toggleDhcpFailure]: state => ({ ...state, processingDhcp: false }),
    [actions.toggleDhcpSuccess]: (state) => {
        const { config } = state;
        const newConfig = { ...config, enabled: !config.enabled };
        const newState = {
            ...state, config: newConfig, check: null, processingDhcp: false,
        };
        return newState;
    },

    [actions.setDhcpConfigRequest]: state => ({ ...state, processingConfig: true }),
    [actions.setDhcpConfigFailure]: state => ({ ...state, processingConfig: false }),
    [actions.setDhcpConfigSuccess]: (state, { payload }) => {
        const { config } = state;
        const newConfig = { ...config, ...payload };
        const newState = { ...state, config: newConfig, processingConfig: false };
        return newState;
    },
}, {
    processing: true,
    processingStatus: false,
    processingInterfaces: false,
    processingDhcp: false,
    processingConfig: false,
    config: {
        enabled: false,
    },
    check: null,
    leases: [],
});

export default combineReducers({
    settings,
    dashboard,
    queryLogs,
    filtering,
    toasts,
    dhcp,
    encryption,
    loadingBar: loadingBarReducer,
    form: formReducer,
});
