import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@solidjs/testing-library';

import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { getTheme, setUITheme } from 'panel/helpers/helpers';
import { mockMatchMedia } from './helpers/matchMedia';

// The store stays userless (no `name`), so App's theme effect follows the
// guest branch and reads the requested theme from `localStorage`; the
// OS-scheme listener is registered regardless of the stored preference.
// Everything else in the chrome is irrelevant here.
vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        processing: true,
        isCoreRunning: true,
        language: '',
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

    it('keeps an explicit guest choice when the OS scheme changes', () => {
        const media = mockMatchMedia(false);
        window.location.hash = '#/dashboard';

        render(() => <App />);

        // What `Footer` does for a userless install: apply and persist the
        // choice without touching the dashboard store.
        setUITheme('dark');
        expect(document.documentElement.dataset.theme).toBe('dark');

        media.set(true);

        // The guest picked Dark explicitly, so the OS change must not swap the
        // theme nor rewrite the stored preference to "auto".
        expect(document.documentElement.dataset.theme).toBe('dark');
        expect(getTheme()).toBe('dark');
    });

    it('follows the OS scheme once the guest selects System', () => {
        localStorage.setItem(LOCAL_STORAGE_KEYS.THEME, JSON.stringify('dark'));

        const media = mockMatchMedia(false);
        window.location.hash = '#/dashboard';

        render(() => <App />);

        expect(document.documentElement.dataset.theme).toBe('dark');

        // The guest switches to System after the app has mounted; the listener
        // registered at startup must already be active.
        setUITheme('auto');
        expect(document.documentElement.dataset.theme).toBe('light');

        media.set(true);

        expect(document.documentElement.dataset.theme).toBe('dark');
        expect(getTheme()).toBe('auto');
    });
});
