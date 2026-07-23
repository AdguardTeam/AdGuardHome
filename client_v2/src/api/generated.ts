import type {
    AccessList,
    AddUrlRequest,
    AddressesInfo,
    BlockedServicesAll,
    BlockedServicesArray,
    BlockedServicesSchedule,
    CheckConfigRequest,
    CheckConfigResponse,
    Client,
    ClientDelete,
    ClientUpdate,
    Clients,
    ClientsFindParams,
    ClientsFindResponse,
    ClientsSearchRequest,
    DHCPNetInterfaces,
    DNSConfig,
    DhcpConfig,
    DhcpFindActiveReq,
    DhcpSearchResult,
    DhcpStaticLeaseBody,
    DhcpStatus,
    DnsInfo200,
    FilterCheckHostResponse,
    FilterConfig,
    FilterRefreshRequest,
    FilterRefreshResponse,
    FilterSetUrl,
    FilterStatus,
    FilteringCheckHostParams,
    GetQueryLogConfigResponse,
    GetStatsConfigResponse,
    GetVersionRequest,
    InitialConfiguration,
    LanguageSettings,
    Login,
    MobileConfigDoHParams,
    MobileConfigDoTParams,
    ParentalStatus200,
    ProfileInfo,
    QueryLog,
    QueryLogConfig,
    QueryLogParams,
    RemoveUrlRequest,
    RewriteEntryBody,
    RewriteList,
    RewriteSettings,
    RewriteSettingsBody,
    RewriteUpdateBody,
    SafeSearchConfig,
    SafebrowsingStatus200,
    ServerStatus,
    SetProtectionRequest,
    SetRulesRequest,
    Stats,
    StatsConfig,
    StatsParams,
    TlsConfig,
    TlsConfigBody,
    UpstreamsConfig,
    UpstreamsConfigResponse,
    VersionInfo,
} from './model';

import { customFetch } from './customFetch';
export const getStatusUrl = () => {
    return `control/status`;
};

/**
 * @summary Get DNS server current status and general settings
 */
export const status = async (options?: RequestInit): Promise<ServerStatus> => {
    return customFetch<ServerStatus>(getStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getDnsInfoUrl = () => {
    return `control/dns_info`;
};

/**
 * @summary Get general DNS parameters
 */
export const dnsInfo = async (options?: RequestInit): Promise<DnsInfo200> => {
    return customFetch<DnsInfo200>(getDnsInfoUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getDnsConfigUrl = () => {
    return `control/dns_config`;
};

/**
 * @summary Set general DNS parameters
 */
export const dnsConfig = async (dNSConfig?: DNSConfig, options?: RequestInit): Promise<void> => {
    return customFetch<void>(getDnsConfigUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(dNSConfig),
    });
};

export const getSetProtectionUrl = () => {
    return `control/protection`;
};

/**
 * @summary Set protection state and duration
 */
export const setProtection = async (
    setProtectionRequest?: SetProtectionRequest,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getSetProtectionUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(setProtectionRequest),
    });
};

export const getCacheClearUrl = () => {
    return `control/cache_clear`;
};

/**
 * @summary Clear DNS cache
 */
export const cacheClear = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getCacheClearUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getTestUpstreamDNSUrl = () => {
    return `control/test_upstream_dns`;
};

/**
 * @summary Test upstream configuration
 */
export const testUpstreamDNS = async (
    upstreamsConfig?: UpstreamsConfig,
    options?: RequestInit,
): Promise<UpstreamsConfigResponse> => {
    return customFetch<UpstreamsConfigResponse>(getTestUpstreamDNSUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(upstreamsConfig),
    });
};

export const getGetVersionJsonUrl = () => {
    return `control/version.json`;
};

/**
 * @summary Gets information about the latest available version of AdGuard

 */
export const getVersionJson = async (
    getVersionRequest: GetVersionRequest,
    options?: RequestInit,
): Promise<VersionInfo> => {
    return customFetch<VersionInfo>(getGetVersionJsonUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(getVersionRequest),
    });
};

export const getBeginUpdateUrl = () => {
    return `control/update`;
};

/**
 * @summary Begin auto-upgrade procedure
 */
