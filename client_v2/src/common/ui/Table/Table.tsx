import React, { useState, useMemo, useCallback, ReactNode } from 'react';
import cn from 'clsx';

import { Loader } from 'panel/common/ui/Loader';
import theme from 'panel/lib/theme';
import { Pagination } from './blocks/Pagination';

import s from './Table.module.pcss';

import { Icon } from '../Icon';

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 40, 50];

export interface TableColumn<T = any> {
    key: string;
    header: {
        text: string;
        className?: string;
    };
    accessor?: keyof T | ((row: T) => any);
    render?: (value: any, row: T, index: number) => ReactNode;
    sortable?: boolean;
    fitContent?: boolean;
    width?: number | string;
    minWidth?: number;
    maxWidth?: number;
    className?: string;
}

export interface TableProps<T = any> {
    data: T[];
    columns: TableColumn<T>[];
    emptyTable: ReactNode;
    loading?: boolean;
    className?: string;
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
}

export interface TableState {
    currentPage: number;
    pageSize: number;
    sortKey: string | null;
    sortDirection: 'asc' | 'desc';
}

export const Table = <T extends Record<string, any>>({
    data,
    columns,
    loading,
    emptyTable,
    pagination = true,
    pageSize: initialPageSize = DEFAULT_PAGE_SIZE_OPTIONS[0],
    onPageSizeChange,
    pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
    sortable = true,
    defaultSort,
    onSortChange,
    getRowId = (row: T, index: number) => index,
}: TableProps<T>) => {
    const [state, setState] = useState<TableState>({
        currentPage: 0,
        pageSize: initialPageSize,
        sortKey: defaultSort?.key || null,
        sortDirection: defaultSort?.direction || 'asc',
    });

    const sortedData = useMemo(() => {
        if (!state.sortKey || !sortable) {
            return data;
        }

        const column = columns.find((col) => col.key === state.sortKey);
        if (!column || column.sortable === false) {
            return data;
        }

        const sortedData = [...data].sort((a, b) => {
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

            // Handle null/undefined values
            if (aValue == null && bValue == null) {
                return 0;
            }
            if (aValue == null) {
                return 1;
            }
            if (bValue == null) {
                return -1;
            }

            // Convert to comparable values
            if (typeof aValue === 'string' && typeof bValue === 'string') {
                aValue = aValue.toLowerCase();
                bValue = bValue.toLowerCase();
            }

            let comparison = 0;
            if (aValue < bValue) {
                comparison = -1;
            } else if (aValue > bValue) {
                comparison = 1;
            }

            return state.sortDirection === 'desc' ? -comparison : comparison;
        });
        return sortedData;
    }, [data, columns, state.sortKey, state.sortDirection, sortable]);

    const paginatedData = useMemo(() => {
        if (!pagination) {
            return sortedData;
        }

        const startIndex = state.currentPage * state.pageSize;
        return sortedData.slice(startIndex, startIndex + state.pageSize);
    }, [sortedData, pagination, state.currentPage, state.pageSize]);

    const totalPages = Math.ceil(sortedData.length / state.pageSize);

    const handleSort = useCallback(
        (columnKey: string) => {
            const column = columns.find((col) => col.key === columnKey);
            if (!column || column.sortable === false) {
                return;
            }

            let newDirection: 'asc' | 'desc' = 'asc';
            if (state.sortKey === columnKey && state.sortDirection === 'asc') {
                newDirection = 'desc';
            }

            setState((prev) => ({
                ...prev,
                sortKey: columnKey,
                sortDirection: newDirection,
                currentPage: 0,
            }));

            onSortChange?.(columnKey, newDirection);
        },
        [columns, state.sortKey, state.sortDirection, onSortChange],
    );

    const handlePageChange = useCallback((page: number) => {
        setState((prev) => ({ ...prev, currentPage: page }));
    }, []);

    const handlePageSizeChange = useCallback(
        (newSize: number) => {
            setState((prev) => ({
                ...prev,
                pageSize: newSize,
                currentPage: 0,
            }));
            onPageSizeChange?.(newSize);
        },
        [onPageSizeChange],
    );

    const renderCell = useCallback((column: TableColumn<T>, row: T, rowIndex: number) => {
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
            return column.accessor(row);
        }

        if (column.accessor) {
            const value = row[column.accessor];
            return value != null ? String(value) : '';
        }

        return '';
    }, []);

    if (loading) {
        return (
            <div className={s.loading}>
                <Loader />
            </div>
        );
    }

    const hasData = paginatedData.length > 0;

    return (
        <div className={s.tableContainer}>
            <div className={s.tableMain}>
                <div className={s.table}>
                    <div className={s.tableHeader}>
                        {columns.map((column) => (
                            <div
                                key={column.key}
                                className={cn(s.tableCell, column.header.className, {
                                    [s.sortable]: column.sortable,
                                    [s.fitContent]: column.fitContent,
                                })}
                                onClick={() => sortable && column.sortable && handleSort(column.key)}
                            >
                                <span className={cn(theme.text.t3, theme.text.condenced, theme.text.semibold)}>
                                    {column.header.text}
                                </span>

                                {sortable && column.sortable && (
                                    <span>
                                        {state.sortKey === column.key && state.sortDirection === 'asc' && (
                                            <Icon icon="arrow" color="gray" className={s.sortAsc} />
                                        )}
                                        {state.sortKey === column.key && state.sortDirection === 'desc' && (
                                            <Icon icon="arrow" color="gray" className={s.sortDesc} />
                                        )}
                                    </span>
                                )}
                            </div>
                        ))}
                    </div>

                    {hasData && (
                        <>
                            {paginatedData.map((row, index) => {
                                const rowId = getRowId(row, index);

                                return (
                                    <div key={rowId} className={s.tableRow}>
                                        {columns.map((column) => (
                                            <div key={column.key} className={s.tableCell}>
                                                {renderCell(column, row, index)}
                                            </div>
                                        ))}
                                    </div>
                                );
                            })}
                        </>
                    )}
                </div>

                {!hasData && <div className={s.emptyTableWrapper}>{emptyTable}</div>}
            </div>

            {pagination && sortedData.length > 0 && (
                <div className={s.tablePagination}>
                    <Pagination
                        currentPage={state.currentPage}
                        totalPages={totalPages}
                        pageSize={state.pageSize}
                        totalItems={sortedData.length}
                        pageSizeOptions={pageSizeOptions}
                        onPageChange={handlePageChange}
                        onPageSizeChange={handlePageSizeChange}
                    />
                </div>
            )}
        </div>
    );
};
