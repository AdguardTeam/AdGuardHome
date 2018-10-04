import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import map from 'lodash/map';

import Card from '../ui/Card';

const QueriedDomains = props => (
    <Card title="Top queried domains" subtitle="in the last 24 hours" bodyType="card-table" refresh={props.refreshButton}>
        <ReactTable
            data={map(props.topQueriedDomains, (value, prop) => (
                { ip: prop, count: value }
            ))}
            columns={[{
                Header: 'IP',
                accessor: 'ip',
            }, {
                Header: 'Request count',
                accessor: 'count',
            }]}
            showPagination={false}
            noDataText="No domains found"
            minRows={6}
            className="-striped -highlight card-table-overflow"
        />
    </Card>
);

QueriedDomains.propTypes = {
    topQueriedDomains: PropTypes.object.isRequired,
    refreshButton: PropTypes.node,
};

export default QueriedDomains;
