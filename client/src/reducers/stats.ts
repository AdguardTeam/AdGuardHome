import { handleActions } from 'redux-actions';

import { normalizeTopClients } from '../helpers/helpers';
import { DAY, HOUR, STATS_INTERVALS_DAYS, TIME_UNITS } from '../helpers/constants';

import * as actions from '../actions/stats';

const defaultStats = {
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
    timeUnits: TIME_UNITS.HOURS,
};

const stats = handleActions(
    {
        [actions.getStatsConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingGetConfig: true,
        }),
        [actions.getStatsConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingGetConfig: false,
        }),
        [actions.getStatsConfigSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,

            customInterval: !STATS_INTERVALS_DAYS.includes(payload.interval) ? payload.interval / HOUR : null,

            processingGetConfig: false,
        }),

        [actions.setStatsConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingSetConfig: true,
        }),
        [actions.setStatsConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingSetConfig: false,
        }),
        [actions.setStatsConfigSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),

        [actions.getStatsRequest.toString()]: (state: any) => ({
            ...state,
            processingStats: true,
        }),
        [actions.getStatsFailure.toString()]: (state: any) => ({
            ...state,
            processingStats: false,
        }),
        [actions.getStatsSuccess.toString()]: (state: any, { payload }: any) => {
            const {
                dns_queries: dnsQueries,
                blocked_filtering: blockedFiltering,
                replaced_parental: replacedParental,
                replaced_safebrowsing: replacedSafebrowsing,
                top_blocked_domains: topBlockedDomains,
                top_clients: topClients,
                top_queried_domains: topQueriedDomains,
                num_blocked_filtering: numBlockedFiltering,
                num_dns_queries: numDnsQueries,
                num_replaced_parental: numReplacedParental,
                num_replaced_safebrowsing: numReplacedSafebrowsing,
                num_replaced_safesearch: numReplacedSafesearch,
                avg_processing_time: avgProcessingTime,
                top_upstreams_responses: topUpstreamsResponses,
                top_upstrems_avg_time: topUpstreamsAvgTime,
                time_units: timeUnits,
            } = payload;

            const newState = {
                ...state,
                processingStats: false,
                dnsQueries,
                blockedFiltering,
                replacedParental,
                replacedSafebrowsing,
                topBlockedDomains,
                topClients,
                normalizedTopClients: normalizeTopClients(topClients),
                topQueriedDomains,
                numBlockedFiltering,
                numDnsQueries,
                numReplacedParental,
                numReplacedSafebrowsing,
                numReplacedSafesearch,
                avgProcessingTime,
                topUpstreamsResponses,
                topUpstreamsAvgTime,
                timeUnits,
            };

            return newState;
        },

        [actions.resetStatsRequest.toString()]: (state: any) => ({
            ...state,
            processingReset: true,
        }),
        [actions.resetStatsFailure.toString()]: (state: any) => ({
            ...state,
            processingReset: false,
        }),
        [actions.resetStatsSuccess.toString()]: (state: any) => ({
            ...state,
            ...defaultStats,
            processingReset: false,
        }),
    },
    {
        processingGetConfig: false,
        processingSetConfig: false,
        processingStats: true,
        processingReset: false,
        interval: DAY,
        customInterval: null,
        ...defaultStats,
    },
);

export default stats;