export const beginUpdate = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getBeginUpdateUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getQueryLogUrl = (params?: QueryLogParams) => {
    const normalizedParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        const explodeParameters = ['reason'];

        if (Array.isArray(value) && explodeParameters.includes(key)) {
            value.forEach((v) => {
                normalizedParams.append(key, v === null ? 'null' : String(v));
            });
            return;
        }

        if (value !== undefined) {
            normalizedParams.append(key, value === null ? 'null' : String(value));
        }
    });

    const stringifiedParams = normalizedParams.toString();

    return stringifiedParams.length > 0
        ? `control/querylog?${stringifiedParams}`
        : `control/querylog`;
};

/**
 * @summary Get DNS server query log.
 */
export const queryLog = async (
    params?: QueryLogParams,
    options?: RequestInit,
): Promise<QueryLog> => {
    return customFetch<QueryLog>(getQueryLogUrl(params), {
        ...options,
        method: 'GET',
    });
};

export const getQueryLogInfoUrl = () => {
    return `control/querylog_info`;
};

/**
 * Deprecated: Use `GET /querylog/config` instead.
 *
 * NOTE: If `interval` was configured by editing configuration file or new
 * HTTP API call `PUT /querylog/config/update` and it's not equal to
 * previous allowed enum values then it will be equal to `90` days for
 * compatibility reasons.
 * @deprecated
 * @summary Get query log parameters
 */
export const queryLogInfo = async (options?: RequestInit): Promise<QueryLogConfig> => {
    return customFetch<QueryLogConfig>(getQueryLogInfoUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getQueryLogConfigUrl = () => {
    return `control/querylog_config`;
};

/**
 * Deprecated: Use `PUT /querylog/config/update` instead.
 * @deprecated
 * @summary Set query log parameters
 */
export const queryLogConfig = async (
    queryLogConfig?: QueryLogConfig,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getQueryLogConfigUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(queryLogConfig),
    });
};

export const getQuerylogClearUrl = () => {
    return `control/querylog_clear`;
};

/**
 * @summary Clear query log
 */
export const querylogClear = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getQuerylogClearUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getGetQueryLogConfigUrl = () => {
    return `control/querylog/config`;
};

/**
 * @summary Get query log parameters
 */
export const getQueryLogConfig = async (
    options?: RequestInit,
): Promise<GetQueryLogConfigResponse> => {
    return customFetch<GetQueryLogConfigResponse>(getGetQueryLogConfigUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getPutQueryLogConfigUrl = () => {
    return `control/querylog/config/update`;
};

/**
 * @summary Set query log parameters
 */
export const putQueryLogConfig = async (
    getQueryLogConfigResponse: GetQueryLogConfigResponse,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getPutQueryLogConfigUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(getQueryLogConfigResponse),
    });
};

export const getStatsUrl = (params?: StatsParams) => {
    const normalizedParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        if (value !== undefined) {
            normalizedParams.append(key, value === null ? 'null' : String(value));
        }
    });

    const stringifiedParams = normalizedParams.toString();

    return stringifiedParams.length > 0 ? `control/stats?${stringifiedParams}` : `control/stats`;
};

/**
 * @summary Get DNS server statistics
 */
export const stats = async (params?: StatsParams, options?: RequestInit): Promise<Stats> => {
    return customFetch<Stats>(getStatsUrl(params), {
        ...options,
        method: 'GET',
    });
};

export const getStatsResetUrl = () => {
    return `control/stats_reset`;
};

/**
 * @summary Reset all statistics to zeroes
 */
export const statsReset = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getStatsResetUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getStatsInfoUrl = () => {
    return `control/stats_info`;
};

/**
 * Deprecated: Use `GET /stats/config` instead.
 *
 * NOTE: If `interval` was configured by editing configuration file or new
 * HTTP API call `PUT /stats/config/update` and it's not equal to
 * previous allowed enum values then it will be equal to `90` days for
 * compatibility reasons.
 * @deprecated
 * @summary Get statistics parameters
 */
export const statsInfo = async (options?: RequestInit): Promise<StatsConfig> => {
    return customFetch<StatsConfig>(getStatsInfoUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getStatsConfigUrl = () => {
    return `control/stats_config`;
};

/**
 * Deprecated: Use `PUT /stats/config/update` instead.
 * @deprecated
 * @summary Set statistics parameters
 */
export const statsConfig = async (
    statsConfig?: StatsConfig,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getStatsConfigUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(statsConfig),
    });
};

export const getGetStatsConfigUrl = () => {
    return `control/stats/config`;
};

/**
 * @summary Get statistics parameters
 */
