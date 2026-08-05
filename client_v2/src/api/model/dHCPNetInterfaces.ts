import type { DHCPNetInterface } from './dHCPNetInterface';

/**
 * DHCP network interfaces dictionary, keys are interface names.
 */
export interface DHCPNetInterfaces {
    [key: string]: DHCPNetInterface;
}
