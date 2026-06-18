import assert from 'node:assert/strict';

export interface BlockedServiceSchedule {
  time_zone?: string;
  sun?: { start: number; end: number };
  mon?: { start: number; end: number };
  tue?: { start: number; end: number };
  wed?: { start: number; end: number };
  thu?: { start: number; end: number };
  fri?: { start: number; end: number };
  sat?: { start: number; end: number };
}

export interface BlockedServicesConfig {
  ids: string[];
  schedule?: BlockedServiceSchedule;
}

export interface ClientConfig {
  name: string;
  ids: string[];
  blocked_services?: string[];
  use_global_blocked_services?: boolean;
  blocked_services_schedule?: BlockedServiceSchedule;
  tags?: string[];
  filtering_enabled?: boolean;
  parental_enabled?: boolean;
  safebrowsing_enabled?: boolean;
  safesearch_enabled?: boolean; // Used for simple bool in some contexts?
  // Extended fields for createNewClient full payload support:
  safe_search?: {
      enabled: boolean;
      bing?: boolean;
      duckduckgo?: boolean;
      ecosia?: boolean;
      google?: boolean;
      pixabay?: boolean;
      yandex?: boolean;
      youtube?: boolean;
  };
  use_global_settings?: boolean;
  ignore_querylog?: boolean;
  ignore_statistics?: boolean;
  upstreams?: string[];
  upstreams_cache_enabled?: boolean;
  upstreams_cache_size?: number;
}

export type FetchLike = typeof fetch;
export type DnsResolver = (domain: string) => Promise<string[]>;

export interface BlockedServicesTestCase {
  name: string;
  // Global config to set
  config?: BlockedServicesConfig;
  // Client to add/test
  client?: ClientConfig;

  // Verification
  domainToResolve: string;
  expectedResolution: string; // "0.0.0.0" or real IP or "IP" (any valid IP)
}

export interface BlockedServicesContext {
  baseUrl: string;
  fetchImpl?: FetchLike;
  resolveDns?: DnsResolver;
}

export async function listBlockedServices(
  baseUrl: string,
  fetchImpl: FetchLike = fetch,
): Promise<string[]> {
  const response = await fetchImpl(`${baseUrl}/control/blocked_services/list`, {
    method: 'GET',
    headers: { 'Accept': 'application/json' },
  });

  assert.equal(response.ok, true, `Failed to list blocked services: ${response.status}`);

  // The API returns simple array of strings (blocked service IDs)
  const data = await response.json();
  if (Array.isArray(data)) {
      // Ensure all elements are strings
      return data.filter((item): item is string => typeof item === 'string');
  }
  // Fallback if API returns object with ids
  if (data && typeof data === 'object' && 'ids' in data && Array.isArray((data as any).ids)) {
      return ((data as any).ids as unknown[]).filter((item): item is string => typeof item === 'string');
  }
  return [];
}

export async function updateBlockedServices(
  baseUrl: string,
  config: BlockedServicesConfig,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/blocked_services/update`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });

  assert.equal(response.ok, true, `Failed to update blocked services: ${response.status}`);
}

export async function listAvailableServices(
    baseUrl: string,
    fetchImpl: FetchLike = fetch,
): Promise<string[]> {
    const response = await fetchImpl(`${baseUrl}/control/blocked_services/services`, {
        method: 'GET',
        headers: { 'Accept': 'application/json' },
    });

    assert.equal(response.ok, true, `Failed to list available services: ${response.status}`);
    const data = await response.json();

    if (Array.isArray(data)) {
        // Return IDs. If elements are strings, return them. If objects, assume {id: string, ...}
        return data.map((s: any) => typeof s === 'string' ? s : s.id).filter((id): id is string => typeof id === 'string');
    }
    return [];
}

export async function addClient(
  baseUrl: string,
  client: ClientConfig,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/clients/add`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(client),
  });

  assert.equal(response.ok, true, `Failed to add client: ${response.status}`);
}

export async function deleteClient(
  baseUrl: string,
  clientName: string,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/clients/delete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: clientName }),
  });

  assert.equal(response.ok, true, `Failed to delete client: ${response.status}`);
}

/**
 * Orchestrator for Blocked Services tests.
 */
export async function runBlockedServicesTestCase(
  testCase: BlockedServicesTestCase,
  context: BlockedServicesContext,
): Promise<void> {
  const fetchImpl = context.fetchImpl ?? fetch;
  const baseUrl = context.baseUrl;

  // 1. Set Global Blocked Services if config provided
  if (testCase.config) {
    await updateBlockedServices(baseUrl, testCase.config, fetchImpl);
  }

  // 2. Add Client if provided
  if (testCase.client) {
    await addClient(baseUrl, testCase.client, fetchImpl);
  }

  // 3. Verify DNS Resolution
  if (context.resolveDns) {
    const results = await context.resolveDns(testCase.domainToResolve);

    let pass = false;
    if (testCase.expectedResolution === '0.0.0.0') {
      pass = results.includes('0.0.0.0');
    } else if (testCase.expectedResolution === 'IP') {
        // Any non-zero IP
        pass = results.some(ip => ip !== '0.0.0.0');
    } else {
      pass = results.includes(testCase.expectedResolution);
    }

    assert.ok(
      pass,
      `DNS Resolution failed for "${testCase.name}". Domain: ${testCase.domainToResolve}. Expected: ${testCase.expectedResolution}, Got: ${results.join(', ')}`
    );
  }

  // 4. Cleanup Client
  if (testCase.client) {
    await deleteClient(baseUrl, testCase.client.name, fetchImpl);
  }

  // Cleanup Global Blocked Services (reset to empty)
  if (testCase.config) {
      await updateBlockedServices(baseUrl, { ids: [] }, fetchImpl);
  }
}
