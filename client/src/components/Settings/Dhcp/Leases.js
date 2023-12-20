import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { Trans, withTranslation } from 'react-i18next';
import { LEASES_TABLE_DEFAULT_PAGE_SIZE, MODAL_TYPE } from '../../../helpers/constants';
import { sortIp } from '../../../helpers/helpers';
import { toggleLeaseModal } from '../../../actions';

class Leases extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row o-hidden">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    convertToStatic = (data) => () => {
        const { dispatch } = this.props;
        dispatch(toggleLeaseModal({
            type: MODAL_TYPE.ADD_LEASE,
            config: data,
        }));
    }

    makeStatic = ({ row }) => {
        const { t, disabledLeasesButton } = this.props;
        return (
            <div className="logs__row logs__row--center">
                <button
                    type="button"
                    className="btn btn-icon btn-icon--green btn-outline-success btn-sm"
                    title={t('make_static')}
                    onClick={this.convertToStatic(row)}
                    disabled={disabledLeasesButton}
                >
                    <svg className="icons icon12">
                        <use xlinkHref="#plus" />
                    </svg>
                </button>
            </div>
        );
    }

    render() {
        const { leases, t } = this.props;
        return (
            <ReactTable
                data={leases || []}
                columns={[
                    {
                        Header: 'MAC',
                        accessor: 'mac',
                        minWidth: 180,
                        Cell: this.cellWrap,
                    }, {
                        Header: 'IP',
                        accessor: 'ip',
                        minWidth: 230,
                        Cell: this.cellWrap,
                        sortMethod: sortIp,
                    }, {
                        Header: <Trans>dhcp_table_hostname</Trans>,
                        accessor: 'hostname',
                        minWidth: 230,
                        Cell: this.cellWrap,
                    }, {
                        Header: <Trans>dhcp_table_expires</Trans>,
                        accessor: 'expires',
                        minWidth: 220,
                        Cell: this.cellWrap,
                    }, {
                        Header: <Trans>actions_table_header</Trans>,
                        Cell: this.makeStatic,
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
    dispatch: PropTypes.func,
    disabledLeasesButton: PropTypes.bool,
};

export default withTranslation()(connect(() => ({}), (dispatch) => ({ dispatch }))(Leases));
