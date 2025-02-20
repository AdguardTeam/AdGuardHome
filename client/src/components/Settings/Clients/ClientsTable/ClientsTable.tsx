/* eslint-disable react/display-name */
/* eslint-disable react/prop-types */
import React, { useEffect } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';

import { useHistory, useLocation } from 'react-router-dom';

// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';

import { getAllBlockedServices, getBlockedServices } from '../../../../actions/services';

import { initSettings } from '../../../../actions';
import { splitByNewLine, countClientsStatistics, sortIp, getService } from '../../../../helpers/helpers';
import { MODAL_TYPE, LOCAL_TIMEZONE_VALUE, TABLES_MIN_ROWS } from '../../../../helpers/constants';

import Card from '../../../ui/Card';

import CellWrap from '../../../ui/CellWrap';

import LogsSearchLink from '../../../ui/LogsSearchLink';

import Modal from '../Modal';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from '../../../../helpers/localStorageHelper';
import { Client, NormalizedTopClients, RootState } from '../../../../initialState';

interface ClientsTableProps {
    clients: Client[];
    normalizedTopClients: NormalizedTopClients;
    toggleClientModal: (...args: unknown[]) => unknown;
    deleteClient: (...args: unknown[]) => string;
    addClient: (...args: unknown[]) => string;
    updateClient: (...args: unknown[]) => string;
    isModalOpen: boolean;
    modalType: string;
    modalClientName: string;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    getStats: (...args: unknown[]) => unknown;
    supportedTags: string[];
}

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
}: ClientsTableProps) => {
    const [t] = useTranslation();
    const dispatch = useDispatch();
    const location = useLocation();
    const history = useHistory();

    const services = useSelector((state: RootState) => state?.services);

    const globalSettings = useSelector((state: RootState) => state?.settings.settingsList);
    const params = new URLSearchParams(location.search);
    const clientId = params.get('clientId');

    useEffect(() => {
        dispatch(getAllBlockedServices());
        dispatch(getBlockedServices());
        dispatch(initSettings());

        if (clientId) {
            toggleClientModal({
                type: MODAL_TYPE.ADD_CLIENT,
            });
        }
    }, []);

    const handleFormAdd = (values: any) => {
        addClient(values);
    };

    const handleFormUpdate = (values: any, name: any) => {
        updateClient(values, name);
    };

    const handleSubmit = (values: any) => {
        const config = { ...values };

        if (values) {
            if (values.blocked_services) {
                config.blocked_services = Object.keys(values.blocked_services).filter(
                    (service) => values.blocked_services[service],
                );
            }

            if (values.upstreams && typeof values.upstreams === 'string') {
                config.upstreams = splitByNewLine(values.upstreams);
            } else {
                config.upstreams = [];
            }

            if (values.tags) {
                config.tags = values.tags.map((tag: any) => tag.value);
            } else {
                config.tags = [];
            }

            if (typeof values.upstreams_cache_size === 'string') {
                config.upstreams_cache_size = 0;
            }
        }

        if (modalType === MODAL_TYPE.EDIT_CLIENT) {
            handleFormUpdate(config, modalClientName);
        } else {
            handleFormAdd(config);
        }

        if (clientId) {
            history.push('/#clients');
        }
    };

    const getOptionsWithLabels = (options: any) =>
        options.map((option: any) => ({
            value: option,
            label: option,
        }));

    const getClient = (name: any, clients: any) => {
        const client = clients.find((item: any) => name === item.name);

        if (client) {
            const { upstreams, tags, ...values } = client;
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
            safe_search: { ...(globalSettings?.safesearch || {}) },
        };
    };

    const handleDelete = (data: any) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(t('client_confirm_delete', { key: data.name }))) {
            deleteClient(data);
            getStats();
        }
    };

    const handleClose = () => {
        toggleClientModal();

        if (clientId) {
            history.push('/#clients');
        }
    };

    const columns = [
        {
            Header: t('table_client'),
            accessor: 'ids',
            minWidth: 150,
            Cell: (row: any) => {
                const { value } = row;

                return (
                    <div className="logs__row o-hidden">
                        <span className="logs__text">
                            {value.map((address: any) => (
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
            Cell: ({ value }: any) => {
                const title = value ? <Trans>settings_global</Trans> : <Trans>settings_custom</Trans>;

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
            Cell: (row: any) => {
                const { value, original } = row;

                if (original.use_global_blocked_services) {
                    return <Trans>settings_global</Trans>;
                }

                if (value && services.allServices) {
                    return (
                        <div className="logs__row logs__row--icons">
                            {value.map((service: any) => {
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

                return <div className="logs__row logs__row--icons">–</div>;
            },
        },
        {
            Header: t('upstreams'),
            accessor: 'upstreams',
            minWidth: 120,
            Cell: ({ value }: any) => {
                const title =
                    value && value.length > 0 ? <Trans>settings_custom</Trans> : <Trans>settings_global</Trans>;

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
            Cell: (row: any) => {
                const { value } = row;

                if (!value || value.length < 1) {
                    return '–';
                }

                return (
                    <div className="logs__row o-hidden">
                        <span className="logs__text">
                            {value.map((tag: any) => (
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
            accessor: (row: any) => countClientsStatistics(row.ids, normalizedTopClients.auto),
            sortMethod: (a: any, b: any) => b - a,
            minWidth: 120,
            Cell: (row: any) => {
                const content = CellWrap(row);

                if (!row.value) {
                    return content;
                }

                return <LogsSearchLink search={row.original.name}>{content}</LogsSearchLink>;
            },
        },
        {
            Header: t('actions_table_header'),
            accessor: 'actions',
            maxWidth: 100,
            sortable: false,
            resizable: false,
            Cell: (row: any) => {
                const clientName = row.original.name;

                return (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-primary btn-sm mr-2"
                            onClick={() =>
                                toggleClientModal({
                                    type: MODAL_TYPE.EDIT_CLIENT,
                                    name: clientName,
                                })
                            }
                            disabled={processingUpdating}
                            title={t('edit_table_action')}>
                            <svg className="icons icon12">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>

                        <button
                            type="button"
                            className="btn btn-icon btn-outline-secondary btn-sm"
                            onClick={() => handleDelete({ name: clientName })}
                            disabled={processingDeleting}
                            title={t('delete_table_action')}>
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
        <Card title={t('clients_title')} subtitle={t('clients_desc')} bodyType="card-body box-body--settings">
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
                    onPageSizeChange={(size: any) =>
                        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE, size)
                    }
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
                    disabled={processingAdding}>
                    <Trans>client_add</Trans>
                </button>

                <Modal
                    isModalOpen={isModalOpen}
                    modalType={modalType}
                    handleClose={handleClose}
                    currentClientData={currentClientData}
                    handleSubmit={handleSubmit}
                    processingAdding={processingAdding}
                    processingUpdating={processingUpdating}
                    tagsOptions={tagsOptions}
                    clientId={clientId}
                />
            </>
        </Card>
    );
};

export default ClientsTable;
