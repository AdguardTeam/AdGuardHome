import ipaddr, { IPv4, IPv6 } from 'ipaddr.js';
import queryString from 'qs';
import intl from 'panel/common/intl';
import { getTrackerData } from './trackers/trackers';
import type { TrackerData } from './trackers/trackers';

import {
    ADDRESS_TYPES,
    CHECK_TIMEOUT,
    COMMENT_LINE_DEFAULT_TOKEN,
    DEFAULT_DATE_FORMAT_OPTIONS,
    DETAILED_DATE_FORMAT_OPTIONS,
    DHCP_VALUES_PLACEHOLDERS,
    FILTERED,
    FILTERED_STATUS,
    R_CLIENT_ID,
    STANDARD_HTTPS_PORT,
    STANDARD_WEB_PORT,
    SPECIAL_FILTER_ID,
    THEMES,
    SHORT_DATE_FORMAT_OPTIONS,
} from './constants';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from './localStorageHelper';
import type { DnsAnswer } from 'panel/api/model/dnsAnswer';
import type { ResultRule } from 'panel/api/model/resultRule';
import type { FilteringReason } from 'panel/api/model/filteringReason';
import type { FilterStatus } from 'panel/api/model/filterStatus';
import type { TopArrayEntry } from 'panel/api/model/topArrayEntry';
import type { ClientsFindEntry } from 'panel/api/model/clientsFindEntry';
import type { ClientFindSubEntry } from 'panel/api/model/clientFindSubEntry';
import type { QueryLogItemClient } from 'panel/api/model/queryLogItemClient';
import type { QueryLogItemClientWhois } from 'panel/api/model/queryLogItemClientWhois';
import type { QueryLogItemClientProto } from 'panel/api/model/queryLogItemClientProto';
import type { QueryLogItem } from 'panel/api/model/queryLogItem';

export type NormalizedDnsResponse = {
    value?: string;
    type?: string;
    ttl?: number;
};

export type NormalizedQueryLogItem = {
    time: string;
    domain: string;
    unicodeName: string;
    type: string;
    response: NormalizedDnsResponse[];
    reason?: FilteringReason;
    client: string;
    client_proto?: QueryLogItemClientProto;
    client_id?: string;
    client_info: QueryLogItemClient | null;
    filterId?: number; // @deprecated
    rule?: string; // @deprecated
    rules: ResultRule[];
    status?: string;
    service_name?: string;
    serviceName?: string;
    originalAnswer?: DnsAnswer[];
    originalResponse: NormalizedDnsResponse[];
    tracker: TrackerData | null;
    answer_dnssec?: boolean;
    elapsedMs?: string;
    upstream?: string;
    cached?: boolean;
    ecs?: string;
};

/**
 * @param dateTime {string} The date to format
 * @param [options] {object} Date.prototype.toLocaleString([locales[, options]]) options argument
 * @returns {string} Returns the date and time in the specified format
 */
export const formatDateTime = (
    dateTime: string,
    options: Intl.DateTimeFormatOptions = DEFAULT_DATE_FORMAT_OPTIONS,
) => {
    if (!dateTime) {
        return '-';
    }

    const parsedTime = new Date(dateTime);

    return parsedTime.toLocaleString(navigator.language, options);
};

/**
 * @param dateTime {string} The date to format
 * @returns {string} Returns the date and time in the format with the full month name
 */
export const formatDetailedDateTime = (dateTime: string) =>
    formatDateTime(dateTime, DETAILED_DATE_FORMAT_OPTIONS);

/**
 * @param dateTime {string} The date to format
 * @returns {string} Returns the date and time in the format with the short month name. Example: 8 Nov, 2024, 13:15
 */
export const formatShortDateTime = (dateTime: string) =>
    formatDateTime(dateTime, SHORT_DATE_FORMAT_OPTIONS);

