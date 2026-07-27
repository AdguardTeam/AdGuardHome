import type { DNSConfig } from './dNSConfig';

export type DnsInfo200 = DNSConfig & {
    default_local_ptr_upstreams?: string[];
};
