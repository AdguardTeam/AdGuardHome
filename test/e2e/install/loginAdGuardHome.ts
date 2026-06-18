import { type FetchLike, type RetryPolicy, type SimpleResponse } from './installhome.ts';

export interface LoginOptions {
  baseUrl: string;
  username: string;
  password: string;
  retry?: Partial<RetryPolicy>;
  timeoutMs?: number;
}

const DEFAULT_RETRY: RetryPolicy = {
  attempts: 5,
  baseDelayMs: 200,
  maxDelayMs: 2_000,
};

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '');
}

function withDefaultRetry(retry?: Partial<RetryPolicy>): RetryPolicy {
  return {
    attempts: retry?.attempts ?? DEFAULT_RETRY.attempts,
    baseDelayMs: retry?.baseDelayMs ?? DEFAULT_RETRY.baseDelayMs,
    maxDelayMs: retry?.maxDelayMs ?? DEFAULT_RETRY.maxDelayMs,
  };
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function backoffMs(attempt: number, retry: RetryPolicy): number {
  const exp = retry.baseDelayMs * 2 ** (attempt - 1);
  return Math.min(exp, retry.maxDelayMs);
}

export async function loginAdGuardHome(
  options: LoginOptions,
  fetchImpl: FetchLike = fetch as unknown as FetchLike,
): Promise<SimpleResponse> {
  if (options.username.trim().length === 0 || options.password.trim().length === 0) {
    throw new Error('Login credentials must not be empty');
  }

  const retry = withDefaultRetry(options.retry);
  const timeoutMs = options.timeoutMs ?? 8_000;
  const endpoint = `${normalizeBaseUrl(options.baseUrl)}/control/login`;

  let lastError: Error | undefined;

  for (let attempt = 1; attempt <= retry.attempts; attempt += 1) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);

    try {
      const response = await fetchImpl(endpoint, {
        method: 'POST',
        headers: {
          Accept: 'application/json, text/plain, */*',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: options.username,
          password: options.password,
        }),
        signal: controller.signal,
      });

      clearTimeout(timer);

      if (response.ok) {
        return response;
      }

      if (response.status === 403) {
        throw new Error('Login failed: wrong username or password (HTTP 403)');
      }

      if (response.status >= 400 && response.status < 500) {
        const details = await response.text();
        throw new Error(`Login rejected with ${response.status}: ${details}`);
      }

      lastError = new Error(`Login endpoint temporary failure: HTTP ${response.status}`);
    } catch (error) {
      clearTimeout(timer);
      lastError = error instanceof Error ? error : new Error(String(error));
    }

    if (attempt < retry.attempts) {
      await sleep(backoffMs(attempt, retry));
    }
  }

  throw new Error(`Failed to login into AdGuardHome: ${lastError?.message ?? 'unknown error'}`);
}
