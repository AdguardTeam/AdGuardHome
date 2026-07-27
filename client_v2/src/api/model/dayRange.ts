/**
 * The single interval within a day.  It begins at the `start` and ends before the `end`.
 */
export interface DayRange {
    /**
     * The number of milliseconds elapsed from the start of a day.  It must be less than `end` and is expected to be rounded to minutes. So the maximum value is `86340000` (23 hours and 59 minutes).
     * @minimum 0
     * @maximum 86340000
     */
    start?: number;
    /**
     * The number of milliseconds elapsed from the start of a day.  It is expected to be rounded to minutes.  The maximum value is `86400000` (24 hours).
     * @minimum 0
     * @maximum 86400000
     */
    end?: number;
}
