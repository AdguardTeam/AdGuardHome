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

const SAFESEARCH_TITLES: Record<string, string> = {
    bing: 'Bing',
    duckduckgo: 'DuckDuckGo',
    ecosia: 'Ecosia',
    google: 'Google',
    pixabay: 'Pixabay',
    yandex: 'Yandex',
    youtube: 'YouTube',
};

export const getSafeSearchProviderTitle = (key: string) => {
    return SAFESEARCH_TITLES[key] ?? captitalizeWords(key);
};
