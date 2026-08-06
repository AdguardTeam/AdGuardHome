import React from 'react';
import cn from 'clsx';

import { formatNumber } from '../../helpers/helpers';

import Card from '../ui/Card';
import Line from '../ui/Line';

import './StatsCard.css';

export const STATS_CARD_VARIANTS = {
    QUERIES: 'queries',
    ADS: 'ads',
    THREATS: 'threats',
    ADULT: 'adult',
} as const;

type StatsCardVariant = typeof STATS_CARD_VARIANTS[keyof typeof STATS_CARD_VARIANTS];

const CHART_COLORS: Record<StatsCardVariant, string> = {
    queries: '#7F7F7F',
    ads: '#F67247',
    threats: '#D58500',
    adult: '#A870B2',
};

type Props = {
    total: number;
    lineData: number[];
    title: React.ReactNode;
    variant: StatsCardVariant;
    percent?: number;
}

export const StatsCard = ({ total, lineData, percent, title, variant }: Props) => {
    const showPercent = typeof percent === 'number';
    const accentColor = CHART_COLORS[variant];

    return (
        <div className={cn('stats-card', `stats-card--${variant}`)}>
            <Card type="card--stats" bodyType="card-wrap">
                <div className="stats-card__inner">
                    <div className="stats-card__header">
                        <div className="stats-card__value">{formatNumber(total)}</div>

                        {showPercent && <div className="stats-card__percent">{Math.round(percent)}%</div>}
                    </div>

                    <div className="stats-card__chart">
                        <Line data={lineData} color={accentColor} />
                    </div>
                </div>
            </Card>

            <div className="stats-card__title">{title}</div>
        </div>
    );
};
