import { handleActions } from 'redux-actions';

import * as actions from '../actions/queryLogs';
import { DEFAULT_LOGS_FILTER, TABLE_DEFAULT_PAGE_SIZE } from '../helpers/constants';

const queryLogs = handleActions(
    {
        [actions.setLogsPagination]: (state, { payload }) => {
            const { page, pageSize } = payload;
            const { allLogs } = state;
            const rowsStart = pageSize * page;
            const rowsEnd = (pageSize * page) + pageSize;
            const logsSlice = allLogs.slice(rowsStart, rowsEnd);
            const pages = Math.ceil(allLogs.length / pageSize);

            return {
                ...state,
                pages,
                logs: logsSlice,
            };
        },

        [actions.setLogsPage]: (state, { payload }) => ({
            ...state,
            page: payload,
        }),

        [actions.setFilteredLogsRequest]: (state) => ({ ...state, processingGetLogs: true }),
        [actions.setFilteredLogsFailure]: (state) => ({ ...state, processingGetLogs: false }),
        [actions.toggleDetailedLogs]: (state, { payload }) => ({
            ...state,
            isDetailed: payload,
        }),

        [actions.setFilteredLogsSuccess]: (state, { payload }) => {
            const { logs, oldest, filter } = payload;
            const pageSize = TABLE_DEFAULT_PAGE_SIZE;
            const page = 0;

            const pages = Math.ceil(logs.length / pageSize);
            const total = logs.length;
            const rowsStart = pageSize * page;
            const rowsEnd = rowsStart + pageSize;
            const logsSlice = logs.slice(rowsStart, rowsEnd);
            const isFiltered = Object.keys(filter).some((key) => filter[key]);

            return {
                ...state,
                oldest,
                filter,
                isFiltered,
                pages,
                total,
                logs: logsSlice,
                allLogs: logs,
                processingGetLogs: false,
            };
        },

        [actions.setLogsFilterRequest]: (state, { payload }) => {
            const { filter } = payload;

            return { ...state, filter };
        },

        [actions.getLogsRequest]: (state) => ({ ...state, processingGetLogs: true }),
        [actions.getLogsFailure]: (state) => ({ ...state, processingGetLogs: false }),
        [actions.getLogsSuccess]: (state, { payload }) => {
            const {
                logs, oldest, older_than, page, pageSize, initial,
            } = payload;
            let logsWithOffset = state.allLogs.length > 0 && !initial ? state.allLogs : logs;
            let allLogs = logs;

            if (older_than) {
                logsWithOffset = [...state.allLogs, ...logs];
                allLogs = [...state.allLogs, ...logs];
            }

            const pages = Math.ceil(logsWithOffset.length / pageSize);
            const total = logsWithOffset.length;
            const rowsStart = pageSize * page;
            const rowsEnd = (pageSize * page) + pageSize;
            const logsSlice = logsWithOffset.slice(rowsStart, rowsEnd);

            return {
                ...state,
                oldest,
                pages,
                total,
                allLogs,
                logs: logsSlice,
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
            ...state, processingAdditionalLogs: false, processingGetLogs: false,
        }),
    },
    {
        processingGetLogs: true,
        processingClear: false,
        processingGetConfig: false,
        processingSetConfig: false,
        processingAdditionalLogs: false,
        logs: [],
        interval: 1,
        allLogs: [],
        page: 0,
        pages: 0,
        total: 0,
        enabled: true,
        oldest: '',
        filter: DEFAULT_LOGS_FILTER,
        isFiltered: false,
        anonymize_client_ip: false,
        isDetailed: true,
    },
);

export default queryLogs;
