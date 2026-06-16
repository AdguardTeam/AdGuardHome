import { createStore } from 'solid-js/store';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import { DAY, DEFAULT_LOGS_FILTER } from 'panel/helpers/constants';

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

export const getLogs = async (params?: any) => {
    setState('processingGetLogs', true);
    try {
        const data = await apiClient.getQueryLog(params);
        setState({
            logs: data.data || [],
            oldest: data.oldest || '',
            isEntireLog: data.is_entire_log || false,
            processingGetLogs: false,
        });
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
            logs: [...state.logs, ...(data.data || [])],
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
            customInterval: data.custom_interval || null,
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
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
    }
};

export const setFilteredLogs = (filter: any) => {
    setState({ filter, isFiltered: true });
    // Fetch filtered logs
    getLogs(filter);
};

export const setLogsFilter = (filter: any) => {
    setState({ filter });
};

export const refreshFilteredLogs = () => {
    setState('processingGetLogs', true);
    return getLogs(state.filter);
};

export const toggleDetailedLogs = () => {
    setState('isDetailed', (prev) => !prev);
};

export const queryLogsState = state;
