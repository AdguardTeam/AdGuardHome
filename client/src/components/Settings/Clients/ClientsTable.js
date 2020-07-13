import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import ReactTable from 'react-table';

import { MODAL_TYPE } from '../../../helpers/constants';
import { normalizeTextarea } from '../../../helpers/helpers';
import Card from '../../ui/Card';
import Modal from './Modal';
import CellWrap from '../../ui/CellWrap';
import LogsSearchLink from '../../ui/LogsSearchLink';

class ClientsTable extends Component {
    handleFormAdd = (values) => {
        this.props.addClient(values);
    };

    handleFormUpdate = (values, name) => {
        this.props.updateClient(values, name);
    };

    handleSubmit = (values) => {
        const config = values;

        if (values) {
            if (values.blocked_services) {
                config.blocked_services = Object
                    .keys(values.blocked_services)
                    .filter((service) => values.blocked_services[service]);
            }

            if (values.upstreams && typeof values.upstreams === 'string') {
                config.upstreams = normalizeTextarea(values.upstreams);
            } else {
                config.upstreams = [];
            }

            if (values.tags) {
                config.tags = values.tags.map((tag) => tag.value);
            } else {
                config.tags = [];
            }
        }

        if (this.props.modalType === MODAL_TYPE.EDIT_FILTERS) {
            this.handleFormUpdate(config, this.props.modalClientName);
        } else {
            this.handleFormAdd(config);
        }
    };

    getOptionsWithLabels = (options) => (
        options.map((option) => ({
            value: option,
            label: option,
        }))
    );

    getClient = (name, clients) => {
        const client = clients.find((item) => name === item.name);

        if (client) {
            const {
                upstreams, tags, whois_info, ...values
            } = client;
            return {
                upstreams: (upstreams && upstreams.join('\n')) || '',
                tags: (tags && this.getOptionsWithLabels(tags)) || [],
                ...values,
            };
        }

        return {
            ids: [''],
            tags: [],
            use_global_settings: true,
            use_global_blocked_services: true,
        };
    };

