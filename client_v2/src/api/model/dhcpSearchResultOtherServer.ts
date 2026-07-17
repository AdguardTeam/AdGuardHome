import type { DhcpSearchResultOtherServerFound } from './dhcpSearchResultOtherServerFound';

export interface DhcpSearchResultOtherServer {
    /** The result of searching the other DHCP server. */
    found?: DhcpSearchResultOtherServerFound;
    /** Set if found=error */
    error?: string;
}
