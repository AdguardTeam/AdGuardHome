import React from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { withNamespaces } from 'react-i18next';

const columns = [{
    Header: 'MAC',
    accessor: 'mac',
}, {
    Header: 'IP',
    accessor: 'ip',
}, {
    Header: 'Hostname',
    accessor: 'hostname',
}, {
    Header: 'Expires',
    accessor: 'expires',
}];

const Leases = props => (
    <ReactTable
        data={props.leases}
        columns={columns}
        showPagination={false}
        noDataText={ props.t('dhcp_leases_not_found') }
        minRows={6}
        className="-striped -highlight card-table-overflow"
    />
);

Leases.propTypes = {
    leases: PropTypes.array,
    t: PropTypes.func,
};

export default withNamespaces()(Leases);
