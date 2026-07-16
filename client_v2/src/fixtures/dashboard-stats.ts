/**
 * Dashboard / Stats test data fixtures.
 *
 * Covers all possible states of the `.statContainer` tables:
 *   - GeneralStatistics  (6 stat rows + EmptyState when no queries)
 *   - TopClients          (sortable table, client WHOIS info, block/unblock)
 *   - TopQueriedDomains   (sortable table, tracker tooltips)
 *   - TopBlockedDomains   (sortable table)
 *   - TopUpstreams        (sortable table)
 *   - UpstreamAvgTime     (sortable table, response time in ms)
 *
 * All fixtures are in **raw backend API format** (what `GET /control/stats`
 * returns).  The frontend normalises `top_*` maps → `{name, count}[]` arrays
 * and converts `avg_processing_time` / `top_upstreams_avg_time` from seconds
 * to milliseconds.
 */

// ---------------------------------------------------------------------------
// 0.  Stats config responses  (GET /control/stats/config)
// ---------------------------------------------------------------------------

export interface StatsConfigResponse {
    interval: number; // ms, e.g. 86400000 = 1 day
    enabled: boolean;
    ignored: string[];
    ignored_enabled: boolean;
}

/** Default config: stats enabled, 90-day retention. */
export const STATS_CONFIG_DEFAULT: StatsConfigResponse = {
    interval: 7776000000, // 90 days in ms
    enabled: true,
    ignored: [],
    ignored_enabled: false,
};

/** Stats explicitly disabled — triggers EmptyState "disabled" mode. */
export const STATS_CONFIG_DISABLED: StatsConfigResponse = {
    interval: 86400000, // 1 day
    enabled: false,
    ignored: [],
    ignored_enabled: false,
};

// ---------------------------------------------------------------------------
// 1.  Backend raw stats response shape
// ---------------------------------------------------------------------------

export interface BackendTopAddr {
    [domainOrIp: string]: number;
}

export interface StatsResponse {
    time_units: 'hours' | 'days';
    top_queried_domains: BackendTopAddr[];
    top_clients: BackendTopAddr[];
    top_blocked_domains: BackendTopAddr[];
    top_upstreams_responses: BackendTopAddr[];
    top_upstreams_avg_time: Array<{ [upstream: string]: number }>; // seconds (float)
    dns_queries: number[];
    blocked_filtering: number[];
    replaced_safebrowsing: number[];
    replaced_parental: number[];
    num_dns_queries: number;
    num_blocked_filtering: number;
    num_replaced_safebrowsing: number;
    num_replaced_safesearch: number;
    num_replaced_parental: number;
    avg_processing_time: number; // seconds (float)
}

// ---------------------------------------------------------------------------
// 2.  Search clients response  (GET /control/clients/search)
// ---------------------------------------------------------------------------

export interface SearchClientEntry {
    [ipOrMac: string]: {
        name?: string;
        whois_info?: {
            orgname?: string;
            country?: string;
        };
        disallowed?: boolean;
    };
}

export interface SearchClientsResponse {
    clients: SearchClientEntry[];
    auto_clients: SearchClientEntry[];
}

// ---------------------------------------------------------------------------
// 3.  Access list response  (GET /control/access/list)
// ---------------------------------------------------------------------------

export interface AccessListResponse {
    allowed_clients: string[];
    disallowed_clients: string[];
    blocked_hosts: string[];
}

/** Empty access list — no clients blocked/unblocked. */
export const ACCESS_LIST_EMPTY: AccessListResponse = {
    allowed_clients: [],
    disallowed_clients: [],
    blocked_hosts: [],
};

// ===========================================================================
// SCENARIO A — EMPTY / NO DATA YET
// ===========================================================================

/**
 * Stats are enabled but no queries have been recorded yet.
 * - GeneralStatistics shows `<EmptyState />` (no queries)
 * - All top-* tables show `<EmptyState />` (no data)
 */
export const STATS_EMPTY: StatsResponse = {
    time_units: 'hours',
    top_queried_domains: [],
    top_clients: [],
    top_blocked_domains: [],
    top_upstreams_responses: [],
    top_upstreams_avg_time: [],
    dns_queries: [],
    blocked_filtering: [],
    replaced_safebrowsing: [],
    replaced_parental: [],
    num_dns_queries: 0,
    num_blocked_filtering: 0,
    num_replaced_safebrowsing: 0,
    num_replaced_safesearch: 0,
    num_replaced_parental: 0,
    avg_processing_time: 0,
};

