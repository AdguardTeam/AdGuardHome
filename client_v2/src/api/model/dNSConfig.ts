import type { DNSConfigBlockingMode } from './dNSConfigBlockingMode';
import type { DNSConfigUpstreamMode } from './dNSConfigUpstreamMode';

/**
 * DNS server configuration
 */
export interface DNSConfig {
    /** Bootstrap servers, port is optional after colon.  Empty value will reset it to default values. */
    bootstrap_dns?: string[];
    /** Upstream servers, port is optional after colon.  Empty value will reset it to default values. */
    upstream_dns?: string[];
    /** List of fallback DNS servers used when upstream DNS servers are not responding.  Empty value will clear the list. */
    fallback_dns?: string[];
    upstream_dns_file?: string;
    protection_enabled?: boolean;
    ratelimit?: number;
    /**
     * Length of the subnet mask for IPv4 addresses.
     * @minimum 0
     * @maximum 32
     */
    ratelimit_subnet_subnet_len_ipv4?: number;
    /**
     * Length of the subnet mask for IPv6 addresses.
     * @minimum 0
     * @maximum 128
     */
    ratelimit_subnet_subnet_len_ipv6?: number;
    /** List of IP addresses excluded from rate limiting. */
    ratelimit_whitelist?: string[];
    blocking_mode?: DNSConfigBlockingMode;
    blocking_ipv4?: string;
    blocking_ipv6?: string;
    /**
     * TTL for blocked responses.
     * @minimum 0
     */
    blocked_response_ttl?: number;
    /** Protection is pause until this time.  Nullable. */
    protection_disabled_until?: string;
    edns_cs_enabled?: boolean;
    edns_cs_use_custom?: boolean;
    edns_cs_custom_ip?: string;
    disable_ipv6?: boolean;
    dnssec_enabled?: boolean;
    cache_size?: number;
    cache_ttl_min?: number;
    cache_ttl_max?: number;
    /**
     * Enables or disables the DNS response cache.
     *
     * If `cache_enabled` is `true`, the companion field `cache_size` must
     * be present and greater than 0, or the `dns.cache_size` setting in
     * the configuration file must already be greater than 0.
     */
    cache_enabled?: boolean;
    cache_optimistic?: boolean;
    /** Upstream modes enumeration. The empty string value is deprecated; use `load_balance` instead. */
    upstream_mode?: DNSConfigUpstreamMode;
    use_private_ptr_resolvers?: boolean;
    resolve_clients?: boolean;
    /** Upstream servers, port is optional after colon.  Empty value will reset it to default values. */
    local_ptr_upstreams?: string[];
    /**
     * The number of seconds to wait for a response from the upstream server
     * @minimum 1
     */
    upstream_timeout?: number;
}
