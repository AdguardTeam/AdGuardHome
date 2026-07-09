import { createSignal, createMemo } from 'solid-js';

import type { Filter } from 'panel/helpers/helpers';
import { formatShortDateTime } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Switch } from 'panel/common/controls/Switch';
import { Icon } from 'panel/common/ui/Icon';
import { SortSelect } from 'panel/common/ui/SortSelect';
import theme from 'panel/lib/theme';

import cn from 'clsx';
import s from './ListsTable.module.pcss';

export const TABLE_IDS = {
    ALLOWLISTS_TABLE: 'allowlists_table',
    BLOCKLISTS_TABLE: 'blocklists_table',
    DNSREWRITES_TABLE: 'dnsrewrites_table',
} as const;

type TableIdsType = (typeof TABLE_IDS)[keyof typeof TABLE_IDS];

type FilterToggleData = {
    name: string;
    url: string;
    enabled: boolean;
};

type Props = {
    tableId: TableIdsType;
    filters: Filter[];
    processingConfigFilter: boolean;
    toggleFilterList: (url: string, data: FilterToggleData) => void;
    addFilterList: () => void;
    editFilterList: (url: string, name: string, enabled: boolean) => void;
    deleteFilterList: (url: string, name: string) => void;
};

export const ListsTable = (props: Props) => {
    const [sortDirection, setSortDirection] = createSignal<'asc' | 'desc'>('asc');

    const pageSize = createMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE) || undefined,
    );

    const sortedFilters = createMemo(() => {
        const items = [...(props.filters || [])];

        const direction = sortDirection();
        items.sort((a, b) => {
            const aName = (a.name || '').toLowerCase();
            const bName = (b.name || '').toLowerCase();

            if (aName < bName) {
                return direction === 'asc' ? -1 : 1;
            }

            if (aName > bName) {
                return direction === 'asc' ? 1 : -1;
            }

            return 0;
        });

        return items;
    });

    const handleToggle = (url: string, name: string, enabled: boolean) => {
        props.toggleFilterList(url, { name, url, enabled: !enabled });
    };

    const handleEdit = (url: string, name: string, enabled: boolean) => {
        props.editFilterList(url, name, enabled);
    };

    const handleDelete = (url: string, name: string) => {
        props.deleteFilterList(url, name);
    };

    const columns = createMemo<TableColumn<Filter>[]>(() => [
        {
            key: 'enabled',
            header: {
                text: '',
                className: s.headerCell,
            },
            accessor: 'enabled',
            sortable: false,
            width: 64,
            className: s.cellNameToggleOuter,
            render: (value: boolean, row: Filter) => {
                const { name, url, enabled } = row;
                const id = `filter_${url}`;

                return (
                    <div class={theme.table.cell}>
                        <span class={s.cellNameLabel}>{name}</span>

                        <div class={s.cellValueToggle}>
                            <Switch
                                id={id}
                                checked={enabled}
                                onChange={() => handleToggle(url, name, enabled)}
                                disabled={props.processingConfigFilter}
                            />
                        </div>
                    </div>
                );
            },
        },
        {
            key: 'name',
            header: {
                text: intl.getMessage('name_label'),
                className: s.headerCell,
            },
            accessor: 'name',
            sortable: true,
            className: s.nameDesktopOnly,
            render: (value: string) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>{intl.getMessage('name_label')}</span>

                    <div class={theme.table.cellValueText}>
                        <span class={theme.common.textOverflow}>{value}</span>
                    </div>
                </div>
            ),
        },
        {
            key: 'url',
            header: {
                text: intl.getMessage('url_label'),
                className: s.headerCell,
            },
            accessor: 'url',
            sortable: true,
            render: (value: string) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>{intl.getMessage('url_label')}</span>

                    <div class={theme.table.cellValueText}>
                        <a
                            href={value}
                            class={cn(theme.link.link, theme.common.textOverflow)}
                            target="_blank"
                            rel="noopener noreferrer nofollow"
                        >
                            {value}
                        </a>

                        <button
                            type="button"
                            class={s.copyButton}
                            onClick={() => navigator.clipboard.writeText(value)}
                            aria-label={intl.getMessage('copy')}
                        >
                            <Icon icon="copy" color="green" />
                        </button>
                    </div>
                </div>
            ),
        },
        {
            key: 'rulesCount',
            header: {
                text: intl.getMessage('rules_label'),
                className: s.headerCell,
            },
            accessor: 'rulesCount',
            sortable: true,
            render: (value: number) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>{intl.getMessage('rules_label')}</span>

                    <div class={theme.table.cellValueText}>
                        <span>{value?.toLocaleString() || 0}</span>
                    </div>
                </div>
            ),
        },
        {
            key: 'lastUpdated',
            header: {
                text: intl.getMessage('last_updated_label'),
                className: s.headerCell,
            },
            accessor: 'lastUpdated',
            sortable: true,
            render: (value: string) => {
                const result = formatShortDateTime(value);

                return (
                    <div class={theme.table.cell}>
                        <span class={theme.table.cellLabel}>
                            {intl.getMessage('last_updated_label')}
                        </span>

                        <div class={theme.table.cellValueText}>
                            <span>{result}</span>
                        </div>
                    </div>
                );
            },
        },
        {
            key: 'actions',
            header: {
                text: '',
                className: s.headerCell,
            },
            accessor: 'url',
            sortable: false,
            width: 80,
            render: (value: string, row: Filter) => {
                const { name, url, enabled } = row;

                return (
                    <div class={theme.table.cell}>
                        <div class={theme.table.cellValue}>
                            <div class={theme.table.cellActions}>
                                <button
                                    type="button"
                                    onClick={() => handleEdit(url, name, enabled)}
                                    disabled={props.processingConfigFilter}
                                    class={theme.table.action}
                                    title={intl.getMessage('edit_table_action')}
                                    aria-label={intl.getMessage('edit_table_action')}
                                    data-table-action
                                >
                                    <Icon icon="edit" color="gray" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('edit_table_action')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => handleDelete(url, name)}
                                    disabled={props.processingConfigFilter}
                                    class={cn(theme.table.action, theme.table.action_danger)}
                                    title={intl.getMessage('delete_table_action')}
                                    aria-label={intl.getMessage('delete_table_action')}
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
                );
            },
        },
    ]);

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE, newSize);
    };

    const emptyTableContent = (id: TableIdsType) => {
        const renderButton = (text: string) => {
            return (
                <button
                    class={cn(theme.text.t3, theme.link.link)}
                    type="button"
                    onClick={() => props.addFilterList()}
                >
                    {text}
                </button>
            );
        };

        const DESC_TEXT: Record<string, string> = {
            [TABLE_IDS.ALLOWLISTS_TABLE]: intl.getMessage('allowlist_empty', {
                button: (text: string) => renderButton(text),
            }),
            [TABLE_IDS.BLOCKLISTS_TABLE]: intl.getMessage('blocklists_empty', {
                button: (text: string) => renderButton(text),
            }),
        };

        const emptyText = DESC_TEXT[id];

        return (
            <div class={s.emptyTableContent}>
                <Icon icon="not_found_search" color="gray" class={s.emptyTableIcon} />

                <div class={cn(theme.text.t3, s.emptyTableDesc)}>{emptyText}</div>
            </div>
        );
    };

    return (
        <>
            <div class={cn(theme.pagination.wrapper, s.sortDropdownMobile)}>
                <SortSelect value={sortDirection()} onChange={setSortDirection} />
            </div>

            <Table<Filter>
                data={sortedFilters()}
                class={s.table}
                columns={columns()}
                emptyTable={emptyTableContent(props.tableId)}
                pageSize={pageSize()}
                onPageSizeChange={handlePageSizeChange}
            />
        </>
    );
};
