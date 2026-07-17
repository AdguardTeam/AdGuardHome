import type { Lang } from './lang';

/**
 * AdGuard Home server status and configuration
 */
export interface ServerStatus {
    dns_addresses: string[];
    /**
     * @minimum 1
     * @maximum 65535
     */
    dns_port: number;
    /**
     * @minimum 1
     * @maximum 65535
     */
    http_port: number;
    protection_enabled: boolean;
    protection_disabled_duration?: number;
    dhcp_available?: boolean;
    running: boolean;
    version: string;
    language: Lang;
    /** Start time of the web API server (Unix time in milliseconds). */
    start_time?: number;
}
