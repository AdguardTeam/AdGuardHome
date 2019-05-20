import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { Trans, withNamespaces } from 'react-i18next';

class StaticLeases extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row logs__row--overflow">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    render() {
        const { staticLeases, t } = this.props;
        return (
            <ReactTable
                data={staticLeases || []}
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
                    },
                ]}
                showPagination={false}
                noDataText={t('dhcp_leases_not_found')}
                minRows={6}
                className="-striped -highlight card-table-overflow"
            />
        );
    }
}

StaticLeases.propTypes = {
    staticLeases: PropTypes.array,
    t: PropTypes.func,
};

export default withNamespaces()(StaticLeases);
