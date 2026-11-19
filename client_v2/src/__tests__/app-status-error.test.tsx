import { describe, it, expect, vi, beforeEach } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@solidjs/testing-library';

import { copy, copyInDom } from 'panel/__tests__/helpers/copy';

const mocks = vi.hoisted(() => ({
    state: {
        processing: false,
        statusFailed: false,
        isCoreRunning: true,
        language: 'en',
        theme: undefined as string | undefined,
        protectionEnabled: false,
    },
    getDnsStatus: vi.fn(),
    getTimerStatus: vi.fn(),
    stopStatusChecks: vi.fn(),
}));

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: mocks.state,
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

const renderApp = () => {
    window.location.hash = '#/dashboard';
    render(() => <App />);
};

describe('App status screens', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.state.processing = false;
        mocks.state.statusFailed = false;
        mocks.state.isCoreRunning = true;
    });

    it('explains a failed bootstrap and hides the route body', async () => {
        mocks.state.statusFailed = true;

        renderApp();

        expect(await screen.findByTestId('status-error')).toBeInTheDocument();
        expect(screen.queryByTestId('route-dashboard')).not.toBeInTheDocument();
        expect(screen.getByText(copyInDom('dns_status_error_short'))).toBeInTheDocument();
    });

    it('explains a stopped core and hides the route body', async () => {
        mocks.state.isCoreRunning = false;

        renderApp();

        expect(await screen.findByTestId('status-error')).toBeInTheDocument();
        expect(screen.queryByTestId('route-dashboard')).not.toBeInTheDocument();
        expect(screen.getByText(copyInDom('dns_not_running'))).toBeInTheDocument();
    });

    it('retries the full status check from the button', async () => {
        mocks.state.statusFailed = true;

        renderApp();

        const button = await screen.findByRole('button', { name: copy('try_again') });
        mocks.getDnsStatus.mockClear();

        fireEvent.click(button);

        expect(mocks.getDnsStatus).toHaveBeenCalledTimes(1);
    });

    it('runs the full status check when the tab becomes visible after a failure', async () => {
        mocks.state.statusFailed = true;

        renderApp();
        mocks.getDnsStatus.mockClear();

        document.dispatchEvent(new Event('visibilitychange'));

        expect(mocks.getDnsStatus).toHaveBeenCalledTimes(1);
        expect(mocks.getTimerStatus).not.toHaveBeenCalled();
    });

    it('runs the full status check when the tab becomes visible after the core stopped', async () => {
        mocks.state.isCoreRunning = false;

        renderApp();
        mocks.getDnsStatus.mockClear();

        document.dispatchEvent(new Event('visibilitychange'));

        expect(mocks.getDnsStatus).toHaveBeenCalledTimes(1);
        expect(mocks.getTimerStatus).not.toHaveBeenCalled();
    });

    it('runs the timer check when the tab becomes visible normally', async () => {
        renderApp();
        mocks.getDnsStatus.mockClear();

        document.dispatchEvent(new Event('visibilitychange'));

        expect(mocks.getTimerStatus).toHaveBeenCalledTimes(1);
        expect(mocks.getDnsStatus).not.toHaveBeenCalled();
    });

    it('stops the pending status checks on unmount', () => {
        renderApp();

        cleanup();

        expect(mocks.stopStatusChecks).toHaveBeenCalled();
    });
});
