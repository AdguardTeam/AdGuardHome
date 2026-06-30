import { adguardGet, adguardPost, adguardPut, type AdGuardRequestContext } from './api.ts';

export interface QueryLogAnswer {
  type?: string;
  value?: string;
}

export interface QueryLogEntry {
  client?: string;
  client_id?: string;
  status?: string;
  reason?: string;
  rule?: string;
  rules?: Array<{ text?: string; filter_list_id?: number }>;
  time?: string;
  upstream?: string;
  elapsedMs?: string;
  question?: {
    name?: string;
    host?: string;
    type?: string;
  };
  answer?: QueryLogAnswer[];
  original_answer?: QueryLogAnswer[];
}

export interface QueryLogResponse {
  data: QueryLogEntry[];
  oldest?: string;
}

export interface QueryLogConfig {
  enabled: boolean;
  interval: number;
  ignored: string[];
  ignored_enabled: boolean;
  anonymize_client_ip?: boolean;
}

export interface QueryLogQuery {
  search?: string;
  response_status?: string;
  older_than?: string;
  limit?: number;
}

function buildQueryString(query: QueryLogQuery): string {
  const params = new URLSearchParams();
  params.set('response_status', query.response_status ?? 'all');
  params.set('search', query.search ?? '');
  params.set('older_than', query.older_than ?? '');
  params.set('limit', String(query.limit ?? 20));
  return params.toString();
}

export async function getQueryLog(
  context: AdGuardRequestContext,
  query: QueryLogQuery = {},
): Promise<QueryLogResponse> {
  return adguardGet<QueryLogResponse>({
    ...context,
    path: `/control/querylog?${buildQueryString(query)}`,
  });
}

export async function getQueryLogConfig(context: AdGuardRequestContext): Promise<QueryLogConfig> {
  return adguardGet<QueryLogConfig>({
    ...context,
    path: '/control/querylog/config',
  });
}

export async function updateQueryLogConfig(
  context: AdGuardRequestContext,
  config: QueryLogConfig,
): Promise<void> {
  await adguardPut({
    ...context,
    path: '/control/querylog/config/update',
    body: config,
  });
}

export async function clearQueryLog(context: AdGuardRequestContext): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/querylog_clear',
  });
}
