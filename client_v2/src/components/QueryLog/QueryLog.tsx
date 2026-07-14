import { createSignal, createEffect, onMount, Show, For } from 'solid-js';
import cn from 'clsx';
import { useNavigate, useLocation } from '@solidjs/router';

import { Loader } from 'panel/common/ui/Loader';
import theme from 'panel/lib/theme';
import {
    queryLogsState,
    getLogsConfig,
    setLogsFilter,
    setFilteredLogs,
    refreshFilteredLogs,
    getAdditionalLogs,
} from 'panel/stores/queryLogs';
import { accessState, getAccessList, toggleClientBlock } from 'panel/stores/access';
import { dashboardState, getClients } from 'panel/stores/dashboard';
import {
    filteringState,
    getFilteringStatus,
    blockDomain,
    unblockDomain,
    blockDomainForClient,
} from 'panel/stores/filtering';
import { servicesState, getAllBlockedServices, allowBlockedService } from 'panel/stores/services';
import {
    DEFAULT_LOGS_FILTER,
    QUERY_LOG_REASON_FILTER_QUERIES,
    QUERY_LOG_STATUS_FILTER_QUERIES,
} from 'panel/helpers/constants';
import { getLogsUrlParams } from 'panel/helpers/helpers';
import { RoutePath, linkPathBuilder } from 'panel/components/Routes/Paths';

import { filterLogsByStatus } from './helpers';
import { LogEntry } from './types';
import { Header } from './blocks/Header';
import { EmptyState, type EmptyStateMode } from './blocks/EmptyState/EmptyState';
import { LogTable } from './blocks/LogTable';
import { LogCard } from './blocks/LogCard';
import { DetailModal } from './blocks/DetailModal';
import { DisallowDialog } from './blocks/DisallowDialog';
import { InfiniteScrollTrigger } from './blocks/InfiniteScrollTrigger';

import s from './QueryLog.module.pcss';

const getEmptyStateMode = (enabled: boolean, interval?: number): EmptyStateMode => {
    if (!enabled) {
        return 'disabled';
    }
    if (interval === 0) {
        return 'rotation-disabled';
    }
    return 'default';
};

