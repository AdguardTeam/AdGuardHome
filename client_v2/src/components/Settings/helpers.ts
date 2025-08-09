import intl from 'panel/common/intl';
import { HOUR, DAY } from 'panel/helpers/constants';

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
