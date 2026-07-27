import type { QueryLogConfigInterval } from './queryLogConfigInterval';

/**
 * Query log configuration
 */
export interface QueryLogConfig {
    /** Is query log enabled */
    enabled?: boolean;
    /** Time period for query log rotation. */
    interval?: QueryLogConfigInterval;
    /** Anonymize clients' IP addresses */
    anonymize_client_ip?: boolean;
}
