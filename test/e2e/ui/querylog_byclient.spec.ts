import { test as base, expect } from '../runtime/fixtures';
import { startCluster, type AghCluster } from '../runtime/network';
import { loginToAdGuardUi, queryLogRow, queryLogSearchInput } from '../shared/ui/adguard-playwright.ts';

// Case 4196 needs AGH to see a non-loopback client, so we run a companion DNS
// client on a shared Docker network and point the browser's baseURL at that AGH.
const test = base.extend<{ cluster: AghCluster }>({
  cluster: async ({}, use) => {
    const c = await startCluster();
    await use(c);
    await c.stop();
  },
  baseURL: async ({ cluster }, use) => {
    await use(cluster.aghBaseUrl);
  },
});

test('4196 — Query log search by client', async ({ page, cluster }) => {
  test.setTimeout(120_000);
  const domain = 'querylog-by-client.example';
  await cluster.client.dnslookup(domain); // query from the companion (non-loopback) client
  const clientIp = cluster.client.ip();

  await loginToAdGuardUi(page);
  await page.getByRole('link', { name: 'Query Log' }).click();
  await expect(page).toHaveURL(/#logs/);
  await expect(page.locator('body')).toContainText(domain);

  await queryLogRow(page, domain).getByText(clientIp, { exact: true }).click();
  await expect.poll(async () => queryLogSearchInput(page).inputValue()).toContain(clientIp);
  await expect(page.locator('body')).toContainText(domain);
});
