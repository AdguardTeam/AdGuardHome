/**
 * Network interface info
 */
export interface NetInterface {
    /** Flags could be any combination of the following values, divided by the "|" character: "up", "broadcast", "loopback", "pointtopoint" and "multicast". */
    flags: string;
    /** The IP address of the gateway. */
    gateway_ip: string;
    hardware_address: string;
    /** The addresses of the interface of v4 family. */
    ipv4_addresses: string[];
    /** The addresses of the interface of v6 family. */
    ipv6_addresses: string[];
    name: string;
}
