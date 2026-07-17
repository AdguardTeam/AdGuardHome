import type { SafeSearchConfig } from './safeSearchConfig';
import type { Schedule } from './schedule';

/**
 * Client information.
 */
export interface Client {
    /** Name */
    name?: string;
    /** IP, CIDR, MAC, or ClientID. */
    ids?: string[];
    use_global_settings?: boolean;
    filtering_enabled?: boolean;
    parental_enabled?: boolean;
    safebrowsing_enabled?: boolean;
    /** @deprecated */
    safesearch_enabled?: boolean;
    safe_search?: SafeSearchConfig;
    use_global_blocked_services?: boolean;
    blocked_services_schedule?: Schedule;
    blocked_services?: string[];
    upstreams?: string[];
    tags?: string[];
    /**
     * NOTE: If `ignore_querylog` is not set in HTTP API `GET /clients/add`
     * request then default value (false) will be used.
     *
     * If `ignore_querylog` is not set in HTTP API `GET /clients/update`
     * request then the existing value will not be changed.
     *
     * This behaviour can be changed in the future versions.
     */
    ignore_querylog?: boolean;
    /**
     * NOTE: If `ignore_statistics` is not set in HTTP API `GET
     * /clients/add` request then default value (false) will be used.
     *
     * If `ignore_statistics` is not set in HTTP API `GET /clients/update`
     * request then the existing value will not be changed.
     *
     * This behaviour can be changed in the future versions.
     */
    ignore_statistics?: boolean;
    /**
     * NOTE: If `upstreams_cache_enabled` is not set in HTTP API
     * `GET /clients/add` request then default value (false) will be used.
     *
     * If `upstreams_cache_enabled` is not set in HTTP API
     * `GET /clients/update` request then the existing value will not be
     * changed.
     *
     * This behaviour can be changed in the future versions.
     */
    upstreams_cache_enabled?: boolean;
    /**
     * NOTE: If `upstreams_cache_enabled` is not set in HTTP API
     * `GET /clients/update` request then the existing value will not be
     * changed.
     *
     * This behaviour can be changed in the future versions.
     */
    upstreams_cache_size?: number;
}
