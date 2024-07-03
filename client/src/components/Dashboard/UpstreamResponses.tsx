import React from 'react';

// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';
import { withTranslation, Trans } from 'react-i18next';

import { TFunction } from 'i18next';
import Card from '../ui/Card';

import Cell from '../ui/Cell';

import DomainCell from './DomainCell';

import { getPercent } from '../../helpers/helpers';
import { DASHBOARD_TABLES_DEFAULT_PAGE_SIZE, STATUS_COLORS, TABLES_MIN_ROWS } from '../../helpers/constants';

const CountCell = (totalBlocked: any) =>
    function cell(row: any) {
        const { value } = row;
        const percent = getPercent(totalBlocked, value);

        return <Cell value={value} percent={percent} color={STATUS_COLORS.green} />;
    };

const getTotalUpstreamRequests = (stats: any) => {
    let total = 0;
    stats.forEach(({ count }: any) => {
        total += count;
    });

    return total;
};

interface UpstreamResponsesProps {
    topUpstreamsResponses: { name: string; count: number }[];
    refreshButton: React.ReactNode;
    subtitle: string;
    t: TFunction;
}

const UpstreamResponses = ({ t, refreshButton, topUpstreamsResponses, subtitle }: UpstreamResponsesProps) => (
    <Card title={t('top_upstreams')} subtitle={subtitle} bodyType="card-table" refresh={refreshButton}>
        <ReactTable
            data={topUpstreamsResponses.map(({ name: domain, count }: { name: string; count: number }) => ({
                domain,
                count,
            }))}
            columns={[
                {
                    Header: <Trans>upstream</Trans>,
                    accessor: 'domain',
                    Cell: DomainCell,
                },
                {
                    Header: <Trans>requests_count</Trans>,
                    accessor: 'count',
                    maxWidth: 190,
                    Cell: CountCell(getTotalUpstreamRequests(topUpstreamsResponses)),
                },
            ]}
            showPagination={false}
            noDataText={t('no_upstreams_data_found')}
            minRows={TABLES_MIN_ROWS}
            defaultPageSize={DASHBOARD_TABLES_DEFAULT_PAGE_SIZE}
            className="-highlight card-table-overflow--limited stats__table"
        />
    </Card>
);

export default withTranslation()(UpstreamResponses);
