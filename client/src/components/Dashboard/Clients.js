import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import map from 'lodash/map';

import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent } from '../../helpers/helpers';
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
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><span className="logs__text" title={value}>{value}</span></div>),
        sortMethod: (a, b) => {
            const nextValue = a.split('.');
            const prevValue = b.split('.');

            for (let i = 0; i < nextValue.length; i += 1) {
                const nextNumber = parseInt(nextValue[i], 10);
                const prevNumber = parseInt(prevValue[i], 10);

                if (nextNumber < prevNumber) {
                    return -1;
                } else if (nextNumber > prevNumber) {
                    return 1;
                }
            }
            return 0;
        },
    }, {
        Header: 'Requests count',
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
        return (
            <Card title="Top clients" subtitle="for the last 24 hours" bodyType="card-table" refresh={this.props.refreshButton}>
                <ReactTable
                    data={map(this.props.topClients, (value, prop) => (
                        { ip: prop, count: value }
                    ))}
                    columns={this.columns}
                    showPagination={false}
                    noDataText="No clients found"
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
};

export default Clients;
