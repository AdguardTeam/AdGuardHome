import type { CheckConfigRequestInfo } from './checkConfigRequestInfo';
import type { Lang } from './lang';

/**
 * Configuration to be checked
 */
export interface CheckConfigRequest {
    dns?: CheckConfigRequestInfo;
    language?: Lang;
    set_static_ip?: boolean;
    web?: CheckConfigRequestInfo;
}
