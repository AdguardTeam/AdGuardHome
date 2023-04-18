import { handleActions } from 'redux-actions';

import * as actions from '../actions/queryLogs';
import {
    DEFAULT_LOGS_FILTER, DAY, QUERY_LOG_INTERVALS_DAYS, HOUR,
} from '../helpers/constants';

const queryLogs = handleActions(
    {
        [actions.setFilteredLogsRequest]: (state) => ({ ...state, processingGetLogs: true }),
        [actions.setFilteredLogsFailure]: (state) => ({ ...state, processingGetLogs: false }),
        [actions.toggleDetailedLogs]: (state, { payload }) => ({
            ...state,
            isDetailed: payload,
        }),

        [actions.setFilteredLogsSuccess]: (state, { payload }) => {
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

        [actions.setLogsFilterRequest]: (state, { payload }) => ({ ...state, filter: payload }),

        [actions.getLogsRequest]: (state) => ({ ...state, processingGetLogs: true }),
        [actions.getLogsFailure]: (state) => ({ ...state, processingGetLogs: false }),
        [actions.getLogsSuccess]: (state, { payload }) => {
            const {
                logs, oldest, older_than,
            } = payload;

            return {
                ...state,
                oldest,
                logs: older_than ? [...state.logs, ...logs] : logs,
                isEntireLog: logs.length < 1,
                processingGetLogs: false,
            };
        },

        [actions.clearLogsRequest]: (state) => ({ ...state, processingClear: true }),
        [actions.clearLogsFailure]: (state) => ({ ...state, processingClear: false }),
        [actions.clearLogsSuccess]: (state) => ({
            ...state,
            logs: [],
            processingClear: false,
        }),

        [actions.getLogsConfigRequest]: (state) => ({ ...state, processingGetConfig: true }),
        [actions.getLogsConfigFailure]: (state) => ({ ...state, processingGetConfig: false }),
        [actions.getLogsConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            customInterval: !QUERY_LOG_INTERVALS_DAYS.includes(payload.interval)
                ? payload.interval / HOUR
                : null,
            processingGetConfig: false,
        }),

        [actions.setLogsConfigRequest]: (state) => ({ ...state, processingSetConfig: true }),
        [actions.setLogsConfigFailure]: (state) => ({ ...state, processingSetConfig: false }),
        [actions.setLogsConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),

        [actions.getAdditionalLogsRequest]: (state) => ({
            ...state, processingAdditionalLogs: true, processingGetLogs: true,
        }),
        [actions.getAdditionalLogsFailure]: (state) => ({
            ...state, processingAdditionalLogs: false, processingGetLogs: false,
        }),
        [actions.getAdditionalLogsSuccess]: (state) => ({
            ...state, processingAdditionalLogs: false, processingGetLogs: false, isEntireLog: true,
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
