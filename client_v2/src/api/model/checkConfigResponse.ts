import type { CheckConfigResponseInfo } from './checkConfigResponseInfo';
import type { CheckConfigStaticIpInfo } from './checkConfigStaticIpInfo';

export interface CheckConfigResponse {
    dns: CheckConfigResponseInfo;
    language: CheckConfigResponseInfo;
    static_ip: CheckConfigStaticIpInfo;
    web: CheckConfigResponseInfo;
}
