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
        const direction = sortDirection();
        const field = sortField();
        return data()
            .toSorted((a, b) => {
                const modifier = direction === 'asc' ? 1 : -1;
                if (field === 'name') {
                    return a.name.localeCompare(b.name) * modifier;
                }
                return (a.count - b.count) * modifier;
            })
            .slice(0, limit);
    });

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
