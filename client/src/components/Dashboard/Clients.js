import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import map from 'lodash/map';
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
    }

    columns = [{
        Header: 'IP',
        accessor: 'ip',
        Cell: ({ value }) => {
            const clientName = getClientName(this.props.clients, value);
            let client;

            if (clientName) {
                client = <span>{clientName} <small>({value})</small></span>;
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
        sortMethod: (a, b) => parseInt(a.replace(/\./g, ''), 10) - parseInt(b.replace(/\./g, ''), 10),
    }, {
        Header: <Trans>requests_count</Trans>,
        accessor: 'count',
        Cell: ({ value }) => {
            const percent = getPercent(this.props.dnsQueries, value);
            const percentColor = this.getPercentColor(percent);

            return (
                <Cell value={value} percent={percent} color={percentColor} />
            );
        },
    }];

    render() {
        const { t } = this.props;
        return (
            <Card title={ t('top_clients') } subtitle={ t('for_last_24_hours') } bodyType="card-table" refresh={this.props.refreshButton}>
                <ReactTable
                    data={map(this.props.topClients, (value, prop) => (
                        { ip: prop, count: value }
                    ))}
                    columns={this.columns}
                    showPagination={false}
                    noDataText={ t('no_clients_found') }
                    minRows={6}
                    className="-striped -highlight card-table-overflow"
                />
            </Card>
        );
    }
}

Clients.propTypes = {
    topClients: PropTypes.object.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    clients: PropTypes.array.isRequired,
    t: PropTypes.func,
};

export default withNamespaces()(Clients);
