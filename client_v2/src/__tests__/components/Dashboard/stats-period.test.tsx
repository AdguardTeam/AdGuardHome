import { For } from 'solid-js';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { describe, it, expect, vi, beforeEach } from 'vitest';

import { DAY } from 'panel/helpers/constants';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';

type StatsStateMock = {
    processingGetConfig: boolean;
    processingStats: boolean;
    configLoaded: boolean;
    configAttempted: boolean;
    statsAttempted: boolean;
    interval: number;
    enabled: boolean;
    numDnsQueries: number;
    dnsQueries: number[];
    topClients: { name: string; count: number }[];
    topQueriedDomains: { name: string; count: number }[];
};

type HeaderPropsMock = {
    selectedPeriod: number;
    periodOptions: { value: number; label: string }[];
    onPeriodChange: (period: number) => void;
};

const mocks = vi.hoisted(() => ({
    getStats: vi.fn(),
    getStatsConfig: vi.fn(),
    statsState: undefined as StatsStateMock | undefined,
}));

vi.mock('panel/stores/stats', async () => {
    const { createMutable } = await import('solid-js/store');

    // A mutable store (not a plain object) so the Dashboard memos react when
    // the config "arrives", exactly like the real store does.
    const statsState = createMutable<StatsStateMock>({
        processingGetConfig: false,
        processingStats: true,
        configLoaded: false,
        configAttempted: false,
        statsAttempted: false,
        interval: DAY,
        enabled: true,
        numDnsQueries: 0,
        dnsQueries: [],
        topClients: [],
        topQueriedDomains: [],
    });
    mocks.statsState = statsState;

    return {
        statsState,
        getStats: mocks.getStats,
        getStatsConfig: mocks.getStatsConfig,
        enableStatistics: vi.fn(),
    };
});

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        protectionEnabled: true,
        processingProtection: false,
        protectionDisabledDuration: null,
    },
    toggleProtection: vi.fn(),
    getClients: vi.fn(),
}));

vi.mock('panel/stores/access', () => ({
    accessState: { processing: false },
    getAccessList: vi.fn(),
}));

// The Header is what puts the period into the dropdown, so the stub renders the
// props the Dashboard passes instead of the real Select.
vi.mock('panel/components/Dashboard/blocks/Header/Header', () => ({
    Header: (props: HeaderPropsMock) => (
        <div>
            <span data-testid="selected-period">{props.selectedPeriod}</span>
            <For each={props.periodOptions}>
                {(option) => (
                    <button
                        type="button"
                        data-testid="period-option"
                        data-period={option.value}
                        onClick={() => props.onPeriodChange(option.value)}
                    >
                        {option.label}
                    </button>
                )}
            </For>
        </div>
    ),
    getPeriodLabel: (interval: number) => `period:${interval}`,
}));

