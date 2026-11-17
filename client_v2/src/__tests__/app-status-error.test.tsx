import { describe, it, expect, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@solidjs/testing-library';

import { copy, copyInDom } from 'panel/__tests__/helpers/copy';

// The bootstrap status check failed, so the store still holds `initialState`
// defaults and App must show the retry surface instead of the route body.
const mocks = vi.hoisted(() => ({
    getDnsStatus: vi.fn(),
    getTimerStatus: vi.fn(),
    stopStatusChecks: vi.fn(),
}));

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        processing: false,
        statusFailed: true,
        isCoreRunning: true,
        language: 'en',
        theme: undefined,
        protectionEnabled: false,
    },
    getDnsStatus: mocks.getDnsStatus,
    getTimerStatus: mocks.getTimerStatus,
    stopStatusChecks: mocks.stopStatusChecks,
}));

vi.mock('panel/common/ui/Header', () => ({ Header: (): null => null }));
vi.mock('panel/common/ui/Banners', () => ({ Banners: (): null => null }));
vi.mock('panel/common/ui/Sidebar', () => ({ Sidebar: (): null => null }));
vi.mock('panel/common/ui/Footer', () => ({ Footer: (): null => null }));
vi.mock('panel/common/ui/Icons', () => ({ Icons: (): null => null }));
vi.mock('panel/components/Toasts', () => ({ Toasts: (): null => null }));

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

describe('App status error surface', () => {
    it('hides the route body and explains the failure', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);

        expect(await screen.findByTestId('status-error')).toBeInTheDocument();
        expect(screen.queryByTestId('route-dashboard')).not.toBeInTheDocument();
        expect(screen.getByText(copyInDom('dns_status_error'))).toBeInTheDocument();
    });

    it('retries the bootstrap status check from the button', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);

        const button = await screen.findByRole('button', { name: copy('try_again') });
        mocks.getDnsStatus.mockClear();

        fireEvent.click(button);

        expect(mocks.getDnsStatus).toHaveBeenCalledTimes(1);
    });

    it('retries the full status check when the tab becomes visible again', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);
        mocks.getDnsStatus.mockClear();

        // Only the full check can fill the store after a failed bootstrap.
        document.dispatchEvent(new Event('visibilitychange'));

        expect(mocks.getDnsStatus).toHaveBeenCalledTimes(1);
        expect(mocks.getTimerStatus).not.toHaveBeenCalled();
    });

    it('stops the pending status checks on unmount', async () => {
        window.location.hash = '#/dashboard';

        render(() => <App />);

        cleanup();

        expect(mocks.stopStatusChecks).toHaveBeenCalled();
    });
});