export const normalizeLogs = (logs: QueryLogItem[]): NormalizedQueryLogItem[] =>
    logs.map((log) => {
        const {
            answer,
            answer_dnssec,
            client,
            client_proto,
            client_id,
            client_info,
            elapsedMs,
            question,
            reason,
            status,
            time,
            filterId,
            rule,
            rules,
            service_name,
            original_answer,
            upstream,
            cached,
            ecs,
        } = log;

        const { name: domain, unicode_name: unicodeName, type } = question || {};

        const processResponse = (data: DnsAnswer[] | undefined): NormalizedDnsResponse[] =>
            Array.isArray(data)
                ? data.map((response: DnsAnswer) => {
                      const { value, type, ttl } = response;

                      return {
                          value,
                          type,
                          ttl,
                      };
                  })
                : [];

        let newRules = Array.isArray(rules) ? rules : [];
        /* TODO 'filterId' and 'rule' are deprecated, will be removed in 0.106 */
        if (rule !== undefined && filterId !== undefined && newRules.length === 0) {
            newRules = [
                {
                    filter_list_id: filterId,
                    text: rule,
                },
            ];
        }

        return {
            time,
            domain,
            unicodeName,
            type,
            response: processResponse(answer),
            reason,
            client,
            client_proto,
            client_id,
            client_info: client_info
                ? {
                      ...client_info,
                      whois: client_info.whois || {},
                  }
                : null,
            /* TODO 'filterId' and 'rule' are deprecated, will be removed in 0.106 */
            filterId,
            rule,
            rules: newRules,
            status,
            service_name,
            serviceName: service_name,
            originalAnswer: original_answer,
            originalResponse: processResponse(original_answer),
            tracker: getTrackerData(domain),
            answer_dnssec,
            elapsedMs,
            upstream,
            cached,
            ecs,
        };
    });

export const normalizeTopStats = (stats: TopArrayEntry[]): TopStat[] =>
    stats.map((item: TopArrayEntry) => ({
        name: Object.keys(item)[0],

        count: Object.values(item)[0] as number,
    }));

export const addClientInfo = (
    data: TopStat[],
    clients: ClientsFindEntry[],
    ...params: string[]
): (TopStat & { info: ClientFindSubEntry })[] =>
    data.map((row: TopStat) => {
        let info: ClientFindSubEntry | null = null;
        params.find((param) => {
            const id = row[param as keyof TopStat];
            if (id) {
                const clientData = clients.find((item: ClientsFindEntry) => item[String(id)]);
                info = clientData?.[String(id)] ?? null;
            }

            return info;
        });

        return {
            ...row,
            info: info ?? {},
        };
    });

export const normalizeFilters = (filters: FilterStatus['filters']) =>
    filters
        ? filters.map((filter) => {
              const {
                  id,
                  url,
                  enabled,
                  last_updated,
                  name = 'Default name',
                  rules_count = 0,
              } = filter;

              return {
                  id,
                  url,
                  enabled,
                  lastUpdated: last_updated,
                  name,
                  rulesCount: rules_count,
              };
          })
        : [];

export const normalizeFilteringStatus = (
    filteringStatus: FilterStatus,
): {
    enabled: boolean | undefined;
    userRules: string;
    filters: Filter[];
    whitelistFilters: Filter[];
    interval: number | undefined;
} => {
    const {
        enabled,
        filters,
        user_rules: userRules,
        interval,
        whitelist_filters,
    } = filteringStatus;
    const newUserRules = Array.isArray(userRules) ? userRules.join('\n') : '';

    return {
        enabled,
        userRules: newUserRules,
        filters: normalizeFilters(filters),
        whitelistFilters: normalizeFilters(whitelist_filters),
        interval,
    };
};

export const captitalizeWords = (text: string): string =>
    text
        .split(/[ -_]/g)
        .map((str: string) => str.charAt(0).toUpperCase() + str.substr(1))
        .join(' ');

type InterfaceWithIpAddresses = { ip_addresses?: string[] };

type TopStat = { name: string; count: number };

type ServiceEntry = { id: string; name: string };

export const getInterfaceIp = (option: InterfaceWithIpAddresses): string | undefined => {
    const addresses = (option?.ip_addresses ?? []).filter((ip: string) => typeof ip === 'string');

    const isIpv6 = (ip: string) => ip.includes(':');
    const isIpv6LinkLocal = (ip: string) => ip.toLowerCase().startsWith('fe80:');
    const hasZoneId = (ip: string) => ip.includes('%');

    const ipv4 = addresses.find((ip: string) => !isIpv6(ip));
    if (ipv4) {
        return ipv4;
    }

    const ipv6Global = addresses.find(
        (ip: string) => isIpv6(ip) && !isIpv6LinkLocal(ip) && !hasZoneId(ip),
    );
    if (ipv6Global) {
        return ipv6Global;
    }

    const ipv6NoZone = addresses.find((ip: string) => isIpv6(ip) && !hasZoneId(ip));
    return ipv6NoZone || addresses[0];
};

