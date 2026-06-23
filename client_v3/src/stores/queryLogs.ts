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
};

const [state, setState] = createStore<QueryLogsState>(initialState);

// ---------- Short-poll helpers (v2 parity) ----------

const REASON_TO_RESPONSE_STATUS: Record<string, string> = {
    [QUERY_LOG_REASON_FILTER.BLOCKED_BY_FILTER.QUERY]: 'blocked',
    [QUERY_LOG_REASON_FILTER.BLOCKED_SERVICES.QUERY]: 'blocked_services',
    [QUERY_LOG_REASON_FILTER.BLOCKED_BY_THREATS.QUERY]: 'blocked_safebrowsing',
    [QUERY_LOG_REASON_FILTER.BLOCKED_BY_PARENTAL_CONTROL.QUERY]: 'blocked_parental',
    [QUERY_LOG_REASON_FILTER.SAFE_SEARCH.QUERY]: 'safe_search',
    [QUERY_LOG_REASON_FILTER.DNS_REWRITES.QUERY]: 'rewritten',
};

const STATUS_TO_RESPONSE_STATUS: Record<string, string> = {
    allowed: 'whitelisted',
    processed: 'processed',
    blocked: 'blocked',
    rewritten: 'rewritten',
};

const getEffectiveResponseStatus = (filter?: any) => {
    const reason = filter?.reason ?? DEFAULT_LOGS_FILTER.reason;
    const status = filter?.status ?? DEFAULT_LOGS_FILTER.status;
    if (reason !== 'all') return REASON_TO_RESPONSE_STATUS[reason] ?? 'all';
    return STATUS_TO_RESPONSE_STATUS[status] ?? 'all';
};

const fetchLogsWithParams = async (olderThan: string, filter?: any) => {
    const raw = await apiClient.getQueryLog({
        search: filter?.search ?? DEFAULT_LOGS_FILTER.search,
        response_status: getEffectiveResponseStatus(filter),
        older_than: olderThan,
    });
    return { logs: normalizeLogs(raw.data || []), oldest: raw.oldest || '' };
};

/** Simple stateless filter: count entries matching the status */
const filterLogsByStatus = (logs: any[], status: string): any[] => {
    if (!status || status === 'all') return logs;
    const statusToReason: Record<string, string[]> = {
        processed: ['Filtered', 'NotFiltered'],
        allowed: ['NotFilteredNotFound', 'NotFilteredWhiteList'],
        blocked: [
            'FilteredBlackList',
            'FilteredBlockedService',
            'FilteredSafeBrowsing',
            'FilteredParental',
            'FilteredInvalid',
        ],
        rewritten: ['Rewrite', 'RewriteEtcHosts', 'RewriteRule'],
    };
    const reasons = statusToReason[status];
    if (!reasons) return logs;
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

export const getAdditionalLogs = async (params?: any) => {
    setState('processingAdditionalLogs', true);
    try {
        const data = await apiClient.getQueryLog(params);
        setState({
            logs: [...state.logs, ...normalizeLogs(data.data || [])],
            oldest: data.oldest || '',
            isEntireLog: data.is_entire_log || false,
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
            processingGetConfig: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingGetConfig', false);
    }
};

export const setLogsConfig = async (values: any) => {
    setState('processingSetConfig', true);
    try {
        await apiClient.setQueryLogConfig(values);
        setState({ ...values, processingSetConfig: false });
        addSuccessToast(intl.getMessage('settings_notify_changes_saved'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
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
