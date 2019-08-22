import { handleActions } from 'redux-actions';

import * as actions from '../actions/stats';

const stats = handleActions(
    {
        [actions.getStatsConfigRequest]: state => ({ ...state, processingGetConfig: true }),
        [actions.getStatsConfigFailure]: state => ({ ...state, processingGetConfig: false }),
        [actions.getStatsConfigSuccess]: (state, { payload }) => ({
            ...state,
            interval: payload.interval,
            processingGetConfig: false,
        }),

        [actions.setStatsConfigRequest]: state => ({ ...state, processingSetConfig: true }),
        [actions.setStatsConfigFailure]: state => ({ ...state, processingSetConfig: false }),
        [actions.setStatsConfigSuccess]: (state, { payload }) => ({
            ...state,
            interval: payload.interval,
            processingSetConfig: false,
        }),

        [actions.getStatsRequest]: state => ({ ...state, processingStats: true }),
        [actions.getStatsFailure]: state => ({ ...state, processingStats: false }),
        [actions.getStatsSuccess]: (state, { payload }) => {
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
                topQueriedDomains,
                numBlockedFiltering,
                numDnsQueries,
                numReplacedParental,
                numReplacedSafebrowsing,
                numReplacedSafesearch,
                avgProcessingTime,
            };

            return newState;
        },
    },
    {
        processingGetConfig: false,
        processingSetConfig: false,
        processingStats: true,
        interval: 1,
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
    },
);

export default stats;
