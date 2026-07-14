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

export const SortableTableHeader = (props: Props) => (
    <div class={cn(theme.text.t3, theme.text.semibold, s.tableHeader)}>
        <button type="button" class={s.sortableHeader} onClick={() => props.onSort('name')}>
            {props.nameLabel}

            <Icon
                icon="arrows_sort"
                class={cn(
                    s.sortIcon,
                    props.sortField === 'name' && props.sortDirection === 'asc' && s.sortIconAsc,
                )}
            />
        </button>
        <button type="button" class={s.sortableHeader} onClick={() => props.onSort('count')}>
            {props.countLabel}

            <Icon
                icon="arrows_sort"
                class={cn(
                    s.sortIcon,
                    props.sortField === 'count' && props.sortDirection === 'asc' && s.sortIconAsc,
                )}
            />
        </button>
    </div>
);
