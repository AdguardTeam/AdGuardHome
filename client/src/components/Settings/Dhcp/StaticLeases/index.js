import React from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { LEASES_TABLE_DEFAULT_PAGE_SIZE } from '../../../../helpers/constants';
import { sortIp } from '../../../../helpers/helpers';
import Modal from './Modal';
import { addStaticLease, removeStaticLease } from '../../../../actions';

const cellWrap = ({ value }) => (
    <div className="logs__row o-hidden">
            <span className="logs__text" title={value}>
                {value}
            </span>
    </div>
);

const StaticLeases = ({
    isModalOpen,
    processingAdding,
    processingDeleting,
    staticLeases,
}) => {
    const [t] = useTranslation();
    const dispatch = useDispatch();

    const handleSubmit = (data) => {
        dispatch(addStaticLease(data));
    };

    const handleDelete = (ip, mac, hostname = '') => {
        const name = hostname || ip;
        // eslint-disable-next-line no-alert
        if (window.confirm(t('delete_confirm', { key: name }))) {
            dispatch(removeStaticLease({
                ip,
                mac,
                hostname,
            }));
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
                        Cell: cellWrap,
                    },
                    {
                        Header: 'IP',
                        accessor: 'ip',
                        sortMethod: sortIp,
                        Cell: cellWrap,
                    },
                    {
                        Header: <Trans>dhcp_table_hostname</Trans>,
                        accessor: 'hostname',
                        Cell: cellWrap,
                    },
                    {
                        Header: <Trans>actions_table_header</Trans>,
                        accessor: 'actions',
                        maxWidth: 150,
                        // eslint-disable-next-line react/display-name
                        Cell: (row) => {
                            const { ip, mac, hostname } = row.original;

                            return <div className="logs__row logs__row--center">
                                <button
                                        type="button"
                                        className="btn btn-icon btn-icon--green btn-outline-secondary btn-sm"
                                        title={t('delete_table_action')}
                                        disabled={processingDeleting}
                                        onClick={() => handleDelete(ip, mac, hostname)}
                                >
                                    <svg className="icons">
                                        <use xlinkHref="#delete" />
                                    </svg>
                                </button>
                            </div>;
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
                handleSubmit={handleSubmit}
                processingAdding={processingAdding}
            />
        </>
    );
};

StaticLeases.propTypes = {
    staticLeases: PropTypes.array.isRequired,
    isModalOpen: PropTypes.bool.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingDeleting: PropTypes.bool.isRequired,
};

cellWrap.propTypes = {
    value: PropTypes.string.isRequired,
};

export default StaticLeases;
