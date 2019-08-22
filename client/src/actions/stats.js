import { createAction } from 'redux-actions';

import Api from '../api/Api';
import { addErrorToast, addSuccessToast } from './index';
import { normalizeTopStats } from '../helpers/helpers';

const apiClient = new Api();

export const getStatsConfigRequest = createAction('GET_LOGS_CONFIG_REQUEST');
export const getStatsConfigFailure = createAction('GET_LOGS_CONFIG_FAILURE');
export const getStatsConfigSuccess = createAction('GET_LOGS_CONFIG_SUCCESS');

export const getStatsConfig = () => async (dispatch) => {
    dispatch(getStatsConfigRequest());
    try {
        const data = await apiClient.getStatsInfo();
        dispatch(getStatsConfigSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getStatsConfigFailure());
    }
};

export const setStatsConfigRequest = createAction('SET_STATS_CONFIG_REQUEST');
export const setStatsConfigFailure = createAction('SET_STATS_CONFIG_FAILURE');
export const setStatsConfigSuccess = createAction('SET_STATS_CONFIG_SUCCESS');

export const setStatsConfig = config => async (dispatch) => {
    dispatch(setStatsConfigRequest());
    try {
        await apiClient.setStatsConfig(config);
        dispatch(addSuccessToast('config_successfully_saved'));
        dispatch(setStatsConfigSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setStatsConfigFailure());
    }
};

export const getStatsRequest = createAction('GET_STATS_REQUEST');
export const getStatsFailure = createAction('GET_STATS_FAILURE');
export const getStatsSuccess = createAction('GET_STATS_SUCCESS');

export const getStats = () => async (dispatch) => {
    dispatch(getStatsRequest());
    try {
        const stats = await apiClient.getStats();

        const normalizedStats = {
            ...stats,
            top_blocked_domains: normalizeTopStats(stats.top_blocked_domains),
            top_clients: normalizeTopStats(stats.top_clients),
            top_queried_domains: normalizeTopStats(stats.top_queried_domains),
        };

        dispatch(getStatsSuccess(normalizedStats));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getStatsFailure());
    }
};
