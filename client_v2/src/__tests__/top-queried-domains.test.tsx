import { describe, it, expect } from 'vitest';
import { render } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';

import { TopQueriedDomains } from 'panel/components/Dashboard/blocks/TopQueriedDomains';
import { setMatchMedia } from './helpers/matchMedia';

describe('TopQueriedDomains', () => {
    const renderTopQueriedDomains = (
        topQueriedDomains: { name: string; count: number }[],
        numDnsQueries: number,
    ) =>
        render(() => (
            <HashRouter>
                <Route
                    path="/"
                    component={() => (
                        <TopQueriedDomains
                            topQueriedDomains={topQueriedDomains}
                            numDnsQueries={numDnsQueries}
                        />
                    )}
                />
            </HashRouter>
        ));

    it('displays only 5 domains on desktop', () => {
        setMatchMedia(true);
        const domains = Array.from({ length: 8 }, (_, i) => ({
            name: `example${i}.org`,
            count: 100 - i,
        }));
        const { container } = renderTopQueriedDomains(domains, 800);

        expect(container.querySelectorAll('[data-testid="top-domain-row"]')).toHaveLength(5);
    });

    it('displays only 5 domains on mobile', () => {
        setMatchMedia(false);
        const domains = Array.from({ length: 8 }, (_, i) => ({
            name: `example${i}.org`,
            count: 100 - i,
        }));
        const { container } = renderTopQueriedDomains(domains, 800);

        expect(container.querySelectorAll('[data-testid="top-domain-row"]')).toHaveLength(5);
    });
});
