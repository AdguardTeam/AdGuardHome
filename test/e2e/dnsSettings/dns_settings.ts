export interface DnsConfig {
    // Upstream settings
    bootstrap_dns?: string[];
    upstream_dns?: string[];
    upstream_dns_file?: string;
    protection_enabled?: boolean;
    fallback_dns?: string[];
    local_ptr_upstreams?: string[];
    use_private_ptr_resolvers?: boolean;
    resolve_clients?: boolean;
    upstream_mode?: string;

    // Rate limit
    ratelimit?: number;
    ratelimit_subnet_len_ipv4?: number;
    ratelimit_subnet_len_ipv6?: number;
    ratelimit_whitelist?: string[];

    // Blocking
    blocking_mode?: 'default' | 'refused' | 'nxdomain' | 'null_ip' | 'custom_ip';
    blocking_ipv4?: string;
    blocking_ipv6?: string;
    blocked_response_ttl?: number;

    // EDNS / DNSSEC / IPv6
    edns_cs_enabled?: boolean;
    edns_cs_use_custom?: boolean;
    edns_cs_custom_ip?: string;
    disable_ipv6?: boolean;
    dnssec_enabled?: boolean;

    // Cache
    cache_size?: number;
    cache_ttl_min?: number;
    cache_ttl_max?: number;
    cache_optimistic?: boolean;
}

export interface AccessConfig {
    allowed_clients?: string[];
    disallowed_clients?: string[];
    blocked_hosts?: string[];
}

export async function setDnsConfig(baseUrl: string, config: DnsConfig, headers: HeadersInit = {}): Promise<Response> {
    const combinedHeaders = {
        'Content-Type': 'application/json',
        ...headers,
    };
    const response = await fetch(`${baseUrl}/control/dns_config`, {
        method: 'POST',
        headers: combinedHeaders,
        body: JSON.stringify(config),
    });
    return response;
}

export async function setAccessConfig(baseUrl: string, config: AccessConfig, headers: HeadersInit = {}): Promise<Response> {
    const combinedHeaders = {
        'Content-Type': 'application/json',
        ...headers,
    };
    const response = await fetch(`${baseUrl}/control/access/set`, {
        method: 'POST',
        headers: combinedHeaders,
        body: JSON.stringify(config),
    });
    return response;
}

export async function getDnsInfo(baseUrl: string, headers: HeadersInit = {}): Promise<DnsConfig> {
    const response = await fetch(`${baseUrl}/control/dns_info`, { headers });
    if (!response.ok) {
        throw new Error(`Failed to get DNS info: ${response.statusText}`);
    }
    return await response.json();
}

export async function clearDnsCache(baseUrl: string, headers: HeadersInit = {}): Promise<Response> {
    // Try /control/cache_clear first, if 404 try /control/dns_cache_clear
    let response = await fetch(`${baseUrl}/control/cache_clear`, {
        method: 'POST',
        headers,
    });
    if (response.status === 404) {
        response = await fetch(`${baseUrl}/control/dns_cache_clear`, {
            method: 'POST',
            headers,
        });
    }
    return response;
}
