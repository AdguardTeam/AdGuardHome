import React from 'react';

// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { LEASES_TABLE_DEFAULT_PAGE_SIZE, MODAL_TYPE } from '../../../../helpers/constants';

import { sortIp } from '../../../../helpers/helpers';

import Modal from './Modal';
import { addStaticLease, removeStaticLease, toggleLeaseModal, updateStaticLease } from '../../../../actions';

interface cellWrapProps {
    value: string;
}

const cellWrap = ({ value }: cellWrapProps) => (
    <div className="logs__row o-hidden">
        <span className="logs__text" title={value}>
            {value}
        </span>
    </div>
);

interface StaticLeasesProps {
    staticLeases: unknown[];
    isModalOpen: boolean;
    modalType: string;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    cidr: string;
    gatewayIp?: string;
}

const StaticLeases = ({
    isModalOpen,
    modalType,
    processingAdding,
    processingDeleting,
    processingUpdating,
    staticLeases,
    cidr,
    gatewayIp,
}: StaticLeasesProps) => {
    const [t] = useTranslation();
    const dispatch = useDispatch();

    const handleSubmit = (data: any) => {
        const { mac, ip, hostname } = data;

        if (modalType === MODAL_TYPE.EDIT_LEASE) {
            dispatch(updateStaticLease({ mac, ip, hostname }));
        } else {
            dispatch(addStaticLease({ mac, ip, hostname }));
        }
    };

    const handleDelete = (ip: any, mac: any, hostname = '') => {
        const name = hostname || ip;
        // eslint-disable-next-line no-alert
        if (window.confirm(t('delete_confirm', { key: name }))) {
            dispatch(
                removeStaticLease({
                    ip,
                    mac,
                    hostname,
                }),
            );
        }
    };

    return (
        <>
            <ReactTable
                data={staticLeases || []}
                columns={[
                    {
                        Header: 'MAC',
                        accessor: 'mac',
                        minWidth: 180,
                        Cell: cellWrap,
                    },
                    {
                        Header: 'IP',
                        accessor: 'ip',
                        minWidth: 230,
                        sortMethod: sortIp,
                        Cell: cellWrap,
                    },
                    {
                        Header: <Trans>dhcp_table_hostname</Trans>,
                        accessor: 'hostname',
                        minWidth: 230,
                        Cell: cellWrap,
                    },
                    {
                        Header: <Trans>actions_table_header</Trans>,
                        accessor: 'actions',
                        maxWidth: 150,
                        sortable: false,
                        resizable: false,
                        // eslint-disable-next-line react/display-name
                        Cell: (row: any) => {
                            const { ip, mac, hostname } = row.original;

                            return (
                                <div className="logs__row logs__row--center">
                                    <button
                                        type="button"
                                        className="btn btn-icon btn-outline-primary btn-sm mr-2"
                                        onClick={() =>
                                            dispatch(
                                                toggleLeaseModal({
                                                    type: MODAL_TYPE.EDIT_LEASE,
                                                    config: { ip, mac, hostname },
                                                }),
                                            )
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
                                        onClick={() => handleDelete(ip, mac, hostname)}
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
                modalType={modalType}
                handleSubmit={handleSubmit}
                processingAdding={processingAdding}
                cidr={cidr}
                gatewayIp={gatewayIp}
            />
        </>
    );
};

export default StaticLeases;
