/**
 * Represent the number of hits or time duration per key (url, domain, or client IP).
 */
export interface TopArrayEntry {
    domain_or_ip?: number;
    [key: string]: unknown;
}