export const QueryLog = () => {
    const navigate = useNavigate();
    const location = useLocation();

    const [selectedEntry, setSelectedEntry] = createSignal<LogEntry | null>(null);
    const [disallowTarget, setDisallowTarget] = createSignal<string | null>(null);
    const [isIncrementalLoad, setIsIncrementalLoad] = createSignal(false);

    onMount(() => {
        getLogsConfig();
        getAccessList();
        getClients();
        getFilteringStatus();
        getAllBlockedServices();
    });

    // Watch location.search for filter changes
    createEffect(() => {
        const searchParams = new URLSearchParams(location.search);
        const search = searchParams.get('search') || DEFAULT_LOGS_FILTER.search;
        const statusParam = searchParams.get('status') || DEFAULT_LOGS_FILTER.status;
        const reasonParam = searchParams.get('reason') || DEFAULT_LOGS_FILTER.reason;
        const status = Object.hasOwn(QUERY_LOG_STATUS_FILTER_QUERIES, statusParam)
            ? statusParam
            : DEFAULT_LOGS_FILTER.status;
        const reason = Object.hasOwn(QUERY_LOG_REASON_FILTER_QUERIES, reasonParam)
            ? reasonParam
            : DEFAULT_LOGS_FILTER.reason;
        const nextFilter = { search, status, reason };
        const nextSearch = getLogsUrlParams(search, status, reason);

        if (location.search !== nextSearch) {
            navigate(nextSearch, { replace: true });
            return;
        }

        setIsIncrementalLoad(false);
        setLogsFilter(nextFilter);
        setFilteredLogs(nextFilter);
    });

    // Reset incremental load when processing finishes
    createEffect(() => {
        if (!queryLogsState.processingGetLogs && !queryLogsState.processingAdditionalLogs) {
            setIsIncrementalLoad(false);
        }
    });

    const currentSearch = () => queryLogsState.filter?.search ?? DEFAULT_LOGS_FILTER.search;
    const currentStatus = () => queryLogsState.filter?.status ?? DEFAULT_LOGS_FILTER.status;
    const currentReason = () => queryLogsState.filter?.reason ?? DEFAULT_LOGS_FILTER.reason;
    const infiniteScrollResetToken = () =>
        `${currentSearch()}:${currentStatus()}:${currentReason()}`;
    const persistentClientIds = () =>
        (dashboardState.clients || []).flatMap(
            (persistentClient: any) => persistentClient.ids ?? [],
        );
    const visibleLogs = () => filterLogsByStatus(queryLogsState.logs || [], currentStatus());
    const emptyStateMode = () => getEmptyStateMode(queryLogsState.enabled, queryLogsState.interval);
    const hasMore = () => !queryLogsState.isEntireLog;
    const logs = () => queryLogsState.logs || [];

    const isRequestInFlight = () =>
        queryLogsState.processingGetLogs || queryLogsState.processingAdditionalLogs;
    const isLoadingMore = () => isIncrementalLoad() && isRequestInFlight();
    const isInitialLoading = () =>
        queryLogsState.processingGetLogs && logs().length === 0 && !isIncrementalLoad();
    const isFilterReloading = () =>
        queryLogsState.processingGetLogs && !isInitialLoading() && !isIncrementalLoad();

    const handleSearch = (search: string) => {
        setIsIncrementalLoad(false);
        navigate(getLogsUrlParams(search.trim(), currentStatus(), currentReason()), {
            replace: true,
        });
    };

    const handleStatusFilterChange = (status: string) => {
        setIsIncrementalLoad(false);
        navigate(getLogsUrlParams(currentSearch(), status, DEFAULT_LOGS_FILTER.reason), {
            replace: true,
        });
    };

    const handleReasonFilterChange = (reason: string) => {
        setIsIncrementalLoad(false);
        navigate(getLogsUrlParams(currentSearch(), currentStatus(), reason), { replace: true });
    };

    const handleRefresh = () => {
        setIsIncrementalLoad(false);
        refreshFilteredLogs();
    };

    const handleBlockDomain = (domain: string) => {
        blockDomain(domain);
    };

    const handleUnblockDomain = (domain: string) => {
        unblockDomain(domain);
    };

    const handleAllowService = (serviceId: string) => {
        allowBlockedService(serviceId);
    };

    const handleBlockClient = (domain: string, client: string) => {
        blockDomainForClient(domain, client);
    };

    const handleDisallowClient = (ip: string) => {
        setDisallowTarget(ip);
    };

    const handleAddPersistentClient = (clientId: string) => {
        navigate(linkPathBuilder(RoutePath.ClientsAdd, undefined, { id: clientId }));
    };

    const handleConfirmDisallow = () => {
        const target = disallowTarget();
        if (target) {
            const disallowedList = accessState.disallowed_clients
                ? accessState.disallowed_clients.split('\n').filter(Boolean)
                : [];
            const isDisallowed = disallowedList.includes(target);
            toggleClientBlock(target, isDisallowed, isDisallowed ? target : '');
            setDisallowTarget(null);
        }
    };

    const handleCloseDisallow = () => {
        setDisallowTarget(null);
    };

    const handleRowClick = (entry: LogEntry) => {
        setSelectedEntry(entry);
    };

    const handleCloseDetail = () => {
        setSelectedEntry(null);
    };

    const handleLoadMore = () => {
        if (isRequestInFlight() || queryLogsState.isEntireLog) {
            return;
        }
        setIsIncrementalLoad(true);
        getAdditionalLogs();
    };

    const allowedClients = (): string[] => {
        const raw = accessState?.allowed_clients;
        if (!raw) {
            return [];
        }
        if (typeof raw === 'string') {
            return raw.split('\n').filter(Boolean);
        }
        if (Array.isArray(raw)) {
            return raw as string[];
        }
        return [];
    };

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, s.page)}>
                <Header
                    onSearch={handleSearch}
                    onRefresh={handleRefresh}
                    onStatusFilterChange={handleStatusFilterChange}
                    onReasonFilterChange={handleReasonFilterChange}
                    currentSearch={currentSearch()}
                    currentStatus={currentStatus()}
                    currentReason={currentReason()}
                    isLoading={!!isRequestInFlight()}
                />

                <div class={s.desktopView}>
                    <LogTable
                        logs={visibleLogs()}
                        emptyStateMode={emptyStateMode()}
                        hasMore={hasMore()}
                        isLoadingMore={isLoadingMore()}
                        isRequestInFlight={isRequestInFlight()}
                        isInitialLoading={isInitialLoading()}
                        isFilterReloading={isFilterReloading()}
                        infiniteScrollResetToken={infiniteScrollResetToken()}
                        onLoadMore={handleLoadMore}
                        onRowClick={handleRowClick}
                        onBlock={handleBlockDomain}
                        onUnblock={handleUnblockDomain}
                        onBlockClient={handleBlockClient}
                        onDisallowClient={handleDisallowClient}
                        onAddPersistentClient={handleAddPersistentClient}
                        onSearchSelect={handleSearch}
                        filters={filteringState.filters || []}
                        services={servicesState.allServices || []}
                        whitelistFilters={filteringState.whitelistFilters || []}
                        persistentClientIds={persistentClientIds()}
                        persistentClientsLoaded={!dashboardState.processingClients}
                    />
                </div>

                <div class={s.mobileView}>
                    <Show
                        when={
                            isInitialLoading() ||
                            (isFilterReloading() && visibleLogs().length === 0)
                        }
                        fallback={
                            <Show
                                when={visibleLogs().length === 0}
                                fallback={
                                    <>
                                        <div class={s.mobileList}>
                                            <For each={visibleLogs()}>
                                                {(entry) => (
                                                    <LogCard
                                                        entry={entry}
                                                        onRowClick={handleRowClick}
                                                        onBlock={handleBlockDomain}
                                                        onUnblock={handleUnblockDomain}
                                                        onBlockClient={handleBlockClient}
                                                        onDisallowClient={handleDisallowClient}
                                                        onAddPersistentClient={
                                                            handleAddPersistentClient
                                                        }
                                                        filters={filteringState.filters || []}
                                                        services={servicesState.allServices || []}
                                                        whitelistFilters={
                                                            filteringState.whitelistFilters || []
                                                        }
                                                        persistentClientIds={persistentClientIds()}
                                                        persistentClientsLoaded={
                                                            !dashboardState.processingClients
                                                        }
                                                    />
                                                )}
                                            </For>
                                        </div>

                                        <InfiniteScrollTrigger
                                            hasMore={hasMore()}
                                            loading={isLoadingMore()}
                                            disabled={isRequestInFlight()}
                                            onLoadMore={handleLoadMore}
                                            resetToken={infiniteScrollResetToken()}
                                            class={s.mobileLoader}
                                        />
                                    </>
                                }
                            >
                                <EmptyState class={s.emptyState} mode={emptyStateMode()} />
                            </Show>
                        }
                    >
                        <div class={s.mobileInitialLoader} data-testid="query-log-initial-loader">
                            <Loader color="green" class={s.loader} />
                        </div>
                    </Show>
                </div>

                <Show when={selectedEntry()}>
                    <DetailModal
                        entry={selectedEntry()!}
                        filters={filteringState.filters || []}
                        services={servicesState.allServices || []}
                        whitelistFilters={filteringState.whitelistFilters || []}
                        onClose={handleCloseDetail}
                        onBlock={handleBlockDomain}
                        onAddToAllowlist={handleUnblockDomain}
                        onAllowService={handleAllowService}
                    />
                </Show>

                <Show when={disallowTarget()}>
                    <DisallowDialog
                        ip={disallowTarget()!}
                        isAllowlistMode={allowedClients().length > 0}
                        onConfirm={handleConfirmDisallow}
                        onClose={handleCloseDisallow}
                    />
                </Show>
            </div>
        </div>
    );
};