const normalizeHost = (host: string) => {
    const isIpv6 = host.includes(':');
    if (!isIpv6) {
        return host;
    }

    const encodeZone = (s: string) => s.replaceAll('%', '%25');

    if (host.startsWith('[') && host.endsWith(']')) {
        const inner = host.slice(1, -1);
        return `[${encodeZone(inner)}]`;
    }

    return `[${encodeZone(host)}]`;
};

/**
 * @param {string} ip
 * @param {number} [port]
 * @returns {string}
 */
export const getWebAddress = (ip: string, port: number = 0): string => {
    const isStandardWebPort = port === STANDARD_WEB_PORT;
    const rawHost = String(ip);

    const host = normalizeHost(rawHost);
    const portPart = port && !isStandardWebPort ? `:${port}` : '';

    return `http://${host}${portPart}`;
};

export const checkRedirect = (url: string, attempts: number = 1): boolean => {
    let count = attempts || 1;

    if (count > 10) {
        window.location.replace(url);
        return false;
    }

    const rmTimeout = (t: ReturnType<typeof setTimeout> | undefined) => t && clearTimeout(t);
    const setRecursiveTimeout = (
        time: number,
        ...args: [string, number]
    ): ReturnType<typeof setTimeout> => setTimeout(checkRedirect, time, ...args);

    let timeout: ReturnType<typeof setTimeout> | undefined;

    fetch(url)
        .then((response) => {
            rmTimeout(timeout);
            if (response.ok) {
                window.location.replace(url);
                return;
            }
            timeout = setRecursiveTimeout(CHECK_TIMEOUT, url, (count += 1));
        })
        .catch(() => {
            rmTimeout(timeout);
            timeout = setRecursiveTimeout(CHECK_TIMEOUT, url, (count += 1));
        });

    return false;
};

type RedirectValues = {
    enabled?: boolean;
    force_https?: boolean;
    port_https?: number;
};

export const redirectToCurrentProtocol = (values: RedirectValues, httpPort = 80) => {
    const { protocol, hostname, hash, port } = window.location;
    const { enabled, force_https, port_https } = values;
    const httpsPort = port_https !== STANDARD_HTTPS_PORT ? `:${port_https}` : '';

    if (protocol !== 'https:' && enabled && force_https && port_https) {
        checkRedirect(`https://${hostname}${httpsPort}/${hash}`);
    } else if (
        protocol === 'https:' &&
        enabled &&
        port_https &&
        port_https !== parseInt(port, 10)
    ) {
        checkRedirect(`https://${hostname}${httpsPort}/${hash}`);
    } else if (protocol === 'https:' && (!enabled || !port_https)) {
        window.location.replace(`http://${hostname}:${httpPort}/${hash}`);
    }
};

/**
 * @param {string} text
 * @returns []string
 */
export const splitByNewLine = (text: string | undefined | null): string[] => {
    if (!text) {
        return [];
    }
    return text.split('\n').filter((n: string) => n.trim());
};

/**
 * @param {string} input
 * @returns {string}
 */
export const trimLinesAndRemoveEmpty = (input: string): string =>
    input
        .split('\n')
        .map((line: string) => line.trim())
        .filter(Boolean)
        .join('\n');

/**
 * Normalizes the topClients array
 *
 * @param {Object[]} topClients
 * @param {string} topClients.name
 * @param {number} topClients.count
 * @param {Object} topClients.info
 * @param {string} topClients.info.name
 * @returns {Object} normalizedTopClients
 * @returns {Object.<string, number>} normalizedTopClients.auto - auto clients
 * @returns {Object.<string, number>} normalizedTopClients.configured - configured clients
 */
export const normalizeTopClients = (
    topClients: (TopStat & { info: ClientFindSubEntry })[],
): { auto: Record<string, number>; configured: Record<string, number> } =>
    topClients.reduce(
        (
            acc: { auto: Record<string, number>; configured: Record<string, number> },
            clientObj: TopStat & { info: ClientFindSubEntry },
        ) => {
            const {
                name,
                count,
                info: { name: infoName },
            } = clientObj;
            acc.auto[name] = count;
            acc.configured[infoName] = count;
            return acc;
        },
        {
            auto: {},
            configured: {},
        },
    );

