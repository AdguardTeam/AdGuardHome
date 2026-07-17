export interface DhcpConfigV4 {
    gateway_ip?: string;
    subnet_mask?: string;
    range_start?: string;
    range_end?: string;
    lease_duration?: number;
}