export const getStatsConfig = async (options?: RequestInit): Promise<GetStatsConfigResponse> => {
    return customFetch<GetStatsConfigResponse>(getGetStatsConfigUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getPutStatsConfigUrl = () => {
    return `control/stats/config/update`;
};

/**
 * @summary Set statistics parameters
 */
export const putStatsConfig = async (
    getStatsConfigResponse: GetStatsConfigResponse,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getPutStatsConfigUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(getStatsConfigResponse),
    });
};

export const getTlsStatusUrl = () => {
    return `control/tls/status`;
};

/**
 * @summary Returns TLS configuration and its status
 */
export const tlsStatus = async (options?: RequestInit): Promise<TlsConfig> => {
    return customFetch<TlsConfig>(getTlsStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getTlsConfigureUrl = () => {
    return `control/tls/configure`;
};

/**
 * @summary Updates current TLS configuration
 */
export const tlsConfigure = async (
    tlsConfigBody: TlsConfigBody,
    options?: RequestInit,
): Promise<TlsConfig> => {
    return customFetch<TlsConfig>(getTlsConfigureUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(tlsConfigBody),
    });
};

export const getTlsValidateUrl = () => {
    return `control/tls/validate`;
};

/**
 * @summary Checks if the current TLS configuration is valid
 */
export const tlsValidate = async (
    tlsConfigBody: TlsConfigBody,
    options?: RequestInit,
): Promise<TlsConfig> => {
    return customFetch<TlsConfig>(getTlsValidateUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(tlsConfigBody),
    });
};

export const getDhcpStatusUrl = () => {
    return `control/dhcp/status`;
};

/**
 * @summary Gets the current DHCP settings and status
 */
export const dhcpStatus = async (options?: RequestInit): Promise<DhcpStatus> => {
    return customFetch<DhcpStatus>(getDhcpStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getDhcpInterfacesUrl = () => {
    return `control/dhcp/interfaces`;
};

/**
 * @summary Gets the available interfaces
 */
export const dhcpInterfaces = async (options?: RequestInit): Promise<DHCPNetInterfaces> => {
    return customFetch<DHCPNetInterfaces>(getDhcpInterfacesUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getDhcpSetConfigUrl = () => {
    return `control/dhcp/set_config`;
};

/**
 * @summary Updates the current DHCP server configuration
 */
export const dhcpSetConfig = async (
    dhcpConfig?: DhcpConfig,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getDhcpSetConfigUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(dhcpConfig),
    });
};

export const getCheckActiveDhcpUrl = () => {
    return `control/dhcp/find_active_dhcp`;
};

/**
 * @summary Searches for an active DHCP server on the network
 */
export const checkActiveDhcp = async (
    dhcpFindActiveReq?: DhcpFindActiveReq,
    options?: RequestInit,
): Promise<DhcpSearchResult> => {
    return customFetch<DhcpSearchResult>(getCheckActiveDhcpUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(dhcpFindActiveReq),
    });
};

export const getDhcpAddStaticLeaseUrl = () => {
    return `control/dhcp/add_static_lease`;
};

/**
 * @summary Adds a static lease
 */
export const dhcpAddStaticLease = async (
    dhcpStaticLeaseBody: DhcpStaticLeaseBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getDhcpAddStaticLeaseUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(dhcpStaticLeaseBody),
    });
};

export const getDhcpRemoveStaticLeaseUrl = () => {
    return `control/dhcp/remove_static_lease`;
};

/**
 * @summary Removes a static lease
 */
export const dhcpRemoveStaticLease = async (
    dhcpStaticLeaseBody: DhcpStaticLeaseBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getDhcpRemoveStaticLeaseUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(dhcpStaticLeaseBody),
    });
};

export const getDhcpUpdateStaticLeaseUrl = () => {
    return `control/dhcp/update_static_lease`;
};

/**
 * Updates IP address, hostname of the static lease.  IP version must be the same as previous.
 * @summary Updates a static lease
 */
export const dhcpUpdateStaticLease = async (
    dhcpStaticLeaseBody: DhcpStaticLeaseBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getDhcpUpdateStaticLeaseUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(dhcpStaticLeaseBody),
    });
};

export const getDhcpResetUrl = () => {
    return `control/dhcp/reset`;
};

/**
 * @summary Reset DHCP configuration
 */
