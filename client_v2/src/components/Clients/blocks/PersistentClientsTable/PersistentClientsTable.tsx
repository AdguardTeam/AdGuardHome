import { createMemo, Show, For } from 'solid-js';
import cn from 'clsx';
import copy from 'copy-to-clipboard';

import intl from 'panel/common/intl';
import type { Client, NormalizedTopClients } from 'panel/initialState';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Icon } from 'panel/common/ui/Icon';
import { Tooltip } from 'panel/common/ui/Tooltip';
import { addSuccessToast } from 'panel/stores/toasts';
import theme from 'panel/lib/theme';

import { ServiceCell } from './ServiceCell';
import { TagCell } from './TagCell';
import type { WebService } from './ServiceIcons';

import { TableEmptyState } from '../TableEmptyState/TableEmptyState';
import s from './PersistentClientsTable.module.pcss';

type Props = {
    clients: Client[];
    normalizedTopClients?: NormalizedTopClients;
    loading?: boolean;
    onEdit: (client: Client) => void;
    onDelete: (name: string) => void;
    editDisabled?: boolean;
    deleteDisabled?: boolean;
    serviceMap: Map<string, WebService>;
};

export const PersistentClientsTable = (props: Props) => {
    const pageSize = createMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE) || undefined,
    );

    const handleCopy = (text: string) => {
        copy(text);
        addSuccessToast(intl.getMessage('copied'));
    };

    const columns = createMemo<TableColumn<Client>[]>(() => {
        // Access serviceMap here so the memo re-runs when it changes.
        // It's used inside render closures below, which SolidJS doesn't track.
        const svcMap = props.serviceMap;

        return [
            {
                key: 'ids',
                header: {
                    text: intl.getMessage('client_identifier'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => row.ids.filter((id) => id.trim() !== '').join(','),
                sortable: true,
                render: (_value: string, row: Client) => {
                    const { ids } = row;
                    // Filter out empty strings — the backend may return trailing empty
                    // entries that would inflate hiddenCount and cause a spurious comma.
                    const nonEmpty = ids.filter((id) => id.trim() !== '');
                    const firstId = nonEmpty[0] || '';
                    const hiddenCount = nonEmpty.length - 1;

                    return (
                        <div class={theme.table.cell}>
                            <span class={theme.table.cellLabel}>
                                {intl.getMessage('client_identifier')}
                            </span>

                            <div class={theme.table.cellValueText}>
                                <div class={s.idsRow}>
                                    <span class={cn(theme.common.textOverflow, s.idsText)}>
                                        {firstId}
                                    </span>
                                    <Show when={hiddenCount > 0}>
                                        <Tooltip
                                            overlayClass={s.idsTooltipOverlay}
                                            content={
                                                <div class={s.idsTooltip}>
                                                    <For each={nonEmpty}>
                                                        {(id) => (
                                                            <span class={s.idsTooltipItem}>
                                                                {id}
                                                            </span>
                                                        )}
                                                    </For>
                                                    <button
                                                        type="button"
                                                        class={cn(
                                                            s.copyBtn,
                                                            s.copyBtnGreen,
                                                            s.copyBtnTopRight,
                                                        )}
                                                        onClick={() =>
                                                            handleCopy(nonEmpty.join(', '))
                                                        }
                                                        title={intl.getMessage('copy')}
                                                    >
                                                        <Icon icon="copy" color="green" />
                                                    </button>
                                                </div>
                                            }
                                        >
                                            <span class={s.countLabel}>{hiddenCount}</span>
                                        </Tooltip>
                                    </Show>
                                </div>
                            </div>
                        </div>
                    );
                },
            },
            {
                key: 'name',
                header: {
                    text: intl.getMessage('name'),
                    className: s.headerCell,
                },
                accessor: 'name',
                sortable: true,
                render: (value: string) => (
                    <div class={theme.table.cell}>
                        <span class={theme.table.cellLabel}>{intl.getMessage('name')}</span>

                        <div class={theme.table.cellValueText}>
                            <Tooltip
                                overlayClass={s.nameTooltipOverlay}
                                content={
                                    <div class={s.nameTooltip}>
                                        <span class={s.nameTooltipText}>{value}</span>
                                        <button
                                            type="button"
                                            class={cn(s.copyBtn, s.copyBtnGreen)}
                                            onClick={() => handleCopy(value)}
                                            title={intl.getMessage('copy')}
                                        >
                                            <Icon icon="copy" color="green" />
                                        </button>
                                    </div>
                                }
                                class={s.nameDropdownInner}
                            >
                                <span class={cn(theme.common.textOverflow, s.nameTrigger)}>
                                    {value}
                                </span>
                            </Tooltip>
                        </div>
                    </div>
                ),
            },
            {
                key: 'settings',
                header: {
                    text: intl.getMessage('settings'),
                    className: s.headerCell,
                },
                accessor: 'use_global_settings',
                sortable: true,
                render: (value: boolean) => (
                    <div class={theme.table.cell}>
                        <span class={theme.table.cellLabel}>{intl.getMessage('settings')}</span>

                        <div class={theme.table.cellValueText}>
                            <span class={theme.common.textOverflow}>
                                {value
                                    ? intl.getMessage('settings_global')
                                    : intl.getMessage('settings_custom')}
                            </span>
                            <Show when={!value}>
                                <Icon icon="user" color="gray" class={s.userIconRight} />
                            </Show>
                        </div>
                    </div>
                ),
            },
            {
                key: 'blocked_services',
                header: {
                    text: intl.getMessage('blocked_services'),
                    className: s.headerCell,
                },
                accessor: 'use_global_blocked_services',
                sortable: true,
                minWidth: 120,
                render: (_value: boolean, row: Client) => (
                    <ServiceCell
                        serviceIds={row.blocked_services || []}
                        useGlobal={row.use_global_blocked_services}
                        serviceMap={svcMap}
                    />
                ),
            },
            {
                key: 'upstreams',
                header: {
                    text: intl.getMessage('upstreams'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => row.upstreams.length > 0,
                sortable: true,
                render: (_value: boolean, row: Client) => (
                    <div class={theme.table.cell}>
                        <span class={theme.table.cellLabel}>{intl.getMessage('upstreams')}</span>

                        <div class={theme.table.cellValueText}>
                            <span class={theme.common.textOverflow}>
                                {row.upstreams.length > 0
                                    ? intl.getMessage('settings_custom')
                                    : intl.getMessage('settings_global')}
                            </span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'tags',
                header: {
                    text: intl.getMessage('tags_title'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => row.tags.join(','),
                sortable: true,
                render: (_value: string, row: Client) => (
                    <TagCell tags={row.tags} onCopy={handleCopy} />
                ),
            },
            {
                key: 'requests',
                header: {
                    text: intl.getMessage('requests_table_header'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => props.normalizedTopClients?.configured[row.name] || 0,
                sortable: true,
                render: (_value: unknown, row: Client) => (
                    <div class={theme.table.cell}>
                        <span class={theme.table.cellLabel}>
                            {intl.getMessage('requests_table_header')}
                        </span>

                        <div class={theme.table.cellValueText}>
                            <span class={theme.common.textOverflow}>
                                {(
                                    props.normalizedTopClients?.configured[row.name] || 0
                                ).toLocaleString()}
                            </span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'actions',
                header: {
                    text: '',
                    className: s.headerCell,
                },
                sortable: false,
                width: 80,
                render: (_value: unknown, row: Client) => (
                    <div class={theme.table.cell}>
                        <div class={theme.table.cellValue}>
                            <div class={theme.table.cellActions}>
                                <button
                                    type="button"
                                    onClick={() => props.onEdit(row)}
                                    disabled={props.editDisabled}
                                    class={theme.table.action}
                                    title={intl.getMessage('edit_table_action')}
                                    aria-label={intl.getMessage('edit_table_action')}
                                    data-testid="clients-edit-button"
                                    data-table-action
                                >
                                    <Icon icon="edit" color="gray" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('edit_table_action')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => props.onDelete(row.name)}
                                    disabled={props.deleteDisabled}
                                    class={cn(theme.table.action, theme.table.action_danger)}
                                    title={intl.getMessage('delete_table_action')}
                                    aria-label={intl.getMessage('delete_table_action')}
                                    data-testid="clients-delete-button"
                                    data-table-action
                                >
                                    <Icon icon="delete" color="red" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('delete_table_action')}
                                    </span>
                                </button>
                            </div>
                        </div>
                    </div>
                ),
            },
        ];
    });

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE, newSize);
    };

    return (
        <Table<Client>
            data={props.clients}
            class={s.table}
            columns={columns()}
            emptyTable={<TableEmptyState />}
            loading={props.loading}
            pageSize={pageSize()}
            onPageSizeChange={handlePageSizeChange}
            getRowId={(row) => row.name}
        />
    );
};
