import React from 'react';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import cn from 'clsx';

import s from './GeneralStatistics.module.pcss';
import { StatRow } from './StatRow';

type Props = {
    numDnsQueries: number;
    numBlockedFiltering: number;
    numReplacedSafebrowsing: number;
    numReplacedParental: number;
    numReplacedSafesearch: number;
    avgProcessingTime: number;
};

export const GeneralStatistics = ({
    numDnsQueries,
    numBlockedFiltering,
    numReplacedSafebrowsing,
    numReplacedParental,
    numReplacedSafesearch,
    avgProcessingTime,
}: Props) => {
    const blockedPercent = numDnsQueries > 0 ? (numBlockedFiltering / numDnsQueries) * 100 : 0;
    const safebrowsingPercent = numDnsQueries > 0 ? (numReplacedSafebrowsing / numDnsQueries) * 100 : 0;
    const parentalPercent = numDnsQueries > 0 ? (numReplacedParental / numDnsQueries) * 100 : 0;
    const safesearchPercent = numDnsQueries > 0 ? (numReplacedSafesearch / numDnsQueries) * 100 : 0;

    const hasStats = numDnsQueries > 0;

    return (
        <div className={s.card}>
            <div className={s.cardHeader}>
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('general_statistics')}</div>
            </div>

            {hasStats ? (
                <div className={s.tableRows}>
                    <StatRow
                        label={intl.getMessage('dns_queries')}
                        value={numDnsQueries}
                        icon="connections"
                        rowTheme="dnsQueries"
                        tooltip={intl.getMessage('dns_queries_tooltip')}
                        isTotal
                    />

                    <StatRow
                        label={intl.getMessage('ads_blocked')}
                        value={numBlockedFiltering}
                        percent={blockedPercent}
                        icon="adblocking"
                        rowTheme="adsBlocked"
                        tooltip={intl.getMessage('ads_blocked_tooltip')}
                    />

                    <StatRow
                        label={intl.getMessage('threats_blocked')}
                        value={numReplacedSafebrowsing}
                        percent={safebrowsingPercent}
                        icon="tracking"
                        rowTheme="threatsBlocked"
                        tooltip={intl.getMessage('threats_blocked_tooltip')}
                    />

                    <StatRow
                        label={intl.getMessage('adult_websites_blocked')}
                        value={numReplacedParental}
                        percent={parentalPercent}
                        icon="parental"
                        rowTheme="adultWebsitesBlocked"
                        tooltip={intl.getMessage('adult_websites_blocked_tooltip')}
                    />

                    <StatRow
                        label={intl.getMessage('safe_search_used')}
                        value={numReplacedSafesearch}
                        percent={safesearchPercent}
                        icon="search"
                        rowTheme="safeSearchUsed"
                        tooltip={intl.getMessage('safe_search_used_tooltip')}
                    />

                    <div className={s.rowDivider}></div>

                    <StatRow
                        label={intl.getMessage('average_time_processing')}
                        value={`${avgProcessingTime.toFixed(0)} ${intl.getMessage('milliseconds_abbreviation')}`}
                        isQueriesValue={false}
                        icon="time"
                        rowTheme="averageProcessingTime"
                        tooltip={intl.getMessage('average_time_processing_tooltip')}
                    />
                </div>
            ) : (
                <div className={s.emptyState}>
                    <Icon icon="not_found_search" className={s.emptyStateIcon} />
                    <div className={s.emptyStateText}>{intl.getMessage('no_stats_yet')}</div>
                </div>
            )}
        </div>
    );
};
