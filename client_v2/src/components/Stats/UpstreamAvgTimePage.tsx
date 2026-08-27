import { createMemo, onMount } from 'solid-js';
import cn from 'clsx';
import { useSearchParams } from '@solidjs/router';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import type { TableColumn } from 'panel/common/ui/Table';
import { statsState, getStats } from 'panel/stores/stats';
import { resolveStatsPeriod } from 'panel/helpers/statistics';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { StatsPage } from './StatsPage';

import s from './UpstreamAvgTimePage.module.pcss';

type UpstreamStat = { name: string; count: number }; // count is already in ms

export const UpstreamAvgTimePage = () => {
    const [searchParams] = useSearchParams<{ period?: string }>();

    onMount(() => getStats(resolveStatsPeriod(searchParams, statsState.interval)));

    // Pre-sorted by name ascending: stable tie-break for equal times.
    const rows = createMemo<UpstreamStat[]>(() =>
        [...statsState.topUpstreamsAvgTime].toSorted((a, b) => a.name.localeCompare(b.name)),
    );

    const columns = (): TableColumn<UpstreamStat>[] => [
        {
            key: 'upstream',
            header: { text: intl.getMessage('upstream') },
            accessor: 'name',
            sortable: true,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, theme.text.condenced, s.nameCell)} title={row.name}>
                    {row.name}
                </span>
            ),
        },
        {
            key: 'time',
            header: { text: intl.getMessage('response_time') },
            accessor: 'count',
            sortable: true,
            sortFn: (a: number, b: number) => a - b,
            render: (_v, row) => (
                <span
                    class={cn(theme.text.t3, theme.text.condenced)}
                    data-testid="stats-avg-time-value"
                >
                    {(row.count ?? 0).toFixed(0)} {intl.getMessage('milliseconds_abbreviation')}
                </span>
            ),
        },
    ];

    return (
        <StatsPage<UpstreamStat>
            title={intl.getMessage('average_upstream_response_time')}
            rows={rows()}
            columns={columns()}
            getRowId={(row) => row.name}
            defaultSort={{ key: 'time', direction: 'desc' }}
            loading={statsState.processingStats}
            emptyText={intl.getMessage('stats_table_empty')}
            onRefresh={() => getStats(resolveStatsPeriod(searchParams, statsState.interval))}
            searchTextForRow={(row) => row.name}
            pageSizeKey={LOCAL_STORAGE_KEYS.UPSTREAM_AVG_TIME_PAGE_SIZE}
            sortStorageKey={LOCAL_STORAGE_KEYS.UPSTREAM_AVG_TIME_SORT}
            renderMobileCard={(row) => (
                <div class={s.mobileCard} data-testid="stats-mobile-card">
                    <span class={cn(theme.text.t3, theme.text.condenced)}>{row.name}</span>
                    <span class={cn(theme.text.t3, theme.text.condenced)}>
                        {(row.count ?? 0).toFixed(0)} {intl.getMessage('milliseconds_abbreviation')}
                    </span>
                </div>
            )}
        />
    );
};
