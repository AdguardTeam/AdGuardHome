import { createMemo, Show, untrack } from 'solid-js';

import intl from 'panel/common/intl';
import { Loader } from 'panel/common/ui/Loader';
import { Table, TableColumn } from 'panel/common/ui/Table/Table';

import { LogEntry, Service } from 'panel/components/QueryLog/types';
import { hasPersistentClient, isBlockedReason } from 'panel/components/QueryLog/helpers';

import { Filter } from 'panel/helpers/helpers';
import { InfiniteScrollTrigger } from '../InfiniteScrollTrigger';
import { EmptyState, type EmptyStateMode } from '../EmptyState/EmptyState';
import { ClientCell, RequestCell, ReasonCell, StatusCell, TimeCell } from './blocks';

import s from './LogTable.module.pcss';
import { ActionsMenu } from '../ActionsMenu';

type Props = {
    logs: LogEntry[];
    emptyStateMode: EmptyStateMode;
    hasMore: boolean;
    isLoadingMore: boolean;
    isRequestInFlight: boolean;
    isInitialLoading: boolean;
    isFilterReloading: boolean;
    infiniteScrollResetToken: string;
    onLoadMore: () => void;
    onRowClick: (entry: LogEntry) => void;
    onBlock: (domain: string) => void;
    onUnblock: (domain: string) => void;
    onBlockClient: (domain: string, client: string) => void;
    onDisallowClient: (ip: string) => void;
    onAddPersistentClient: (clientId: string) => void;
    onSearchSelect: (value: string) => void;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
    persistentClientIds: string[];
    persistentClientsLoaded: boolean;
};

export const LogTable = (props: Props) => {
    const handleSearchSelect = (value: string) => (event: MouseEvent) => {
        event.stopPropagation();
        untrack(() => props.onSearchSelect(value));
    };

    const columns = createMemo<TableColumn<LogEntry>[]>(() => [
        {
            key: 'time',
            header: { text: intl.getMessage('time_table_header') },
            render: (_value: unknown, row: LogEntry) => <TimeCell row={row} />,
            width: 116,
            sortable: false,
        },
        {
            key: 'domain',
            header: { text: intl.getMessage('request_table_header') },
            render: (_value: unknown, row: LogEntry) => <RequestCell row={row} />,
            sortable: false,
        },
        {
            key: 'status',
            header: { text: intl.getMessage('status_table_header') },
            render: (_value: unknown, row: LogEntry) => <StatusCell row={row} />,
            width: 'minmax(108px, 0.7fr)',
            sortable: false,
        },
        {
            key: 'reason',
            header: { text: intl.getMessage('reason_table_header') },
            render: (_value: unknown, row: LogEntry) => {
                return (
                    <ReasonCell
                        row={row}
                        filters={props.filters}
                        services={props.services}
                        whitelistFilters={props.whitelistFilters}
                    />
                );
            },
            width: 'minmax(136px, 0.9fr)',
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
            header: { text: '', render: () => null },
            render: (_value: unknown, row: LogEntry) => (
                <div
                    class={s.actionsCell}
                    data-testid="query-log-actions-cell"
                    data-client={row.client}
                    onClick={(e) => e.stopPropagation()}
                >
                    <ActionsMenu
                        domain={row.domain}
                        client={row.client}
                        clientId={row.client_id || row.client}
                        onBlock={props.onBlock}
                        onUnblock={props.onUnblock}
                        onBlockClient={props.onBlockClient}
                        onDisallowClient={() => props.onDisallowClient(row.client)}
                        onAddPersistentClient={props.onAddPersistentClient}
                        isBlocked={isBlockedReason(row.reason)}
                        showAddPersistentClient={
                            props.persistentClientsLoaded &&
                            !hasPersistentClient(row, props.persistentClientIds)
                        }
                        testIdPrefix="query-log-row"
                    />
                </div>
            ),
            width: 48,
            sortable: false,
        },
    ]);

    return (
        <div class={s.tableContainer}>
            <Table
                data={props.logs}
                columns={columns()}
                emptyTable={
                    props.isInitialLoading || props.isFilterReloading ? (
                        <div class={s.initialLoader} data-testid="query-log-initial-loader">
                            <Loader color="green" class={s.loader} />
                        </div>
                    ) : (
                        <EmptyState
                            class={s.emptyTableWrapper}
                            mode={props.emptyStateMode}
                            messageClass={s.emptyTableTitle}
                        />
                    )
                }
                pagination={false}
                sortable={false}
                class={s.table}
                onRowClick={props.onRowClick}
                tableRowClass={s.tableRow}
                tableHeaderClass={s.tableHeader}
            />

            <Show when={props.logs.length > 0}>
                <InfiniteScrollTrigger
                    hasMore={props.hasMore}
                    loading={props.isLoadingMore}
                    disabled={props.isRequestInFlight}
                    onLoadMore={props.onLoadMore}
                    resetToken={props.infiniteScrollResetToken}
                    class={s.loadingRow}
                />
            </Show>
        </div>
    );
};
