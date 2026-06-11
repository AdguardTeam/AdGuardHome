import { Action } from 'redux';
import { createAction } from 'redux-actions';
import { ThunkAction, ThunkDispatch } from 'redux-thunk';

import intl from 'panel/common/intl';
import { QueryLogsData, RootState } from 'panel/initialState';
import { filterLogsByStatus } from 'panel/components/QueryLog/helpers';
import { LogEntry } from 'panel/components/QueryLog/types';
import { apiClient } from '../api/Api';

import { normalizeLogs } from '../helpers/helpers';
import { DEFAULT_LOGS_FILTER, QUERY_LOGS_PAGE_LIMIT } from '../helpers/constants';
import { addErrorToast, addSuccessToast } from './toasts';

export type SearchFormValues = {
    search: string;
    status: string;
    reason: string;
};

export type QueryLogConfigPayload = Pick<
    QueryLogsData,
    'anonymize_client_ip' | 'enabled' | 'ignored'
> & {
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

// Maps the frontend reason filter values to backend response_status values.
const REASON_TO_RESPONSE_STATUS: Record<string, string> = {
    FilteredBlackList: 'blocked',
    FilteredBlockedService: 'blocked_services',
    FilteredSafeBrowsing: 'blocked_safebrowsing',
    FilteredParental: 'blocked_parental',
    FilteredSafeSearch: 'safe_search',
    Rewrite: 'rewritten',
    RewriteEtcHosts: 'rewritten',
    RewriteRule: 'rewritten',
    NotFilteredWhiteList: 'whitelisted',
    NotFilteredNotFound: 'processed',
};

// Maps the frontend status categories to backend response_status values
// when the reason filter is 'all'.
const STATUS_TO_RESPONSE_STATUS: Record<string, string> = {
    allowed: 'whitelisted',
    processed: 'processed',
    blocked: 'blocked',
    rewritten: 'rewritten',
};

const getEffectiveResponseStatus = (filter?: SearchFormValues): string => {
    const reason = filter?.reason ?? DEFAULT_LOGS_FILTER.reason;
    const status = filter?.status ?? DEFAULT_LOGS_FILTER.status;

    if (reason !== 'all') {
        return REASON_TO_RESPONSE_STATUS[reason] ?? 'all';
    }

    return STATUS_TO_RESPONSE_STATUS[status] ?? 'all';
};

const getLogsWithParams = async (config: GetLogsParams): Promise<LogsResponse> => {
    const { older_than, filter, ...values } = config;
    const requestFilter = {
        search: filter?.search ?? DEFAULT_LOGS_FILTER.search,
        response_status: getEffectiveResponseStatus(filter),
    };
    const rawLogs = await apiClient.getQueryLog({
        ...requestFilter,
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
    const totalData = total
        ? {
              logs: [...total.logs, ...logs],
              oldest,
          }
        : {
              logs,
              oldest,
          };
    const visibleCount = filterLogsByStatus(
        totalData.logs,
        filter?.status || DEFAULT_LOGS_FILTER.status,
    ).length;

    const previousQuery = filter?.search;
    const isQueryTheSame =
        typeof previousQuery === 'string' &&
        typeof currentQuery === 'string' &&
        previousQuery === currentQuery;

    const isShortPollingNeeded =
        visibleCount < QUERY_LOGS_PAGE_LIMIT && oldest !== '' && isQueryTheSame;

    if (!isShortPollingNeeded) {
        dispatch(getAdditionalLogsSuccess());
        return totalData;
    }

    dispatch(getAdditionalLogsRequest());

    try {
        const additionalLogs = await getLogsWithParams({
            older_than: oldest,
            filter,
        });

        return await shortPollQueryLogs(additionalLogs, filter, dispatch, currentQuery, totalData);
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getAdditionalLogsFailure(error));
        return totalData;
    }
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

export const getLogs =
    (currentQuery?: string): AppThunk<Promise<void>> =>
    async (dispatch, getState) => {
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
                const additionalData = await shortPollQueryLogs(
                    data,
                    filter,
                    dispatch,
                    currentQuery,
                );
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
 * @param {string} filter.status 'QUERY' field of QUERY_LOG_STATUS_FILTER object
 * @param {string} filter.reason 'QUERY' field of QUERY_LOG_REASON_FILTER object
 * @returns function
 */
export const setLogsFilter = (filter: SearchFormValues) => setLogsFilterRequest(filter);

export const setFilteredLogsRequest = createAction('SET_FILTERED_LOGS_REQUEST');
export const setFilteredLogsFailure = createAction<unknown>('SET_FILTERED_LOGS_FAILURE');
export const setFilteredLogsSuccess = createAction<LogsResponse & { filter?: SearchFormValues }>(
    'SET_FILTERED_LOGS_SUCCESS',
);

export const setFilteredLogs =
    (filter?: SearchFormValues): AppThunk<Promise<boolean>> =>
    async (dispatch) => {
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

            return true;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(setFilteredLogsFailure(error));

            return false;
        }
    };

export const resetFilteredLogs = () => setFilteredLogs(DEFAULT_LOGS_FILTER);

export const refreshFilteredLogs = (): AppThunk<Promise<boolean>> => async (dispatch, getState) => {
    const queryLogs = getState().queryLogs as QueryLogsDataState | undefined;
    if (!queryLogs) {
        return false;
    }

    const { filter } = queryLogs;
    const refreshed = await dispatch(setFilteredLogs(filter));

    if (refreshed) {
        dispatch(
            addSuccessToast({
                message: intl.getMessage('notify_updated'),
                code: 'notify_updated',
            }),
        );
    }

    return refreshed;
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
        const data = (await apiClient.getQueryLogConfig()) as QueryLogConfigPayload;
        dispatch(getLogsConfigSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getLogsConfigFailure());
    }
};

export const setLogsConfigRequest = createAction('SET_LOGS_CONFIG_REQUEST');
export const setLogsConfigFailure = createAction('SET_LOGS_CONFIG_FAILURE');
export const setLogsConfigSuccess = createAction<QueryLogConfigPayload>('SET_LOGS_CONFIG_SUCCESS');

export const setLogsConfig =
    (config: QueryLogConfigPayload): AppThunk<Promise<void>> =>
    async (dispatch) => {
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
