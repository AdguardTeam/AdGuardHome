import { createAction } from 'redux-actions';

import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './index';
import { normalizeLogs } from '../helpers/helpers';

const getLogsWithParams = async (config) => {
    const { older_than, filter, ...values } = config;
    const rawLogs = await apiClient.getQueryLog({ ...filter, older_than });
    const { data, oldest } = rawLogs;
    const logs = normalizeLogs(data);

    return {
        logs, oldest, older_than, filter, ...values,
    };
};

export const setLogsPagination = createAction('LOGS_PAGINATION');
export const setLogsPage = createAction('SET_LOG_PAGE');

export const getLogsRequest = createAction('GET_LOGS_REQUEST');
export const getLogsFailure = createAction('GET_LOGS_FAILURE');
export const getLogsSuccess = createAction('GET_LOGS_SUCCESS');

export const getLogs = config => async (dispatch) => {
    dispatch(getLogsRequest());
    try {
        const logs = await getLogsWithParams(config);
        dispatch(getLogsSuccess(logs));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsFailure(error));
    }
};

export const setLogsFilterRequest = createAction('SET_LOGS_FILTER_REQUEST');
export const setLogsFilterFailure = createAction('SET_LOGS_FILTER_FAILURE');
export const setLogsFilterSuccess = createAction('SET_LOGS_FILTER_SUCCESS');

export const setLogsFilter = filter => async (dispatch) => {
    dispatch(setLogsFilterRequest());
    try {
        const logs = await getLogsWithParams({ older_than: '', filter });
        dispatch(setLogsFilterSuccess(logs));
        dispatch(setLogsPage(0));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setLogsFilterFailure(error));
    }
};

export const clearLogsRequest = createAction('CLEAR_LOGS_REQUEST');
export const clearLogsFailure = createAction('CLEAR_LOGS_FAILURE');
export const clearLogsSuccess = createAction('CLEAR_LOGS_SUCCESS');

export const clearLogs = () => async (dispatch) => {
    dispatch(clearLogsRequest());
    try {
        await apiClient.clearQueryLog();
        dispatch(clearLogsSuccess());
        dispatch(addSuccessToast('query_log_cleared'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(clearLogsFailure(error));
    }
};

export const getLogsConfigRequest = createAction('GET_LOGS_CONFIG_REQUEST');
export const getLogsConfigFailure = createAction('GET_LOGS_CONFIG_FAILURE');
export const getLogsConfigSuccess = createAction('GET_LOGS_CONFIG_SUCCESS');

export const getLogsConfig = () => async (dispatch) => {
    dispatch(getLogsConfigRequest());
    try {
        const data = await apiClient.getQueryLogInfo();
        dispatch(getLogsConfigSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsConfigFailure());
    }
};

export const setLogsConfigRequest = createAction('SET_LOGS_CONFIG_REQUEST');
export const setLogsConfigFailure = createAction('SET_LOGS_CONFIG_FAILURE');
export const setLogsConfigSuccess = createAction('SET_LOGS_CONFIG_SUCCESS');

export const setLogsConfig = config => async (dispatch) => {
    dispatch(setLogsConfigRequest());
    try {
        await apiClient.setQueryLogConfig(config);
        dispatch(addSuccessToast('config_successfully_saved'));
        dispatch(setLogsConfigSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setLogsConfigFailure());
    }
};
