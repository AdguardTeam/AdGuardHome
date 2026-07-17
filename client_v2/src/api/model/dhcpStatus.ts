import type { DhcpConfigV4 } from './dhcpConfigV4';
import type { DhcpConfigV6 } from './dhcpConfigV6';
import type { DhcpLease } from './dhcpLease';
import type { DhcpStaticLease } from './dhcpStaticLease';

/**
 * Built-in DHCP server configuration and status
 */
export interface DhcpStatus {
    enabled?: boolean;
    interface_name?: string;
    v4?: DhcpConfigV4;
    v6?: DhcpConfigV6;
    leases: DhcpLease[];
    static_leases?: DhcpStaticLease[];
}
