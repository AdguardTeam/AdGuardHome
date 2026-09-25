import { render, screen, fireEvent, waitFor, within } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, beforeEach } from 'vitest';

import { GeneralStatistics } from 'panel/components/Dashboard/blocks/GeneralStatistics';
import { QUERY_LOG_REASON_FILTER, QUERY_LOG_STATUS_FILTER } from 'panel/helpers/constants';
import { formatNumber } from 'panel/helpers/helpers';
import { copy } from 'panel/__tests__/helpers/copy';
import { mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

/**
 * Counts are deliberately above `QueriesTooltip`'s threshold (1000) and the
 * blocked counts give round percentages, so the value cell is easy to assert on.
 */
const STATS = {
    numDnsQueries: 2000,
    numBlockedFiltering: 500,
    numReplacedSafebrowsing: 100,
    numReplacedParental: 40,
    numReplacedSafesearch: 8,
    avgProcessingTime: 152,
};

/** Every row, in the order `GeneralStatistics` renders them. */
const ROWS = [
    { label: copy('dns_queries'), description: copy('dns_queries_tooltip') },
    { label: copy('ads_blocked'), description: copy('ads_blocked_tooltip') },
    { label: copy('threats_blocked'), description: copy('threats_blocked_tooltip') },
    { label: copy('adult_websites_blocked'), description: copy('adult_websites_blocked_tooltip') },
    { label: copy('safe_search_used'), description: copy('safe_search_used_tooltip') },
    {
        label: copy('average_time_processing'),
        description: copy('average_time_processing_tooltip'),
    },
];

const renderCard = () =>
    render(() => (
        <HashRouter>
            <Route path="/" component={() => <GeneralStatistics {...STATS} />} />
        </HashRouter>
    ));

/** The tooltip trigger of the row whose label is `label`. */
const rowTrigger = (label: string) => {
    const trigger = screen.getByText(label).closest('[data-part="trigger"]');
    expect(trigger).not.toBeNull();

    return trigger as HTMLElement;
};

/** The tooltip content of the row whose description is `description`. */
const tooltipContent = (description: string) => {
    const content = screen.getByText(description).closest('[data-part="content"]');
    expect(content).not.toBeNull();

    return content as HTMLElement;
};

const hover = async (target: HTMLElement, description: string) => {
    const content = tooltipContent(description);

    expect(content.hasAttribute('hidden')).toBe(true);

    fireEvent.pointerOver(target, { pointerType: 'mouse' });

    // SHOW_DELAY is 200 ms; real timers, so use a generous timeout.
    await waitFor(() => expect(content.hasAttribute('hidden')).toBe(false), { timeout: 1000 });
};

describe('Dashboard general statistics row tooltips', () => {
    beforeEach(() => {
        // The global setup mock answers `false` to every query, i.e. a desktop
        // without hover media support, matching the Tooltip tests.
        mockMatchMedia(false);
    });

    it('makes the whole row the tooltip trigger, not just the label', () => {
        renderCard();

        // The value cell (and its bar) are inside the trigger, so hovering the
        // number or the bar reveals the same description as hovering the label.
        const dnsRow = rowTrigger(copy('dns_queries'));
        expect(within(dnsRow).getByText(`(${copy('total')})`)).toBeInTheDocument();
        expect(dnsRow.querySelector('[class*="queryBar"]')).not.toBeNull();

        const blockedRow = rowTrigger(copy('ads_blocked'));
        expect(within(blockedRow).getByText('(25.0%)')).toBeInTheDocument();
    });

    it('opens the description when hovering the value area of a row', async () => {
        renderCard();

        const value = within(rowTrigger(copy('dns_queries'))).getByText(`(${copy('total')})`);

        await hover(value, copy('dns_queries_tooltip'));
    });

    it.each(ROWS)('opens the description of the $label row on hover', async (row) => {
        renderCard();

        await hover(rowTrigger(row.label), row.description);
    });

    it('renders a single tooltip per row', () => {
        const { container } = renderCard();

        expect(container.querySelectorAll('[data-part="trigger"]')).toHaveLength(ROWS.length);

        // The nested `QueriesTooltip` is gone: the exact count is no longer
        // revealed, because it would open on top of the row description.
        expect(
            screen.queryByText(copy('queries_tooltip', { value: formatNumber(2000) })),
        ).toBeNull();
    });

    it('keeps the row link to the query log', () => {
        renderCard();

        const dnsRow = screen.getByText(copy('dns_queries')).closest('a');
        expect(dnsRow?.getAttribute('href')).toContain('/logs');

        const blockedRow = screen.getByText(copy('ads_blocked')).closest('a');
        const blockedHref = blockedRow?.getAttribute('href') ?? '';
        expect(blockedHref).toContain(QUERY_LOG_STATUS_FILTER.BLOCKED.QUERY);
        expect(blockedHref).toContain(QUERY_LOG_REASON_FILTER.BLOCKED_BY_FILTER.QUERY);
    });

    it('keeps the average processing time row a plain row with its value', () => {
        renderCard();

        const label = screen.getByText(copy('average_time_processing'));
        expect(label.closest('a')).toBeNull();
        expect(
            within(rowTrigger(copy('average_time_processing'))).getByText(
                copy('processing_time_ms', { value: '152' }),
            ),
        ).toBeInTheDocument();
    });

    describe('touch device', () => {
        beforeEach(() => {
            mockMatchMedia((query) => query === '(hover: none)');
        });

        it('renders no tooltips at all', () => {
            const { container } = renderCard();

            expect(container.querySelector('[data-scope="tooltip"]')).toBeNull();
            expect(container.querySelectorAll('[data-part="trigger"]')).toHaveLength(0);
        });

        it('still renders the rows as query log links', () => {
            renderCard();

            const dnsRow = screen.getByText(copy('dns_queries')).closest('a');
            expect(dnsRow?.getAttribute('href')).toContain('/logs');
        });
    });
});
