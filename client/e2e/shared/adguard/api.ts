import { jsonRequest, type JsonFetchLike } from '../api/json-client.ts';

export interface AdGuardRequestContext {
  baseUrl: string;
  fetchImpl?: JsonFetchLike;
  fetch?: JsonFetchLike;
  headers?: Record<string, string>;
  authHeaders?: Record<string, string>;
}

export interface AdGuardRequestOptions extends AdGuardRequestContext {
  path: string;
  body?: unknown;
}

export async function adguardGet<T>(options: AdGuardRequestOptions): Promise<T> {
  return jsonRequest<T>({
    baseUrl: options.baseUrl,
    path: options.path,
    method: 'GET',
    headers: options.headers ?? options.authHeaders,
    fetchImpl: options.fetchImpl ?? options.fetch,
  });
}

export async function adguardPost<T = void>(options: AdGuardRequestOptions): Promise<T> {
  return jsonRequest<T>({
    baseUrl: options.baseUrl,
    path: options.path,
    method: 'POST',
    body: options.body,
    headers: options.headers ?? options.authHeaders,
    fetchImpl: options.fetchImpl ?? options.fetch,
  });
}

export async function adguardPut<T = void>(options: AdGuardRequestOptions): Promise<T> {
  return jsonRequest<T>({
    baseUrl: options.baseUrl,
    path: options.path,
    method: 'PUT',
    body: options.body,
    headers: options.headers ?? options.authHeaders,
    fetchImpl: options.fetchImpl ?? options.fetch,
  });
}