export const msToSeconds = (milliseconds: number): number => Math.floor(milliseconds / 1000);

export const msToMinutes = (milliseconds: number): number => Math.floor(milliseconds / 1000 / 60);

export const msToHours = (milliseconds: number): number =>
    Math.floor(milliseconds / 1000 / 60 / 60);

export const secondsToMilliseconds = (seconds: number): number => {
    if (seconds) {
        return seconds * 1000;
    }

    return seconds;
};

export const normalizeRulesTextarea = (text: string): string | undefined =>
    text?.replace(/^\n/g, '').replace(/\n\s*\n/g, '\n');

export const normalizeWhois = (
    whois: QueryLogItemClientWhois,
): Partial<QueryLogItemClientWhois> & { location?: string } => {
    if (Object.keys(whois).length > 0) {
        const { city, country, ...values } = whois;
        let location = country || '';

        if (city && location) {
            location = `${location}, ${city}`;
        } else if (city) {
            location = city;
        }

        if (location) {
            return {
                location,
                ...values,
            };
        }

        return { ...values };
    }

    return {
        location: 'New York, US',
        orgname: 'Example Organization',
    };
};

export const getPathWithQueryString = (
    path: string,
    params: Record<string, string | string[] | undefined | null> | undefined,
): string => {
    const searchParams = new URLSearchParams();

    Object.entries(params || {}).forEach(([key, value]) => {
        if (value === '' || value === undefined || value === null) {
            return;
        }

        if (Array.isArray(value)) {
            value.forEach((v) => {
                if (v !== '' && v !== undefined && v !== null) {
                    searchParams.append(key, String(v));
                }
            });
        } else {
            searchParams.append(key, String(value));
        }
    });

    return `${path}?${searchParams.toString()}`;
};

export const getParamsForClientsSearch = (
    data: Record<string, unknown>[],
    param: string,
    additionalParam?: string,
): { clients: { id: string }[] } => {
    const clients = new Set<string | number>();
    data.forEach((e: Record<string, unknown>) => {
        clients.add(e[param] as string | number);
        if (e[additionalParam as string]) {
            clients.add(e[additionalParam as string] as string | number);
        }
    });

    return {
        clients: Array.from(clients).map((id) => ({ id: id as string })),
    };
};

export const checkFiltered = (reason: FilteringReason): boolean => reason.indexOf(FILTERED) === 0;
export const checkBlockedService = (reason: FilteringReason): boolean =>
    reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

/**
 * @param num {number} to format
 * @returns {string} Returns a string with a language-sensitive representation of this number
 */
export const formatNumber = (num: number): string => {
    // Use browser's default locale since we don't have access to i18n language
    return num.toLocaleString();
};

/**
 * Formats a number in compact notation (e.g., 10.2K, 1.5M)
 * @param num {number} The number to format
 * @param decimals {number} Number of decimal places (default: 1)
 * @returns {string} Formatted string like "10.2K", "1.5M", "2.3B"
 */
export const formatCompactNumber = (num: number, decimals: number = 1): string => {
    if (num === 0) return '0';

    const absNum = Math.abs(num);
    const sign = num < 0 ? '-' : '';

    if (absNum < 1000) {
        return sign + absNum.toString();
    }

    const suffixes = ['', 'K', 'M', 'B', 'T'];
    const tier = Math.floor(Math.log10(absNum) / 3);
    const suffix = suffixes[Math.min(tier, suffixes.length - 1)];
    const scale = 10 ** (tier * 3);
    const scaled = absNum / scale;

    const formatted = scaled.toFixed(decimals).replace(/\.0+$/, '');

    return sign + formatted + suffix;
};

/**
 * @param parsedIp {object} ipaddr.js IPv4 or IPv6 object
 * @param parsedCidr {array} ipaddr.js CIDR array
 * @returns {boolean}
 */
const isIpMatchCidr = (parsedIp: IPv4 | IPv6, parsedCidr: [IPv4 | IPv6, number]): boolean => {
    try {
        const cidrIpVersion = parsedCidr[0].kind();
        const ipVersion = parsedIp.kind();

        return ipVersion === cidrIpVersion && parsedIp.match(parsedCidr);
    } catch (_e) {
        return false;
    }
};

