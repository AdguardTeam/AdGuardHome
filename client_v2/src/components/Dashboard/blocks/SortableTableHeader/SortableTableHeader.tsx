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
        <button
            type="button"
            className={s.sortableHeader}
            onClick={() => onSort('name')}
        >
            {nameLabel}

            <Icon icon="arrows_sort" className={cn(s.sortIcon, sortField === 'name' && sortDirection === 'asc' && s.sortIconAsc)} />
        </button>
        <button
            type="button"
            className={s.sortableHeader}
            onClick={() => onSort('count')}
        >
            {countLabel}

            <Icon icon="arrows_sort" className={cn(s.sortIcon, sortField === 'count' && sortDirection === 'asc' && s.sortIconAsc)} />
        </button>
    </div>
);
