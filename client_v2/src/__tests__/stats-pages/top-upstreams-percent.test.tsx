import { render, screen } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import { TopUpstreamsPage } from 'panel/components/Stats/TopUpstreamsPage';
import { byViewportWidth, mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

const mocks = vi.hoisted(() => ({ getStats: vi.fn(), getStatsConfig: vi.fn() }));

vi.mock('panel/stores/stats', async () => {
    const { createMutable } = await import('solid-js/store');

    const statsState = createMutable({
        configLoaded: true,
        statsInitialized: true,
        interval: 24 * 60 * 60 * 1000,
        topUpstreamsResponses: [
            { name: 'https://cloudflare-dns.com/dns-query', count: 600 },
            { name: 'https://dns.google/dns-query', count: 500 },
            { name: 'tls://dns.quad9.net', count: 200 },
        ],
        // Much larger than the upstream total on purpose: the page must not use
        // it as the denominator.
        numDnsQueries: 100_000,
    });

    return { statsState, getStats: mocks.getStats, getStatsConfig: mocks.getStatsConfig };
});

const renderPage = () =>
    render(() => (
        <HashRouter>
            <Route path="/" component={() => <TopUpstreamsPage />} />
        </HashRouter>
    ));

describe('TopUpstreamsPage percentages', () => {
    beforeEach(() => {
        localStorage.clear();
        window.location.hash = '#/';
        vi.clearAllMocks();
        mockMatchMedia(byViewportWidth(1300));
    });

    it('takes the table shares from the upstream responses', () => {
        renderPage();

        // Pin the desktop branch: `stats-percent` exists in the mobile cards
        // too, so without this the test could turn into a copy of the mobile
        // one if the media semantics changed.
        expect(screen.queryByTestId('stats-mobile-card')).toBeNull();

        const shares = screen.getAllByTestId('stats-percent').map((el) => el.textContent);

        expect(shares).toHaveLength(3);
        expect(shares).toEqual(expect.arrayContaining(['(46%)', '(38%)', '(15%)']));
    });

    it('uses the same total for the mobile progress bar', () => {
        mockMatchMedia(byViewportWidth(375));
        renderPage();

        const widths = screen
            .getAllByTestId('stats-mobile-progress-fill')
            .map((fill) => Number.parseFloat(fill.style.width))
            .sort((a, b) => b - a);

        expect(widths).toHaveLength(3);

        expect(widths[0]).toBeCloseTo(46.1538, 3);
        expect(widths[1]).toBeCloseTo(38.4615, 3);
        expect(widths[2]).toBeCloseTo(15.3846, 3);
    });
});
