/**
 * Rewrite rule
 */
export interface RewriteEntry {
    /** Domain name */
    domain?: string;
    /** value of A, AAAA or CNAME DNS record */
    answer?: string;
    /** Optional. If omitted on add, defaults to `true`. On update, omitted preserves previous value. */
    enabled?: boolean;
}
