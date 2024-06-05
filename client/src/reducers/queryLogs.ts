import { handleActions } from 'redux-actions';

import * as actions from '../actions/queryLogs';
import { DEFAULT_LOGS_FILTER, DAY, QUERY_LOG_INTERVALS_DAYS, HOUR } from '../helpers/constants';

const queryLogs = handleActions(
    {
        [actions.setFilteredLogsRequest.toString()]: (state: any) => ({
            ...state,
            processingGetLogs: true,
        }),
        [actions.setFilteredLogsFailure.toString()]: (state: any) => ({
            ...state,
            processingGetLogs: false,
        }),
        [actions.toggleDetailedLogs.toString()]: (state, { payload }: any) => ({
            ...state,
            isDetailed: payload,
        }),

        [actions.setFilteredLogsSuccess.toString()]: (state: any, { payload }: any) => {
            const { logs, oldest, filter } = payload;

            const isFiltered = filter && Object.keys(filter).some((key) => filter[key]);

            return {
                ...state,
                oldest,
                filter,
                isFiltered,
                logs,
                isEntireLog: logs.length < 1,
                processingGetLogs: false,
            };
        },

        [actions.setLogsFilterRequest.toString()]: (state, { payload }: any) => ({
            ...state,
            filter: payload,
        }),

        [actions.getLogsRequest.toString()]: (state: any) => ({
            ...state,
            processingGetLogs: true,
        }),
        [actions.getLogsFailure.toString()]: (state: any) => ({
            ...state,
            processingGetLogs: false,
        }),
        [actions.getLogsSuccess.toString()]: (state: any, { payload }: any) => {
            const { logs, oldest, older_than } = payload;

            return {
                ...state,
                oldest,
                logs: older_than ? [...state.logs, ...logs] : logs,
                isEntireLog: logs.length < 1,
                processingGetLogs: false,
            };
        },

        [actions.clearLogsRequest.toString()]: (state: any) => ({
            ...state,
            processingClear: true,
        }),
        [actions.clearLogsFailure.toString()]: (state: any) => ({
            ...state,
            processingClear: false,
        }),
        [actions.clearLogsSuccess.toString()]: (state: any) => ({
            ...state,
            logs: [],
            processingClear: false,
        }),

        [actions.getLogsConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingGetConfig: true,
        }),
        [actions.getLogsConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingGetConfig: false,
        }),
        [actions.getLogsConfigSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,

            customInterval: !QUERY_LOG_INTERVALS_DAYS.includes(payload.interval) ? payload.interval / HOUR : null,

            processingGetConfig: false,
        }),

        [actions.setLogsConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingSetConfig: true,
        }),
        [actions.setLogsConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingSetConfig: false,
        }),
        [actions.setLogsConfigSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),

        [actions.getAdditionalLogsRequest.toString()]: (state: any) => ({
            ...state,
            processingAdditionalLogs: true,
            processingGetLogs: true,
        }),
        [actions.getAdditionalLogsFailure.toString()]: (state: any) => ({
            ...state,
            processingAdditionalLogs: false,
            processingGetLogs: false,
        }),
        [actions.getAdditionalLogsSuccess.toString()]: (state: any) => ({
            ...state,
            processingAdditionalLogs: false,
            processingGetLogs: false,
            isEntireLog: true,
        }),
    },
    {
        processingGetLogs: true,
        processingClear: false,
        processingGetConfig: false,
        processingSetConfig: false,
        processingAdditionalLogs: false,
        interval: DAY,
        logs: [],
        enabled: true,
        oldest: '',
        filter: DEFAULT_LOGS_FILTER,
        isFiltered: false,
        anonymize_client_ip: false,
        isDetailed: true,
        isEntireLog: false,
        customInterval: null,
    },
);

export default queryLogs;
