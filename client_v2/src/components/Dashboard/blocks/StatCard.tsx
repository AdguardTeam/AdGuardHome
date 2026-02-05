import React from 'react';
import cn from 'clsx';
import { AreaChart, Area, ResponsiveContainer, Tooltip, TooltipProps } from 'recharts';

import { formatNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';

import s from './StatCard.module.pcss';

export const CARDS_THEME = {
    QUERIES: 'queries',
    ADS: 'ads',
    THREATS: 'threats',
    ADULT: 'adult',
};

const formatDate = (date: Date): string => {
    return date.toLocaleDateString(navigator.language, {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
    });
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

export type StatCardProps = {
    value: number;
    label: string;
    data: number[];
    color: string;
    percentValue?: number;
    cardTheme: typeof CARDS_THEME[keyof typeof CARDS_THEME];
};

export const StatCard = ({ value, label, data, color, percentValue, cardTheme }: StatCardProps) => {
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
                                strokeWidth={0}
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
