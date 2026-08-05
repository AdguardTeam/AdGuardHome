import type { AddressInfo } from './addressInfo';
import type { Lang } from './lang';

/**
 * AdGuard Home initial configuration for the first-install wizard.
 */
export interface InitialConfiguration {
    dns: AddressInfo;
    web: AddressInfo;
    language?: Lang;
    /** Basic auth password */
    password: string;
    /** Basic auth username */
    username: string;
}
