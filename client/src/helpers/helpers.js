import dateParse from 'date-fns/parse';
import dateFormat from 'date-fns/format';
import subHours from 'date-fns/sub_hours';
import addHours from 'date-fns/add_hours';
import addDays from 'date-fns/add_days';
import subDays from 'date-fns/sub_days';
import round from 'lodash/round';
import axios from 'axios';
import i18n from 'i18next';

import {
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
    STANDARD_HTTPS_PORT,
    CHECK_TIMEOUT,
} from './constants';

export const formatTime = (time) => {
    const parsedTime = dateParse(time);
    return dateFormat(parsedTime, 'HH:mm:ss');
};

export const formatDateTime = (dateTime) => {
    const currentLanguage = i18n.languages[0] || 'en';
    const parsedTime = dateParse(dateTime);
    const options = {
        year: 'numeric',
        month: 'numeric',
        day: 'numeric',
        hour: 'numeric',
        minute: 'numeric',
        hour12: false,
    };

    return parsedTime.toLocaleString(currentLanguage, options);
};

export const normalizeLogs = logs => logs.map((log) => {
    const {
        time,
        question,
        answer: response,
        reason,
        client,
        filterId,
        rule,
        service_name,
        status,
    } = log;
    const { host: domain, type } = question;
    const responsesArray = response ? response.map((response) => {
        const { value, type, ttl } = response;
        return `${type}: ${value} (ttl=${ttl})`;
    }) : [];
    return {
        time,
        domain,
        type,
        response: responsesArray,
        reason,
        client,
        filterId,
        rule,
        serviceName: service_name,
        status,
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

export const normalizeTopStats = stats => (
    stats.map(item => ({
        name: Object.keys(item)[0],
        count: Object.values(item)[0],
    }))
);

export const normalizeFilteringStatus = (filteringStatus) => {
    const {
        enabled, filters, user_rules: userRules, interval,
    } = filteringStatus;
    const newFilters = filters
        ? filters.map((filter) => {
            const {
                id,
                url,
                enabled,
                last_updated,
                name = 'Default name',
                rules_count: rules_count = 0,
            } = filter;

            return {
                id,
                url,
                enabled,
                lastUpdated: last_updated ? formatDateTime(last_updated) : '–',
                name,
                rulesCount: rules_count,
            };
        })
        : [];
    const newUserRules = Array.isArray(userRules) ? userRules.join('\n') : '';

    return {
        enabled,
        userRules: newUserRules,
        filters: newFilters,
        interval,
    };
};

export const getPercent = (amount, number) => {
    if (amount > 0 && number > 0) {
        return round(100 / (amount / number), 2);
    }
    return 0;
};

export const captitalizeWords = text => text.split(/[ -_]/g).map(str => str.charAt(0).toUpperCase() + str.substr(1)).join(' ');

export const getInterfaceIp = (option) => {
    const onlyIPv6 = option.ip_addresses.every(ip => ip.includes(':'));
    let interfaceIP = option.ip_addresses[0];

    if (!onlyIPv6) {
        option.ip_addresses.forEach((ip) => {
            if (!ip.includes(':')) {
                interfaceIP = ip;
            }
        });
    }

    return interfaceIP;
};

export const getIpList = (interfaces) => {
    let list = [];

    Object.keys(interfaces).forEach((item) => {
        list = [...list, ...interfaces[item].ip_addresses];
    });

    return list.sort();
};

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

    const rmTimeout = t => t && clearTimeout(t);
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

export const normalizeTextarea = text => text && text.replace(/[;, ]/g, '\n').split('\n').filter(n => n);

export const getClientName = (clients, ip) => {
    const client = clients.find(item => ip === item.ip);
    return (client && client.name) || '';
};

export const sortClients = (clients) => {
    const compare = (a, b) => {
        const nameA = a.name.toUpperCase();
        const nameB = b.name.toUpperCase();

        if (nameA > nameB) {
            return 1;
        } else if (nameA < nameB) {
            return -1;
        }

        return 0;
    };

    return clients.sort(compare);
};

export const toggleAllServices = (services, change, isSelected) => {
    services.forEach(service => change(`blocked_services.${service.id}`, isSelected));
};

export const secondsToMilliseconds = (seconds) => {
    if (seconds) {
        return seconds * 1000;
    }

    return seconds;
};

export const normalizeRulesTextarea = text => text && text.replace(/^\n/g, '').replace(/\n\s*\n/g, '\n');
