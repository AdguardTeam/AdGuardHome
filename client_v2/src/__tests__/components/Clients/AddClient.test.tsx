import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';

import { AddClient } from 'panel/components/Clients/AddClient/AddClient';
import { clearClientForm, initClientForm, updateClientFormField } from 'panel/stores/clientForm';

// jsdom has no matchMedia; the tags Select and some hooks depend on it.
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

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        clients: [],
        supportedTags: [],
    },
    getClients: vi.fn(),
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const renderAddClient = () => {
    const history = createMemoryHistory();
    history.set({ value: '/' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route path="/" component={() => <AddClient />} />
        </MemoryRouter>
    ));
};

describe('AddClient — global settings independence (AGH-234)', () => {
    beforeEach(() => {
        clearClientForm();
    });

    it('keeps Blocked services, Upstream, and DNS cache active when use_global_settings is on', () => {
        initClientForm(null);
        updateClientFormField('use_global_settings', true);
        renderAddClient();

        // AC 1: «Use global settings» blocks only Protection — the Protection
        // link is disabled (rendered as a non-anchor), everything else stays
        // interactive.
        expect(screen.queryByRole('link', { name: /Protection/ })).toBeNull();
        expect(screen.getByRole('link', { name: /Blocked services/ })).toBeInTheDocument();

        const upstreams = document.getElementById('client-upstreams') as HTMLTextAreaElement;
        expect(upstreams).not.toBeDisabled();

        const dnsCache = document.getElementById('use-dns-cache') as HTMLInputElement;
        expect(dnsCache).not.toBeDisabled();
    });

    it('disables the Protection link only while use_global_settings is on', () => {
        initClientForm(null);
        renderAddClient();

        expect(screen.getByRole('link', { name: /Protection/ })).toBeInTheDocument();

        updateClientFormField('use_global_settings', true);

        expect(screen.queryByRole('link', { name: /Protection/ })).toBeNull();
        // The row itself is still rendered (just not as a link).
        expect(screen.getByText(/Enable or disable protection features/)).toBeInTheDocument();
    });
});
