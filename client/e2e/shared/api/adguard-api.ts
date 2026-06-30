import { normalizeBaseUrl } from '../config/env.ts';
import { jsonRequest, type JsonFetchLike } from './json-client.ts';

export interface AdGuardCredentials {
  username: string;
  password: string;
}

export interface AdGuardStatus {
  version: string;
  protection_enabled: boolean;
  protection_disabled_duration: number;
  running: boolean;
  dns_addresses: string[];
  dns_port: number;
  http_port: number;
}

export interface AdGuardStatsConfig {
  enabled: boolean;
  interval: number;
  ignored: string[];
  ignored_enabled: boolean;
}

export interface AdGuardStats {
  top_queried_domains: Array<Record<string, number>>;
  top_clients: Array<Record<string, number>>;
  top_blocked_domains: Array<Record<string, number>>;
  num_dns_queries: number;
  num_blocked_filtering: number;
  num_replaced_safesearch: number;
  num_replaced_parental: number;
}

export interface AdGuardQueryLogConfig {
  ignored: string[];
  interval: number;
  enabled: boolean;
  ignored_enabled: boolean;
  anonymize_client_ip: boolean;
}

export interface AdGuardAccessList {
  allowed_clients?: string[];
  disallowed_clients?: string[];
  blocked_hosts?: string[];
}

export interface AdGuardQueryLogAnswer {
  type?: string;
  value?: string;
  ttl?: number;
}

export interface AdGuardQueryLogRecord {
  answer?: AdGuardQueryLogAnswer[];
  answer_dnssec?: boolean;
  cached?: boolean;
  client?: string;
  client_proto?: string;
  elapsedMs?: string;
  question?: {
    class?: string;
    name?: string;
    type?: string;
  };
  reason?: string;
  rules?: Array<{
    filter_list_id?: number;
    text?: string;
  }>;
  status?: string;
  time?: string;
  upstream?: string;
}

export interface AdGuardQueryLogResponse {
  data: AdGuardQueryLogRecord[];
  oldest?: string;
}

export interface AdGuardClient {
  name: string;
  ids: string[];
  tags?: string[];
  use_global_settings?: boolean;
  use_global_blocked_services?: boolean;
  filtering_enabled?: boolean;
  parental_enabled?: boolean;
  safebrowsing_enabled?: boolean;
  safesearch_enabled?: boolean;
  ignore_querylog?: boolean;
  ignore_statistics?: boolean;
  upstreams?: string[];
}

export interface AdGuardClientsResponse {
  clients?: AdGuardClient[];
  auto_clients?: Array<{
    ip: string;
    name: string;
    source: string;
    whois_info?: Record<string, unknown>;
  }>;
  supported_tags?: string[];
}

export interface AdGuardProfile {
  language: string;
  theme: string;
}

export interface AdGuardApiClient {
  baseUrl: string;
  authHeaders: HeadersInit;
  fetch: JsonFetchLike;
}

export interface QueryLogSearchParams {
  search?: string;
  response_status?: string;
  older_than?: string;
  limit?: number;
}

function buildCookieHeader(setCookie: string | null): string {
  if (!setCookie) {
    throw new Error('Login did not return a session cookie');
  }

  return setCookie.split(';', 1)[0];
}

function authFetch(cookie: string, fetchImpl: JsonFetchLike): JsonFetchLike {
  return async (input, init) => {
    const headers = new Headers(init?.headers);
    headers.set('Cookie', cookie);

    return fetchImpl(input, {
      ...init,
      headers,
    });
  };
}

function buildQueryLogPath(params: QueryLogSearchParams = {}): string {
  const query = new URLSearchParams({
    response_status: params.response_status ?? 'all',
    search: params.search ?? '',
    older_than: params.older_than ?? '',
    limit: String(params.limit ?? 20),
  });

  return `/control/querylog?${query.toString()}`;
}

export async function loginToAdGuardApi(
  baseUrl: string,
  credentials: AdGuardCredentials,
  fetchImpl: JsonFetchLike = fetch,
): Promise<AdGuardApiClient> {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);
  const response = await fetchImpl(`${normalizedBaseUrl}/control/login`, {
    method: 'POST',
    headers: {
      Accept: 'application/json, text/plain, */*',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      name: credentials.username,
      password: credentials.password,
    }),
  });

  if (!response.ok) {
    const details = await response.text().catch(() => '');
    throw new Error(`Failed to login to AdGuard Home: ${response.status}${details ? ` ${details}` : ''}`);
  }

  const cookie = buildCookieHeader(response.headers.get('set-cookie'));
  return {
    baseUrl: normalizedBaseUrl,
    authHeaders: {
      Cookie: cookie,
    },
    fetch: authFetch(cookie, fetchImpl),
  };
}

