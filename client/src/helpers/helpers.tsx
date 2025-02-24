import 'url-polyfill';
import dateParse from 'date-fns/parse';
import dateFormat from 'date-fns/format';
import round from 'lodash/round';
import axios from 'axios';
import i18n from 'i18next';
import ipaddr, { IPv4, IPv6 } from 'ipaddr.js';
import queryString from 'query-string';
import React from 'react';
import { getTrackerData } from './trackers/trackers';

import {
    ADDRESS_TYPES,
    CHECK_TIMEOUT,
    COMMENT_LINE_DEFAULT_TOKEN,
    DEFAULT_DATE_FORMAT_OPTIONS,
    DEFAULT_LANGUAGE,
    DEFAULT_TIME_FORMAT,
    DETAILED_DATE_FORMAT_OPTIONS,
    DHCP_VALUES_PLACEHOLDERS,
    FILTERED,
    FILTERED_STATUS,
    R_CLIENT_ID,
    STANDARD_DNS_PORT,
    STANDARD_HTTPS_PORT,
    STANDARD_WEB_PORT,
    SPECIAL_FILTER_ID,
    THEMES,
} from './constants';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from './localStorageHelper';
import { DhcpInterface } from '../initialState';

/**
 * @param time {string} The time to format
 * @param options {string}
 * @returns {string} Returns the time in the format HH:mm:ss
 */
export const formatTime = (time: any, options = DEFAULT_TIME_FORMAT) => {
    const parsedTime = dateParse(time);
    return dateFormat(parsedTime, options);
};

/**
 * @param dateTime {string} The date to format
 * @param [options] {object} Date.prototype.toLocaleString([locales[, options]]) options argument
 * @returns {string} Returns the date and time in the specified format
 */
export const formatDateTime = (dateTime: string, options: Intl.DateTimeFormatOptions = DEFAULT_DATE_FORMAT_OPTIONS) => {
    const parsedTime = new Date(dateTime);

    return parsedTime.toLocaleString(navigator.language, options);
};

/**
 * @param dateTime {string} The date to format
 * @returns {string} Returns the date and time in the format with the full month name
 */
export const formatDetailedDateTime = (dateTime: string) => formatDateTime(dateTime, DETAILED_DATE_FORMAT_OPTIONS);