/** Empty search clients response. */
export const SEARCH_CLIENTS_EMPTY: SearchClientsResponse = {
    clients: [],
    auto_clients: [],
};

// ===========================================================================
// SCENARIO B — NORMAL, SMALL DATA  (2–4 entries per category)
// ===========================================================================

export const STATS_NORMAL_SMALL: StatsResponse = {
    time_units: 'hours',
    top_queried_domains: [
        { 'google.com': 4521 },
        { 'youtube.com': 3210 },
        { 'facebook.com': 1890 },
    ],
    top_clients: [{ '192.168.1.10': 3500 }, { '192.168.1.20': 2800 }, { '192.168.1.30': 1500 }],
    top_blocked_domains: [{ 'doubleclick.net': 1230 }, { 'googletagmanager.com': 890 }],
    top_upstreams_responses: [{ '8.8.8.8:53': 6000 }, { '1.1.1.1:53': 3200 }],
    top_upstreams_avg_time: [{ '8.8.8.8:53': 0.012 }, { '1.1.1.1:53': 0.008 }],
    dns_queries: [120, 340, 560, 200],
    blocked_filtering: [15, 42, 78, 30],
    replaced_safebrowsing: [2, 5, 8, 3],
    replaced_parental: [0, 1, 0, 2],
    num_dns_queries: 1220,
    num_blocked_filtering: 165,
    num_replaced_safebrowsing: 18,
    num_replaced_safesearch: 5,
    num_replaced_parental: 3,
    avg_processing_time: 0.01,
};

export const SEARCH_CLIENTS_SMALL: SearchClientsResponse = {
    clients: [],
    auto_clients: [
        {
            '192.168.1.10': {
                name: 'Office Desktop',
                whois_info: { orgname: 'ACME Corp', country: 'US' },
            },
        },
        {
            '192.168.1.20': {
                name: "John's MacBook Pro",
                whois_info: { orgname: '', country: 'GB' },
            },
        },
    ],
};

// ===========================================================================
// SCENARIO C — FULL DATA  (10+ entries per category, covers sorting/pagination)
// ===========================================================================

export const STATS_NORMAL_FULL: StatsResponse = {
    time_units: 'days',
    top_queried_domains: [
        { 'google.com': 52341 },
        { 'youtube.com': 41230 },
        { 'facebook.com': 28901 },
        { 'github.com': 18234 },
        { 'stackoverflow.com': 15230 },
        { 'reddit.com': 14567 },
        { 'twitter.com': 12340 },
        { 'wikipedia.org': 11023 },
        { 'amazon.com': 9876 },
        { 'netflix.com': 8765 },
        { 'linkedin.com': 7654 },
        { 'microsoft.com': 6543 },
    ],
    top_clients: [
        { '192.168.1.10': 45000 },
        { '192.168.1.20': 38000 },
        { '192.168.1.30': 29000 },
        { '192.168.1.40': 22000 },
        { '192.168.1.50': 18000 },
        { 'AA:BB:CC:DD:EE:01': 15000 },
        { 'AA:BB:CC:DD:EE:02': 12000 },
        { '10.0.0.5': 9500 },
        { '10.0.0.6': 7200 },
        { '10.0.0.7': 5100 },
        { '10.0.0.8': 3400 },
        { '10.0.0.9': 2100 },
    ],
    top_blocked_domains: [
        { 'doubleclick.net': 15230 },
        { 'googletagmanager.com': 12340 },
        { 'googleadservices.com': 9870 },
        { 'ads.example.com': 7650 },
        { 'tracker.malware.com': 5430 },
        { 'banner.ads.net': 4320 },
        { 'pixel.tracker.io': 3210 },
        { 'analytics.spy.com': 2100 },
        { 'ad.doubleclick.net': 1890 },
        { 'stats.track.org': 1560 },
        { 'beacon.metric.co': 1230 },
        { 'telemetry.evil.io': 890 },
    ],
    top_upstreams_responses: [
        { '8.8.8.8:53': 50000 },
        { '1.1.1.1:53': 42000 },
        { '9.9.9.9:53': 28000 },
        { '208.67.222.222:53': 19000 },
        { 'tls://dns.google': 12000 },
        { 'https://dns.cloudflare.com/dns-query': 8500 },
        { 'quic://dns.adguard.com': 5200 },
        { 'tcp://8.8.4.4:53': 3100 },
        { 'tls://1.0.0.1': 1800 },
        { '[/::1]:53': 900 },
    ],
    top_upstreams_avg_time: [
        { '8.8.8.8:53': 0.032 },
        { '1.1.1.1:53': 0.018 },
        { '9.9.9.9:53': 0.045 },
        { '208.67.222.222:53': 0.067 },
        { 'tls://dns.google': 0.089 },
        { 'https://dns.cloudflare.com/dns-query': 0.105 },
        { 'quic://dns.adguard.com': 0.012 },
        { 'tcp://8.8.4.4:53': 0.156 },
        { 'tls://1.0.0.1': 0.078 },
        { '[/::1]:53': 0.003 },
    ],
    dns_queries: Array.from({ length: 24 }, () => 5000 + Math.floor(Math.random() * 3000)),
    blocked_filtering: Array.from({ length: 24 }, () => 500 + Math.floor(Math.random() * 400)),
    replaced_safebrowsing: Array.from({ length: 24 }, () => 20 + Math.floor(Math.random() * 30)),
    replaced_parental: Array.from({ length: 24 }, () => 5 + Math.floor(Math.random() * 10)),
    num_dns_queries: 156780,
    num_blocked_filtering: 23450,
    num_replaced_safebrowsing: 890,
    num_replaced_safesearch: 345,
    num_replaced_parental: 120,
    avg_processing_time: 0.025,
};

