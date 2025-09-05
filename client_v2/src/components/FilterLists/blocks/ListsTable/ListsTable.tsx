import React, { useMemo } from 'react';

import { formatDetailedDateTime, Filter } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { MODAL_TYPE } from 'panel/helpers/constants';
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

type ModalPayload = {
    type: string;
    url?: string;
};

type Props = {
    tableId: TableIdsType;
    filters: Filter[];
    processingConfigFilter: boolean;
    toggleFilterList: (url: string, data: FilterToggleData) => void;
    addFilterList: () => void;
    editFilterList: (payload?: ModalPayload) => void;
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
                header: intl.getMessage('enabled_table_header'),
                accessor: 'enabled',
                sortable: false,
                fitContent: true,
                render: (value: boolean, row: Filter) => {
                    const { name, url, enabled } = row;
                    const id = `filter_${url}`;

                    return (
                        <Switch
                            id={id}
                            checked={enabled}
                            onChange={() => toggleFilterList(url, { name, url, enabled: !enabled })}
                            disabled={processingConfigFilter}
                        />
                    );
                },
            },
            {
                key: 'name',
                header: intl.getMessage('name_label'),
                accessor: 'name',
                sortable: true,
                render: (value: string, row: Filter) => (
                    <div>
                        <span title={value}>{value}</span>
                        {(row as any).checksum && (
                            <div title={(row as any).checksum}>
                                {intl.getMessage('checksum_table_header')}: {(row as any).checksum}
                            </div>
                        )}
                    </div>
                ),
            },
            {
                key: 'url',
                header: intl.getMessage('url_label'),
                accessor: 'url',
                sortable: true,
                render: (value: string) => (
                    <div>
                        <span title={value}>{value}</span>
                    </div>
                ),
            },
            {
                key: 'rulesCount',
                header: intl.getMessage('rules_label'),
                accessor: 'rulesCount',
                sortable: true,
                render: (value: number) => <div>{value?.toLocaleString() || 0}</div>,
            },
            {
                key: 'lastUpdated',
                header: intl.getMessage('last_updated_label'),
                accessor: 'lastUpdated',
                sortable: true,
                render: (value: string) => {
                    const result = formatDetailedDateTime(value);
                    return typeof result === 'string' ? <span>{result}</span> : result;
                },
            },
            {
                key: 'actions',
                header: intl.getMessage('actions_label'),
                accessor: 'url',
                sortable: false,
                render: (value: string, row: Filter) => (
                    <div>
                        <button
                            type="button"
                            onClick={() => editFilterList({ type: MODAL_TYPE.EDIT_FILTERS, url: value })}
                            disabled={processingConfigFilter}
                            className={theme.table.action}
                        >
                            <Icon icon="edit" color="gray" />
                        </button>

                        <button
                            type="button"
                            onClick={() => deleteFilterList(value, row.name)}
                            disabled={processingConfigFilter}
                            className={theme.table.action}
                        >
                            <Icon icon="delete" color="red" />
                        </button>
                    </div>
                ),
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
