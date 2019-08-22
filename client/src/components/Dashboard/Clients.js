import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent, getClientName } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

class Clients extends Component {
    getPercentColor = (percent) => {
        if (percent > 50) {
            return STATUS_COLORS.green;
        } else if (percent > 10) {
            return STATUS_COLORS.yellow;
        }
        return STATUS_COLORS.red;
    };

    columns = [
        {
            Header: 'IP',
            accessor: 'ip',
            Cell: ({ value }) => {
                const clientName =
                    getClientName(this.props.clients, value) ||
                    getClientName(this.props.autoClients, value);
                let client;

                if (clientName) {
                    client = (
                        <span>
                            {clientName} <small>({value})</small>
                        </span>
                    );
                } else {
                    client = value;
                }

                return (
                    <div className="logs__row logs__row--overflow">
                        <span className="logs__text" title={value}>
                            {client}
                        </span>
                    </div>
                );
            },
            sortMethod: (a, b) =>
                parseInt(a.replace(/\./g, ''), 10) - parseInt(b.replace(/\./g, ''), 10),
        },
        {
            Header: <Trans>requests_count</Trans>,
            accessor: 'count',
            Cell: ({ value }) => {
                const percent = getPercent(this.props.dnsQueries, value);
                const percentColor = this.getPercentColor(percent);

                return <Cell value={value} percent={percent} color={percentColor} />;
            },
        },
    ];

    render() {
        const {
            t, refreshButton, topClients, subtitle,
        } = this.props;

        return (
            <Card
                title={t('top_clients')}
                subtitle={subtitle}
                bodyType="card-table"
                refresh={refreshButton}
            >
                <ReactTable
                    data={topClients.map(item => ({
                        ip: item.name,
                        count: item.count,
                    }))}
                    columns={this.columns}
                    showPagination={false}
                    noDataText={t('no_clients_found')}
                    minRows={6}
                    className="-striped -highlight card-table-overflow"
                />
            </Card>
        );
    }
}

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
