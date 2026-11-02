import type { SafeSearchConfig } from './safeSearchConfig';
import type { WhoisInfo } from './whoisInfo';

/**
 * Client information.
 */
export interface ClientFindSubEntry {
    /** Name */
    name?: string;
    /** IP, CIDR, MAC, or ClientID. */
    ids?: string[];
    use_global_settings?: boolean;
    filtering_enabled?: boolean;
    parental_enabled?: boolean;
    safebrowsing_enabled?: boolean;
    /** @deprecated */
    safesearch_enabled?: boolean;
    safe_search?: SafeSearchConfig;
    use_global_blocked_services?: boolean;
    blocked_services?: string[];
    upstreams?: string[];
    whois_info?: WhoisInfo;
    /** Whether the client's IP is blocked or not. */
    disallowed?: boolean;
    /** The rule due to which the client is disallowed.  If disallowed is set to true, and this string is empty, then the client IP is disallowed by the "allowed IP list", that is it is not included in the allowed list. */
    disallowed_rule?: string;
    ignore_querylog?: boolean;
    ignore_statistics?: boolean;
}
