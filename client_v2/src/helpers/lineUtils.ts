import { format, subDays, subHours } from 'date-fns';

import { TIME_UNITS } from './constants';

/**
 * Returns a human-readable label for a history chart point.
 *
 * The backend returns plain count arrays without timestamps, so labels are
 * synthesized client-side based on the index of the point and the number of
 * points.  The time units ("hours" or "days") are determined by the backend
 * response: buckets are hourly when the interval is 7 days or less, otherwise
 * they are daily.
 *
 * @param index Index of the point, where 0 is the oldest point.
 * @param pointsCount Total number of points in the series.
 * @param timeUnits Time units of the series, "hours" or "days".
 * @param now Current timestamp, used as the reference for the newest point.
 */
export const formatHistoryLabel = (
    index: number,
    pointsCount: number,
    timeUnits: string,
    now: number | Date = Date.now(),
): string => {
    const stepsAgo = pointsCount - 1 - index;

    if (timeUnits === TIME_UNITS.HOURS) {
        return format(subHours(now, stepsAgo), 'd MMM HH:00');
    }

    return format(subDays(now, stepsAgo), 'd MMM yyyy');
};
