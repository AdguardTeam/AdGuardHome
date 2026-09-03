import { createMemo } from 'solid-js';

type SortableItem = {
    count: number;
};

export const DEFAULT_VISIBLE_ITEMS = 5;

export const TOP_CLIENTS_VISIBLE_ITEMS = 4;

export const useSortedData = <T extends SortableItem>(
    data: () => T[],
    limit: number = DEFAULT_VISIBLE_ITEMS,
): { sortedData: () => T[] } => {
    const sortedData = createMemo(() =>
        data()
            .toSorted((a, b) => b.count - a.count)
            .slice(0, limit),
    );

    return { sortedData };
};
