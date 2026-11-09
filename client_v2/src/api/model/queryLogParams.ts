import type { FilteringReason } from './filteringReason';

export type QueryLogParams = {
    /**
     * Filter by older than
     */
    older_than?: string;
    /**
     * Specify the ranking number of the first item on the page.  Even though it is possible to use "offset" and "older_than", we recommend choosing one of them and sticking to it.
     */
    offset?: number;
    /**
     * Limit the number of records to be returned
     */
    limit?: number;
    /**
     * Filter by domain name or client IP
     */
    search?: string;
    /**
     * Filter by response filtering reason.  Multiple reasons can be provided.
     */
    reason?: FilteringReason[];
};
