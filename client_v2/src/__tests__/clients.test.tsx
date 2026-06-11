import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { RootState } from 'panel/initialState';
import { initialState } from 'panel/initialState';
import { Clients } from 'panel/components/Clients/Clients';

const mocks = vi.hoisted(() => ({
    dispatch: vi.fn((action) => action),
    state: null as unknown as RootState,
    getClients: vi.fn(() => ({ type: 'GET_CLIENTS' })),
    getStats: vi.fn(() => ({ type: 'GET_STATS' })),
    toggleClientModal: vi.fn(() => ({ type: 'TOGGLE_CLIENT_MODAL' })),
    deleteClient: vi.fn(() => ({ type: 'DELETE_CLIENT' })),
    initClientForm: vi.fn(() => ({ type: 'INIT_CLIENT_FORM' })),
    getAllBlockedServices: vi.fn(() => ({ type: 'GET_ALL_BLOCKED_SERVICES' })),
}));

vi.mock('react-redux', () => ({
    batch: (fn: () => void) => fn(),
    useDispatch: () => mocks.dispatch,
    useSelector: (selector: (state: RootState) => unknown) => selector(mocks.state),
}));

vi.mock('panel/actions', () => ({
    getClients: mocks.getClients,
}));

vi.mock('panel/actions/stats', () => ({
    getStats: mocks.getStats,
}));

vi.mock('panel/actions/clients', () => ({
    toggleClientModal: mocks.toggleClientModal,
    deleteClient: mocks.deleteClient,
}));

vi.mock('panel/actions/services', () => ({
    getAllBlockedServices: mocks.getAllBlockedServices,
}));

vi.mock('panel/actions/clientForm', () => ({
    initClientForm: mocks.initClientForm,
    getAllBlockedServices: mocks.getAllBlockedServices,
}));

vi.mock('react-router-dom', () => ({
    HashRouter: ({ children }: { children: React.ReactNode }) => <>{children}</>,
    Route: ({ component: Component }: { component: React.ComponentType }) => (
        <>{Component ? <Component /> : null}</>
    ),
    MemoryRouter: ({ children }: { children: React.ReactNode }) => <>{children}</>,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
        <a href={to}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
}));