export const SEARCH_CLIENTS_FULL: SearchClientsResponse = {
    clients: [],
    auto_clients: [
        {
            '192.168.1.10': {
                name: 'Office Desktop',
                whois_info: { orgname: 'ACME Corporation', country: 'US' },
            },
        },
        {
            '192.168.1.20': {
                name: "John's MacBook Pro",
                whois_info: { orgname: '', country: 'GB' },
            },
        },
        {
            '192.168.1.30': {
                name: 'Living Room TV',
                whois_info: { orgname: 'Samsung Electronics', country: 'KR' },
            },
        },
        {
            '192.168.1.40': {
                name: 'Nest Thermostat',
                whois_info: { orgname: 'Google LLC', country: 'US' },
            },
        },
        {
            '192.168.1.50': {
                name: 'iPhone 15 Pro',
                whois_info: { orgname: '', country: '' },
            },
        },
        { 'AA:BB:CC:DD:EE:01': { name: 'Smart Fridge', whois_info: {} } },
        {
            'AA:BB:CC:DD:EE:02': {
                name: 'Ring Doorbell',
                whois_info: { orgname: 'Amazon', country: 'US' },
            },
        },
        { '10.0.0.5': { name: 'Guest Phone', whois_info: {} } },
        {
            '10.0.0.6': {
                name: 'Printer HP LaserJet',
                whois_info: { orgname: 'HP Inc.', country: 'US' },
            },
        },
        {
            '10.0.0.7': {
                name: 'Xbox Series X',
                whois_info: { orgname: 'Microsoft', country: 'US' },
            },
        },
        { '10.0.0.8': { name: 'PlayStation 5', whois_info: { orgname: 'Sony', country: 'JP' } } },
        {
            '10.0.0.9': {
                name: 'Apple TV 4K',
                whois_info: { orgname: 'Apple Inc.', country: 'US' },
            },
        },
    ],
};

/** Access list with some blocked clients for testing block/unblock UI. */
export const ACCESS_LIST_WITH_BLOCKS: AccessListResponse = {
    allowed_clients: [],
    disallowed_clients: ['192.168.1.30', '10.0.0.9'],
    blocked_hosts: [],
};

// ===========================================================================
// SCENARIO D — EDGE CASE: LARGE NUMBERS
// ===========================================================================

