import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { Trans, withNamespaces } from 'react-i18next';
import { SMALL_TABLE_DEFAULT_PAGE_SIZE } from '../../../helpers/constants';

class Leases extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row logs__row--overflow">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    render() {
        const { leases, t } = this.props;
        return (
            <ReactTable
                data={leases || []}
                columns={[
                    {
                        Header: 'MAC',
                        accessor: 'mac',
                        Cell: this.cellWrap,
                    }, {
                        Header: 'IP',
                        accessor: 'ip',
                        Cell: this.cellWrap,
                    }, {
                        Header: <Trans>dhcp_table_hostname</Trans>,
                        accessor: 'hostname',
                        Cell: this.cellWrap,
                    }, {
                        Header: <Trans>dhcp_table_expires</Trans>,
                        accessor: 'expires',
                        Cell: this.cellWrap,
                    },
                ]}
                pageSize={SMALL_TABLE_DEFAULT_PAGE_SIZE}
                showPageSizeOptions={false}
                showPagination={leases.length > SMALL_TABLE_DEFAULT_PAGE_SIZE}
                noDataText={t('dhcp_leases_not_found')}
                minRows={6}
                className="-striped -highlight card-table-overflow"
            />
        );
    }
}

Leases.propTypes = {
    leases: PropTypes.array,
    t: PropTypes.func,
};

export default withNamespaces()(Leases);
