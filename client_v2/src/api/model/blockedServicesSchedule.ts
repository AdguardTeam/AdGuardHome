import type { Schedule } from './schedule';

export interface BlockedServicesSchedule {
    schedule?: Schedule;
    /** The names of the blocked services. */
    ids?: string[];
}
