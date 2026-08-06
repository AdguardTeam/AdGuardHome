import { HashRouter, Route } from '@solidjs/router';
import { fireEvent, render, screen } from '@solidjs/testing-library';
import { createSignal, type JSX } from 'solid-js';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { DAY } from 'panel/helpers/constants';
import { Header as DashboardHeader } from 'panel/components/Dashboard/blocks/Header/Header';
import { isQueryLogBusy } from 'panel/components/QueryLog/QueryLog';
import { Header as QueryLogHeader } from 'panel/components/QueryLog/blocks/Header';

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
        getPlural: (key: string, count: number) => `${key}:${count}`,
    },
}));

const renderInRouter = (component: () => JSX.Element) => {
    window.location.hash = '#/';

    return render(() => (
        <HashRouter>
            <Route path="/" component={component} />
        </HashRouter>
    ));
};

describe('Activity clear actions', () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    it('blocks query-log loading while a clear is in progress', () => {
        expect(isQueryLogBusy(false, false, true)).toBe(true);
    });

    it('confirms before clearing the query log', () => {
        const onClear = vi.fn().mockResolvedValue(undefined);

        renderInRouter(() => (
            <QueryLogHeader
                onSearch={vi.fn()}
                onRefresh={vi.fn()}
                onClear={onClear}
                onStatusFilterChange={vi.fn()}
                onReasonFilterChange={vi.fn()}
                currentSearch=""
                currentStatus="all"
                currentReason="all"
                isLoading={false}
                isClearing={false}
            />
        ));

        fireEvent.click(screen.getByTestId('query-log-clear-button-desktop'));

        expect(onClear).not.toHaveBeenCalled();
        expect(screen.getByText('settings_confirm_clear_query_log')).toBeInTheDocument();

        fireEvent.click(screen.getByTestId('query-log-clear-confirm'));

        expect(onClear).toHaveBeenCalledTimes(1);
    });

    it('disables query-log clear actions while clearing', () => {
        renderInRouter(() => (
            <QueryLogHeader
                onSearch={vi.fn()}
                onRefresh={vi.fn()}
                onClear={vi.fn().mockResolvedValue(undefined)}
                onStatusFilterChange={vi.fn()}
                onReasonFilterChange={vi.fn()}
                currentSearch=""
                currentStatus="all"
                currentReason="all"
                isLoading={false}
                isClearing
            />
        ));

        expect(screen.getByTestId('query-log-clear-button-mobile')).toBeDisabled();
        expect(screen.getByTestId('query-log-clear-button-desktop')).toBeDisabled();
        expect(screen.getByTestId('query-log-refresh-button-mobile')).toBeDisabled();
        expect(screen.getByTestId('query-log-refresh-button-desktop')).toBeDisabled();
        expect(screen.getByPlaceholderText('domain_or_client')).toBeDisabled();
    });

    it('disables query-log clear actions while logs are loading', () => {
        renderInRouter(() => (
            <QueryLogHeader
                onSearch={vi.fn()}
                onRefresh={vi.fn()}
                onClear={vi.fn().mockResolvedValue(undefined)}
                onStatusFilterChange={vi.fn()}
                onReasonFilterChange={vi.fn()}
                currentSearch=""
                currentStatus="all"
                currentReason="all"
                isLoading
                isClearing={false}
            />
        ));

        expect(screen.getByTestId('query-log-clear-button-mobile')).toBeDisabled();
        expect(screen.getByTestId('query-log-clear-button-desktop')).toBeDisabled();
    });

    it('disables query-log confirmation if loading starts while it is open', () => {
        const [isLoading, setIsLoading] = createSignal(false);

        renderInRouter(() => (
            <QueryLogHeader
                onSearch={vi.fn()}
                onRefresh={vi.fn()}
                onClear={vi.fn().mockResolvedValue(undefined)}
                onStatusFilterChange={vi.fn()}
                onReasonFilterChange={vi.fn()}
                currentSearch=""
                currentStatus="all"
                currentReason="all"
                isLoading={isLoading()}
                isClearing={false}
            />
        ));

        fireEvent.click(screen.getByTestId('query-log-clear-button-desktop'));
        setIsLoading(true);

        expect(screen.getByTestId('query-log-clear-confirm')).toBeDisabled();
    });

    it('cancels a pending query-log search when clear confirmation opens', () => {
        vi.useFakeTimers();
        const onSearch = vi.fn();

        renderInRouter(() => (
            <QueryLogHeader
                onSearch={onSearch}
                onRefresh={vi.fn()}
                onClear={vi.fn().mockResolvedValue(undefined)}
                onStatusFilterChange={vi.fn()}
                onReasonFilterChange={vi.fn()}
                currentSearch=""
                currentStatus="all"
                currentReason="all"
                isLoading={false}
                isClearing={false}
            />
        ));

        fireEvent.input(screen.getByPlaceholderText('domain_or_client'), {
            target: { value: 'pending.example' },
        });
        fireEvent.click(screen.getByTestId('query-log-clear-button-desktop'));
        vi.runAllTimers();

        expect(screen.getByText('settings_confirm_clear_query_log')).toBeInTheDocument();
        expect(onSearch).not.toHaveBeenCalled();
    });

    it('confirms before clearing dashboard statistics', () => {
        const onClear = vi.fn().mockResolvedValue(undefined);

        renderInRouter(() => (
            <DashboardHeader
                protectionEnabled
                processingProtection={false}
                remainingTime={null}
                selectedPeriod={DAY}
                periodOptions={[{ value: DAY, label: 'last day' }]}
                isLoading={false}
                isClearing={false}
                onToggleProtection={vi.fn()}
                onRefreshStats={vi.fn()}
                onClear={onClear}
                onPeriodChange={vi.fn()}
            />
        ));

        fireEvent.click(screen.getByTestId('dashboard-clear-stats-button-desktop'));

        expect(onClear).not.toHaveBeenCalled();
        expect(screen.getByText('settings_confirm_clear_statistics')).toBeInTheDocument();

        fireEvent.click(screen.getByTestId('dashboard-clear-stats-confirm'));

        expect(onClear).toHaveBeenCalledTimes(1);
    });

    it('disables dashboard clear actions while clearing', () => {
        renderInRouter(() => (
            <DashboardHeader
                protectionEnabled
                processingProtection={false}
                remainingTime={null}
                selectedPeriod={DAY}
                periodOptions={[{ value: DAY, label: 'last day' }]}
                isLoading={false}
                isClearing
                onToggleProtection={vi.fn()}
                onRefreshStats={vi.fn()}
                onClear={vi.fn().mockResolvedValue(undefined)}
                onPeriodChange={vi.fn()}
            />
        ));

        expect(screen.getByTestId('dashboard-clear-stats-button-mobile')).toBeDisabled();
        expect(screen.getByTestId('dashboard-clear-stats-button-desktop')).toBeDisabled();
        for (const refreshButton of screen.getAllByTitle('refresh_btn')) {
            expect(refreshButton).toBeDisabled();
        }
    });

    it('disables dashboard clear actions while statistics are loading', () => {
        renderInRouter(() => (
            <DashboardHeader
                protectionEnabled
                processingProtection={false}
                remainingTime={null}
                selectedPeriod={DAY}
                periodOptions={[{ value: DAY, label: 'last day' }]}
                isLoading
                isClearing={false}
                onToggleProtection={vi.fn()}
                onRefreshStats={vi.fn()}
                onClear={vi.fn().mockResolvedValue(undefined)}
                onPeriodChange={vi.fn()}
            />
        ));

        expect(screen.getByTestId('dashboard-clear-stats-button-mobile')).toBeDisabled();
        expect(screen.getByTestId('dashboard-clear-stats-button-desktop')).toBeDisabled();
    });
});
