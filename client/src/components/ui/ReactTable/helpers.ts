import React from 'react';
import { ColumnDef } from '@tanstack/react-table';

/**
 * Helper to convert v6-style column definitions to v8 format
 * This makes migration easier by allowing similar syntax
 */
export interface V6ColumnDef<TData = any> {
    Header?: React.ReactNode | ((props: any) => React.ReactNode);
    accessor?: string | ((row: TData) => any);
    Cell?: React.ComponentType<any> | ((props: any) => React.ReactNode);
    id?: string;
    width?: number;
    minWidth?: number;
    maxWidth?: number;
    className?: string;
    sortable?: boolean;
    resizable?: boolean;
    sortMethod?: (a: any, b: any, desc?: boolean) => number;
}

/**
 * Creates a TanStack Table v8 column definition from v6-style props
 */
export function createColumnHelper<TData = any>(v6Column: V6ColumnDef<TData>): ColumnDef<TData, any> {
    const {
        Header,
        accessor,
        Cell,
        id,
        width,
        minWidth,
        maxWidth,
        className,
        sortable = true,
        resizable = true,
        sortMethod,
    } = v6Column;

    const column: any = {
        header: Header as any,
        enableSorting: sortable,
        enableResizing: resizable,
        meta: {
            className,
        },
    };

    // Set id
    if (id) {
        column.id = id;
    } else if (typeof accessor === 'string') {
        column.id = accessor;
    }

    // Handle accessor
    if (typeof accessor === 'string') {
        column.accessorKey = accessor;
    } else if (typeof accessor === 'function') {
        column.accessorFn = accessor;
        if (!column.id) {
            column.id = `accessor_fn_${  Math.random().toString(36).substring(7)}`;
        }
    }

    // Handle Cell
    if (Cell) {
        column.cell = (info: any) => {
            // Create v6-compatible cell props
            const v6Props = {
                value: info.getValue(),
                row: info.row,
                original: info.row.original,
                index: info.row.index,
                column: info.column,
            };
            
            // Use createElement which handles both functional and class components
            return React.createElement(Cell as any, v6Props);
        };
    }

    // Handle custom sort method
    if (sortMethod) {
        column.sortingFn = (rowA: any, rowB: any, columnId: any) => {
            const a = rowA.getValue(columnId);
            const b = rowB.getValue(columnId);
            return sortMethod(a, b);
        };
    }

    // Size hints (v8 uses size instead of width)
    if (width) {
        column.size = width;
    }
    if (minWidth) {
        column.minWidth = minWidth;
    }
    if (maxWidth) {
        column.maxWidth = maxWidth;
    }

    return column as ColumnDef<TData, any>;
}

/**
 * Converts an array of v6 column definitions to v8 format
 */
export function convertColumns<TData = any>(v6Columns: V6ColumnDef<TData>[]): ColumnDef<TData, any>[] {
    return v6Columns.map((col) => createColumnHelper(col));
}
