import { createStore } from 'solid-js/store';
import { apiClient } from 'panel/api/Api';
import { addErrorToast } from './toasts';
import { DAY, TIME_UNITS } from 'panel/helpers/constants';

/**
 * Normalizes API top-list format `[{ "key": value }]` to `[{ name: "key", count: value }]`.
 */
const normalizeTopList = <T extends number>(
    raw: Record<string, T>[],
): { name: string; count: T }[] =>
    raw.map((item) => {
        const [[name, count]] = Object.entries(item);
        return { name, count };
    });

type StatsState = {
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    processingStats: boolean;
    processingReset: boolean;
    interval: number;
    customInterval: number | null;
    dnsQueries: number[];
    blockedFiltering: number[];
    replacedParental: number[];
    replacedSafebrowsing: number[];
    topBlockedDomains: { name: string; count: number }[];
    topClients: { name: string; count: number; info: any }[];
    topQueriedDomains: { name: string; count: number }[];
    numBlockedFiltering: number;
    numDnsQueries: number;
    numReplacedParental: number;
    numReplacedSafebrowsing: number;
    numReplacedSafesearch: number;
    avgProcessingTime: number;
    timeUnits: string;
    enabled: boolean;
    topUpstreamsAvgTime: { name: string; count: number }[];
    topUpstreamsResponses: { name: string; count: number }[];
    ignored: string[];
};

const initialState: StatsState = {
    processingGetConfig: false,
    processingSetConfig: false,
    processingStats: true,
    processingReset: false,
    interval: DAY,
    customInterval: null,
    dnsQueries: [],
    blockedFiltering: [],
    replacedParental: [],
    replacedSafebrowsing: [],
    topBlockedDomains: [],
    topClients: [],
    topQueriedDomains: [],
    numBlockedFiltering: 0,
    numDnsQueries: 0,
    numReplacedParental: 0,
    numReplacedSafebrowsing: 0,
    numReplacedSafesearch: 0,
    avgProcessingTime: 0,
    timeUnits: TIME_UNITS?.HOURS || 'hours',
    enabled: true,
    topUpstreamsAvgTime: [],
    topUpstreamsResponses: [],
    ignored: [],
};

const [state, setState] = createStore<StatsState>(initialState);

export const getStats = async (period?: number) => {
    setState('processingStats', true);
    try {
        const data = await apiClient.getStats(period ?? 0);
        setState({
            dnsQueries: data.dns_queries || [],
            blockedFiltering: data.blocked_filtering || [],
            replacedParental: data.replaced_parental || [],
            replacedSafebrowsing: data.replaced_safebrowsing || [],
            topBlockedDomains: normalizeTopList(data.top_blocked_domains || []),
            topClients: normalizeTopList(data.top_clients || []) as { name: string; count: number; info: any }[],
            topQueriedDomains: normalizeTopList(data.top_queried_domains || []),
            numBlockedFiltering: data.num_blocked_filtering || 0,
            numDnsQueries: data.num_dns_queries || 0,
            numReplacedParental: data.num_replaced_parental || 0,
            numReplacedSafebrowsing: data.num_replaced_safebrowsing || 0,
            numReplacedSafesearch: data.num_replaced_safesearch || 0,
            avgProcessingTime: data.avg_processing_time || 0,
            timeUnits: data.time_units || initialState.timeUnits,
            topUpstreamsAvgTime: normalizeTopList(data.top_upstreams_avg_time || []),
            topUpstreamsResponses: normalizeTopList(data.top_upstreams_responses || []),
            processingStats: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingStats', false);
    }
};

export const getStatsConfig = async () => {
    setState('processingGetConfig', true);
    try {
        const data = await apiClient.getStatsConfig();
        setState({
            interval: data.interval || DAY,
            enabled: data.enabled ?? true,
            customInterval: data.custom_interval || null,
            ignored: data.ignored || [],
            processingGetConfig: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingGetConfig', false);
    }
};

export const setStatsConfig = async (values: any) => {
    setState('processingSetConfig', true);
    try {
        await apiClient.setStatsConfig(values);
        setState({ ...values, processingSetConfig: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
    }
};

export const resetStats = async () => {
    setState('processingReset', true);
    try {
        await apiClient.resetStats();
        setState('processingReset', false);
        await getStats();
    } catch (error) {
        addErrorToast({ error });
        setState('processingReset', false);
    }
};

export const statsState = state;
