import type { DhcpConfigV4 } from './dhcpConfigV4';
import type { DhcpConfigV6 } from './dhcpConfigV6';

export interface DhcpConfig {
    enabled?: boolean;
    interface_name?: string;
    v4?: DhcpConfigV4;
    v6?: DhcpConfigV6;
}
