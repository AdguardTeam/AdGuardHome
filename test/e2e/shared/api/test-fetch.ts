import type { AdGuardApiClient } from './adguard-api.ts';

/** Hostname AGH (inside its container) uses to reach host-side mock upstreams. */
export const UPSTREAM_HOST = 'host.docker.internal';

function toHeaderRecord(headers: HeadersInit | undefined): Record<string, string> {
  if (!headers) return {};
  if (headers instanceof Headers) return Object.fromEntries(headers.entries());
  if (Array.isArray(headers)) return Object.fromEntries(headers);
  return { ...(headers as Record<string, string>) };
}

/** A `fetch` bound with the authenticated client's headers (handles any HeadersInit form). */
export function authed(api: AdGuardApiClient) {
  return (url: string, init?: RequestInit) =>
    fetch(url, { ...init, headers: { ...toHeaderRecord(api.authHeaders), ...toHeaderRecord(init?.headers) } });
}

/** `{ baseUrl, fetchImpl }` context expected by the shared adguard helpers. */
export function ctxOf(agh: { baseUrl: string }, api: AdGuardApiClient) {
  return { baseUrl: agh.baseUrl, fetchImpl: authed(api) };
}
