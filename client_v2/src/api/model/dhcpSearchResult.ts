import type { DhcpSearchV4 } from './dhcpSearchV4';
import type { DhcpSearchV6 } from './dhcpSearchV6';

/**
 * Information about a DHCP server discovered in the current network.
 */
export interface DhcpSearchResult {
    v4?: DhcpSearchV4;
    v6?: DhcpSearchV6;
}
