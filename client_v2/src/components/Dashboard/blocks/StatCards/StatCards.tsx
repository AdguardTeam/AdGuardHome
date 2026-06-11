import React from 'react';

import intl from 'panel/common/intl';
import { RoutePath } from 'panel/components/Routes/Paths';
import { QUERY_LOG_REASON_FILTER, QUERY_LOG_STATUS_FILTER } from 'panel/helpers/constants';

import s from '../StatCard/StatCard.module.pcss';
import { StatCard, CARDS_THEME, CARDS_COLORS } from '../StatCard';

type Props = {
    numDnsQueries: number;
    numBlockedFiltering: number;
    numReplacedSafebrowsing: number;
    numReplacedParental: number;
    dnsQueries: number[];
    blockedFiltering: number[];
    replacedSafebrowsing: number[];
    replacedParental: number[];
};

export const StatCards = ({
    numDnsQueries,
    numBlockedFiltering,
    numReplacedSafebrowsing,
    numReplacedParental,
    dnsQueries,
    blockedFiltering,
    replacedSafebrowsing,
    replacedParental,
}: Props) => {
    const blockedPercent = numDnsQueries > 0 ? (numBlockedFiltering / numDnsQueries) * 100 : 0;
    const threatsPercent = numDnsQueries > 0 ? (numReplacedSafebrowsing / numDnsQueries) * 100 : 0;
    const parentalPercent = numDnsQueries > 0 ? (numReplacedParental / numDnsQueries) * 100 : 0;

    return (
        <div className={s.statsCards}>
            <StatCard
                value={numDnsQueries}
                label={intl.getMessage('dns_query')}
                data={dnsQueries}
                color={CARDS_COLORS.QUERIES}
                cardTheme={CARDS_THEME.QUERIES}
                linkTo={RoutePath.QueryLog}
            />
            <StatCard
                value={numBlockedFiltering}
                label={intl.getMessage('ads_blocked_card')}
                data={blockedFiltering}
                color={CARDS_COLORS.ADS}
                percentValue={blockedPercent}
                cardTheme={CARDS_THEME.ADS}
                linkTo={RoutePath.QueryLog}
                query={{ status: QUERY_LOG_STATUS_FILTER.BLOCKED.QUERY }}
            />
            <StatCard
                value={numReplacedSafebrowsing}
                label={intl.getMessage('blocked_threats_chart')}
                data={replacedSafebrowsing}
                color={CARDS_COLORS.THREATS}
                percentValue={threatsPercent}
                cardTheme={CARDS_THEME.THREATS}
                linkTo={RoutePath.QueryLog}
                query={{ reason: QUERY_LOG_REASON_FILTER.BLOCKED_BY_THREATS.QUERY }}
            />
            <StatCard
                value={numReplacedParental}
                label={intl.getMessage('stats_adult')}
                data={replacedParental}
                color={CARDS_COLORS.ADULT}
                percentValue={parentalPercent}
                cardTheme={CARDS_THEME.ADULT}
                linkTo={RoutePath.QueryLog}
                query={{ reason: QUERY_LOG_REASON_FILTER.BLOCKED_BY_PARENTAL_CONTROL.QUERY }}
            />
        </div>
    );
};
