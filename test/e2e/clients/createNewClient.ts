import {
  addClient,
  type PersistentClient,
} from '../shared/adguard/clients.ts';

export async function createNewClient(options: {
  baseUrl: string;
  client: PersistentClient;
  headers?: Record<string, string>;
}) {
  await addClient({
    baseUrl: options.baseUrl,
    headers: options.headers,
  }, options.client);
}