export const STATS_LARGE_NUMBERS: StatsResponse = {
    time_units: 'days',
    top_queried_domains: [
        { 'google.com': 9_876_543_210 },
        { 'youtube.com': 5_432_109_876 },
        { 'facebook.com': 2_109_876_543 },
    ],
    top_clients: [{ '192.168.1.1': 5_000_000_000 }, { '192.168.1.2': 2_500_000_000 }],
    top_blocked_domains: [{ 'doubleclick.net': 1_500_000_000 }],
    top_upstreams_responses: [{ '8.8.8.8:53': 8_000_000_000 }],
    top_upstreams_avg_time: [{ '8.8.8.8:53': 1.234 }],
    dns_queries: Array.from({ length: 720 }, () => 1_000_000),
    blocked_filtering: Array.from({ length: 720 }, () => 100_000),
    replaced_safebrowsing: Array.from({ length: 720 }, () => 10_000),
    replaced_parental: Array.from({ length: 720 }, () => 1_000),
    num_dns_queries: 7_200_000_000,
    num_blocked_filtering: 720_000_000,
    num_replaced_safebrowsing: 72_000_000,
    num_replaced_safesearch: 3_600_000,
    num_replaced_parental: 720_000,
    avg_processing_time: 1.5,
};

export const SEARCH_CLIENTS_LARGE: SearchClientsResponse = {
    clients: [],
    auto_clients: [
        {
            '192.168.1.1': {
                name: 'Main Router',
                whois_info: { orgname: 'ISP Corp', country: 'DE' },
            },
        },
        { '192.168.1.2': { name: 'Secondary AP', whois_info: {} } },
    ],
};

// ===========================================================================
// SCENARIO E — EDGE CASE: MIXED ZEROS  (some categories populated, some empty)
// ===========================================================================

export const STATS_MIXED_ZEROS: StatsResponse = {
    time_units: 'hours',
    top_queried_domains: [{ 'example.com': 500 }, { 'test.org': 200 }],
    top_clients: [{ '192.168.1.99': 700 }],
    top_blocked_domains: [], // no blocked domains at all
    top_upstreams_responses: [{ '1.1.1.1:53': 700 }],
    top_upstreams_avg_time: [], // empty — shows EmptyState
    dns_queries: [100, 200, 150, 250],
    blocked_filtering: [0, 0, 0, 0],
    replaced_safebrowsing: [0, 0, 0, 0],
    replaced_parental: [0, 0, 0, 0],
    num_dns_queries: 700,
    num_blocked_filtering: 0, // zero blocked — StatRow shows "0" with 0% bar
    num_replaced_safebrowsing: 0,
    num_replaced_safesearch: 0,
    num_replaced_parental: 0,
    avg_processing_time: 0.005,
};

export const SEARCH_CLIENTS_MIXED: SearchClientsResponse = {
    clients: [],
    auto_clients: [{ '192.168.1.99': { name: 'Test Device', whois_info: {} } }],
};

// ===========================================================================
// SCENARIO F — EDGE CASE: SINGLE ENTRY  (exactly 1 item per category)
// ===========================================================================

export const STATS_SINGLE_ENTRY: StatsResponse = {
    time_units: 'hours',
    top_queried_domains: [{ 'only-one-domain.local': 42 }],
    top_clients: [{ '10.0.0.42': 42 }],
    top_blocked_domains: [{ 'ads.only-one.net': 10 }],
    top_upstreams_responses: [{ '192.168.0.1:53': 42 }],
    top_upstreams_avg_time: [{ '192.168.0.1:53': 0.003 }],
    dns_queries: [42],
    blocked_filtering: [10],
    replaced_safebrowsing: [0],
    replaced_parental: [0],
    num_dns_queries: 42,
    num_blocked_filtering: 10,
    num_replaced_safebrowsing: 0,
    num_replaced_safesearch: 0,
    num_replaced_parental: 0,
    avg_processing_time: 0.003,
};

export const SEARCH_CLIENTS_SINGLE: SearchClientsResponse = {
    clients: [],
    auto_clients: [{ '10.0.0.42': { name: 'Lone Device', whois_info: {} } }],
};

// ===========================================================================
// SCENARIO G — EDGE CASE: IPv6 clients & special-char names
// ===========================================================================

