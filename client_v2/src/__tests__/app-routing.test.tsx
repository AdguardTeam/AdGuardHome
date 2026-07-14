import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

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
vi.mock('panel/components/Toasts', () => ({ default: (): null => null }));

// Deterministic marker so the assertion does not depend on Dashboard data/i18n.
vi.mock('panel/components/Dashboard', () => ({
    Dashboard: () => <div data-testid="route-dashboard" />,
}));

import App from '../components/App';

describe('App routing', () => {
    it('registers the /dashboard route and renders matched route content', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);

        expect(await screen.findByTestId('route-dashboard')).toBeInTheDocument();
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
