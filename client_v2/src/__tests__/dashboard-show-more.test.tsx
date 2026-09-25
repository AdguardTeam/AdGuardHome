import { type JSX } from 'solid-js';
import { render, screen, within } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, beforeEach } from 'vitest';

import { TopClients } from 'panel/components/Dashboard/blocks/TopClients';
import { TopQueriedDomains } from 'panel/components/Dashboard/blocks/TopQueriedDomains';
import { TopBlockedDomains } from 'panel/components/Dashboard/blocks/TopBlockedDomains';
import { TopUpstreams } from 'panel/components/Dashboard/blocks/TopUpstreams';
import { UpstreamAvgTime } from 'panel/components/Dashboard/blocks/UpstreamAvgTime';
import { DAY } from 'panel/helpers/constants';
import { formatCompactNumber } from 'panel/helpers/helpers';
import { copyInDom } from 'panel/__tests__/helpers/copy';
import { mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

const renderWithRouter = (ui: () => JSX.Element) =>
    render(() => (
        <HashRouter>
            <Route path="/" component={ui} />
        </HashRouter>
    ));

const getLinkHref = (testid: string) => screen.getByTestId(testid).getAttribute('href') ?? '';

/**
 * The row that shows `domain`.  The row's tooltip repeats the domain name, so
 * the lookup has to stay scoped to the row itself.
 */
const getDomainRow = (domain: string) => {
    const row = screen
        .getAllByTestId('top-domain-row')
        .find((candidate) => within(candidate).queryByText(domain) !== null);

    expect(row).toBeDefined();

    return row as HTMLElement;
};

/** Rows a card shows before the "Show more" footer would reveal anything. */
const DOMAINS_VISIBLE_ITEMS = 5;
const CLIENTS_VISIBLE_ITEMS = 4;

/** Filler rows, used to push a list past the card's visible limit. */
const fillers = (count: number) =>
    Array.from({ length: count }, (_, index) => ({ name: `filler-${index}`, count: 1 }));

describe('Dashboard "Show more" links', () => {
    beforeEach(() => {
        // Drives `useIsDesktop` (768px breakpoint) for the responsive assertions.
        mockMatchMedia(false);
    });

    it('Top clients card links to /top_clients', () => {
        renderWithRouter(() => (
            <TopClients
                topClients={[{ name: '10.0.0.1', count: 5 }, ...fillers(CLIENTS_VISIBLE_ITEMS)]}
                numDnsQueries={100}
            />
        ));
        expect(screen.getByText(copyInDom('show_more'))).toBeInTheDocument();
        expect(getLinkHref('show-more-top-clients')).toContain('/top_clients');
    });

    it('Top queried domains card links to /top_queried_domains', () => {
        renderWithRouter(() => (
            <TopQueriedDomains
                topQueriedDomains={[{ name: 'a.org', count: 5 }, ...fillers(DOMAINS_VISIBLE_ITEMS)]}
                numDnsQueries={100}
            />
        ));
        expect(getLinkHref('show-more-top-queried-domains')).toContain('/top_queried_domains');

        // The domain name is a QueryLog link filtered by the domain.
        const domainLink = getDomainRow('a.org');
        expect(domainLink.tagName).toBe('A');
        expect(domainLink.getAttribute('href')).toContain('/logs');
        expect(domainLink.getAttribute('href')).toContain('a.org');
    });

    it('Top blocked domains card links to /top_blocked_domains', () => {
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={[{ name: 'a.org', count: 5 }, ...fillers(DOMAINS_VISIBLE_ITEMS)]}
                numBlockedFiltering={100}
            />
        ));
        expect(getLinkHref('show-more-top-blocked-domains')).toContain('/top_blocked_domains');

        const domainLink = getDomainRow('a.org');
        expect(domainLink.tagName).toBe('A');
        expect(domainLink.getAttribute('href')).toContain('/logs');
        expect(domainLink.getAttribute('href')).toContain('a.org');

        // The blocked total links to QueryLog filtered by blocked status.
        expect(getLinkHref('blocked-total-link')).toContain('/logs');
        expect(getLinkHref('blocked-total-link')).toContain('status=blocked');
    });

    it('Top blocked domains shows only the number of blocked queries on mobile', () => {
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={[{ name: 'a.org', count: 5 }]}
                numBlockedFiltering={100}
            />
        ));
        expect(screen.getByTestId('blocked-total-link').textContent).toBe(formatCompactNumber(100));
    });

    it('Top blocked domains shows the blocked total copy on desktop', () => {
        mockMatchMedia(true);
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={[{ name: 'a.org', count: 5 }]}
                numBlockedFiltering={100}
            />
        ));
        expect(screen.getByTestId('blocked-total-link').textContent).toBe(
            copyInDom('blocked_total', { value: formatCompactNumber(100) }),
        );
    });

    it('Top upstreams card links to /top_upstreams', () => {
        renderWithRouter(() => (
            <TopUpstreams
                topUpstreamsResponses={[
                    { name: 'tls://x:853', count: 5 },
                    ...fillers(DOMAINS_VISIBLE_ITEMS),
                ]}
                numDnsQueries={100}
            />
        ));
        expect(getLinkHref('show-more-top-upstreams')).toContain('/top_upstreams');
    });

    it('Average upstream response time card links to /upstream_avg_time', () => {
        renderWithRouter(() => (
            <UpstreamAvgTime
                topUpstreamsAvgTime={[
                    { name: 'tls://x:853', count: 12 },
                    ...fillers(DOMAINS_VISIBLE_ITEMS),
                ]}
                avgProcessingTime={12}
            />
        ));
        expect(getLinkHref('show-more-upstream-avg-time')).toContain('/upstream_avg_time');
    });

    it('the link is hidden when the card shows every row it has', () => {
        renderWithRouter(() => (
            <TopQueriedDomains
                topQueriedDomains={fillers(DOMAINS_VISIBLE_ITEMS)}
                numDnsQueries={100}
            />
        ));
        expect(screen.queryByTestId('show-more-top-queried-domains')).toBeNull();
    });

    it('the link is hidden when the card is empty', () => {
        renderWithRouter(() => <TopQueriedDomains topQueriedDomains={[]} numDnsQueries={100} />);
        expect(screen.queryByTestId('show-more-top-queried-domains')).toBeNull();
    });

    it('the link is hidden for top blocked domains with no hidden rows', () => {
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={fillers(DOMAINS_VISIBLE_ITEMS)}
                numBlockedFiltering={100}
            />
        ));
        expect(screen.queryByTestId('show-more-top-blocked-domains')).toBeNull();
    });

    it('the link is hidden for top upstreams with no hidden rows', () => {
        renderWithRouter(() => (
            <TopUpstreams
                topUpstreamsResponses={fillers(DOMAINS_VISIBLE_ITEMS)}
                numDnsQueries={100}
            />
        ));
        expect(screen.queryByTestId('show-more-top-upstreams')).toBeNull();
    });

    it('the link is hidden for average upstream response time with no hidden rows', () => {
        renderWithRouter(() => (
            <UpstreamAvgTime
                topUpstreamsAvgTime={fillers(DOMAINS_VISIBLE_ITEMS)}
                avgProcessingTime={12}
            />
        ));
        expect(screen.queryByTestId('show-more-upstream-avg-time')).toBeNull();
    });

    it('Top clients shows the link once a fifth client is hidden', () => {
        const { unmount } = renderWithRouter(() => (
            <TopClients topClients={fillers(CLIENTS_VISIBLE_ITEMS)} numDnsQueries={100} />
        ));
        expect(screen.queryByTestId('show-more-top-clients')).toBeNull();
        unmount();

        renderWithRouter(() => (
            <TopClients topClients={fillers(CLIENTS_VISIBLE_ITEMS + 1)} numDnsQueries={100} />
        ));
        expect(getLinkHref('show-more-top-clients')).toContain('/top_clients');
    });

    it('the link includes the selected stats period', () => {
        renderWithRouter(() => (
            <TopBlockedDomains
                topBlockedDomains={[{ name: 'a.org', count: 5 }, ...fillers(DOMAINS_VISIBLE_ITEMS)]}
                numBlockedFiltering={100}
                period={DAY}
            />
        ));
        expect(getLinkHref('show-more-top-blocked-domains')).toContain(`period=${DAY}`);
    });
});
