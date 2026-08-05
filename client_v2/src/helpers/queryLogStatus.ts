import { FILTERED_STATUS } from './constants';

export type QueryStatusKey = 'all' | 'allowed' | 'processed' | 'blocked' | 'rewritten' | 'error';

type QueryStatusEntry = {
    reason?: string;
    originalResponse?: { type?: string; value?: string }[];
    status?: string;
};

const isErrorResponse = (responseStatus?: string): boolean =>
    !!responseStatus && responseStatus !== 'NOERROR';

export const getQueryStatusKey = (
    reason?: string,
    originalResponse: { value?: string; type?: string; ttl?: number }[] = [],
    responseStatus?: string,
): Exclude<QueryStatusKey, 'all'> => {
    switch (reason) {
        case FILTERED_STATUS.NOT_FILTERED_WHITE_LIST:
            return isErrorResponse(responseStatus) ? 'error' : 'allowed';
        case FILTERED_STATUS.REWRITE:
        case FILTERED_STATUS.REWRITE_HOSTS:
        case FILTERED_STATUS.REWRITE_RULE:
        case FILTERED_STATUS.FILTERED_SAFE_SEARCH:
            return 'rewritten';
        case FILTERED_STATUS.NOT_FILTERED_NOT_FOUND:
            return isErrorResponse(responseStatus) ? 'error' : 'processed';
        case FILTERED_STATUS.NOT_FILTERED_ERROR:
        case FILTERED_STATUS.FILTERED_INVALID:
            return 'error';
        default:
            if (originalResponse.length > 0) {
                return 'rewritten';
            }

            if (reason && reason.startsWith('Filtered')) {
                return 'blocked';
            }

            return isErrorResponse(responseStatus) ? 'error' : 'processed';
    }
};

export const filterLogsByStatus = <T extends QueryStatusEntry>(
    logs: T[],
    status: QueryStatusKey | string,
): T[] => {
    if (status === 'all') {
        return logs;
    }

    return logs.filter(
        (log) =>
            getQueryStatusKey(log.reason ?? '', log.originalResponse ?? [], log.status) === status,
    );
};
