import type { NetInterfaces } from './netInterfaces';

/**
 * AdGuard Home addresses configuration
 */
export interface AddressesInfo {
    dns_port: number;
    interfaces: NetInterfaces;
    version: string;
    web_port: number;
}
