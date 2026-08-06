import type { FilteringReason } from './filteringReason';
import type { QueryLogResponseStatus } from './queryLogResponseStatus';

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
     * Deprecated: Use 'reason' parameter instead Filter by response status
     * NOTE: This parameter cannot be used with 'reason' parameter.
     */
    response_status?: QueryLogResponseStatus;
    /**
     * Filter by response filtering reason.  Multiple reasons can be provided.
     * NOTE: This parameter cannot be used with 'response_status' parameter.
     */
    reason?: FilteringReason[];
};
