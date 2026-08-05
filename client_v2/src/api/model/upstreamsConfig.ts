/**
 * Upstream configuration to be tested
 */
export interface UpstreamsConfig {
    /** Bootstrap DNS servers, port is optional after colon. */
    bootstrap_dns: string[];
    /** Upstream DNS servers, port is optional after colon. */
    upstream_dns: string[];
    /** Fallback DNS servers, port is optional after colon. */
    fallback_dns?: string[];
    /** Local PTR resolvers, port is optional after colon. */
    private_upstream?: string[];
}
