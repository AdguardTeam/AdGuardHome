import {
    ALL_INTERFACES_IP,
    BLOCKING_MODES,
    DAY,
    DEFAULT_LOGS_FILTER,
    INSTALL_FIRST_STEP,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
    TIME_UNITS,
} from './helpers/constants';
import { DEFAULT_BLOCKING_IPV4, DEFAULT_BLOCKING_IPV6 } from './reducers/dnsConfig';
import { Filter } from './helpers/helpers';

export type InstallData = {
    step: number;
    processingDefault: boolean;
    processingSubmit: boolean;
    processingCheck: boolean;
    web: {
        ip: string;
        port: number;
        status: string;
        can_autofix: boolean;
    };
    dns: {
        ip: string;
        port: number;
        status: string;
        can_autofix: boolean;
    };
    staticIp: {
        static: string;
        ip: string;
        error: string;
    };
    interfaces: {
        flags: string;
        hardware_address: string;
        ip_addresses: string[];
        mtu: number;
        name: string;
    }[];
    dnsVersion: string;
};

export type EncryptionData = {
    processing: boolean;
    processingConfig: boolean;
    processingValidate: boolean;
    enabled: boolean;
    serve_plain_dns: boolean;
    dns_names: any;
    force_https: boolean;
    issuer: string;
    key_type: string;
    not_after: string;
    not_before: string;
    port_dns_over_tls?: number;
    port_dns_over_quic?: number;
    port_https?: number;
    port_dnscrypt?: number;
    subject: string;
    valid_chain: boolean;
    valid_key: boolean;
    valid_cert: boolean;
    valid_pair: boolean;
    status_cert: string;
    status_key: string;
    private_key: string;
    server_name: string;
    warning_validation: string;
    certificate_chain: string;
    certificate_path: string;
    private_key_path: string;
    private_key_saved: boolean;
    allow_unencrypted_doh?: boolean;
    dnscrypt_config_file?: string;
};

export type Client = {
    blocked_services: string[],
    blocked_services_schedule: {
        sun?: { start: number, end: number },
        mon?: { start: number, end: number },
        tue?: { start: number, end: number },
        wed?: { start: number, end: number },
        thu?: { start: number, end: number },
        fri?: { start: number, end: number },
        sat?: { start: number, end: number },
        time_zone: string;
    },
    filtering_enabled: boolean;
    ids: string[];
    ignore_querylog: boolean;
    ignore_statistics: boolean;
    name: string;
    parental_enabled: boolean;
    safe_search: Record<string, boolean>;
    safebrowsing_enabled: boolean;
    safesearch_enabled: boolean;
    tags: string[];
    upstreams: string[];
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
    use_global_blocked_services: boolean;
    use_global_settings: boolean;
}

export type AutoClient = {
    ip: string;
    name: string;
    source: string;
    whois_info: any;
}

export type DashboardData = {
    processing: boolean;
    isCoreRunning: boolean;
    processingVersion: boolean;
    processingClients: boolean;
    processingUpdate: boolean;
    processingProfile: boolean;
    protectionEnabled: boolean;
    protectionDisabledDuration: any;
    protectionCountdownActive: boolean;
    processingProtection: boolean;
    httpPort: number;
    dnsPort: number;
    dnsAddresses: string[];
    dnsVersion: string;
    clients: Client[];
    autoClients: AutoClient[];
    supportedTags: string[];
    name: string;
    theme: string | null;
    checkUpdateFlag: boolean;
    announcementUrl: string;
    newVersion: string;
    canAutoUpdate: boolean;
    language: string;
    isUpdateAvailable: boolean;
};

export type SettingsData = {
    processing: boolean;
    processingTestUpstream: boolean;
    processingDhcpStatus: boolean;
    settingsList?: {
        parental: {
            enabled: boolean;
            order: number;
            subtitle: string;
            title: string;
        },
        safebrowsing: {
            enabled: boolean;
            order: number;
            subtitle: string;
            title: string;
        },
        safesearch: Record<string, boolean>;
    };
};

export type RewritesData = {
    processing: boolean;
    processingAdd: boolean;
    processingDelete: boolean;
    processingUpdate: boolean;
    isModalOpen: boolean;
    modalType: string;
    currentRewrite?: {
        answer: string;
        domain: string;
    };
    list: {
        answer: string;
        domain: string;
    }[];
};

export type NormalizedTopClients = {
    auto: Record<string, number>;
    configured: Record<string, number>;
}

export type StatsData = {
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    processingStats: boolean;
    processingReset: boolean;
    interval: number;
    customInterval?: number;
    dnsQueries: number[];
    blockedFiltering: number[];
    replacedParental: number[];
    replacedSafebrowsing: number[];
    topBlockedDomains: { name: string; count: number }[];
    topClients: {
        name: string;
        count: number;
        info: any;
    }[];
    normalizedTopClients?: NormalizedTopClients;
    topQueriedDomains: { name: string; count: number }[];
    numBlockedFiltering: number;
    numDnsQueries: number;
    numReplacedParental: number;
    numReplacedSafebrowsing: number;
    numReplacedSafesearch: number;
    avgProcessingTime: number;
    timeUnits: string;
    enabled: boolean;
    topUpstreamsAvgTime: { name: string; count: number }[];
    topUpstreamsResponses: { name: string; count: number }[];
};

