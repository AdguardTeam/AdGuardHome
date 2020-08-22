import 'url-polyfill';
import dateParse from 'date-fns/parse';
import dateFormat from 'date-fns/format';
import subHours from 'date-fns/sub_hours';
import addHours from 'date-fns/add_hours';
import addDays from 'date-fns/add_days';
import subDays from 'date-fns/sub_days';
import round from 'lodash/round';
import axios from 'axios';
import i18n from 'i18next';
import uniqBy from 'lodash/uniqBy';
import ipaddr from 'ipaddr.js';
import queryString from 'query-string';
import { getTrackerData } from './trackers/trackers';

import {
    CHECK_TIMEOUT,
    DEFAULT_DATE_FORMAT_OPTIONS,
    DEFAULT_LANGUAGE,
    DEFAULT_TIME_FORMAT,
    DETAILED_DATE_FORMAT_OPTIONS,
    DHCP_VALUES_PLACEHOLDERS,
    FILTERED,
    FILTERED_STATUS,
    IP_MATCH_LIST_STATUS,
    STANDARD_DNS_PORT,
    STANDARD_HTTPS_PORT,
    STANDARD_WEB_PORT,
} from './constants';

/**
 * @param time {string} The time to format
 * @param options {string}
 * @returns {string} Returns the time in the format HH:mm:ss
 */
export const formatTime = (time, options = DEFAULT_TIME_FORMAT) => {
    const parsedTime = dateParse(time);
    return dateFormat(parsedTime, options);
};

/**
 * @param dateTime {string} The date to format
 * @param [options] {object} Date.prototype.toLocaleString([locales[, options]]) options argument
 * @returns {string} Returns the date and time in the specified format
 */
export const formatDateTime = (dateTime, options = DEFAULT_DATE_FORMAT_OPTIONS) => {
    const { language } = navigator;
    const currentLanguage = (language.slice(0, 2) === 'en' || !language) ? 'en-GB' : language;

    const parsedTime = new Date(dateTime);

    return parsedTime.toLocaleString(currentLanguage, options);
};

/**
 * @param dateTime {string} The date to format
 * @returns {string} Returns the date and time in the format with the full month name
 */
export const formatDetailedDateTime = (dateTime) => formatDateTime(
    dateTime, DETAILED_DATE_FORMAT_OPTIONS,
);

export const normalizeLogs = (logs) => logs.map((log) => {
    const {
        answer,
        answer_dnssec,
        client,
        client_proto,
        elapsedMs,
        question,
        reason,
        status,
        time,
        filterId,
        rule,
        service_name,
        original_answer,
        upstream,
    } = log;

    const { host: domain, type } = question;

    const processResponse = (data) => (data ? data.map((response) => {
        const { value, type, ttl } = response;
        return `${type}: ${value} (ttl=${ttl})`;
    }) : []);

    return {
        time,
        domain,
        type,
        response: processResponse(answer),
        reason,
        client,
        client_proto,
        filterId,
        rule,
        status,
        serviceName: service_name,
        originalAnswer: original_answer,
        originalResponse: processResponse(original_answer),
        tracker: getTrackerData(domain),
        answer_dnssec,
        elapsedMs,
        upstream,
    };
});

export const normalizeHistory = (history, interval) => {
    if (interval === 1 || interval === 7) {
        const hoursAgo = subHours(Date.now(), 24 * interval);
        return history.map((item, index) => ({
            x: dateFormat(addHours(hoursAgo, index), 'D MMM HH:00'),
            y: round(item, 2),
        }));
    }

    const daysAgo = subDays(Date.now(), interval - 1);
    return history.map((item, index) => ({
        x: dateFormat(addDays(daysAgo, index), 'D MMM YYYY'),
        y: round(item, 2),
    }));
};

export const normalizeTopStats = (stats) => (
    stats.map((item) => ({
        name: Object.keys(item)[0],
        count: Object.values(item)[0],
    }))
);

export const addClientInfo = (data, clients, param) => (
    data.map((row) => {
        const clientIp = row[param];
        const info = clients.find((item) => item[clientIp]) || '';
        return {
            ...row,
            info: info?.[clientIp] ?? '',
        };
    })
);

export const normalizeFilters = (filters) => (
    filters ? filters.map((filter) => {
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
    }) : []
);

