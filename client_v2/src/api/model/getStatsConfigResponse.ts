/**
 * Statistics configuration
 */
export interface GetStatsConfigResponse {
    /** Are statistics enabled */
    enabled: boolean;
    /** Statistics rotation interval in milliseconds */
    interval: number;
    /** List of host names, which should not be counted */
    ignored: string[];
    /** If true, the host names in the `ignored` array are excluded from the statistics. */
    ignored_enabled?: boolean;
}
