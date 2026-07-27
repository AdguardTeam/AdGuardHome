/**
 * Request for checking for other DHCP servers in the network.
 */
export interface DhcpFindActiveReq {
    /** The name of the network interface */
    interface?: string;
}