export const normalizeFilteringStatus = (filteringStatus) => {
    const {
        enabled, filters, user_rules: userRules, interval, whitelist_filters,
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

export const getPercent = (amount, number) => {
    if (amount > 0 && number > 0) {
        return round(100 / (amount / number), 2);
    }
    return 0;
};

export const captitalizeWords = (text) => text.split(/[ -_]/g)
    .map((str) => str.charAt(0)
        .toUpperCase() + str.substr(1))
    .join(' ');

export const getInterfaceIp = (option) => {
    const onlyIPv6 = option.ip_addresses.every((ip) => ip.includes(':'));
    let [interfaceIP] = option.ip_addresses;

    if (!onlyIPv6) {
        option.ip_addresses.forEach((ip) => {
            if (!ip.includes(':')) {
                interfaceIP = ip;
            }
        });
    }

    return interfaceIP;
};

export const getIpList = (interfaces) => Object.values(interfaces)
    .reduce((acc, curr) => acc.concat(curr.ip_addresses), [])
    .sort();

export const getDnsAddress = (ip, port = '') => {
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

export const getWebAddress = (ip, port = '') => {
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

export const checkRedirect = (url, attempts) => {
    let count = attempts || 1;

    if (count > 10) {
        window.location.replace(url);
        return false;
    }

    const rmTimeout = (t) => t && clearTimeout(t);
    const setRecursiveTimeout = (time, ...args) => setTimeout(
        checkRedirect,
        time,
        ...args,
    );

    let timeout;

    axios.get(url)
        .then((response) => {
            rmTimeout(timeout);
            if (response) {
                window.location.replace(url);
                return;
            }
            timeout = setRecursiveTimeout(CHECK_TIMEOUT, url, count += 1);
        })
        .catch((error) => {
            rmTimeout(timeout);
            if (error.response) {
                window.location.replace(url);
                return;
            }
            timeout = setRecursiveTimeout(CHECK_TIMEOUT, url, count += 1);
        });

    return false;
};

export const redirectToCurrentProtocol = (values, httpPort = 80) => {
    const {
        protocol, hostname, hash, port,
    } = window.location;
    const { enabled, port_https } = values;
    const httpsPort = port_https !== STANDARD_HTTPS_PORT ? `:${port_https}` : '';

    if (protocol !== 'https:' && enabled && port_https) {
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
export const splitByNewLine = (text) => text.split('\n')
    .filter((n) => n.trim());

/**
 * @param {string} text
 * @returns {string}
 */
export const trimMultilineString = (text) => splitByNewLine(text)
    .map((line) => line.trim())
    .join('\n');

/**
 * @param {string} text
 * @returns {string}
 */
export const removeEmptyLines = (text) => splitByNewLine(text)
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
export const normalizeTopClients = (topClients) => topClients.reduce(
    (acc, clientObj) => {
        const { name, count, info: { name: infoName } } = clientObj;
        acc.auto[name] = count;
        acc.configured[infoName] = count;
        return acc;
    }, {
        auto: {},
        configured: {},
    },
);

export const sortClients = (clients) => {
    const compare = (a, b) => {
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

export const toggleAllServices = (services, change, isSelected) => {
    services.forEach((service) => change(`blocked_services.${service.id}`, isSelected));
};

export const secondsToMilliseconds = (seconds) => {
    if (seconds) {
        return seconds * 1000;
    }

    return seconds;
};

export const normalizeRulesTextarea = (text) => text?.replace(/^\n/g, '')
    .replace(/\n\s*\n/g, '\n');

export const normalizeWhois = (whois) => {
    if (Object.keys(whois).length > 0) {
        const {
            city, country, ...values
        } = whois;
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

export const getPathWithQueryString = (path, params) => {
    const searchParams = new URLSearchParams(params);

    return `${path}?${searchParams.toString()}`;
};

export const getParamsForClientsSearch = (data, param) => {
    const uniqueClients = uniqBy(data, param);
    return uniqueClients
        .reduce((acc, item, idx) => {
            const key = `ip${idx}`;
            acc[key] = item[param];
            return acc;
        }, {});
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
export const createOnBlurHandler = (event, input, normalizeOnBlur) => (
    normalizeOnBlur
        ? input.onBlur(normalizeOnBlur(event.target.value))
        : input.onBlur());

export const checkFiltered = (reason) => reason.indexOf(FILTERED) === 0;
export const checkRewrite = (reason) => reason === FILTERED_STATUS.REWRITE;
export const checkRewriteHosts = (reason) => reason === FILTERED_STATUS.REWRITE_HOSTS;
export const checkBlackList = (reason) => reason === FILTERED_STATUS.FILTERED_BLACK_LIST;
export const checkWhiteList = (reason) => reason === FILTERED_STATUS.NOT_FILTERED_WHITE_LIST;
// eslint-disable-next-line max-len
export const checkNotFilteredNotFound = (reason) => reason === FILTERED_STATUS.NOT_FILTERED_NOT_FOUND;
export const checkSafeSearch = (reason) => reason === FILTERED_STATUS.FILTERED_SAFE_SEARCH;
export const checkSafeBrowsing = (reason) => reason === FILTERED_STATUS.FILTERED_SAFE_BROWSING;
export const checkParental = (reason) => reason === FILTERED_STATUS.FILTERED_PARENTAL;
export const checkBlockedService = (reason) => reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

export const getCurrentFilter = (url, filters) => {
    const filter = filters?.find((item) => url === item.url);

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
export const getObjDiff = (initialValues, values) => Object.entries(values)
    .reduce((acc, [key, value]) => {
        if (value !== initialValues[key]) {
            acc[key] = value;
        }
        return acc;
    }, {});

/**
 * @param num {number} to format
 * @returns {string} Returns a string with a language-sensitive representation of this number
 */
export const formatNumber = (num) => {
    const currentLanguage = i18n.languages[0] || DEFAULT_LANGUAGE;
    return num.toLocaleString(currentLanguage);
};

/**
 * @param arr {array}
 * @param key {string}
 * @param value {string}
 * @returns {object}
 */
export const getMap = (arr, key, value) => arr.reduce((acc, curr) => {
    acc[curr[key]] = curr[value];
    return acc;
}, {});

/**
 * @param parsedIp {object} ipaddr.js IPv4 or IPv6 object
 * @param parsedCidr {array} ipaddr.js CIDR array
 * @returns {boolean}
 */
const isIpMatchCidr = (parsedIp, parsedCidr) => {
    try {
        const cidrIpVersion = parsedCidr[0].kind();
        const ipVersion = parsedIp.kind();

        return ipVersion === cidrIpVersion && parsedIp.match(parsedCidr);
    } catch (e) {
        return false;
    }
};

/**
 * The purpose of this method is to quickly check
 * if this IP can possibly be in the specified CIDR range.
 *
 * @param ip {string}
 * @param listItem {string}
 * @returns {boolean}
 */
const isIpQuickMatchCIDR = (ip, listItem) => {
    const ipv6 = ip.indexOf(':') !== -1;
    const cidrIpv6 = listItem.indexOf(':') !== -1;
    if (ipv6 !== cidrIpv6) {
        // CIDR is for a different IP type
        return false;
    }

    if (cidrIpv6) {
        // We don't do quick check for IPv6 addresses
        return true;
    }

    const idx = listItem.indexOf('/');
    if (idx === -1) {
        // Not a CIDR, return false immediately
        return false;
    }

    const cidrIp = listItem.substring(0, idx);
    const cidrRange = parseInt(listItem.substring(idx + 1), 10);
    if (Number.isNaN(cidrRange)) {
        // Not a valid CIDR
        return false;
    }

    const parts = cidrIp.split('.');
    if (parts.length !== 4) {
        // Invalid IP, return immediately
        return false;
    }

    // Now depending on the range we check if the IP can possibly be in that range
    if (cidrRange < 8) {
        // Use the slow approach
        return true;
    }

    if (cidrRange < 16) {
        // Check the first part
        // Example: 0.0.0.0/8 matches 0.*.*.*
        return ip.indexOf(`${parts[0]}.`) === 0;
    }

    if (cidrRange < 24) {
        // Check the first two parts
        // Example: 0.0.0.0/16 matches 0.0.*.*
        return ip.indexOf(`${parts[0]}.${parts[1]}.`) === 0;
    }

    if (cidrRange <= 32) {
        // Check the first two parts
        // Example: 0.0.0.0/16 matches 0.0.*.*
        return ip.indexOf(`${parts[0]}.${parts[1]}.${parts[2]}.`) === 0;
    }

    // range for IPv4 CIDR cannot be more than 32
    // no need to check further, this CIDR is invalid
    return false;
};

/**
 * @param ip {string}
 * @param list {string}
 * @returns {'EXACT' | 'CIDR' | 'NOT_FOND'}
 */
export const getIpMatchListStatus = (ip, list) => {
    if (!ip || !list) {
        return IP_MATCH_LIST_STATUS.NOT_FOUND;
    }

    const listArr = list.trim()
        .split('\n');

    try {
        for (let i = 0; i < listArr.length; i += 1) {
            const listItem = listArr[i];

            if (ip === listItem.trim()) {
                return IP_MATCH_LIST_STATUS.EXACT;
            }

            // Using ipaddr.js is quite slow so we first do a quick check
            // to see if it's possible that this IP may be in the specified CIDR range
            if (isIpQuickMatchCIDR(ip, listItem)) {
                const parsedIp = ipaddr.parse(ip);
                const isItemAnIp = ipaddr.isValid(listItem);
                const parsedItem = isItemAnIp ? ipaddr.parse(listItem) : ipaddr.parseCIDR(listItem);

                if (isItemAnIp && parsedIp.toString() === parsedItem.toString()) {
                    return IP_MATCH_LIST_STATUS.EXACT;
                }

                if (!isItemAnIp && isIpMatchCidr(parsedIp, parsedItem)) {
                    return IP_MATCH_LIST_STATUS.CIDR;
                }
            }
        }
        return IP_MATCH_LIST_STATUS.NOT_FOUND;
    } catch (e) {
        console.error(e);
        return IP_MATCH_LIST_STATUS.NOT_FOUND;
    }
};


/**
 * @param {string} elapsedMs
 * @param {function} t translate
 * @returns {string}
 */
export const formatElapsedMs = (elapsedMs, t) => {
    const formattedElapsedMs = parseInt(elapsedMs, 10) || parseFloat(elapsedMs)
        .toFixed(2);
    return `${formattedElapsedMs} ${t('milliseconds_abbreviation')}`;
};

/**
 * @param language {string}
 */
export const setHtmlLangAttr = (language) => {
    window.document.documentElement.lang = language;
};

/**
 * @param values {object}
 * @returns {object}
 */
export const selectCompletedFields = (values) => Object.entries(values)
    .reduce((acc, [key, value]) => {
        if (value || value === 0) {
            acc[key] = value;
        }
        return acc;
    }, {});

/**
 * @param {string} search
 * @param {string} [response_status]
 * @returns {string}
 */
export const getLogsUrlParams = (search, response_status) => `?${queryString.stringify({
    search,
    response_status,
})}`;

export const processContent = (
    content,
) => (Array.isArray(content)
    ? content.filter(([, value]) => value)
        .reduce((acc, val) => acc.concat(val), [])
    : content);
/**
 * @param object {object}
 * @param sortKey {string}
 * @returns {string[]}
 */
export const getObjectKeysSorted = (object, sortKey) => Object.entries(object)
    .sort(([, { [sortKey]: order1 }], [, { [sortKey]: order2 }]) => order1 - order2)
    .map(([key]) => key);

/**
 * @param ip
 * @returns {[IPv4|IPv6, 33|129]}
 */
const getParsedIpWithPrefixLength = (ip) => {
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
const getAddressesComparisonBytes = (item) => {
    // Sort ipv4 before ipv6
    const IP_V4_COMPARISON_CODE = 0;
    const IP_V6_COMPARISON_CODE = 1;

    const [parsedIp, cidr] = ipaddr.isValid(item)
        ? getParsedIpWithPrefixLength(item)
        : ipaddr.parseCIDR(item);

    const [normalizedBytes, ipVersionComparisonCode] = parsedIp.kind() === 'ipv4'
        ? [parsedIp.toIPv4MappedAddress().parts, IP_V4_COMPARISON_CODE]
        : [parsedIp.parts, IP_V6_COMPARISON_CODE];

    return [ipVersionComparisonCode, ...normalizedBytes, cidr];
};

/**
 * Compare function for IP and CIDR sort in ascending order (supports both v4 and v6)
 * @param a
 * @param b
 * @returns {number} -1 | 0 | 1
 */
export const sortIp = (a, b) => {
    try {
        const comparisonBytesA = getAddressesComparisonBytes(a);
        const comparisonBytesB = getAddressesComparisonBytes(b);

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
        console.error(e);
        return 0;
    }
};

/**
 * @param ip {string}
 * @param gateway_ip {string}
 * @returns {{range_end: string, subnet_mask: string, range_start: string,
 * lease_duration: string, gateway_ip: string}}
 */
export const calculateDhcpPlaceholdersIpv4 = (ip, gateway_ip) => {
    const LAST_OCTET_IDX = 3;
    const LAST_OCTET_RANGE_START = 100;
    const LAST_OCTET_RANGE_END = 200;

    const addr = ipaddr.parse(ip);
    addr.octets[LAST_OCTET_IDX] = LAST_OCTET_RANGE_START;
    const range_start = addr.toString();

    addr.octets[LAST_OCTET_IDX] = LAST_OCTET_RANGE_END;
    const range_end = addr.toString();

    const {
        subnet_mask,
        lease_duration,
    } = DHCP_VALUES_PLACEHOLDERS.ipv4;

    return {
        gateway_ip: gateway_ip || ip,
        subnet_mask,
        range_start,
        range_end,
        lease_duration,
    };
};

export const calculateDhcpPlaceholdersIpv6 = () => {
    const {
        range_start,
        range_end,
        lease_duration,
    } = DHCP_VALUES_PLACEHOLDERS.ipv6;

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
export const enrichWithConcatenatedIpAddresses = (interfaces) => Object.entries(interfaces)
    .reduce((acc, [k, v]) => {
        const ipv4_addresses = v.ipv4_addresses ?? [];
        const ipv6_addresses = v.ipv6_addresses ?? [];

        acc[k].ip_addresses = ipv4_addresses.concat(ipv6_addresses);
        return acc;
    }, interfaces);
