import { createMemo, onMount } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import type { TableColumn } from 'panel/common/ui/Table';
import { statsState } from 'panel/stores/stats';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import type { IOption } from 'panel/lib/helpers/utils';
import { StatsPage } from '../StatsPage';
import { NameCell } from '../blocks/NameCell';
import { StatMobileCard } from '../blocks/StatMobileCard';
import { useStatsRefresh } from '../hooks/useStatsRefresh';

type UpstreamStat = { name: string; count: number };

export const UpstreamAvgTimePage = () => {
    const refreshStats = useStatsRefresh();

    onMount(refreshStats);

    const rows = createMemo<UpstreamStat[]>(() =>
        [...statsState.topUpstreamsAvgTime].toSorted((a, b) => a.name.localeCompare(b.name)),
    );

    const columns = (): TableColumn<UpstreamStat>[] => [
        {
            key: 'upstream',
            header: { text: intl.getMessage('upstream') },
            accessor: 'name',
            sortable: true,
            render: (_v, row) => <NameCell name={row.name} />,
        },
        {
            key: 'time',
            header: { text: intl.getMessage('avg_response_time') },
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

    const mobileSortOptions = (): IOption<string>[] => [
        { value: 'upstream:asc', label: intl.getMessage('sort_upstream_asc') },
        { value: 'upstream:desc', label: intl.getMessage('sort_upstream_desc') },
        { value: 'time:desc', label: intl.getMessage('sort_response_time_desc') },
        { value: 'time:asc', label: intl.getMessage('sort_response_time_asc') },
    ];

    return (
        <StatsPage<UpstreamStat>
            title={intl.getMessage('average_upstream_response_time')}
            rows={rows()}
            columns={columns()}
            getRowId={(row) => row.name}
            defaultSort={{ key: 'time', direction: 'desc' }}
            loading={statsState.processingStats}
            emptyText={intl.getMessage('nothing_found')}
            onRefresh={refreshStats}
            searchTextForRow={(row) => row.name}
            pageSizeKey={LOCAL_STORAGE_KEYS.UPSTREAM_AVG_TIME_PAGE_SIZE}
            sortStorageKey={LOCAL_STORAGE_KEYS.UPSTREAM_AVG_TIME_SORT}
            mobileSortOptions={mobileSortOptions()}
            renderMobileCard={(row) => (
                <StatMobileCard
                    title={row.name}
                    items={[
                        {
                            label: intl.getMessage('avg_response_time'),
                            value: (
                                <span>
                                    {(row.count ?? 0).toFixed(0)}{' '}
                                    {intl.getMessage('milliseconds_abbreviation')}
                                </span>
                            ),
                        },
                    ]}
                />
            )}
        />
    );
};
