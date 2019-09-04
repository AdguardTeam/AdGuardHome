import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';
import DomainCell from './DomainCell';

import { STATUS_COLORS } from '../../helpers/constants';
import { getPercent } from '../../helpers/helpers';

const getQueriedPercentColor = (percent) => {
    if (percent > 10) {
        return STATUS_COLORS.red;
    } else if (percent > 5) {
        return STATUS_COLORS.yellow;
    }
    return STATUS_COLORS.green;
};

const countCell = dnsQueries =>
    function cell(row) {
        const { value } = row;
        const percent = getPercent(dnsQueries, value);
        const percentColor = getQueriedPercentColor(percent);

        return <Cell value={value} percent={percent} color={percentColor} />;
    };

const QueriedDomains = ({
    t, refreshButton, topQueriedDomains, subtitle, dnsQueries,
}) => (
    <Card
        title={t('stats_query_domain')}
        subtitle={subtitle}
        bodyType="card-table"
        refresh={refreshButton}
    >
        <ReactTable
            data={topQueriedDomains.map(({ name: domain, count }) => ({
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
            minRows={6}
            className="-striped -highlight card-table-overflow stats__table"
        />
    </Card>
);

QueriedDomains.propTypes = {
    topQueriedDomains: PropTypes.array.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(QueriedDomains);
