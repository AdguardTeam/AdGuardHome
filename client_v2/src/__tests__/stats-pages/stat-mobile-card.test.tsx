import { type JSX } from 'solid-js';
import { render, screen } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect } from 'vitest';

import { StatMobileCard } from 'panel/components/Stats/blocks/StatMobileCard';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const renderWithRouter = (ui: () => JSX.Element) =>
    render(() => (
        <HashRouter>
            <Route path="/" component={ui} />
        </HashRouter>
    ));

const items = [{ label: 'Queries', value: '5' }];

describe('StatMobileCard', () => {
    it('renders the whole card as a query log link when cardLink is set', () => {
        renderWithRouter(() => (
            <StatMobileCard
                title="a.org"
                cardLink={{ to: 'QueryLog', query: { search: '"a.org"' } }}
                items={items}
            />
        ));

        const card = screen.getByTestId('stats-mobile-card');
        expect(card.tagName).toBe('A');
        expect(card.getAttribute('href')).toContain('/logs');
        expect(card.getAttribute('href')).toContain('a.org');
        expect(card.querySelectorAll('a').length).toBe(0);
    });

    it('keeps the card a plain container without cardLink', () => {
        renderWithRouter(() => <StatMobileCard title="a.org" items={items} />);

        const card = screen.getByTestId('stats-mobile-card');
        expect(card.tagName).toBe('DIV');
        expect(card.getAttribute('href')).toBeNull();
    });

    it('does not follow the card link when an action is clicked', () => {
        renderWithRouter(() => (
            <StatMobileCard
                title="a.org"
                cardLink={{ to: 'QueryLog', query: { search: '"a.org"' } }}
                items={items}
                actions={<button type="button">{copyInDom('block_client')}</button>}
            />
        ));

        const click = new MouseEvent('click', { bubbles: true, cancelable: true });
        screen.getByRole('button', { name: copyInDom('block_client') }).dispatchEvent(click);

        expect(click.defaultPrevented).toBe(true);
    });
});
