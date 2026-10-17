import { render, screen, fireEvent } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { FiltersConfig } from 'panel/components/Settings/FiltersConfig';
import { getFilteringStatus } from 'panel/stores/filtering';

const mocks = vi.hoisted(() => ({
    apiSetFiltersConfig: vi.fn(() => Promise.resolve(undefined)),
    apiGetFilteringStatus: vi.fn(() => Promise.resolve({})),
}));

vi.mock('panel/api/generated', () => ({
    filteringConfig: mocks.apiSetFiltersConfig,
    filteringStatus: mocks.apiGetFilteringStatus,
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

/**
 * `FiltersConfig` puts router links in its description, so it must render
 * inside a route for `<Link>` to resolve.
 */
const renderFiltersConfig = (initialValues: { interval: number; enabled: boolean }) => {
    const history = createMemoryHistory();
    history.set({ value: '/' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route
                path="/"
                component={() => (
                    <FiltersConfig initialValues={initialValues} processing={false} />
                )}
            />
        </MemoryRouter>
    ));
};

describe('FiltersConfig', () => {
    beforeEach(() => vi.clearAllMocks());

    it('does not call setFiltersConfig on mount when values did not change', () => {
        renderFiltersConfig({ interval: 24, enabled: true });

        expect(mocks.apiSetFiltersConfig).not.toHaveBeenCalled();
    });

    it('does not overwrite a 1-hour interval while the status is still loading', async () => {
        mocks.apiGetFilteringStatus.mockResolvedValue({
            enabled: true,
            interval: 1,
            filters: [],
            whitelist_filters: [],
            clients_filters: [],
            user_rules: [],
        });

        const statusPromise = getFilteringStatus();

        renderFiltersConfig({ interval: 24, enabled: true });

        await statusPromise;

        expect(mocks.apiSetFiltersConfig).not.toHaveBeenCalled();
        expect(mocks.apiGetFilteringStatus).toHaveBeenCalled();
    });

    it('calls setFiltersConfig when the switch is toggled', async () => {
        renderFiltersConfig({ interval: 1, enabled: true });

        fireEvent.click(screen.getByRole('checkbox'));

        expect(mocks.apiSetFiltersConfig).toHaveBeenCalledWith({
            interval: 1,
            enabled: false,
        });
    });
});
