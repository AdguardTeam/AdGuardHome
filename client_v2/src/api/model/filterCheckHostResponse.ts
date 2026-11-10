import type { FilteringReason } from './filteringReason';
import type { ResultRule } from './resultRule';

/**
 * Check Host Result
 */
export interface FilterCheckHostResponse {
    reason?: FilteringReason;
    /** Applied rules. */
    rules?: ResultRule[];
    /** Set if reason=FilteredBlockedService */
    service_name?: string;
    /** Set if reason=Rewrite */
    cname?: string;
    /** Set if reason=Rewrite */
    ip_addrs?: string[];
}
