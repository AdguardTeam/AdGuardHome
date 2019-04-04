export const R_URL_REQUIRES_PROTOCOL = /^https?:\/\/\w[\w_\-.]*\.[a-z]{2,8}[^\s]*$/;
export const R_IPV4 = /^(?:(?:^|\.)(?:2(?:5[0-5]|[0-4]\d)|1?\d?\d)){4}$/g;

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
    TRACKERS_DB: 'https://github.com/AdguardTeam/AdGuardHome/tree/master/client/src/helpers/trackers/adguard.json',
};

export const LANGUAGES = [
    {
        key: 'en',
        name: 'English',
    },
    {
        key: 'es',
        name: 'Español',
    },
    {
        key: 'fr',
        name: 'Français',
    },
    {
        key: 'pt-br',
        name: 'Português (BR)',
    },
    {
        key: 'sv',
        name: 'Svenska',
    },
    {
        key: 'vi',
        name: 'Tiếng Việt',
    },
    {
        key: 'bg',
        name: 'Български',
    },
    {
        key: 'ru',
        name: 'Русский',
    },
    {
        key: 'ja',
        name: '日本語',
    },
    {
        key: 'zh-tw',
        name: '正體中文',
    },
    {
        key: 'zh-cn',
        name: '简体中文',
    },
];

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
export const CHECK_TIMEOUT = 1000;
export const STOP_TIMEOUT = 10000;

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

export const RESPONSE_STATUS = {
    YES: 'yes',
    NO: 'no',
    ERROR: 'error',
};
