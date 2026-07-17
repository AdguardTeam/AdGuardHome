import type { BlockedService } from './blockedService';
import type { ServiceGroup } from './serviceGroup';

export interface BlockedServicesAll {
    blocked_services: BlockedService[];
    groups: ServiceGroup[];
}
