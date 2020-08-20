import { handleActions } from 'redux-actions';
import * as actions from '../actions';
import { areEqualVersions } from '../helpers/version';
import { STANDARD_DNS_PORT, STANDARD_WEB_PORT } from '../helpers/constants';

const dashboard = handleActions(
    {
        [actions.setDnsRunningStatus]: (state, { payload }) => (
            {
                ...state,
                isCoreRunning: payload,
            }
        ),
        [actions.dnsStatusRequest]: (state) => ({
            ...state,
            processing: true,
        }),
        [actions.dnsStatusFailure]: (state) => ({
            ...state,
            processing: false,
        }),
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

        [actions.getVersionRequest]: (state) => ({
            ...state,
            processingVersion: true,
        }),
        [actions.getVersionFailure]: (state) => ({
            ...state,
            processingVersion: false,
        }),
        [actions.getVersionSuccess]: (state, { payload }) => {
            const currentVersion = state.dnsVersion === 'undefined' ? 0 : state.dnsVersion;

            if (!payload.disabled && !areEqualVersions(currentVersion, payload.new_version)) {
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
                    checkUpdateFlag: !payload.disabled,
                };
                return newState;
            }

            return {
                ...state,
                processingVersion: false,
                checkUpdateFlag: !payload.disabled,
            };
        },

        [actions.getUpdateRequest]: (state) => ({
            ...state,
            processingUpdate: true,
        }),
        [actions.getUpdateFailure]: (state) => ({
            ...state,
            processingUpdate: false,
        }),
        [actions.getUpdateSuccess]: (state) => {
            const newState = {
                ...state,
                processingUpdate: false,
            };
            return newState;
        },

        [actions.toggleProtectionRequest]: (state) => ({
            ...state,
            processingProtection: true,
        }),
        [actions.toggleProtectionFailure]: (state) => ({
            ...state,
            processingProtection: false,
        }),
        [actions.toggleProtectionSuccess]: (state) => {
            const newState = {
                ...state,
                protectionEnabled: !state.protectionEnabled,
                processingProtection: false,
            };
            return newState;
        },

        [actions.getLanguageSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                language: payload,
            };
            return newState;
        },

        [actions.getClientsRequest]: (state) => ({
            ...state,
            processingClients: true,
        }),
        [actions.getClientsFailure]: (state) => ({
            ...state,
            processingClients: false,
        }),
        [actions.getClientsSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                ...payload,
                processingClients: false,
            };
            return newState;
        },

        [actions.getProfileRequest]: (state) => ({
            ...state,
            processingProfile: true,
        }),
        [actions.getProfileFailure]: (state) => ({
            ...state,
            processingProfile: false,
        }),
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
        httpPort: STANDARD_WEB_PORT,
        dnsPort: STANDARD_DNS_PORT,
        dnsAddresses: [],
        dnsVersion: '',
        clients: [],
        autoClients: [],
        supportedTags: [],
        name: '',
        checkUpdateFlag: false,
    },
);

export default dashboard;
