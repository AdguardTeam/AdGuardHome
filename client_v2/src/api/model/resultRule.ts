/**
 * Applied rule.
 */
export interface ResultRule {
    /** In case if there's a rule applied to this DNS request, this is ID of the filter list that the rule belongs to. */
    filter_list_id?: number;
    /** The text of the filtering rule applied to the request (if any). */
    text?: string;
}