export type ClientsData = {
    processing: boolean;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    isModalOpen: boolean;
    modalClientName: string;
    modalType: string;
};

export type AccessData = {
    processing: boolean;
    processingSet: boolean;
    allowed_clients: string;
    disallowed_clients: string;
    blocked_hosts: string;
};

export type DhcpInterface = {
    name: string;
    flags: string;
    gateway_ip: string;
    ip_addresses: string[];
    ipv4_addresses: string[];
    ipv6_addresses: string[];
    hardware_address: string;
};

export type DhcpData = {
    processing: boolean;
    processingStatus: boolean;
    processingInterfaces: boolean;
    processingDhcp: boolean;
    processingConfig: boolean;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    enabled: boolean;
    interface_name: string;
    check?: {
        v4?: {
            other_server?: { found: string; error?: string },
            static_ip?: {static: string, ip: string},
        },
        v6?: {
            other_server?: { found: string; error?: string },
            static_ip?: {static: string, ip: string},
        },
    };
    v4: {
        gateway_ip: string;
        subnet_mask: string;
        range_start: string;
        range_end: string;
        lease_duration: number;
    };
    v6: {
        range_start: string;
        lease_duration: number;
    };
    leases: {
        hostname: string;
        ip: string;
        mac: string;
    }[];
    staticLeases: {
        hostname: string;
        ip: string;
        mac: string;
    }[];
    isModalOpen: boolean;
    leaseModalConfig?: {
        hostname: string;
        ip: string;
        mac: string;
    };
    modalType: string;
    dhcp_available: boolean;
    interfaces?: DhcpInterface[];
};

export type DnsConfigData = {
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    blocking_mode: string;
    ratelimit: number;
    blocking_ipv4: string;
    blocking_ipv6: string;
    blocked_response_ttl: number;
    edns_cs_enabled: boolean;
    disable_ipv6: boolean;
    dnssec_enabled: boolean;
    upstream_dns_file: string;
    upstream_dns: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    ratelimit_whitelist: string;
    upstream_mode: string;
    resolve_clients: boolean;
    use_private_ptr_resolvers: boolean;
    default_local_ptr_upstreams: any[];
    ratelimit_subnet_len_ipv4?: number;
    ratelimit_subnet_len_ipv6?: number;
    edns_cs_use_custom?: boolean;
    edns_cs_custom_ip?: boolean;
    cache_size?: number;
    cache_ttl_max?: number;
    cache_ttl_min?: number;
    cache_optimistic?: boolean;
};

export type FilteringData = {
    isModalOpen: boolean;
    processingFilters: boolean;
    processingRules: boolean;
    processingAddFilter: boolean;
    processingRefreshFilters: boolean;
    processingConfigFilter: boolean;
    processingRemoveFilter: boolean;
    processingSetConfig: boolean;
    processingCheck: boolean;
    isFilterAdded: boolean;
    filters: Filter[];
    whitelistFilters: any[];
    userRules: string;
    interval: number;
    enabled: boolean;
    modalType: string;
    modalFilterUrl: string;
    check: any;
};

export type QueryLogsData = {
    processingGetLogs: boolean;
    processingClear: boolean;
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    processingAdditionalLogs: boolean;
    interval: any;
    logs: any[];
    enabled: boolean;
    oldest: string;
    filter: any;
    isFiltered: boolean;
    anonymize_client_ip: boolean;
    isDetailed: boolean;
    isEntireLog: boolean;
    customInterval: any;
};

export type ServicesData = {
    processing: boolean;
    processingAll: boolean;
    processingSet: boolean;
    list: any;
    allServices: any[];
};

export type RootState = {
    access?: AccessData;
    clients?: ClientsData;
    dashboard?: DashboardData;
    dhcp?: DhcpData;
    dnsConfig?: DnsConfigData;
    encryption?: EncryptionData;
    filtering?: FilteringData;
    queryLogs?: QueryLogsData;
    rewrites?: RewritesData;
    services?: ServicesData;
    settings?: SettingsData;
    stats?: StatsData;
    install?: InstallData;
    toasts: { notices: any[] };
    loadingBar: any;
    form: any;
};

export type InstallState = {
    install: InstallData;
    toasts: { notices: any[] };
};

export type LoginState = {
    login: {
        processingLogin: false;
        email: string;
        password: string;
    };
    toasts: { notices: any[] };
};

