import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { Trans, withTranslation } from 'react-i18next';
import { LEASES_TABLE_DEFAULT_PAGE_SIZE } from '../../../helpers/constants';
import { sortIp } from '../../../helpers/helpers';

class Leases extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row o-hidden">
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
                        sortMethod: sortIp,
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
                pageSize={LEASES_TABLE_DEFAULT_PAGE_SIZE}
                showPageSizeOptions={false}
                showPagination={leases.length > LEASES_TABLE_DEFAULT_PAGE_SIZE}
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

export default withTranslation()(Leases);
