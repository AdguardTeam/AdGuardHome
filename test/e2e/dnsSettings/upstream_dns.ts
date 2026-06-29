import assert from 'node:assert/strict';

export interface DnsConfig {
  upstream_dns: string[];
  upstream_mode?: string;
  fallback_dns?: string[];
  bootstrap_dns?: string[];
  resolve_clients?: boolean;
  local_ptr_upstreams?: string[];
  use_private_ptr_resolvers?: boolean;
  upstream_timeout?: number;
}

export type FetchFn = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export interface DnsInfo {
  upstream_dns: string[];
  upstream_mode: string;
}

export async function getUpstreamDNS(
  fetchImpl: FetchFn,
  baseUrl: string,
): Promise<DnsInfo> {
  const response = await fetchImpl(`${baseUrl}/control/dns_info`, {
    method: 'GET',
    headers: { 'Accept': 'application/json' },
  });

  if (!response.ok) {
      throw new Error(`Failed to get DNS info: ${response.status}`);
  }
  return (await response.json()) as DnsInfo;
}

export async function setUpstreamDNS(
  fetchImpl: FetchFn,
  baseUrl: string,
  config: Partial<DnsConfig>,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/dns_config`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });

  if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to set upstream DNS: ${response.status} ${text}`);
  }
}

export interface QueryLogEntry {
  question: {
    name: string;
  };
  upstream: string;
  time?: string;
}

export interface QueryLogResponse {
  data: QueryLogEntry[];
}

export interface CheckQueryLogOptions {
  timeoutMs?: number;
  minRecordTimeMs?: number;
}

function normalizeDomain(value: string): string {
  return value.replace(/\.$/, '').toLowerCase();
}

function getEntryTimeMs(entry: QueryLogEntry): number | undefined {
  if (!entry.time) {
    return undefined;
  }

  const timeMs = Date.parse(entry.time);
  return Number.isNaN(timeMs) ? undefined : timeMs;
}

function findLatestMatchingQueryLogEntry(
  entries: QueryLogEntry[],
  domain: string,
  minRecordTimeMs?: number,
): QueryLogEntry | undefined {
  const normalizedDomain = normalizeDomain(domain);
  const matchingEntries = entries.filter(
    (entry) => normalizeDomain(entry.question.name) === normalizedDomain,
  );

  const timedMatchingEntries = matchingEntries
    .map((entry) => ({
      entry,
      timeMs: getEntryTimeMs(entry),
    }))
    .filter((candidate): candidate is { entry: QueryLogEntry; timeMs: number } => candidate.timeMs !== undefined);

  const candidateEntries = (minRecordTimeMs === undefined ? timedMatchingEntries : timedMatchingEntries.filter(
    (candidate) => candidate.timeMs >= minRecordTimeMs,
  )).sort((left, right) => right.timeMs - left.timeMs);

  if (candidateEntries.length > 0) {
    return candidateEntries[0].entry;
  }

  if (minRecordTimeMs !== undefined) {
    return undefined;
  }

  return matchingEntries[0];
}

export async function checkQueryLog(
  fetchImpl: FetchFn,
  baseUrl: string,
  domain: string,
  expectedUpstream: string,
  options: CheckQueryLogOptions = {},
): Promise<void> {
  const timeoutMs = options.timeoutMs ?? 10000;
  const start = Date.now();
  let lastError: Error | undefined;

  while (Date.now() - start < timeoutMs) {
    try {
        const response = await fetchImpl(`${baseUrl}/control/querylog?limit=20`, {
            method: 'GET',
            headers: { 'Accept': 'application/json' },
        });

        if (response.ok) {
            const json = (await response.json()) as QueryLogResponse;
            const entry = findLatestMatchingQueryLogEntry(
              json.data,
              domain,
              options.minRecordTimeMs,
            );

            if (entry) {
                if (entry.upstream === expectedUpstream) {
                    return; // Success
                }
                // If upstream doesn't match, maybe we picked up an old query?
                // But usually we just made one query.
                // We'll retry just in case a new log entry appears.
            }
        }
    } catch (e) {
        lastError = e as Error;
    }
    await new Promise(resolve => setTimeout(resolve, 500));
  }

  // If we reach here, we failed. Fetch one last time to throw assertion error.
  const response = await fetchImpl(`${baseUrl}/control/querylog?limit=20`, {
      method: 'GET',
      headers: { 'Accept': 'application/json' },
  });

  assert.equal(response.ok, true, `Failed to fetch query log: ${response.status}`);
  const json = (await response.json()) as QueryLogResponse;
  const entry = findLatestMatchingQueryLogEntry(
    json.data,
    domain,
    options.minRecordTimeMs,
  );

  assert.ok(
    entry,
    [
      `No query log entry found for domain: ${domain} after ${timeoutMs}ms`,
      options.minRecordTimeMs === undefined
        ? undefined
        : `Expected an entry newer than ${new Date(options.minRecordTimeMs).toISOString()}`,
      `Latest entries: ${JSON.stringify(json.data, null, 2)}`,
      lastError ? `Last fetch error: ${lastError.message}` : undefined,
    ].filter(Boolean).join('\n'),
  );
  assert.equal(entry.upstream, expectedUpstream, `Expected upstream ${expectedUpstream} for ${domain}, but got ${entry.upstream}`);
}
