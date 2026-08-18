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
    /**
     * Whether to use the filter lists in `filter_list_ids` and
     * `allow_filter_list_ids` instead of the globally enabled ones.  Rules
     * from the user's own list apply either way.
     *
     * NOTE: If `use_own_filter_lists` is not set in the
     * `POST /clients/add` request then the default value (false) will be
     * used, and the client uses the globally enabled filter lists.
     *
     * If it is not set in the `POST /clients/update` request then the
     * existing value will not be changed.
     */
    use_own_filter_lists?: boolean;
    /**
     * IDs of the blocking filter lists of the client.  They are only used
     * if `use_own_filter_lists` is `true`.
     *
     * NOTE: If `filter_list_ids` is not set in the `POST /clients/update`
     * request then the existing value will not be changed.  Set it to an
     * empty array to clear it.
     */
    filter_list_ids?: number[];
    /**
     * IDs of the allowing filter lists of the client.  They are only used
     * if `use_own_filter_lists` is `true`.
     *
     * NOTE: If `allow_filter_list_ids` is not set in the
     * `POST /clients/update` request then the existing value will not be
     * changed.  Set it to an empty array to clear it.
     */
    allow_filter_list_ids?: number[];
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
