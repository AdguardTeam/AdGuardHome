import type { QueryLogItem } from './queryLogItem';

/**
 * Query log
 */
export interface QueryLog {
    oldest?: string;
    data?: QueryLogItem[];
}