export const isIpInCidr = (ip: string, cidr: string): boolean => {
    try {
        const parsedIp = ipaddr.parse(ip);
        const parsedCidr = ipaddr.parseCIDR(cidr);

        return isIpMatchCidr(parsedIp, parsedCidr);
    } catch (e) {
        console.error(e);
        return false;
    }
};

/**
 * Validates an IPv6 address using ipaddr.js, including zone IDs (e.g., fe80::1%eth0).
 * @param value - The string to validate.
 * @returns true if the value is a valid IPv6 address.
 */
export const isValidIpv6 = (value: string): boolean => {
    try {
        return ipaddr.IPv6.isValid(value);
    } catch (_e) {
        return false;
    }
};

/**
 *
 * @param {string} subnetMask
 * @returns {IPv4 | null}
 */
export const parseSubnetMask = (subnetMask: string): number | null => {
    try {
        return ipaddr.parse(subnetMask).prefixLengthFromSubnetMask();
    } catch (e) {
        console.error(e);
        return null;
    }
};

/**
 *
 * @param {string} subnetMask
 * @returns {*}
 */
export const subnetMaskToBitMask = (subnetMask: string): number =>
    subnetMask
        .split('.')
        .reduce((acc: number, cur: string) => acc - Math.log2(256 - Number(cur)), 32);

/**
 *
 * @param ipOrCidr
 * @returns {'IP' | 'CIDR' | 'CLIENT_ID' | 'UNKNOWN'}
 *
 */
export const findAddressType = (address: string): string => {
    try {
        const cidrMaybe = address.includes('/');

        if (!cidrMaybe && ipaddr.isValid(address)) {
            return ADDRESS_TYPES.IP;
        }
        if (cidrMaybe && ipaddr.parseCIDR(address)) {
            return ADDRESS_TYPES.CIDR;
        }
        if (R_CLIENT_ID.test(address)) {
            return ADDRESS_TYPES.CLIENT_ID;
        }

        return ADDRESS_TYPES.UNKNOWN;
    } catch (_e) {
        return ADDRESS_TYPES.UNKNOWN;
    }
};

/**
 * @param ids {string[]}
 * @returns {Object}
 */
export const separateIpsAndCidrs = (
    ids: string[],
): { ips: string[]; cidrs: string[]; clientIds: string[] } =>
    ids.reduce(
        (acc: { ips: string[]; cidrs: string[]; clientIds: string[] }, curr: string) => {
            const addressType = findAddressType(curr);

            if (addressType === ADDRESS_TYPES.IP) {
                acc.ips.push(curr);
            }
            if (addressType === ADDRESS_TYPES.CIDR) {
                acc.cidrs.push(curr);
            }
            if (addressType === ADDRESS_TYPES.CLIENT_ID) {
                acc.clientIds.push(curr);
            }
            return acc;
        },
        { ips: [], cidrs: [], clientIds: [] },
    );

export const countClientsStatistics = (
    ids: string[],
    autoClients: Record<string, number>,
): number => {
    const { ips, cidrs, clientIds } = separateIpsAndCidrs(ids);

    const ipsCount = ips.reduce((acc: number, curr: string) => {
        const count = autoClients[curr] || 0;
        return acc + count;
    }, 0);

    const clientIdsCount = clientIds.reduce((acc: number, curr: string) => {
        const count = autoClients[curr] || 0;
        return acc + count;
    }, 0);

    const cidrsCount = Object.entries(autoClients).reduce((acc: number, curr: [string, number]) => {
        const [id, count] = curr;
        if (!ipaddr.isValid(id)) {
            return acc;
        }
        if (cidrs.some((cidr: string) => isIpInCidr(id, cidr))) {
            // eslint-disable-next-line no-param-reassign
            acc += count;
        }
        return acc;
    }, 0);

    return ipsCount + cidrsCount + clientIdsCount;
};

/**
 * @param {string} elapsedMs
 * @param {string} millisecondsLabel
 * @returns {string}
 */
export const formatElapsedMs = (elapsedMs: string, millisecondsLabel: string) => {
    const parsedElapsedMs = parseFloat(elapsedMs);

    if (Number.isNaN(parsedElapsedMs)) {
        return elapsedMs;
    }

    const formattedValue =
        parsedElapsedMs < 1 ? parsedElapsedMs.toFixed(2) : Math.floor(parsedElapsedMs).toString();

    return `${formattedValue} ${millisecondsLabel}`;
};

