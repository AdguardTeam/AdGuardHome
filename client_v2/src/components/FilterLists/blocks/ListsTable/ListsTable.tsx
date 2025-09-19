import React, { useMemo } from 'react';

import { Filter, formatShortDateTime } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table as ReactTable, TableColumn } from 'panel/common/ui/Table';
import { Switch } from 'panel/common/controls/Switch';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import cn from 'clsx';
import s from './ListsTable.module.pcss';

const DEFAULT_PAGE_SIZE = 10;

export const TABLE_IDS = {
    ALLOWLISTS_TABLE: 'allowlists_table',
    BLOCKLISTS_TABLE: 'blocklists_table',
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
    const pageSize = useMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE) || DEFAULT_PAGE_SIZE,
        [],
    );

    const columns: TableColumn<Filter>[] = useMemo(
        () => [
            {
                key: 'enabled',
                header: {
                    text: intl.getMessage('enabled_table_header'),
                    className: s.headerCell,
                },
                accessor: 'enabled',
                sortable: false,
                fitContent: true,
                render: (value: boolean, row: Filter) => {
                    const { name, url, enabled } = row;
                    const id = `filter_${url}`;

                    return (
                        <div className={s.cell}>
                            <span className={s.cellLabel}>{intl.getMessage('enabled_table_header')}</span>

                            <div className={s.cellValue}>
                                <Switch
                                    id={id}
                                    checked={enabled}
                                    onChange={() => toggleFilterList(url, { name, url, enabled: !enabled })}
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
                render: (value: string, row: Filter) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('name_label')}</span>

                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow}>{value}</span>

                            {(row as any).checksum && (
                                <div title={(row as any).checksum}>
                                    {intl.getMessage('checksum_table_header')}: {(row as any).checksum}
                                </div>
                            )}
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
                            <span className={s.cellLabel}>{intl.getMessage('last_updated_label')}</span>

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
                    text: intl.getMessage('actions_label'),
                    className: s.headerCell,
                },
                accessor: 'url',
                sortable: false,
                render: (value: string, row: Filter) => {
                    const { name, url, enabled } = row;

                    return (
                        <div className={s.cell}>
                            <span className={s.cellLabel}>{intl.getMessage('actions_label')}</span>

                            <div className={s.cellValue}>
                                <div className={s.cellActions}>
                                    <button
                                        type="button"
                                        onClick={() => editFilterList(url, name, enabled)}
                                        disabled={processingConfigFilter}
                                        className={s.action}
                                    >
                                        <Icon icon="edit" color="gray" />
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
                <button className={cn(theme.text.t3, theme.link.link)} type="button" onClick={() => addFilterList()}>
                    {text}
                </button>
            );
        };

        const DESC_TEXT = {
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
        <ReactTable<Filter>
            data={filters}
            className={s.table}
            columns={columns}
            emptyTable={emptyTableContent(tableId)}
            pageSize={pageSize}
            onPageSizeChange={handlePageSizeChange}
        />
    );
};
