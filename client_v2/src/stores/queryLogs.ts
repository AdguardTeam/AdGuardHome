import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { queryLog, querylogClear, getQueryLogConfig, putQueryLogConfig } from 'panel/api/generated';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import {
    DAY,
    DEFAULT_LOGS_FILTER,
    HOUR,
    QUERY_LOGS_PAGE_LIMIT,
    QUERY_LOG_INTERVALS_DAYS,
    QUERY_LOG_REASON_FILTER,
    type QueryLogFilter,
} from 'panel/helpers/constants';
import { normalizeLogs, type NormalizedQueryLogItem } from 'panel/helpers/helpers';
import type { GetQueryLogConfigResponse } from 'panel/api/model/getQueryLogConfigResponse';

type QueryLogsState = {
    processingGetLogs: boolean;
    processingClear: boolean;
    processingGetConfig: boolean;
    configInitialized: boolean;
    processingSetConfig: boolean;
    processingAdditionalLogs: boolean;
    interval: number;
    logs: NormalizedQueryLogItem[];
    enabled: boolean;
    oldest: string;
    filter: QueryLogFilter;
    isFiltered: boolean;
    anonymize_client_ip: boolean;
    isDetailed: boolean;
    isEntireLog: boolean;
    customInterval: number | null;
    ignored: string[];
    ignored_enabled: boolean;
};

const initialState: QueryLogsState = {
    processingGetLogs: true,
    processingClear: false,
    processingGetConfig: false,
    configInitialized: false,
    processingSetConfig: false,
    processingAdditionalLogs: false,
    interval: DAY,
    logs: [],
    enabled: true,
    oldest: '',
    filter: DEFAULT_LOGS_FILTER,
    isFiltered: false,
    anonymize_client_ip: false,
    isDetailed: true,
    isEntireLog: false,
    customInterval: null,
    ignored: [],
    ignored_enabled: false,
};

const [state, setState] = createStore<QueryLogsState>(initialState);

// A response may land after the search it belongs to was replaced. Each log
// request takes a token and only the latest token may write state or continue
// paging; discarding is enough because `customFetch` turns AbortError into a
// generic error and would surface a false toast.
let latestLogsRequest = 0;
let latestAdditionalRequest = 0;

const beginLogsRequest = () => {
    latestLogsRequest += 1;
    return latestLogsRequest;
};

const isLatestLogsRequest = (id: number) => id === latestLogsRequest;

// ---------- Short-poll helpers (v2 parity) ----------

/** Maps frontend status filter → exact backend reason strings */
const STATUS_TO_REASONS: Record<string, string[]> = {
    blocked: [
        'FilteredBlackList',
        'FilteredSafeBrowsing',
        'FilteredParental',
        'FilteredBlockedService',
    ],
    rewritten: ['Rewrite', 'RewriteEtcHosts', 'RewriteRule', 'FilteredSafeSearch'],
    processed: ['NotFilteredNotFound'],
    allowed: ['NotFilteredWhiteList'],
    error: ['NotFilteredError', 'FilteredInvalid'],
    all: [],
};

/** Maps frontend reason filter query → exact backend reason strings */
const REASON_FILTER_TO_REASONS: Record<string, string[]> = {
    [QUERY_LOG_REASON_FILTER.BLOCKED_BY_FILTER.QUERY]: ['FilteredBlackList'],
    [QUERY_LOG_REASON_FILTER.BLOCKED_SERVICES.QUERY]: ['FilteredBlockedService'],
    [QUERY_LOG_REASON_FILTER.BLOCKED_BY_THREATS.QUERY]: ['FilteredSafeBrowsing'],
    [QUERY_LOG_REASON_FILTER.BLOCKED_BY_PARENTAL_CONTROL.QUERY]: ['FilteredParental'],
    [QUERY_LOG_REASON_FILTER.SAFE_SEARCH.QUERY]: ['FilteredSafeSearch'],
    [QUERY_LOG_REASON_FILTER.DNS_REWRITES.QUERY]: ['Rewrite', 'RewriteEtcHosts', 'RewriteRule'],
};

const getReasons = (filter?: QueryLogFilter): string[] => {
    const reason = filter?.reason ?? DEFAULT_LOGS_FILTER.reason;
    const status = filter?.status ?? DEFAULT_LOGS_FILTER.status;
    if (reason !== 'all') {
        return REASON_FILTER_TO_REASONS[reason] ?? [];
    }
    return STATUS_TO_REASONS[status] ?? [];
};

const fetchLogsWithParams = async (olderThan: string, filter?: QueryLogFilter) => {
    const params: Record<string, string | string[] | number | undefined> = {
        search: filter?.search ?? DEFAULT_LOGS_FILTER.search,
        older_than: olderThan,
        limit: QUERY_LOGS_PAGE_LIMIT,
    };
    const reasons = getReasons(filter);
    if (reasons.length > 0) {
        params.reason = reasons;
    }
    const raw = await queryLog(params);
    return { logs: normalizeLogs(raw.data || []), oldest: raw.oldest || '' };
};

/** Simple stateless filter: count entries matching the status */
const filterLogsByStatus = (
    logs: NormalizedQueryLogItem[],
    status: string,
): NormalizedQueryLogItem[] => {
    if (!status || status === 'all') return logs;
    const reasons = STATUS_TO_REASONS[status];
    if (!reasons || reasons.length === 0) return logs;
    return logs.filter((log) => reasons.includes(log.reason ?? ''));
};

