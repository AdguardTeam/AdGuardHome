import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { withTranslation, Trans } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';
import DomainCell from './DomainCell';

import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

const CountCell = (totalBlocked) => (
    function cell(row) {
        const { value } = row;
        const percent = getPercent(totalBlocked, value);

        return (
            <Cell
                value={value}
                percent={percent}
                color={STATUS_COLORS.green}
            />
        );
    }
);

const UpstreamResponses = ({
    t,
    refreshButton,
    topUpstreamsResponses,
    dnsQueries,
    subtitle,
}) => (
    <Card
        title={t('top_upstreams')}
        subtitle={subtitle}
        bodyType="card-table"
        refresh={refreshButton}
    >
        <ReactTable
            data={topUpstreamsResponses.map(({ name: domain, count }) => ({
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
                    Cell: CountCell(dnsQueries),
                },
            ]}
            showPagination={false}
            noDataText={t('no_upstreams_data_found')}
            minRows={6}
            defaultPageSize={100}
            className="-highlight card-table-overflow--limited stats__table"
        />
    </Card>
);

UpstreamResponses.propTypes = {
    topUpstreamsResponses: PropTypes.array.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(UpstreamResponses);
