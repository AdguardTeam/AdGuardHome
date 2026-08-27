import { render, screen, waitFor } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { JSX } from 'solid-js';

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

import { TopQueriedDomainsPage } from 'panel/components/Stats/TopQueriedDomainsPage';
import { TopBlockedDomainsPage } from 'panel/components/Stats/TopBlockedDomainsPage';
import { TopUpstreamsPage } from 'panel/components/Stats/TopUpstreamsPage';
import { UpstreamAvgTimePage } from 'panel/components/Stats/UpstreamAvgTimePage';

type TopStatEntry = { name: string; count: number };

type StatsFixture = {
    time_units: string;
    dns_queries: number[];
    top_clients: Record<string, number>[];
    top_queried_domains: TopStatEntry[];
    top_blocked_domains: TopStatEntry[];
    top_upstreams_responses: TopStatEntry[];
    top_upstreams_avg_time: TopStatEntry[];
    num_dns_queries: number;
    num_blocked_filtering: number;
    avg_processing_time: number;
};

const baseFixture = (): StatsFixture => ({
    time_units: 'days',
    dns_queries: [1, 2],
    top_clients: [],
    top_queried_domains: [],
    top_blocked_domains: [],
    top_upstreams_responses: [],
    top_upstreams_avg_time: [],
    num_dns_queries: 10000,
    num_blocked_filtering: 0,
    avg_processing_time: 0,
});

const rowsText = () =>
    Array.from(document.querySelectorAll('[class*="tableRow"]'))
        .map((el) => el.textContent)
        .join(' | ');

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

const renderPage = (page: () => JSX.Element) =>
    render(() => (
        <HashRouter>
            <Route path="/" component={page} />
        </HashRouter>
    ));

describe('Domains detail pages', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
        mockMatchMedia(true);
        mocks.stats.mockResolvedValue(baseFixture());
        mocks.clientsSearch.mockResolvedValue([]);
        mocks.accessList.mockResolvedValue({
            allowed_clients: [],
            disallowed_clients: [],
            blocked_hosts: [],
        });
        mocks.accessSet.mockResolvedValue({});
    });

    it('Top queried domains: columns Domain/Queries, count + 1-decimal % of total queries, desc default', async () => {
        mocks.stats.mockResolvedValue({
            ...baseFixture(),
            top_queried_domains: [{ 'a.org': 10 }, { 'b.org': 2000 }],
        });
        renderPage(() => <TopQueriedDomainsPage />);
        await waitFor(() => expect(screen.getByText('b.org')).toBeInTheDocument());
        expect(screen.getByText('Domain')).toBeInTheDocument();
        expect(screen.getByText('Queries')).toBeInTheDocument();

        const rows = rowsText();
        const bIndex = rows.indexOf('b.org');
        const aIndex = rows.indexOf('a.org');
        expect(bIndex).toBeGreaterThan(-1);
        expect(aIndex).toBeGreaterThan(-1);
        expect(bIndex).toBeLessThan(aIndex);
        expect(rows).toContain('2K');
        expect(rows).toContain('(20.0%)');
        expect(rows).toContain('(0.1%)');
    });

    it('Top blocked domains: label "Blocked queries", % against total blocked, zero total → 0%', async () => {
        mocks.stats.mockResolvedValue({
            ...baseFixture(),
            top_blocked_domains: [{ 'ads.org': 30 }],
            num_blocked_filtering: 300,
        });
        renderPage(() => <TopBlockedDomainsPage />);
        await waitFor(() => expect(screen.getByText('ads.org')).toBeInTheDocument());
        expect(screen.getByText('Blocked queries')).toBeInTheDocument();
        expect(screen.getByText('(10.0%)')).toBeInTheDocument();
    });

    it('Top blocked domains: renders an empty state when nothing is blocked', async () => {
        mocks.stats.mockResolvedValue({
            ...baseFixture(),
            top_blocked_domains: [],
            num_blocked_filtering: 0,
        });
        renderPage(() => <TopBlockedDomainsPage />);
        await waitFor(() => expect(screen.getByText('Nothing found')).toBeInTheDocument());
    });

    it('Top upstreams: columns Upstream/Queries, % of total queries', async () => {
        mocks.stats.mockResolvedValue({
            ...baseFixture(),
            top_upstreams_responses: [{ 'tls://dns.cloudflare.com:853': 5000 }],
        });
        renderPage(() => <TopUpstreamsPage />);
        await waitFor(() =>
            expect(screen.getByText('tls://dns.cloudflare.com:853')).toBeInTheDocument(),
        );
        expect(screen.getByText('Upstream')).toBeInTheDocument();
        expect(screen.getByText('(50.0%)')).toBeInTheDocument();
    });

    it('Average upstream response time: ms values, integer, no percentages', async () => {
        mocks.stats.mockResolvedValue({
            ...baseFixture(),
            top_upstreams_avg_time: [{ 'tls://dns.cloudflare.com:853': 0.152 }],
            avg_processing_time: 0.2,
        });
        renderPage(() => <UpstreamAvgTimePage />);
        await waitFor(() =>
            expect(screen.getByText('tls://dns.cloudflare.com:853')).toBeInTheDocument(),
        );
        expect(screen.getByText('152 ms')).toBeInTheDocument();
        expect(screen.queryByText(/\(.*%/)).toBeNull();
    });
});
