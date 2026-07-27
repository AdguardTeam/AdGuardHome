import type { NetInterface } from './netInterface';

/**
 * Network interfaces dictionary, keys are interface names.
 */
export interface NetInterfaces {
    [key: string]: NetInterface;
}
