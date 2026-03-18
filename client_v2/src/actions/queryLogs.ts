import { Action } from 'redux';
import { createAction } from 'redux-actions';
import { ThunkAction, ThunkDispatch } from 'redux-thunk';

import intl from 'panel/common/intl';
import { QueryLogsData, RootState } from 'panel/initialState';
import { LogEntry } from 'panel/components/QueryLog/types';
import { apiClient } from '../api/Api';

import { normalizeLogs } from '../helpers/helpers';
import { DEFAULT_LOGS_FILTER, QUERY_LOGS_PAGE_LIMIT } from '../helpers/constants';
import { addErrorToast, addSuccessToast } from './toasts';

export type SearchFormValues = {
    search: string;
    response_status: string;
};

export type QueryLogConfigPayload = Pick<QueryLogsData, 'anonymize_client_ip' | 'enabled' | 'ignored'> & {
    ignore_enabled: boolean;
    interval: number;
};

type AppThunk<ReturnType = void> = ThunkAction<ReturnType, RootState, unknown, Action<string>>;
type AppDispatch = ThunkDispatch<RootState, unknown, Action<string>>;

type QueryLogsDataState = NonNullable<RootState['queryLogs']>;

type GetLogsParams = {
    older_than: string;
    filter?: SearchFormValues;
};

type LogsResponse = {
    logs: LogEntry[];
    oldest: string;
    older_than: string;
    filter?: SearchFormValues;
};

type ShortPollTotal = {
    logs: LogEntry[];
    oldest?: string;
};

const getLogsWithParams = async (config: GetLogsParams): Promise<LogsResponse> => {
    const { older_than, filter, ...values } = config;
    const rawLogs = await apiClient.getQueryLog({
        ...filter,
        older_than,
    });
    const { data, oldest } = rawLogs;

    return {
        logs: normalizeLogs(data),
        oldest,
        older_than,
        filter,
        ...values,
    };
};

export const getAdditionalLogsRequest = createAction('GET_ADDITIONAL_LOGS_REQUEST');
export const getAdditionalLogsFailure = createAction<unknown>('GET_ADDITIONAL_LOGS_FAILURE');
export const getAdditionalLogsSuccess = createAction('GET_ADDITIONAL_LOGS_SUCCESS');

