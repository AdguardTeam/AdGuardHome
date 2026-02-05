import React from 'react';

import intl from 'panel/common/intl';

import s from './StatCard.module.pcss';
import { StatCard, CARDS_THEME } from './StatCard';

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
                color="#7F7F7F"
                cardTheme={CARDS_THEME.QUERIES}
            />
            <StatCard
                value={numBlockedFiltering}
                label={intl.getMessage('ads_blocked_card')}
                data={blockedFiltering}
                color="#E07575"
                percentValue={blockedPercent}
                cardTheme={CARDS_THEME.ADS}
            />
            <StatCard
                value={numReplacedSafebrowsing}
                label={intl.getMessage('blocked_threats_chart')}
                data={replacedSafebrowsing}
                color="#F5A623"
                percentValue={threatsPercent}
                cardTheme={CARDS_THEME.THREATS}
            />
            <StatCard
                value={numReplacedParental}
                label={intl.getMessage('stats_adult')}
                data={replacedParental}
                color="#9B59B6"
                percentValue={parentalPercent}
                cardTheme={CARDS_THEME.ADULT}
            />
        </div>
    );
};
