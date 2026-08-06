import type { DayRange } from './dayRange';

/**
 * Sets periods of inactivity for filtering blocked services.  The schedule contains 7 days (Sunday to Saturday) and a time zone.
 */
export interface Schedule {
    /** Time zone name according to IANA time zone database.  For example `Europe/Brussels`.  `Local` represents the system's local time zone. */
    time_zone?: string;
    sun?: DayRange;
    mon?: DayRange;
    tue?: DayRange;
    wed?: DayRange;
    thu?: DayRange;
    fri?: DayRange;
    sat?: DayRange;
}
