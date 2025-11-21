import React, { useMemo } from 'react';
import round from 'lodash/round';
import { withTranslation, Trans } from 'react-i18next';
import { TFunction } from 'i18next';

import Card from '../ui/Card';
import DomainCell from './DomainCell';
import { Table, convertColumns } from '../ui/ReactTable';

import { DASHBOARD_TABLES_DEFAULT_PAGE_SIZE, TABLES_MIN_ROWS } from '../../helpers/constants';
import { formatNumber } from '../../helpers/helpers';

interface TimeCellProps {
    value?: string | number;
}

const TimeCell = ({ value }: TimeCellProps) => {
    if (!value) {
        return '–';
    }

    const valueInMilliseconds = formatNumber(round(Number(value) * 1000));

    return (
        <div className="logs__row o-hidden">
            <span className="logs__text logs__text--full" title={valueInMilliseconds.toString()}>
                {valueInMilliseconds}&nbsp;ms
            </span>
        </div>
    );
};

interface UpstreamAvgTimeProps {
    topUpstreamsAvgTime: { name: string; count: number }[];
    refreshButton: React.ReactNode;
    subtitle: string;
    t: TFunction;
}

const UpstreamAvgTime = ({ t, refreshButton, topUpstreamsAvgTime, subtitle }: UpstreamAvgTimeProps) => {
    const columns = useMemo(
        () =>
            convertColumns([
                {
                    Header: <Trans>upstream</Trans>,
                    accessor: 'domain',
                    Cell: DomainCell,
                },
                {
                    Header: <Trans>response_time</Trans>,
                    accessor: 'count',
                    maxWidth: 190,
                    Cell: TimeCell,
                },
            ]),
        []
    );

    const tableData = useMemo(
        () =>
            topUpstreamsAvgTime.map(({ name: domain, count }: { name: string; count: number }) => ({
                domain,
                count,
            })),
        [topUpstreamsAvgTime]
    );

    return (
        <Card title={t('average_upstream_response_time')} subtitle={subtitle} bodyType="card-table" refresh={refreshButton}>
            <Table
                data={tableData}
                columns={columns}
                showPagination={false}
                noDataText={t('no_upstreams_data_found')}
                minRows={TABLES_MIN_ROWS}
                defaultPageSize={DASHBOARD_TABLES_DEFAULT_PAGE_SIZE}
                className="-highlight card-table-overflow--limited stats__table"
            />
        </Card>
    );
};

export default withTranslation()(UpstreamAvgTime);