export const normalizeLogs = (logs: any) =>
    logs.map((log: any) => {
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

        const { name: domain, unicode_name: unicodeName, type } = question;

        const processResponse = (data: any) =>
            data
                ? data.map((response: any) => {
                      const { value, type, ttl } = response;
                      return `${type}: ${value} (ttl=${ttl})`;
                  })
                : [];

        let newRules = rules;
        /* TODO 'filterId' and 'rule' are deprecated, will be removed in 0.106 */
        if (rule !== undefined && filterId !== undefined && rules !== undefined && rules.length === 0) {
            newRules = {
                filter_list_id: filterId,
                text: rule,
            };
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
            client_info,
            /* TODO 'filterId' and 'rule' are deprecated, will be removed in 0.106 */
            filterId,
            rule,
            rules: newRules,
            status,
            service_name,
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

export const normalizeHistory = (history: any) =>
    history.map((item, idx) => ({
        x: idx,
        y: item,
    }));

export const normalizeTopStats = (stats: any) =>
    stats.map((item: any) => ({
        name: Object.keys(item)[0],

        count: Object.values(item)[0],
    }));

export const addClientInfo = (data: any, clients: any, ...params: any[]) =>
    data.map((row: any) => {
        let info = '';
        params.find((param) => {
            const id = row[param];
            if (id) {
                const client = clients.find((item: any) => item[id]) || '';
                info = client?.[id] ?? '';
            }

            return info;
        });

        return {
            ...row,
            info,
        };
    });

export const normalizeFilters = (filters: any) =>
    filters
        ? filters.map((filter: any) => {
              const { id, url, enabled, last_updated, name = 'Default name', rules_count = 0 } = filter;

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

export const normalizeFilteringStatus = (filteringStatus: any) => {
    const { enabled, filters, user_rules: userRules, interval, whitelist_filters } = filteringStatus;
    const newUserRules = Array.isArray(userRules) ? userRules.join('\n') : '';

    return {
        enabled,
        userRules: newUserRules,
        filters: normalizeFilters(filters),
        whitelistFilters: normalizeFilters(whitelist_filters),
        interval,
    };
};

export const getPercent = (amount: any, number: any) => {
    if (amount > 0 && number > 0) {
        return round(100 / (amount / number), 2);
    }
    return 0;
};

export const captitalizeWords = (text: any) =>
    text
        .split(/[ -_]/g)
        .map((str: any) => str.charAt(0).toUpperCase() + str.substr(1))
        .join(' ');

export const getInterfaceIp = (option: any) => {
    const onlyIPv6 = option.ip_addresses.every((ip: any) => ip.includes(':'));
    let [interfaceIP] = option.ip_addresses;

    if (!onlyIPv6) {
        option.ip_addresses.forEach((ip: any) => {
            if (!ip.includes(':')) {
                interfaceIP = ip;
            }
        });
    }

    return interfaceIP;
};

export const getIpList = (interfaces: DhcpInterface[]) =>
    Object.values(interfaces)
        .reduce((acc: string[], curr: DhcpInterface) => acc.concat(curr.ip_addresses), [] as string[])
        .sort();

/**
 * @param {string} ip
 * @param {number} [port]
 * @returns {string}
 */
export const getDnsAddress = (ip: any, port = 0) => {
    const isStandardDnsPort = port === STANDARD_DNS_PORT;
    let address = ip;

    if (port) {
        if (ip.includes(':') && !isStandardDnsPort) {
            address = `[${ip}]:${port}`;
        } else if (!isStandardDnsPort) {
            address = `${ip}:${port}`;
        }
    }

    return address;
};

/**
 * @param {string} ip
 * @param {number} [port]
 * @returns {string}
 */
export const getWebAddress = (ip: any, port = 0) => {
    const isStandardWebPort = port === STANDARD_WEB_PORT;
    let address = `http://${ip}`;

    if (port && !isStandardWebPort) {
        if (ip.includes(':') && !ip.includes('[')) {
            address = `http://[${ip}]:${port}`;
        } else {
            address = `http://${ip}:${port}`;
        }
    }

    return address;
};

export const checkRedirect = (url: any, attempts: number = 1) => {
    let count = attempts || 1;

    if (count > 10) {
        window.location.replace(url);
        return false;
    }

    const rmTimeout = (t: any) => t && clearTimeout(t);
    const setRecursiveTimeout = (time: any, ...args: any[]) => setTimeout(checkRedirect, time, ...args);

    let timeout: any;

    axios
        .get(url)
        .then((response) => {
            rmTimeout(timeout);
            if (response) {
                window.location.replace(url);
                return;
            }
            timeout = setRecursiveTimeout(CHECK_TIMEOUT, url, (count += 1));
        })
        .catch((error) => {
            rmTimeout(timeout);
            if (error.response) {
                window.location.replace(url);
                return;
            }
            timeout = setRecursiveTimeout(CHECK_TIMEOUT, url, (count += 1));
        });

    return false;
};

export const redirectToCurrentProtocol = (values: any, httpPort = 80) => {
    const { protocol, hostname, hash, port } = window.location;
    const { enabled, force_https, port_https } = values;
    const httpsPort = port_https !== STANDARD_HTTPS_PORT ? `:${port_https}` : '';

    if (protocol !== 'https:' && enabled && force_https && port_https) {
        checkRedirect(`https://${hostname}${httpsPort}/${hash}`);
    } else if (protocol === 'https:' && enabled && port_https && port_https !== parseInt(port, 10)) {
        checkRedirect(`https://${hostname}${httpsPort}/${hash}`);
    } else if (protocol === 'https:' && (!enabled || !port_https)) {
        window.location.replace(`http://${hostname}:${httpPort}/${hash}`);
    }
};

/**
 * @param {string} text
 * @returns []string
 */
export const splitByNewLine = (text: any) => text.split('\n').filter((n: any) => n.trim());

/**
 * @param {string} text
 * @returns {string}
 */
export const trimMultilineString = (text: any) =>
    splitByNewLine(text)
        .map((line: any) => line.trim())
        .join('\n');

/**
 * @param {string} text
 * @returns {string}
 */
export const removeEmptyLines = (text: any) => splitByNewLine(text).join('\n');

/**
 * @param {string} input
 * @returns {string}
 */
export const trimLinesAndRemoveEmpty = (input: any) =>
    input
        .split('\n')
        .map((line: any) => line.trim())
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
export const normalizeTopClients = (topClients: any) =>
    topClients.reduce(
        (acc: any, clientObj: any) => {
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

export const sortClients = (clients: any) => {
    const compare = (a: any, b: any) => {
        const nameA = a.name.toUpperCase();
        const nameB = b.name.toUpperCase();

        if (nameA > nameB) {
            return 1;
        }
        if (nameA < nameB) {
            return -1;
        }

        return 0;
    };

    return clients.sort(compare);
};

export const toggleAllServices = (services: any, change: any, isSelected: any) => {
    services.forEach((service: any) => change(`blocked_services.${service.id}`, isSelected));
};

export const msToSeconds = (milliseconds: any) => Math.floor(milliseconds / 1000);

export const msToMinutes = (milliseconds: any) => Math.floor(milliseconds / 1000 / 60);

export const msToHours = (milliseconds: any) => Math.floor(milliseconds / 1000 / 60 / 60);

export const secondsToMilliseconds = (seconds: any) => {
    if (seconds) {
        return seconds * 1000;
    }

    return seconds;
};

export const msToDays = (milliseconds: any) => Math.floor(milliseconds / 1000 / 60 / 60 / 24);

export const normalizeRulesTextarea = (text: any) => text?.replace(/^\n/g, '').replace(/\n\s*\n/g, '\n');

export const normalizeWhois = (whois: any) => {
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

    return whois;
};

export const getPathWithQueryString = (path: any, params: any) => {
    const searchParams = new URLSearchParams(params);

    return `${path}?${searchParams.toString()}`;
};

export const getParamsForClientsSearch = (data: any, param: any, additionalParam?: any) => {
    const clients = new Set();
    data.forEach((e: any) => {
        clients.add(e[param]);
        if (e[additionalParam]) {
            clients.add(e[additionalParam]);
        }
    });

    return {
        clients: Array.from(clients).map((id) => ({ id })),
    };
};

/**
 * Creates onBlur handler that can normalize input if normalization function is specified
 *
 * @param {Object} event
 * @param {Object} event.target
 * @param {string} event.target.value
 * @param {Object} input
 * @param {function} input.onBlur
 * @param {function} [normalizeOnBlur]
 * @returns {function}
 */
export const createOnBlurHandler = (event: any, input: any, normalizeOnBlur: any) =>
    normalizeOnBlur ? input.onBlur(normalizeOnBlur(event.target.value)) : input.onBlur();

export const checkFiltered = (reason: any) => reason.indexOf(FILTERED) === 0;
export const checkRewrite = (reason: any) => reason === FILTERED_STATUS.REWRITE;
export const checkRewriteHosts = (reason: any) => reason === FILTERED_STATUS.REWRITE_HOSTS;
export const checkBlackList = (reason: any) => reason === FILTERED_STATUS.FILTERED_BLACK_LIST;
export const checkWhiteList = (reason: any) => reason === FILTERED_STATUS.NOT_FILTERED_WHITE_LIST;
// eslint-disable-next-line max-len
export const checkNotFilteredNotFound = (reason: any) => reason === FILTERED_STATUS.NOT_FILTERED_NOT_FOUND;
export const checkSafeSearch = (reason: any) => reason === FILTERED_STATUS.FILTERED_SAFE_SEARCH;
export const checkSafeBrowsing = (reason: any) => reason === FILTERED_STATUS.FILTERED_SAFE_BROWSING;
export const checkParental = (reason: any) => reason === FILTERED_STATUS.FILTERED_PARENTAL;
export const checkBlockedService = (reason: any) => reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

export const getCurrentFilter = (url: any, filters: any) => {
    const filter = filters?.find((item: any) => url === item.url);

    if (filter) {
        const { enabled, name, url } = filter;
        return {
            enabled,
            name,
            url,
        };
    }

    return {
        name: '',
        url: '',
    };
};

/**
 * @param {object} initialValues
 * @param {object} values
 * @returns {object} Returns different values of objects
 */

export const getObjDiff = (initialValues: any, values: any) =>
    Object.entries(values)

        .reduce((acc: any, [key, value]) => {
            if (value !== initialValues[key]) {
                acc[key] = value;
            }
            return acc;
        }, {});

/**
 * @param num {number} to format
 * @returns {string} Returns a string with a language-sensitive representation of this number
 */
export const formatNumber = (num: number): string => {
    const currentLanguage = i18n.languages[0] || DEFAULT_LANGUAGE;
    return num.toLocaleString(currentLanguage);
};

/**
 * @param arr {array}
 * @param key {string}
 * @param value {string}
 * @returns {object}
 */
export const getMap = (arr: any, key: any, value: any) =>
    arr.reduce((acc: any, curr: any) => {
        acc[curr[key]] = curr[value];
        return acc;
    }, {});

/**
 * @param parsedIp {object} ipaddr.js IPv4 or IPv6 object
 * @param parsedCidr {array} ipaddr.js CIDR array
 * @returns {boolean}
 */
const isIpMatchCidr = (parsedIp: any, parsedCidr: any) => {
    try {
        const cidrIpVersion = parsedCidr[0].kind();
        const ipVersion = parsedIp.kind();

        return ipVersion === cidrIpVersion && parsedIp.match(parsedCidr);
    } catch (e) {
        return false;
    }
};

export const isIpInCidr = (ip: any, cidr: any) => {
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
 *
 * @param {string} subnetMask
 * @returns {IPv4 | null}
 */
export const parseSubnetMask = (subnetMask: any) => {
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
export const subnetMaskToBitMask = (subnetMask: any) =>
    subnetMask.split('.').reduce((acc: any, cur: any) => acc - Math.log2(256 - Number(cur)), 32);

/**
 *
 * @param ipOrCidr
 * @returns {'IP' | 'CIDR' | 'CLIENT_ID' | 'UNKNOWN'}
 *
 */
export const findAddressType = (address: any) => {
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
    } catch (e) {
        return ADDRESS_TYPES.UNKNOWN;
    }
};

/**
 * @param ids {string[]}
 * @returns {Object}
 */
export const separateIpsAndCidrs = (ids: any) =>
    ids.reduce(
        (acc: any, curr: any) => {
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

export const countClientsStatistics = (ids: any, autoClients: any) => {
    const { ips, cidrs, clientIds } = separateIpsAndCidrs(ids);

    const ipsCount = ips.reduce((acc: any, curr: any) => {
        const count = autoClients[curr] || 0;
        return acc + count;
    }, 0);

    const clientIdsCount = clientIds.reduce((acc: any, curr: any) => {
        const count = autoClients[curr] || 0;
        return acc + count;
    }, 0);

    const cidrsCount = Object.entries(autoClients).reduce((acc: any, curr: any) => {
        const [id, count] = curr;
        if (!ipaddr.isValid(id)) {
            return false;
        }
        if (cidrs.some((cidr: any) => isIpInCidr(id, cidr))) {
            // eslint-disable-next-line no-param-reassign
            acc += count;
        }
        return acc;
    }, 0);

    return ipsCount + cidrsCount + clientIdsCount;
};

/**
 * @param {string} elapsedMs
 * @param {function} t translate
 * @returns {string}
 */
export const formatElapsedMs = (elapsedMs: string, t: (key: string) => string) => {
    const parsedElapsedMs = parseInt(elapsedMs, 10);

    if (Number.isNaN(parsedElapsedMs)) {
        return elapsedMs;
    }

    const formattedMs = formatNumber(parsedElapsedMs);

    return `${formattedMs} ${t('milliseconds_abbreviation')}`;
};

/**
 * @param language {string}
 */
export const setHtmlLangAttr = (language: any) => {
    window.document.documentElement.lang = language;
};

/**
 * Set local storage theme field
 *
 * @param {string} theme
 */
export const setTheme = (theme: any) => {
    LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.THEME, theme);
};

/**
 * Get local storage theme field
 *
 * @returns {string}
 */

export const getTheme = () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.THEME) || THEMES.light;

/**
 * Sets UI theme.
 *
 * @param theme
 */
export const setUITheme = (theme: any) => {
    let currentTheme = theme || getTheme();

    if (currentTheme === THEMES.auto) {
        const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        currentTheme = prefersDark ? THEMES.dark : THEMES.light;
    }
    setTheme(currentTheme);
    document.body.dataset.theme = currentTheme;
};

/**
 * @param values {object}
 * @returns {object}
 */

export const replaceEmptyStringsWithZeroes = (values: any) =>
    Object.entries(values)

        .reduce((acc: any, [key, value]) => {
            acc[key] = value === '' ? 0 : value;
            return acc;
        }, {});

/**
 * @param value {number || string}
 * @returns {string}
 */
export const replaceZeroWithEmptyString = (value: any) => (parseInt(value, 10) === 0 ? '' : value);

/**
 * @param {string} search
 * @param {string} [response_status]
 * @returns {string}
 */
export const getLogsUrlParams = (search: any, response_status: any) =>
    `?${queryString.stringify({
        search: search || undefined,
        response_status: response_status || undefined,
    })}`;

export const processContent = (content: any) =>
    Array.isArray(content) ? content.filter(([, value]) => value).reduce((acc, val) => acc.concat(val), []) : content;

// TODO check getObjectKeysSorted
type NestedObject = {
    [key: string]: any;
    order: number;
};

export const getObjectKeysSorted = <T extends Record<string, NestedObject>, K extends keyof NestedObject>(
    object: T,
    sortKey: K,
): string[] => {
    return Object.entries(object)
        .sort(([, a], [, b]) => (a[sortKey] as number) - (b[sortKey] as number))
        .map(([key]) => key);
};

/**
 * @param ip
 * @returns {[IPv4|IPv6, 33|129]}
 */
const getParsedIpWithPrefixLength = (ip: any) => {
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
const getAddressesComparisonBytes = (item: any) => {
    // Sort ipv4 before ipv6
    const IP_V4_COMPARISON_CODE = 0;
    const IP_V6_COMPARISON_CODE = 1;

    const [parsedIp, cidr] = ipaddr.isValid(item) ? getParsedIpWithPrefixLength(item) : ipaddr.parseCIDR(item);

    const [normalizedBytes, ipVersionComparisonCode] =
        (parsedIp as IPv4 | IPv6).kind() === 'ipv4'
            ? [(parsedIp as IPv4).toIPv4MappedAddress().parts, IP_V4_COMPARISON_CODE]
            : [(parsedIp as IPv6).parts, IP_V6_COMPARISON_CODE];

    return [ipVersionComparisonCode, ...normalizedBytes, cidr];
};

/**
 * Compare function for IP and CIDR sort in ascending order (supports both v4 and v6)
 * @param a
 * @param b
 * @returns {number} -1 | 0 | 1
 */
export const sortIp = (a: any, b: any) => {
    try {
        const comparisonBytesA = Array.isArray(a) ? getAddressesComparisonBytes(a[0]) : getAddressesComparisonBytes(a);
        const comparisonBytesB = Array.isArray(b) ? getAddressesComparisonBytes(b[0]) : getAddressesComparisonBytes(b);

        for (let i = 0; i < comparisonBytesA.length; i += 1) {
            const byteA = comparisonBytesA[i];
            const byteB = comparisonBytesB[i];

            if (byteA === byteB) {
                // eslint-disable-next-line no-continue
                continue;
            }
            return byteA > byteB ? 1 : -1;
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
export const getSpecialFilterName = (filterId: any) => {
    switch (filterId) {
        case SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES:
            return i18n.t('custom_filter_rules');
        case SPECIAL_FILTER_ID.SYSTEM_HOSTS:
            return i18n.t('system_host_files');
        case SPECIAL_FILTER_ID.BLOCKED_SERVICES:
            return i18n.t('blocked_services');
        case SPECIAL_FILTER_ID.PARENTAL:
            return i18n.t('parental_control');
        case SPECIAL_FILTER_ID.SAFE_BROWSING:
            return i18n.t('safe_browsing');
        case SPECIAL_FILTER_ID.SAFE_SEARCH:
            return i18n.t('safe_search');
        default:
            return i18n.t('unknown_filter', { filterId });
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
    filter_list_id: number;
    text: string;
};

export const getFilterName = (
    filters: Filter[],
    whitelistFilters: Filter[],
    filterId: number,
    resolveFilterName = (filter: Filter) => (filter ? filter.name : i18n.t('unknown_filter', { filterId })),
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
    rules.map(({ filter_list_id }: any) => getFilterName(filters, whitelistFilters, filter_list_id));

export const getRuleNames = (rules: Rule[]) => rules.map(({ text }: Rule) => text);

export const getFilterNameToRulesMap = (rules: Rule[], filters: Filter[], whitelistFilters: Filter[]) =>
    rules.reduce((acc: any, { text, filter_list_id }: Rule) => {
        const filterName = getFilterName(filters, whitelistFilters, filter_list_id);

        acc[filterName] = (acc[filterName] || []).concat(text);
        return acc;
    }, {});

export const getRulesToFilterList = (
    rules: Rule[],
    filters: Filter[],
    whitelistFilters: Filter[],
    classes = {
        list: 'filteringRules',
        rule: 'filteringRules__rule font-monospace',
        filter: 'filteringRules__filter',
    },
) => {
    const filterNameToRulesMap: { string: string[] } = getFilterNameToRulesMap(rules, filters, whitelistFilters);

    return (
        <dl className={classes.list}>
            {Object.entries(filterNameToRulesMap).reduce(
                (acc: any, [filterName, rulesArr]) =>
                    acc
                        .concat(
                            rulesArr.map((rule: any, i: any) => (
                                <dd key={i} className={classes.rule}>
                                    {rule}
                                </dd>
                            )),
                        )
                        .concat(
                            <dt className={classes.filter} key={classes.filter}>
                                {filterName}
                            </dt>,
                        ),
                [],
            )}
        </dl>
    );
};

/**
 * @param ip {string}
 * @param gateway_ip {string}
 * @returns {{range_end: string, subnet_mask: string, range_start: string,
 * lease_duration: string, gateway_ip: string}}
 */
export const calculateDhcpPlaceholdersIpv4 = (ip: string, gateway_ip: string) => {
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
        gateway_ip: gateway_ip || ip,
        subnet_mask,
        range_start,
        range_end,
        lease_duration,
    };
};

export const calculateDhcpPlaceholdersIpv6 = () => {
    const { range_start, range_end, lease_duration } = DHCP_VALUES_PLACEHOLDERS.ipv6;

    return {
        range_start,
        range_end,
        lease_duration,
    };
};

/**
 * Add ip_addresses property - concatenated ipv4_addresses and ipv6_addresses for every interface
 * @param interfaces
 * @param interfaces.ipv4_addresses {string[]}
 * @param interfaces.ipv6_addresses {string[]}
 * @returns interfaces Interfaces enriched with ip_addresses property
 */

export const enrichWithConcatenatedIpAddresses = (interfaces: DhcpInterface[]) =>
    Object.entries(interfaces)

        .reduce((acc: any, [k, v]) => {
            const ipv4_addresses = v.ipv4_addresses ?? [];
            const ipv6_addresses = v.ipv6_addresses ?? [];

            acc[k].ip_addresses = ipv4_addresses.concat(ipv6_addresses);
            return acc;
        }, interfaces);

export const isScrolledIntoView = (el: any) => {
    const rect = el.getBoundingClientRect();
    const elemTop = rect.top;
    const elemBottom = rect.bottom;

    return elemTop < window.innerHeight && elemBottom >= 0;
};

/**
 * If this is a manually created client, return its name.
 * If this is a "runtime" client, return it's IP address.
 * @param clients {Array.<object>}
 * @param ip {string}
 * @returns {string}
 */
export const getBlockingClientName = (clients: any, ip: any) => {
    for (let i = 0; i < clients.length; i += 1) {
        const client = clients[i];

        if (client.ids.includes(ip)) {
            return client.name;
        }
    }
    return ip;
};

/**
 * @param {string[]} lines
 * @returns {string[]}
 */
export const filterOutComments = (lines: any) =>
    lines.filter((line: any) => !line.startsWith(COMMENT_LINE_DEFAULT_TOKEN));

/**
 * @param {array} services
 * @param {string} id
 * @returns {string}
 */
export const getService = (services: any, id: any) => services.find((s: any) => s.id === id);

/**
 * @param {array} services
 * @param {string} id
 * @returns {string}
 */
export const getServiceName = (services: any, id: any) => getService(services, id)?.name;

/**
 * @param {array} services
 * @param {string} id
 * @returns {string}
 */
export const getServiceIcon = (services: any, id: any) => getService(services, id)?.icon_svg;
