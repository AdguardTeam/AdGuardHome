import { createSignal, createMemo, untrack } from 'solid-js';

export type SortField = 'name' | 'count';
export type SortDirection = 'asc' | 'desc';

type SortableItem = {
    name: string;
    count: number;
};

type UseSortedDataResult<T extends SortableItem> = {
    sortedData: () => T[];
    sortField: () => SortField;
    sortDirection: () => SortDirection;
    handleSort: (field: SortField) => void;
};

export const DEFAULT_VISIBLE_ITEMS = 10;

export const useSortedData = <T extends SortableItem>(
    data: () => T[],
    defaultSortField: SortField = 'count',
    defaultSortDirection: SortDirection = 'desc',
    limit: number = DEFAULT_VISIBLE_ITEMS,
): UseSortedDataResult<T> => {
    const [sortField, setSortField] = createSignal<SortField>(defaultSortField);
    const [sortDirection, setSortDirection] = createSignal<SortDirection>(defaultSortDirection);

    const sortedData = createMemo(() => {
        return (
            [...data()]
                // eslint-disable-next-line solid/reactivity -- sort callback inside createMemo; reads already tracked
                .sort((a, b) => {
                    const modifier = sortDirection() === 'asc' ? 1 : -1;
                    if (sortField() === 'name') {
                        return a.name.localeCompare(b.name) * modifier;
                    }
                    return (a.count - b.count) * modifier;
                })
                .slice(0, limit)
        );
    });

    // eslint-disable-next-line solid/reactivity -- returned to consumers, called from JSX (tracked)
    const handleSort = (field: SortField) => {
        untrack(() => {
            if (sortField() === field) {
                setSortDirection(sortDirection() === 'asc' ? 'desc' : 'asc');
            } else {
                setSortField(field);
                setSortDirection(field === 'name' ? 'asc' : 'desc');
            }
        });
    };

    return {
        sortedData,
        sortField,
        sortDirection,
        handleSort,
    };
};
