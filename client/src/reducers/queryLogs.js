import { handleActions } from 'redux-actions';

import * as actions from '../actions/queryLogs';

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

        [actions.getLogsRequest]: state => ({ ...state, processingGetLogs: true }),
        [actions.getLogsFailure]: state => ({ ...state, processingGetLogs: false }),
        [actions.getLogsSuccess]: (state, { payload }) => {
            const {
                logs, lastRowTime, page, pageSize, filtered, refreshLogs,
            } = payload;
            let logsWithOffset = state.allLogs.length > 0 ? state.allLogs : logs;
            let allLogs = logs;

            if (lastRowTime) {
                logsWithOffset = [...state.allLogs, ...logs];
                allLogs = [...state.allLogs, ...logs];
            }

            if (filtered || refreshLogs) {
                logsWithOffset = logs;
                allLogs = logs;
            }

            const pages = Math.ceil(logsWithOffset.length / pageSize);
            const total = logsWithOffset.length;
            const rowsStart = pageSize * page;
            const rowsEnd = (pageSize * page) + pageSize;
            const logsSlice = logsWithOffset.slice(rowsStart, rowsEnd);

            return {
                ...state,
                pages,
                total,
                allLogs,
                logs: logsSlice,
                processingGetLogs: false,
            };
        },

        [actions.clearLogsRequest]: state => ({ ...state, processingClear: true }),
        [actions.clearLogsFailure]: state => ({ ...state, processingClear: false }),
        [actions.clearLogsSuccess]: state => ({
            ...state,
            logs: [],
            processingClear: false,
        }),

        [actions.getLogsConfigRequest]: state => ({ ...state, processingGetConfig: true }),
        [actions.getLogsConfigFailure]: state => ({ ...state, processingGetConfig: false }),
        [actions.getLogsConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingGetConfig: false,
        }),

        [actions.setLogsConfigRequest]: state => ({ ...state, processingSetConfig: true }),
        [actions.setLogsConfigFailure]: state => ({ ...state, processingSetConfig: false }),
        [actions.setLogsConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),
    },
    {
        processingGetLogs: true,
        processingClear: false,
        processingGetConfig: false,
        processingSetConfig: false,
        logs: [],
        interval: 1,
        allLogs: [],
        pages: 0,
        offset: 0,
        total: 0,
        enabled: true,
    },
);

export default queryLogs;
