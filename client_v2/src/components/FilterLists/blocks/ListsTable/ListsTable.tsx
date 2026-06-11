import React, { useMemo, useState } from 'react';

import { Filter, formatShortDateTime } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table as ReactTable, TableColumn } from 'panel/common/ui/Table';
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

export const ListsTable = ({
    tableId,
    filters,
    processingConfigFilter,
    addFilterList,
    editFilterList,
    deleteFilterList,
    toggleFilterList,
}: Props) => {
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

    const pageSize = useMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE) || undefined,
        [],
    );

    const sortedFilters = useMemo(() => {
        const items = [...(filters || [])];

        items.sort((a, b) => {
            const aName = (a.name || '').toLowerCase();
            const bName = (b.name || '').toLowerCase();

            if (aName < bName) {
                return sortDirection === 'asc' ? -1 : 1;
            }

            if (aName > bName) {
                return sortDirection === 'asc' ? 1 : -1;
            }

            return 0;
        });

        return items;
    }, [filters, sortDirection]);

    const columns: TableColumn<Filter>[] = useMemo(
        () => [
            {
                key: 'enabled',
                header: {
                    text: '',
                    className: s.headerCell,
                },
                accessor: 'enabled',
                sortable: false,
                fitContent: true,
                className: s.cellNameToggleOuter,
                render: (value: boolean, row: Filter) => {
                    const { name, url, enabled } = row;
                    const id = `filter_${url}`;

                    return (
                        <div className={s.cell}>
                            <span className={s.cellNameLabel}>{name}</span>

                            <div className={s.cellValueToggle}>
                                <Switch
                                    id={id}
                                    checked={enabled}
                                    onChange={() =>
                                        toggleFilterList(url, { name, url, enabled: !enabled })
                                    }
                                    disabled={processingConfigFilter}
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
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('name_label')}</span>

                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow}>{value}</span>
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
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('url_label')}</span>

                        <div className={s.cellValue}>
                            <a
                                href={value}
                                className={cn(theme.link.link, theme.common.textOverflow)}
                                target="_blank"
                                rel="noopener noreferrer nofollow"
                            >
                                {value}
                            </a>

                            <button
                                type="button"
                                className={s.copyButton}
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
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('rules_label')}</span>

                        <div className={s.cellValue}>
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
                        <div className={s.cell}>
                            <span className={s.cellLabel}>
                                {intl.getMessage('last_updated_label')}
                            </span>

                            <div className={s.cellValue}>
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
                        <div className={s.cell}>
                            <div className={s.cellActions}>
                                <button
                                    type="button"
                                    onClick={() => editFilterList(url, name, enabled)}
                                    disabled={processingConfigFilter}
                                    className={s.editAction}
                                >
                                    <span className={cn(s.editActionLabel, theme.text.t2)}>
                                        {intl.getMessage('edit_table_action')}
                                    </span>
                                    <span className={s.editActionIcon}>
                                        <Icon icon="edit" color="gray" />
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => deleteFilterList(url, name)}
                                    disabled={processingConfigFilter}
                                    className={s.action}
                                >
                                    <Icon icon="delete" color="red" />
                                </button>
                            </div>
                        </div>
                    );
                },
            },
        ],
        [processingConfigFilter, toggleFilterList, editFilterList, deleteFilterList],
    );

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE, newSize);
    };

    const emptyTableContent = (id: TableIdsType) => {
        const renderButton = (text: string) => {
            return (
                <button
                    className={cn(theme.text.t3, theme.link.link)}
                    type="button"
                    onClick={() => addFilterList()}
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
            <div className={s.emptyTableContent}>
                <Icon icon="not_found_search" color="gray" className={s.emptyTableIcon} />

                <div className={cn(theme.text.t3, s.emptyTableDesc)}>{emptyText}</div>
            </div>
        );
    };

    return (
        <>
            <div className={cn(theme.pagination.wrapper, s.sortDropdownMobile)}>
                <SortSelect value={sortDirection} onChange={setSortDirection} />
            </div>

            <ReactTable<Filter>
                data={sortedFilters}
                className={s.table}
                columns={columns}
                emptyTable={emptyTableContent(tableId)}
                pageSize={pageSize}
                onPageSizeChange={handlePageSizeChange}
            />
        </>
    );
};
