export const R_URL_REQUIRES_PROTOCOL = /^https?:\/\/[^/\s]+(\/.*)?$/;

// matches hostname or *.wildcard
export const R_HOST = /^(\*\.)?[\w.-]+$/;

export const R_IPV4 = /^(?:(?:^|\.)(?:2(?:5[0-5]|[0-4]\d)|1?\d?\d)){4}$/;

export const R_IPV6 = /^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$/;

export const R_CIDR = /^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2]))$/;

export const R_MAC = /^((([a-fA-F0-9][a-fA-F0-9]+[-:]){5})([a-fA-F0-9]{2})$)|^((([a-fA-F0-9][a-fA-F0-9]+[-:]){7})([a-fA-F0-9]{2})$)|^([a-fA-F0-9][a-fA-F0-9][a-fA-F0-9][a-fA-F0-9]+[.]){2}([a-fA-F0-9]{4})$|^([a-fA-F0-9][a-fA-F0-9][a-fA-F0-9][a-fA-F0-9]+[.]){3}([a-fA-F0-9]{4})$/;
export const R_MAC_WITHOUT_COLON = /^([a-fA-F0-9]{2}){5}([a-fA-F0-9]{2})$|^([a-fA-F0-9]{2}){7}([a-fA-F0-9]{2})$/;

export const R_CIDR_IPV6 = /^s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:)))(%.+)?s*(\/(12[0-8]|1[0-1][0-9]|[1-9][0-9]|[0-9]))$/;

