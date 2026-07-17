import type { WhoisInfo } from './whoisInfo';

/**
 * Auto-Client information
 */
export interface ClientAuto {
    /** IP address */
    ip?: string;
    /** Name */
    name?: string;
    /** The source of this information */
    source?: string;
    whois_info?: WhoisInfo;
}
