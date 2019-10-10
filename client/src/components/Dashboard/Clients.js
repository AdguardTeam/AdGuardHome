import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';
import { formatClientCell } from '../../helpers/formatClientCell';

const getClientsPercentColor = (percent) => {
    if (percent > 50) {
        return STATUS_COLORS.green;
    } else if (percent > 10) {
        return STATUS_COLORS.yellow;
    }
    return STATUS_COLORS.red;
};

const countCell = dnsQueries =>
    function cell(row) {
        const { value } = row;
        const percent = getPercent(dnsQueries, value);
        const percentColor = getClientsPercentColor(percent);

        return <Cell value={value} percent={percent} color={percentColor} />;
    };

const clientCell = (clients, autoClients, t) =>
    function cell(row) {
        const { value } = row;

        return (
            <div className="logs__row logs__row--overflow logs__row--column">
                {formatClientCell(value, clients, autoClients, t)}
            </div>
        );
    };

const Clients = ({
    t, refreshButton, topClients, subtitle, clients, autoClients, dnsQueries,
}) => (
    <Card
        title={t('top_clients')}
        subtitle={subtitle}
        bodyType="card-table"
        refresh={refreshButton}
    >
        <ReactTable
            data={topClients.map(({ name: ip, count }) => ({
                ip,
                count,
            }))}
            columns={[
                {
                    Header: 'IP',
                    accessor: 'ip',
                    sortMethod: (a, b) =>
                        parseInt(a.replace(/\./g, ''), 10) - parseInt(b.replace(/\./g, ''), 10),
                    Cell: clientCell(clients, autoClients, t),
                },
                {
                    Header: <Trans>requests_count</Trans>,
                    accessor: 'count',
                    minWidth: 180,
                    maxWidth: 200,
                    Cell: countCell(dnsQueries),
                },
            ]}
            showPagination={false}
            noDataText={t('no_clients_found')}
            minRows={6}
            defaultPageSize={100}
            className="-striped -highlight card-table-overflow"
        />
    </Card>
);

Clients.propTypes = {
    topClients: PropTypes.array.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    clients: PropTypes.array.isRequired,
    autoClients: PropTypes.array.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Clients);
