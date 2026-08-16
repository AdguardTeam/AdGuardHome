import type { QueryLogItemClientWhois } from './queryLogItemClientWhois';

/**
 * Client information for a query log item.
 */
export interface QueryLogItemClient {
    /** Whether the client's IP is blocked or not. */
    disallowed: boolean;
    /** The rule due to which the client is allowed or blocked. */
    disallowed_rule: string;
    /** Persistent client's name or runtime client's hostname.  May be empty. */
    name: string;
    whois: QueryLogItemClientWhois;
}
