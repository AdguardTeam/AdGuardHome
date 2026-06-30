import type { AdGuardApiClient } from './adguard-api.ts';

/** Hostname AGH (inside its container) uses to reach host-side mock upstreams. */
export const UPSTREAM_HOST = 'host.docker.internal';

function toHeaderRecord(headers: HeadersInit | undefined): Record<string, string> {
  if (!headers) return {};
  if (headers instanceof Headers) return Object.fromEntries(headers.entries());
  if (Array.isArray(headers)) return Object.fromEntries(headers);
  return { ...(headers as Record<string, string>) };
}

// Node's global fetch (undici) keeps connections alive and pools them. After an
// idle gap (e.g. while DNS is polled via container exec) AGH can close a pooled
// keep-alive connection; the next reused socket then throws "other side closed".
// That's a client transport artifact, not a product fault — retry only this
// connection-reset family. Request timeouts/aborts are NOT retried (they may be
// a genuinely slow or stuck endpoint), but each request is still capped so it
// fails fast instead of hanging until the test timeout.
const TRANSIENT = /other side closed|ECONNRESET|UND_ERR_SOCKET|socket hang up/i;
const REQUEST_TIMEOUT_MS = 25_000;
const MAX_ATTEMPTS = 3;

function isTransient(err: unknown): boolean {
  const e = err as { message?: string; name?: string; cause?: { code?: string; message?: string } };
  const parts = [e?.message, e?.name, e?.cause?.code, e?.cause?.message].filter(Boolean).join(' ');
  return TRANSIENT.test(parts);
}

/** A `fetch` bound with the authenticated client's headers, resilient to
 *  transient keep-alive resets and stuck connections. */
export function authed(api: AdGuardApiClient) {
  return async (url: string, init?: RequestInit): Promise<Response> => {
    const headers = { ...toHeaderRecord(api.authHeaders), ...toHeaderRecord(init?.headers) };
    let lastErr: unknown;
    for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt += 1) {
      try {
        const signal = init?.signal ?? AbortSignal.timeout(REQUEST_TIMEOUT_MS);
        return await fetch(url, { ...init, headers, signal });
      } catch (err) {
        lastErr = err;
        if (!isTransient(err) || attempt === MAX_ATTEMPTS) throw err;
        await new Promise((r) => setTimeout(r, 250 * attempt));
      }
    }
    throw lastErr;
  };
}

/** `{ baseUrl, fetchImpl }` context expected by the shared adguard helpers. */
export function ctxOf(agh: { baseUrl: string }, api: AdGuardApiClient) {
  return { baseUrl: agh.baseUrl, fetchImpl: authed(api) };
}
