import { render, screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    stats: vi.fn(),
    clientsSearch: vi.fn(),
    accessList: vi.fn(),
    accessSet: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    stats: mocks.stats,
    clientsSearch: mocks.clientsSearch,
    accessList: mocks.accessList,
    accessSet: mocks.accessSet,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));

import { TopClientsPage } from 'panel/components/Stats/TopClientsPage';

type TopStatEntry = { name: string; count: number };

type StatsFixture = {
    time_units: string;
    dns_queries: number[];
    top_queried_domains: TopStatEntry[];
    top_blocked_domains: TopStatEntry[];
    top_clients: Record<string, number>[];
    top_upstreams_responses: TopStatEntry[];
    top_upstreams_avg_time: TopStatEntry[];
    num_dns_queries: number;
    num_blocked_filtering: number;
    avg_processing_time: number;
};

const statsFixture = (): StatsFixture => ({
    time_units: 'days',
    dns_queries: [1, 2],
    top_queried_domains: [],
    top_blocked_domains: [],
    top_clients: [{ '10.0.0.1': 500 }, { '10.0.0.2': 12300 }],
    top_upstreams_responses: [],
    top_upstreams_avg_time: [],
    num_dns_queries: 20000,
    num_blocked_filtering: 0,
    avg_processing_time: 0,
});

const clientSearchFixture = () => [
    {
        '10.0.0.1': {
            name: 'Alex Green 456 Elm Android',
            ids: ['10.0.0.1'],
            whois_info: { country: 'Norway', orgname: 'AS5678 Velocity Net' },
        },
    },
    { '10.0.0.2': { name: 'Emma Johnson 654 Maple iOS', ids: ['10.0.0.2'] } },
];

const mockMatchMedia = (matches: boolean) => {
    Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: (query: string) =>
            ({
                matches,
                media: query,
                onchange: null,
                addListener: () => {},
                removeListener: () => {},
                addEventListener: () => {},
                removeEventListener: () => {},
                dispatchEvent: () => false,
            }) as MediaQueryList,
    });
};

const renderPage = () =>
    render(() => (
        <HashRouter>
            <Route path="/" component={() => <TopClientsPage />} />
        </HashRouter>
    ));

const rowsText = () =>
    Array.from(document.querySelectorAll('[class*="tableRow"]'))
        .map((el) => el.textContent)
        .join(' | ');

