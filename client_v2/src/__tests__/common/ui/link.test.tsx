import { render, screen, fireEvent } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

import { Link } from 'panel/common/ui/Link';
import { RoutePath } from 'panel/components/Routes/Paths';

import type { QueryParams } from 'panel/components/Routes/Paths';

/**
 * A memory router keeps each case on its own history: following a link in one
 * test would otherwise leave the next one on a route it does not declare.
 */
const renderRow = (children?: unknown, query?: QueryParams) => {
    const history = createMemoryHistory();
    history.set({ value: '/' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route
                path="/"
                component={() => (
                    <Link to={RoutePath.QueryLog} query={query} data-testid="row">
                        {(children ?? 'row content') as never}
                    </Link>
                )}
            />
        </MemoryRouter>
    ));
};

describe('Link scroll-to-top', () => {
    let scrollTo: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
        vi.useFakeTimers();
        scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
    });

    afterEach(() => {
        scrollTo.mockRestore();
        vi.useRealTimers();
    });

    it('scrolls to the top after following the link', () => {
        renderRow();
        fireEvent.click(screen.getByTestId('row'));

        vi.advanceTimersByTime(150);

        expect(scrollTo).toHaveBeenCalledWith({ top: 0 });
    });

    it('does not scroll when a nested control cancels the navigation', () => {
        renderRow(
            <button type="button" onClick={(e) => e.preventDefault()}>
                Row action
            </button>,
        );
        fireEvent.click(screen.getByRole('button'));

        vi.advanceTimersByTime(150);

        expect(scrollTo).not.toHaveBeenCalled();
    });

    it('does not scroll when the target section handles it', () => {
        renderRow(undefined, { section: 'settings' });
        fireEvent.click(screen.getByTestId('row'));

        vi.advanceTimersByTime(150);

        expect(scrollTo).not.toHaveBeenCalled();
    });
});
