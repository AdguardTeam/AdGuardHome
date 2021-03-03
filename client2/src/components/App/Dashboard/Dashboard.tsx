import React, { FC, useContext } from 'react';
import { Row, Col } from 'antd';
import { observer } from 'mobx-react-lite';

import Store from 'Store';
import { InnerLayout } from 'Common/ui/layouts';
import theme from 'Lib/theme';
import { BlockCard, TopDomains, BlockedQueries, TopClients, ServerStatistics } from './components';

const Dashboard:FC = observer(() => {
    const store = useContext(Store);
    const {
        dashboard: { stats, filteringConfig },
        system: { status },
        ui: { intl },
    } = store;

    if (!stats || !filteringConfig) {
        return null;
    }

    const {
        numBlockedFiltering,
        numReplacedParental,
        numReplacedSafebrowsing,
        replacedParental,
        replacedSafebrowsing,
        avgProcessingTime,
        blockedFiltering,

        topBlockedDomains,
        topQueriedDomains,
        dnsQueries,
        numDnsQueries,

    } = stats;

    const { filters } = filteringConfig!;
    const allFilters = filters?.length;
    const allRules = filters?.reduce((prev, e) => prev + (e.rulesCount || 0), 0);
    const enabled = filters?.filter((e) => e.enabled).length;

    return (
        <InnerLayout title={`AdGuard Home ${status?.version}`}>
            <div className={theme.content.container}>
                <Row gutter={[24, 24]}>
                    <Col span={24} md={12}>
                        <TopDomains
                            title={intl.getMessage('stats_query_domain')}
                            overal={numDnsQueries!}
                            chartData={dnsQueries!}
                            tableData={topQueriedDomains!}
                            color={theme.chartColors.green}
                        />
                    </Col>
                    <Col span={24} md={12}>
                        <TopDomains
                            useValueColor
                            title={intl.getMessage('top_blocked_domains')}
                            overal={numBlockedFiltering!}
                            chartData={blockedFiltering!}
                            tableData={topBlockedDomains!}
                            color={theme.chartColors.red}
                        />
                    </Col>
                </Row>
                <Row gutter={[24, 24]}>
                    <Col span={24} md={18}>
                        <Row gutter={[24, 24]}>
                            <Col span={24} md={8}>
                                <BlockCard
                                    title={intl.getMessage('dashboard_blocked_ads')}
                                    overal={numBlockedFiltering!}
                                    data={blockedFiltering!}
                                    color={theme.chartColors.red}
                                />
                            </Col>
                            <Col span={24} md={8}>
                                <BlockCard
                                    title={intl.getMessage('dashboard_blocked_trackers')}
                                    overal={numBlockedFiltering!}
                                    data={blockedFiltering!}
                                    color={theme.chartColors.orange}
                                />
                            </Col>
                            <Col span={24} md={8}>
                                <BlockCard
                                    title={intl.getMessage('stats_adult')}
                                    overal={numReplacedParental!}
                                    data={replacedParental!}
                                    color={theme.chartColors.purple}
                                />
                            </Col>
                            <Col span={24} md={8}>
                                <BlockCard
                                    title={intl.getMessage('stats_malware_phishing')}
                                    overal={numReplacedSafebrowsing!}
                                    data={replacedSafebrowsing!}
                                    color={theme.chartColors.red}
                                />
                            </Col>
                            <Col span={24} md={8}>
                                <BlockCard
                                    title={intl.getMessage('average_processing_time')}
                                    overal={`${Math.round(avgProcessingTime! * 100)} ${intl.getMessage('milliseconds_abbreviation')}`}
                                    data={blockedFiltering!}
                                    color={theme.chartColors.green}
                                />
                            </Col>
                            <Col span={24} md={8}>
                                <BlockCard
                                    title={intl.getMessage('dashboard_filter_rules')}
                                    overal={allRules!}
                                    text={intl.getMessage('dashboard_filter_rules_count', { enabled, all: allFilters })}
                                    color={theme.chartColors.green}
                                />
                            </Col>
                        </Row>
                    </Col>
                    <Col span={24} md={6}>
                        {/* TODO: fix chart */}
                        <BlockedQueries
                            other={numBlockedFiltering! / 3}
                            ads={numBlockedFiltering!}
                            trackers={numBlockedFiltering!}
                        />
                    </Col>
                </Row>
                <TopClients />
                <ServerStatistics />
            </div>
        </InnerLayout>
    );
});

export default Dashboard;
