import { createMemo } from 'solid-js';

import intl from 'panel/common/intl';
import { sortIp } from 'panel/helpers/helpers';
import type { AutoClient, NormalizedTopClients, WhoisInfo } from 'panel/initialState';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import theme from 'panel/lib/theme';

import { WhoisCell } from './WhoisCell';
import { TableEmptyState } from '../TableEmptyState/TableEmptyState';
import s from './RuntimeClientsTable.module.pcss';

type Props = {
    autoClients: AutoClient[];
    normalizedTopClients?: NormalizedTopClients;
    loading?: boolean;
};

export const RuntimeClientsTable = (props: Props) => {
    const pageSize = createMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.AUTO_CLIENTS_PAGE_SIZE) || undefined,
    );

    const columns = createMemo<TableColumn<AutoClient>[]>(() => [
        {
            key: 'ip',
            header: {
                text: intl.getMessage('ip_address'),
                className: s.headerCell,
            },
            accessor: 'ip',
            sortable: true,
            sortFn: sortIp,
            render: (value: string) => (
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('ip_address')}</span>

                    <div class={s.cellValue}>
                        <span class={theme.common.textOverflow} title={value}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('name_table_header')}</span>

                    <div class={s.cellValue}>
                        <span class={theme.common.textOverflow} title={value || '-'}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('source_label')}</span>

                    <div class={s.cellValue}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('whois')}</span>

                    <div class={s.cellValue}>
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
            accessor: (row: AutoClient) => props.normalizedTopClients?.auto[row.ip] || 0,
            sortable: true,
            render: (_value: unknown, row: AutoClient) => (
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('requests_table_header')}</span>

                    <div class={s.cellValue}>
                        <span>
                            {(props.normalizedTopClients?.auto[row.ip] || 0).toLocaleString()}
                        </span>
                    </div>
                </div>
            ),
        },
    ]);

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.AUTO_CLIENTS_PAGE_SIZE, newSize);
    };

    return (
        <Table<AutoClient>
            data={props.autoClients}
            class={s.table}
            columns={columns()}
            emptyTable={<TableEmptyState />}
            loading={props.loading}
            pageSize={pageSize()}
            onPageSizeChange={handlePageSizeChange}
            getRowId={(row) => row.ip}
        />
    );
};
