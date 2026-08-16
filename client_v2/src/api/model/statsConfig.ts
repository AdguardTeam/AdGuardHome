import type { StatsConfigInterval } from './statsConfigInterval';

/**
 * Statistics configuration
 */
export interface StatsConfig {
    /** Time period to keep the data.  `0` means that the statistics is disabled. */
    interval?: StatsConfigInterval;
}