export const dhcpReset = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getDhcpResetUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getDhcpResetLeasesUrl = () => {
    return `control/dhcp/reset_leases`;
};

/**
 * @summary Reset DHCP leases
 */
export const dhcpResetLeases = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getDhcpResetLeasesUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getFilteringStatusUrl = () => {
    return `control/filtering/status`;
};

/**
 * @summary Get filtering parameters
 */
export const filteringStatus = async (options?: RequestInit): Promise<FilterStatus> => {
    return customFetch<FilterStatus>(getFilteringStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getFilteringConfigUrl = () => {
    return `control/filtering/config`;
};

/**
 * @summary Set filtering parameters
 */
export const filteringConfig = async (
    filterConfig: FilterConfig,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getFilteringConfigUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(filterConfig),
    });
};

export const getFilteringAddURLUrl = () => {
    return `control/filtering/add_url`;
};

/**
 * @summary Add filter URL or an absolute file path
 */
export const filteringAddURL = async (
    addUrlRequest: AddUrlRequest,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getFilteringAddURLUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(addUrlRequest),
    });
};

export const getFilteringRemoveURLUrl = () => {
    return `control/filtering/remove_url`;
};

/**
 * @summary Remove filter URL
 */
export const filteringRemoveURL = async (
    removeUrlRequest: RemoveUrlRequest,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getFilteringRemoveURLUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(removeUrlRequest),
    });
};

export const getFilteringSetURLUrl = () => {
    return `control/filtering/set_url`;
};

/**
 * @summary Set URL parameters
 */
export const filteringSetURL = async (
    filterSetUrl?: FilterSetUrl,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getFilteringSetURLUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(filterSetUrl),
    });
};

export const getFilteringRefreshUrl = () => {
    return `control/filtering/refresh`;
};

/**
 * @summary Reload filtering rules from URLs.  This might be needed if new URL was just added and you don't want to wait for automatic refresh to kick in. This API request is ratelimited, so you can call it freely as often as you like, it wont create unnecessary burden on servers that host the URL.  This should work as intended, a `force` parameter is offered as last-resort attempt to make filter lists fresh.  If you ever find yourself using `force` to make something work that otherwise wont, this is a bug and report it accordingly.

 */
export const filteringRefresh = async (
    filterRefreshRequest?: FilterRefreshRequest,
    options?: RequestInit,
): Promise<FilterRefreshResponse> => {
    return customFetch<FilterRefreshResponse>(getFilteringRefreshUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(filterRefreshRequest),
    });
};

export const getFilteringSetRulesUrl = () => {
    return `control/filtering/set_rules`;
};

/**
 * @summary Set user-defined filter rules
 */
export const filteringSetRules = async (
    setRulesRequest?: SetRulesRequest,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getFilteringSetRulesUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(setRulesRequest),
    });
};

export const getFilteringCheckHostUrl = (params: FilteringCheckHostParams) => {
    const normalizedParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        if (value !== undefined) {
            normalizedParams.append(key, value === null ? 'null' : String(value));
        }
    });

    const stringifiedParams = normalizedParams.toString();

    return stringifiedParams.length > 0
        ? `control/filtering/check_host?${stringifiedParams}`
        : `control/filtering/check_host`;
};

/**
 * @summary Check if host name is filtered
 */
export const filteringCheckHost = async (
    params: FilteringCheckHostParams,
    options?: RequestInit,
): Promise<FilterCheckHostResponse> => {
    return customFetch<FilterCheckHostResponse>(getFilteringCheckHostUrl(params), {
        ...options,
        method: 'GET',
    });
};

export const getSafebrowsingEnableUrl = () => {
    return `control/safebrowsing/enable`;
};

/**
 * @summary Enable safebrowsing
 */
export const safebrowsingEnable = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getSafebrowsingEnableUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getSafebrowsingDisableUrl = () => {
    return `control/safebrowsing/disable`;
};

/**
 * @summary Disable safebrowsing
 */
export const safebrowsingDisable = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getSafebrowsingDisableUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getSafebrowsingStatusUrl = () => {
    return `control/safebrowsing/status`;
};

/**
 * @summary Get safebrowsing status
 */
