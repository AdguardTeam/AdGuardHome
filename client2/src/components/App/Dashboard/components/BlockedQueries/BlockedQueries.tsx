import theme from 'Lib/theme';
import React, { FC, useContext, useState } from 'react';
import { PieChart, Pie, ResponsiveContainer, Sector, Cell } from 'recharts';

import Store from 'Store';

import s from './BlockedQueries.module.pcss';

interface BlockCardProps {
    ads: number;
    trackers: number;
    other: number;
}

const renderActiveShape = (props: any): any => {
    const {
        cx, cy, innerRadius, outerRadius, startAngle, endAngle,
        fill, payload, percent,
    } = props;
    return (
        <g>
            <text x={cx} y={cy - 11} dy={8} textAnchor="middle" fill={fill}>{payload.name}</text>
            <text x={cx} y={cy + 18} dy={8} fontSize={24} textAnchor="middle" >{Math.round(percent * 100)}%</text>
            <Sector
                cx={cx}
                cy={cy}
                innerRadius={innerRadius + 5}
                outerRadius={outerRadius + 5}
                startAngle={startAngle + 1}
                endAngle={endAngle - 1}
                fill={fill}
            />
        </g>
    );
};

const BlockedQueries: FC<BlockCardProps> = ({ ads, trackers, other }) => {
    const store = useContext(Store);
    const [activeIndex, setActiveIndex] = useState(0);
    const { ui: { intl } } = store;
    const data = [
        { name: intl.getMessage('other'), value: other, color: theme.chartColors.gray700 },
        { name: intl.getMessage('ads'), value: ads, color: theme.chartColors.red },
        { name: intl.getMessage('trackers'), value: trackers, color: theme.chartColors.orange },
    ];
    const onChart: any = (_: any, index: number) => {
        setActiveIndex(index);
    };
    return (
        <div className={s.container}>
            <div className={s.title}>{intl.getMessage('dashboard_blocked_queries')}</div>
            <div className={s.pie}>
                <ResponsiveContainer width="100%" height={190}>
                    <PieChart>
                        <Pie
                            activeIndex={activeIndex}
                            data={data}
                            dataKey="value"
                            nameKey="name"
                            innerRadius={60}
                            outerRadius={80}
                            activeShape={renderActiveShape}
                            onClick={onChart}
                        >
                            {data.map((entry, index) => (
                                <Cell key={`cell-${index}`} fill={entry.color} />
                            ))}
                        </Pie>
                    </PieChart>
                </ResponsiveContainer>
            </div>
        </div>
    );
};

export default BlockedQueries;
