export const R_URL_REQUIRES_PROTOCOL = /^https?:\/\/[^/\s]+(\/.*)?$/;
export const R_HOST = /^(\*\.)?([\w-]+\.)+[\w-]+$/;
export const R_IPV4 = /^(?:(?:^|\.)(?:2(?:5[0-5]|[0-4]\d)|1?\d?\d)){4}$/;
export const R_IPV6 = /^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$/;
export const R_CIDR = /^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2]))$/;
export const R_MAC = /^((([a-fA-F0-9][a-fA-F0-9]+[-]){5}|([a-fA-F0-9][a-fA-F0-9]+[:]){5})([a-fA-F0-9][a-fA-F0-9])$)|(^([a-fA-F0-9][a-fA-F0-9][a-fA-F0-9][a-fA-F0-9]+[.]){2}([a-fA-F0-9][a-fA-F0-9][a-fA-F0-9][a-fA-F0-9]))$/;
export const R_CIDR_IPV6 = /^s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:)))(%.+)?s*(\/(12[0-8]|1[0-1][0-9]|[1-9][0-9]|[0-9]))$/;
export const R_PATH_LAST_PART = /\/[^/]*$/;
// eslint-disable-next-line no-control-regex
export const R_UNIX_ABSOLUTE_PATH = /^(\/[^/\x00]+)+$/;
// eslint-disable-next-line no-control-regex
export const R_WIN_ABSOLUTE_PATH = /^([a-zA-Z]:)?(\\|\/)(?:[^\\/:*?"<>|\x00]+\\)*[^\\/:*?"<>|\x00]*$/;

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
        'https://github.com/AdguardTeam/AdGuardHome/tree/master/client/src/helpers/trackers/adguard.json',
    ISSUES: 'https://github.com/AdguardTeam/AdGuardHome/issues/new/choose',
};

export const PRIVACY_POLICY_LINK = 'https://adguard.com/privacy/home.html';

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

export const EMPTY_DATE = '0001-01-01T00:00:00Z';

export const DEBOUNCE_TIMEOUT = 300;
export const DEBOUNCE_FILTER_TIMEOUT = 500;
export const CHECK_TIMEOUT = 1000;
export const SUCCESS_TOAST_TIMEOUT = 5000;
export const FAILURE_TOAST_TIMEOUT = 30000;
export const HIDE_TOOLTIP_DELAY = 300;
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

export const DHCP_STATUS_RESPONSE = {
    YES: 'yes',
    NO: 'no',
    ERROR: 'error',
};