export const safebrowsingStatus = async (options?: RequestInit): Promise<SafebrowsingStatus200> => {
    return customFetch<SafebrowsingStatus200>(getSafebrowsingStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getParentalEnableUrl = () => {
    return `control/parental/enable`;
};

/**
 * @summary Enable parental filtering
 */
export const parentalEnable = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getParentalEnableUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getParentalDisableUrl = () => {
    return `control/parental/disable`;
};

/**
 * @summary Disable parental filtering
 */
export const parentalDisable = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getParentalDisableUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getParentalStatusUrl = () => {
    return `control/parental/status`;
};

/**
 * @summary Get parental filtering status
 */
export const parentalStatus = async (options?: RequestInit): Promise<ParentalStatus200> => {
    return customFetch<ParentalStatus200>(getParentalStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getSafesearchEnableUrl = () => {
    return `control/safesearch/enable`;
};

/**
 * @deprecated
 * @summary Enable safesearch
 */
export const safesearchEnable = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getSafesearchEnableUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getSafesearchDisableUrl = () => {
    return `control/safesearch/disable`;
};

/**
 * @deprecated
 * @summary Disable safesearch
 */
export const safesearchDisable = async (options?: RequestInit): Promise<void> => {
    return customFetch<void>(getSafesearchDisableUrl(), {
        ...options,
        method: 'POST',
    });
};

export const getSafesearchSettingsUrl = () => {
    return `control/safesearch/settings`;
};

/**
 * @summary Update safesearch settings
 */
export const safesearchSettings = async (
    safeSearchConfig?: SafeSearchConfig,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getSafesearchSettingsUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(safeSearchConfig),
    });
};

export const getSafesearchStatusUrl = () => {
    return `control/safesearch/status`;
};

/**
 * @summary Get safesearch status
 */
export const safesearchStatus = async (options?: RequestInit): Promise<SafeSearchConfig> => {
    return customFetch<SafeSearchConfig>(getSafesearchStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getClientsStatusUrl = () => {
    return `control/clients`;
};

/**
 * @summary Get information about configured clients
 */
export const clientsStatus = async (options?: RequestInit): Promise<Clients> => {
    return customFetch<Clients>(getClientsStatusUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getClientsAddUrl = () => {
    return `control/clients/add`;
};

/**
 * @summary Add a new client
 */
export const clientsAdd = async (client: Client, options?: RequestInit): Promise<void> => {
    return customFetch<void>(getClientsAddUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(client),
    });
};

export const getClientsDeleteUrl = () => {
    return `control/clients/delete`;
};

/**
 * @summary Remove a client
 */
export const clientsDelete = async (
    clientDelete: ClientDelete,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getClientsDeleteUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(clientDelete),
    });
};

export const getClientsUpdateUrl = () => {
    return `control/clients/update`;
};

/**
 * @summary Update client information
 */
export const clientsUpdate = async (
    clientUpdate: ClientUpdate,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getClientsUpdateUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(clientUpdate),
    });
};

export const getClientsFindUrl = (params?: ClientsFindParams) => {
    const normalizedParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        if (value !== undefined) {
            normalizedParams.append(key, value === null ? 'null' : String(value));
        }
    });

    const stringifiedParams = normalizedParams.toString();

    return stringifiedParams.length > 0
        ? `control/clients/find?${stringifiedParams}`
        : `control/clients/find`;
};

/**
 * Deprecated: Use `POST /clients/search` instead.
 * @deprecated
 * @summary Get information about clients by their IP addresses or ClientIDs.

 */
export const clientsFind = async (
    params?: ClientsFindParams,
    options?: RequestInit,
): Promise<ClientsFindResponse> => {
    return customFetch<ClientsFindResponse>(getClientsFindUrl(params), {
        ...options,
        method: 'GET',
    });
};

export const getClientsSearchUrl = () => {
    return `control/clients/search`;
};

/**
 * @summary Retrieve information about clients by performing an exact match search using IP addresses, CIDRs, MAC addresses, or ClientIDs.

 */
export const clientsSearch = async (
    clientsSearchRequest: ClientsSearchRequest,
    options?: RequestInit,
): Promise<ClientsFindResponse> => {
    return customFetch<ClientsFindResponse>(getClientsSearchUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(clientsSearchRequest),
    });
};

export const getAccessListUrl = () => {
    return `control/access/list`;
};

/**
 * @summary List (dis)allowed clients, blocked hosts, etc.
 */