export const STATS_IPV6_AND_SPECIAL: StatsResponse = {
    time_units: 'hours',
    top_queried_domains: [
        { 'münchen.de': 1234 },
        { 'россия.рф': 987 },
        { 'ドメイン.test': 543 },
        {
            'very-long-subdomain.this-is-a-really-really-long-domain-name-that-might-wrap.example.com': 210,
        },
    ],
    top_clients: [
        { '2001:0db8:85a3:0000:0000:8a2e:0370:7334': 5000 },
        { 'fe80::1': 2500 },
        { '::1': 1200 },
        { '2001:4860:4860::8888': 800 },
    ],
    top_blocked_domains: [{ 'evil-ads.xn--p1ai': 450 }, { 'tracker.münchen.de': 320 }],
    top_upstreams_responses: [{ '[2001:4860:4860::8888]:53': 4000 }, { '[::1]:53': 500 }],
    top_upstreams_avg_time: [{ '[2001:4860:4860::8888]:53': 0.045 }, { '[::1]:53': 0.001 }],
    dns_queries: [1000, 900, 1100, 850],
    blocked_filtering: [80, 70, 90, 60],
    replaced_safebrowsing: [5, 3, 7, 2],
    replaced_parental: [1, 0, 2, 1],
    num_dns_queries: 3850,
    num_blocked_filtering: 300,
    num_replaced_safebrowsing: 17,
    num_replaced_safesearch: 8,
    num_replaced_parental: 4,
    avg_processing_time: 0.023,
};

export const SEARCH_CLIENTS_IPV6: SearchClientsResponse = {
    clients: [],
    auto_clients: [
        {
            '2001:0db8:85a3:0000:0000:8a2e:0370:7334': {
                name: 'IPv6 Laptop',
                whois_info: { orgname: 'IPv6 ISP', country: 'JP' },
            },
        },
        { 'fe80::1': { name: 'Link-Local Device', whois_info: {} } },
        { '::1': { name: 'localhost-v6', whois_info: {} } },
        {
            '2001:4860:4860::8888': {
                name: 'Google DNS v6',
                whois_info: { orgname: 'Google LLC', country: 'US' },
            },
        },
    ],
};

// ===========================================================================
// SCENARIO H — EDGE CASE: ZERO avg_processing_time
// ===========================================================================

export const STATS_ZERO_AVG_TIME: StatsResponse = {
    time_units: 'hours',
    top_queried_domains: [{ 'fast.local': 100 }],
    top_clients: [{ '127.0.0.1': 100 }],
    top_blocked_domains: [],
    top_upstreams_responses: [{ '127.0.0.1:53': 100 }],
    top_upstreams_avg_time: [{ '127.0.0.1:53': 0 }], // zero avg time — edge case
    dns_queries: [100],
    blocked_filtering: [0],
    replaced_safebrowsing: [0],
    replaced_parental: [0],
    num_dns_queries: 100,
    num_blocked_filtering: 0,
    num_replaced_safebrowsing: 0,
    num_replaced_safesearch: 0,
    num_replaced_parental: 0,
    avg_processing_time: 0,
};

export const SEARCH_CLIENTS_ZERO_AVG: SearchClientsResponse = {
    clients: [],
    auto_clients: [{ '127.0.0.1': { name: 'localhost', whois_info: {} } }],
};

// ===========================================================================
// 4.  Scenario map — convenient lookup for tests
// ===========================================================================

export type ScenarioName =
    | 'empty'
    | 'disabled'
    | 'normal-small'
    | 'normal-full'
    | 'large-numbers'
    | 'mixed-zeros'
    | 'single-entry'
    | 'ipv6-special'
    | 'zero-avg-time';

export interface ScenarioFixture {
    stats: StatsResponse;
    statsConfig: StatsConfigResponse;
    searchClients: SearchClientsResponse;
    accessList: AccessListResponse;
    description: string;
}

