import React, { useEffect, useCallback, useState } from 'react';
import cn from 'clsx';
import { useSelector, useDispatch } from 'react-redux';
import { useHistory, useLocation } from 'react-router-dom';
import { Action } from 'redux';
import { ThunkDispatch } from 'redux-thunk';

import intl from 'panel/common/intl';
import { PageLoader } from 'panel/common/ui/Loader';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';
import {
    getLogs,
    setLogsFilter,
    setFilteredLogs,
    refreshFilteredLogs,
    getLogsConfig,
} from 'panel/actions/queryLogs';
import { toggleBlocking, toggleBlockingForClient } from 'panel/actions';
import { toggleClientBlock, getAccessList } from 'panel/actions/access';
import { DEFAULT_LOGS_FILTER, RESPONSE_FILTER_QUERIES } from 'panel/helpers/constants';
import { getLogsUrlParams } from 'panel/helpers/helpers';

import { LogEntry } from './types';
import { Header } from './blocks/Header';
import { LogTable } from './blocks/LogTable';
import { LogCard } from './blocks/LogCard';
import { DetailModal } from './blocks/DetailModal';
import { DisallowDialog } from './blocks/DisallowDialog';
import { InfiniteScrollTrigger } from './blocks/InfiniteScrollTrigger';

import s from './QueryLog.module.pcss';

export const QueryLog = () => {
    const dispatch = useDispatch<ThunkDispatch<RootState, unknown, Action<string>>>();
    const history = useHistory();
    const location = useLocation();

    const queryLogs = useSelector((state: RootState) => state.queryLogs);
    const access = useSelector((state: RootState) => state.access);
    const filters = useSelector((state: RootState) => state.filtering.filters);
    const whitelistFilters = useSelector((state: RootState) => state.filtering.whitelistFilters);
    const services = useSelector((state: RootState) => state.services?.allServices);

    const {
        logs = [],
        processingGetLogs,
        processingAdditionalLogs,
        filter,
        isEntireLog,
    } = queryLogs || {};

    const [selectedEntry, setSelectedEntry] = useState<LogEntry | null>(null);
    const [disallowTarget, setDisallowTarget] = useState<string | null>(null);
    const [isIncrementalLoad, setIsIncrementalLoad] = useState(false);

    useEffect(() => {
        dispatch(getLogsConfig());
        dispatch(getAccessList());
    }, [dispatch]);

    useEffect(() => {
        const searchParams = new URLSearchParams(location.search);
        const search = searchParams.get('search') || DEFAULT_LOGS_FILTER.search;
        const responseStatusParam = searchParams.get('response_status') || '';
        const response_status = Object.prototype.hasOwnProperty.call(
            RESPONSE_FILTER_QUERIES,
            responseStatusParam,
        )
            ? responseStatusParam
            : DEFAULT_LOGS_FILTER.response_status;
        const nextFilter = {
            search,
            response_status,
        };
        const nextSearch = getLogsUrlParams(search, response_status);

        if (location.search !== nextSearch) {
            history.replace(nextSearch);
            return;
        }

        setIsIncrementalLoad(false);
        dispatch(setLogsFilter(nextFilter));
        dispatch(setFilteredLogs(nextFilter));
    }, [dispatch, history, location.search]);

    useEffect(() => {
        if (!processingGetLogs && !processingAdditionalLogs) {
            setIsIncrementalLoad(false);
        }
    }, [processingAdditionalLogs, processingGetLogs]);

    const currentSearch = filter?.search ?? DEFAULT_LOGS_FILTER.search;
    const currentFilter = filter?.response_status ?? DEFAULT_LOGS_FILTER.response_status;
    const hasMore = !isEntireLog;

    const handleSearch = useCallback(
        (search: string) => {
            setIsIncrementalLoad(false);
            history.replace(getLogsUrlParams(search.trim(), currentFilter));
        },
        [currentFilter, history],
    );

    const handleFilterChange = useCallback(
        (response_status: string) => {
            setIsIncrementalLoad(false);
            history.replace(getLogsUrlParams(currentSearch, response_status));
        },
        [currentSearch, history],
    );

    const handleRefresh = useCallback(() => {
        setIsIncrementalLoad(false);
        dispatch(refreshFilteredLogs());
    }, [dispatch]);

    const handleToggleBlock = useCallback(
        (type: string, domain: string) => {
            dispatch(toggleBlocking(type, domain));
        },
        [dispatch],
    );

    const handleBlockClient = useCallback(
        (type: string, domain: string, client: string) => {
            dispatch(toggleBlockingForClient(type, domain, client));
        },
        [dispatch],
    );

    const handleDisallowClient = useCallback((ip: string) => {
        setDisallowTarget(ip);
    }, []);

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

    if (processingGetLogs && logs.length === 0) {
        return <PageLoader />;
    }

    return (
        <div className={theme.layout.container}>
            <div className={cn(theme.layout.containerIn, s.page)}>
                <Header
                    onSearch={handleSearch}
                    onRefresh={handleRefresh}
                    onFilterChange={handleFilterChange}
                    currentSearch={currentSearch}
                    currentFilter={currentFilter}
                    isLoading={!!isRequestInFlight}
                />

                <div className={s.desktopView}>
                    <LogTable
                        logs={logs}
                        hasMore={hasMore}
                        isLoadingMore={isLoadingMore}
                        isRequestInFlight={isRequestInFlight}
                        onLoadMore={handleLoadMore}
                        onRowClick={handleRowClick}
                        onBlock={handleToggleBlock}
                        onUnblock={handleToggleBlock}
                        onBlockClient={handleBlockClient}
                        onDisallowClient={handleDisallowClient}
                        onSearchSelect={handleSearch}
                        filters={filters}
                        services={services}
                        whitelistFilters={whitelistFilters}
                    />
                </div>

                <div className={s.mobileView}>
                    {logs.length === 0 ? (
                        <div className={s.emptyState}>
                            <span className={theme.text.t2}>{intl.getMessage('no_logs_found')}</span>
                        </div>
                    ) : (
                        <>
                            <div className={s.mobileList}>
                                {logs.map((entry: LogEntry) => (
                                    <LogCard
                                        key={`${entry.time}-${entry.domain}-${entry.client}`}
                                        entry={entry}
                                        onRowClick={handleRowClick}
                                        onBlock={handleToggleBlock}
                                        onUnblock={handleToggleBlock}
                                        onBlockClient={handleBlockClient}
                                        onDisallowClient={handleDisallowClient}
                                        filters={filters}
                                        services={services}
                                        whitelistFilters={whitelistFilters}
                                    />
                                ))}
                            </div>

                            <InfiniteScrollTrigger
                                hasMore={hasMore}
                                loading={isLoadingMore}
                                disabled={isRequestInFlight}
                                onLoadMore={handleLoadMore}
                                className={s.mobileLoader}
                            />
                        </>
                    )}
                </div>

                {selectedEntry && (
                    <DetailModal
                        entry={selectedEntry}
                        onClose={handleCloseDetail}
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
