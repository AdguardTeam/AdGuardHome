import {
    BLOCKING_MODES,
    DAY,
    DEFAULT_DNS_CACHE_SIZE,
    DEFAULT_LOGS_FILTER,
    ModalType,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
    TIME_UNITS,
} from './helpers/constants';
import { DEFAULT_BLOCKING_IPV4, DEFAULT_BLOCKING_IPV6 } from './stores/dnsConfig';
import { Filter, type NormalizedQueryLogItem } from './helpers/helpers';
import type { WhoisInfo } from './api/model/whoisInfo';
import type { ClientAuto as AutoClient } from './api/model/clientAuto';
import type { Client } from './api/model/client';
import type { DHCPNetInterfaces } from './api/model/dHCPNetInterfaces';
import type { TlsConfig } from './api/model/tlsConfig';
import type { TlsConfigKeyType } from './api/model/tlsConfigKeyType';
import type { DnsInfo200 } from './api/model/dnsInfo200';
import type { DNSConfigBlockingMode, DNSConfigUpstreamMode } from './api/model';
import type { FilterStatus } from './api/model/filterStatus';
import type { DhcpStaticLease } from './api/model/dhcpStaticLease';
import type { DhcpSearchResult } from './api/model/dhcpSearchResult';
import type { Stats } from './api/model/stats';
import type { GetStatsConfigResponse } from './api/model/getStatsConfigResponse';
import type { GetQueryLogConfigResponse } from './api/model/getQueryLogConfigResponse';
import type { RewriteEntry } from './api/model/rewriteEntry';
import type { RewriteSettings } from './api/model/rewriteSettings';
import type { BlockedServicesSchedule } from './api/model/blockedServicesSchedule';
import type { BlockedService } from './api/model/blockedService';
import type { ServiceGroup } from './api/model/serviceGroup';
import type { QueryLogFilter } from './helpers/constants';
import type { ToastNotice } from './stores/toasts';

export type InstallInterface = {
    flags: string;
    hardware_address: string;
    ip_addresses: string[];
    mtu: number;
    name: string;
};

export type InstallData = {
    step: number;
    processingDefault: boolean;
    processingSubmit: boolean;
    processingCheck: boolean;
    submitted: boolean;
    auth: {
        username: string;
        password: string;
        privacy_consent: boolean;
    };
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
    interfaces: InstallInterface[];
    dnsVersion: string;
};

export type EncryptionData = Partial<
    Omit<
        TlsConfig,
        'port_https' | 'port_dns_over_tls' | 'port_dns_over_quic' | 'port_dnscrypt' | 'dns_names'
    >
> & {
    // UI-only fields NOT in API model:
    processing: boolean;
    processingConfig: boolean;
    processingValidate: boolean;
    status_cert: string; // UI concatenation
    status_key: string; // UI concatenation
    allow_unencrypted_doh: boolean;
    // Port fields: number from API, string from form input (initialized as ''):
    port_https: number | string;
    port_dns_over_tls: number | string;
    port_dns_over_quic: number | string;
    port_dnscrypt: number | string;
    // Store initializes as null, API returns string[]:
    dns_names: string[] | null;
};

export { type WhoisInfo, type AutoClient, type Client };

export type DashboardData = {
    processing: boolean;
    isCoreRunning: boolean;
    processingVersion: boolean;
    processingClients: boolean;
    processingUpdate: boolean;
    processingProfile: boolean;
    protectionEnabled: boolean;
    protectionDisabledDuration: number | null;
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
        };
        safebrowsing: {
            enabled: boolean;
        };
        safesearch: Record<string, boolean>;
    };
};

export type RewritesData = RewriteSettings & {
    processing: boolean;
    processingAdd: boolean;
    processingDelete: boolean;
    processingUpdate: boolean;
    processingSettings: boolean;
    isModalOpen: boolean;
    modalType: string;
    currentRewrite?: RewriteEntry;
    list: RewriteEntry[];
};

export type NormalizedTopClients = {
    auto: Record<string, number>;
    configured: Record<string, number>;
};

export type StatsData = Omit<
    Stats,
    | 'top_queried_domains'
    | 'top_clients'
    | 'top_blocked_domains'
    | 'top_upstreams_responses'
    | 'top_upstreams_avg_time'
    | 'time_units'
> &
    Omit<GetStatsConfigResponse, 'interval'> & {
        processingGetConfig: boolean;
        processingSetConfig: boolean;
        processingStats: boolean;
        processingReset: boolean;
        interval: number;
        customInterval?: number | null;
        // Normalized top stats (from normalizeTopStats):
        topBlockedDomains: { name: string; count: number }[];
        topClients: { name: string; count: number; info: string }[]; // info is string!
        topQueriedDomains: { name: string; count: number }[];
        topUpstreamsAvgTime: { name: string; count: number }[];
        topUpstreamsResponses: { name: string; count: number }[];
        normalizedTopClients?: NormalizedTopClients;
        timeUnits: string;
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
    // Use generated DhcpSearchResult:
    check: DhcpSearchResult | null;
    // Keep inline v4/v6 (required — always present after init):
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
    // UI-normalized leases (flat without expires):
    leases: { hostname: string; ip: string; mac: string }[];
    staticLeases: DhcpStaticLease[];
    isModalOpen: boolean;
    leaseModalConfig?: { hostname: string; ip: string; mac: string };
    modalType: string;
    dhcp_available: boolean;
    interfaces?: DHCPNetInterfaces;
};

export type DnsConfigData = Omit<
    DnsInfo200,
    | 'upstream_dns'
    | 'fallback_dns'
    | 'bootstrap_dns'
    | 'local_ptr_upstreams'
    | 'ratelimit_whitelist'
    | 'blocking_mode'
    | 'upstream_mode'
    | 'protection_enabled'
    | 'protection_disabled_until'
