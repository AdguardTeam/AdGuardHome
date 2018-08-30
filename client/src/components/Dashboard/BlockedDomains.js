import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import map from 'lodash/map';

import Card from '../ui/Card';

const Clients = props => (
    <Card title="Top blocked domains" subtitle="in the last 3 minutes" bodyType="card-table" refresh={props.refreshButton}>
        <ReactTable
            data={map(props.topBlockedDomains, (value, prop) => (
                { ip: prop, domain: value }
            ))}
            columns={[{
                Header: 'IP',
                accessor: 'ip',
            }, {
                Header: 'Domain name',
                accessor: 'domain',
            }]}
            showPagination={false}
            noDataText="No domains found"
            minRows={6}
            className="-striped -highlight card-table-overflow"
        />
    </Card>
);

Clients.propTypes = {
    topBlockedDomains: PropTypes.object.isRequired,
    refreshButton: PropTypes.node,
};

export default Clients;
