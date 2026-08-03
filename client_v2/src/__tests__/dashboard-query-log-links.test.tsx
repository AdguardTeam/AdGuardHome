import { HashRouter, Route } from '@solidjs/router';
import { render, screen } from '@solidjs/testing-library';
import type { JSX } from 'solid-js';
import { describe, expect, it } from 'vitest';

import { TopClients } from 'panel/components/Dashboard/blocks/TopClients';
import { TopQueriedDomains } from 'panel/components/Dashboard/blocks/TopQueriedDomains';

const renderAtDashboard = (component: () => JSX.Element) => {
    window.location.hash = '#/';

    return render(() => (
        <HashRouter>
            <Route path="/" component={component} />
        </HashRouter>
    ));
};

describe('Dashboard query log links', () => {
    it('links a top queried domain to an exact query log search', () => {
        const domain = 'matching.example.org';

        renderAtDashboard(() => (
            <TopQueriedDomains
                topQueriedDomains={[{ name: domain, count: 7 }]}
                numDnsQueries={10}
            />
        ));

        expect(screen.getByRole('link', { name: domain })).toHaveAttribute(
            'href',
            '#/logs?search=%22matching.example.org%22',
        );
    });

    it('links a top client to an exact query log search', () => {
        const client = '192.0.2.1';

        renderAtDashboard(() => (
            <TopClients topClients={[{ name: client, count: 6 }]} numDnsQueries={10} />
        ));

        expect(screen.getByRole('link', { name: client })).toHaveAttribute(
            'href',
            '#/logs?search=%22192.0.2.1%22',
        );
    });
});
