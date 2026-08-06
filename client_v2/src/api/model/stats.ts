import type { StatsTimeUnits } from './statsTimeUnits';
import type { TopArrayEntry } from './topArrayEntry';

/**
 * Server statistics data
 */
export interface Stats {
    /** Time units */
    time_units?: StatsTimeUnits;
    /** Total number of DNS queries */
    num_dns_queries?: number;
    /** Number of requests blocked by filtering rules */
    num_blocked_filtering?: number;
    /** Number of requests blocked by safebrowsing module */
    num_replaced_safebrowsing?: number;
    /** Number of requests blocked by safesearch module */
    num_replaced_safesearch?: number;
    /** Number of blocked adult websites */
    num_replaced_parental?: number;
    /** Average time in seconds on processing a DNS request */
    avg_processing_time?: number;
    top_queried_domains?: TopArrayEntry[];
    top_clients?: TopArrayEntry[];
    top_blocked_domains?: TopArrayEntry[];
    /**
     * Total number of responses from each upstream.
     * @maxItems 100
     */
    top_upstreams_responses?: TopArrayEntry[];
    /**
     * Average processing time in seconds of requests from each upstream.
     * @maxItems 100
     */
    top_upstreams_avg_time?: TopArrayEntry[];
    dns_queries?: number[];
    blocked_filtering?: number[];
    replaced_safebrowsing?: number[];
    replaced_parental?: number[];
}
