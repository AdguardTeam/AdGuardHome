export type StatsParams = {
    /**
     * The lookback period for statistics in milliseconds.  The interval must
     * be a multiple of one hour and must not be greater than the value of
     * `statistics.interval`.
     */
    recent?: number;
};
