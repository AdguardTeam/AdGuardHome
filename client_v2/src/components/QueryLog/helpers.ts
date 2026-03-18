import { format as dateFormat, isValid } from 'date-fns';

import intl from 'panel/common/intl';
import {
    FILTERED_STATUS,
    FILTERED_STATUS_TO_COLOR_MAP,
    QUERY_STATUS_COLORS,
    SCHEME_TO_PROTOCOL_MAP,
    DEFAULT_TIME_FORMAT,
    LONG_TIME_FORMAT,
} from 'panel/helpers/constants';
import { formatElapsedMs, getFilterNames, getServiceName, type Filter, type Rule } from 'panel/helpers/helpers';
import { ResponseEntry } from './types';

const parseLogDate = (time: string): Date | null => {
    const parsedTime = new Date(time);

    return isValid(parsedTime) ? parsedTime : null;
};

export const getStatusLabel = (
    reason: string,
    originalResponse: ResponseEntry[],
    isBlockedByResponse: boolean,
): string => {
    if (isBlockedByResponse) {
        return intl.getMessage('blocked_by_cname_or_ip');
    }

    switch (reason) {
        case FILTERED_STATUS.NOT_FILTERED_WHITE_LIST:
            return intl.getMessage('allowed');
        case FILTERED_STATUS.NOT_FILTERED_NOT_FOUND:
            return intl.getMessage('show_processed_responses');
        case FILTERED_STATUS.FILTERED_BLOCKED_SERVICE:
            return intl.getMessage('blocked_service');
        case FILTERED_STATUS.FILTERED_SAFE_SEARCH:
            return intl.getMessage('safe_search');
        case FILTERED_STATUS.FILTERED_BLACK_LIST:
            return intl.getMessage('show_blocked_responses');
        case FILTERED_STATUS.REWRITE:
        case FILTERED_STATUS.REWRITE_HOSTS:
        case FILTERED_STATUS.REWRITE_RULE:
            return intl.getMessage('rewritten');
        case FILTERED_STATUS.FILTERED_SAFE_BROWSING:
            return intl.getMessage('blocked_threats');
        case FILTERED_STATUS.FILTERED_PARENTAL:
            return intl.getMessage('blocked_adult_websites');
        case FILTERED_STATUS.NOT_FILTERED_ERROR:
            return intl.getMessage('error');
        case FILTERED_STATUS.FILTERED_INVALID:
            return intl.getMessage('invalid');
        default:
            if (originalResponse && originalResponse.length > 0) {
                return intl.getMessage('rewritten');
            }

            return intl.getMessage('show_processed_responses');
    }
};

export const getStatusColor = (
    reason: string,
): (typeof QUERY_STATUS_COLORS)[keyof typeof QUERY_STATUS_COLORS] | '' =>
    FILTERED_STATUS_TO_COLOR_MAP[reason as keyof typeof FILTERED_STATUS_TO_COLOR_MAP] || '';

export const getStatusClassName = (styles: Record<string, string>, reason: string): string => {
    const statusColor = getStatusColor(reason);

    if (!statusColor) {
        return '';
    }

    const statusClassName = `status${statusColor.charAt(0).toUpperCase()}${statusColor.slice(1)}`;

    return styles[statusClassName] || '';
};

export const isBlockedReason = (reason: string): boolean =>
    reason.startsWith('Filtered') && reason !== 'FilteredSafeSearch';

export const getProtocolName = (clientProto: string): string => {
    const key = SCHEME_TO_PROTOCOL_MAP[clientProto as keyof typeof SCHEME_TO_PROTOCOL_MAP];
    if (key) {
        return intl.getMessage(key);
    }
    return intl.getMessage('plain_dns');
};

export const formatLogTime = (time: string): string => {
    const parsedTime = parseLogDate(time);

    return parsedTime ? dateFormat(parsedTime, DEFAULT_TIME_FORMAT) : time;
};

export const formatLogDate = (time: string): string => {
    const parsedTime = parseLogDate(time);

    return parsedTime
        ? parsedTime.toLocaleDateString(navigator.language, {
              day: 'numeric',
              month: 'short',
              year: 'numeric',
          })
        : time;
};

export const formatLogTimeDetailed = (time: string): string => {
    const parsedTime = parseLogDate(time);

    return parsedTime ? dateFormat(parsedTime, LONG_TIME_FORMAT) : time;
};

type ResponseDetailsParams = {
    elapsedMs: string;
    filters: Filter[];
    reason: string;
    rules: Rule[];
    serviceName?: string;
    services?: { id: string; name: string }[];
    whitelistFilters: Filter[];
};

export const getResponseDetails = ({
    elapsedMs,
    filters,
    reason,
    rules,
    serviceName,
    services,
    whitelistFilters,
}: ResponseDetailsParams): string => {
    const formattedElapsedMs = formatElapsedMs(elapsedMs, (key) => intl.getMessage(key));

    switch (reason) {
        case FILTERED_STATUS.FILTERED_BLOCKED_SERVICE:
            return (
                (services && serviceName && getServiceName(services, serviceName))
                || serviceName
                || formattedElapsedMs
            );
        case FILTERED_STATUS.FILTERED_BLACK_LIST:
        case FILTERED_STATUS.NOT_FILTERED_WHITE_LIST: {
            const filterNames = getFilterNames(rules, filters, whitelistFilters).filter(Boolean).join(', ');

            return filterNames || formattedElapsedMs;
        }
        default:
            return formattedElapsedMs;
    }
};

export const getBlockClientInfo = (
    ip: string,
    disallowed: boolean,
    disallowedRule: string,
    allowedClients: string[],
) => {
    const isInAllowlist = allowedClients.length > 0;
    const isLastAllowlistEntry = isInAllowlist && allowedClients.length === 1;

    return {
        isInAllowlist,
        isLastAllowlistEntry,
        disallowed,
        disallowedRule: disallowedRule || ip,
    };
};