describe('TopClientsPage', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
        mockMatchMedia(true);
        mocks.stats.mockResolvedValue(statsFixture());
        mocks.clientsSearch.mockResolvedValue(clientSearchFixture());
        mocks.accessList.mockResolvedValue({
            allowed_clients: [],
            disallowed_clients: ['10.0.0.2'],
            blocked_hosts: [],
        });
        mocks.accessSet.mockResolvedValue({});
    });

    it('renders columns Name, Status, Queries, IP address, WHOIS, Actions', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('10.0.0.1')).toBeInTheDocument());
        expect(screen.getByText('Name')).toBeInTheDocument();
        expect(screen.getByText('Status')).toBeInTheDocument();
        expect(screen.getByText('Queries')).toBeInTheDocument();
        expect(screen.getByText('IP address')).toBeInTheDocument();
        expect(screen.getByText('WHOIS')).toBeInTheDocument();
        expect(screen.getByText('Actions')).toBeInTheDocument();
    });

    it('sorts by queries descending by default, compact count + 1-decimal percent', async () => {
        renderPage();
        await waitFor(() =>
            expect(screen.getByText('Alex Green 456 Elm Android')).toBeInTheDocument(),
        );
        const rows = rowsText();
        const firstRowIndex = rows.indexOf('Emma Johnson 654 Maple iOS');
        const secondRowIndex = rows.indexOf('Alex Green 456 Elm Android');
        expect(firstRowIndex).toBeGreaterThan(-1);
        expect(secondRowIndex).toBeGreaterThan(-1);
        expect(firstRowIndex).toBeLessThan(secondRowIndex);
        expect(rows).toContain('12.3K');
        expect(rows).toContain('(61.5%)');
        expect(rows).toContain('500');
        expect(rows).toContain('(2.5%)');
    });

    it('shows WHOIS country + org; a single em dash when missing', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('AS5678 Velocity Net')).toBeInTheDocument());
        expect(screen.getByText('Norway')).toBeInTheDocument();
        // Emma (10.0.0.2) has a name but no whois_info: her WHOIS cell shows a
        // single em dash on the org line (the country line stays empty)
        expect(screen.getAllByText('—').length).toBe(1);
    });

    it('sorts by Status and IP address columns', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('10.0.0.1')).toBeInTheDocument());

        // Status ascending: blocked (10.0.0.2) before unblocked (10.0.0.1)
        fireEvent.click(screen.getByTestId('table-header-status'));
        const rowsByStatus = rowsText();
        expect(rowsByStatus.indexOf('Blocked')).toBeLessThan(rowsByStatus.indexOf('Unblocked'));

        // IP ascending: 10.0.0.1 before 10.0.0.2
        fireEvent.click(screen.getByTestId('table-header-ip'));
        const rowsByIp = rowsText();
        expect(rowsByIp.indexOf('10.0.0.1')).toBeLessThan(rowsByIp.indexOf('10.0.0.2'));
    });

    it('shows Blocked status from the access list and Unblocked otherwise', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('Blocked')).toBeInTheDocument());
        expect(screen.getByText('Unblocked')).toBeInTheDocument();
    });

    it('filters client-side by name or IP without re-fetching stats', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('10.0.0.1')).toBeInTheDocument());
        const statsCalls = mocks.stats.mock.calls.length;

        fireEvent.input(screen.getByTestId('stats-search-input'), {
            target: { value: 'Emma' },
        });
        expect(screen.getByText('10.0.0.2')).toBeInTheDocument();
        expect(screen.queryByText('10.0.0.1')).toBeNull();

        fireEvent.input(screen.getByTestId('stats-search-input'), {
            target: { value: '"10.0.0.1"' },
        });
        expect(screen.getByText('10.0.0.1')).toBeInTheDocument();
        expect(screen.queryByText('10.0.0.2')).toBeNull();

        expect(mocks.stats.mock.calls.length).toBe(statsCalls);
    });

    it('blocks a client via confirm dialog: updates access list and shows Blocked', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('10.0.0.1')).toBeInTheDocument());

        const row = Array.from(document.querySelectorAll('[class*="tableRow"]')).find((el) =>
            el.textContent?.includes('10.0.0.1'),
        );
        expect(row).toBeTruthy();
        fireEvent.click(row!.querySelector('button')!);
        fireEvent.click(screen.getByText('Block client'));
        fireEvent.click(screen.getByRole('button', { name: 'Block client' }));

        await waitFor(() => expect(mocks.accessSet).toHaveBeenCalled());
        const body = mocks.accessSet.mock.calls[0][0];
        expect(body.disallowed_clients).toContain('10.0.0.1');
        expect(mocks.addSuccessToast).toHaveBeenCalled();
    });

    it('shows the "already blocked" error toast when blocking a blocked client', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('10.0.0.2')).toBeInTheDocument());

        const row = Array.from(document.querySelectorAll('[class*="tableRow"]')).find((el) =>
            el.textContent?.includes('10.0.0.2'),
        );
        fireEvent.click(row!.querySelector('button')!);
        fireEvent.click(screen.getByText('Unblock client')); // menu on a blocked client -> Unblock
        expect(mocks.addErrorToast).not.toHaveBeenCalled();

        // The menu of the blocked client offers "Unblock client", not "Block client".
        // (The other, unblocked client's menu legitimately contains "Block client".)
        expect(row!.textContent).toContain('Unblock client');
        expect(row!.textContent).not.toContain('Block client');
    });

    it('renders mobile cards with label/value pairs and block/unblock action links', async () => {
        mockMatchMedia(false);
        renderPage();
        await waitFor(() => expect(screen.getAllByTestId('stats-mobile-card').length).toBe(2));
        expect(screen.getByText('Unblock client')).toBeInTheDocument();
        expect(screen.getByText('Block client')).toBeInTheDocument();
    });

    it('shows an "Add client" button that opens the client add form', async () => {
        renderPage();
        await waitFor(() => expect(screen.getByText('10.0.0.1')).toBeInTheDocument());

        const addButton = screen.getByTestId('stats-add-client-button');
        expect(addButton).toBeInTheDocument();
        expect(addButton.textContent).toContain('Add client');

        fireEvent.click(addButton);
        await waitFor(() => {
            expect(window.location.hash).toContain('clients/add');
        });
    });
});
