import { createSignal } from 'solid-js';
import { render, screen, fireEvent, cleanup } from '@solidjs/testing-library';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { Header } from 'panel/components/QueryLog/blocks/Header/Header';

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const renderHeader = (initialSearch = '') => {
    const [currentSearch, setCurrentSearch] = createSignal(initialSearch);
    const onSearch = vi.fn();

    render(() => (
        <Header
            currentSearch={currentSearch()}
            currentStatus="all"
            currentReason="all"
            isLoading={false}
            onSearch={onSearch}
            onRefresh={vi.fn()}
            onStatusFilterChange={vi.fn()}
            onReasonFilterChange={vi.fn()}
        />
    ));

    return { setCurrentSearch, onSearch };
};

const searchInput = () => screen.getByTestId('query-log-search-input') as HTMLInputElement;

const typeIntoSearch = (value: string) => {
    fireEvent.input(searchInput(), { target: { value } });
};

describe('QueryLog Header search input', () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
        cleanup();
    });

    it('keeps the typed text when the submitted value is echoed back', () => {
        const { setCurrentSearch, onSearch } = renderHeader();

        typeIntoSearch('A ');
        vi.advanceTimersByTime(500);

        expect(onSearch).toHaveBeenCalledWith('A');

        setCurrentSearch('A');

        expect(searchInput().value).toBe('A ');
    });

    it('applies an external search value', () => {
        const { setCurrentSearch } = renderHeader();

        setCurrentSearch('blocked.example');

        expect(searchInput().value).toBe('blocked.example');
    });

    it('applies an external value that repeats an earlier submission', () => {
        const { setCurrentSearch, onSearch } = renderHeader();

        typeIntoSearch('A');
        vi.advanceTimersByTime(500);
        expect(onSearch).toHaveBeenCalledWith('A');

        setCurrentSearch('B');
        expect(searchInput().value).toBe('B');

        setCurrentSearch('A');
        expect(searchInput().value).toBe('A');
    });

    it('cancels the pending search when cleared and stays empty', () => {
        const { setCurrentSearch, onSearch } = renderHeader();

        typeIntoSearch('A');
        fireEvent.click(screen.getByTestId('query-log-search-clear-button'));

        vi.advanceTimersByTime(500);

        expect(onSearch).toHaveBeenCalledTimes(1);
        expect(onSearch).toHaveBeenCalledWith('');

        setCurrentSearch('');
        expect(searchInput().value).toBe('');
    });
});
