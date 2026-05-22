import React, { useEffect, useCallback, useState } from 'react';
import cn from 'clsx';
import { batch, useSelector, useDispatch } from 'react-redux';
import { useHistory, useLocation } from 'react-router-dom';
import { Action } from 'redux';
import { ThunkDispatch } from 'redux-thunk';

import { Loader } from 'panel/common/ui/Loader';
import { initialState, RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';
import {
    getLogs,
    setLogsFilter,
    setFilteredLogs,
    refreshFilteredLogs,
    getLogsConfig,
} from 'panel/actions/queryLogs';
import { blockDomain, blockDomainForClient, getClients, unblockDomain } from 'panel/actions';
import { toggleClientBlock, getAccessList } from 'panel/actions/access';
import { getFilteringStatus } from 'panel/actions/filtering';
import { allowBlockedService, getAllBlockedServices } from 'panel/actions/services';
import {
    DEFAULT_LOGS_FILTER,
    QUERY_LOG_REASON_FILTER_QUERIES,
    QUERY_LOG_STATUS_FILTER_QUERIES,
} from 'panel/helpers/constants';
import { getLogsUrlParams } from 'panel/helpers/helpers';
import { Paths } from 'panel/components/Routes/Paths';

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
    const dispatch = useDispatch<ThunkDispatch<RootState, unknown, Action<string>>>();
    const history = useHistory();
    const location = useLocation();

    const queryLogs = useSelector((state: RootState) => state.queryLogs) ?? initialState.queryLogs;
    const access = useSelector((state: RootState) => state.access);
    const persistentClients = useSelector((state: RootState) => state.dashboard.clients);
    const processingClients = useSelector((state: RootState) => state.dashboard.processingClients);
    const filters = useSelector((state: RootState) => state.filtering.filters);
    const whitelistFilters = useSelector((state: RootState) => state.filtering.whitelistFilters);
    const services = useSelector((state: RootState) => state.services?.allServices ?? []);

    const {
        logs = [],
        processingGetLogs,
        processingAdditionalLogs,
        filter,
        isEntireLog,
        enabled,
        interval,
    } = queryLogs;

    const [selectedEntry, setSelectedEntry] = useState<LogEntry | null>(null);
    const [disallowTarget, setDisallowTarget] = useState<string | null>(null);
    const [isIncrementalLoad, setIsIncrementalLoad] = useState(false);

    useEffect(() => {
        batch(() => {
            dispatch(getLogsConfig());
            dispatch(getAccessList());
            dispatch(getClients());
            dispatch(getFilteringStatus());
            dispatch(getAllBlockedServices());
        });
    }, [dispatch]);

    useEffect(() => {
        const searchParams = new URLSearchParams(location.search);
        const search = searchParams.get('search') || DEFAULT_LOGS_FILTER.search;
        const statusParam = searchParams.get('status') || DEFAULT_LOGS_FILTER.status;
        const reasonParam = searchParams.get('reason') || DEFAULT_LOGS_FILTER.reason;
        const status = Object.prototype.hasOwnProperty.call(QUERY_LOG_STATUS_FILTER_QUERIES, statusParam)
            ? statusParam
            : DEFAULT_LOGS_FILTER.status;
        const reason = Object.prototype.hasOwnProperty.call(
            QUERY_LOG_REASON_FILTER_QUERIES,
            reasonParam,
        )
            ? reasonParam
            : DEFAULT_LOGS_FILTER.reason;
        const nextFilter = {
            search,
            status,
            reason,
        };
        const nextSearch = getLogsUrlParams(search, status, reason);

        if (location.search !== nextSearch) {
            history.replace(nextSearch);
            return;
        }

        setIsIncrementalLoad(false);
        dispatch(setLogsFilter(nextFilter));
        dispatch(setFilteredLogs(nextFilter));
    }, [history, location.search]);

    useEffect(() => {
        if (!processingGetLogs && !processingAdditionalLogs) {
            setIsIncrementalLoad(false);
        }
    }, [processingAdditionalLogs, processingGetLogs]);

    const currentSearch = filter?.search ?? DEFAULT_LOGS_FILTER.search;
    const currentStatus = filter?.status ?? DEFAULT_LOGS_FILTER.status;
    const currentReason = filter?.reason ?? DEFAULT_LOGS_FILTER.reason;
    const infiniteScrollResetToken = `${currentSearch}:${currentStatus}:${currentReason}`;
    const persistentClientIds = persistentClients.flatMap((persistentClient) => persistentClient.ids ?? []);
    const visibleLogs = filterLogsByStatus(logs, currentStatus);
    const emptyStateMode = getEmptyStateMode(enabled, interval);
    const hasMore = !isEntireLog;

    const handleSearch = useCallback(
        (search: string) => {
            setIsIncrementalLoad(false);
            history.replace(getLogsUrlParams(search.trim(), currentStatus, currentReason));
        },
        [currentReason, currentStatus, history],
    );

    const handleStatusFilterChange = useCallback(
        (status: string) => {
            setIsIncrementalLoad(false);
            history.replace(getLogsUrlParams(currentSearch, status, DEFAULT_LOGS_FILTER.reason));
        },
        [currentSearch, history],
    );

    const handleReasonFilterChange = useCallback(
        (reason: string) => {
            setIsIncrementalLoad(false);
            history.replace(getLogsUrlParams(currentSearch, currentStatus, reason));
        },
        [currentSearch, currentStatus, history],
    );

    const handleRefresh = useCallback(() => {
        setIsIncrementalLoad(false);
        dispatch(refreshFilteredLogs());
    }, [dispatch]);

    const handleBlockDomain = useCallback(
        (domain: string) => {
            dispatch(blockDomain(domain));
        },
        [dispatch],
    );

    const handleUnblockDomain = useCallback(
        (domain: string) => {
            dispatch(unblockDomain(domain));
        },
        [dispatch],
    );

    const handleAllowService = useCallback(
        (serviceId: string) => {
            dispatch(allowBlockedService(serviceId));
        },
        [dispatch],
    );

    const handleBlockClient = useCallback(
        (domain: string, client: string) => {
            dispatch(blockDomainForClient(domain, client));
        },
        [dispatch],
    );

    const handleDisallowClient = useCallback((ip: string) => {
        setDisallowTarget(ip);
    }, []);

    const handleAddPersistentClient = useCallback(
        (clientId: string) => {
            history.push(`${Paths.Clients}?clientId=${encodeURIComponent(clientId)}`);
        },
        [history],
    );

    const handleConfirmDisallow = useCallback(() => {
        if (disallowTarget) {
            dispatch(toggleClientBlock(disallowTarget, false, ''));
            setDisallowTarget(null);
        }
    }, [dispatch, disallowTarget]);

    const handleCloseDisallow = useCallback(() => {
        setDisallowTarget(null);
    }, []);

    const handleRowClick = useCallback((entry: LogEntry) => {
        setSelectedEntry(entry);
    }, []);

    const handleCloseDetail = useCallback(() => {
        setSelectedEntry(null);
    }, []);

    const isRequestInFlight = processingGetLogs || processingAdditionalLogs;
    const isLoadingMore = isIncrementalLoad && isRequestInFlight;
    const isInitialLoading = processingGetLogs && logs.length === 0 && !isIncrementalLoad;
    const isFilterReloading = processingGetLogs && !isInitialLoading && !isIncrementalLoad;

    const handleLoadMore = useCallback(() => {
        if (isRequestInFlight || isEntireLog) {
            return;
        }

        setIsIncrementalLoad(true);
        dispatch(getLogs(currentSearch));
    }, [currentSearch, dispatch, isEntireLog, isRequestInFlight]);

    const getAllowedClients = (): string[] => {
        const raw = access?.allowed_clients;
        if (!raw) {
            return [];
        }
        if (typeof raw === 'string') {
            return raw.split('\n').filter(Boolean);
        }
        if (Array.isArray(raw)) {
            return raw;
        }
        return [];
    };
    const allowedClients = getAllowedClients();

    const renderMobileContent = () => {
        if (isInitialLoading || (isFilterReloading && visibleLogs.length === 0)) {
            return (
                <div className={s.mobileInitialLoader} data-testid="query-log-initial-loader">
                    <Loader color="green" className={s.loader} />
                </div>
            );
        }

        if (visibleLogs.length === 0) {
            return (
                <EmptyState
                    className={s.emptyState}
                    mode={emptyStateMode}
                />
            );
        }

        return (
            <>
                <div className={s.mobileList}>
                    {visibleLogs.map((entry: LogEntry) => (
                        <LogCard
                            key={`${entry.time}-${entry.domain}-${entry.client}`}
                            entry={entry}
                            onRowClick={handleRowClick}
                            onBlock={handleBlockDomain}
                            onUnblock={handleUnblockDomain}
                            onBlockClient={handleBlockClient}
                            onDisallowClient={handleDisallowClient}
                            onAddPersistentClient={handleAddPersistentClient}
                            filters={filters}
                            services={services}
                            whitelistFilters={whitelistFilters}
                            persistentClientIds={persistentClientIds}
                            persistentClientsLoaded={!processingClients}
                        />
                    ))}
                </div>

                <InfiniteScrollTrigger
                    hasMore={hasMore}
                    loading={isLoadingMore}
                    disabled={isRequestInFlight}
                    onLoadMore={handleLoadMore}
                    resetToken={infiniteScrollResetToken}
                    className={s.mobileLoader}
                />
            </>
        );
    }

    return (
        <div className={theme.layout.container}>
            <div className={cn(theme.layout.containerIn, s.page)}>
                <Header
                    onSearch={handleSearch}
                    onRefresh={handleRefresh}
                    onStatusFilterChange={handleStatusFilterChange}
                    onReasonFilterChange={handleReasonFilterChange}
                    currentSearch={currentSearch}
                    currentStatus={currentStatus}
                    currentReason={currentReason}
                    isLoading={!!isRequestInFlight}
                />

                <div className={s.desktopView}>
                    <LogTable
                        logs={visibleLogs}
                        emptyStateMode={emptyStateMode}
                        hasMore={hasMore}
                        isLoadingMore={isLoadingMore}
                        isRequestInFlight={isRequestInFlight}
                        isInitialLoading={isInitialLoading}
                        isFilterReloading={isFilterReloading}
                        infiniteScrollResetToken={infiniteScrollResetToken}
                        onLoadMore={handleLoadMore}
                        onRowClick={handleRowClick}
                        onBlock={handleBlockDomain}
                        onUnblock={handleUnblockDomain}
                        onBlockClient={handleBlockClient}
                        onDisallowClient={handleDisallowClient}
                        onAddPersistentClient={handleAddPersistentClient}
                        onSearchSelect={handleSearch}
                        filters={filters}
                        services={services}
                        whitelistFilters={whitelistFilters}
                        persistentClientIds={persistentClientIds}
                        persistentClientsLoaded={!processingClients}
                    />
                </div>

                <div className={s.mobileView}>
                    {renderMobileContent()}
                </div>

                {selectedEntry && (
                    <DetailModal
                        entry={selectedEntry}
                        filters={filters}
                        services={services}
                        whitelistFilters={whitelistFilters}
                        onClose={handleCloseDetail}
                        onBlock={handleBlockDomain}
                        onAddToAllowlist={handleUnblockDomain}
                        onAllowService={handleAllowService}
                    />
                )}

                {disallowTarget && (
                    <DisallowDialog
                        ip={disallowTarget}
                        isAllowlistMode={allowedClients.length > 0}
                        onConfirm={handleConfirmDisallow}
                        onClose={handleCloseDisallow}
                    />
                )}
            </div>
        </div>
    );
};
