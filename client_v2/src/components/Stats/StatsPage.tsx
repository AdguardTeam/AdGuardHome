import { For, Show, createMemo, createSignal, type JSX } from 'solid-js';
import cn from 'clsx';
import { useSearchParams } from '@solidjs/router';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { RoutePath } from 'panel/components/Routes/Paths';
import { useIsMobile } from 'panel/hooks/useIsMobile';
import { LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { isQueryMatch } from 'panel/helpers/statistics';

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
    renderMobileCard: (row: T) => JSX.Element;
    children?: JSX.Element;
};

export function StatsPage<T>(props: StatsPageProps<T>) {
    const [searchQuery, setSearchQuery] = createSignal('');
    const isMobile = useIsMobile();
    const [searchParams, setSearchParams] = useSearchParams<{ sort?: string; dir?: string }>();

    const filteredRows = createMemo(() => {
        const query = searchQuery();
        if (!query.trim()) {
            return props.rows;
        }
        return props.rows.filter((row) => isQueryMatch(props.searchTextForRow(row), query));
    });

    const pageSize = createMemo(
        () => LocalStorageHelper.getItem<number>(props.pageSizeKey) || undefined,
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

    return (
        <div class={cn(theme.layout.container, s.containerOverride)}>
            <div class={cn(theme.layout.containerIn, s.page)}>
                <div class={s.header}>
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
                        <h1 class={cn(theme.title.h3_tablet, s.pageTitle)} data-testid="stats-page-title">
                            {props.title}
                        </h1>
                    </div>

                    <div class={s.headerControls}>
                        <Input
                            data-testid="stats-search-input"
                            class={s.searchField}
                            value={searchQuery()}
                            onInput={(e: Event) =>
                                setSearchQuery((e.target as HTMLInputElement).value)
                            }
                            placeholder={intl.getMessage('search_placeholder')}
                            size="small"
                            prefixIcon={<Icon icon="search" class={s.searchIcon} />}
                            suffixIcon={
                                <FaqTooltip text={intl.getMessage('query_log_strict_search')} />
                            }
                        />

                        <Button
                            class={s.refreshButton}
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
                    </div>
                </div>

                {props.children}

                <Show
                    when={!isMobile()}
                    fallback={
                        <div class={s.mobileCards} data-testid="stats-mobile-list">
                            <For each={filteredRows()}>{(row) => props.renderMobileCard(row)}</For>
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
                        onPageSizeChange={(size: number) =>
                            LocalStorageHelper.setItem(props.pageSizeKey, size)
                        }
                        emptyTable={
                            <div
                                class={cn(theme.text.t2, s.emptyText)}
                                data-testid="stats-table-empty"
                            >
                                {props.emptyText}
                            </div>
                        }
                    />
                </Show>
            </div>
        </div>
    );
}
