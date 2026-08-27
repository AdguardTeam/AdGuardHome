import { render, screen, fireEvent } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import { StatsPage } from 'panel/components/Stats/StatsPage';
import type { TableColumn } from 'panel/common/ui/Table';
import { LocalStorageHelper } from 'panel/helpers/localStorageHelper';

type Row = { name: string; count: number };

const rows: Row[] = [
    { name: 'a.org', count: 1 },
    { name: 'b.org', count: 2 },
    { name: 'c.org', count: 3 },
];

const columns: TableColumn<Row>[] = [
    { key: 'name', header: { text: 'Domain' }, accessor: 'name', sortable: true },
    {
        key: 'count',
        header: { text: 'Queries' },
        accessor: 'count',
        sortable: true,
    },
];

const mockMatchMedia = (matches: boolean) => {
    Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: (query: string) =>
            ({
                matches,
                media: query,
                onchange: null,
                addListener: () => {},
                removeListener: () => {},
                addEventListener: () => {},
                removeEventListener: () => {},
                dispatchEvent: () => false,
            }) as MediaQueryList,
    });
};

const renderPage = (overrides: Partial<Parameters<typeof StatsPage<Row>>[0]> = {}) =>
    render(() => (
        <HashRouter>
            <Route
                path="/"
                component={() => (
                    <StatsPage<Row>
                        title="Top queried domains"
                        rows={rows}
                        columns={columns}
                        getRowId={(row) => row.name}
                        defaultSort={{ key: 'count', direction: 'desc' }}
                        loading={false}
                        emptyText="Nothing found"
                        onRefresh={vi.fn()}
                        searchTextForRow={(row) => row.name}
                        pageSizeKey="top_queried_domains_page_size"
                        sortStorageKey="top_queried_domains_sort"
                        renderMobileCard={(row) => (
                            <div data-testid="mobile-card">
                                {row.name}: {row.count}
                            </div>
                        )}
                        {...overrides}
                    />
                )}
            />
        </HashRouter>
    ));

const getRowsText = () =>
    Array.from(document.querySelectorAll('[class*="tableRow"]')).map(
        (el) => el.textContent ?? '',
    );

describe('StatsPage', () => {
    beforeEach(() => {
        mockMatchMedia(true);
        localStorage.clear();
        window.location.hash = '#/';
    });

    it('renders breadcrumb, title, search and refresh controls (desktop)', () => {
        renderPage();
        expect(screen.getByText('Dashboard')).toBeInTheDocument();
        expect(
            screen.getByRole('heading', { name: 'Top queried domains' }),
        ).toBeInTheDocument();
        expect(screen.getByTestId('stats-search-input')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument();
    });

    it('sorts by defaultSort count desc and filters client-side on search', () => {
        renderPage();
        const firstRowText = () =>
            Array.from(document.querySelectorAll('[class*="tableRow"]'))
                .map((el) => el.textContent)
                .join(' ');

        expect(firstRowText()).toContain('c.org');
        expect(firstRowText()).toContain('a.org');

        fireEvent.input(screen.getByTestId('stats-search-input'), {
            target: { value: 'b' },
        });
        expect(firstRowText()).toContain('b.org');
        expect(firstRowText()).not.toContain('a.org');
    });

    it('renders mobile cards and filters them (mobile viewport)', () => {
        mockMatchMedia(false);
        renderPage();
        expect(screen.getAllByTestId('mobile-card').length).toBe(3);

        fireEvent.input(screen.getByTestId('stats-search-input'), {
            target: { value: 'a' },
        });
        expect(screen.getAllByTestId('mobile-card').length).toBe(1);
        expect(screen.getByTestId('mobile-card').textContent).toContain('a.org');
    });

    it('renders children between the header and the table', () => {
        renderPage({ children: <div data-testid="below-header-content">Add client</div> });
        expect(screen.getByTestId('below-header-content')).toBeInTheDocument();
    });

    it('calls onRefresh when refresh is clicked', () => {
        const onRefresh = vi.fn();
        renderPage({ onRefresh });
        fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
        expect(onRefresh).toHaveBeenCalledTimes(1);
    });

    it('shows the empty state text when nothing matches', () => {
        renderPage();
        fireEvent.input(screen.getByTestId('stats-search-input'), {
            target: { value: 'zzz' },
        });
        expect(screen.getByText('Nothing found')).toBeInTheDocument();
    });

    it('uses the stored sort when no URL params are present', () => {
        LocalStorageHelper.setItem('top_queried_domains_sort', {
            key: 'name',
            direction: 'asc',
        });
        renderPage();

        const rowsText = getRowsText();
        expect(rowsText[0]).toContain('a.org');
        expect(rowsText[2]).toContain('c.org');
    });

    it('uses sort from URL params, overriding stored options', () => {
        LocalStorageHelper.setItem('top_queried_domains_sort', {
            key: 'name',
            direction: 'asc',
        });
        window.location.hash = '#/?sort=name&dir=desc';
        renderPage();

        const rowsText = getRowsText();
        expect(rowsText[0]).toContain('c.org');
        expect(rowsText[2]).toContain('a.org');
    });

    it('persists sort to localStorage and URL on header click', async () => {
        // jsdom does not reflect history.replaceState in location.hash, so
        // assert on the history call itself.
        const replaceSpy = vi.spyOn(window.history, 'replaceState');

        renderPage();
        fireEvent.click(screen.getByTestId('table-header-count'));

        expect(LocalStorageHelper.getItem('top_queried_domains_sort')).toEqual({
            key: 'count',
            direction: 'asc',
        });

        // The router navigates inside a Solid transition, flushed in a microtask.
        await new Promise((resolve) => setTimeout(resolve, 0));
        await new Promise((resolve) => setTimeout(resolve, 0));

        const calls = replaceSpy.mock.calls.map((call) => String(call[2]));
        expect(
            calls.some((url) => url.includes('sort=count') && url.includes('dir=asc')),
        ).toBe(true);
    });
});
