import React from 'react';
import { Area, AreaChart, ResponsiveContainer, Tooltip, TooltipProps } from 'recharts';
import { useSelector } from 'react-redux';
import './Line.css';

import { RootState } from '../../initialState';
import { buildChartData, HistoryPoint } from './lineUtils';

interface LineProps {
    data: number[];
    color?: string;
    width?: number;
    height?: number;
}

const gradientId = (color: string) => `line-gradient-${color.replace('#', '')}`;

const CustomTooltip = ({ active, payload }: TooltipProps<number, string>) => {
    if (!active || !payload || payload.length === 0) {
        return null;
    }

    const point = payload[0].payload as HistoryPoint;

    return (
        <div className="line__tooltip">
            <strong className="line__tooltip-value">{point.value}</strong>
            <small className="line__tooltip-label">{point.label}</small>
        </div>
    );
};

const Line = ({ data, color = 'black' }: LineProps) => {
    const interval = useSelector((state: RootState) => state.stats.interval);
    const timeUnits = useSelector((state: RootState) => state.stats.timeUnits);
    const chartData = buildChartData(data, interval, timeUnits);

    return (
        <ResponsiveContainer width="100%" height="100%" minHeight={16}>
            <AreaChart data={chartData} margin={{ top: 8, right: 8, bottom: 12, left: 8 }}>
                <defs>
                    <linearGradient id={gradientId(color)} x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={color} stopOpacity={0.82} />
                        <stop offset="64%" stopColor={color} stopOpacity={0.6} />
                        <stop offset="92%" stopColor={color} stopOpacity={0.16} />
                        <stop offset="100%" stopColor={color} stopOpacity={0} />
                    </linearGradient>
                </defs>

                <Tooltip content={<CustomTooltip />} cursor={{ stroke: color, strokeWidth: 1 }} />

                <Area
                    type="monotone"
                    dataKey="value"
                    strokeWidth={1}
                    stroke={color}
                    fill={`url(#${gradientId(color)})`}
                    isAnimationActive={false}
                    activeDot={{ r: 4, fill: color }}
                />
            </AreaChart>
        </ResponsiveContainer>
    );
};

export default Line;
