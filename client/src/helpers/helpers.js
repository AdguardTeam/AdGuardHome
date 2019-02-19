import dateParse from 'date-fns/parse';
import dateFormat from 'date-fns/format';
import subHours from 'date-fns/sub_hours';
import addHours from 'date-fns/add_hours';
import round from 'lodash/round';

import { STATS_NAMES, STANDARD_DNS_PORT, STANDARD_WEB_PORT } from './constants';

export const formatTime = (time) => {
    const parsedTime = dateParse(time);
    return dateFormat(parsedTime, 'HH:mm:ss');
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
    };
});

export const normalizeHistory = history => Object.keys(history).map((key) => {
    let id = STATS_NAMES[key];
    if (!id) {
        id = key.replace(/_/g, ' ').replace(/^\w/, c => c.toUpperCase());
    }

    const dayAgo = subHours(Date.now(), 24);

    const data = history[key].map((item, index) => {
        const formatHour = dateFormat(addHours(dayAgo, index), 'ddd HH:00');
        const roundValue = round(item, 2);

        return {
            x: formatHour,
            y: roundValue,
        };
    });

    return {
        id,
        data,
    };
});

export const normalizeFilteringStatus = (filteringStatus) => {
    const { enabled, filters, user_rules: userRules } = filteringStatus;
    const newFilters = filters ? filters.map((filter) => {
        const {
            id, url, enabled, lastUpdated: lastUpdated = Date.now(), name = 'Default name', rulesCount: rulesCount = 0,
        } = filter;

        return {
            id, url, enabled, lastUpdated: formatTime(lastUpdated), name, rulesCount,
        };
    }) : [];
    const newUserRules = Array.isArray(userRules) ? userRules.join('\n') : '';
    return { enabled, userRules: newUserRules, filters: newFilters };
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

    if (port) {
        if (ip.includes(':') && !isStandardWebPort) {
            address = `http://[${ip}]:${port}`;
        } else if (!isStandardWebPort) {
            address = `http://${ip}:${port}`;
        }
    }

    return address;
};

export const redirectToCurrentProtocol = (values) => {
    const { protocol, hostname, hash } = window.location;
    const { enabled, port_https } = values;

    if (protocol !== 'https:' && enabled && port_https) {
        const port = port_https !== 443 ? `:${port_https}` : '';
        window.location.replace(`https://${hostname}${port}/${hash}`);
    } else if (protocol === 'https:' && (!enabled || !port_https)) {
        window.location.replace(`http://${hostname}/${hash}`);
    }
};
