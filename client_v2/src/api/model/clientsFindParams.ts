export type ClientsFindParams = {
    /**
     * Filter by IP address or ClientIDs.  Parameters with names `ip1`, `ip2`, and so on are also accepted and interpreted as "ip0 OR ip1 OR ip2".
     * TODO(a.garipov): Replace with a better query API.
     */
    ip0?: string;
};
