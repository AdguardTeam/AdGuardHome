import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import { DAY, HOUR, STATS_INTERVALS_DAYS, TIME_UNITS } from 'panel/helpers/constants';
import {
    normalizeTopStats,
    normalizeTopClients,
    addClientInfo,
    getParamsForClientsSearch,
    secondsToMilliseconds,
} from 'panel/helpers/helpers';

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
    normalizedTopClients: {
        auto: Record<string, number>;
        configured: Record<string, number>;
    };
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
    ignored_enabled: boolean;
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
    normalizedTopClients: { auto: {}, configured: {} },
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
    ignored_enabled: false,
};

const [state, setState] = createStore<StatsState>(initialState);

export const getStats = async (period?: number) => {
    setState('processingStats', true);
    try {
        const data = await apiClient.getStats(period ?? 0);

        const normalizedTopClientsList = normalizeTopStats(data.top_clients || []);
        const clientsParams = getParamsForClientsSearch(normalizedTopClientsList, 'name');
        const clients = await apiClient.searchClients(clientsParams);
        const topClientsWithInfo = addClientInfo(normalizedTopClientsList, clients, 'name');

        setState({
            dnsQueries: data.dns_queries || [],
            blockedFiltering: data.blocked_filtering || [],
            replacedParental: data.replaced_parental || [],
            replacedSafebrowsing: data.replaced_safebrowsing || [],
            topBlockedDomains: normalizeTopStats(data.top_blocked_domains || []),
            topClients: topClientsWithInfo,
            normalizedTopClients: normalizeTopClients(topClientsWithInfo),
            topQueriedDomains: normalizeTopStats(data.top_queried_domains || []),
            numBlockedFiltering: data.num_blocked_filtering || 0,
            numDnsQueries: data.num_dns_queries || 0,
            numReplacedParental: data.num_replaced_parental || 0,
            numReplacedSafebrowsing: data.num_replaced_safebrowsing || 0,
            numReplacedSafesearch: data.num_replaced_safesearch || 0,
            avgProcessingTime: secondsToMilliseconds(data.avg_processing_time),
            timeUnits: data.time_units || initialState.timeUnits,
            topUpstreamsAvgTime: normalizeTopStats(data.top_upstreams_avg_time || []),
            topUpstreamsResponses: normalizeTopStats(data.top_upstreams_responses || []),
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
            customInterval: !STATS_INTERVALS_DAYS.includes(data.interval)
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

export const setStatsConfig = async (values: any): Promise<boolean> => {
    setState('processingSetConfig', true);
    try {
        await apiClient.setStatsConfig(values);
        setState({ ...values, processingSetConfig: false });
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
        return false;
    }
};

export const resetStats = async () => {
    setState('processingReset', true);
    try {
        await apiClient.resetStats();
        setState('processingReset', false);
        addSuccessToast(intl.getMessage('settings_notify_statistics_cleared'));
        await getStats();
    } catch (error) {
        addErrorToast({ error });
        setState('processingReset', false);
    }
};

export const statsState = untrack(() => state);
