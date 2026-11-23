import { type JSX } from 'solid-js';
import { render, screen } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect } from 'vitest';

import { TopUpstreams } from 'panel/components/Dashboard/blocks/TopUpstreams';

const renderWithRouter = (ui: () => JSX.Element) =>
    render(() => (
        <HashRouter>
            <Route path="/" component={ui} />
        </HashRouter>
    ));

describe('TopUpstreams percentages', () => {
    // Cache hits and blocked or rewritten queries never reach an upstream, so a
    // share taken from all DNS queries leaves the rows short of 100%.
    it('takes the share from the upstream responses, not from all DNS queries', () => {
        renderWithRouter(() => (
            <TopUpstreams
                topUpstreamsResponses={[
                    { name: 'https://cloudflare-dns.com/dns-query', count: 600 },
                    { name: 'https://dns.google/dns-query', count: 500 },
                    { name: 'tls://dns.quad9.net', count: 200 },
                ]}
            />
        ));

        expect(screen.getByText('(46%)')).toBeInTheDocument();
        expect(screen.getByText('(38%)')).toBeInTheDocument();
        expect(screen.getByText('(15%)')).toBeInTheDocument();
    });

    it('shows a lone upstream as 100%', () => {
        renderWithRouter(() => (
            <TopUpstreams topUpstreamsResponses={[{ name: 'tls://x:853', count: 7 }]} />
        ));

        expect(screen.getByText('(100%)')).toBeInTheDocument();
    });
});
