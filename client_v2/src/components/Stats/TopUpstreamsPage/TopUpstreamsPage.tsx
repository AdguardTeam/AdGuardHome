import { createMemo, onMount } from 'solid-js';

import intl from 'panel/common/intl';
import type { TableColumn } from 'panel/common/ui/Table';
import { statsState } from 'panel/stores/stats';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { computePercent } from 'panel/helpers/statistics';
import type { IOption } from 'panel/lib/helpers/utils';
import { StatsPage } from '../StatsPage';
import { CountWithPercent } from '../blocks/CountWithPercent';
import { NameCell } from '../blocks/NameCell';
import { StatMobileCard } from '../blocks/StatMobileCard';
import { useStatsRefresh } from '../hooks/useStatsRefresh';

type UpstreamStat = { name: string; count: number };

export const TopUpstreamsPage = () => {
    const refreshStats = useStatsRefresh();

    onMount(refreshStats);

    const rows = createMemo<UpstreamStat[]>(() =>
        [...statsState.topUpstreamsResponses].toSorted((a, b) => a.name.localeCompare(b.name)),
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
            key: 'queries',
            header: { text: intl.getMessage('queries') },
            accessor: 'count',
            sortable: true,
            sortFn: (a: number, b: number) => a - b,
            render: (_v, row) => (
                <CountWithPercent count={row.count} total={statsState.numDnsQueries} />
            ),
        },
    ];

    const mobileSortOptions = (): IOption<string>[] => [
        { value: 'upstream:asc', label: intl.getMessage('sort_upstream_asc') },
        { value: 'upstream:desc', label: intl.getMessage('sort_upstream_desc') },
        { value: 'queries:desc', label: intl.getMessage('sort_queries_desc') },
        { value: 'queries:asc', label: intl.getMessage('sort_queries_asc') },
    ];

    return (
        <StatsPage<UpstreamStat>
            title={intl.getMessage('top_upstreams')}
            rows={rows()}
            columns={columns()}
            getRowId={(row) => row.name}
            defaultSort={{ key: 'queries', direction: 'desc' }}
            loading={statsState.processingStats}
            emptyText={intl.getMessage('nothing_found')}
            onRefresh={refreshStats}
            searchTextForRow={(row) => row.name}
            pageSizeKey={LOCAL_STORAGE_KEYS.TOP_UPSTREAMS_PAGE_SIZE}
            sortStorageKey={LOCAL_STORAGE_KEYS.TOP_UPSTREAMS_SORT}
            mobileSortOptions={mobileSortOptions()}
            renderMobileCard={(row) => (
                <StatMobileCard
                    title={row.name}
                    items={[
                        {
                            label: intl.getMessage('queries'),
                            value: (
                                <CountWithPercent
                                    count={row.count}
                                    total={statsState.numDnsQueries}
                                />
                            ),
                            progress: computePercent(row.count, statsState.numDnsQueries),
                        },
                    ]}
                />
            )}
        />
    );
};
