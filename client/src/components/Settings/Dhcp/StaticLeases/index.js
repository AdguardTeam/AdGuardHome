import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { Trans, withTranslation } from 'react-i18next';
import { LEASES_TABLE_DEFAULT_PAGE_SIZE } from '../../../../helpers/constants';

import Modal from './Modal';

class StaticLeases extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row o-hidden">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    handleSubmit = (data) => {
        this.props.addStaticLease(data);
    };

    handleDelete = (ip, mac, hostname = '') => {
        const name = hostname || ip;
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('delete_confirm', { key: name }))) {
            this.props.removeStaticLease({ ip, mac, hostname });
        }
    };

    render() {
        const {
            isModalOpen,
            toggleLeaseModal,
            processingAdding,
            processingDeleting,
            staticLeases,
            t,
        } = this.props;
        return (
            <Fragment>
                <ReactTable
                    data={staticLeases || []}
                    columns={[
                        {
                            Header: 'MAC',
                            accessor: 'mac',
                            Cell: this.cellWrap,
                        },
                        {
                            Header: 'IP',
                            accessor: 'ip',
                            Cell: this.cellWrap,
                        },
                        {
                            Header: <Trans>dhcp_table_hostname</Trans>,
                            accessor: 'hostname',
                            Cell: this.cellWrap,
                        },
                        {
                            Header: <Trans>actions_table_header</Trans>,
                            accessor: 'actions',
                            maxWidth: 150,
                            Cell: (row) => {
                                const { ip, mac, hostname } = row.original;

                                return (
                                    <div className="logs__row logs__row--center">
                                        <button
                                            type="button"
                                            className="btn btn-icon btn-icon--green btn-outline-secondary btn-sm"
                                            title={t('delete_table_action')}
                                            disabled={processingDeleting}
                                            onClick={() => this.handleDelete(ip, mac, hostname)
                                            }
                                        >
                                            <svg className="icons">
                                                <use xlinkHref="#delete"/>
                                            </svg>
                                        </button>
                                    </div>
                                );
                            },
                        },
                    ]}
                    pageSize={LEASES_TABLE_DEFAULT_PAGE_SIZE}
                    showPageSizeOptions={false}
                    showPagination={staticLeases.length > LEASES_TABLE_DEFAULT_PAGE_SIZE}
                    noDataText={t('dhcp_static_leases_not_found')}
                    className="-striped -highlight card-table-overflow"
                    minRows={6}
                />
                <Modal
                    isModalOpen={isModalOpen}
                    toggleLeaseModal={toggleLeaseModal}
                    handleSubmit={this.handleSubmit}
                    processingAdding={processingAdding}
                />
            </Fragment>
        );
    }
}

StaticLeases.propTypes = {
    staticLeases: PropTypes.array.isRequired,
    isModalOpen: PropTypes.bool.isRequired,
    toggleLeaseModal: PropTypes.func.isRequired,
    removeStaticLease: PropTypes.func.isRequired,
    addStaticLease: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingDeleting: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(StaticLeases);