export const SCENARIOS: Record<ScenarioName, ScenarioFixture> = {
    empty: {
        stats: STATS_EMPTY,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_EMPTY,
        accessList: ACCESS_LIST_EMPTY,
        description: 'Stats enabled but no queries recorded. All tables show EmptyState.',
    },
    disabled: {
        stats: STATS_EMPTY,
        statsConfig: STATS_CONFIG_DISABLED,
        searchClients: SEARCH_CLIENTS_EMPTY,
        accessList: ACCESS_LIST_EMPTY,
        description:
            'Stats explicitly disabled. Shows EmptyState "disabled" mode with settings link.',
    },
    'normal-small': {
        stats: STATS_NORMAL_SMALL,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_SMALL,
        accessList: ACCESS_LIST_EMPTY,
        description: '2–4 entries per category. Normal numbers with client WHOIS info.',
    },
    'normal-full': {
        stats: STATS_NORMAL_FULL,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_FULL,
        accessList: ACCESS_LIST_WITH_BLOCKS,
        description: '10+ entries per category, covers sort/pagination/block UI.',
    },
    'large-numbers': {
        stats: STATS_LARGE_NUMBERS,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_LARGE,
        accessList: ACCESS_LIST_EMPTY,
        description: 'Very large numbers (billions) — tests compact formatting.',
    },
    'mixed-zeros': {
        stats: STATS_MIXED_ZEROS,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_MIXED,
        accessList: ACCESS_LIST_EMPTY,
        description:
            'Some categories have data, others are zero/empty (TopBlockedDomains & UpstreamAvgTime empty).',
    },
    'single-entry': {
        stats: STATS_SINGLE_ENTRY,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_SINGLE,
        accessList: ACCESS_LIST_EMPTY,
        description: 'Exactly one entry per category — tests minimum non-empty state.',
    },
    'ipv6-special': {
        stats: STATS_IPV6_AND_SPECIAL,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_IPV6,
        accessList: ACCESS_LIST_EMPTY,
        description: 'IPv6 addresses, internationalized domain names, long names.',
    },
    'zero-avg-time': {
        stats: STATS_ZERO_AVG_TIME,
        statsConfig: STATS_CONFIG_DEFAULT,
        searchClients: SEARCH_CLIENTS_ZERO_AVG,
        accessList: ACCESS_LIST_EMPTY,
        description:
            'avg_processing_time and upstream avg time are 0 — edge case for division/math.',
    },
};

// ===========================================================================
// 5.  Dashboard protection status  (GET /control/status)
// ===========================================================================

export interface ProtectionStatusResponse {
    protection_enabled: boolean;
    protection_disabled_duration: number | null; // ms until re-enable
}

export const PROTECTION_ENABLED: ProtectionStatusResponse = {
    protection_enabled: true,
    protection_disabled_duration: null,
};

export const PROTECTION_DISABLED: ProtectionStatusResponse = {
    protection_enabled: false,
    protection_disabled_duration: null, // permanently disabled
};

export const PROTECTION_DISABLED_TIMER: ProtectionStatusResponse = {
    protection_enabled: false,
    protection_disabled_duration: 60000, // 1 minute
};

// ===========================================================================
// 6.  Normalization helpers — convert raw backend data into the
//     frontend-normalized format that Dashboard child components expect as
//     props.  Mirrors what `getStats()` does in `stores/stats.ts`.
// ===========================================================================

/** The `{ name, count }` shape used by all top-* tables. */
export interface NormalizedTopEntry {
    name: string;
    count: number;
}

/** TopClients entries also carry optional WHOIS/name info. */
export interface NormalizedClientEntry extends NormalizedTopEntry {
    info?: {
        name?: string;
        whois_info?: {
            orgname?: string;
            country?: string;
        };
        disallowed?: boolean;
    };
}

/** All component props extracted from a single scenario. */
export interface DashboardComponentProps {
    /** Props for `<GeneralStatistics>` */
    generalStatistics: {
        numDnsQueries: number;
        numBlockedFiltering: number;
        numReplacedSafebrowsing: number;
        numReplacedParental: number;
        numReplacedSafesearch: number;
        avgProcessingTime: number; // ms
    };
    /** Props for `<TopClients>` */
    topClients: {
        topClients: NormalizedClientEntry[];
        numDnsQueries: number;
    };
    /** Props for `<TopQueriedDomains>` */
    topQueriedDomains: {
        topQueriedDomains: NormalizedTopEntry[];
        numDnsQueries: number;
    };
    /** Props for `<TopBlockedDomains>` */
    topBlockedDomains: {
        topBlockedDomains: NormalizedTopEntry[];
        numBlockedFiltering: number;
    };
    /** Props for `<TopUpstreams>` */
    topUpstreams: {
        topUpstreamsResponses: NormalizedTopEntry[];
        numDnsQueries: number;
    };
    /** Props for `<UpstreamAvgTime>` */
    upstreamAvgTime: {
        topUpstreamsAvgTime: NormalizedTopEntry[]; // count is in ms
        avgProcessingTime: number; // ms
    };
    /** Props for `<StatCards>` */
    statCards: {
        numDnsQueries: number;
        numBlockedFiltering: number;
        numReplacedSafebrowsing: number;
        numReplacedParental: number;
        dnsQueries: number[];
        blockedFiltering: number[];
        replacedSafebrowsing: number[];
        replacedParental: number[];
    };
    /** Whether stats are enabled (controls EmptyState "disabled" mode). */
    enabled: boolean;
}

