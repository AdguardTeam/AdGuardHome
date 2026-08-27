import { type JSX } from 'solid-js';
import { render, screen } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect } from 'vitest';

import { TopClients } from 'panel/components/Dashboard/blocks/TopClients';
import { TopQueriedDomains } from 'panel/components/Dashboard/blocks/TopQueriedDomains';
import { TopBlockedDomains } from 'panel/components/Dashboard/blocks/TopBlockedDomains';
import { TopUpstreams } from 'panel/components/Dashboard/blocks/TopUpstreams';
import { UpstreamAvgTime } from 'panel/components/Dashboard/blocks/UpstreamAvgTime';
import { DAY } from 'panel/helpers/constants';

const renderWithRouter = (ui: () => JSX.Element) =>
    render(() => (
        <HashRouter>
            <Route path="/" component={ui} />
        </HashRouter>
    ));

const getLinkHref = (testid: string) =>
    screen.getByTestId(testid).getAttribute('href') ?? '';

describe('Dashboard "Show more" links', () => {
    it('Top clients card links to /top_clients', () => {
        renderWithRouter(() => (
            <TopClients topClients={[{ name: '10.0.0.1', count: 5 }]} numDnsQueries={100} />
        ));
        expect(screen.getByText('Show more')).toBeInTheDocument();
        expect(getLinkHref('show-more-top-clients')).toContain('/top_clients');
    });

    it('Top queried domains card links to /top_queried_domains', () => {
        renderWithRouter(() => (
            <TopQueriedDomains
                topQueriedDomains={[{ name: 'a.org', count: 5 }]}
                numDnsQueries={100}
            />
        ));
        expect(getLinkHref('show-more-top-queried-domains')).toContain('/top_queried_domains');
    });

    it('Top blocked domains card links to /top_blocked_domains', () => {
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={[{ name: 'a.org', count: 5 }]}
                numBlockedFiltering={100}
            />
        ));
        expect(getLinkHref('show-more-top-blocked-domains')).toContain('/top_blocked_domains');
    });

    it('Top upstreams card links to /top_upstreams', () => {
        renderWithRouter(() => (
            <TopUpstreams
                topUpstreamsResponses={[{ name: 'tls://x:853', count: 5 }]}
                numDnsQueries={100}
            />
        ));
        expect(getLinkHref('show-more-top-upstreams')).toContain('/top_upstreams');
    });

    it('Average upstream response time card links to /upstream_avg_time', () => {
        renderWithRouter(() => (
            <UpstreamAvgTime
                topUpstreamsAvgTime={[{ name: 'tls://x:853', count: 12 }]}
                avgProcessingTime={12}
            />
        ));
        expect(getLinkHref('show-more-upstream-avg-time')).toContain('/upstream_avg_time');
    });

    it('the link is visible even when the card is empty', () => {
        renderWithRouter(() => (
            <TopQueriedDomains topQueriedDomains={[]} numDnsQueries={100} />
        ));
        expect(getLinkHref('show-more-top-queried-domains')).toContain('/top_queried_domains');
    });

    it('the link includes the selected stats period', () => {
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={[{ name: 'a.org', count: 5 }]}
                numBlockedFiltering={100}
                period={DAY}
            />
        ));
        expect(getLinkHref('show-more-top-blocked-domains')).toContain(`period=${DAY}`);
    });
});
