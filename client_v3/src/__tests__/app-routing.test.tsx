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
});
