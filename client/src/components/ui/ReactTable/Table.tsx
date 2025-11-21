import React, { useMemo } from 'react';
import {
    useReactTable,
    getCoreRowModel,
    getSortedRowModel,
    getPaginationRowModel,
    flexRender,
    SortingState,
    PaginationState,
    ColumnDef,
    Row,
} from '@tanstack/react-table';

export interface TableProps<TData> {
    data: TData[];
    columns: ColumnDef<TData, any>[];
    defaultSorted?: SortingState;
    showPagination?: boolean;
    defaultPageSize?: number;
    pageSize?: number;
    onPageSizeChange?: (size: number) => void;
    minRows?: number;
    noDataText?: string;
    loading?: boolean;
    loadingText?: string;
    className?: string;
    getTrProps?: (state: any, rowInfo?: Row<TData>) => React.HTMLAttributes<HTMLTableRowElement>;
    // Pagination text props
    ofText?: string;
    previousText?: string;
    nextText?: string;
    pageText?: string;
    rowsText?: string;
    showPageSizeOptions?: boolean;
}

function Table<TData>({
    data,
    columns,
    defaultSorted = [],
    showPagination = true,
    defaultPageSize = 10,
    pageSize: controlledPageSize,
    onPageSizeChange,
    minRows = 0,
    noDataText = 'No data available',
    loading = false,
    loadingText = 'Loading...',
    className = '',
    getTrProps,
    ofText = '/',
    previousText = 'Previous',
    nextText = 'Next',
    pageText = 'Page',
    rowsText = 'rows',
    showPageSizeOptions = true,
}: TableProps<TData>) {
    const [sorting, setSorting] = React.useState<SortingState>(defaultSorted);
    const [pagination, setPagination] = React.useState<PaginationState>({
        pageIndex: 0,
        pageSize: controlledPageSize || defaultPageSize,
    });

    // Update page size when controlled prop changes
    React.useEffect(() => {
        if (controlledPageSize !== undefined && controlledPageSize !== pagination.pageSize) {
            setPagination((prev) => ({ ...prev, pageSize: controlledPageSize }));
        }
    }, [controlledPageSize]);

    const table = useReactTable({
        data,
        columns,
        state: {
            sorting,
            pagination,
        },
        onSortingChange: setSorting,
        onPaginationChange: (updater) => {
            setPagination((old) => {
                const newState = typeof updater === 'function' ? updater(old) : updater;
                if (onPageSizeChange && newState.pageSize !== old.pageSize) {
                    onPageSizeChange(newState.pageSize);
                }
                return newState;
            });
        },
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        getPaginationRowModel: showPagination ? getPaginationRowModel() : undefined,
        manualPagination: false,
    });

    const {rows} = table.getRowModel();
    const pageCount = table.getPageCount();
    const canPreviousPage = table.getCanPreviousPage();
    const canNextPage = table.getCanNextPage();
    const { pageIndex, pageSize } = table.getState().pagination;

    // Calculate rows to display with minRows
    const rowsToDisplay = useMemo(() => {
        if (rows.length >= minRows) {
            return rows;
        }
        // Create empty rows to fill minRows
        return rows;
    }, [rows, minRows]);

    // Calculate empty rows count
    const emptyRowsCount = Math.max(0, minRows - rows.length);

    return (
        <div className="ReactTable">
            <div className={`rt-table ${className}`}>
                <div className="rt-thead -header">
                    {table.getHeaderGroups().map((headerGroup) => (
                        <div key={headerGroup.id} className="rt-tr" role="row">
                            {headerGroup.headers.map((header) => {
                                const canSort = header.column.getCanSort();
                                const isSorted = header.column.getIsSorted();
                                return (
                                    <div
                                        key={header.id}
                                        className={`rt-th rt-resizable-header ${
                                            isSorted ? `-sort-${  isSorted}` : ''
                                        } ${canSort ? '-cursor-pointer' : ''}`}
                                        onClick={canSort ? header.column.getToggleSortingHandler() : undefined}
                                        style={{
                                            flex: header.column.columnDef.size
                                                ? `${header.column.columnDef.size} 0 auto`
                                                : '1 0 auto',
                                            minWidth: (header.column.columnDef as any).minWidth || undefined,
                                            maxWidth: (header.column.columnDef as any).maxWidth || undefined,
                                            width: (header.column.columnDef as any).width || undefined,
                                        }}
                                        role="columnheader">
                                        <div className="rt-resizable-header-content">
                                            {flexRender(header.column.columnDef.header, header.getContext())}
                                            {canSort && (
                                                <span className="rt-sort-indicator">
                                                    {isSorted === 'asc' && ' 🔼'}
                                                    {isSorted === 'desc' && ' 🔽'}
                                                    {!isSorted && ' ↕'}
                                                </span>
                                            )}
                                        </div>
                                    </div>
                                );
                            })}
                        </div>
                    ))}
                </div>
                <div className="rt-tbody">
                    {loading && (
                        <div className="rt-tr-group">
                            <div className="rt-tr -padRow">
                                <div className="-loading">
                                    <div className="-loading-inner">{loadingText}</div>
                                </div>
                            </div>
                        </div>
                    )}
                    {!loading && rowsToDisplay.length === 0 && (
                        <div className="rt-tr-group">
                            <div className="rt-tr -padRow">
                                <div className="rt-noData">{noDataText}</div>
                            </div>
                        </div>
                    )}
                    {!loading && rowsToDisplay.length > 0 && (
                        <>
                            {rowsToDisplay.map((row) => {
                                const trProps = getTrProps ? getTrProps({}, row) : {};
                                return (
                                    <div
                                        key={row.id}
                                        className={`rt-tr-group ${trProps.className || ''}`}
                                        role="rowgroup">
                                        <div className="rt-tr -odd" role="row" {...trProps}>
                                            {row.getVisibleCells().map((cell) => (
                                                <div
                                                    key={cell.id}
                                                    className={`rt-td ${
                                                        (cell.column.columnDef as any).className || ''
                                                    }`}
                                                    style={{
                                                        flex: cell.column.columnDef.size
                                                            ? `${cell.column.columnDef.size} 0 auto`
                                                            : '1 0 auto',
                                                        minWidth: (cell.column.columnDef as any).minWidth || undefined,
                                                        maxWidth: (cell.column.columnDef as any).maxWidth || undefined,
                                                        width: (cell.column.columnDef as any).width || undefined,
                                                    }}
                                                    role="gridcell">
                                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                );
                            })}
                            {/* Render empty rows if needed */}
                            {emptyRowsCount > 0 &&
                                Array.from({ length: emptyRowsCount }).map((_, index) => (
                                    <div key={`empty-${index}`} className="rt-tr-group" role="rowgroup">
                                        <div className="rt-tr -padRow -even" role="row">
                                            {columns.map((_, colIndex) => (
                                                <div key={colIndex} className="rt-td" role="gridcell">
                                                    &nbsp;
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                ))}
                        </>
                    )}
                </div>
            </div>
            {showPagination && data.length > 0 && (
                <div className="rt-pagination -pagination">
                    <div className="-previous">
                        <button
                            type="button"
                            className="-btn"
                            onClick={() => table.previousPage()}
                            disabled={!canPreviousPage}>
                            {previousText}
                        </button>
                    </div>
                    <div className="-center">
                        <span className="-pageInfo">
                            {pageText}{' '}
                            <div className="-pageJump">
                                <input
                                    type="number"
                                    value={pageIndex + 1}
                                    onChange={(e) => {
                                        const page = e.target.value ? Number(e.target.value) - 1 : 0;
                                        table.setPageIndex(page);
                                    }}
                                    onBlur={(e) => {
                                        const page = e.target.value ? Number(e.target.value) - 1 : 0;
                                        const safePageIndex = Math.min(Math.max(0, page), pageCount - 1);
                                        if (page !== safePageIndex) {
                                            table.setPageIndex(safePageIndex);
                                        }
                                    }}
                                    style={{ width: '70px' }}
                                />
                            </div>{' '}
                            {ofText} <span className="-totalPages">{pageCount || 1}</span>
                        </span>
                        {showPageSizeOptions && (
                            <span className="select-wrap -pageSizeOptions">
                                <select
                                    value={pageSize}
                                    onChange={(e) => {
                                        table.setPageSize(Number(e.target.value));
                                    }}>
                                    {[5, 10, 20, 25, 50, 100].map((size) => (
                                        <option key={size} value={size}>
                                            {size} {rowsText}
                                        </option>
                                    ))}
                                </select>
                            </span>
                        )}
                    </div>
                    <div className="-next">
                        <button type="button" className="-btn" onClick={() => table.nextPage()} disabled={!canNextPage}>
                            {nextText}
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}

export default Table;
