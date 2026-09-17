import { For, Show, createEffect, createMemo, createSignal, untrack, type JSX } from 'solid-js';
import cn from 'clsx';
import { useSearchParams } from '@solidjs/router';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { Table, Pagination, type TableColumn } from 'panel/common/ui/Table';
import { DEFAULT_PAGE_SIZE, DEFAULT_PAGE_SIZE_OPTIONS } from 'panel/common/ui/Table/Table';
import { Loader } from 'panel/common/ui/Loader';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { RoutePath } from 'panel/components/Routes/Paths';
import { useIsMobile } from 'panel/hooks/useIsMobile';
import { LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { isQueryMatch } from 'panel/helpers/statistics';
import type { IOption } from 'panel/lib/helpers/utils';
import { EmptyState } from '../blocks/EmptyState';

import s from './StatsPage.module.pcss';

type SortState = {
    key: string;
    direction: 'asc' | 'desc';
};

type StatsPageProps<T> = {
    title: string;
    rows: T[];
    columns: TableColumn<T>[];
    getRowId: (row: T, index: number) => string | number;
    defaultSort: SortState;
    loading: boolean;
    emptyText: string;
    onRefresh: () => void;
    searchTextForRow: (row: T) => string;
    pageSizeKey: string;
    sortStorageKey: string;
    mobileSortOptions: IOption<string>[];
    renderMobileCard: (row: T) => JSX.Element;
    controlsBefore?: JSX.Element;
    controlsAfter?: JSX.Element;
    controlsUnderTitle?: boolean;
    children?: JSX.Element;
};

export function StatsPage<T>(props: StatsPageProps<T>) {
    const [searchQuery, setSearchQuery] = createSignal('');
    const [currentPage, setCurrentPage] = createSignal(0);
    const isMobile = useIsMobile();
    const [searchParams, setSearchParams] = useSearchParams<{ sort?: string; dir?: string }>();

    const filteredRows = createMemo(() => {
        const query = searchQuery();
        if (!query.trim()) {
            return props.rows;
        }
        return props.rows.filter((row) => isQueryMatch(props.searchTextForRow(row), query));
    });

    const [pageSize, setPageSize] = createSignal(
        untrack(() => LocalStorageHelper.getItem<number>(props.pageSizeKey) ?? DEFAULT_PAGE_SIZE),
    );

    const getStoredSort = (): SortState | null => {
        const stored = LocalStorageHelper.getItem<SortState>(props.sortStorageKey);
        if (!stored?.key || (stored.direction !== 'asc' && stored.direction !== 'desc')) {
            return null;
        }
        return stored;
    };

    const resolvedSort = (): SortState => {
        if (searchParams.sort && (searchParams.dir === 'asc' || searchParams.dir === 'desc')) {
            return { key: searchParams.sort, direction: searchParams.dir };
        }
        return getStoredSort() ?? props.defaultSort;
    };

    const handleSortChange = (key: string, direction: 'asc' | 'desc') => {
        LocalStorageHelper.setItem(props.sortStorageKey, { key, direction });
        setSearchParams({ sort: key, dir: direction }, { replace: true });
    };

    // Mobile sort select shares the desktop sort state (localStorage + URL).
    const currentSortValue = createMemo(() => {
        const sort = resolvedSort();
        return `${sort.key}:${sort.direction}`;
    });

    const handleMobileSortChange = (option: IOption<string>) => {
        const [key, direction] = option.value.split(':') as [string, 'asc' | 'desc'];
        handleSortChange(key, direction);
        setCurrentPage(0);
    };

    const mobileTotalPages = () => Math.max(1, Math.ceil(filteredRows().length / pageSize()));

    // The rows-per-page select is only meaningful once there is at least a full
    // page of rows, so the whole footer is hidden for short lists — the same
    // rule the desktop Table applies.
    const showMobilePagination = () => filteredRows().length >= DEFAULT_PAGE_SIZE;

    // The desktop Table sorts internally; mirror that logic here so the mobile
    // card list is ordered the same way (resolvedSort drives both).
    const sortedMobileRows = createMemo(() => {
        const data = filteredRows();
        const { key, direction } = resolvedSort();
        const column = props.columns.find((col) => col.key === key);
        if (!column || column.sortable === false) {
            return data;
        }

        const accessor =
            typeof column.accessor === 'function'
                ? column.accessor
                : column.accessor
                  ? (row: any) => (row as Record<string, unknown>)[column.accessor as string]
                  : null;
        if (!accessor) {
            return data;
        }

        return [...data].sort((a, b) => {
            let aValue = accessor(a);
            let bValue = accessor(b);

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

            return direction === 'desc' ? -comparison : comparison;
        });
    });

    const mobilePagedRows = createMemo(() => {
        const start = currentPage() * pageSize();
        return sortedMobileRows().slice(start, start + pageSize());
    });

    const handlePageSizeChange = (size: number) => {
        LocalStorageHelper.setItem(props.pageSizeKey, size);
        setPageSize(size);
        setCurrentPage(0);
    };

    createEffect(() => {
        if (currentPage() >= mobileTotalPages()) {
            setCurrentPage(0);
        }
    });

    return (
        <div class={cn(theme.layout.container, s.containerOverride)}>
            <div class={cn(theme.layout.containerIn, s.page)}>
                <div class={cn(s.header, props.controlsUnderTitle && s.headerUnderTitle)}>
                    <div class={s.headerLeft}>
                        <Breadcrumbs
                            parentLinks={[
                                {
                                    path: RoutePath.Dashboard,
                                    title: intl.getMessage('dashboard'),
                                    dataTestid: 'breadcrumbs-dashboard',
                                },
                            ]}
                            currentTitle={props.title}
                        />
                        <div class={s.titleRow}>
                            <h1
                                class={cn(theme.title.h4, theme.title.h3_tablet, s.pageTitle)}
                                data-testid="stats-page-title"
                            >
                                {props.title}
                            </h1>
                            <Show when={isMobile()}>
                                <Button
                                    class={cn(s.refreshButton, s.refreshButtonMobile)}
                                    variant="ghost"
                                    size="small"
                                    compact
                                    aria-label={intl.getMessage('refresh_btn')}
                                    title={intl.getMessage('refresh_btn')}
                                    onClick={() => props.onRefresh()}
                                    disabled={props.loading}
                                >
                                    <Icon icon="refresh" class={s.refreshIcon} />
                                </Button>
                            </Show>
                        </div>
                    </div>

                    <div class={s.headerControls}>
                        <Show when={props.controlsBefore}>
                            <div class={cn(s.controlSlot, s.controlBefore)}>
                                {props.controlsBefore}
                            </div>
                        </Show>
                        <Input
                            data-testid="stats-search-input"
                            class={s.searchField}
                            value={searchQuery()}
                            onInput={(e: Event) => {
                                setSearchQuery((e.target as HTMLInputElement).value);
                                setCurrentPage(0);
                            }}
                            placeholder={intl.getMessage('search_placeholder')}
                            size={isMobile() ? 'large' : 'small'}
                            prefixIcon={<Icon icon="search" class={s.searchIcon} />}
                            suffixIcon={
                                <FaqTooltip text={intl.getMessage('stats_strict_search')} />
                            }
                            isClearable
                            onClear={() => {
                                setSearchQuery('');
                                setCurrentPage(0);
                            }}
                        />
                        <Show when={!isMobile()}>
                            <Button
                                class={cn(s.refreshButton, s.refreshButtonDesktop)}
                                variant="ghost"
                                size="small"
                                compact
                                aria-label={intl.getMessage('refresh_btn')}
                                title={intl.getMessage('refresh_btn')}
                                onClick={() => props.onRefresh()}
                                disabled={props.loading}
                            >
                                <Icon icon="refresh" class={s.refreshIcon} />
                            </Button>
                        </Show>
                        <Show when={props.controlsAfter}>
                            <div class={cn(s.controlSlot, s.controlAfter)}>
                                {props.controlsAfter}
                            </div>
                        </Show>
                    </div>
                </div>

                {props.children}

                <Show
                    when={!isMobile()}
                    fallback={
                        <div class={s.mobileContent} data-testid="stats-mobile-list">
                            <Show
                                when={!props.loading}
                                fallback={
                                    <div class={s.mobileLoader} data-testid="stats-mobile-loader">
                                        <Loader class={s.mobileLoaderIcon} />
                                    </div>
                                }
                            >
                                <Show
                                    when={filteredRows().length > 0}
                                    fallback={
                                        <EmptyState
                                            message={props.emptyText}
                                            class={s.mobileEmptyState}
                                        />
                                    }
                                >
                                    <div class={s.mobileSort}>
                                        <span class={cn(theme.text.t3, s.mobileSortLabel)}>
                                            {intl.getMessage('sort_by')}
                                        </span>
                                        <div data-testid="stats-sort-select">
                                            <Select<string>
                                                options={props.mobileSortOptions}
                                                value={props.mobileSortOptions.find(
                                                    (option) => option.value === currentSortValue(),
                                                )}
                                                onChange={handleMobileSortChange}
                                                height="big"
                                                isSearchable={false}
                                                optionTestIdPrefix="stats-sort-option"
                                            />
                                        </div>
                                    </div>
                                    <div class={s.mobileCards}>
                                        <For each={mobilePagedRows()}>
                                            {(row) => props.renderMobileCard(row)}
                                        </For>
                                    </div>
                                    <Show when={showMobilePagination()}>
                                        <div class={s.mobilePagination}>
                                            <Pagination
                                                currentPage={currentPage()}
                                                totalPages={mobileTotalPages()}
                                                pageSize={pageSize()}
                                                totalItems={filteredRows().length}
                                                pageSizeOptions={DEFAULT_PAGE_SIZE_OPTIONS}
                                                onPageChange={(page: number) => setCurrentPage(page)}
                                                onPageSizeChange={handlePageSizeChange}
                                            />
                                        </div>
                                    </Show>
                                </Show>
                            </Show>
                        </div>
                    }
                >
                    <Table<T>
                        data={filteredRows()}
                        columns={props.columns}
                        getRowId={props.getRowId}
                        defaultSort={resolvedSort()}
                        onSortChange={handleSortChange}
                        loading={props.loading}
                        pageSize={pageSize()}
                        onPageSizeChange={handlePageSizeChange}
                        emptyTable={<EmptyState message={props.emptyText} />}
                    />
                </Show>
            </div>
        </div>
    );
}
