import { createAction } from 'redux-actions';

import apiClient from '../api/Api';
import { normalizeLogs, getParamsForClientsSearch, addClientInfo } from '../helpers/helpers';
import {
    DEFAULT_LOGS_FILTER, FORM_NAME, QUERY_LOGS_PAGE_LIMIT,
} from '../helpers/constants';
import { addErrorToast, addSuccessToast } from './toasts';

const enrichWithClientInfo = async (logs) => {
    const clientsParams = getParamsForClientsSearch(logs, 'client');

    if (Object.keys(clientsParams).length > 0) {
        const clients = await apiClient.findClients(clientsParams);
        return addClientInfo(logs, clients, 'client');
    }

    return logs;
};

const getLogsWithParams = async (config) => {
    const { older_than, filter, ...values } = config;
    const rawLogs = await apiClient.getQueryLog({
        ...filter,
        older_than,
    });
    const { data, oldest } = rawLogs;
    const normalizedLogs = normalizeLogs(data);
    const logs = await enrichWithClientInfo(normalizedLogs);

    return {
        logs,
        oldest,
        older_than,
        filter,
        ...values,
    };
};

export const getAdditionalLogsRequest = createAction('GET_ADDITIONAL_LOGS_REQUEST');
export const getAdditionalLogsFailure = createAction('GET_ADDITIONAL_LOGS_FAILURE');
export const getAdditionalLogsSuccess = createAction('GET_ADDITIONAL_LOGS_SUCCESS');

const shortPollQueryLogs = async (data, filter, dispatch, getState, total) => {
    const { logs, oldest } = data;
    const totalData = total || { logs };

    const queryForm = getState().form[FORM_NAME.LOGS_FILTER];
    const currentQuery = queryForm && queryForm.values.search;
    const previousQuery = filter?.search;
    const isQueryTheSame = typeof previousQuery === 'string'
            && typeof currentQuery === 'string'
            && previousQuery === currentQuery;

    const isShortPollingNeeded = (logs.length < QUERY_LOGS_PAGE_LIMIT
            || totalData.logs.length < QUERY_LOGS_PAGE_LIMIT)
            && oldest !== '' && isQueryTheSame;

    if (isShortPollingNeeded) {
        dispatch(getAdditionalLogsRequest());

        try {
            const additionalLogs = await getLogsWithParams({
                older_than: oldest,
                filter,
            });
            if (additionalLogs.oldest.length > 0) {
                return await shortPollQueryLogs(additionalLogs, filter, dispatch, getState, {
                    logs: [...totalData.logs, ...additionalLogs.logs],
                    oldest: additionalLogs.oldest,
                });
            }
            dispatch(getAdditionalLogsSuccess());
            return totalData;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(getAdditionalLogsFailure(error));
        }
    }

    dispatch(getAdditionalLogsSuccess());
    return totalData;
};

export const toggleDetailedLogs = createAction('TOGGLE_DETAILED_LOGS');

export const getLogsRequest = createAction('GET_LOGS_REQUEST');
export const getLogsFailure = createAction('GET_LOGS_FAILURE');
export const getLogsSuccess = createAction('GET_LOGS_SUCCESS');

export const updateLogs = () => async (dispatch, getState) => {
    try {
        const { logs, oldest, older_than } = getState().queryLogs;

        const enrichedLogs = await enrichWithClientInfo(logs);

        dispatch(getLogsSuccess({
            logs: enrichedLogs,
            oldest,
            older_than,
        }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsFailure(error));
    }
};

export const getLogs = () => async (dispatch, getState) => {
    dispatch(getLogsRequest());
    try {
        const { isFiltered, filter, oldest } = getState().queryLogs;
        const data = await getLogsWithParams({
            older_than: oldest,
            filter,
        });

        if (isFiltered) {
            const additionalData = await shortPollQueryLogs(data, filter, dispatch, getState);
            const updatedData = additionalData.logs ? { ...data, ...additionalData } : data;
            dispatch(getLogsSuccess(updatedData));
        } else {
            dispatch(getLogsSuccess(data));
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsFailure(error));
    }
};

export const setLogsFilterRequest = createAction('SET_LOGS_FILTER_REQUEST');

/**
 *
 * @param filter
 * @param {string} filter.search
 * @param {string} filter.response_status 'QUERY' field of RESPONSE_FILTER object
 * @returns function
 */
export const setLogsFilter = (filter) => setLogsFilterRequest(filter);

export const setFilteredLogsRequest = createAction('SET_FILTERED_LOGS_REQUEST');
export const setFilteredLogsFailure = createAction('SET_FILTERED_LOGS_FAILURE');
export const setFilteredLogsSuccess = createAction('SET_FILTERED_LOGS_SUCCESS');

export const setFilteredLogs = (filter) => async (dispatch, getState) => {
    dispatch(setFilteredLogsRequest());
    try {
        const data = await getLogsWithParams({
            older_than: '',
            filter,
        });
        const additionalData = await shortPollQueryLogs(data, filter, dispatch, getState);
        const updatedData = additionalData.logs ? { ...data, ...additionalData } : data;

        dispatch(setFilteredLogsSuccess({
            ...updatedData,
            filter,
        }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setFilteredLogsFailure(error));
    }
};

export const resetFilteredLogs = () => setFilteredLogs(DEFAULT_LOGS_FILTER);

export const refreshFilteredLogs = () => async (dispatch, getState) => {
    const { filter } = getState().queryLogs;
    await dispatch(setFilteredLogs(filter));
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

export const setLogsConfig = (config) => async (dispatch) => {
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
