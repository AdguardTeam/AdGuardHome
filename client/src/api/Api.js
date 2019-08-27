import axios from 'axios';

export default class Api {
    baseUrl = 'control';

    async makeRequest(path, method = 'POST', config) {
        try {
            const response = await axios({
                url: `${this.baseUrl}/${path}`,
                method,
                ...config,
            });
            return response.data;
        } catch (error) {
            console.error(error);
            const errorPath = `${this.baseUrl}/${path}`;
            if (error.response) {
                throw new Error(`${errorPath} | ${error.response.data} | ${error.response.status}`);
            }
            throw new Error(`${errorPath} | ${error.message ? error.message : error}`);
        }
    }

    // Global methods
    GLOBAL_START = { path: 'start', method: 'POST' };
    GLOBAL_STATUS = { path: 'status', method: 'GET' };
    GLOBAL_STOP = { path: 'stop', method: 'POST' };
    GLOBAL_QUERY_LOG = { path: 'querylog', method: 'GET' };
    GLOBAL_QUERY_LOG_ENABLE = { path: 'querylog_enable', method: 'POST' };
    GLOBAL_QUERY_LOG_DISABLE = { path: 'querylog_disable', method: 'POST' };
    GLOBAL_SET_UPSTREAM_DNS = { path: 'set_upstreams_config', method: 'POST' };
    GLOBAL_TEST_UPSTREAM_DNS = { path: 'test_upstream_dns', method: 'POST' };
    GLOBAL_VERSION = { path: 'version.json', method: 'POST' };
    GLOBAL_ENABLE_PROTECTION = { path: 'enable_protection', method: 'POST' };
    GLOBAL_DISABLE_PROTECTION = { path: 'disable_protection', method: 'POST' };
    GLOBAL_UPDATE = { path: 'update', method: 'POST' };

    startGlobalFiltering() {
        const { path, method } = this.GLOBAL_START;
        return this.makeRequest(path, method);
    }

    stopGlobalFiltering() {
        const { path, method } = this.GLOBAL_STOP;
        return this.makeRequest(path, method);
    }

    getGlobalStatus() {
        const { path, method } = this.GLOBAL_STATUS;
        return this.makeRequest(path, method);
    }

    getQueryLog() {
        const { path, method } = this.GLOBAL_QUERY_LOG;
        return this.makeRequest(path, method);
    }

    downloadQueryLog() {
        const { path, method } = this.GLOBAL_QUERY_LOG;
        const queryString = '?download=1';
        return this.makeRequest(path + queryString, method);
    }

    enableQueryLog() {
        const { path, method } = this.GLOBAL_QUERY_LOG_ENABLE;
        return this.makeRequest(path, method);
    }

    disableQueryLog() {
        const { path, method } = this.GLOBAL_QUERY_LOG_DISABLE;
        return this.makeRequest(path, method);
    }

    setUpstream(url) {
        const { path, method } = this.GLOBAL_SET_UPSTREAM_DNS;
        const config = {
            data: url,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, config);
    }

    testUpstream(servers) {
        const { path, method } = this.GLOBAL_TEST_UPSTREAM_DNS;
        const config = {
            data: servers,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, config);
    }

    getGlobalVersion(data) {
        const { path, method } = this.GLOBAL_VERSION;
        const config = {
            data,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, config);
    }

    enableGlobalProtection() {
        const { path, method } = this.GLOBAL_ENABLE_PROTECTION;
        return this.makeRequest(path, method);
    }

    disableGlobalProtection() {
        const { path, method } = this.GLOBAL_DISABLE_PROTECTION;
        return this.makeRequest(path, method);
    }

    getUpdate() {
        const { path, method } = this.GLOBAL_UPDATE;
        return this.makeRequest(path, method);
    }

    // Filtering
    FILTERING_STATUS = { path: 'filtering/status', method: 'GET' };
    FILTERING_ENABLE = { path: 'filtering/enable', method: 'POST' };
    FILTERING_DISABLE = { path: 'filtering/disable', method: 'POST' };
    FILTERING_ADD_FILTER = { path: 'filtering/add_url', method: 'POST' };
    FILTERING_REMOVE_FILTER = { path: 'filtering/remove_url', method: 'POST' };
    FILTERING_SET_RULES = { path: 'filtering/set_rules', method: 'POST' };
    FILTERING_ENABLE_FILTER = { path: 'filtering/enable_url', method: 'POST' };
    FILTERING_DISABLE_FILTER = { path: 'filtering/disable_url', method: 'POST' };
    FILTERING_REFRESH = { path: 'filtering/refresh', method: 'POST' };

