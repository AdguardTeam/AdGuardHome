import type { DhcpSearchResultStaticIPStatic } from './dhcpSearchResultStaticIPStatic';

export interface DhcpSearchResultStaticIP {
    /** The result of determining static IP address. */
    static?: DhcpSearchResultStaticIPStatic;
    /** Set if static=no */
    ip?: string;
}
