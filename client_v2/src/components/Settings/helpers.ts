import intl from 'panel/common/intl';
import { HOUR, DAY, RETENTION_CUSTOM } from 'panel/helpers/constants';
import { captitalizeWords } from '../../helpers/helpers';

export const formatIntervalText = (intervalMs: number) => {
    if (intervalMs === 6 * HOUR) {
        return intl.getPlural('settings_hours', 6);
    }
    if (intervalMs === DAY) {
        return intl.getPlural('settings_hours', 24);
    }
    if (intervalMs % DAY === 0) {
        return intl.getPlural('settings_days', intervalMs / DAY);
    }
    return intl.getPlural('settings_hours', Math.floor(intervalMs / HOUR));
};

export const getIntervalTitle = (intervalMs: number) => {
    if (intervalMs === RETENTION_CUSTOM) {
        return intl.getMessage('settings_custom');
    }
    return formatIntervalText(intervalMs);
};

export const getDefaultInterval = (customInterval?: number, interval?: number) => {
    if (customInterval && customInterval > 0) {
        return RETENTION_CUSTOM;
    }
    return interval || DAY;
};

export const resolveInterval = (interval: number, customInterval?: number | null): number => {
    if (customInterval) {
        return customInterval >= HOUR ? customInterval : customInterval * HOUR;
    }

    return interval;
};

export const getRetentionSummary = (intervalMs: number) => {
    if (intervalMs === 6 * HOUR) {
        return intl.getPlural('last_hours', 6);
    }
    if (intervalMs === DAY) {
        return intl.getPlural('last_hours', 24);
    }
    if (intervalMs % DAY === 0) {
        return intl.getPlural('last_days', intervalMs / DAY);
    }
    return intl.getPlural('last_hours', Math.floor(intervalMs / HOUR));
};

const SAFESEARCH_TITLES = {
    bing: 'Bing',
    duckduckgo: 'DuckDuckGo',
    ecosia: 'Ecosia',
    google: 'Google',
    pixabay: 'Pixabay',
    yandex: 'Yandex',
    youtube: 'YouTube',
} as const;

export const getSafeSearchProviderTitle = (key: string) => {
    return SAFESEARCH_TITLES[key as keyof typeof SAFESEARCH_TITLES] ?? captitalizeWords(key);
};

export type QueryLogConfig = {
    enabled: boolean;
    anonymize_client_ip: boolean;
    interval: number;
    ignored: string[];
    ignored_enabled: boolean;
};

export type StatsConfig = {
    enabled: boolean;
    interval: number;
    ignored: string[];
    ignored_enabled: boolean;
};

/**
 * Builds a backend-ready query-log config payload from the queryLogs store
 * (or any object with the required fields), applying optional overrides.
 */
export const buildQueryLogConfig = (
    state: QueryLogConfig,
    overrides?: Partial<QueryLogConfig>,
): QueryLogConfig => ({
    enabled: state.enabled,
    anonymize_client_ip: state.anonymize_client_ip,
    interval: state.interval,
    ignored: state.ignored,
    ignored_enabled: state.ignored_enabled,
    ...overrides,
});

/**
 * Builds a backend-ready stats config payload from the stats store,
 * applying optional overrides. Strips runtime-only fields.
 */
export const buildStatsConfig = (
    state: StatsConfig,
    overrides?: Partial<StatsConfig>,
): StatsConfig => ({
    enabled: state.enabled,
    interval: state.interval,
    ignored: state.ignored,
    ignored_enabled: state.ignored_enabled,
    ...overrides,
});
