/**
 * Query log configuration
 */
export interface GetQueryLogConfigResponse {
    /** Is query log enabled */
    enabled: boolean;
    /** Time period for query log rotation in milliseconds. */
    interval: number;
    /** Anonymize clients' IP addresses */
    anonymize_client_ip: boolean;
    /** List of host names, which should not be written to log */
    ignored: string[];
    /** If true, the host names in the `ignored` array are excluded from the query log. */
    ignored_enabled?: boolean;
}
