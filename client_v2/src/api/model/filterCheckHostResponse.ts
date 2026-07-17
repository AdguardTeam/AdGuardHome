import type { FilteringReason } from './filteringReason';
import type { ResultRule } from './resultRule';

/**
 * Check Host Result
 */
export interface FilterCheckHostResponse {
    reason?: FilteringReason;
    /**
     * In case if there's a rule applied to this DNS request, this is ID of the filter list that the rule belongs to.
     * Deprecated: use `rules[*].filter_list_id` instead.
     * @deprecated
     */
    filter_id?: number;
    /**
     * Filtering rule applied to the request (if any).
     * Deprecated: use `rules[*].text` instead.
     * @deprecated
     */
    rule?: string;
    /** Applied rules. */
    rules?: ResultRule[];
    /** Set if reason=FilteredBlockedService */
    service_name?: string;
    /** Set if reason=Rewrite */
    cname?: string;
    /** Set if reason=Rewrite */
    ip_addrs?: string[];
}