export const accessList = async (options?: RequestInit): Promise<AccessList> => {
    return customFetch<AccessList>(getAccessListUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getAccessSetUrl = () => {
    return `control/access/set`;
};

/**
 * @summary Set (dis)allowed clients, blocked hosts, etc.
 */
export const accessSet = async (accessList: AccessList, options?: RequestInit): Promise<void> => {
    return customFetch<void>(getAccessSetUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(accessList),
    });
};

export const getBlockedServicesAvailableServicesUrl = () => {
    return `control/blocked_services/services`;
};

/**
 * Deprecated: Use `GET /blocked_services/all` instead.
 * @deprecated
 * @summary Get available services to use for blocking
 */
export const blockedServicesAvailableServices = async (
    options?: RequestInit,
): Promise<BlockedServicesArray> => {
    return customFetch<BlockedServicesArray>(getBlockedServicesAvailableServicesUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getBlockedServicesAllUrl = () => {
    return `control/blocked_services/all`;
};

/**
 * @summary Get available services to use for blocking
 */
export const blockedServicesAll = async (options?: RequestInit): Promise<BlockedServicesAll> => {
    return customFetch<BlockedServicesAll>(getBlockedServicesAllUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getBlockedServicesListUrl = () => {
    return `control/blocked_services/list`;
};

/**
 * Deprecated: Use `GET /blocked_services/get` instead.
 * @deprecated
 * @summary Get blocked services list
 */
export const blockedServicesList = async (options?: RequestInit): Promise<BlockedServicesArray> => {
    return customFetch<BlockedServicesArray>(getBlockedServicesListUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getBlockedServicesSetUrl = () => {
    return `control/blocked_services/set`;
};

/**
 * Deprecated: Use `PUT /blocked_services/update` instead.
 * @deprecated
 * @summary Set blocked services list
 */
export const blockedServicesSet = async (
    blockedServicesArray?: BlockedServicesArray,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getBlockedServicesSetUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(blockedServicesArray),
    });
};

export const getBlockedServicesScheduleUrl = () => {
    return `control/blocked_services/get`;
};

/**
 * @summary Get blocked services
 */
export const blockedServicesSchedule = async (
    options?: RequestInit,
): Promise<BlockedServicesSchedule> => {
    return customFetch<BlockedServicesSchedule>(getBlockedServicesScheduleUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getBlockedServicesScheduleUpdateUrl = () => {
    return `control/blocked_services/update`;
};

/**
 * @summary Update blocked services
 */
export const blockedServicesScheduleUpdate = async (
    blockedServicesSchedule: BlockedServicesSchedule,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getBlockedServicesScheduleUpdateUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(blockedServicesSchedule),
    });
};

export const getRewriteListUrl = () => {
    return `control/rewrite/list`;
};

/**
 * @summary Get list of Rewrite rules
 */
export const rewriteList = async (options?: RequestInit): Promise<RewriteList> => {
    return customFetch<RewriteList>(getRewriteListUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getRewriteAddUrl = () => {
    return `control/rewrite/add`;
};

/**
 * @summary Add a new Rewrite rule
 */
export const rewriteAdd = async (
    rewriteEntryBody: RewriteEntryBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getRewriteAddUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(rewriteEntryBody),
    });
};

export const getRewriteDeleteUrl = () => {
    return `control/rewrite/delete`;
};

/**
 * @summary Remove a Rewrite rule
 */
export const rewriteDelete = async (
    rewriteEntryBody: RewriteEntryBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getRewriteDeleteUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(rewriteEntryBody),
    });
};

export const getRewriteSettingsGetUrl = () => {
    return `control/rewrite/settings`;
};

/**
 * @summary Get rewrite settings
 */
export const rewriteSettingsGet = async (options?: RequestInit): Promise<RewriteSettings> => {
    return customFetch<RewriteSettings>(getRewriteSettingsGetUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getRewriteSettingsUpdateUrl = () => {
    return `control/rewrite/settings/update`;
};

/**
 * @summary Update rewrite settings
 */
export const rewriteSettingsUpdate = async (
    rewriteSettingsBody: RewriteSettingsBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getRewriteSettingsUpdateUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(rewriteSettingsBody),
    });
};

export const getRewriteUpdateUrl = () => {
    return `control/rewrite/update`;
};

/**
 * @summary Update a Rewrite rule
 */
export const rewriteUpdate = async (
    rewriteUpdateBody: RewriteUpdateBody,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getRewriteUpdateUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(rewriteUpdateBody),
    });
};

export const getChangeLanguageUrl = () => {
    return `control/i18n/change_language`;
};

/**
 * Deprecated: Use `PUT /control/profile` instead.
 * @deprecated
 * @summary Change current language.  Argument must be an ISO 639-1 two-letter code.

 */
export const changeLanguage = async (
    languageSettings?: LanguageSettings,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getChangeLanguageUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(languageSettings),
    });
};

