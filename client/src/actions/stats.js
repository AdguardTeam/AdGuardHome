import { createAction } from 'redux-actions';
import Api from '../api/Api';
import { addErrorToast, addSuccessToast } from './index';

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