export const MODAL_TYPE = {
    SELECT_MODAL_TYPE: 'SELECT_MODAL_TYPE',
    ADD_FILTERS: 'ADD_FILTERS',
    EDIT_FILTERS: 'EDIT_FILTERS',
    CHOOSE_FILTERING_LIST: 'CHOOSE_FILTERING_LIST',
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

export const SERVICES = [
    {
        id: 'facebook',
        name: 'Facebook',
    },
    {
        id: 'whatsapp',
        name: 'WhatsApp',
    },
    {
        id: 'instagram',
        name: 'Instagram',
    },
    {
        id: 'twitter',
        name: 'Twitter',
    },
    {
        id: 'youtube',
        name: 'YouTube',
    },
    {
        id: 'netflix',
        name: 'Netflix',
    },
    {
        id: 'snapchat',
        name: 'Snapchat',
    },
    {
        id: 'twitch',
        name: 'Twitch',
    },
    {
        id: 'discord',
        name: 'Discord',
    },
    {
        id: 'skype',
        name: 'Skype',
    },
    {
        id: 'amazon',
        name: 'Amazon',
    },
    {
        id: 'ebay',
        name: 'eBay',
    },
    {
        id: 'origin',
        name: 'Origin',
    },
    {
        id: 'cloudflare',
        name: 'Cloudflare',
    },
    {
        id: 'steam',
        name: 'Steam',
    },
    {
        id: 'epic_games',
        name: 'Epic Games',
    },
    {
        id: 'reddit',
        name: 'Reddit',
    },
    {
        id: 'ok',
        name: 'OK',
    },
    {
        id: 'vk',
        name: 'VK',
    },
    {
        id: 'mail_ru',
        name: 'mail.ru',
    },
    {
        id: 'tiktok',
        name: 'TikTok',
    },
];

export const ENCRYPTION_SOURCE = {
    PATH: 'path',
    CONTENT: 'content',
};

export const FILTERED = 'Filtered';
export const NOT_FILTERED = 'NotFiltered';

export const STATS_INTERVALS_DAYS = [1, 7, 30, 90];

export const QUERY_LOG_INTERVALS_DAYS = [1, 7, 30, 90];

export const FILTERS_INTERVALS_HOURS = [0, 1, 12, 24, 72, 168];

export const BLOCKING_MODES = {
    default: 'default',
    nxdomain: 'nxdomain',
    null_ip: 'null_ip',
    custom_ip: 'custom_ip',
};

export const WHOIS_ICONS = {
    location: 'location',
    orgname: 'network',
    netname: 'network',
    descr: '',
};

export const DNS_RECORD_TYPES = [
    'A',
    'AAAA',
    'AFSDB',
    'APL',
    'CAA',
    'CDNSKEY',
    'CDS',
    'CERT',
    'CNAME',
    'CSYNC',
    'DHCID',
    'DLV',
    'DNAME',
    'DNSKEY',
    'DS',
    'HIP',
    'IPSECKEY',
    'KEY',
    'KX',
    'LOC',
    'MX',
    'NAPTR',
    'NS',
    'NSEC',
    'NSEC3',
    'NSEC3PARAM',
    'OPENPGPKEY',
    'PTR',
    'RRSIG',
    'RP',
    'SIG',
    'SMIMEA',
    'SOA',
    'SRV',
    'SSHFP',
    'TA',
    'TKEY',
    'TLSA',
    'TSIG',
    'TXT',
    'URI',
];

export const DEFAULT_LOGS_FILTER = {
    search: '',
    response_status: '',
};

export const DEFAULT_LANGUAGE = 'en';

export const TABLE_DEFAULT_PAGE_SIZE = 50;

export const TABLE_FIRST_PAGE = 0;

export const LEASES_TABLE_DEFAULT_PAGE_SIZE = 20;

export const FILTERED_STATUS = {
    FILTERED_BLACK_LIST: 'FilteredBlackList',
    NOT_FILTERED_WHITE_LIST: 'NotFilteredWhiteList',
    NOT_FILTERED_NOT_FOUND: 'NotFilteredNotFound',
    FILTERED_BLOCKED_SERVICE: 'FilteredBlockedService',
    REWRITE: 'Rewrite',
    REWRITE_HOSTS: 'RewriteEtcHosts',
    FILTERED_SAFE_SEARCH: 'FilteredSafeSearch',
    FILTERED_SAFE_BROWSING: 'FilteredSafeBrowsing',
    FILTERED_PARENTAL: 'FilteredParental',
};

export const RESPONSE_FILTER = {
    ALL: {
        query: 'all',
        label: 'show_all_responses',
    },
    FILTERED: {
        query: 'filtered',
        label: 'filtered',
    },
    PROCESSED: {
        query: 'processed',
        label: 'show_processed_responses',
    },
    SPACE: {
        query: 'all',
        label: '',
        disabled: true,
    },
    BLOCKED: {
        query: 'blocked',
        label: 'show_blocked_responses',
    },
    BLOCKED_THREATS: {
        query: 'blocked_safebrowsing',
        label: 'blocked_threats',
    },
    BLOCKED_ADULT_WEBSITES: {
        query: 'blocked_parental',
        label: 'blocked_adult_websites',
    },
    ALLOWED: {
        query: 'whitelisted',
        label: 'allowed',
    },
    REWRITTEN: {
        query: 'rewritten',
        label: 'rewritten',
    },
    SAFE_SEARCH: {
        query: 'safe_search',
        label: 'safe_search',
    },
};

export const RESPONSE_FILTER_QUERIES = Object.values(RESPONSE_FILTER).reduce((acc, { query }) => {
    acc[query] = query;
    return acc;
}, {});

export const FILTERED_STATUS_TO_META_MAP = {
    [FILTERED_STATUS.NOT_FILTERED_WHITE_LIST]: {
        label: RESPONSE_FILTER.ALLOWED.label,
        color: 'green',
    },
    [FILTERED_STATUS.NOT_FILTERED_NOT_FOUND]: {
        label: RESPONSE_FILTER.PROCESSED.label,
        color: 'white',
    },
    [FILTERED_STATUS.FILTERED_BLOCKED_SERVICE]: {
        label: RESPONSE_FILTER.BLOCKED.label,
        color: 'red',
    },
    [FILTERED_STATUS.FILTERED_SAFE_SEARCH]: {
        label: RESPONSE_FILTER.SAFE_SEARCH.label,
        color: 'yellow',
    },
    [FILTERED_STATUS.FILTERED_BLACK_LIST]: {
        label: RESPONSE_FILTER.BLOCKED.label,
        color: 'red',
    },
    [FILTERED_STATUS.REWRITE]: {
        label: RESPONSE_FILTER.REWRITTEN.label,
        color: 'blue',
    },
    [FILTERED_STATUS.REWRITE_HOSTS]: {
        label: RESPONSE_FILTER.REWRITTEN.label,
        color: 'blue',
    },
    [FILTERED_STATUS.FILTERED_SAFE_BROWSING]: {
        label: RESPONSE_FILTER.BLOCKED_THREATS.label,
        color: 'yellow',
    },
    [FILTERED_STATUS.FILTERED_PARENTAL]: {
        label: RESPONSE_FILTER.BLOCKED_ADULT_WEBSITES.label,
        color: 'yellow',
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
    minute: 'numeric',
    hour12: false,
};

export const DETAILED_DATE_FORMAT_OPTIONS = {
    ...DEFAULT_DATE_FORMAT_OPTIONS,
    month: 'long',
};

export const CUSTOM_FILTERING_RULES_ID = 0;

export const BLOCK_ACTIONS = {
    BLOCK: 'block',
    UNBLOCK: 'unblock',
};

export const SCHEME_TO_PROTOCOL_MAP = {
    doh: 'dns_over_https',
    dot: 'dns_over_tls',
    '': 'plain_dns',
};

export const DNS_REQUEST_OPTIONS = {
    PARALLEL: 'parallel',
    FASTEST_ADDR: 'fastest_addr',
    LOAD_BALANCING: '',
};

export const IP_MATCH_LIST_STATUS = {
    NOT_FOUND: 'NOT_FOUND', // not found in the list
    EXACT: 'EXACT', // found exact match (including the match of short and long forms)
    CIDR: 'CIDR', // the ip is in the specified CIDR range
};

export const FORM_NAME = {
    UPSTREAM: 'upstream',
    DOMAIN_CHECK: 'domainCheck',
    FILTER: 'filter',
    REWRITES: 'rewrites',
    LOGS_FILTER: 'logsFilter',
    CLIENT: 'client',
    DHCP: 'dhcp',
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
};

export const smallScreenSize = 767;

export const SECONDS_IN_HOUR = 60 * 60;
