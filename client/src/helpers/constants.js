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