describe('Clients Page', () => {
    beforeEach(() => {
        mocks.state = JSON.parse(JSON.stringify(initialState));
        mocks.dispatch.mockClear();
        mocks.getClients.mockClear();
        mocks.getStats.mockClear();
        mocks.toggleClientModal.mockClear();
        mocks.deleteClient.mockClear();
        mocks.initClientForm.mockClear();
        mocks.getAllBlockedServices.mockClear();
        localStorage.clear();
    });

    it('dispatches getClients on mount', async () => {
        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;

        render(<Clients />);

        await waitFor(() => {
            expect(mocks.getClients).toHaveBeenCalledTimes(1);
        });
    });

    it('dispatches getStats on mount', async () => {
        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;

        render(<Clients />);

        await waitFor(() => {
            expect(mocks.getStats).toHaveBeenCalledTimes(1);
        });
    });

    it('triggers the add client flow when the add button is clicked', async () => {
        const user = userEvent.setup();

        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;

        render(<Clients />);

        await user.click(screen.getByRole('button', { name: 'Add Client' }));

        expect(mocks.initClientForm).toHaveBeenCalledWith(null);
    });

    it('renders persistent client rows with correct data', async () => {
        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = [
            {
                name: 'My Laptop',
                ids: ['192.168.1.10', '00:11:22:33:44:55'],
                use_global_settings: true,
                filtering_enabled: true,
                parental_enabled: false,
                safebrowsing_enabled: false,
                safe_search: {},
                safesearch_enabled: false,
                use_global_blocked_services: true,
                blocked_services: [],
                blocked_services_schedule: { time_zone: 'UTC' },
                upstreams: [],
                upstreams_cache_enabled: false,
                upstreams_cache_size: 0,
                tags: ['user_admin'],
                ignore_querylog: false,
                ignore_statistics: false,
            },
        ];
        mocks.state.dashboard.autoClients = [];
        mocks.state.stats.normalizedTopClients = {
            auto: {},
            configured: { 'My Laptop': 1234 },
        };

        render(<Clients />);

        await waitFor(() => {
            expect(screen.getByText('My Laptop')).toBeInTheDocument();
            // IP address text includes a trailing comma when multiple IDs exist
            expect(screen.getByText(/192\.168\.1\.10/)).toBeInTheDocument();
            // Second ID is inside a dropdown tooltip, not visible by default
            expect(screen.getByText('user_admin')).toBeInTheDocument();
            // Requests count uses locale-specific thousands separator (space or comma)
            expect(screen.getByText((content) => content.includes('234'))).toBeInTheDocument();
        });
    });

    it('renders runtime client rows with correct data', async () => {
        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = [];
        mocks.state.dashboard.autoClients = [
            {
                ip: '10.0.0.5',
                name: 'phone.local',
                source: 'rDNS',
                whois_info: { country: 'US', org: 'Cloudflare' },
            },
        ];
        mocks.state.stats.normalizedTopClients = {
            auto: { '10.0.0.5': 567 },
            configured: {},
        };

        render(<Clients />);

        await waitFor(() => {
            expect(screen.getByText('10.0.0.5')).toBeInTheDocument();
            expect(screen.getByText('phone.local')).toBeInTheDocument();
            expect(screen.getByText('rDNS')).toBeInTheDocument();
            expect(screen.getByText('567')).toBeInTheDocument();
        });
    });

    it('uses the persisted persistent-clients page size from localStorage', async () => {
        localStorage.setItem('clients_page_size', JSON.stringify(5));

        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = Array.from({ length: 6 }, (_, index) => ({
            name: `Client ${index + 1}`,
            ids: [`192.168.1.${index + 1}`],
            use_global_settings: true,
            filtering_enabled: true,
            parental_enabled: false,
            safebrowsing_enabled: false,
            safe_search: {},
            safesearch_enabled: false,
            use_global_blocked_services: true,
            blocked_services: [] as string[],
            blocked_services_schedule: { time_zone: 'UTC' },
            upstreams: [] as string[],
            upstreams_cache_enabled: false,
            upstreams_cache_size: 0,
            tags: [] as string[],
            ignore_querylog: false,
            ignore_statistics: false,
        }));
        mocks.state.dashboard.autoClients = [];

        render(<Clients />);

        await waitFor(() => {
            expect(screen.getByText('Client 1')).toBeInTheDocument();
            expect(screen.getByText('Client 5')).toBeInTheDocument();
            expect(screen.queryByText('Client 6')).not.toBeInTheDocument();
        });
    });

    it('displays WHOIS tooltip content on hover', async () => {
        const user = userEvent.setup();

        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = [];
        mocks.state.dashboard.autoClients = [
            {
                ip: '10.0.0.5',
                name: 'device',
                source: 'rDNS',
                whois_info: { country: 'DE', org: 'Hetzner' },
            },
        ];

        render(<Clients />);

        // The WHOIS cell now shows country code "DE" inline — hover it to show tooltip
        const countryLabel = await screen.findByText('DE');

        await user.hover(countryLabel);

        await waitFor(() => {
            // Tooltip title appears on hover
            expect(screen.getByText('Client details')).toBeInTheDocument();
            // Hetzner appears both inline and in tooltip
            const hetznerElements = screen.getAllByText(/Hetzner/);
            expect(hetznerElements.length).toBeGreaterThanOrEqual(2);
        });
    });

    it('renders dash for runtime client without WHOIS', async () => {
        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = [];
        mocks.state.dashboard.autoClients = [
            {
                ip: '10.0.0.6',
                name: 'unknown',
                source: 'ARP',
                whois_info: {},
            },
        ];

        render(<Clients />);

        await waitFor(() => {
            expect(screen.getByText('10.0.0.6')).toBeInTheDocument();
            expect(screen.getAllByText('-').length).toBeGreaterThan(0);
        });
    });

    it('allows sorting persistent clients table by name', async () => {
        const user = userEvent.setup();

        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = [
            {
                name: 'Zebra',
                ids: ['10.0.0.1'],
                use_global_settings: true,
                filtering_enabled: true,
                parental_enabled: false,
                safebrowsing_enabled: false,
                safe_search: {},
                safesearch_enabled: false,
                use_global_blocked_services: true,
                blocked_services: [],
                blocked_services_schedule: { time_zone: 'UTC' },
                upstreams: [],
                upstreams_cache_enabled: false,
                upstreams_cache_size: 0,
                tags: [],
                ignore_querylog: false,
                ignore_statistics: false,
            },
            {
                name: 'Alpha',
                ids: ['10.0.0.2'],
                use_global_settings: false,
                filtering_enabled: true,
                parental_enabled: false,
                safebrowsing_enabled: false,
                safe_search: {},
                safesearch_enabled: false,
                use_global_blocked_services: false,
                blocked_services: [],
                blocked_services_schedule: { time_zone: 'UTC' },
                upstreams: ['1.1.1.1'],
                upstreams_cache_enabled: false,
                upstreams_cache_size: 0,
                tags: [],
                ignore_querylog: false,
                ignore_statistics: false,
            },
        ];
        mocks.state.dashboard.autoClients = [];
        mocks.state.stats.normalizedTopClients = {
            auto: {},
            configured: { Zebra: 100, Alpha: 200 },
        };

        render(<Clients />);

        // Before sorting: Zebra appears first (API order)
        const rowsBefore = screen.getAllByText(/Zebra|Alpha/);
        const firstRowBefore = rowsBefore[0].closest('[class*="tableRow"]');
        expect(firstRowBefore).toHaveTextContent('Zebra');

        // Click the "Name" header to sort ascending
        // getAllByText matches both header cells and mobile cell labels;
        // persistent table header renders first in DOM
        const nameHeaders = screen.getAllByText('Name');
        const persistentNameHeader = nameHeaders[0];
        await user.click(persistentNameHeader);

        // After sorting ascending, Alpha should appear before Zebra
        await waitFor(() => {
            const rows = screen.getAllByText(/Alpha|Zebra/);
            const firstRow = rows[0].closest('[class*="tableRow"]');
            expect(firstRow).toHaveTextContent('Alpha');
        });
    });

    it('allows sorting runtime clients table by requests count', async () => {
        const user = userEvent.setup();

        mocks.state.dashboard.processingClients = false;
        mocks.state.stats.processingStats = false;
        mocks.state.dashboard.clients = [];
        mocks.state.dashboard.autoClients = [
            { ip: '10.0.0.1', name: 'host-a', source: 'DHCP', whois_info: {} },
            { ip: '10.0.0.2', name: 'host-b', source: 'rDNS', whois_info: {} },
        ];
        mocks.state.stats.normalizedTopClients = {
            auto: { '10.0.0.1': 10, '10.0.0.2': 500 },
            configured: {},
        };

        render(<Clients />);

        // Click "Requests" header to sort ascending (lowest first)
        // getAllByText matches both header cells and mobile cell labels;
        // persistent table header renders first in DOM, runtime second
        const requestsHeaders = screen.getAllByText('Requests');
        const runtimeRequestsHeader = requestsHeaders[1];
        await user.click(runtimeRequestsHeader);

        // host-a has 10 requests, host-b has 500 — ascending: host-a first
        await waitFor(() => {
            const rows = screen.getAllByText(/host-a|host-b/);
            const firstRow = rows[0].closest('[class*="tableRow"]');
            expect(firstRow).toHaveTextContent('host-a');
        });
    });
});
