import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@solidjs/testing-library';

import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { mockMatchMedia } from './helpers/matchMedia';

// Keep the store at "auto" so App's theme effect registers the OS-scheme
// listener this test drives; everything else in the chrome is irrelevant.
vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        processing: true,
        isCoreRunning: true,
        language: '',
        theme: 'auto',
        protectionEnabled: false,
    },
    getDnsStatus: vi.fn(),
    getTimerStatus: vi.fn(),
}));

vi.mock('panel/common/ui/Header', () => ({ Header: (): null => null }));
vi.mock('panel/common/ui/Banners', () => ({ Banners: (): null => null }));
vi.mock('panel/common/ui/Sidebar', () => ({ Sidebar: (): null => null }));
vi.mock('panel/common/ui/Footer', () => ({ Footer: (): null => null }));
vi.mock('panel/common/ui/Icons', () => ({ Icons: (): null => null }));
vi.mock('panel/components/Toasts', () => ({ Toasts: (): null => null }));
vi.mock('panel/components/Dashboard', () => ({ Dashboard: (): null => null }));
vi.mock('panel/components/Stats', () => ({
    TopClientsPage: (): null => null,
    TopQueriedDomainsPage: (): null => null,
    TopBlockedDomainsPage: (): null => null,
    TopUpstreamsPage: (): null => null,
    UpstreamAvgTimePage: (): null => null,
}));

import App from 'panel/components/App';

describe('App theme effect', () => {
    afterEach(() => localStorage.clear());

    it('keeps "auto" in storage when the OS scheme changes', () => {
        const media = mockMatchMedia(false);
        window.location.hash = '#/dashboard';

        render(() => <App />);

        expect(document.documentElement.dataset.theme).toBe('light');

        media.set(true);

        // The applied theme follows the OS, but the stored preference must stay
        // "auto", so the next reload resolves it again instead of freezing it.
        expect(document.documentElement.dataset.theme).toBe('dark');
        expect(localStorage.getItem(LOCAL_STORAGE_KEYS.THEME)).toBe(JSON.stringify('auto'));
    });
});