/**
 * @param language {string}
 */
export const setHtmlLangAttr = (language: string): void => {
    window.document.documentElement.lang = language;
};

/**
 * Set local storage theme field
 *
 * @param {string} theme
 */
export const setTheme = (theme: string): void => {
    LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.THEME, theme);
};

/**
 * Get local storage theme field
 *
 * @returns {string}
 */

export const getTheme = () =>
    LocalStorageHelper.getItem<string>(LOCAL_STORAGE_KEYS.THEME) || THEMES.light;

/**
 * Sets UI theme.
 *
 * @param theme
 */
export const setUITheme = (theme?: string): void => {
    let currentTheme = theme || getTheme();

    if (currentTheme === THEMES.auto) {
        const prefersDark =
            window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        currentTheme = prefersDark ? THEMES.dark : THEMES.light;
    }
    setTheme(currentTheme);
    document.documentElement.dataset.theme = currentTheme;
    document.documentElement.style.colorScheme = currentTheme;
};

/**
 * @param {string} search
 * @param {string} status
 * @param {string} [reason]
 * @returns {string}
 */
export const getLogsUrlParams = (search: string, status: string, reason: string): string =>
    `?${queryString.stringify({
        search: search || undefined,
        status: status || undefined,
        reason: reason || undefined,
    })}`;

/**
 * @param ip
 * @returns {[IPv4|IPv6, 33|129]}
 */
const getParsedIpWithPrefixLength = (ip: string): [IPv4 | IPv6, number] => {
    const MAX_PREFIX_LENGTH_V4 = 32;
    const MAX_PREFIX_LENGTH_V6 = 128;

    const parsedIp = ipaddr.parse(ip);
    const prefixLength = parsedIp.kind() === 'ipv4' ? MAX_PREFIX_LENGTH_V4 : MAX_PREFIX_LENGTH_V6;

    // Increment prefix length to always put IP after CIDR, e.g. 127.0.0.1/32, 127.0.0.1
    return [parsedIp, prefixLength + 1];
};

/**
 * Helper function for IP and CIDR comparison (supports both v4 and v6)
 * @param item - ip or cidr
 * @returns {number[]}
 */
const getAddressesComparisonBytes = (item: string): number[] => {
    // Sort ipv4 before ipv6
    const IP_V4_COMPARISON_CODE = 0;
    const IP_V6_COMPARISON_CODE = 1;

    const [parsedIp, cidr] = ipaddr.isValid(item)
        ? getParsedIpWithPrefixLength(item)
        : ipaddr.parseCIDR(item);

    const [normalizedBytes, ipVersionComparisonCode] =
        (parsedIp as IPv4 | IPv6).kind() === 'ipv4'
            ? [(parsedIp as IPv4).toIPv4MappedAddress().parts, IP_V4_COMPARISON_CODE]
            : [(parsedIp as IPv6).parts, IP_V6_COMPARISON_CODE];

    return [ipVersionComparisonCode, ...normalizedBytes, cidr];
};

/**
 * Compare function for IP addresses and CIDR ranges in ascending order.
 * Supports both IPv4 and IPv6. IPv4 addresses are sorted before IPv6.
 * Individual IPs are sorted after their equivalent /32 (or /128) CIDR range.
 */
export const sortIp = (a: string, b: string): number => {
    try {
        const comparisonBytesA = Array.isArray(a)
            ? getAddressesComparisonBytes(a[0])
            : getAddressesComparisonBytes(a);
        const comparisonBytesB = Array.isArray(b)
            ? getAddressesComparisonBytes(b[0])
            : getAddressesComparisonBytes(b);

        for (let i = 0; i < comparisonBytesA.length; i += 1) {
            const byteA = comparisonBytesA[i];
            const byteB = comparisonBytesB[i];

            if (byteA !== byteB) {
                return byteA > byteB ? 1 : -1;
            }
        }

        return 0;
    } catch (e) {
        console.warn(e);
        return 0;
    }
};

/**
 * @param {number} filterId
 * @returns {string}
 */
