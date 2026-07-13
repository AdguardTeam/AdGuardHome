import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import {
    DAY,
    DEFAULT_LOGS_FILTER,
    HOUR,
    QUERY_LOGS_PAGE_LIMIT,
    QUERY_LOG_INTERVALS_DAYS,
    QUERY_LOG_REASON_FILTER,
} from 'panel/helpers/constants';
import { normalizeLogs } from 'panel/helpers/helpers';

type QueryLogsState = {
    processingGetLogs: boolean;
    processingClear: boolean;
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    processingAdditionalLogs: boolean;
    interval: number;
    logs: any[];
    enabled: boolean;
    oldest: string;
    filter: any;
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

const getReasons = (filter?: any): string[] => {
    const reason = filter?.reason ?? DEFAULT_LOGS_FILTER.reason;
    const status = filter?.status ?? DEFAULT_LOGS_FILTER.status;
    if (reason !== 'all') {
        return REASON_FILTER_TO_REASONS[reason] ?? [];
    }
    return STATUS_TO_REASONS[status] ?? [];
};

const fetchLogsWithParams = async (olderThan: string, filter?: any) => {
    const params: Record<string, any> = {
        search: filter?.search ?? DEFAULT_LOGS_FILTER.search,
        older_than: olderThan,
    };
    const reasons = getReasons(filter);
    if (reasons.length > 0) {
        params.reason = reasons;
    }
    const raw = await apiClient.getQueryLog(params);
    return { logs: normalizeLogs(raw.data || []), oldest: raw.oldest || '' };
};

/** Simple stateless filter: count entries matching the status */
const filterLogsByStatus = (logs: any[], status: string): any[] => {
    if (!status || status === 'all') return logs;
    const reasons = STATUS_TO_REASONS[status];
    if (!reasons || reasons.length === 0) return logs;
    return logs.filter((log: any) => reasons.includes(log.reason));
};

const shortPollQueryLogs = async (
    data: { logs: any[]; oldest: string },
    filter: any,
    total?: { logs: any[]; oldest: string },
): Promise<{ logs: any[]; oldest: string }> => {
    const totalData = total
        ? { logs: [...total.logs, ...data.logs], oldest: data.oldest }
        : { logs: data.logs, oldest: data.oldest };
    const visible = filterLogsByStatus(
        totalData.logs,
        filter?.status || DEFAULT_LOGS_FILTER.status,
    ).length;
    if (visible >= QUERY_LOGS_PAGE_LIMIT || totalData.oldest === '') return totalData;
    const more = await fetchLogsWithParams(totalData.oldest, filter);
    return shortPollQueryLogs(more, filter, totalData);
};

// ---------- Public actions ----------

export const getLogs = async (currentQuery?: string) => {
    setState('processingGetLogs', true);
    try {
        const { isFiltered, filter, oldest } = untrack(() => state);
        const data = await fetchLogsWithParams(oldest, filter);
        if (isFiltered) {
            const accumulated = await shortPollQueryLogs(data, filter);
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
        addErrorToast({ error });
        setState('processingGetLogs', false);
    }
};

export const getAdditionalLogs = async () => {
    setState('processingAdditionalLogs', true);
    try {
        const { filter, oldest } = untrack(() => state);
        const data = await fetchLogsWithParams(oldest, filter);
        setState({
            logs: [...state.logs, ...data.logs],
            oldest: data.oldest,
            isEntireLog: data.oldest === '',
            processingAdditionalLogs: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingAdditionalLogs', false);
    }
};

export const clearLogs = async () => {
    setState('processingClear', true);
    try {
        await apiClient.clearQueryLog();
        setState({
            logs: [],
            oldest: '',
            isEntireLog: false,
            processingClear: false,
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
        const data = await apiClient.getQueryLogConfig();
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
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingGetConfig', false);
    }
};

export const setLogsConfig = async (values: any): Promise<boolean> => {
    setState('processingSetConfig', true);
    try {
        await apiClient.setQueryLogConfig(values);
        setState({ ...values, processingSetConfig: false });
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
        return false;
    }
};

export const setFilteredLogs = async (filter?: any): Promise<boolean> => {
    setState({
        filter: filter ?? DEFAULT_LOGS_FILTER,
        isFiltered: true,
        processingGetLogs: true,
    });
    try {
        const data = await fetchLogsWithParams('', filter);
        const accumulated = await shortPollQueryLogs(data, filter);
        setState({
            logs: accumulated.logs,
            oldest: accumulated.oldest,
            isEntireLog: accumulated.oldest === '',
            filter: filter ?? DEFAULT_LOGS_FILTER,
            processingGetLogs: false,
        });
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingGetLogs', false);
        return false;
    }
};

export const setLogsFilter = (filter: any) => {
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
