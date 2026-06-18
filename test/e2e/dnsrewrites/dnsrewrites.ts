import assert from 'node:assert/strict';

export interface DnsRewrite {
  domain: string;
  answer: string;
  enabled?: boolean;
}

export interface DnsRewriteSettings {
  enabled: boolean;
}

export type FetchLike = typeof fetch;
export type DnsResolver = (domain: string) => Promise<string[]>;

export interface DnsRewriteTestCase {
  name: string;
  rewrite: DnsRewrite;
  // If true, the test will enable global rewrite settings first
  enableGlobal?: boolean;
  // If true, verify DNS resolution. If false, just verify configuration.
  verifyResolution?: boolean;
  // Optional explicit expected resolution result, overrides rewrite.answer check
  expectedResolution?: string;
}

export interface DnsRewriteContext {
  baseUrl: string;
  fetchImpl?: FetchLike;
  resolveDns?: DnsResolver;
}

export async function addRewrite(
  baseUrl: string,
  rewrite: DnsRewrite,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/rewrite/add`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rewrite),
  });

  assert.equal(response.ok, true, `Failed to add rewrite rule: ${response.status}`);
}

export async function deleteRewrite(
  baseUrl: string,
  rewrite: DnsRewrite,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/rewrite/delete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rewrite),
  });

  assert.equal(response.ok, true, `Failed to delete rewrite rule: ${response.status}`);
}

export async function listRewrites(
  baseUrl: string,
  fetchImpl: FetchLike = fetch,
): Promise<DnsRewrite[]> {
  const response = await fetchImpl(`${baseUrl}/control/rewrite/list`, {
    method: 'GET',
    headers: { 'Accept': 'application/json' },
  });

  assert.equal(response.ok, true, `Failed to list rewrite rules: ${response.status}`);
  return (await response.json()) as DnsRewrite[];
}

export async function updateRewriteSettings(
  baseUrl: string,
  settings: DnsRewriteSettings,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/rewrite/settings/update`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  });

  assert.equal(response.ok, true, `Failed to update rewrite settings: ${response.status}`);
}

export async function updateRewrite(
  baseUrl: string,
  target: DnsRewrite,
  update: DnsRewrite,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/rewrite/update`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ target, update }),
  });

  assert.equal(response.ok, true, `Failed to update rewrite rule: ${response.status}`);
}

/**
 * Runs a complete test case for DNS Rewrites:
 * 1. Enable global settings (optional).
 * 2. Add the rewrite rule.
 * 3. Verify DNS resolution (optional).
 * 4. Cleanup (Delete the rule).
 */
export async function runDnsRewriteTestCase(
  testCase: DnsRewriteTestCase,
  context: DnsRewriteContext,
): Promise<void> {
  const fetchImpl = context.fetchImpl ?? fetch;
  const baseUrl = context.baseUrl;

  // 1. Enable Global Settings if requested
  if (testCase.enableGlobal) {
    await updateRewriteSettings(baseUrl, { enabled: true }, fetchImpl);
  }

  // 2. Add Rewrite Rule
  await addRewrite(baseUrl, testCase.rewrite, fetchImpl);

  // 3. Verify Resolution if requested
  if (testCase.verifyResolution && context.resolveDns) {
    const results = await context.resolveDns(testCase.rewrite.domain);

    // Normalize logic: "answer" in DnsRewrite might be an IP or a domain (CNAME)
    // The resolver mock should return an array. We check if the answer is contained.
    const expected = testCase.expectedResolution ?? testCase.rewrite.answer;
    const found = results.some(r => r === expected);

    assert.ok(
      found,
      `DNS Resolution failed for case "${testCase.name}". Expected: ${expected}, Got: ${results.join(', ')}`
    );
  }

  // 4. Cleanup
  await deleteRewrite(baseUrl, testCase.rewrite, fetchImpl);
}
