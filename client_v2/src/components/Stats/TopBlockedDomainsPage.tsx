import { createMemo, onMount } from 'solid-js';
import cn from 'clsx';
import { useSearchParams } from '@solidjs/router';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import type { TableColumn } from 'panel/common/ui/Table';
import { statsState, getStats } from 'panel/stores/stats';
import { formatCompactNumber } from 'panel/helpers/helpers';
import { computePercent, resolveStatsPeriod } from 'panel/helpers/statistics';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { StatsPage } from './StatsPage';

import s from './TopBlockedDomainsPage.module.pcss';

type DomainStat = { name: string; count: number };

export const TopBlockedDomainsPage = () => {
    const [searchParams] = useSearchParams<{ period?: string }>();

    onMount(() => getStats(resolveStatsPeriod(searchParams, statsState.interval)));

    // Pre-sorted by name ascending: stable tie-break for equal counts.
    const rows = createMemo<DomainStat[]>(() =>
        [...statsState.topBlockedDomains].toSorted((a, b) => a.name.localeCompare(b.name)),
    );

    const columns = (): TableColumn<DomainStat>[] => [
        {
            key: 'domain',
            header: { text: intl.getMessage('domain') },
            accessor: 'name',
            sortable: true,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, theme.text.condenced, s.nameCell)} title={row.name}>
                    {row.name}
                </span>
            ),
        },
        {
            key: 'queries',
            header: { text: intl.getMessage('blocked_queries') },
            accessor: 'count',
            sortable: true,
            sortFn: (a: number, b: number) => a - b,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, theme.text.condenced, s.countCell, s.blockedCount)}>
                    {formatCompactNumber(row.count)}
                    <span class={s.percent} data-testid="stats-percent">
                        ({computePercent(row.count, statsState.numBlockedFiltering).toFixed(1)}%)
                    </span>
                </span>
            ),
        },
    ];

    return (
        <StatsPage<DomainStat>
            title={intl.getMessage('top_blocked_domains')}
            rows={rows()}
            columns={columns()}
            getRowId={(row) => row.name}
            defaultSort={{ key: 'queries', direction: 'desc' }}
            loading={statsState.processingStats}
            emptyText={intl.getMessage('stats_table_empty')}
            onRefresh={() => getStats(resolveStatsPeriod(searchParams, statsState.interval))}
            searchTextForRow={(row) => row.name}
            pageSizeKey={LOCAL_STORAGE_KEYS.TOP_BLOCKED_DOMAINS_PAGE_SIZE}
            sortStorageKey={LOCAL_STORAGE_KEYS.TOP_BLOCKED_DOMAINS_SORT}
            renderMobileCard={(row) => (
                <div class={s.mobileCard} data-testid="stats-mobile-card">
                    <span class={cn(theme.text.t3, theme.text.condenced)}>{row.name}</span>
                    <span class={cn(theme.text.t3, theme.text.condenced, s.countCell, s.blockedCount)}>
                        {formatCompactNumber(row.count)}
                        <span class={s.percent} data-testid="stats-percent">
                            ({computePercent(row.count, statsState.numBlockedFiltering).toFixed(1)}
                            %)
                        </span>
                    </span>
                </div>
            )}
        />
    );
};
