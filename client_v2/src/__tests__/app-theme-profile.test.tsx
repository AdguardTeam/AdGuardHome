import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@solidjs/testing-library';

import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { getTheme } from 'panel/helpers/helpers';

// The real dashboard store is used here: these tests drive the profile request
// through the network layer and assert how its theme competes with the theme
// the guest has stored in the browser.
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
import { dashboardState } from 'panel/stores/dashboard';

type JsonBody = Record<string, unknown>;

const jsonResponse = (body: JsonBody) => new Response(JSON.stringify(body), { status: 200 });

const runningStatus: JsonBody = {
    running: true,
    version: 'v1.0.0',
    dns_port: 53,
    dns_addresses: [],
    protection_enabled: true,
    http_port: 80,
};

/** Resolves once the profile response has been handed to the store. */
let profileServed: Promise<void>;

/**
 * Serves the endpoints the dashboard store hits during startup and records
 * when the profile request has been answered.
 */
const mockStartupRequests = (profile: JsonBody) => {
    profileServed = new Promise((resolve) => {
        vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
            const url = String(input);

            if (url.includes('control/version.json')) {
                return jsonResponse({ disabled: true, new_version: 'x' });
            }
            if (url.includes('control/tls/status')) {
                return jsonResponse({ enabled: false });
            }
            if (url.includes('control/profile')) {
                resolve();
                return jsonResponse(profile);
            }

            return jsonResponse(runningStatus);
        });
    });
};

/**
 * Renders the app and waits until the profile has been applied, so that the
 * assertions see the final theme rather than the pre-profile one.
 */
const renderApp = async (profile: JsonBody) => {
    mockStartupRequests(profile);

    window.location.hash = '#/dashboard';
    render(() => <App />);

    await profileServed;
    // Let the store update and the App effects run.
    await new Promise((resolve) => setTimeout(resolve, 0));
};

const appliedTheme = () => document.documentElement.dataset.theme;

describe('App theme effect and the profile', () => {
    beforeEach(() => {
        localStorage.clear();
        document.documentElement.removeAttribute('data-theme');
        document.documentElement.style.colorScheme = '';
    });

    afterEach(() => vi.restoreAllMocks());

    it('keeps the stored guest theme when no user is signed in', async () => {
        localStorage.setItem(LOCAL_STORAGE_KEYS.THEME, JSON.stringify('dark'));

        // A userless install (auth disabled or GL-iNet) reports the config
        // default here, not a theme the guest has chosen.
        await renderApp({ name: '', theme: 'auto' });

        await waitFor(() => expect(appliedTheme()).toBe('dark'));

        expect(dashboardState.name).toBe('');
        expect(dashboardState.theme).toBeUndefined();
        expect(getTheme()).toBe('dark');
    });

    it('applies the profile theme for a signed-in operator', async () => {
        localStorage.setItem(LOCAL_STORAGE_KEYS.THEME, JSON.stringify('dark'));

        await renderApp({ name: 'admin', theme: 'light' });

        await waitFor(() => expect(appliedTheme()).toBe('light'));

        expect(dashboardState.theme).toBe('light');
        expect(document.documentElement.style.colorScheme).toBe('light');
        // The server value is mirrored into localStorage so that the inlined
        // pre-paint script applies it on the next load instead of the stale
        // theme.
        expect(getTheme()).toBe('light');
    });
});
