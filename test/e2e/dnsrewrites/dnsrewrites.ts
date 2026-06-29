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
