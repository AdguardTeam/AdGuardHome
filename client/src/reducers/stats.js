import { handleActions } from 'redux-actions';

import * as actions from '../actions/stats';

const stats = handleActions({
    [actions.getStatsConfigRequest]: state => ({ ...state, getConfigProcessing: true }),
    [actions.getStatsConfigFailure]: state => ({ ...state, getConfigProcessing: false }),
    [actions.getStatsConfigSuccess]: (state, { payload }) => ({
        ...state,
        interval: payload.interval,
        getConfigProcessing: false,
    }),

    [actions.setStatsConfigRequest]: state => ({ ...state, setConfigProcessing: true }),
    [actions.setStatsConfigFailure]: state => ({ ...state, setConfigProcessing: false }),
    [actions.setStatsConfigSuccess]: (state, { payload }) => ({
        ...state,
        interval: payload.interval,
        setConfigProcessing: false,
    }),
}, {
    getConfigProcessing: false,
    setConfigProcessing: false,
    interval: 1,
});

export default stats;
