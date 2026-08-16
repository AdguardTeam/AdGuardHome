import type { ClientsArray } from './clientsArray';
import type { ClientsAutoArray } from './clientsAutoArray';

export interface Clients {
    clients?: ClientsArray;
    auto_clients?: ClientsAutoArray;
    supported_tags?: string[];
}
