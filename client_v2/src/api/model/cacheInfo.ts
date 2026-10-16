/**
 * Current state of the DNS cache
 */
export interface CacheInfo {
    /** Is the DNS cache enabled */
    enabled: boolean;
    /** Number of entries currently stored in the DNS cache */
    entries: number;
    /** Size of the data currently stored in the DNS cache, in bytes */
    size_bytes: number;
    /** Maximum size of the data that can be stored in the DNS cache, in bytes.  It is doubled when EDNS Client Subnet is enabled, since the responses for subnets are stored separately. */
    max_size_bytes: number;
}
