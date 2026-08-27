import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen, waitFor } from '@solidjs/testing-library';

// jsdom has no matchMedia; App's theme effect and some hooks depend on it.
beforeAll(() => {
    if (!window.matchMedia) {
        window.matchMedia = (query: string) =>
            ({
                matches: false,
                media: query,
                onchange: null,
                addEventListener: () => {},
                removeEventListener: () => {},
                addListener: () => {},
                removeListener: () => {},
                dispatchEvent: () => false,
            }) as unknown as MediaQueryList;
    }
});

// Keep the test focused on ROUTING: stub the store (no network/effects) and
// the always-on chrome so the assertion only depends on route registration.
vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        processing: false,
        isCoreRunning: true,
        language: 'en',
        theme: undefined,
        protectionEnabled: false,
    },
    getDnsStatus: vi.fn(),
    getTimerStatus: vi.fn(),
}));

vi.mock('panel/common/ui/Header', () => ({
    Header: () => <div data-testid="chrome-header" />,
}));
vi.mock('panel/common/ui/Banners', () => ({
    Banners: () => <div data-testid="chrome-banners" />,
}));
vi.mock('panel/common/ui/Sidebar', () => ({
    Sidebar: () => <div data-testid="chrome-sidebar" />,
}));
vi.mock('panel/common/ui/Footer', () => ({
    Footer: () => <div data-testid="chrome-footer" />,
}));
vi.mock('panel/common/ui/Icons', () => ({ Icons: (): null => null }));
vi.mock('panel/components/Toasts', () => ({ Toasts: (): null => null }));

// Deterministic marker so the assertion does not depend on Dashboard data/i18n.
vi.mock('panel/components/Dashboard', () => ({
    Dashboard: () => <div data-testid="route-dashboard" />,
}));
vi.mock('panel/components/Stats', () => ({
    TopClientsPage: () => <div data-testid="route-top-clients" />,
    TopQueriedDomainsPage: () => <div data-testid="route-top-queried-domains" />,
    TopBlockedDomainsPage: () => <div data-testid="route-top-blocked-domains" />,
    TopUpstreamsPage: () => <div data-testid="route-top-upstreams" />,
    UpstreamAvgTimePage: () => <div data-testid="route-upstream-avg-time" />,
}));

import App from '../components/App';

describe('App routing', () => {
    it('registers the /dashboard route and renders matched route content', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);

        expect(await screen.findByTestId('route-dashboard')).toBeInTheDocument();
    });

    it.each([
        ['#/top_clients', 'route-top-clients'],
        ['#/top_queried_domains', 'route-top-queried-domains'],
        ['#/top_blocked_domains', 'route-top-blocked-domains'],
        ['#/top_upstreams', 'route-top-upstreams'],
        ['#/upstream_avg_time', 'route-upstream-avg-time'],
    ])('renders the stats page for %s', async (hash, testid) => {
        window.location.hash = hash;
        render(() => <App />);
        await waitFor(() => expect(screen.getByTestId(testid)).toBeInTheDocument());
    });

    it('mounts Banners between Header and the wrapper in the main entry', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);

        const banners = await screen.findByTestId('chrome-banners');
        expect(banners).toBeInTheDocument();

        // Verify ordering: Header → Banners → Sidebar (wrapper starts)
        const header = screen.getByTestId('chrome-header');
        const sidebar = screen.getByTestId('chrome-sidebar');

        // Banners should be after Header in DOM
        expect(
            header.compareDocumentPosition(banners) & Node.DOCUMENT_POSITION_FOLLOWING,
        ).toBeTruthy();

        // Banners should be before Sidebar in DOM
        expect(
            banners.compareDocumentPosition(sidebar) & Node.DOCUMENT_POSITION_FOLLOWING,
        ).toBeTruthy();
    });
});
