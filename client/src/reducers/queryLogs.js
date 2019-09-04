import { handleActions } from 'redux-actions';

import * as actions from '../actions/queryLogs';

const queryLogs = handleActions(
    {
        [actions.getLogsRequest]: state => ({ ...state, processingGetLogs: true }),
        [actions.getLogsFailure]: state => ({ ...state, processingGetLogs: false }),
        [actions.getLogsSuccess]: (state, { payload }) => {
            const newState = { ...state, logs: payload, processingGetLogs: false };
            return newState;
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
        enabled: true,
    },
);

export default queryLogs;
