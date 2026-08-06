/**
 * Network interface info
 */
export interface NetInterface {
    /** Flags could be any combination of the following values, divided by the "|" character: "up", "broadcast", "loopback", "pointtopoint" and "multicast". */
    flags: string;
    hardware_address: string;
    /** The addresses of the interface. */
    ip_addresses: string[];
    /** MTU value of the interface. */
    mtu: number;
    name: string;
}
