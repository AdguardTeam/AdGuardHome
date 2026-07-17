import type { FilterSetUrlData } from './filterSetUrlData';

/**
 * Filtering URL settings
 */
export interface FilterSetUrl {
    data?: FilterSetUrlData;
    url?: string;
    whitelist?: boolean;
}