    handleDelete = (data) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('client_confirm_delete', { key: data.name }))) {
            this.props.deleteClient(data);
            this.props.getStats();
        }
    };

    columns = [
        {
            Header: this.props.t('table_client'),
            accessor: 'ids',
            minWidth: 150,
            Cell: (row) => {
                const { value } = row;

                return (
                    <div className="logs__row o-hidden">
                        <span className="logs__text">
                            {value.map((address) => (
                                <div key={address} title={address}>
                                    {address}
                                </div>
                            ))}
                        </span>
                    </div>
                );
            },
        },
        {
            Header: this.props.t('table_name'),
            accessor: 'name',
            minWidth: 120,
            Cell: CellWrap,
        },
        {
            Header: this.props.t('settings'),
            accessor: 'use_global_settings',
            minWidth: 120,
            Cell: ({ value }) => {
                const title = value ? (
                    <Trans>settings_global</Trans>
                ) : (
                    <Trans>settings_custom</Trans>
                );

                return (
                    <div className="logs__row o-hidden">
                        <div className="logs__text">{title}</div>
                    </div>
                );
            },
        },
        {
            Header: this.props.t('blocked_services'),
            accessor: 'blocked_services',
            minWidth: 180,
            Cell: (row) => {
                const { value, original } = row;

                if (original.use_global_blocked_services) {
                    return <Trans>settings_global</Trans>;
                }

                return (
                    <div className="logs__row logs__row--icons">
                        {value && value.length > 0
                            ? value.map((service) => (
                                <svg
                                    className="service__icon service__icon--table"
                                    title={service}
                                    key={service}
                                >
                                    <use xlinkHref={`#service_${service}`} />
                                </svg>
                            ))
                            : '–'}
                    </div>
                );
            },
        },
        {
            Header: this.props.t('upstreams'),
            accessor: 'upstreams',
            minWidth: 120,
            Cell: ({ value }) => {
                const title = value && value.length > 0 ? (
                    <Trans>settings_custom</Trans>
                ) : (
                    <Trans>settings_global</Trans>
                );

                return (
                    <div className="logs__row o-hidden">
                        <div className="logs__text">{title}</div>
                    </div>
                );
            },
        },
        {
            Header: this.props.t('tags_title'),
            accessor: 'tags',
            minWidth: 140,
            Cell: (row) => {
                const { value } = row;

                if (!value || value.length < 1) {
                    return '–';
                }

                return (
                    <div className="logs__row o-hidden">
                        <span className="logs__text">
                            {value.map((tag) => (
                                <div key={tag} title={tag} className="small">
                                    {tag}
                                </div>
                            ))}
                        </span>
                    </div>
                );
            },
        },
        {
            Header: this.props.t('requests_count'),
            id: 'statistics',
            accessor: (row) => this.props.normalizedTopClients.configured[row.name] || 0,
            sortMethod: (a, b) => b - a,
            minWidth: 120,
            Cell: (row) => {
                const content = CellWrap(row);

                if (!row.value) {
                    return content;
                }

                return <LogsSearchLink search={row.original.ids[0]}>{content}</LogsSearchLink>;
            },
        },
        {
            Header: this.props.t('actions_table_header'),
            accessor: 'actions',
            maxWidth: 100,
            Cell: (row) => {
                const clientName = row.original.name;
                const {
                    toggleClientModal, processingDeleting, processingUpdating, t,
                } = this.props;

                return (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-primary btn-sm mr-2"
                            onClick={() => toggleClientModal({
                                type: MODAL_TYPE.EDIT_FILTERS,
                                name: clientName,
                            })
                            }
                            disabled={processingUpdating}
                            title={t('edit_table_action')}
                        >
                            <svg className="icons">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-secondary btn-sm"
                            onClick={() => this.handleDelete({ name: clientName })}
                            disabled={processingDeleting}
                            title={t('delete_table_action')}
                        >
                            <svg className="icons">
                                <use xlinkHref="#delete" />
                            </svg>
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
            modalType,
            modalClientName,
            toggleClientModal,
            processingAdding,
            processingUpdating,
            supportedTags,
        } = this.props;

        const currentClientData = this.getClient(modalClientName, clients);
        const tagsOptions = this.getOptionsWithLabels(supportedTags);

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
                        defaultSorted={[
                            {
                                id: 'statistics',
                                asc: true,
                            },
                        ]}
                        className="-striped -highlight card-table-overflow"
                        showPagination
                        defaultPageSize={10}
                        minRows={5}
                        showPageSizeOptions={false}
                        showPageJump={false}
                        renderTotalPagesCount={() => false}
                        previousText={
                            <svg className="icons icon--small icon--gray w-100 h-100">
                                <use xlinkHref="#arrow-left" />
                            </svg>}
                        nextText={
                            <svg className="icons icon--small icon--gray w-100 h-100">
                                <use xlinkHref="#arrow-right" />
                            </svg>}
                        loadingText={t('loading_table_status')}
                        pageText=''
                        ofText=''
                        rowsText={t('rows_table_footer_text')}
                        noDataText={t('clients_not_found')}
                        getPaginationProps={() => ({ className: 'custom-pagination' })}
                    />
                    <button
                        type="button"
                        className="btn btn-success btn-standard mt-3"
                        onClick={() => toggleClientModal(MODAL_TYPE.ADD_FILTERS)}
                        disabled={processingAdding}
                    >
                        <Trans>client_add</Trans>
                    </button>
                    <Modal
                        isModalOpen={isModalOpen}
                        modalType={modalType}
                        toggleClientModal={toggleClientModal}
                        currentClientData={currentClientData}
                        handleSubmit={this.handleSubmit}
                        processingAdding={processingAdding}
                        processingUpdating={processingUpdating}
                        tagsOptions={tagsOptions}
                    />
                </Fragment>
            </Card>
        );
    }
}

ClientsTable.propTypes = {
    t: PropTypes.func.isRequired,
    clients: PropTypes.array.isRequired,
    normalizedTopClients: PropTypes.object.isRequired,
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
    getStats: PropTypes.func.isRequired,
    supportedTags: PropTypes.array.isRequired,
};

export default withTranslation()(ClientsTable);