export const getSpecialFilterName = (filterId: number): string => {
    switch (filterId) {
        case SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES:
            return intl.getMessage('custom_rules');
        case SPECIAL_FILTER_ID.SYSTEM_HOSTS:
            return intl.getMessage('system_host_files');
        case SPECIAL_FILTER_ID.BLOCKED_SERVICES:
            return intl.getMessage('blocked_services');
        case SPECIAL_FILTER_ID.PARENTAL:
            return intl.getMessage('parental_control');
        case SPECIAL_FILTER_ID.SAFE_BROWSING:
            return intl.getMessage('safe_browsing');
        case SPECIAL_FILTER_ID.SAFE_SEARCH:
            return intl.getMessage('safe_search');
        default:
            return intl.getMessage('unknown_filter', { filterId });
    }
};

export type Filter = {
    enabled: boolean;
    id: number;
    lastUpdated: string;
    name: string;
    rulesCount: number;
    url: string;
};

export type Rule = {
    filter_list_id?: number;
    text?: string;
};

export const getFilterName = (
    filters: Filter[],
    whitelistFilters: Filter[],
    filterId: number,
    resolveFilterName = (filter: Filter) =>
        filter ? filter.name : intl.getMessage('unknown_filter', { filterId }),
) => {
    const specialFilterIds = Object.values(SPECIAL_FILTER_ID);
    if (specialFilterIds.includes(filterId)) {
        return getSpecialFilterName(filterId);
    }

    const matchIdPredicate = (filter: Filter) => filter.id === filterId;
    const filter = filters.find(matchIdPredicate) || whitelistFilters.find(matchIdPredicate);
    return resolveFilterName(filter);
};

export const getFilterNames = (rules: Rule[], filters: Filter[], whitelistFilters: Filter[]) =>
    rules
        .filter((r): r is Required<Rule> => r.filter_list_id != null)
        .map(({ filter_list_id }) =>
            getFilterName(filters, whitelistFilters, filter_list_id),
        );

/**
 * @param {string[]} lines
 * @returns {string[]}
 */
export const filterOutComments = (lines: string[]): string[] =>
    lines.filter((line: string) => !line.startsWith(COMMENT_LINE_DEFAULT_TOKEN));

/**
 * Computes DHCP v4 placeholder values from the interface IP address.
 * Replaces the last octet with 100 for range_start and 200 for range_end.
 * @param ip - The interface's IPv4 address (e.g. "192.168.1.1")
 * @param gatewayIp - The interface's gateway IP (falls back to `ip` if empty)
 * @returns Pre-filled v4 config values
 */
export const calculateDhcpPlaceholdersIpv4 = (ip: string, gatewayIp: string) => {
    const LAST_OCTET_IDX = 3;
    const LAST_OCTET_RANGE_START = 100;
    const LAST_OCTET_RANGE_END = 200;

    const addr = ipaddr.parse(ip) as IPv4;

    addr.octets[LAST_OCTET_IDX] = LAST_OCTET_RANGE_START;
    const range_start = addr.toString();

    addr.octets[LAST_OCTET_IDX] = LAST_OCTET_RANGE_END;
    const range_end = addr.toString();

    const { subnet_mask, lease_duration } = DHCP_VALUES_PLACEHOLDERS.ipv4;

    return {
        gateway_ip: gatewayIp || ip,
        subnet_mask,
        range_start,
        range_end,
        lease_duration,
    };
};

/**
 * Computes DHCP v6 placeholder values (static defaults).
 * @returns Pre-filled v6 config values
 */
export const calculateDhcpPlaceholdersIpv6 = () => {
    const { range_start, lease_duration } = DHCP_VALUES_PLACEHOLDERS.ipv6;

    return {
        range_start,
        lease_duration,
    };
};

/**
 * @param {array} services
 * @param {string} id
 * @returns {string}
 */
export const getService = (services: ServiceEntry[], id: string): ServiceEntry | undefined =>
    services.find((s: ServiceEntry) => s.id === id);

/**
 * @param {array} services
 * @param {string} id
 * @returns {string}
 */
export const getServiceName = (services: ServiceEntry[], id: string): string | undefined =>
    getService(services, id)?.name;

/**
 * Decodes a base64-encoded SVG string. Returns an empty string on failure.
 */
export const decodeSvg = (iconSvg: string): string => {
    if (!iconSvg) {
        return '';
    }
    // Try base64 decode first; fall back to raw SVG if that fails.
    try {
        return atob(iconSvg);
    } catch {
        return iconSvg;
    }
};

export const delay = (ms: number): Promise<void> =>
    new Promise((resolve) => {
        setTimeout(resolve, ms);
    });
