import { type JSX, createMemo, For, Show, untrack } from 'solid-js';
import { createStore } from 'solid-js/store';
import cn from 'clsx';

import { Loader } from 'panel/common/ui/Loader';
import theme from 'panel/lib/theme';
import { Pagination } from './blocks/Pagination/Pagination';

import s from './Table.module.pcss';

import { Icon } from '../Icon';

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 40, 50];
export const DEFAULT_PAGE_SIZE = DEFAULT_PAGE_SIZE_OPTIONS[0];

export interface TableColumn<T = any> {
    key: string;
    header: {
        text: string;
        class?: string;
        render?: () => JSX.Element;
    };
    accessor?: keyof T | ((row: T) => any);
    render?: (value: any, row: T, index: number) => JSX.Element;
    sortable?: boolean;
    sortFn?: (a: any, b: any) => number;
    fitContent?: boolean;
    width?: number | string;
    minWidth?: number;
    maxWidth?: number;
    class?: string;
}

export interface TableProps<T = any> {
    data: T[];
    columns: TableColumn<T>[];
    emptyTable?: JSX.Element;
    loading?: boolean;
    class?: string;
    pagination?: boolean;
    pageSize?: number;
    onPageSizeChange?: (size: number) => void;
    pageSizeOptions?: number[];
    sortable?: boolean;
    defaultSort?: {
        key: string;
        direction: 'asc' | 'desc';
    };
    onSortChange?: (key: string, direction: 'asc' | 'desc') => void;
    getRowId?: (row: T, index: number) => string | number;
    onRowClick?: (row: T) => void;
    tableHeaderClass?: string;
    tableRowClass?: string;
}

