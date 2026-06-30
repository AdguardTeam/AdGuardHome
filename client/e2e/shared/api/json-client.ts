import { normalizeBaseUrl } from '../config/env.ts';

export type JsonFetchLike = typeof fetch;

export interface JsonRequestOptions {
  baseUrl: string;
  path: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: unknown;
  headers?: Record<string, string>;
  fetchImpl?: JsonFetchLike;
}

function buildUrl(baseUrl: string, path: string): string {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizedBaseUrl}${normalizedPath}`;
}

export async function jsonRequest<T = void>(options: JsonRequestOptions): Promise<T> {
  const response = await (options.fetchImpl ?? fetch)(buildUrl(options.baseUrl, options.path), {
    method: options.method,
    headers: {
      Accept: 'application/json, text/plain, */*',
      ...(options.body === undefined ? {} : { 'Content-Type': 'application/json' }),
      ...options.headers,
    },
    ...(options.body === undefined ? {} : { body: JSON.stringify(options.body) }),
  });

  if (!response.ok) {
    const details = await response.text().catch(() => '');
    throw new Error(`Request failed: ${response.status}${details ? ` ${details}` : ''}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get('content-type') ?? '';
  if (!contentType.includes('application/json')) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
