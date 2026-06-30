import { adguardGet, adguardPost, type AdGuardRequestContext } from './api.ts';

export interface PersistentClient {
  name: string;
  ids: string[];
  tags?: string[];
  use_global_settings?: boolean;
  use_global_blocked_services?: boolean;
  filtering_enabled?: boolean;
  parental_enabled?: boolean;
  safebrowsing_enabled?: boolean;
  ignore_querylog?: boolean;
  ignore_statistics?: boolean;
  upstreams?: string[];
}

export interface AutoClient {
  name: string;
  ids: string[];
  source?: string;
}

export interface ClientsResponse {
  clients: PersistentClient[];
  auto_clients: AutoClient[];
  supported_tags?: string[];
}

export async function listClients(context: AdGuardRequestContext): Promise<ClientsResponse> {
  const response = await adguardGet<ClientsResponse>({
    ...context,
    path: '/control/clients',
  });
  // AGH v0.107.74 returns `clients: null` when the list is empty instead of `clients: []`
  return { ...response, clients: response.clients ?? [] };
}

export async function addClient(context: AdGuardRequestContext, client: PersistentClient): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/clients/add',
    body: client,
  });
}

export async function updateClient(
  context: AdGuardRequestContext,
  targetName: string,
  update: PersistentClient,
): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/clients/update',
    body: {
      name: targetName,
      data: update,
    },
  });
}

export async function deleteClient(context: AdGuardRequestContext, name: string): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/clients/delete',
    body: { name },
  });
}