vi.mock('panel/components/Dashboard/blocks/StatCards', () => ({ StatCards: (): null => null }));
vi.mock('panel/components/Dashboard/blocks/EmptyState/EmptyState', () => ({
    EmptyState: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/GeneralStatistics', () => ({
    GeneralStatistics: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/TopClients', () => ({ TopClients: (): null => null }));
vi.mock('panel/components/Dashboard/blocks/TopQueriedDomains', () => ({
    TopQueriedDomains: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/TopBlockedDomains', () => ({
    TopBlockedDomains: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/TopUpstreams', () => ({
    TopUpstreams: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/UpstreamAvgTime', () => ({
    UpstreamAvgTime: (): null => null,
}));

import { Dashboard } from 'panel/components/Dashboard/Dashboard';

const renderDashboard = () => render(() => <Dashboard />);

const selectedPeriodInHeader = () => Number(screen.getByTestId('selected-period').textContent);

const periodOptionValues = () =>
    screen
        .queryAllByTestId('period-option')
        .map((option) => Number(option.getAttribute('data-period')));

const clickPeriodOption = (period: number) => {
    const option = screen
        .queryAllByTestId('period-option')
        .find((candidate) => candidate.getAttribute('data-period') === String(period));

    expect(option).toBeDefined();
    fireEvent.click(option as HTMLElement);
};

/** Resolves the config request the way a successful `getStatsConfig` does. */
const resolveConfigWith = (interval: number) => {
    mocks.getStatsConfig.mockImplementation(async () => {
        const statsState = mocks.statsState as StatsStateMock;
        statsState.interval = interval;
        statsState.configLoaded = true;
        statsState.configAttempted = true;
    });
};

/** Resolves the config request the way a failed `getStatsConfig` does. */
const failConfig = () => {
    mocks.getStatsConfig.mockImplementation(async () => {
        (mocks.statsState as StatsStateMock).configAttempted = true;
    });
};

describe('Dashboard stats period', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();

        const statsState = mocks.statsState as StatsStateMock;
        statsState.processingGetConfig = false;
        statsState.processingStats = true;
        statsState.configLoaded = false;
        statsState.configAttempted = false;
        statsState.statsAttempted = false;
        statsState.interval = DAY;
    });

    it('fetches the stored period once, after the stats config resolves', async () => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, DAY * 30);
        resolveConfigWith(DAY * 30);

        renderDashboard();

        await vi.waitFor(() => expect(selectedPeriodInHeader()).toBe(DAY * 30));

        // The default DAY period must never be requested: that request is what
        // painted 24h data before the selected period.
        expect(mocks.getStats).toHaveBeenCalledTimes(1);
        expect(mocks.getStats).toHaveBeenCalledWith(DAY * 30);
        expect(mocks.getStatsConfig).toHaveBeenCalledTimes(1);
    });

    it('shows the stored period while the stats config is still loading', () => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, DAY * 30);
        mocks.getStatsConfig.mockReturnValue(new Promise(() => {}));

        renderDashboard();

        expect(selectedPeriodInHeader()).toBe(DAY * 30);
        expect(periodOptionValues()).toContain(DAY * 30);
        expect(mocks.getStats).not.toHaveBeenCalled();
    });

    it('falls back to the default period when the stats config cannot be loaded', async () => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, DAY * 30);
        // A failed config request marks the attempt so the dashboard stops
        // waiting and fetches; the store clamps the period by the default DAY.
        failConfig();

        renderDashboard();

        await vi.waitFor(() => expect(mocks.getStats).toHaveBeenCalledWith(DAY * 30));
        expect(selectedPeriodInHeader()).toBe(DAY);
        expect(periodOptionValues()).not.toContain(DAY * 30);
    });

    it('fetches a newly selected period without re-reading the stats config', async () => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, DAY * 30);
        resolveConfigWith(DAY * 30);

        renderDashboard();

        await vi.waitFor(() => expect(mocks.getStats).toHaveBeenCalledWith(DAY * 30));

        clickPeriodOption(DAY * 7);

        await vi.waitFor(() => expect(mocks.getStats).toHaveBeenCalledWith(DAY * 7));
        expect(mocks.getStatsConfig).toHaveBeenCalledTimes(1);
    });

    it('never requests stats before the first config attempt settles', async () => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, DAY * 30);

        let resolveConfig!: () => void;
        mocks.getStatsConfig.mockImplementation(
            () =>
                new Promise<void>((resolve) => {
                    resolveConfig = () => {
                        const statsState = mocks.statsState as StatsStateMock;
                        statsState.interval = DAY * 30;
                        statsState.configLoaded = true;
                        statsState.configAttempted = true;
                        resolve();
                    };
                }),
        );

        renderDashboard();

        // Config still in flight: no stats request at all, especially not one
        // for the default DAY that would paint 24h values first.
        expect(mocks.getStats).not.toHaveBeenCalled();

        resolveConfig();

        await vi.waitFor(() => expect(mocks.getStats).toHaveBeenCalledTimes(1));
        expect(mocks.getStats).toHaveBeenCalledWith(DAY * 30);
        expect(mocks.getStats).not.toHaveBeenCalledWith(DAY);
    });
});
