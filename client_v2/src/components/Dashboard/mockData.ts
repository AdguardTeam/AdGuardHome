// Mock data for testing charts
export const MOCK_DATA = {
    dnsQueries: [
        120, 150, 180, 200, 170, 190, 220, 250, 230, 210, 240, 280, 300, 320, 290, 310, 350, 380, 360, 340, 370
    ],
    blockedFiltering: [
        10, 15, 12, 18, 14, 20, 25, 22, 28, 24, 30, 35, 32, 38, 34, 40, 45, 42, 48, 44, 50, 55, 52, 58
    ],
    replacedSafebrowsing: [
        2, 3, 1, 4, 2, 5, 3, 6, 4, 7, 5, 8, 6, 9, 7, 10, 8, 11, 9, 12, 10, 13, 11, 14
    ],
    replacedParental: [
        1, 2, 1, 3, 2, 4, 3, 5, 4, 6, 5, 7, 6, 8, 7, 9, 8, 10, 9, 11, 10, 12, 11, 13
    ],
    numDnsQueries: 6420,
    numBlockedFiltering: 782,
    numReplacedSafebrowsing: 156,
    numReplacedParental: 168,
    numReplacedSafesearch: 45,
    avgProcessingTime: 42,
    topQueriedDomains: [
        { name: 'google.com', count: 1250 },
        { name: 'facebook.com', count: 890 },
        { name: 'youtube.com', count: 756 },
        { name: 'twitter.com', count: 542 },
        { name: 'instagram.com', count: 423 },
        { name: 'reddit.com', count: 312 },
        { name: 'github.com', count: 287 },
        { name: 'stackoverflow.com', count: 198 },
        { name: 'amazon.com', count: 156 },
        { name: 'netflix.com', count: 134 },
    ],
    topBlockedDomains: [
        { name: 'doubleclick.net', count: 245 },
        { name: 'googlesyndication.com', count: 189 },
        { name: 'facebook.com', count: 156 },
        { name: 'analytics.google.com', count: 98 },
        { name: 'ads.yahoo.com', count: 67 },
        { name: 'tracking.example.com', count: 45 },
        { name: 'adserver.example.org', count: 34 },
    ],
    topClients: [
        { name: '192.168.1.10', count: 2340, info: { name: 'MacBook Pro' } },
        { name: '192.168.1.15', count: 1890, info: { name: 'iPhone' } },
        { name: '192.168.1.20', count: 1245, info: { name: 'iPad' } },
        { name: '192.168.1.25', count: 678, info: { name: 'Smart TV' } },
        { name: '192.168.1.30', count: 267, info: null },
    ],
    topUpstreamsResponses: [
        { name: 'https://dns.google/dns-query', count: 3200 },
        { name: 'https://cloudflare-dns.com/dns-query', count: 2100 },
        { name: 'https://dns.quad9.net/dns-query', count: 1120 },
    ],
    topUpstreamsAvgTime: [
        { name: 'https://dns.google/dns-query', count: 35 },
        { name: 'https://cloudflare-dns.com/dns-query', count: 28 },
        { name: 'https://dns.quad9.net/dns-query', count: 45 },
    ],
};

export const USE_MOCK_DATA = true; // Set to false to use real data