export const initialState: RootState = {
    access: {
        processing: true,
        processingSet: false,
        allowed_clients: '',
        disallowed_clients: '',
        blocked_hosts: '',
    },
    clients: {
        processing: true,
        processingAdding: false,
        processingDeleting: false,
        processingUpdating: false,
        isModalOpen: false,
        modalClientName: '',
        modalType: '',
    },
    dashboard: {
        processing: true,
        isCoreRunning: true,
        processingVersion: true,
        processingClients: true,
        processingUpdate: false,
        processingProfile: true,
        protectionEnabled: false,
        protectionDisabledDuration: null,
        protectionCountdownActive: false,
        processingProtection: false,
        httpPort: STANDARD_WEB_PORT,
        dnsPort: STANDARD_DNS_PORT,
        dnsAddresses: [],
        dnsVersion: '',
        clients: [],
        autoClients: [],
        supportedTags: [],
        name: '',
        theme: undefined,
        checkUpdateFlag: false,
        announcementUrl: '',
        newVersion: '',
        canAutoUpdate: false,
        language: '', // ???
        isUpdateAvailable: false,
    },
    dhcp: {
        processing: true,
        processingStatus: false,
        processingInterfaces: false,
        processingDhcp: false,
        processingConfig: false,
        processingAdding: false,
        processingDeleting: false,
        processingUpdating: false,
        enabled: false,
        interface_name: '',
        check: null,
        v4: {
            gateway_ip: '',
            subnet_mask: '',
            range_start: '',
            range_end: '',
            lease_duration: 0,
        },
        v6: {
            range_start: '',
            lease_duration: 0,
        },
        leases: [],
        staticLeases: [],
        isModalOpen: false,
        leaseModalConfig: undefined,
        modalType: '',
        dhcp_available: false,
    },
    dnsConfig: {
        processingGetConfig: false,
        processingSetConfig: false,
        blocking_mode: BLOCKING_MODES.default,
        ratelimit: 20,
        blocking_ipv4: DEFAULT_BLOCKING_IPV4,
        blocking_ipv6: DEFAULT_BLOCKING_IPV6,
        blocked_response_ttl: 10,
        edns_cs_enabled: false,
        disable_ipv6: false,
        dnssec_enabled: false,
        upstream_dns_file: '',
        upstream_dns: '',
        fallback_dns: '',
        bootstrap_dns: '',
        local_ptr_upstreams: '',
        ratelimit_whitelist: '',
        upstream_mode: '',
        resolve_clients: false,
        use_private_ptr_resolvers: false,
        default_local_ptr_upstreams: [],
    },
    encryption: {
        processing: true,
        processingConfig: false,
        processingValidate: false,
        enabled: false,
        serve_plain_dns: false,
        dns_names: null,
        force_https: false,
        issuer: '',
        key_type: '',
        not_after: '',
        not_before: '',
        subject: '',
        valid_chain: false,
        valid_key: false,
        valid_cert: false,
        valid_pair: false,
        status_cert: '',
        status_key: '',
        certificate_chain: '',
        private_key: '',
        server_name: '',
        warning_validation: '',
        certificate_path: '',
        private_key_path: '',
        private_key_saved: false,
    },
    filtering: {
        isModalOpen: false,
        processingFilters: false,
        processingRules: false,
        processingAddFilter: false,
        processingRefreshFilters: false,
        processingConfigFilter: false,
        processingRemoveFilter: false,
        processingSetConfig: false,
        processingCheck: false,
        isFilterAdded: false,
        filters: [],
        whitelistFilters: [],
        userRules: '',
        interval: 24,
        enabled: true,
        modalType: '',
        modalFilterUrl: '',
        check: {},
    },
    queryLogs: {
        processingGetLogs: true,
        processingClear: false,
        processingGetConfig: false,
        processingSetConfig: false,
        processingAdditionalLogs: false,
        interval: DAY,
        logs: [],
        enabled: true,
        oldest: '',
        filter: DEFAULT_LOGS_FILTER,
        isFiltered: false,
        anonymize_client_ip: false,
        isDetailed: true,
        isEntireLog: false,
        customInterval: null,
    },
    rewrites: {
        processing: true,
        processingAdd: false,
        processingDelete: false,
        processingUpdate: false,
        isModalOpen: false,
        modalType: '',
        list: [],
    },
    services: {
        processing: true,
        processingAll: true,
        processingSet: false,
        list: {},
        allServices: [],
    },
    settings: {
        processing: true,
        processingTestUpstream: false,
        processingDhcpStatus: false,
    },
    stats: {
        processingGetConfig: false,
        processingSetConfig: false,
        processingStats: true,
        processingReset: false,
        interval: DAY,
        customInterval: null,
        dnsQueries: [],
        blockedFiltering: [],
        replacedParental: [],
        replacedSafebrowsing: [],
        topBlockedDomains: [],
        topClients: [],
        topQueriedDomains: [],
        numBlockedFiltering: 0,
        numDnsQueries: 0,
        numReplacedParental: 0,
        numReplacedSafebrowsing: 0,
        numReplacedSafesearch: 0,
        avgProcessingTime: 0,
        timeUnits: TIME_UNITS.HOURS,
        enabled: true,
        topUpstreamsAvgTime: [],
        topUpstreamsResponses: [],
    },
    toasts: { notices: [] },
    loadingBar: {},
    form: {},
};
