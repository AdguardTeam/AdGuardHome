import axios from 'axios';

import { BASE_URL } from '../../constants';

import { getPathWithQueryString } from '../helpers/helpers';
import { QUERY_LOGS_PAGE_LIMIT, HTML_PAGES, R_PATH_LAST_PART, THEMES } from '../helpers/constants';
import i18n from '../i18n';
import { LANGUAGES } from '../helpers/twosky';

class Api {
    baseUrl = BASE_URL;

    async makeRequest(path: any, method = 'POST', config: any = {}) {
        const url = `${this.baseUrl}/${path}`;

        const axiosConfig = config || {};
        if (method !== 'GET' && axiosConfig.data) {
            axiosConfig.headers = axiosConfig.headers || {};
            axiosConfig.headers['Content-Type'] = axiosConfig.headers['Content-Type'] || 'application/json';
        }

        try {
            const response = await axios({
                url,
                method,
                ...axiosConfig,
            });
            return response.data;
        } catch (error) {
            const errorPath = url;

            if (error.response) {
                const { pathname } = document.location;
                const shouldRedirect = pathname !== HTML_PAGES.LOGIN && pathname !== HTML_PAGES.INSTALL;

                if (error.response.status === 403 && shouldRedirect) {
                    const loginPageUrl = window.location.href.replace(R_PATH_LAST_PART, HTML_PAGES.LOGIN);
                    window.location.replace(loginPageUrl);
                    return false;
                }

                throw new Error(`${errorPath} | ${error.response.data} | ${error.response.status}`);
            }

            throw new Error(`${errorPath} | ${error.message || error}`);
        }
    }

    // Global methods
    GLOBAL_STATUS = { path: 'status', method: 'GET' };

    GLOBAL_TEST_UPSTREAM_DNS = { path: 'test_upstream_dns', method: 'POST' };

    GLOBAL_VERSION = { path: 'version.json', method: 'POST' };

    GLOBAL_UPDATE = { path: 'update', method: 'POST' };

    getGlobalStatus() {
        const { path, method } = this.GLOBAL_STATUS;

        return this.makeRequest(path, method);
    }

    testUpstream(servers: any) {
        const { path, method } = this.GLOBAL_TEST_UPSTREAM_DNS;
        const config = {
            data: servers,
        };
        return this.makeRequest(path, method, config);
    }

    getGlobalVersion(data: any) {
        const { path, method } = this.GLOBAL_VERSION;
        const config = {
            data,
        };
        return this.makeRequest(path, method, config);
    }

    getUpdate() {
        const { path, method } = this.GLOBAL_UPDATE;

        return this.makeRequest(path, method);
    }

    // Filtering
    FILTERING_STATUS = { path: 'filtering/status', method: 'GET' };

    FILTERING_ADD_FILTER = { path: 'filtering/add_url', method: 'POST' };

    FILTERING_REMOVE_FILTER = { path: 'filtering/remove_url', method: 'POST' };

    FILTERING_SET_RULES = { path: 'filtering/set_rules', method: 'POST' };

    FILTERING_REFRESH = { path: 'filtering/refresh', method: 'POST' };

    FILTERING_SET_URL = { path: 'filtering/set_url', method: 'POST' };

    FILTERING_CONFIG = { path: 'filtering/config', method: 'POST' };

    FILTERING_CHECK_HOST = { path: 'filtering/check_host', method: 'GET' };

    getFilteringStatus() {
        const { path, method } = this.FILTERING_STATUS;

        return this.makeRequest(path, method);
    }

    refreshFilters(config: any) {
        const { path, method } = this.FILTERING_REFRESH;
        const parameters = {
            data: config,
        };

        return this.makeRequest(path, method, parameters);
    }

    addFilter(config: any) {
        const { path, method } = this.FILTERING_ADD_FILTER;
        const parameters = {
            data: config,
        };

        return this.makeRequest(path, method, parameters);
    }

    removeFilter(config: any) {
        const { path, method } = this.FILTERING_REMOVE_FILTER;
        const parameters = {
            data: config,
        };

        return this.makeRequest(path, method, parameters);
    }

    setRules(rules: any) {
        const { path, method } = this.FILTERING_SET_RULES;
        const parameters = {
            data: rules,
        };
        return this.makeRequest(path, method, parameters);
    }

