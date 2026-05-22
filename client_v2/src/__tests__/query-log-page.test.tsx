import React from 'react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { render } from '@testing-library/react';

import { QueryLog } from 'panel/components/QueryLog/QueryLog';

const {
    batch,
    dispatch,
    historyPush,
    historyReplace,
    mockState,
    getAccessList,
    getAllBlockedServices,
    getClients,
    getFilteringStatus,
    getLogs,
    getLogsConfig,
    refreshFilteredLogs,
    setFilteredLogs,
    setLogsFilter,
} = vi.hoisted(() => ({
    batch: vi.fn((callback: () => void) => callback()),
    dispatch: vi.fn(),
    historyPush: vi.fn(),
    historyReplace: vi.fn(),
    mockState: {
        access: {},
        dashboard: {
            clients: [] as Array<{ ids?: string[] }>,
            processingClients: false,
        },
        filtering: {
            filters: [] as Array<{ id: number; name: string }>,
            whitelistFilters: [] as Array<{ id: number; name: string }>,
        },
        queryLogs: {
            enabled: true,
            filter: {
                reason: 'all',
                search: '',
                status: 'all',
            },
            interval: 24,
            isEntireLog: true,
            logs: [] as Array<Record<string, never>>,
            processingAdditionalLogs: false,
            processingGetLogs: false,
        },
        services: {
            allServices: [] as Array<{ id: string; name: string }>,
        },
    },
    getAccessList: vi.fn(() => ({ type: 'GET_ACCESS_LIST' })),
    getAllBlockedServices: vi.fn(() => ({ type: 'GET_ALL_BLOCKED_SERVICES' })),
    getClients: vi.fn(() => ({ type: 'GET_CLIENTS' })),
    getFilteringStatus: vi.fn(() => ({ type: 'GET_FILTERING_STATUS' })),
    getLogs: vi.fn(() => ({ type: 'GET_LOGS' })),
    getLogsConfig: vi.fn(() => ({ type: 'GET_LOGS_CONFIG' })),
    refreshFilteredLogs: vi.fn(() => ({ type: 'REFRESH_FILTERED_LOGS' })),
    setFilteredLogs: vi.fn((filter) => ({ payload: filter, type: 'SET_FILTERED_LOGS' })),
    setLogsFilter: vi.fn((filter) => ({ payload: filter, type: 'SET_LOGS_FILTER' })),
}));

vi.mock('react-redux', () => ({
    batch,
    useDispatch: () => dispatch,
    useSelector: (selector: (state: typeof mockState) => unknown) => selector(mockState),
}));

vi.mock('react-router-dom', () => ({
    useHistory: () => ({
        push: historyPush,
        replace: historyReplace,
    }),
    useLocation: () => ({
        search: '?status=all&reason=all',
    }),
}));

vi.mock('panel/actions/queryLogs', () => ({
    getLogs,
    getLogsConfig,
    refreshFilteredLogs,
    setFilteredLogs,
    setLogsFilter,
}));

vi.mock('panel/actions', () => ({
    blockDomain: vi.fn(() => ({ type: 'BLOCK_DOMAIN' })),
    blockDomainForClient: vi.fn(() => ({ type: 'BLOCK_DOMAIN_FOR_CLIENT' })),
    getClients,
    unblockDomain: vi.fn(() => ({ type: 'UNBLOCK_DOMAIN' })),
}));

vi.mock('panel/actions/access', () => ({
    getAccessList,
    toggleClientBlock: vi.fn(() => ({ type: 'TOGGLE_CLIENT_BLOCK' })),
}));

vi.mock('panel/actions/filtering', () => ({
    getFilteringStatus,
}));

vi.mock('panel/actions/services', () => ({
    allowBlockedService: vi.fn(() => ({ type: 'ALLOW_BLOCKED_SERVICE' })),
    getAllBlockedServices,
}));

vi.mock('panel/common/ui/Loader', () => ({
    Loader: (): React.JSX.Element => <div>loader</div>,
}));

vi.mock('panel/lib/theme', () => ({
    default: {
        compact: false,
        layout: {
            container: 'container',
            containerIn: 'containerIn',
        },
        status: {
            statusBlue: 'statusBlue',
            statusGreen: 'statusGreen',
            statusRed: 'statusRed',
            statusYellow: 'statusYellow',
        },
    },
}));

vi.mock('panel/components/QueryLog/blocks/Header', () => ({
    Header: (): React.JSX.Element => <div data-testid="query-log-header" />,
}));

vi.mock('panel/components/QueryLog/blocks/EmptyState/EmptyState', () => ({
    EmptyState: (): React.JSX.Element => <div data-testid="query-log-empty-state" />,
}));

vi.mock('panel/components/QueryLog/blocks/LogTable', () => ({
    LogTable: (): React.JSX.Element => <div data-testid="query-log-table" />,
}));

vi.mock('panel/components/QueryLog/blocks/LogCard', () => ({
    LogCard: (): React.JSX.Element => <div data-testid="query-log-card" />,
}));

vi.mock('panel/components/QueryLog/blocks/DetailModal', () => ({
    DetailModal: (): null => null,
}));

vi.mock('panel/components/QueryLog/blocks/DisallowDialog', () => ({
    DisallowDialog: (): null => null,
}));

vi.mock('panel/components/QueryLog/blocks/InfiniteScrollTrigger', () => ({
    InfiniteScrollTrigger: (): null => null,
}));

afterEach(() => {
    vi.clearAllMocks();
});

describe('QueryLog page', () => {
    test('requests filtering status on mount so filter IDs can resolve to names', () => {
        render(<QueryLog />);

        expect(getLogsConfig).toHaveBeenCalledTimes(1);
        expect(getAccessList).toHaveBeenCalledTimes(1);
        expect(getClients).toHaveBeenCalledTimes(1);
        expect(getFilteringStatus).toHaveBeenCalledTimes(1);
        expect(getAllBlockedServices).toHaveBeenCalledTimes(1);
        expect(batch).toHaveBeenCalledTimes(1);
        expect(setLogsFilter).toHaveBeenCalledWith({
            reason: 'all',
            search: '',
            status: 'all',
        });
        expect(setFilteredLogs).toHaveBeenCalledWith({
            reason: 'all',
            search: '',
            status: 'all',
        });

        expect(dispatch).toHaveBeenCalledWith({ type: 'GET_LOGS_CONFIG' });
        expect(dispatch).toHaveBeenCalledWith({ type: 'GET_ACCESS_LIST' });
        expect(dispatch).toHaveBeenCalledWith({ type: 'GET_CLIENTS' });
        expect(dispatch).toHaveBeenCalledWith({ type: 'GET_FILTERING_STATUS' });
        expect(dispatch).toHaveBeenCalledWith({ type: 'GET_ALL_BLOCKED_SERVICES' });
        expect(historyReplace).not.toHaveBeenCalled();
    });
});