export const getCurrentLanguageUrl = () => {
    return `control/i18n/current_language`;
};

/**
 * Deprecated: Use `GET /control/profile` instead.
 * @deprecated
 * @summary Get currently set language.  Result is ISO 639-1 two-letter code.  Empty result means default language.

 */
export const currentLanguage = async (options?: RequestInit): Promise<LanguageSettings> => {
    return customFetch<LanguageSettings>(getCurrentLanguageUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getInstallGetAddressesUrl = () => {
    return `control/install/get_addresses`;
};

/**
 * @summary Gets the network interfaces information.
 */
export const installGetAddresses = async (options?: RequestInit): Promise<AddressesInfo> => {
    return customFetch<AddressesInfo>(getInstallGetAddressesUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getInstallCheckConfigUrl = () => {
    return `control/install/check_config`;
};

/**
 * @summary Checks configuration
 */
export const installCheckConfig = async (
    checkConfigRequest: CheckConfigRequest,
    options?: RequestInit,
): Promise<CheckConfigResponse> => {
    return customFetch<CheckConfigResponse>(getInstallCheckConfigUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(checkConfigRequest),
    });
};

export const getInstallConfigureUrl = () => {
    return `control/install/configure`;
};

/**
 * @summary Applies the initial configuration.
 */
export const installConfigure = async (
    initialConfiguration: InitialConfiguration,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getInstallConfigureUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(initialConfiguration),
    });
};

export const getLoginUrl = () => {
    return `control/login`;
};

/**
 * @summary Perform administrator log-in
 */
export const login = async (login: Login, options?: RequestInit): Promise<void> => {
    return customFetch<void>(getLoginUrl(), {
        ...options,
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(login),
    });
};

export const getLogoutUrl = () => {
    return `control/logout`;
};

/**
 * @summary Perform administrator log-out
 */
export const logout = async (options?: RequestInit): Promise<unknown> => {
    return customFetch<unknown>(getLogoutUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getUpdateProfileUrl = () => {
    return `control/profile/update`;
};

/**
 * @summary Updates current user info
 */
export const updateProfile = async (
    profileInfo?: ProfileInfo,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getUpdateProfileUrl(), {
        ...options,
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...options?.headers },
        body: JSON.stringify(profileInfo),
    });
};

export const getGetProfileUrl = () => {
    return `control/profile`;
};

export const getProfile = async (options?: RequestInit): Promise<ProfileInfo> => {
    return customFetch<ProfileInfo>(getGetProfileUrl(), {
        ...options,
        method: 'GET',
    });
};

export const getMobileConfigDoHUrl = (params: MobileConfigDoHParams) => {
    const normalizedParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        if (value !== undefined) {
            normalizedParams.append(key, value === null ? 'null' : String(value));
        }
    });

    const stringifiedParams = normalizedParams.toString();

    return stringifiedParams.length > 0
        ? `control/apple/doh.mobileconfig?${stringifiedParams}`
        : `control/apple/doh.mobileconfig`;
};

/**
 * @summary Get DNS over HTTPS .mobileconfig.
 */
export const mobileConfigDoH = async (
    params: MobileConfigDoHParams,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getMobileConfigDoHUrl(params), {
        ...options,
        method: 'GET',
    });
};

export const getMobileConfigDoTUrl = (params: MobileConfigDoTParams) => {
    const normalizedParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        if (value !== undefined) {
            normalizedParams.append(key, value === null ? 'null' : String(value));
        }
    });

    const stringifiedParams = normalizedParams.toString();

    return stringifiedParams.length > 0
        ? `control/apple/dot.mobileconfig?${stringifiedParams}`
        : `control/apple/dot.mobileconfig`;
};

/**
 * @summary Get DNS over TLS .mobileconfig.
 */
export const mobileConfigDoT = async (
    params: MobileConfigDoTParams,
    options?: RequestInit,
): Promise<void> => {
    return customFetch<void>(getMobileConfigDoTUrl(params), {
        ...options,
        method: 'GET',
    });
};