export async function getStatus(client: AdGuardApiClient): Promise<AdGuardStatus> {
  return jsonRequest<AdGuardStatus>({
    baseUrl: client.baseUrl,
    path: '/control/status',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function setProtection(
  client: AdGuardApiClient,
  options: { enabled: boolean; durationMs?: number | null },
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/protection',
    method: 'POST',
    body: {
      enabled: options.enabled,
      duration: options.durationMs ?? null,
    },
    fetchImpl: client.fetch,
  });
}

export async function getStats(client: AdGuardApiClient): Promise<AdGuardStats> {
  return jsonRequest<AdGuardStats>({
    baseUrl: client.baseUrl,
    path: '/control/stats',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function getStatsConfig(client: AdGuardApiClient): Promise<AdGuardStatsConfig> {
  return jsonRequest<AdGuardStatsConfig>({
    baseUrl: client.baseUrl,
    path: '/control/stats/config',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function updateStatsConfig(
  client: AdGuardApiClient,
  config: AdGuardStatsConfig,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/stats/config/update',
    method: 'PUT',
    body: config,
    fetchImpl: client.fetch,
  });
}

export async function resetStats(client: AdGuardApiClient): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/stats_reset',
    method: 'POST',
    fetchImpl: client.fetch,
  });
}

export async function getQueryLogConfig(client: AdGuardApiClient): Promise<AdGuardQueryLogConfig> {
  return jsonRequest<AdGuardQueryLogConfig>({
    baseUrl: client.baseUrl,
    path: '/control/querylog/config',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function updateQueryLogConfig(
  client: AdGuardApiClient,
  config: AdGuardQueryLogConfig,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/querylog/config/update',
    method: 'PUT',
    body: config,
    fetchImpl: client.fetch,
  });
}

export async function clearQueryLog(client: AdGuardApiClient): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/querylog_clear',
    method: 'POST',
    fetchImpl: client.fetch,
  });
}

export async function getAccessList(client: AdGuardApiClient): Promise<AdGuardAccessList> {
  return jsonRequest<AdGuardAccessList>({
    baseUrl: client.baseUrl,
    path: '/control/access/list',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function setAccessList(
  client: AdGuardApiClient,
  config: AdGuardAccessList,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/access/set',
    method: 'POST',
    body: config,
    fetchImpl: client.fetch,
  });
}

export async function getQueryLog(
  client: AdGuardApiClient,
  params: QueryLogSearchParams = {},
): Promise<AdGuardQueryLogResponse> {
  return jsonRequest<AdGuardQueryLogResponse>({
    baseUrl: client.baseUrl,
    path: buildQueryLogPath(params),
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function getProfile(client: AdGuardApiClient): Promise<AdGuardProfile> {
  return jsonRequest<AdGuardProfile>({
    baseUrl: client.baseUrl,
    path: '/control/profile',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function updateProfile(
  client: AdGuardApiClient,
  profile: Partial<AdGuardProfile>,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/profile/update',
    method: 'PUT',
    body: profile,
    fetchImpl: client.fetch,
  });
}

export async function getClients(client: AdGuardApiClient): Promise<AdGuardClientsResponse> {
  return jsonRequest<AdGuardClientsResponse>({
    baseUrl: client.baseUrl,
    path: '/control/clients',
    method: 'GET',
    fetchImpl: client.fetch,
  });
}

export async function searchClients(
  client: AdGuardApiClient,
  ids: string[],
): Promise<Array<{ name?: string; id?: string }>> {
  return jsonRequest<Array<{ name?: string; id?: string }>>({
    baseUrl: client.baseUrl,
    path: '/control/clients/search',
    method: 'POST',
    body: {
      clients: ids.map((id) => ({ id })),
    },
    fetchImpl: client.fetch,
  });
}

export async function addClient(
  client: AdGuardApiClient,
  payload: AdGuardClient,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/clients/add',
    method: 'POST',
    body: payload,
    fetchImpl: client.fetch,
  });
}

export async function updateClient(
  client: AdGuardApiClient,
  currentName: string,
  payload: AdGuardClient,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/clients/update',
    method: 'POST',
    body: {
      name: currentName,
      data: payload,
    },
    fetchImpl: client.fetch,
  });
}

export async function deleteClient(
  client: AdGuardApiClient,
  name: string,
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/clients/delete',
    method: 'POST',
    body: { name },
    fetchImpl: client.fetch,
  });
}

export async function setCustomRules(
  client: AdGuardApiClient,
  rules: string[],
): Promise<void> {
  await jsonRequest({
    baseUrl: client.baseUrl,
    path: '/control/filtering/set_rules',
    method: 'POST',
    body: { rules },
    fetchImpl: client.fetch,
  });
}