    getFilteringStatus() {
        const { path, method } = this.FILTERING_STATUS;
        return this.makeRequest(path, method);
    }

    enableFiltering() {
        const { path, method } = this.FILTERING_ENABLE;
        return this.makeRequest(path, method);
    }

    disableFiltering() {
        const { path, method } = this.FILTERING_DISABLE;
        return this.makeRequest(path, method);
    }

    // TODO find out when to use force parameter
    refreshFilters() {
        const { path, method } = this.FILTERING_REFRESH;
        return this.makeRequest(path, method);
    }

    addFilter(url, name) {
        const { path, method } = this.FILTERING_ADD_FILTER;
        const config = {
            data: {
                name,
                url,
            },
        };
        return this.makeRequest(path, method, config);
    }

    removeFilter(config) {
        const { path, method } = this.FILTERING_REMOVE_FILTER;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };

        return this.makeRequest(path, method, parameters);
    }

    setRules(rules) {
        const { path, method } = this.FILTERING_SET_RULES;
        const parameters = {
            data: rules,
            headers: { 'Content-Type': 'text/plain' },
        };
        return this.makeRequest(path, method, parameters);
    }

    enableFilter(url) {
        const { path, method } = this.FILTERING_ENABLE_FILTER;
        const parameter = 'url';
        const requestBody = `${parameter}=${url}`;
        const config = {
            data: requestBody,
            header: { 'Content-Type': 'text/plain' },
        };
        return this.makeRequest(path, method, config);
    }

    disableFilter(url) {
        const { path, method } = this.FILTERING_DISABLE_FILTER;
        const parameter = 'url';
        const requestBody = `${parameter}=${url}`;
        const config = {
            data: requestBody,
            header: { 'Content-Type': 'text/plain' },
        };
        return this.makeRequest(path, method, config);
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
        const parameter = 'sensitivity=TEEN'; // this parameter TEEN is hardcoded
        const config = {
            data: parameter,
            headers: { 'Content-Type': 'text/plain' },
        };
        return this.makeRequest(path, method, config);
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
    SAFESEARCH_ENABLE = { path: 'safesearch/enable', method: 'POST' };
    SAFESEARCH_DISABLE = { path: 'safesearch/disable', method: 'POST' };

    getSafesearchStatus() {
        const { path, method } = this.SAFESEARCH_STATUS;
        return this.makeRequest(path, method);
    }

    enableSafesearch() {
        const { path, method } = this.SAFESEARCH_ENABLE;
        return this.makeRequest(path, method);
    }

    disableSafesearch() {
        const { path, method } = this.SAFESEARCH_DISABLE;
        return this.makeRequest(path, method);
    }

    // Language
    CURRENT_LANGUAGE = { path: 'i18n/current_language', method: 'GET' };
    CHANGE_LANGUAGE = { path: 'i18n/change_language', method: 'POST' };

    getCurrentLanguage() {
        const { path, method } = this.CURRENT_LANGUAGE;
        return this.makeRequest(path, method);
    }

    changeLanguage(lang) {
        const { path, method } = this.CHANGE_LANGUAGE;
        const parameters = {
            data: lang,
            headers: { 'Content-Type': 'text/plain' },
        };
        return this.makeRequest(path, method, parameters);
    }

    // DHCP
    DHCP_STATUS = { path: 'dhcp/status', method: 'GET' };
    DHCP_SET_CONFIG = { path: 'dhcp/set_config', method: 'POST' };
    DHCP_FIND_ACTIVE = { path: 'dhcp/find_active_dhcp', method: 'POST' };
    DHCP_INTERFACES = { path: 'dhcp/interfaces', method: 'GET' };
    DHCP_ADD_STATIC_LEASE = { path: 'dhcp/add_static_lease', method: 'POST' };
    DHCP_REMOVE_STATIC_LEASE = { path: 'dhcp/remove_static_lease', method: 'POST' };

    getDhcpStatus() {
        const { path, method } = this.DHCP_STATUS;
        return this.makeRequest(path, method);
    }

    getDhcpInterfaces() {
        const { path, method } = this.DHCP_INTERFACES;
        return this.makeRequest(path, method);
    }

    setDhcpConfig(config) {
        const { path, method } = this.DHCP_SET_CONFIG;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    findActiveDhcp(name) {
        const { path, method } = this.DHCP_FIND_ACTIVE;
        const parameters = {
            data: name,
            headers: { 'Content-Type': 'text/plain' },
        };
        return this.makeRequest(path, method, parameters);
    }

    addStaticLease(config) {
        const { path, method } = this.DHCP_ADD_STATIC_LEASE;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    removeStaticLease(config) {
        const { path, method } = this.DHCP_REMOVE_STATIC_LEASE;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    // Installation
    INSTALL_GET_ADDRESSES = { path: 'install/get_addresses', method: 'GET' };
    INSTALL_CONFIGURE = { path: 'install/configure', method: 'POST' };
    INSTALL_CHECK_CONFIG = { path: 'install/check_config', method: 'POST' };

    getDefaultAddresses() {
        const { path, method } = this.INSTALL_GET_ADDRESSES;
        return this.makeRequest(path, method);
    }

    setAllSettings(config) {
        const { path, method } = this.INSTALL_CONFIGURE;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    checkConfig(config) {
        const { path, method } = this.INSTALL_CHECK_CONFIG;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
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

    setTlsConfig(config) {
        const { path, method } = this.TLS_CONFIG;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    validateTlsConfig(config) {
        const { path, method } = this.TLS_VALIDATE;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    // Per-client settings
    GET_CLIENTS = { path: 'clients', method: 'GET' };
    ADD_CLIENT = { path: 'clients/add', method: 'POST' };
    DELETE_CLIENT = { path: 'clients/delete', method: 'POST' };
    UPDATE_CLIENT = { path: 'clients/update', method: 'POST' };

    getClients() {
        const { path, method } = this.GET_CLIENTS;
        return this.makeRequest(path, method);
    }

    addClient(config) {
        const { path, method } = this.ADD_CLIENT;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    deleteClient(config) {
        const { path, method } = this.DELETE_CLIENT;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    updateClient(config) {
        const { path, method } = this.UPDATE_CLIENT;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
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

    setAccessList(config) {
        const { path, method } = this.ACCESS_SET;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    // DNS rewrites
    REWRITES_LIST = { path: 'rewrite/list', method: 'GET' };
    REWRITE_ADD = { path: 'rewrite/add', method: 'POST' };
    REWRITE_DELETE = { path: 'rewrite/delete', method: 'POST' };

    getRewritesList() {
        const { path, method } = this.REWRITES_LIST;
        return this.makeRequest(path, method);
    }

    addRewrite(config) {
        const { path, method } = this.REWRITE_ADD;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    deleteRewrite(config) {
        const { path, method } = this.REWRITE_DELETE;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    // Blocked services
    BLOCKED_SERVICES_LIST = { path: 'blocked_services/list', method: 'GET' };
    BLOCKED_SERVICES_SET = { path: 'blocked_services/set', method: 'POST' };

    getBlockedServices() {
        const { path, method } = this.BLOCKED_SERVICES_LIST;
        return this.makeRequest(path, method);
    }

    setBlockedServices(config) {
        const { path, method } = this.BLOCKED_SERVICES_SET;
        const parameters = {
            data: config,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, parameters);
    }

    // Settings for statistics
    GET_STATS = { path: 'stats', method: 'GET' };
    STATS_INFO = { path: 'stats_info', method: 'GET' };
    STATS_CONFIG = { path: 'stats_config', method: 'POST' };

    getStats() {
        const { path, method } = this.GET_STATS;
        return this.makeRequest(path, method);
    }

    getStatsInfo() {
        const { path, method } = this.STATS_INFO;
        return this.makeRequest(path, method);
    }

    setStatsConfig(data) {
        const { path, method } = this.STATS_CONFIG;
        const config = {
            data,
            headers: { 'Content-Type': 'application/json' },
        };
        return this.makeRequest(path, method, config);
    }
}
