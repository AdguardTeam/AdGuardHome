export interface DhcpConfigV4 {
    gateway_ip: string;
    subnet_mask: string;
    range_start: string;
    range_end: string;
    lease_duration: number;
    /** Custom DHCPv4 option strings.  Each string has the form `code type value`; see the technical documentation for the supported types. When updating the configuration, omit this property to preserve existing options, or send an empty array to clear them.  Custom options can override built-in DHCP settings, including the DNS server list. */
    options?: string[];
}
