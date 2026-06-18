import { listClients } from '../shared/adguard/clients.ts';

export async function fetchClients(options: {
  baseUrl: string;
  headers?: Record<string, string>;
}) {
  return listClients({
    baseUrl: options.baseUrl,
    headers: options.headers,
  });
}
