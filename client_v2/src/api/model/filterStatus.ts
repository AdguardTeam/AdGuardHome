import type { Filter } from './filter';

/**
 * Filtering settings
 */
export interface FilterStatus {
    enabled?: boolean;
    interval?: number;
    filters?: Filter[];
    whitelist_filters?: Filter[];
    user_rules?: string[];
}
