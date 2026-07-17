export type MobileConfigDoHParams = {
    /**
     * Host for which the config is generated.  If no host is provided, `tls.server_name` from the configuration file is used.  If `tls.server_name` is not set, the API returns an error with a 500 status.
     */
    host: string;
    /**
     * ClientID.
     */
    client_id?: string;
};
