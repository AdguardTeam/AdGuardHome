import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import ReactTable from 'react-table';

import { MODAL_TYPE, CLIENT_ID } from '../../../helpers/constants';
import Card from '../../ui/Card';
import Modal from './Modal';

class Clients extends Component {
    handleFormAdd = (values) => {
        this.props.addClient(values);
    };

    handleFormUpdate = (values, name) => {
        this.props.updateClient(values, name);
    };

    handleSubmit = (values) => {
        if (this.props.modalType === MODAL_TYPE.EDIT) {
            this.handleFormUpdate(values, this.props.modalClientName);
        } else {
            this.handleFormAdd(values);
        }
    };

    cellWrap = ({ value }) => (
        <div className="logs__row logs__row--overflow">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    getClient = (name, clients) => {
        const client = clients.find(item => name === item.name);

        if (client) {
            const identifier = client.mac ? CLIENT_ID.MAC : CLIENT_ID.IP;

            return {
                identifier,
                use_global_settings: true,
                ...client,
            };
        }

        return {
            identifier: 'ip',
            use_global_settings: true,
        };
    };

    getStats = (ip, stats) => {
        if (stats && stats.top_clients) {
            return stats.top_clients[ip];
        }

        return '';
    };

    columns = [
        {
            Header: this.props.t('table_client'),
            accessor: 'ip',
            Cell: (row) => {
                if (row.value) {
                    return (
                        <div className="logs__row logs__row--overflow">
                            <span className="logs__text" title={row.value}>
                                {row.value} <em>(IP)</em>
                            </span>
                        </div>
                    );
                } else if (row.original && row.original.mac) {
                    return (
                        <div className="logs__row logs__row--overflow">
                            <span className="logs__text" title={row.original.mac}>
                                {row.original.mac} <em>(MAC)</em>
                            </span>
                        </div>
                    );
                }

                return '';
            },
        },
        {
            Header: this.props.t('table_name'),
            accessor: 'name',
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('settings'),
            accessor: 'use_global_settings',
            maxWidth: 180,
            minWidth: 150,
            Cell: ({ value }) => {
                const title = value ? (
                    <Trans>settings_global</Trans>
                ) : (
                    <Trans>settings_custom</Trans>
                );

                return (
                    <div className="logs__row logs__row--overflow">
                        <div className="logs__text" title={title}>
                            {title}
                        </div>
                    </div>
                );
            },
        },
        {
            Header: this.props.t('table_statistics'),
            accessor: 'statistics',
            Cell: (row) => {
                const clientIP = row.original.ip;
                const clientStats = clientIP && this.getStats(clientIP, this.props.topStats);

                if (clientStats) {
                    return (
                        <div className="logs__row">
                            <div className="logs__text" title={clientStats}>
                                {clientStats}
                            </div>
                        </div>
                    );
                }

                return '–';
            },
        },
        {
            Header: this.props.t('actions_table_header'),
            accessor: 'actions',
            maxWidth: 220,
            minWidth: 150,
            Cell: (row) => {
                const clientName = row.original.name;
                const {
                    toggleClientModal,
                    deleteClient,
                    processingDeleting,
                    processingUpdating,
                } = this.props;

                return (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-outline-primary btn-sm mr-2"
                            onClick={() =>
                                toggleClientModal({
                                    type: MODAL_TYPE.EDIT,
                                    name: clientName,
                                })
                            }
                            disabled={processingUpdating}
                        >
                            <Trans>edit_table_action</Trans>
                        </button>
                        <button
                            type="button"
                            className="btn btn-outline-secondary btn-sm"
                            onClick={() => deleteClient({ name: clientName })}
                            disabled={processingDeleting}
                        >
                            <Trans>delete_table_action</Trans>
                        </button>
                    </div>
                );
            },
        },
    ];

    render() {
        const {
            t,
            clients,
            isModalOpen,
            modalClientName,
            toggleClientModal,
            processingAdding,
            processingUpdating,
        } = this.props;

        const currentClientData = this.getClient(modalClientName, clients);

        return (
            <Card
                title={t('clients_title')}
                subtitle={t('clients_desc')}
                bodyType="card-body box-body--settings"
            >
                <Fragment>
                    <ReactTable
                        data={clients || []}
                        columns={this.columns}
                        className="-striped -highlight card-table-overflow"
                        showPagination={true}
                        defaultPageSize={10}
                        minRows={5}
                        resizable={false}
                        previousText={t('previous_btn')}
                        nextText={t('next_btn')}
                        loadingText={t('loading_table_status')}
                        pageText={t('page_table_footer_text')}
                        ofText={t('of_table_footer_text')}
                        rowsText={t('rows_table_footer_text')}
                        noDataText={t('clients_not_found')}
                    />
                    <button
                        type="button"
                        className="btn btn-success btn-standard mt-3"
                        onClick={() => toggleClientModal(MODAL_TYPE.ADD)}
                        disabled={processingAdding}
                    >
                        <Trans>add_client</Trans>
                    </button>

                    <Modal
                        isModalOpen={isModalOpen}
                        toggleClientModal={toggleClientModal}
                        currentClientData={currentClientData}
                        handleSubmit={this.handleSubmit}
                        processingAdding={processingAdding}
                        processingUpdating={processingUpdating}
                    />
                </Fragment>
            </Card>
        );
    }
}

Clients.propTypes = {
    t: PropTypes.func.isRequired,
    clients: PropTypes.array.isRequired,
    topStats: PropTypes.object.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    deleteClient: PropTypes.func.isRequired,
    addClient: PropTypes.func.isRequired,
    updateClient: PropTypes.func.isRequired,
    isModalOpen: PropTypes.bool.isRequired,
    modalType: PropTypes.string.isRequired,
    modalClientName: PropTypes.string.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingDeleting: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
};

export default withNamespaces()(Clients);
