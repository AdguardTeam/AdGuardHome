export type FilteringCheckHostParams = {
    /**
     * Filter by host name
     */
    name: string;
    /**
     * Optional ClientID or client IP address
     */
    client?: string;
    /**
     * Optional DNS type
     */
    qtype?: string;
};
