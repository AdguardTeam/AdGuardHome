import assert from 'node:assert/strict';

export interface BlockList {
  name: string;
  url: string;
  whitelist?: boolean;
  enabled?: boolean;
}

export type FetchLike = typeof fetch;

export async function addBlockList(
  baseUrl: string,
  blocklist: BlockList,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/filtering/add_url`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(blocklist),
  });

  if (!response.ok) {
      throw new Error(`Failed to add blocklist: ${response.status}`);
  }
}

export async function removeBlockList(
  baseUrl: string,
  blocklist: BlockList,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl}/control/filtering/remove_url`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      url: blocklist.url,
      whitelist: blocklist.whitelist ?? false,
    }),
  });

  if (!response.ok) {
      throw new Error(`Failed to remove blocklist: ${response.status}`);
  }
}

export async function updateBlockList(
  baseUrl: string,
  blocklist: BlockList,
  data: BlockList,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
    const payload = {
        url: blocklist.url,
        whitelist: blocklist.whitelist ?? false,
        data: {
            name: data.name,
            url: data.url,
            enabled: data.enabled ?? true,
            whitelist: data.whitelist ?? false,
        }
    };

  const response = await fetchImpl(`${baseUrl}/control/filtering/set_url`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
      throw new Error(`Failed to update blocklist: ${response.status}`);
  }
}

export async function fetchFilteringStatus(
    baseUrl: string,
    fetchImpl: FetchLike = fetch,
): Promise<any> {
    const response = await fetchImpl(`${baseUrl}/control/filtering/status`, {
        method: 'GET',
    });
    if (!response.ok) {
        throw new Error(`Failed to fetch filtering status: ${response.status}`);
    }
    return response.json();
}
