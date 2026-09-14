import { createMemo, onMount } from 'solid-js';

import intl from 'panel/common/intl';
import type { TableColumn } from 'panel/common/ui/Table';
import { RoutePath } from 'panel/components/Routes/Paths';
import { statsState } from 'panel/stores/stats';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { computePercent } from 'panel/helpers/statistics';
import type { IOption } from 'panel/lib/helpers/utils';
import { StatsPage } from '../StatsPage';
import { CountWithPercent } from '../blocks/CountWithPercent';
import { NameCell } from '../blocks/NameCell';
import { StatMobileCard } from '../blocks/StatMobileCard';
import { useStatsRefresh } from '../hooks/useStatsRefresh';

type DomainStat = { name: string; count: number };

export const TopQueriedDomainsPage = () => {
    const refreshStats = useStatsRefresh();

    onMount(refreshStats);

    const rows = createMemo<DomainStat[]>(() =>
        [...statsState.topQueriedDomains].toSorted((a, b) => a.name.localeCompare(b.name)),
    );

    const columns = (): TableColumn<DomainStat>[] => [
        {
            key: 'domain',
            header: { text: intl.getMessage('domain') },
            accessor: 'name',
            sortable: true,
            render: (_v, row) => <NameCell name={row.name} queryLogSearch={row.name} />,
        },
        {
            key: 'queries',
            header: { text: intl.getMessage('queries') },
            accessor: 'count',
            sortable: true,
            sortFn: (a: number, b: number) => a - b,
            render: (_v, row) => (
                <CountWithPercent
                    count={row.count}
                    total={statsState.numDnsQueries}
                    queryLogSearch={row.name}
                    progress={computePercent(row.count, statsState.numDnsQueries)}
                />
            ),
        },
    ];

    const mobileSortOptions = (): IOption<string>[] => [
        { value: 'domain:asc', label: intl.getMessage('sort_domain_asc') },
        { value: 'domain:desc', label: intl.getMessage('sort_domain_desc') },
        { value: 'queries:desc', label: intl.getMessage('sort_queries_desc') },
        { value: 'queries:asc', label: intl.getMessage('sort_queries_asc') },
    ];

    return (
        <StatsPage<DomainStat>
            title={intl.getMessage('stats_query_domain')}
            rows={rows()}
            columns={columns()}
            getRowId={(row) => row.name}
            defaultSort={{ key: 'queries', direction: 'desc' }}
            loading={statsState.processingStats}
            emptyText={intl.getMessage('nothing_found')}
            onRefresh={refreshStats}
            searchTextForRow={(row) => row.name}
            pageSizeKey={LOCAL_STORAGE_KEYS.TOP_QUERIED_DOMAINS_PAGE_SIZE}
            sortStorageKey={LOCAL_STORAGE_KEYS.TOP_QUERIED_DOMAINS_SORT}
            mobileSortOptions={mobileSortOptions()}
            renderMobileCard={(row) => (
                <StatMobileCard
                    title={row.name}
                    titleLink={{
                        to: RoutePath.QueryLog,
                        query: { search: `"${row.name}"` },
                        title: row.name,
                    }}
                    items={[
                        {
                            label: intl.getMessage('queries'),
                            value: (
                                <CountWithPercent
                                    count={row.count}
                                    total={statsState.numDnsQueries}
                                    queryLogSearch={row.name}
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
