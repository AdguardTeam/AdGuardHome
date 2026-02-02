import React from 'react';

import intl from 'panel/common/intl';
import { Icon, IconType } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import cn from 'clsx';
import { formatCompactNumber, formatNumber, msToDays, msToHours } from 'panel/helpers/helpers';
import { TIME_UNITS } from 'panel/helpers/constants';

import s from '../Dashboard.module.pcss';

type Props = {
    numDnsQueries: number;
    numBlockedFiltering: number;
    numReplacedSafebrowsing: number;
    numReplacedParental: number;
    numReplacedSafesearch: number;
    avgProcessingTime: number;
    interval: number;
    timeUnits: string;
};

type StatRowProps = {
    icon: IconType;
    label: string;
    value: string | number;
    percent?: number;
    isTotal?: boolean;
    isQueriesValue?: boolean;
    tooltip: string,
    rowTheme: 'dnsQueries' | 'adsBlocked' | 'threatsBlocked' | 'adultWebsitesBlocked' | 'safeSearchUsed' | 'averageProcessingTime';
};

const StatRow = ({ icon, label, value, percent, isTotal, isQueriesValue = true, tooltip, rowTheme }: StatRowProps) => (
    <div className={cn(s.statRow, s[rowTheme])}>
        <Dropdown
            trigger="hover"
            position="bottomLeft"
            noIcon
            disableAnimation
            overlayClassName={s.queryTooltipOverlay}
            menu={<div className={s.statTooltip}>{tooltip}</div>}
        >
            <div className={cn(theme.text.t3, theme.text.condenced, s.statRowLeft)}>
                {<Icon icon={icon} className={s.tableRowIcon} />}

                {label}
            </div>
        </Dropdown>

        <div className={s.statRowValue}>
            {isQueriesValue ? (
                <Dropdown
                    trigger="hover"
                    position="top"
                    noIcon
                    disableAnimation
                    overlayClassName={s.queryTooltipOverlay}
                    menu={
                        <div className={s.queryTooltip}>
                            {typeof value === 'number' ? formatNumber(value) : value} {intl.getMessage('queries').toLowerCase()}
                        </div>
                    }
                >
                    <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                        {typeof value === 'number' ? formatCompactNumber(value) : value}

                        <div className={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}>
                            {isTotal ? (
                                <span>({intl.getMessage('total')})</span>
                            ) : percent !== undefined && percent > 0 && (
                                <span>({percent.toFixed(1)}%)</span>
                            )}
                        </div>
                    </div>
                </Dropdown>
            ) : (
                <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                    {value}
                </div>
            )}

            {isQueriesValue && (
                <div className={s.queryBar}>
                    <div
                        className={s.queryBarFill}
                        style={{ width: `${isTotal ? 100 : (percent || 0)}%` }}
                    />
                </div>
            )}
        </div>
    </div>
);

export const GeneralStatistics = ({
    numDnsQueries,
    numBlockedFiltering,
    numReplacedSafebrowsing,
    numReplacedParental,
    numReplacedSafesearch,
    avgProcessingTime,
    interval,
    timeUnits,
}: Props) => {
    const blockedPercent = numDnsQueries > 0 ? (numBlockedFiltering / numDnsQueries) * 100 : 0;
    const safebrowsingPercent = numDnsQueries > 0 ? (numReplacedSafebrowsing / numDnsQueries) * 100 : 0;
    const parentalPercent = numDnsQueries > 0 ? (numReplacedParental / numDnsQueries) * 100 : 0;
    const safesearchPercent = numDnsQueries > 0 ? (numReplacedSafesearch / numDnsQueries) * 100 : 0;

    const getIntervalDescription = () => {
        if (timeUnits === TIME_UNITS.HOURS) {
            return intl.getMessage('number_of_dns_query_hours', { count: msToHours(interval) });
        }
        return intl.getMessage('number_of_dns_query_days', { count: msToDays(interval) });
    };

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
