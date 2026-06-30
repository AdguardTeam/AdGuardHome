import { test, expect } from '../runtime/fixtures';
import { authed } from '../shared/api/test-fetch.ts';

import { addBlockList, removeBlockList, updateBlockList, type BlockList } from '../blocklists/blocklists.ts';
import { waitForAnswers } from '../shared/dns/dns-test-helpers.ts';
import { startMockUpstream } from '../shared/dns/mock-upstream.ts';
import type { AdGuardContainer } from '../runtime/adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';

function useAllowlistUpstream(agh: AdGuardContainer, api: AdGuardApiClient, domains: string[]) {
  return startMockUpstream(agh, api, domains.map((domain, index) => ({ domain, type: 'A', data: `203.0.113.${index + 10}` })));
}

const waitForResolution = (agh: AdGuardContainer, domain: string, predicate: (answers: string[]) => boolean) =>
  waitForAnswers(agh, domain, 'A', predicate);

test('4175 — Allowlist overrides blocklist', async ({ agh, api }) => {
  test.setTimeout(90_000);
  const domains = ['vk.com', 'football.ua', 'cybersport.ru'];
  const upstream = await useAllowlistUpstream(agh, api, domains);
  try {
    const blocklistUrl = await agh.serveRules('blocklist.txt', '||vk.com^\n||football.ua^\n||cybersport.ru^');
    const allowlistUrl = await agh.serveRules('allowlist.txt', '@@||vk.com^\n@@||football.ua^\n@@||cybersport.ru^');
    const blocklist: BlockList = { name: 'Integration Blocklist', url: blocklistUrl, whitelist: false };
    const allowlist: BlockList = { name: 'Integration Allowlist', url: allowlistUrl, whitelist: true, enabled: true };

    await addBlockList(agh.baseUrl, blocklist, authed(api));
    await addBlockList(agh.baseUrl, allowlist, authed(api));

    for (const domain of domains) {
      const answers = await waitForResolution(agh, domain, (v) => v.length > 0 && !v.includes('0.0.0.0'));
      expect(answers.some((v) => v !== '0.0.0.0'), `Expected ${domain} reachable while allowlist enabled`).toBeTruthy();
    }

    await updateBlockList(agh.baseUrl, allowlist, { ...allowlist, enabled: false }, authed(api));
    for (const domain of domains) {
      const answers = await waitForResolution(agh, domain, (v) => v.includes('0.0.0.0'));
      expect(answers.includes('0.0.0.0'), `Expected ${domain} blocked after disabling allowlist`).toBeTruthy();
    }
  } finally { await upstream.stop(); }
});

test('4170/4172 — Add and remove allowlist', async ({ agh, api }) => {
  test.setTimeout(90_000);
  const domain = 'remove-allowlist.example';
  const upstream = await useAllowlistUpstream(agh, api, [domain]);
  try {
    const blocklistUrl = await agh.serveRules('blocklist.txt', `||${domain}^\n`);
    const allowlistUrl = await agh.serveRules('allowlist.txt', `@@||${domain}^\n`);
    const blocklist: BlockList = { name: 'Remove Allowlist Blocklist', url: blocklistUrl, whitelist: false };
    const allowlist: BlockList = { name: 'Remove Allowlist Test', url: allowlistUrl, whitelist: true, enabled: true };

    await addBlockList(agh.baseUrl, blocklist, authed(api));
    await addBlockList(agh.baseUrl, allowlist, authed(api));
    expect(!(await waitForResolution(agh, domain, (a) => a.length > 0 && !a.includes('0.0.0.0'))).includes('0.0.0.0')).toBeTruthy();

    await removeBlockList(agh.baseUrl, allowlist, authed(api));
    expect((await waitForResolution(agh, domain, (a) => a.includes('0.0.0.0'))).includes('0.0.0.0')).toBeTruthy();
  } finally { await upstream.stop(); }
});

test('4174 — Add allowlist with duplicate URL', async ({ agh, api }) => {
  const domain = 'dup-allowlist.example';
  const upstream = await useAllowlistUpstream(agh, api, [domain]);
  try {
    const allowlistUrl = await agh.serveRules('dup-allowlist.txt', `@@||${domain}^\n`);
    const allowlist: BlockList = { name: 'Dup Allowlist', url: allowlistUrl, whitelist: true };
    await addBlockList(agh.baseUrl, allowlist, authed(api));
    await expect(() => addBlockList(agh.baseUrl, allowlist, authed(api))).rejects.toThrow(/Failed to add blocklist: 400/);
  } finally { await upstream.stop(); }
});
