import React from 'react';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import s from './SortableTableHeader.module.pcss';

type SortField = 'name' | 'count';
type SortDirection = 'asc' | 'desc';

type Props = {
    nameLabel: string;
    countLabel: string;
    sortField: SortField;
    sortDirection: SortDirection;
    onSort: (field: SortField) => void;
};

export const SortableTableHeader = ({
    nameLabel,
    countLabel,
    sortField,
    sortDirection,
    onSort,
}: Props) => (
    <div className={cn(theme.text.t3, theme.text.semibold, s.tableHeader)}>
        <span
            className={s.sortableHeader}
            onClick={() => onSort('name')}
        >
            {nameLabel}
            {sortField === 'name' ? (
                <Icon icon="arrow_bottom" className={cn(s.sortIcon, sortDirection === 'asc' && s.sortIconAsc)} />
            ) : (
                <span className={s.sortDash}>—</span>
            )}
        </span>
        <span
            className={s.sortableHeader}
            onClick={() => onSort('count')}
        >
            {countLabel}
            {sortField === 'count' ? (
                <Icon icon="arrow_bottom" className={cn(s.sortIcon, sortDirection === 'asc' && s.sortIconAsc)} />
            ) : (
                <span className={s.sortDash}>—</span>
            )}
        </span>
    </div>
);
