import { render, screen, waitFor } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { filteringSetURL, filteringStatus } from 'panel/api/generated';
import intl from 'panel/common/intl';
import { FilterToggleProgress } from 'panel/components/FilterLists/blocks/FilterToggleProgress';
import { editFilter, filteringState, toggleFilterStatus } from 'panel/stores/filtering';

vi.mock('panel/api/generated', () => ({
    filteringAddURL: vi.fn(),
    filteringCheckHost: vi.fn(),
    filteringConfig: vi.fn(),
    filteringRefresh: vi.fn(),
    filteringRemoveURL: vi.fn(),
    filteringSetRules: vi.fn(),
    filteringSetURL: vi.fn(),
    filteringStatus: vi.fn(),
}));

const createDeferred = <T,>() => {
    let resolve: (value: T) => void = () => {};
    let reject: (reason?: unknown) => void = () => {};
    const promise = new Promise<T>((resolvePromise, rejectPromise) => {
        resolve = resolvePromise;
        reject = rejectPromise;
    });

    return { promise, reject, resolve };
};

const filterURL = 'https://example.org/filter.txt';
const filterData = {
    enabled: false,
    name: 'Example',
    url: filterURL,
};

describe('FilterToggleProgress', () => {
    afterEach(() => {
        vi.mocked(filteringSetURL).mockReset();
        vi.mocked(filteringStatus).mockReset();
    });

    it('blocks the interface and explains the filter rebuild while it is pending', async () => {
        const request = createDeferred<void>();
        const status = createDeferred<Awaited<ReturnType<typeof filteringStatus>>>();
        vi.mocked(filteringSetURL).mockReturnValue(request.promise);
        vi.mocked(filteringStatus).mockReturnValue(status.promise);

        const user = userEvent.setup();
        render(() => (
            <>
                <button type="button">Background action</button>
                <FilterToggleProgress />
            </>
        ));
        const backgroundAction = screen.getByRole('button', { name: 'Background action' });

        const togglePromise = toggleFilterStatus(filterURL, filterData, false);

        const dialog = await screen.findByRole('dialog', {
            name: intl.getMessage('filter_toggle_progress_title'),
        });
        expect(dialog).toHaveAttribute('aria-modal', 'true');
        expect(dialog).toHaveAttribute('aria-busy', 'true');
        await waitFor(() => {
            expect(backgroundAction.closest('[aria-hidden="true"]')).not.toBeNull();
            expect(dialog.contains(document.activeElement)).toBe(true);
        });
        expect(
            screen.getByText(intl.getMessage('filter_toggle_progress_desc')),
        ).toBeInTheDocument();

        await user.keyboard('{Escape}');
        expect(dialog).toBeInTheDocument();

        request.resolve();
        await waitFor(() => {
            expect(filteringStatus).toHaveBeenCalledOnce();
        });
        expect(dialog).toBeInTheDocument();

        status.resolve({});
        await togglePromise;

        await waitFor(() => {
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        });
    });

    it('keeps edit progress visible until the refreshed status is loaded', async () => {
        const request = createDeferred<void>();
        const status = createDeferred<Awaited<ReturnType<typeof filteringStatus>>>();
        vi.mocked(filteringSetURL).mockReturnValue(request.promise);
        vi.mocked(filteringStatus).mockReturnValue(status.promise);

        render(() => <FilterToggleProgress />);
        const editPromise = editFilter(filterURL, filterData, false);

        const dialog = await screen.findByRole('dialog');
        request.resolve();
        await waitFor(() => {
            expect(filteringStatus).toHaveBeenCalledOnce();
        });
        expect(dialog).toBeInTheDocument();

        status.resolve({});
        await editPromise;
        await waitFor(() => {
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        });
        expect(filteringState.processingConfigFilter).toBe(false);
    });

    it.each([
        ['toggle', () => toggleFilterStatus(filterURL, filterData, false)],
        ['edit', () => editFilter(filterURL, filterData, false)],
    ])('clears %s progress when the update request is rejected', async (_name, runAction) => {
        const request = createDeferred<void>();
        vi.mocked(filteringSetURL).mockReturnValue(request.promise);

        render(() => <FilterToggleProgress />);
        const actionPromise = runAction();

        await screen.findByRole('dialog');
        request.reject(new Error('request failed'));
        await actionPromise;

        await waitFor(() => {
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        });
        expect(filteringStatus).not.toHaveBeenCalled();
        expect(filteringState.processingConfigFilter).toBe(false);
    });

    it('keeps progress visible until all overlapping updates finish', async () => {
        const firstRequest = createDeferred<void>();
        const secondRequest = createDeferred<void>();
        vi.mocked(filteringSetURL)
            .mockReturnValueOnce(firstRequest.promise)
            .mockReturnValueOnce(secondRequest.promise);
        vi.mocked(filteringStatus).mockResolvedValue({});

        render(() => <FilterToggleProgress />);
        const firstAction = toggleFilterStatus(filterURL, filterData, false);
        const secondAction = editFilter(filterURL, filterData, false);

        await screen.findByRole('dialog');
        firstRequest.resolve();
        await firstAction;

        try {
            expect(filteringState.processingConfigFilter).toBe(true);
            expect(screen.getByRole('dialog')).toBeInTheDocument();
        } finally {
            secondRequest.resolve();
            await secondAction;
        }

        await waitFor(() => {
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        });
        expect(filteringStatus).toHaveBeenCalledTimes(2);
        expect(filteringState.processingConfigFilter).toBe(false);
    });
});
