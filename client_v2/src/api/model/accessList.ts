/**
 * Client and host access list.  Each of the lists should contain only unique elements.  In addition, allowed and disallowed lists cannot contain the same elements.
 */
export interface AccessList {
    /** The allowlist of clients: IP addresses, CIDRs, or ClientIDs. */
    allowed_clients?: string[];
    /** The blocklist of clients: IP addresses, CIDRs, or ClientIDs. */
    disallowed_clients?: string[];
    /** The blocklist of hosts. */
    blocked_hosts?: string[];
}
