import type { Client } from './client';

/**
 * Client update request
 */
export interface ClientUpdate {
    name?: string;
    data?: Client;
}
