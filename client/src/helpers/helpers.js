import dateParse from 'date-fns/parse';
import dateFormat from 'date-fns/format';
import subHours from 'date-fns/sub_hours';
import addHours from 'date-fns/add_hours';
import round from 'lodash/round';

const formatTime = (time) => {
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
        rule,
    } = log;
    const { host: domain, type } = question;
    const responsesArray = response ? response.map((response) => {
        const { value, type, ttl } = response;
        return `${type}: ${value} (ttl=${ttl})`;
    }) : [];
    return {
        time: formatTime(time),
        domain,
        type,
        response: responsesArray,
        reason,
        client,
        rule,
    };
});

export const normalizeHistory = history => Object.keys(history).map((key) => {
    const id = key.replace(/_/g, ' ').replace(/^\w/, c => c.toUpperCase());

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
            url, enabled, last_updated: lastUpdated = Date.now(), name = 'Default name', rules_count: rulesCount = 0,
        } = filter;

        return {
            url, enabled, lastUpdated: formatTime(lastUpdated), name, rulesCount,
        };
    }) : [];
    const newUserRules = Array.isArray(userRules) ? userRules.join('\n') : '';
    return { enabled, userRules: newUserRules, filters: newFilters };
};