/**
 * Convert `[{ "key": val }]` → `[{ name: "key", count: val }]`.
 * Mirrors the `normalizeTopStats` helper in `helpers/helpers.tsx`.
 */
function normalizeTopAddrs(arr: Array<{ [key: string]: number }>): NormalizedTopEntry[] {
    return arr.map((item) => {
        const key = Object.keys(item)[0];
        return { name: key, count: item[key] };
    });
}

/**
 * Convert seconds to milliseconds.
 * Mirrors `secondsToMilliseconds` in `helpers/helpers.tsx`.
 */
function toMs(seconds: number): number {
    return seconds * 1000;
}

/**
 * Enrich top client entries with WHOIS/name info from the searchClients API.
 * Mirrors `addClientInfo` in `helpers/helpers.tsx`.
 */
function enrichClients(
    topClients: NormalizedTopEntry[],
    searchClients: SearchClientsResponse,
): NormalizedClientEntry[] {
    const allClients = [...searchClients.clients, ...searchClients.auto_clients];
    return topClients.map((entry) => {
        // Each searchClients entry is `{ "ip_or_mac": { name, whois_info, ... } }`
        const match = allClients.find((c) => entry.name in c);
        return {
            ...entry,
            info: match ? match[entry.name] : undefined,
        };
    });
}

/**
 * Given a scenario fixture, returns ALL the frontend-normalized props
 * that Dashboard child components expect.  Use this to render individual
 * child components in isolation:
 *
 * ```tsx
 * import { getComponentProps } from 'tests/fixtures/dashboard-stats';
 *
 * const props = getComponentProps(SCENARIOS['normal-small']);
 * render(() => <GeneralStatistics {...props.generalStatistics} />);
 * render(() => <TopClients {...props.topClients} />);
 * ```
 */
export function getComponentProps(fixture: ScenarioFixture): DashboardComponentProps {
    const { stats, searchClients } = fixture;

    const avgProcessingTimeMs = toMs(stats.avg_processing_time);

    return {
        generalStatistics: {
            numDnsQueries: stats.num_dns_queries,
            numBlockedFiltering: stats.num_blocked_filtering,
            numReplacedSafebrowsing: stats.num_replaced_safebrowsing,
            numReplacedParental: stats.num_replaced_parental,
            numReplacedSafesearch: stats.num_replaced_safesearch,
            avgProcessingTime: avgProcessingTimeMs,
        },

        topClients: {
            topClients: enrichClients(normalizeTopAddrs(stats.top_clients), searchClients),
            numDnsQueries: stats.num_dns_queries,
        },

        topQueriedDomains: {
            topQueriedDomains: normalizeTopAddrs(stats.top_queried_domains),
            numDnsQueries: stats.num_dns_queries,
        },

        topBlockedDomains: {
            topBlockedDomains: normalizeTopAddrs(stats.top_blocked_domains),
            numBlockedFiltering: stats.num_blocked_filtering,
        },

        topUpstreams: {
            topUpstreamsResponses: normalizeTopAddrs(stats.top_upstreams_responses),
            numDnsQueries: stats.num_dns_queries,
        },

        upstreamAvgTime: {
            topUpstreamsAvgTime: normalizeTopAddrs(stats.top_upstreams_avg_time).map((entry) => ({
                ...entry,
                count: toMs(entry.count),
            })),
            avgProcessingTime: avgProcessingTimeMs,
        },

        statCards: {
            numDnsQueries: stats.num_dns_queries,
            numBlockedFiltering: stats.num_blocked_filtering,
            numReplacedSafebrowsing: stats.num_replaced_safebrowsing,
            numReplacedParental: stats.num_replaced_parental,
            dnsQueries: stats.dns_queries,
            blockedFiltering: stats.blocked_filtering,
            replacedSafebrowsing: stats.replaced_safebrowsing,
            replacedParental: stats.replaced_parental,
        },

        enabled: fixture.statsConfig.enabled,
    };
}
