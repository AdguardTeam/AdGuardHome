import type { RewriteEntry } from './rewriteEntry';

/**
 * Rewrite rule update object
 */
export interface RewriteUpdate {
    target?: RewriteEntry;
    update?: RewriteEntry;
}
