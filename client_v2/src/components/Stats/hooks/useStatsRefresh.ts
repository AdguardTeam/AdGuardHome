import { useSearchParams } from '@solidjs/router';

import { statsState, getStats, getStatsConfig } from 'panel/stores/stats';
import { resolveStatsPeriod } from 'panel/helpers/statistics';

/**
 * Refresh callback shared by all stats detail pages: fetches stats for the
 * period from the URL (`?period=<ms>`) or the last stored period.
 *
 * The stats config is loaded first (if not yet fetched) so that the URL period
 * is clamped by the real server retention and not by the default `DAY` value
 * that the store starts with before `getStatsConfig` resolves.
 */
export const useStatsRefresh = () => {
    const [searchParams] = useSearchParams<{ period?: string }>();

    return async () => {
        if (!statsState.configLoaded) {
            await getStatsConfig();
        }
        getStats(resolveStatsPeriod(searchParams, statsState.interval));
    };
};
