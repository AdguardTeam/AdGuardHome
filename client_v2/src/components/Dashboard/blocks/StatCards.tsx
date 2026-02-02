import React from 'react';
import cn from 'clsx';
import { AreaChart, Area, ResponsiveContainer, Tooltip, TooltipProps } from 'recharts';

import intl from 'panel/common/intl';
import { formatNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';

import s from '../Dashboard.module.pcss';

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

const CARDS_THEME = {
    QUERIES: 'queries',
    ADS: 'ads',
    THREATS: 'threats',
    ADULT: 'adult',
};

const formatDate = (date: Date): string => {
    const day = date.getDate();
    const month = date.toLocaleString('en', { month: 'short' });
    const year = date.getFullYear();
    return `${day} ${month} ${year}`;
};

const CustomTooltip = ({ active, payload }: TooltipProps<number, string>) => {
    if (active && payload && payload.length > 0) {
        const data = payload[0].payload;
        return (
            <div className={s.chartTooltip}>
                <div className={s.chartTooltipValue}>{formatNumber(data.value)}</div>
                <div className={s.chartTooltipDate}>{data.date}</div>
            </div>
        );
    }
    return null;
};

type StatCardProps = {
    value: number;
    label: string;
    data: number[];
    color: string;
    percentValue?: number;
    cardTheme: typeof CARDS_THEME[keyof typeof CARDS_THEME];
};

const StatCard = ({ value, label, data, color, percentValue, cardTheme }: StatCardProps) => {
    const chartData = data.map((v, i) => {
        const date = new Date();
        date.setDate(date.getDate() - (data.length - 1 - i));
        return { value: v, index: i, date: formatDate(date) };
    });
    const percent = percentValue !== undefined ? percentValue : 0;

    return (
        <div className={cn(s.statCard, {
            [s.statCardQueries]: cardTheme === CARDS_THEME.QUERIES,
            [s.statCardAds]: cardTheme === CARDS_THEME.ADS,
            [s.statCardThreats]: cardTheme === CARDS_THEME.THREATS,
            [s.statCardAdult]: cardTheme === CARDS_THEME.ADULT,
        })}>
            <div className={s.statCardInner}>
                <div className={s.statCardHeader}>
                    <div>
                        <div className={s.statCardValue}>{formatNumber(value)}</div>
                    </div>

                    {cardTheme !== CARDS_THEME.QUERIES && (
                        <div className={cn(theme.text.t1, s.statCardPercent)}>
                            {percent.toFixed(0)}%
                        </div>
                    )}
                </div>
                <div className={s.statCardChart}>
                    <ResponsiveContainer width="100%" height="100%" minHeight={16}>
                        <AreaChart data={chartData}>
                            <defs>
                                <linearGradient id={`gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                                    <stop offset="100%" stopColor={color} stopOpacity={0} />
                                </linearGradient>
                            </defs>
                            <Area
                                type="monotone"
                                dataKey="value"
                                stroke={color}
                                strokeWidth={1}
                                fill={`url(#gradient-${color})`}
                                isAnimationActive={false}
                            />
                            <Tooltip content={<CustomTooltip />} />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>
            </div>
            <div className={cn(theme.text.t3, s.statCardLabel)}>{label}</div>
        </div>
    );
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
