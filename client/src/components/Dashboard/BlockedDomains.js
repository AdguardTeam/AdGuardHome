import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import map from 'lodash/map';

import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

class BlockedDomains extends Component {
    columns = [{
        Header: 'IP',
        accessor: 'ip',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><span className="logs__text" title={value}>{value}</span></div>),
    }, {
        Header: 'Requests count',
        accessor: 'domain',
        Cell: ({ value }) => {
            const {
                blockedFiltering,
                replacedSafebrowsing,
                replacedParental,
            } = this.props;
            const blocked = blockedFiltering + replacedSafebrowsing + replacedParental;
            const percent = getPercent(blocked, value);

            return (
                <Cell value={value} percent={percent} color={STATUS_COLORS.red} />
            );
        },
    }];

    render() {
        return (
            <Card title="Top blocked domains" subtitle="for the last 24 hours" bodyType="card-table" refresh={this.props.refreshButton}>
                <ReactTable
                    data={map(this.props.topBlockedDomains, (value, prop) => (
                        { ip: prop, domain: value }
                    ))}
                    columns={this.columns}
                    showPagination={false}
                    noDataText="No domains found"
                    minRows={6}
                    className="-striped -highlight card-table-overflow"
                />
            </Card>
        );
    }
}

BlockedDomains.propTypes = {
    topBlockedDomains: PropTypes.object.isRequired,
    blockedFiltering: PropTypes.number.isRequired,
    replacedSafebrowsing: PropTypes.number.isRequired,
    replacedParental: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
};

export default BlockedDomains;
