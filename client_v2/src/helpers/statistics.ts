import { DAY, HOUR } from 'panel/helpers/constants';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';

/**
 * Client-side search match: substring, case-insensitive.
 * A double-quoted query ("...") matches a whitespace-separated token of the
 * value exactly (so an IP matches a "IP display-name" search text).
 * Blank/whitespace/quotes-only queries match everything.
 */
export const isQueryMatch = (value: string, query: string): boolean => {
    const trimmed = query.trim();
    if (!trimmed) {
        return true;
    }

    const quoted = trimmed.match(/^"([\s\S]*)"$/);
    if (quoted) {
        return !quoted[1] || value.split(/\s+/).includes(quoted[1]);
    }

    return value.toLowerCase().includes(trimmed.toLowerCase());
};

/** Stats period saved by the dashboard; defaults to the default stats interval (24h = DAY). */
export const getStoredStatsPeriod = (): number => {
    const savedPeriod = LocalStorageHelper.getItem<number>(LOCAL_STORAGE_KEYS.STATS_PERIOD);
    return typeof savedPeriod === 'number' && Number.isFinite(savedPeriod) && savedPeriod > 0
        ? savedPeriod
        : DAY;
};

/** Maximum stats interval from the server config, clamped to at least a day. */
const getClampedMaxInterval = (maxInterval: number): number =>
    maxInterval >= HOUR ? maxInterval : DAY;

/** Saved period clamped by the maximum interval available in the stats config. */
export const getEffectiveStatsPeriod = (maxInterval: number): number =>
    Math.min(getStoredStatsPeriod(), getClampedMaxInterval(maxInterval));

/** Stats period from URL search params; null when absent or invalid. */
export const getStatsPeriodFromUrl = (params: { period?: string }): number | null => {
    if (params.period == null) {
        return null;
    }
    const period = Number(params.period);
    return Number.isFinite(period) && period > 0 ? period : null;
};

/** Stats period from URL (if valid, clamped by the max interval), otherwise stored/period. */
export const resolveStatsPeriod = (params: { period?: string }, maxInterval: number): number => {
    const urlPeriod = getStatsPeriodFromUrl(params);
    if (urlPeriod !== null) {
        return Math.min(urlPeriod, getClampedMaxInterval(maxInterval));
    }
    return getEffectiveStatsPeriod(maxInterval);
};

/** Percentage of a part in a total; 0 when the total is 0 (never NaN). */
export const computePercent = (count: number, total: number): number =>
    total > 0 ? (count / total) * 100 : 0;