> & {
    // UI-only processing flags:
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    // Normalized fields (string[] → newline-joined string):
    blocking_mode: DNSConfigBlockingMode;
    upstream_mode: DNSConfigUpstreamMode;
    upstream_dns: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    ratelimit_whitelist: string;
};

export type FilteringData = Omit<FilterStatus, 'filters' | 'whitelist_filters' | 'user_rules'> & {
    // UI-only fields:
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
    isFilterRemoved: boolean;
    isFilterEdited: boolean;
    modalType: string;
    modalFilterUrl: string;
    check: Record<string, unknown> | Record<string, never>;
    // Normalized fields (camelCase from normalizeFilteringStatus):
    filters: Filter[];
    whitelistFilters: Filter[]; // Note: whitelist (no underscore) — matches store
    userRules: string;
};

export type QueryLogsData = Omit<GetQueryLogConfigResponse, 'interval'> & {
    processingGetLogs: boolean;
    processingClear: boolean;
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    processingAdditionalLogs: boolean;
    interval: number;
    customInterval: number | null;
    logs: NormalizedQueryLogItem[];
    oldest: string;
    filter: QueryLogFilter;
    isFiltered: boolean;
    isDetailed: boolean;
    isEntireLog: boolean;
};

export type ServicesData = BlockedServicesSchedule & {
    processing: boolean;
    processingAll: boolean;
    processingSet: boolean;
    allServices: BlockedService[];
    allGroups: ServiceGroup[];
};

export type ModalsData = {
    modalId: ModalType | null;
};

export type ClientFormState = {
    mode: 'add' | 'edit';
    originalName: string;
    name: string;
    ids: string[];
    tags: string[];
    use_global_settings: boolean;
    filtering_enabled: boolean;
    safebrowsing_enabled: boolean;
    parental_enabled: boolean;
    safe_search: {
        enabled: boolean;
        google: boolean;
        youtube: boolean;
        bing: boolean;
        duckduckgo: boolean;
        yandex: boolean;
        pixabay: boolean;
        ecosia: boolean;
    };
    ignore_querylog: boolean;
    ignore_statistics: boolean;
    blocked_services: string[];
    use_global_blocked_services: boolean;
    blocked_services_schedule: {
        time_zone: string;
        sun?: { start: number; end: number };
        mon?: { start: number; end: number };
        tue?: { start: number; end: number };
        wed?: { start: number; end: number };
        thu?: { start: number; end: number };
        fri?: { start: number; end: number };
        sat?: { start: number; end: number };
    };
    upstreams: string;
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
    processingSave: boolean;
    formErrors: Record<string, string | string[]>;
};

export const getInitialClientFormState = (): ClientFormState => ({
    mode: 'add',
    originalName: '',
    name: '',
    ids: [''],
    tags: [],
    use_global_settings: false,
    filtering_enabled: false,
    safebrowsing_enabled: false,
    parental_enabled: false,
    safe_search: {
        enabled: false,
        google: false,
        youtube: false,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
        ecosia: false,
    },
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: [],
    use_global_blocked_services: false,
    blocked_services_schedule: {
        time_zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
    upstreams: '',
    upstreams_cache_enabled: false,
    upstreams_cache_size: DEFAULT_DNS_CACHE_SIZE,
    processingSave: false,
    formErrors: {},
});

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
    toasts: { notices: ToastNotice[] };
    modals: ModalsData;
    clientForm: ClientFormState;
};

export type InstallState = {
    install: InstallData;
    toasts: { notices: ToastNotice[] };
};

export type LoginState = {
    login: {
        processingLogin: boolean;
        email: string;
        password: string;
        error: unknown;
    };
    toasts: { notices: ToastNotice[] };
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
        upstream_timeout: 10,
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
        key_type: '' as TlsConfigKeyType,
        not_after: '',
        not_before: '',
        subject: '',
        valid_chain: false,
        valid_key: false,
        valid_cert: false,
        valid_pair: false,
        status_cert: '',
        status_key: '',
        allow_unencrypted_doh: false,
        certificate_chain: '',
        private_key: '',
        server_name: '',
        warning_validation: '',
        certificate_path: '',
        private_key_path: '',
        private_key_saved: false,
        port_https: '',
        port_dns_over_tls: '',
        port_dns_over_quic: '',
        port_dnscrypt: '',
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
        isFilterRemoved: false,
        isFilterEdited: false,
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
        ignored: [],
    },
    rewrites: {
        processing: true,
        processingAdd: false,
        processingDelete: false,
        processingUpdate: false,
        processingSettings: false,
        isModalOpen: false,
        modalType: '',
        list: [],
        enabled: true,
    },
    services: {
        processing: true,
        processingAll: true,
        processingSet: false,
        allServices: [],
        allGroups: [],
    } as ServicesData,
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
        dns_queries: [],
        blocked_filtering: [],
        replaced_parental: [],
        replaced_safebrowsing: [],
        topBlockedDomains: [],
        topClients: [],
        topQueriedDomains: [],
        num_blocked_filtering: 0,
        num_dns_queries: 0,
        num_replaced_parental: 0,
        num_replaced_safebrowsing: 0,
        num_replaced_safesearch: 0,
        avg_processing_time: 0,
        timeUnits: TIME_UNITS.HOURS,
        enabled: true,
        topUpstreamsAvgTime: [],
        topUpstreamsResponses: [],
        ignored: [],
    },
    toasts: { notices: [] },
    modals: { modalId: null },
    clientForm: getInitialClientFormState(),
};
