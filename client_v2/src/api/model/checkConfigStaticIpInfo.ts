import type { CheckConfigStaticIpInfoStatic } from './checkConfigStaticIpInfoStatic';

export interface CheckConfigStaticIpInfo {
    static?: CheckConfigStaticIpInfoStatic;
    /** Current dynamic IP address. Set if static=no */
    ip?: string;
    /** Error text. Set if static=error */
    error?: string;
}
