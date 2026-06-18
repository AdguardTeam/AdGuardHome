export interface AdGuardServiceBinding {
  ip: string;
  port: number;
}

export interface AdGuardInstallRequest {
  web: AdGuardServiceBinding;
  dns: AdGuardServiceBinding;
  username: string;
  password: string;
}

export interface RetryPolicy {
  attempts: number;
  baseDelayMs: number;
  maxDelayMs: number;
}

export interface InstallOptions {
  baseUrl: string;
  request: AdGuardInstallRequest;
  retry?: Partial<RetryPolicy>;
  timeoutMs?: number;
}

export interface SimpleResponse {
  ok: boolean;
  status: number;
  text(): Promise<string>;
}

export type FetchLike = (
  input: string,
  init?: {
    method?: string;
    headers?: Record<string, string>;
    body?: string;
    signal?: AbortSignal;
  },
) => Promise<SimpleResponse>;

const DEFAULT_RETRY: RetryPolicy = {
  attempts: 6,
  baseDelayMs: 250,
  maxDelayMs: 2_500,
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

function validateInstallRequest(request: AdGuardInstallRequest): void {
  const hasCredentials = request.username.trim().length > 0 && request.password.trim().length > 0;
  if (!hasCredentials) {
    throw new Error('AdGuard install request must include non-empty username and password');
  }

  for (const binding of [request.web, request.dns]) {
    if (binding.ip.trim().length === 0) {
      throw new Error('AdGuard install request must include non-empty bind IP for DNS and WEB');
    }
    if (!Number.isInteger(binding.port) || binding.port <= 0 || binding.port > 65535) {
      throw new Error(`Invalid AdGuard bind port: ${binding.port}`);
    }
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function backoffMs(attempt: number, retry: RetryPolicy): number {
  const exp = retry.baseDelayMs * 2 ** (attempt - 1);
  return Math.min(exp, retry.maxDelayMs);
}

export async function installAdGuardHome(
  options: InstallOptions,
  fetchImpl: FetchLike = fetch as unknown as FetchLike,
): Promise<SimpleResponse> {
  const retry = withDefaultRetry(options.retry);
  const timeoutMs = options.timeoutMs ?? 10_000;
  validateInstallRequest(options.request);

  const endpoint = `${normalizeBaseUrl(options.baseUrl)}/control/install/configure`;
  const payload = {
    web: {
      ip: options.request.web.ip,
      port: options.request.web.port,
      status: '',
      can_autofix: false,
    },
    dns: {
      ip: options.request.dns.ip,
      port: options.request.dns.port,
      status: '',
      can_autofix: false,
    },
    username: options.request.username,
    password: options.request.password,
  };

  let lastError: Error | undefined;
  let shouldRetry = true;

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
        body: JSON.stringify(payload),
        signal: controller.signal,
      });

      clearTimeout(timer);

      if (response.ok) {
        return response;
      }

      const details = await response.text();
      throw new Error(`Install request failed with ${response.status}: ${details}`);
    } catch (error) {
      clearTimeout(timer);
      lastError = error instanceof Error ? error : new Error(String(error));
      shouldRetry =
        lastError.name === 'AbortError'
        || /fetch failed|network/i.test(lastError.message)
        || /ECONNREFUSED|ECONNRESET|ENOTFOUND|EHOSTUNREACH|ETIMEDOUT/i.test(lastError.message);
    }

    if (!shouldRetry) {
      break;
    }

    if (attempt < retry.attempts) {
      await sleep(backoffMs(attempt, retry));
    }
  }

  throw new Error(`Failed to configure AdGuardHome install flow: ${lastError?.message ?? 'unknown error'}`);
}
