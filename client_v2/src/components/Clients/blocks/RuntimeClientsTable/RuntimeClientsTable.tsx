import React, { useMemo } from 'react';

import intl from 'panel/common/intl';
import { AutoClient, NormalizedTopClients, WhoisInfo } from 'panel/initialState';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table as ReactTable, TableColumn } from 'panel/common/ui/Table';
import theme from 'panel/lib/theme';

import { WhoisCell } from './WhoisCell';
import { TableEmptyState } from '../TableEmptyState/TableEmptyState';
import s from './RuntimeClientsTable.module.pcss';

type Props = {
    autoClients: AutoClient[];
    normalizedTopClients?: NormalizedTopClients;
    loading?: boolean;
};

export const RuntimeClientsTable = ({
    autoClients,
    normalizedTopClients,
    loading = false,
}: Props) => {
    const pageSize = useMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.AUTO_CLIENTS_PAGE_SIZE) || undefined,
        [],
    );

    const columns: TableColumn<AutoClient>[] = useMemo(
        () => [
            {
                key: 'ip',
                header: {
                    text: intl.getMessage('ip_address'),
                    className: s.headerCell,
                },
                accessor: 'ip',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('ip_address')}</span>

                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow} title={value}>
                                {value}
                            </span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'name',
                header: {
                    text: intl.getMessage('name_table_header'),
                    className: s.headerCell,
                },
                accessor: 'name',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('name_table_header')}</span>

                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow} title={value || '-'}>
                                {value || '-'}
                            </span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'source',
                header: {
                    text: intl.getMessage('source_label'),
                    className: s.headerCell,
                },
                accessor: 'source',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('source_label')}</span>

                        <div className={s.cellValue}>
                            <span>{value || '-'}</span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'whois',
                header: {
                    text: intl.getMessage('whois'),
                    className: s.headerCell,
                },
                accessor: 'whois_info',
                sortable: false,
                render: (value: WhoisInfo, row: AutoClient) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('whois')}</span>

                        <div className={s.cellValue}>
                            <WhoisCell whoisInfo={value} ip={row.ip} />
                        </div>
                    </div>
                ),
            },
            {
                key: 'requests',
                header: {
                    text: intl.getMessage('requests_table_header'),
                    className: s.headerCell,
                },
                accessor: (row: AutoClient) => normalizedTopClients?.auto[row.ip] || 0,
                sortable: true,
                render: (_value: unknown, row: AutoClient) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>
                            {intl.getMessage('requests_table_header')}
                        </span>

                        <div className={s.cellValue}>
                            <span>
                                {(normalizedTopClients?.auto[row.ip] || 0).toLocaleString()}
                            </span>
                        </div>
                    </div>
                ),
            },
        ],
        [normalizedTopClients],
    );

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.AUTO_CLIENTS_PAGE_SIZE, newSize);
    };

    const emptyTableContent = <TableEmptyState />;

    return (
        <ReactTable<AutoClient>
            data={autoClients}
            className={s.table}
            columns={columns}
            emptyTable={emptyTableContent}
            loading={loading}
            pageSize={pageSize}
            onPageSizeChange={handlePageSizeChange}
            getRowId={(row) => row.ip}
        />
    );
};
