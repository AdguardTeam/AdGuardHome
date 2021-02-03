import React, { FC, useContext } from 'react';
import { Progress } from 'antd';
import cn from 'classnames';
import { AreaChart, Area, ResponsiveContainer } from 'recharts';

import TopArrayEntry from 'Entities/TopArrayEntry';
import theme from 'Lib/theme';
import Store from 'Store';

import s from './TopDomains.module.pcss';

interface TopDomainsProps {
    title: string;
    overal: number;
    chartData: number[];
    tableData: TopArrayEntry[];
    color: string;
    useValueColor?: boolean;
}

const TopDomains: FC<TopDomainsProps> = (
    { title, overal, chartData, tableData, color, useValueColor },
) => {
    const store = useContext(Store);
    const { ui: { intl } } = store;
    const data = tableData.map((e) => {
        const [domain, value] = Object.entries(e.numberData)[0];
        return { domain, value };
    });

    return (
        <div className={s.container}>
            <div className={s.title}>{title}</div>
            <div className={s.content}>
                <div className={s.overal}>
                    {overal.toLocaleString('en')}
                    <ResponsiveContainer width="100%" height={45}>
                        <AreaChart data={chartData.map((n) => ({ name: 'data', value: n }))}>
                            <Area dataKey="value" stroke={color} fill={color} dot={false} strokeWidth={2} />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>
                <div className={s.table}>
                    <div className={cn(s.tableHeader, s.tableRow)}>
                        <div>
                            {intl.getMessage('domain')}
                        </div>
                        <div>
                            {intl.getMessage('all_queries')}
                        </div>
                        <div>
                            %
                        </div>
                    </div>
                    {data.map(({ domain, value }) => (
                        <div className={s.tableRow} key={domain}>
                            <div className={s.domain}>{domain}</div>
                            <div style={{ color: useValueColor ? color : 'initial' }}>{value}</div>
                            <Progress
                                percent={Math.round((value / overal) * 100)}
                                strokeLinecap="square"
                                strokeColor={theme.chartColors.gray700}
                                trailColor={theme.chartColors.gray300}
                            />
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default TopDomains;