export const R_DOMAIN = /^([a-zA-Z0-9][a-zA-Z0-9-_]*\.)*[a-zA-Z0-9]*[a-zA-Z0-9-_]*[[a-zA-Z0-9]+$/;

export const R_PATH_LAST_PART = /\/[^/]*$/;

// eslint-disable-next-line no-control-regex
export const R_UNIX_ABSOLUTE_PATH = /^(\/[^/\x00]+)+$/;

// eslint-disable-next-line no-control-regex
export const R_WIN_ABSOLUTE_PATH = /^([a-zA-Z]:)?(\\|\/)(?:[^\\/:*?"<>|\x00]+\\)*[^\\/:*?"<>|\x00]*$/;

export const R_CLIENT_ID = /^[a-z0-9-]{1,63}$/;

export const R_IPV4_SUBNET = /^([0-9]|[1-2][0-9]|3[0-2])?$/;

export const R_IPV6_SUBNET = /^([0-9]|[1-9][0-9]|1[0-1][0-9]|12[0-8])?$/;

export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 72;

export const HTML_PAGES = {
    INSTALL: '/install.html',
    LOGIN: '/login.html',
    MAIN: '/',
};

export const STATS_NAMES = {
    avg_processing_time: 'average_processing_time',
    blocked_filtering: 'Blocked by filters',
    dns_queries: 'DNS queries',
    replaced_parental: 'stats_adult',
    replaced_safebrowsing: 'stats_malware_phishing',
    replaced_safesearch: 'enforced_save_search',
};

export const STATUS_COLORS = {
    blue: '#467fcf',
    red: '#cd201f',
    green: '#5eba00',
    yellow: '#f1c40f',
};

export const REPOSITORY = {
    URL: 'https://github.com/AdguardTeam/AdGuardHome',
    TRACKERS_DB:
        'https://github.com/AdguardTeam/AdGuardHome/tree/master/client/src/helpers/trackers/trackers.json',
    ISSUES: 'https://github.com/AdguardTeam/AdGuardHome/issues/new/choose',
};

export const CLIENT_ID_LINK = 'https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#clientid';
export const MANUAL_UPDATE_LINK = 'https://github.com/AdguardTeam/AdGuardHome/wiki/FAQ#manual-update';
export const PORT_53_FAQ_LINK = 'https://github.com/AdguardTeam/AdGuardHome/wiki/FAQ#bindinuse';
export const PRIVACY_POLICY_LINK = 'https://link.adtidy.org/forward.html?action=privacy&from=ui&app=home';
export const UPSTREAM_CONFIGURATION_WIKI_LINK = 'https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams';

export const FILTERS_RELATIVE_LINK = '#filters';

export const ADDRESS_IN_USE_TEXT = 'address already in use';

export const INSTALL_FIRST_STEP = 1;
export const INSTALL_TOTAL_STEPS = 5;

export const SETTINGS_NAMES = {
    filtering: 'filtering',
    safebrowsing: 'safebrowsing',
    parental: 'parental',
    safesearch: 'safesearch',
};

export const STANDARD_DNS_PORT = 53;
export const STANDARD_WEB_PORT = 80;
export const STANDARD_HTTPS_PORT = 443;
export const DNS_OVER_TLS_PORT = 853;
export const DNS_OVER_QUIC_PORT = 853;
export const MAX_PORT = 65535;

export const EMPTY_DATE = '0001-01-01T00:00:00Z';

export const DEBOUNCE_TIMEOUT = 300;
export const DEBOUNCE_FILTER_TIMEOUT = 500;
export const CHECK_TIMEOUT = 1000;
export const HIDE_TOOLTIP_DELAY = 300;
export const SHOW_TOOLTIP_DELAY = 200;
export const MODAL_OPEN_TIMEOUT = 150;

export const UNSAFE_PORTS = [
    1,
    7,
    9,
    11,
    13,
    15,
    17,
    19,
    20,
    21,
    22,
    23,
    25,
    37,
    42,
    43,
    53,
    77,
    79,
    87,
    95,
    101,
    102,
    103,
    104,
    109,
    110,
    111,
    113,
    115,
    117,
    119,
    123,
    135,
    139,
    143,
    179,
    389,
    465,
    512,
    513,
    514,
    515,
    526,
    530,
    531,
    532,
    540,
    556,
    563,
    587,
    601,
    636,
    993,
    995,
    2049,
    3659,
    4045,
    6000,
    6665,
    6666,
    6667,
    6668,
    6669,
];

export const ALL_INTERFACES_IP = '0.0.0.0';

export const STATUS_RESPONSE = {
    YES: 'yes',
    NO: 'no',
    ERROR: 'error',
};

export const MODAL_TYPE = {
    SELECT_MODAL_TYPE: 'SELECT_MODAL_TYPE',
    ADD_FILTERS: 'ADD_FILTERS',
    EDIT_FILTERS: 'EDIT_FILTERS',
    CHOOSE_FILTERING_LIST: 'CHOOSE_FILTERING_LIST',
    ADD_REWRITE: 'ADD_REWRITE',
    EDIT_REWRITE: 'EDIT_REWRITE',
    EDIT_LEASE: 'EDIT_LEASE',
    ADD_LEASE: 'ADD_LEASE',
};

export const CLIENT_ID = {
    MAC: 'mac',
    IP: 'ip',
};

export const MENU_URLS = {
    root: '/',
    logs: '/logs',
    guide: '/guide',
};

export const SETTINGS_URLS = {
    encryption: '/encryption',
    dhcp: '/dhcp',
    dns: '/dns',
    settings: '/settings',
    clients: '/clients',
};

export const FILTERS_URLS = {
    dns_blocklists: '/filters',
    dns_allowlists: '/dns_allowlists',
    dns_rewrites: '/dns_rewrites',
    custom_rules: '/custom_rules',
    blocked_services: '/blocked_services',
};

export const ENCRYPTION_SOURCE = {
    PATH: 'path',
    CONTENT: 'content',
};

export const FILTERED = 'Filtered';
export const NOT_FILTERED = 'NotFiltered';

export const DISABLED_STATS_INTERVAL = 0;

export const HOUR = 60 * 60 * 1000;

export const DAY = HOUR * 24;

export const STATS_INTERVALS_DAYS = [DAY, DAY * 7, DAY * 30, DAY * 90];

export const QUERY_LOG_INTERVALS_DAYS = [HOUR * 6, DAY, DAY * 7, DAY * 30, DAY * 90];

export const RETENTION_CUSTOM = 1;

export const RETENTION_CUSTOM_INPUT = 'custom_retention_input';

export const CUSTOM_INTERVAL = 'customInterval';

export const FILTERS_INTERVALS_HOURS = [0, 1, 12, 24, 72, 168];

// Note that translation strings contain these modes (blocking_mode_CONSTANT)
// i.e. blocking_mode_default, blocking_mode_null_ip
export const BLOCKING_MODES = {
    default: 'default',
    refused: 'refused',
    nxdomain: 'nxdomain',
    null_ip: 'null_ip',
    custom_ip: 'custom_ip',
};

// Note that translation strings contain these modes (theme_CONSTANT)
// i.e. theme_auto, theme_light.
export const THEMES = {
    auto: 'auto',
    dark: 'dark',
    light: 'light',
};

export const WHOIS_ICONS = {
    location: 'location',
    orgname: 'network',
    netname: 'network',
    descr: '',
};

export const DEFAULT_LOGS_FILTER = {
    search: '',
    response_status: '',
};

export const DEFAULT_LANGUAGE = 'en';

export const QUERY_LOGS_PAGE_LIMIT = 20;

export const LEASES_TABLE_DEFAULT_PAGE_SIZE = 20;

export const FILTERED_STATUS = {
    FILTERED_BLACK_LIST: 'FilteredBlackList',
    NOT_FILTERED_WHITE_LIST: 'NotFilteredWhiteList',
    NOT_FILTERED_NOT_FOUND: 'NotFilteredNotFound',
    FILTERED_BLOCKED_SERVICE: 'FilteredBlockedService',
    REWRITE: 'Rewrite',
    REWRITE_HOSTS: 'RewriteEtcHosts',
    REWRITE_RULE: 'RewriteRule',
    FILTERED_SAFE_SEARCH: 'FilteredSafeSearch',
    FILTERED_SAFE_BROWSING: 'FilteredSafeBrowsing',
    FILTERED_PARENTAL: 'FilteredParental',
};

export const RESPONSE_FILTER = {
    ALL: {
        QUERY: 'all',
        LABEL: 'all_queries',
    },
    FILTERED: {
        QUERY: 'filtered',
        LABEL: 'filtered',
    },
    PROCESSED: {
        QUERY: 'processed',
        LABEL: 'show_processed_responses',
    },
    BLOCKED: {
        QUERY: 'blocked',
        LABEL: 'show_blocked_responses',
    },
    BLOCKED_SERVICES: {
        QUERY: 'blocked_services',
        LABEL: 'blocked_services',
    },
    BLOCKED_THREATS: {
        QUERY: 'blocked_safebrowsing',
        LABEL: 'blocked_threats',
    },
    BLOCKED_ADULT_WEBSITES: {
        QUERY: 'blocked_parental',
        LABEL: 'blocked_adult_websites',
    },
    ALLOWED: {
        QUERY: 'whitelisted',
        LABEL: 'allowed',
    },
    REWRITTEN: {
        QUERY: 'rewritten',
        LABEL: 'rewritten',
    },
    SAFE_SEARCH: {
        QUERY: 'safe_search',
        LABEL: 'safe_search',
    },
};

export const RESPONSE_FILTER_QUERIES = Object.values(RESPONSE_FILTER)
    .reduce((acc, { QUERY }) => {
        acc[QUERY] = QUERY;
        return acc;
    }, {});

export const QUERY_STATUS_COLORS = {
    BLUE: 'blue',
    GREEN: 'green',
    RED: 'red',
    WHITE: 'white',
    YELLOW: 'yellow',
};

export const FILTERED_STATUS_TO_META_MAP = {
    [FILTERED_STATUS.NOT_FILTERED_WHITE_LIST]: {
        LABEL: RESPONSE_FILTER.ALLOWED.LABEL,
        COLOR: QUERY_STATUS_COLORS.GREEN,
    },
    [FILTERED_STATUS.NOT_FILTERED_NOT_FOUND]: {
        LABEL: RESPONSE_FILTER.PROCESSED.LABEL,
        COLOR: QUERY_STATUS_COLORS.WHITE,
    },
    [FILTERED_STATUS.FILTERED_BLOCKED_SERVICE]: {
        LABEL: 'blocked_service',
        COLOR: QUERY_STATUS_COLORS.RED,
    },
    [FILTERED_STATUS.FILTERED_SAFE_SEARCH]: {
        LABEL: RESPONSE_FILTER.SAFE_SEARCH.LABEL,
        COLOR: QUERY_STATUS_COLORS.YELLOW,
    },
    [FILTERED_STATUS.FILTERED_BLACK_LIST]: {
        LABEL: RESPONSE_FILTER.BLOCKED.LABEL,
        COLOR: QUERY_STATUS_COLORS.RED,
    },
    [FILTERED_STATUS.REWRITE]: {
        LABEL: RESPONSE_FILTER.REWRITTEN.LABEL,
        COLOR: QUERY_STATUS_COLORS.BLUE,
    },
    [FILTERED_STATUS.REWRITE_HOSTS]: {
        LABEL: RESPONSE_FILTER.REWRITTEN.LABEL,
        COLOR: QUERY_STATUS_COLORS.BLUE,
    },
    [FILTERED_STATUS.REWRITE_RULE]: {
        LABEL: RESPONSE_FILTER.REWRITTEN.LABEL,
        COLOR: QUERY_STATUS_COLORS.BLUE,
    },
    [FILTERED_STATUS.FILTERED_SAFE_BROWSING]: {
        LABEL: RESPONSE_FILTER.BLOCKED_THREATS.LABEL,
        COLOR: QUERY_STATUS_COLORS.YELLOW,
    },
    [FILTERED_STATUS.FILTERED_PARENTAL]: {
        LABEL: RESPONSE_FILTER.BLOCKED_ADULT_WEBSITES.LABEL,
        COLOR: QUERY_STATUS_COLORS.YELLOW,
    },
};

export const DEFAULT_TIME_FORMAT = 'HH:mm:ss';

export const LONG_TIME_FORMAT = 'HH:mm:ss.SSS';

export const DEFAULT_SHORT_DATE_FORMAT_OPTIONS = {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour12: false,
};

export const DEFAULT_DATE_FORMAT_OPTIONS = {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: 'numeric',
    hourCycle: 'h23',
    minute: 'numeric',
};

export const DETAILED_DATE_FORMAT_OPTIONS = {
    ...DEFAULT_DATE_FORMAT_OPTIONS,
    month: 'long',
};

export const SPECIAL_FILTER_ID = {
    CUSTOM_FILTERING_RULES: 0,
    SYSTEM_HOSTS: -1,
    BLOCKED_SERVICES: -2,
    PARENTAL: -3,
    SAFE_BROWSING: -4,
    SAFE_SEARCH: -5,
};

export const BLOCK_ACTIONS = {
    BLOCK: 'block',
    UNBLOCK: 'unblock',
};

export const SCHEME_TO_PROTOCOL_MAP = {
    dnscrypt: 'dnscrypt',
    doh: 'dns_over_https',
    dot: 'dns_over_tls',
    doq: 'dns_over_quic',
    '': 'plain_dns',
};

export const DNS_REQUEST_OPTIONS = {
    PARALLEL: 'parallel',
    FASTEST_ADDR: 'fastest_addr',
    LOAD_BALANCING: '',
};

export const DHCP_FORM_NAMES = {
    DHCPv4: 'dhcpv4',
    DHCPv6: 'dhcpv6',
    DHCP_INTERFACES: 'dhcpInterfaces',
};

export const FORM_NAME = {
    UPSTREAM: 'upstream',
    DOMAIN_CHECK: 'domainCheck',
    FILTER: 'filter',
    REWRITES: 'rewrites',
    LOGS_FILTER: 'logsFilter',
    CLIENT: 'client',
    LEASE: 'lease',
    ACCESS: 'access',
    BLOCKING_MODE: 'blockingMode',
    ENCRYPTION: 'encryption',
    FILTER_CONFIG: 'filterConfig',
    LOG_CONFIG: 'logConfig',
    SERVICES: 'services',
    STATS_CONFIG: 'statsConfig',
    INSTALL: 'install',
    LOGIN: 'login',
    CACHE: 'cache',
    MOBILE_CONFIG: 'mobileConfig',
    ...DHCP_FORM_NAMES,
};

export const SMALL_SCREEN_SIZE = 767;
export const MEDIUM_SCREEN_SIZE = 1024;

export const SECONDS_IN_DAY = 60 * 60 * 24;

export const UINT32_RANGE = {
    MIN: 0,
    MAX: 4294967295,
};

export const RETENTION_RANGE = {
    MIN: 1,
    MAX: 365 * 24,
};

export const DHCP_VALUES_PLACEHOLDERS = {
    ipv4: {
        subnet_mask: '255.255.255.0',
        lease_duration: SECONDS_IN_DAY.toString(),
    },
    ipv6: {
        range_start: '2001::1',
        range_end: 'ff',
        lease_duration: SECONDS_IN_DAY.toString(),
    },
};

export const DHCP_DESCRIPTION_PLACEHOLDERS = {
    ipv4: {
        gateway_ip: 'dhcp_form_gateway_input',
        subnet_mask: 'dhcp_form_subnet_input',
        range_start: 'dhcp_form_range_start',
        range_end: 'dhcp_form_range_end',
        lease_duration: 'dhcp_form_lease_input',
    },
    ipv6: {
        range_start: 'dhcp_form_range_start',
        range_end: 'dhcp_form_range_end',
        lease_duration: 'dhcp_form_lease_input',
    },
};

export const TOAST_TRANSITION_TIMEOUT = 500;

export const TOAST_TYPES = {
    SUCCESS: 'success',
    ERROR: 'error',
    NOTICE: 'notice',
};

export const SUCCESS_TOAST_TIMEOUT = 5000;

export const ONE_SECOND_IN_MS = 1000;
export const FAILURE_TOAST_TIMEOUT = 30000;

export const TOAST_TIMEOUTS = {
    [TOAST_TYPES.SUCCESS]: SUCCESS_TOAST_TIMEOUT,
    [TOAST_TYPES.ERROR]: FAILURE_TOAST_TIMEOUT,
    [TOAST_TYPES.NOTICE]: FAILURE_TOAST_TIMEOUT,
};

export const ADDRESS_TYPES = {
    IP: 'IP',
    CIDR: 'CIDR',
    CLIENT_ID: 'CLIENT_ID',
    UNKNOWN: 'UNKNOWN',
};

export const CACHE_CONFIG_FIELDS = {
    cache_size: 'cache_size',
    cache_ttl_min: 'cache_ttl_min',
    cache_ttl_max: 'cache_ttl_max',
};

export const isFirefox = navigator.userAgent.indexOf('Firefox') !== -1;
export const COMMENT_LINE_DEFAULT_TOKEN = '#';

export const MOBILE_CONFIG_LINKS = {
    DOT: 'apple/dot.mobileconfig',
    DOH: 'apple/doh.mobileconfig',
};

// Timings for disable protection in milliseconds
export const DISABLE_PROTECTION_TIMINGS = {
    HALF_MINUTE: 30 * 1000,
    MINUTE: 60 * 1000,
    TEN_MINUTES: 10 * 60 * 1000,
    HOUR: 60 * 60 * 1000,
    TOMORROW: 24 * 60 * 60 * 1000,
};

export const LOCAL_TIMEZONE_VALUE = 'Local';

export const TABLES_MIN_ROWS = 5;

export const DASHBOARD_TABLES_DEFAULT_PAGE_SIZE = 100;

export const TIME_UNITS = {
    HOURS: 'hours',
    DAYS: 'days',
};
