import { combineReducers } from 'redux';
import { handleActions } from 'redux-actions';
import { loadingBarReducer } from 'react-redux-loading-bar';
import { reducer as formReducer } from 'redux-form';
import { isVersionGreater } from '../helpers/helpers';

import * as actions from '../actions';
import toasts from './toasts';
import encryption from './encryption';
import clients from './clients';
import access from './access';
import rewrites from './rewrites';
import services from './services';
import stats from './stats';
import queryLogs from './queryLogs';
import dnsConfig from './dnsConfig';
import filtering from './filtering';

const settings = handleActions(
    {
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
        [actions.testUpstreamRequest]: state => ({ ...state, processingTestUpstream: true }),
        [actions.testUpstreamFailure]: state => ({ ...state, processingTestUpstream: false }),
        [actions.testUpstreamSuccess]: state => ({ ...state, processingTestUpstream: false }),
    },
    {
        processing: true,
        processingTestUpstream: false,
        processingDhcpStatus: false,
        settingsList: {},
    },
);

const dashboard = handleActions(
    {
        [actions.setDnsRunningStatus]: (state, { payload }) =>
            ({ ...state, isCoreRunning: payload }),
        [actions.dnsStatusRequest]: state => ({ ...state, processing: true }),
        [actions.dnsStatusFailure]: state => ({ ...state, processing: false }),
        [actions.dnsStatusSuccess]: (state, { payload }) => {
            const {
                version,
                dns_port: dnsPort,
                dns_addresses: dnsAddresses,
                protection_enabled: protectionEnabled,
                http_port: httpPort,
                language,
            } = payload;
            const newState = {
                ...state,
                isCoreRunning: true,
                processing: false,
                dnsVersion: version,
                dnsPort,
                dnsAddresses,
                protectionEnabled,
                language,
                httpPort,
            };
            return newState;
        },

        [actions.getVersionRequest]: state => ({ ...state, processingVersion: true }),
        [actions.getVersionFailure]: state => ({ ...state, processingVersion: false }),
        [actions.getVersionSuccess]: (state, { payload }) => {
            const currentVersion = state.dnsVersion === 'undefined' ? 0 : state.dnsVersion;

            if (payload && isVersionGreater(currentVersion, payload.new_version)) {
                const {
                    announcement_url: announcementUrl,
                    new_version: newVersion,
                    can_autoupdate: canAutoUpdate,
                } = payload;

                const newState = {
                    ...state,
                    announcementUrl,
                    newVersion,
                    canAutoUpdate,
                    isUpdateAvailable: true,
                    processingVersion: false,
                };
                return newState;
            }

            return {
                ...state,
                processingVersion: false,
            };
        },

        [actions.getUpdateRequest]: state => ({ ...state, processingUpdate: true }),
        [actions.getUpdateFailure]: state => ({ ...state, processingUpdate: false }),
        [actions.getUpdateSuccess]: (state) => {
            const newState = { ...state, processingUpdate: false };
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

        [actions.getLanguageSuccess]: (state, { payload }) => {
            const newState = { ...state, language: payload };
            return newState;
        },

        [actions.getClientsRequest]: state => ({ ...state, processingClients: true }),
        [actions.getClientsFailure]: state => ({ ...state, processingClients: false }),
        [actions.getClientsSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                ...payload,
                processingClients: false,
            };
            return newState;
        },

        [actions.getProfileRequest]: state => ({ ...state, processingProfile: true }),
        [actions.getProfileFailure]: state => ({ ...state, processingProfile: false }),
        [actions.getProfileSuccess]: (state, { payload }) => ({
            ...state,
            name: payload.name,
            processingProfile: false,
        }),
    },
    {
        processing: true,
        isCoreRunning: true,
        processingVersion: true,
        processingClients: true,
        processingUpdate: false,
        processingProfile: true,
        protectionEnabled: false,
        processingProtection: false,
        httpPort: 80,
        dnsPort: 53,
        dnsAddresses: [],
        dnsVersion: '',
        clients: [],
        autoClients: [],
        supportedTags: [],
        name: '',
    },
);

const dhcp = handleActions(
    {
        [actions.getDhcpStatusRequest]: state => ({ ...state, processing: true }),
        [actions.getDhcpStatusFailure]: state => ({ ...state, processing: false }),
        [actions.getDhcpStatusSuccess]: (state, { payload }) => {
            const { static_leases: staticLeases, ...values } = payload;

            const newState = {
                ...state,
                staticLeases,
                processing: false,
                ...values,
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
            const { other_server: otherServer, static_ip: staticIP } = payload;

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
                ...state,
                config: newConfig,
                check: null,
                processingDhcp: false,
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

        [actions.resetDhcpRequest]: state => ({ ...state, processingReset: true }),
        [actions.resetDhcpFailure]: state => ({ ...state, processingReset: false }),
        [actions.resetDhcpSuccess]: state => ({
            ...state,
            processingReset: false,
            config: {
                enabled: false,
            },
        }),

        [actions.toggleLeaseModal]: (state) => {
            const newState = {
                ...state,
                isModalOpen: !state.isModalOpen,
            };
            return newState;
        },

        [actions.addStaticLeaseRequest]: state => ({ ...state, processingAdding: true }),
        [actions.addStaticLeaseFailure]: state => ({ ...state, processingAdding: false }),
        [actions.addStaticLeaseSuccess]: (state, { payload }) => {
            const { ip, mac, hostname } = payload;
            const newLease = {
                ip,
                mac,
                hostname: hostname || '',
            };
            const leases = [...state.staticLeases, newLease];
            const newState = {
                ...state,
                staticLeases: leases,
                processingAdding: false,
            };
            return newState;
        },

        [actions.removeStaticLeaseRequest]: state => ({ ...state, processingDeleting: true }),
        [actions.removeStaticLeaseFailure]: state => ({ ...state, processingDeleting: false }),
        [actions.removeStaticLeaseSuccess]: (state, { payload }) => {
            const leaseToRemove = payload.ip;
            const leases = state.staticLeases.filter(item => item.ip !== leaseToRemove);
            const newState = {
                ...state,
                staticLeases: leases,
                processingDeleting: false,
            };
            return newState;
        },
    },
    {
        processing: true,
        processingStatus: false,
        processingInterfaces: false,
        processingDhcp: false,
        processingConfig: false,
        processingAdding: false,
        processingDeleting: false,
        config: {
            enabled: false,
        },
        check: null,
        leases: [],
        staticLeases: [],
        isModalOpen: false,
    },
);

export default combineReducers({
    settings,
    dashboard,
    queryLogs,
    filtering,
    toasts,
    dhcp,
    encryption,
    clients,
    access,
    rewrites,
    services,
    stats,
    dnsConfig,
    loadingBar: loadingBarReducer,
    form: formReducer,
});
