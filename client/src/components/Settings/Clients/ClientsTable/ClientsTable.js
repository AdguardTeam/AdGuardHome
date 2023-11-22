/* eslint-disable react/display-name */
/* eslint-disable react/prop-types */
import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';
import ReactTable from 'react-table';

import { getAllBlockedServices, getBlockedServices } from '../../../../actions/services';
import { initSettings } from '../../../../actions';
import {
    splitByNewLine,
    countClientsStatistics,
    sortIp,
    getService,
} from '../../../../helpers/helpers';
import { MODAL_TYPE, LOCAL_TIMEZONE_VALUE, TABLES_MIN_ROWS } from '../../../../helpers/constants';
import Card from '../../../ui/Card';
import CellWrap from '../../../ui/CellWrap';
import LogsSearchLink from '../../../ui/LogsSearchLink';
import Modal from '../Modal';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from '../../../../helpers/localStorageHelper';

const ClientsTable = ({
    clients,
    normalizedTopClients,
    isModalOpen,
    modalClientName,
    modalType,
    addClient,
    updateClient,
    deleteClient,
    toggleClientModal,
    processingAdding,
    processingDeleting,
    processingUpdating,
    getStats,
    supportedTags,
}) => {
    const [t] = useTranslation();
    const dispatch = useDispatch();
    const services = useSelector((store) => store?.services);
    const globalSettings = useSelector((store) => store?.settings.settingsList) || {};

    const { safesearch } = globalSettings;

    useEffect(() => {
        dispatch(getAllBlockedServices());
        dispatch(getBlockedServices());
        dispatch(initSettings());
    }, []);

    const handleFormAdd = (values) => {
        addClient(values);
    };

    const handleFormUpdate = (values, name) => {
        updateClient(values, name);
    };

    const handleSubmit = (values) => {
        const config = { ...values };

        if (values) {
            if (values.blocked_services) {
                config.blocked_services = Object
                    .keys(values.blocked_services)
                    .filter((service) => values.blocked_services[service]);
            }

            if (values.upstreams && typeof values.upstreams === 'string') {
                config.upstreams = splitByNewLine(values.upstreams);
            } else {
                config.upstreams = [];
            }

            if (values.tags) {
                config.tags = values.tags.map((tag) => tag.value);
            } else {
                config.tags = [];
            }

            if (typeof values.upstreams_cache_size === 'string') {
                config.upstreams_cache_size = 0;
            }
        }

        if (modalType === MODAL_TYPE.EDIT_FILTERS) {
            handleFormUpdate(config, modalClientName);
        } else {
            handleFormAdd(config);
        }
    };

    const getOptionsWithLabels = (options) => (
        options.map((option) => ({
            value: option,
            label: option,
        }))
    );

    const getClient = (name, clients) => {
        const client = clients.find((item) => name === item.name);

        if (client) {
            const {
                upstreams, tags, whois_info, ...values
            } = client;
            return {
                upstreams: (upstreams && upstreams.join('\n')) || '',
                tags: (tags && getOptionsWithLabels(tags)) || [],
                ...values,
            };
        }

        return {
            ids: [''],
            tags: [],
            use_global_settings: true,
            use_global_blocked_services: true,
            blocked_services_schedule: {
                time_zone: LOCAL_TIMEZONE_VALUE,
            },
            safe_search: { ...(safesearch || {}) },
        };
    };

    const handleDelete = (data) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(t('client_confirm_delete', { key: data.name }))) {
            deleteClient(data);
            getStats();
        }
    };

    const columns = [
        {
            Header: t('table_client'),
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
            sortMethod: sortIp,
        },
        {
            Header: t('table_name'),
            accessor: 'name',
            minWidth: 120,
            Cell: CellWrap,
        },
        {
            Header: t('settings'),
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
            Header: t('blocked_services'),
            accessor: 'blocked_services',
            minWidth: 180,
            Cell: (row) => {
                const { value, original } = row;

                if (original.use_global_blocked_services) {
                    return <Trans>settings_global</Trans>;
                }

                if (value && services.allServices) {
                    return (
                        <div className="logs__row logs__row--icons">
                            {value.map((service) => {
                                const serviceInfo = getService(services.allServices, service);

                                if (serviceInfo?.icon_svg) {
                                    return (
                                        <div
                                            key={serviceInfo.name}
                                            dangerouslySetInnerHTML={{
                                                __html: window.atob(serviceInfo.icon_svg),
                                            }}
                                            className="service__icon service__icon--table"
                                            title={serviceInfo.name}
                                        />
                                    );
                                }

                                return null;
                            })}
                        </div>
                    );
                }

                return (
                    <div className="logs__row logs__row--icons">
                        –
                    </div>
                );
            },
        },
        {
            Header: t('upstreams'),
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
            Header: t('tags_title'),
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
                                <div key={tag} title={tag} className="logs__tag small">
                                    {tag}
                                </div>
                            ))}
                        </span>
                    </div>
                );
            },
        },
        {
            Header: t('requests_count'),
            id: 'statistics',
            accessor: (row) => countClientsStatistics(
                row.ids,
                normalizedTopClients.auto,
            ),
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
            Header: t('actions_table_header'),
            accessor: 'actions',
            maxWidth: 100,
            sortable: false,
            resizable: false,
            Cell: (row) => {
                const clientName = row.original.name;

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
                            <svg className="icons icon12">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-secondary btn-sm"
                            onClick={() => handleDelete({ name: clientName })}
                            disabled={processingDeleting}
                            title={t('delete_table_action')}
                        >
                            <svg className="icons icon12">
                                <use xlinkHref="#delete" />
                            </svg>
                        </button>
                    </div>
                );
            },
        },
    ];

    const currentClientData = getClient(modalClientName, clients);
    const tagsOptions = getOptionsWithLabels(supportedTags);

    return (
        <Card
            title={t('clients_title')}
            subtitle={t('clients_desc')}
            bodyType="card-body box-body--settings"
        >
            <>
                <ReactTable
                    data={clients || []}
                    columns={columns}
                    defaultSorted={[
                        {
                            id: 'statistics',
                            asc: true,
                        },
                    ]}
                    className="-striped -highlight card-table-overflow"
                    showPagination
                    defaultPageSize={LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE) || 10}
                    onPageSizeChange={(size) => (
                        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE, size)
                    )}
                    minRows={TABLES_MIN_ROWS}
                    ofText="/"
                    previousText={t('previous_btn')}
                    nextText={t('next_btn')}
                    pageText={t('page_table_footer_text')}
                    rowsText={t('rows_table_footer_text')}
                    loadingText={t('loading_table_status')}
                    noDataText={t('clients_not_found')}
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
                    handleSubmit={handleSubmit}
                    processingAdding={processingAdding}
                    processingUpdating={processingUpdating}
                    tagsOptions={tagsOptions}
                />
            </>
        </Card>
    );
};

ClientsTable.propTypes = {
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

export default ClientsTable;