export const Table = <T extends Record<string, any>>(props: TableProps<T>) => {
    const [state, setState] = createStore({
        currentPage: 0,
        pageSize: untrack(() => props.pageSize) ?? DEFAULT_PAGE_SIZE_OPTIONS[0],
        sortKey: props.defaultSort?.key ?? (null as string | null),
        sortDirection: props.defaultSort?.direction ?? ('asc' as 'asc' | 'desc'),
    });

    const sortedData = createMemo(() => {
        const data = props.data;
        const columns = props.columns;
        const sortKey = state.sortKey;
        const sortDirection = state.sortDirection;
        const sortable = props.sortable ?? true;

        if (!sortKey || !sortable) {
            return data;
        }

        const column = columns.find((col) => col.key === sortKey);
        if (!column || column.sortable === false) {
            return data;
        }

        const sorted = [...data].sort((a, b) => {
            let aValue: any;
            let bValue: any;

            if (typeof column.accessor === 'function') {
                aValue = column.accessor(a);
                bValue = column.accessor(b);
            } else if (column.accessor) {
                aValue = a[column.accessor];
                bValue = b[column.accessor];
            } else {
                return 0;
            }

            if (aValue == null && bValue == null) return 0;
            if (aValue == null) return 1;
            if (bValue == null) return -1;

            if (typeof aValue === 'string' && typeof bValue === 'string') {
                aValue = aValue.toLowerCase();
                bValue = bValue.toLowerCase();
            }

            let comparison: number;
            if (column.sortFn) {
                comparison = column.sortFn(aValue, bValue);
            } else {
                comparison = 0;
                if (aValue < bValue) comparison = -1;
                else if (aValue > bValue) comparison = 1;
            }

            return sortDirection === 'desc' ? -comparison : comparison;
        });
        return sorted;
    });

    const paginatedData = createMemo(() => {
        const pagination = props.pagination ?? true;
        if (!pagination) {
            return sortedData();
        }
        const startIndex = state.currentPage * state.pageSize;
        return sortedData().slice(startIndex, startIndex + state.pageSize);
    });

    const totalPages = () => Math.ceil(sortedData().length / state.pageSize);

    const tableGridTemplate = createMemo(() =>
        props.columns
            .map((column) => {
                if (typeof column.width === 'number') return `${column.width}px`;
                if (typeof column.width === 'string') return column.width;
                if (column.fitContent) return 'fit-content(100%)';
                if (column.minWidth && column.maxWidth)
                    return `minmax(${column.minWidth}px, ${column.maxWidth}px)`;
                if (column.minWidth) return `minmax(${column.minWidth}px, 1fr)`;
                return 'minmax(0, 1fr)';
            })
            .join(' '),
    );

    const tableStyle = () =>
        ({
            '--table-columns': tableGridTemplate(),
        }) as Record<string, string>;

    const handleSort = (columnKey: string) => {
        const column = props.columns.find((col) => col.key === columnKey);
        if (!column || column.sortable === false) return;

        let newDirection: 'asc' | 'desc' = 'asc';
        if (state.sortKey === columnKey && state.sortDirection === 'asc') {
            newDirection = 'desc';
        }

        setState('sortKey', columnKey);
        setState('sortDirection', newDirection);
        setState('currentPage', 0);

        props.onSortChange?.(columnKey, newDirection);
    };

    const handlePageChange = (page: number) => {
        setState('currentPage', page);
    };

    const handlePageSizeChange = (newSize: number) => {
        setState('pageSize', newSize);
        setState('currentPage', 0);
        props.onPageSizeChange?.(newSize);
    };

    const renderCell = (column: TableColumn<T>, row: T, rowIndex: number) => {
        if (column.render) {
            let value = null;
            if (typeof column.accessor === 'function') {
                value = column.accessor(row);
            } else if (column.accessor) {
                value = row[column.accessor];
            }
            return column.render(value, row, rowIndex);
        }

        if (typeof column.accessor === 'function') {
            return column.accessor(row) as JSX.Element;
        }

        if (column.accessor) {
            const value = row[column.accessor];
            return value != null ? String(value) : '';
        }

        return '';
    };

    const hasData = () => paginatedData().length > 0;

    return (
        <Show
            when={!props.loading}
            fallback={
                <div class={s.loading}>
                    <Loader class={s.tableLoader} />
                </div>
            }
        >
            <div class={s.tableContainer}>
                <div class={s.tableMain}>
                    <div class={cn(s.table, props.class)}>
                        <div class={cn(s.tableHeader, props.tableHeaderClass)} style={tableStyle()}>
                            <For each={props.columns}>
                                {(column) => (
                                    <div
                                        class={cn(
                                            s.tableCell,
                                            s.tableHeaderCell,
                                            column.header.class,
                                            {
                                                [s.sortable]: column.sortable,
                                                [s.fitContent]: column.fitContent,
                                            },
                                        )}
                                        onClick={() =>
                                            (props.sortable ?? true) &&
                                            column.sortable &&
                                            handleSort(column.key)
                                        }
                                    >
                                        {column.header.render ? (
                                            column.header.render()
                                        ) : (
                                            <span
                                                class={cn(
                                                    theme.text.t3,
                                                    theme.text.condenced,
                                                    theme.text.semibold,
                                                )}
                                            >
                                                {column.header.text}
                                            </span>
                                        )}

                                        {(props.sortable ?? true) && column.sortable && (
                                            <span>
                                                <Show
                                                    when={
                                                        state.sortKey === column.key &&
                                                        state.sortDirection === 'asc'
                                                    }
                                                >
                                                    <Icon
                                                        icon="arrow"
                                                        color="gray"
                                                        class={s.sortAsc}
                                                    />
                                                </Show>
                                                <Show
                                                    when={
                                                        state.sortKey === column.key &&
                                                        state.sortDirection === 'desc'
                                                    }
                                                >
                                                    <Icon
                                                        icon="arrow"
                                                        color="gray"
                                                        class={s.sortDesc}
                                                    />
                                                </Show>
                                            </span>
                                        )}
                                    </div>
                                )}
                            </For>
                        </div>

                        <Show when={hasData()}>
                            <For each={paginatedData()}>
                                {(row, index) => {
                                    return (
                                        <div
                                            class={cn(s.tableRow, props.tableRowClass)}
                                            style={tableStyle()}
                                            onClick={() => props.onRowClick?.(row)}
                                        >
                                            <For each={props.columns}>
                                                {(column) => (
                                                    <div
                                                        class={cn(
                                                            s.tableCell,
                                                            s.tableBodyCell,
                                                            column.class,
                                                            {
                                                                [s.fitContent]: column.fitContent,
                                                                [s.clickableCell]:
                                                                    !!props.onRowClick,
                                                            },
                                                        )}
                                                    >
                                                        {renderCell(column, row, index())}
                                                    </div>
                                                )}
                                            </For>
                                        </div>
                                    );
                                }}
                            </For>
                        </Show>
                    </div>

                    <Show when={!hasData() && props.emptyTable}>
                        <div class={s.emptyTableWrapper}>{props.emptyTable}</div>
                    </Show>
                </div>

                <Show when={(props.pagination ?? true) && sortedData().length >= DEFAULT_PAGE_SIZE}>
                    <div class={s.tablePagination}>
                        <Pagination
                            currentPage={state.currentPage}
                            totalPages={totalPages()}
                            pageSize={state.pageSize}
                            totalItems={sortedData().length}
                            pageSizeOptions={props.pageSizeOptions ?? DEFAULT_PAGE_SIZE_OPTIONS}
                            onPageChange={handlePageChange}
                            onPageSizeChange={handlePageSizeChange}
                        />
                    </div>
                </Show>
            </div>
        </Show>
    );
};
