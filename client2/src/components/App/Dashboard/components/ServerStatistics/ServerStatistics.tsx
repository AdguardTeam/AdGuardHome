import React, { FC, useContext } from 'react';
import { Row, Col } from 'antd';
import { AreaChart, Area, ResponsiveContainer } from 'recharts';

import Store from 'Store';
import theme from 'Lib/theme';

import s from './ServerStatistics.module.pcss';

const ServerStatistics: FC = () => {
    const store = useContext(Store);
    const { ui: { intl } } = store;

    const data = [0, 10, 2, 14, 12, 24, 5, 8, 10, 0, 3, 5, 7, 8, 3];
    return (
        <div className={s.container}>
            <div className={s.title}>{intl.getMessage('dashboard_server_statistics')}</div>
            <Row>
                <Col span={24} md={6} className={s.cardBorder}>
                    <div className={s.card}>
                        <div className={s.cardTitle}>
                            Average server load
                        </div>
                        <div className={s.cardDesc}>
                            <div>
                                Processes: 213
                            </div>
                            <div>
                                Cores: 2
                            </div>
                        </div>
                        <ResponsiveContainer width="100%" height={25} className={s.chart}>
                            <AreaChart data={data.map((n) => ({ name: 'data', value: n }))}>
                                <Area dataKey="value" stroke={theme.chartColors.green} fill={theme.chartColors.green} dot={false} />
                            </AreaChart>
                        </ResponsiveContainer>
                    </div>
                </Col>
                <Col span={24} md={6} className={s.cardBorder}>
                    <div className={s.card}>
                        <div className={s.cardTitle}>
                            Memory usage
                        </div>
                        <div className={s.cardValue}>
                            236 Mb
                        </div>
                        <ResponsiveContainer width="100%" height={25} className={s.chart}>
                            <AreaChart data={data.map((n) => ({ name: 'data', value: n }))}>
                                <Area dataKey="value" stroke={theme.chartColors.orange} fill={theme.chartColors.orange} dot={false} />
                            </AreaChart>
                        </ResponsiveContainer>
                    </div>
                </Col>
                <Col span={24} md={6} className={s.cardBorder}>
                    <div className={s.card}>
                        <div className={s.cardTitle}>
                            DNS cashe size
                        </div>
                        <div className={s.cardValue}>
                            2 363 records
                        </div>
                        <div className={s.cardDesc}>
                            <div>
                                32 Mb
                            </div>
                        </div>
                    </div>
                </Col>
                <Col span={24} md={6} className={s.cardBorder}>
                    <div className={s.card}>
                        <div className={s.cardTitle}>
                            Upstream servers data
                        </div>
                        <div className={s.cardDesc}>
                            <div>
                                Processes: 213
                            </div>
                            <div>
                                Cores: 2
                            </div>
                        </div>
                    </div>
                </Col>
            </Row>
        </div>
    );
};

export default ServerStatistics;
