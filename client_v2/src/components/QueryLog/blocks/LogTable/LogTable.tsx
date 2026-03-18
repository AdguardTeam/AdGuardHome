import React, { useMemo, useCallback } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Table, TableColumn } from 'panel/common/ui/Table/Table';

import { LogEntry, Service } from 'panel/components/QueryLog/types';
import { isBlockedReason } from 'panel/components/QueryLog/helpers';
import { Icon } from 'panel/common/ui/Icon';

import { Filter } from 'panel/helpers/helpers';
import { InfiniteScrollTrigger } from '../InfiniteScrollTrigger';
import { ClientCell, RequestCell, ResponseCell, TimeCell } from './blocks';

import s from './LogTable.module.pcss';
import { ActionsMenu } from '../ActionsMenu';

type Props = {
    logs: LogEntry[];
    hasMore: boolean;
    isLoadingMore: boolean;
    isRequestInFlight: boolean;
    onLoadMore: () => void;
    onRowClick: (entry: LogEntry) => void;
    onBlock: (type: string, domain: string) => void;
    onUnblock: (type: string, domain: string) => void;
    onBlockClient: (type: string, domain: string, client: string) => void;
    onDisallowClient: (ip: string) => void;
    onSearchSelect: (value: string) => void;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
};

export const LogTable = ({
    logs,
    hasMore,
    isLoadingMore,
    isRequestInFlight,
    onLoadMore,
    onRowClick,
    onBlock,
    onUnblock,
    onBlockClient,
    onDisallowClient,
    onSearchSelect,
    filters,
    services,
    whitelistFilters,
}: Props) => {

    const handleSearchSelect = useCallback(
        (value: string) => (event: React.MouseEvent<HTMLButtonElement>) => {
            event.stopPropagation();
            onSearchSelect(value);
        },
        [onSearchSelect],
    );

    const columns: TableColumn<LogEntry>[] = useMemo(
        () => [
            {
                key: 'time',
                header: { text: intl.getMessage('time_table_header') },
                render: (_value: unknown, row: LogEntry) => <TimeCell row={row} />,
                width: 120,
                sortable: false,
            },
            {
                key: 'domain',
                header: { text: intl.getMessage('request_table_header') },
                render: (_value: unknown, row: LogEntry) => <RequestCell row={row} />,
                sortable: false,
            },
            {
                key: 'response',
                header: { text: intl.getMessage('response_table_header') },
                render: (_value: unknown, row: LogEntry) => {
                    return (
                        <ResponseCell
                            row={row}
                            filters={filters}
                            services={services}
                            whitelistFilters={whitelistFilters}
                        />
                    );
                },
                sortable: false,
            },
            {
                key: 'client',
                header: { text: intl.getMessage('client_table_header') },
                render: (_value: unknown, row: LogEntry) => (
                    <ClientCell onSearchSelect={handleSearchSelect} row={row} />
                ),
                sortable: false,
            },
            {
                key: 'actions',
                header: { text: intl.getMessage('actions_table_header') },
                render: (_value: unknown, row: LogEntry) => (
                    <div className={s.actionsCell} onClick={(e) => e.stopPropagation()}>
                        <ActionsMenu
                            domain={row.domain}
                            client={row.client}
                            onBlock={onBlock}
                            onUnblock={onUnblock}
                            onBlockClient={onBlockClient}
                            onDisallowClient={() => onDisallowClient(row.client)}
                            isBlocked={isBlockedReason(row.reason)}
                        />
                    </div>
                ),
                width: 80,
                sortable: false,
            },
        ],
        [filters, handleSearchSelect, onBlock, onUnblock, onBlockClient, onDisallowClient, services, whitelistFilters],
    );

    return (
        <div className={s.tableContainer}>
            <Table
                data={logs}
                columns={columns}
                emptyTable={(
                    <div className={s.emptyTableWrapper}>
                        <Icon icon="not_found_search" className={s.emptyTableIcon} />
                        <div className={cn(s.emptyTableTitle, theme.text.t3)}>
                            {intl.getMessage('not_enough_data')}
                        </div>
                    </div>
                )}
                pagination={false}
                sortable={false}
                className={s.table}
                onRowClick={onRowClick}
                tableRowClassName={s.tableRow}
            />

            {logs.length > 0 && (
                <InfiniteScrollTrigger
                    hasMore={hasMore}
                    loading={isLoadingMore}
                    disabled={isRequestInFlight}
                    onLoadMore={onLoadMore}
                    className={s.loadingRow}
                />
            )}
        </div>
    );
};