const shortPollQueryLogs = async (
    data: LogsResponse,
    filter: SearchFormValues | undefined,
    dispatch: AppDispatch,
    currentQuery?: string,
    total?: ShortPollTotal,
): Promise<ShortPollTotal | undefined> => {
    const { logs, oldest } = data;
    const totalData = total || { logs };

    const previousQuery = filter?.search;
    const isQueryTheSame =
        typeof previousQuery === 'string' && typeof currentQuery === 'string' && previousQuery === currentQuery;

    const isShortPollingNeeded =
        (logs.length < QUERY_LOGS_PAGE_LIMIT || totalData.logs.length < QUERY_LOGS_PAGE_LIMIT) &&
        oldest !== '' &&
        isQueryTheSame;

    if (isShortPollingNeeded) {
        dispatch(getAdditionalLogsRequest());

        try {
            const additionalLogs = await getLogsWithParams({
                older_than: oldest,
                filter,
            });
            if (additionalLogs.oldest.length > 0) {
                return await shortPollQueryLogs(additionalLogs, filter, dispatch, currentQuery, {
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
export const getLogsFailure = createAction<unknown>('GET_LOGS_FAILURE');
export const getLogsSuccess = createAction<LogsResponse>('GET_LOGS_SUCCESS');

export const updateLogs = (): AppThunk => async (dispatch, getState) => {
    try {
        const queryLogs = getState().queryLogs as QueryLogsDataState | undefined;
        if (!queryLogs) {
            return;
        }

        const { logs, oldest } = queryLogs;

        dispatch(
            getLogsSuccess({
                logs,
                oldest,
                older_than: oldest,
            }),
        );
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsFailure(error));
    }
};

export const getLogs = (currentQuery?: string): AppThunk<Promise<void>> => async (dispatch, getState) => {
    dispatch(getLogsRequest());
    try {
        const queryLogs = getState().queryLogs as QueryLogsDataState | undefined;
        if (!queryLogs) {
            dispatch(getLogsFailure(new Error('Query logs state is unavailable')));
            return;
        }

        const { isFiltered, filter, oldest } = queryLogs;

        const data = await getLogsWithParams({
            older_than: oldest,
            filter,
        });

        if (isFiltered) {
            const additionalData = await shortPollQueryLogs(data, filter, dispatch, currentQuery);
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
export const setLogsFilter = (filter: SearchFormValues) => setLogsFilterRequest(filter);

export const setFilteredLogsRequest = createAction('SET_FILTERED_LOGS_REQUEST');
export const setFilteredLogsFailure = createAction<unknown>('SET_FILTERED_LOGS_FAILURE');
export const setFilteredLogsSuccess = createAction<LogsResponse & { filter?: SearchFormValues }>('SET_FILTERED_LOGS_SUCCESS');

export const setFilteredLogs = (filter?: SearchFormValues): AppThunk<Promise<void>> => async (dispatch) => {
    dispatch(setFilteredLogsRequest());
    try {
        const data = await getLogsWithParams({
            older_than: '',
            filter,
        });

        const currentQuery = filter?.search;

        const additionalData = await shortPollQueryLogs(data, filter, dispatch, currentQuery);
        const updatedData = additionalData.logs ? { ...data, ...additionalData } : data;

        dispatch(
            setFilteredLogsSuccess({
                ...updatedData,
                filter,
            }),
        );
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setFilteredLogsFailure(error));
    }
};

export const resetFilteredLogs = () => setFilteredLogs(DEFAULT_LOGS_FILTER);

export const refreshFilteredLogs = (): AppThunk<Promise<void>> => async (dispatch, getState) => {
    const queryLogs = getState().queryLogs as QueryLogsDataState | undefined;
    if (!queryLogs) {
        return;
    }

    const { filter } = queryLogs;
    await dispatch(setFilteredLogs(filter));
};

export const clearLogsRequest = createAction('CLEAR_LOGS_REQUEST');
export const clearLogsFailure = createAction<unknown>('CLEAR_LOGS_FAILURE');
export const clearLogsSuccess = createAction('CLEAR_LOGS_SUCCESS');

export const clearLogs = (): AppThunk<Promise<void>> => async (dispatch) => {
    dispatch(clearLogsRequest());
    try {
        await apiClient.clearQueryLog();
        dispatch(clearLogsSuccess());
        dispatch(addSuccessToast(intl.getMessage('settings_notify_query_log_cleared')));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(clearLogsFailure(error));
    }
};

export const getLogsConfigRequest = createAction('GET_LOGS_CONFIG_REQUEST');
export const getLogsConfigFailure = createAction('GET_LOGS_CONFIG_FAILURE');
export const getLogsConfigSuccess = createAction<QueryLogConfigPayload>('GET_LOGS_CONFIG_SUCCESS');

export const getLogsConfig = (): AppThunk<Promise<void>> => async (dispatch) => {
    dispatch(getLogsConfigRequest());
    try {
        const data = await apiClient.getQueryLogConfig() as QueryLogConfigPayload;
        dispatch(getLogsConfigSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsConfigFailure());
    }
};

export const setLogsConfigRequest = createAction('SET_LOGS_CONFIG_REQUEST');
export const setLogsConfigFailure = createAction('SET_LOGS_CONFIG_FAILURE');
export const setLogsConfigSuccess = createAction<QueryLogConfigPayload>('SET_LOGS_CONFIG_SUCCESS');

export const setLogsConfig = (config: QueryLogConfigPayload): AppThunk<Promise<void>> => async (dispatch) => {
    dispatch(setLogsConfigRequest());
    try {
        await apiClient.setQueryLogConfig(config);
        dispatch(addSuccessToast(intl.getMessage('settings_notify_changes_saved')));
        dispatch(setLogsConfigSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setLogsConfigFailure());
    }
};
