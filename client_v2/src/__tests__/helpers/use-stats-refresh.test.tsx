import { onMount } from 'solid-js';
import { render } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => {
    const statsState = { interval: 86_400_000, configLoaded: true };

    return {
        statsState,
        getStats: vi.fn(),
        getStatsConfig: vi.fn(),
    };
});

vi.mock('panel/stores/stats', () => ({
    statsState: mocks.statsState,
    getStats: mocks.getStats,
    getStatsConfig: mocks.getStatsConfig,
}));

import { useStatsRefresh } from 'panel/components/Stats/hooks/useStatsRefresh';

function Harness(): null {
    const refreshStats = useStatsRefresh();
    onMount(() => {
        void refreshStats();
    });

    return null;
}

const renderAt = (path: string) => {
    const history = createMemoryHistory();
    history.set({ value: path });

    return render(() => (
        <MemoryRouter history={history}>
            <Route path="/" component={Harness} />
        </MemoryRouter>
    ));
};

describe('useStatsRefresh', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
        mocks.statsState.configLoaded = true;
        mocks.statsState.interval = 86_400_000;
        mocks.getStatsConfig.mockResolvedValue(undefined);
    });

    it('calls getStats with the ?period URL param (clamped by the max interval)', () => {
        renderAt('/?period=3600000');
        expect(mocks.getStats).toHaveBeenCalledTimes(1);
        expect(mocks.getStats).toHaveBeenCalledWith(3_600_000);
    });

    it('falls back to the default DAY period when there is no URL param', () => {
        renderAt('/');
        expect(mocks.getStats).toHaveBeenCalledTimes(1);
        expect(mocks.getStats).toHaveBeenCalledWith(86_400_000);
    });

    it('loads the stats config before fetching stats when it is not loaded yet', async () => {
        // Simulate a fresh page reload directly on a stats page: the config has
        // not been fetched yet, so the store interval is still the default DAY.
        mocks.statsState.configLoaded = false;
        mocks.statsState.interval = 86_400_000;
        mocks.getStatsConfig.mockImplementation(async () => {
            // The server reports a 30-day retention interval.
            mocks.statsState.configLoaded = true;
            mocks.statsState.interval = 2_592_000_000;
        });

        renderAt('/?period=2592000000');

        await vi.waitFor(() => {
            expect(mocks.getStatsConfig).toHaveBeenCalledTimes(1);
        });
        // The URL period must not be clamped by the uninitialized DAY default.
        expect(mocks.getStats).toHaveBeenCalledWith(2_592_000_000);
    });
});