    setFiltersConfig(config: any) {
        const { path, method } = this.FILTERING_CONFIG;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    setFilterUrl(config: any) {
        const { path, method } = this.FILTERING_SET_URL;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    checkHost(params: any) {
        const { path, method } = this.FILTERING_CHECK_HOST;
        const url = getPathWithQueryString(path, params);

        return this.makeRequest(url, method);
    }

    // Parental
    PARENTAL_STATUS = { path: 'parental/status', method: 'GET' };

    PARENTAL_ENABLE = { path: 'parental/enable', method: 'POST' };

    PARENTAL_DISABLE = { path: 'parental/disable', method: 'POST' };

    getParentalStatus() {
        const { path, method } = this.PARENTAL_STATUS;

        return this.makeRequest(path, method);
    }

    enableParentalControl() {
        const { path, method } = this.PARENTAL_ENABLE;

        return this.makeRequest(path, method);
    }

    disableParentalControl() {
        const { path, method } = this.PARENTAL_DISABLE;

        return this.makeRequest(path, method);
    }

    // Safebrowsing
    SAFEBROWSING_STATUS = { path: 'safebrowsing/status', method: 'GET' };

    SAFEBROWSING_ENABLE = { path: 'safebrowsing/enable', method: 'POST' };

    SAFEBROWSING_DISABLE = { path: 'safebrowsing/disable', method: 'POST' };

    getSafebrowsingStatus() {
        const { path, method } = this.SAFEBROWSING_STATUS;

        return this.makeRequest(path, method);
    }

    enableSafebrowsing() {
        const { path, method } = this.SAFEBROWSING_ENABLE;

        return this.makeRequest(path, method);
    }

    disableSafebrowsing() {
        const { path, method } = this.SAFEBROWSING_DISABLE;

        return this.makeRequest(path, method);
    }

    // Safesearch
    SAFESEARCH_STATUS = { path: 'safesearch/status', method: 'GET' };

    SAFESEARCH_UPDATE = { path: 'safesearch/settings', method: 'PUT' };

    getSafesearchStatus() {
        const { path, method } = this.SAFESEARCH_STATUS;

        return this.makeRequest(path, method);
    }

    /**
     * interface SafeSearchConfig {
        "enabled": boolean,
        "bing": boolean,
        "duckduckgo": boolean,
        "google": boolean,
        "pixabay": boolean,
        "yandex": boolean,
        "youtube": boolean
     * }
     * @param {*} data - SafeSearchConfig
     * @returns 200 ok
     */
    updateSafesearch(data: any) {
        const { path, method } = this.SAFESEARCH_UPDATE;
        return this.makeRequest(path, method, { data });
    }

    // enableSafesearch() {
    //     const { path, method } = this.SAFESEARCH_ENABLE;
    //     return this.makeRequest(path, method);
    // }

    // disableSafesearch() {
    //     const { path, method } = this.SAFESEARCH_DISABLE;
    //     return this.makeRequest(path, method);
    // }

    // Language

    async changeLanguage(config: any) {
        const profile = await this.getProfile();
        profile.language = config.language;

        return this.setProfile(profile);
    }

    // Theme

    async changeTheme(config: any) {
        const profile = await this.getProfile();
        profile.theme = config.theme;

        return this.setProfile(profile);
    }

    // DHCP
    DHCP_STATUS = { path: 'dhcp/status', method: 'GET' };

    DHCP_SET_CONFIG = { path: 'dhcp/set_config', method: 'POST' };

    DHCP_FIND_ACTIVE = { path: 'dhcp/find_active_dhcp', method: 'POST' };

    DHCP_INTERFACES = { path: 'dhcp/interfaces', method: 'GET' };

    DHCP_ADD_STATIC_LEASE = { path: 'dhcp/add_static_lease', method: 'POST' };

    DHCP_REMOVE_STATIC_LEASE = { path: 'dhcp/remove_static_lease', method: 'POST' };

    DHCP_UPDATE_STATIC_LEASE = { path: 'dhcp/update_static_lease', method: 'POST' };

    DHCP_RESET = { path: 'dhcp/reset', method: 'POST' };

    DHCP_LEASES_RESET = { path: 'dhcp/reset_leases', method: 'POST' };

    getDhcpStatus() {
        const { path, method } = this.DHCP_STATUS;

        return this.makeRequest(path, method);
    }

    getDhcpInterfaces() {
        const { path, method } = this.DHCP_INTERFACES;

        return this.makeRequest(path, method);
    }

    setDhcpConfig(config: any) {
        const { path, method } = this.DHCP_SET_CONFIG;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    findActiveDhcp(req: any) {
        const { path, method } = this.DHCP_FIND_ACTIVE;
        const parameters = {
            data: req,
        };
        return this.makeRequest(path, method, parameters);
    }

    addStaticLease(config: any) {
        const { path, method } = this.DHCP_ADD_STATIC_LEASE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    removeStaticLease(config: any) {
        const { path, method } = this.DHCP_REMOVE_STATIC_LEASE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    updateStaticLease(config: any) {
        const { path, method } = this.DHCP_UPDATE_STATIC_LEASE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    resetDhcp() {
        const { path, method } = this.DHCP_RESET;

        return this.makeRequest(path, method);
    }

    resetDhcpLeases() {
        const { path, method } = this.DHCP_LEASES_RESET;

        return this.makeRequest(path, method);
    }

    // Installation
    INSTALL_GET_ADDRESSES = { path: 'install/get_addresses', method: 'GET' };

    INSTALL_CONFIGURE = { path: 'install/configure', method: 'POST' };

    INSTALL_CHECK_CONFIG = { path: 'install/check_config', method: 'POST' };

    getDefaultAddresses() {
        const { path, method } = this.INSTALL_GET_ADDRESSES;

        return this.makeRequest(path, method);
    }

    setAllSettings(config: any) {
        const { path, method } = this.INSTALL_CONFIGURE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    checkConfig(config: any) {
        const { path, method } = this.INSTALL_CHECK_CONFIG;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    // DNS-over-HTTPS and DNS-over-TLS
    TLS_STATUS = { path: 'tls/status', method: 'GET' };

    TLS_CONFIG = { path: 'tls/configure', method: 'POST' };

    TLS_VALIDATE = { path: 'tls/validate', method: 'POST' };

    getTlsStatus() {
        const { path, method } = this.TLS_STATUS;

        return this.makeRequest(path, method);
    }

    setTlsConfig(config: any) {
        const { path, method } = this.TLS_CONFIG;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    validateTlsConfig(config: any) {
        const { path, method } = this.TLS_VALIDATE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    // Per-client settings
    GET_CLIENTS = { path: 'clients', method: 'GET' };

    SEARCH_CLIENTS = { path: 'clients/search', method: 'POST' };

    ADD_CLIENT = { path: 'clients/add', method: 'POST' };

    DELETE_CLIENT = { path: 'clients/delete', method: 'POST' };

    UPDATE_CLIENT = { path: 'clients/update', method: 'POST' };

    getClients() {
        const { path, method } = this.GET_CLIENTS;

        return this.makeRequest(path, method);
    }

    addClient(config: any) {
        const { path, method } = this.ADD_CLIENT;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    deleteClient(config: any) {
        const { path, method } = this.DELETE_CLIENT;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    updateClient(config: any) {
        const { path, method } = this.UPDATE_CLIENT;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    searchClients(config: any) {
        const { path, method } = this.SEARCH_CLIENTS;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    // DNS access settings
    ACCESS_LIST = { path: 'access/list', method: 'GET' };

    ACCESS_SET = { path: 'access/set', method: 'POST' };

    getAccessList() {
        const { path, method } = this.ACCESS_LIST;

        return this.makeRequest(path, method);
    }

    setAccessList(config: any) {
        const { path, method } = this.ACCESS_SET;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    // DNS rewrites
    REWRITES_LIST = { path: 'rewrite/list', method: 'GET' };

    REWRITE_ADD = { path: 'rewrite/add', method: 'POST' };

    REWRITE_UPDATE = { path: 'rewrite/update', method: 'PUT' };

    REWRITE_DELETE = { path: 'rewrite/delete', method: 'POST' };

    getRewritesList() {
        const { path, method } = this.REWRITES_LIST;

        return this.makeRequest(path, method);
    }

    addRewrite(config: any) {
        const { path, method } = this.REWRITE_ADD;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    updateRewrite(config: any) {
        const { path, method } = this.REWRITE_UPDATE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    deleteRewrite(config: any) {
        const { path, method } = this.REWRITE_DELETE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    // Blocked services
    BLOCKED_SERVICES_GET = { path: 'blocked_services/get', method: 'GET' };

    BLOCKED_SERVICES_UPDATE = { path: 'blocked_services/update', method: 'PUT' };

    BLOCKED_SERVICES_ALL = { path: 'blocked_services/all', method: 'GET' };

    getAllBlockedServices() {
        const { path, method } = this.BLOCKED_SERVICES_ALL;

        return this.makeRequest(path, method);
    }

    getBlockedServices() {
        const { path, method } = this.BLOCKED_SERVICES_GET;

        return this.makeRequest(path, method);
    }

    updateBlockedServices(config: any) {
        const { path, method } = this.BLOCKED_SERVICES_UPDATE;
        const parameters = {
            data: config,
        };
        return this.makeRequest(path, method, parameters);
    }

    // Settings for statistics
    GET_STATS = { path: 'stats', method: 'GET' };

    GET_STATS_CONFIG = { path: 'stats/config', method: 'GET' };

    UPDATE_STATS_CONFIG = { path: 'stats/config/update', method: 'PUT' };

    STATS_RESET = { path: 'stats_reset', method: 'POST' };

    getStats() {
        const { path, method } = this.GET_STATS;

        return this.makeRequest(path, method);
    }

    getStatsConfig() {
        const { path, method } = this.GET_STATS_CONFIG;

        return this.makeRequest(path, method);
    }

    setStatsConfig(data: any) {
        const { path, method } = this.UPDATE_STATS_CONFIG;
        const config = {
            data,
        };
        return this.makeRequest(path, method, config);
    }

    resetStats() {
        const { path, method } = this.STATS_RESET;

        return this.makeRequest(path, method);
    }

    // Query log
    GET_QUERY_LOG = { path: 'querylog', method: 'GET' };

    UPDATE_QUERY_LOG_CONFIG = { path: 'querylog/config/update', method: 'PUT' };

    GET_QUERY_LOG_CONFIG = { path: 'querylog/config', method: 'GET' };

    QUERY_LOG_CLEAR = { path: 'querylog_clear', method: 'POST' };

    getQueryLog(params: any) {
        const { path, method } = this.GET_QUERY_LOG;
        // eslint-disable-next-line no-param-reassign
        params.limit = QUERY_LOGS_PAGE_LIMIT;
        const url = getPathWithQueryString(path, params);

        return this.makeRequest(url, method);
    }

    getQueryLogConfig() {
        const { path, method } = this.GET_QUERY_LOG_CONFIG;

        return this.makeRequest(path, method);
    }

    setQueryLogConfig(data: any) {
        const { path, method } = this.UPDATE_QUERY_LOG_CONFIG;
        const config = {
            data,
        };
        return this.makeRequest(path, method, config);
    }

    clearQueryLog() {
        const { path, method } = this.QUERY_LOG_CLEAR;

        return this.makeRequest(path, method);
    }

    // Login
    LOGIN = { path: 'login', method: 'POST' };

    login(data: any) {
        const { path, method } = this.LOGIN;
        const config = {
            data,
        };
        return this.makeRequest(path, method, config);
    }

    // Profile
    GET_PROFILE = { path: 'profile', method: 'GET' };

    UPDATE_PROFILE = { path: 'profile/update', method: 'PUT' };

    getProfile() {
        const { path, method } = this.GET_PROFILE;

        return this.makeRequest(path, method);
    }

    setProfile(data: any) {
        const theme = data.theme ? data.theme : THEMES.auto;
        const defaultLanguage = i18n.language ? i18n.language : LANGUAGES.en;
        const language = data.language ? data.language : defaultLanguage;

        const { path, method } = this.UPDATE_PROFILE;
        const config = { data: { theme, language } };

        return this.makeRequest(path, method, config);
    }

    // DNS config
    GET_DNS_CONFIG = { path: 'dns_info', method: 'GET' };

    SET_DNS_CONFIG = { path: 'dns_config', method: 'POST' };

    getDnsConfig() {
        const { path, method } = this.GET_DNS_CONFIG;

        return this.makeRequest(path, method);
    }

    setDnsConfig(data: any) {
        const { path, method } = this.SET_DNS_CONFIG;
        const config = {
            data,
        };
        return this.makeRequest(path, method, config);
    }

    SET_PROTECTION = { path: 'protection', method: 'POST' };

    setProtection(data: any) {
        const { enabled, duration } = data;
        const { path, method } = this.SET_PROTECTION;

        return this.makeRequest(path, method, { data: { enabled, duration } });
    }

    // Cache
    CLEAR_CACHE = { path: 'cache_clear', method: 'POST' };

    clearCache() {
        const { path, method } = this.CLEAR_CACHE;

        return this.makeRequest(path, method);
    }
}

const apiClient = new Api();
export default apiClient;
