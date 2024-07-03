import React from 'react';

// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';
import { withTranslation, Trans } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';
import DomainCell from './DomainCell';

import { DASHBOARD_TABLES_DEFAULT_PAGE_SIZE, STATUS_COLORS, TABLES_MIN_ROWS } from '../../helpers/constants';

import { getPercent } from '../../helpers/helpers';

const getQueriedPercentColor = (percent: any) => {
    if (percent > 10) {
        return STATUS_COLORS.red;
    }
    if (percent > 5) {
        return STATUS_COLORS.yellow;
    }
    return STATUS_COLORS.green;
};

const countCell = (dnsQueries: any) =>
    function cell(row: any) {
        const { value } = row;
        const percent = getPercent(dnsQueries, value);
        const percentColor = getQueriedPercentColor(percent);

        return <Cell value={value} percent={percent} color={percentColor} search={row.original.domain} />;
    };

interface QueriedDomainsProps {
    topQueriedDomains: unknown[];
    dnsQueries: number;
    refreshButton: React.ReactNode;
    subtitle: string;
    t: (...args: unknown[]) => string;
}

const QueriedDomains = ({ t, refreshButton, topQueriedDomains, subtitle, dnsQueries }: QueriedDomainsProps) => (
    <Card title={t('stats_query_domain')} subtitle={subtitle} bodyType="card-table" refresh={refreshButton}>
        <ReactTable
            data={topQueriedDomains.map(({ name: domain, count }: any) => ({
                domain,
                count,
            }))}
            columns={[
                {
                    Header: <Trans>domain</Trans>,
                    accessor: 'domain',
                    Cell: DomainCell,
                },
                {
                    Header: <Trans>requests_count</Trans>,
                    accessor: 'count',
                    maxWidth: 190,
                    Cell: countCell(dnsQueries),
                },
            ]}
            showPagination={false}
            noDataText={t('no_domains_found')}
            minRows={TABLES_MIN_ROWS}
            defaultPageSize={DASHBOARD_TABLES_DEFAULT_PAGE_SIZE}
            className="-highlight card-table-overflow--limited stats__table"
        />
    </Card>
);

export default withTranslation()(QueriedDomains);
