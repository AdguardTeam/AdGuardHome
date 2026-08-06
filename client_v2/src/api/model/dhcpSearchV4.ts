import type { DhcpSearchResultOtherServer } from './dhcpSearchResultOtherServer';
import type { DhcpSearchResultStaticIP } from './dhcpSearchResultStaticIP';

export interface DhcpSearchV4 {
    other_server?: DhcpSearchResultOtherServer;
    static_ip?: DhcpSearchResultStaticIP;
}