const shortPollQueryLogs = async (
    data: { logs: NormalizedQueryLogItem[]; oldest: string },
    filter: QueryLogFilter,
    requestId: number,
    total?: { logs: NormalizedQueryLogItem[]; oldest: string },
): Promise<{ logs: NormalizedQueryLogItem[]; oldest: string }> => {
    const totalData = total
        ? { logs: [...total.logs, ...data.logs], oldest: data.oldest }
        : { logs: data.logs, oldest: data.oldest };
    if (!isLatestLogsRequest(requestId)) return totalData;
    const visible = filterLogsByStatus(
        totalData.logs,
        filter?.status || DEFAULT_LOGS_FILTER.status,
    ).length;
    if (visible >= QUERY_LOGS_PAGE_LIMIT || totalData.oldest === '') return totalData;
    const more = await fetchLogsWithParams(totalData.oldest, filter);
    return shortPollQueryLogs(more, filter, requestId, totalData);
};

// ---------- Public actions ----------

export const getLogs = async (currentQuery?: string) => {
    const requestId = beginLogsRequest();
    setState('processingGetLogs', true);
    try {
        const { isFiltered, filter, oldest } = untrack(() => state);
        const data = await fetchLogsWithParams(oldest, filter);
        if (!isLatestLogsRequest(requestId)) {
            return;
        }
        if (isFiltered) {
            const accumulated = await shortPollQueryLogs(data, filter, requestId);
            if (!isLatestLogsRequest(requestId)) {
                return;
            }
            setState({
                logs: accumulated.logs,
                oldest: accumulated.oldest,
                isEntireLog: accumulated.oldest === '',
                processingGetLogs: false,
            });
        } else {
            setState({
                logs: data.logs,
                oldest: data.oldest,
                isEntireLog: data.oldest === '',
                processingGetLogs: false,
            });
        }
        void currentQuery; // retained for v2 compatibility
    } catch (error) {
        if (!isLatestLogsRequest(requestId)) {
            return;
        }
        addErrorToast({ error });
        setState('processingGetLogs', false);
    }
};

export const getAdditionalLogs = async () => {
    if (untrack(() => state.processingAdditionalLogs || state.processingGetLogs)) {
        return;
    }

    latestAdditionalRequest += 1;
    const token = latestAdditionalRequest;
    setState('processingAdditionalLogs', true);
    try {
        const { filter, oldest, logs } = untrack(() => state);
        const logsRequestId = latestLogsRequest;
        const data = await fetchLogsWithParams(oldest, filter);
        if (token !== latestAdditionalRequest || logsRequestId !== latestLogsRequest) {
            return;
        }
        setState({
            logs: [...logs, ...data.logs],
            oldest: data.oldest,
            isEntireLog: data.oldest === '',
        });
    } catch (error) {
        if (token !== latestAdditionalRequest) {
            return;
        }
        addErrorToast({ error });
    } finally {
        if (token === latestAdditionalRequest) {
            setState('processingAdditionalLogs', false);
        }
    }
};

export const clearLogs = async () => {
    setState('processingClear', true);
    try {
        await querylogClear();
        // The log is empty now: a search or a page that is still in flight
        // must neither repopulate it nor leave its loading flag stuck.
        latestLogsRequest += 1;
        latestAdditionalRequest += 1;
        setState({
            logs: [],
            oldest: '',
            isEntireLog: false,
            processingClear: false,
            processingGetLogs: false,
            processingAdditionalLogs: false,
        });
        addSuccessToast(intl.getMessage('settings_notify_query_log_cleared'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingClear', false);
    }
};

export const getLogsConfig = async () => {
    setState('processingGetConfig', true);
    try {
        const data = await getQueryLogConfig();
        setState({
            interval: data.interval || DAY,
            enabled: data.enabled ?? true,
            anonymize_client_ip: data.anonymize_client_ip ?? false,
            customInterval: !QUERY_LOG_INTERVALS_DAYS.includes(data.interval)
                ? data.interval / HOUR
                : null,
            ignored: data.ignored || [],
            ignored_enabled: data.ignored_enabled ?? false,
            processingGetConfig: false,
            configInitialized: true,
        });
    } catch (error) {
        addErrorToast({ error });
        setState({ processingGetConfig: false, configInitialized: true });
    }
};

export const setLogsConfig = async (values: GetQueryLogConfigResponse): Promise<boolean> => {
    setState('processingSetConfig', true);
    try {
        await putQueryLogConfig(values);
        setState({ ...values, processingSetConfig: false });
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
        return false;
    }
};

export const setFilteredLogs = async (filter?: QueryLogFilter): Promise<boolean> => {
    const requestId = beginLogsRequest();
    const nextFilter = filter ?? DEFAULT_LOGS_FILTER;
    setState({
        filter: nextFilter,
        isFiltered: true,
        processingGetLogs: true,
    });
    try {
        const data = await fetchLogsWithParams('', filter);
        if (!isLatestLogsRequest(requestId)) {
            return false;
        }
        const accumulated = await shortPollQueryLogs(data, filter, requestId);
        if (!isLatestLogsRequest(requestId)) {
            return false;
        }
        setState({
            logs: accumulated.logs,
            oldest: accumulated.oldest,
            isEntireLog: accumulated.oldest === '',
            filter: nextFilter,
            processingGetLogs: false,
        });
        return true;
    } catch (error) {
        if (!isLatestLogsRequest(requestId)) {
            return false;
        }
        addErrorToast({ error });
        setState('processingGetLogs', false);
        return false;
    }
};

export const setLogsFilter = (filter: QueryLogFilter): void => {
    setState({ filter });
};

export const refreshFilteredLogs = async (): Promise<boolean> => {
    const ok = await setFilteredLogs(untrack(() => state.filter));
    if (ok) {
        addSuccessToast({
            message: intl.getMessage('notify_updated'),
            code: 'notify_updated',
        });
    }
    return ok;
};

export const toggleDetailedLogs = () => {
    setState('isDetailed', (prev) => !prev);
};

export const queryLogsState = untrack(() => state);
