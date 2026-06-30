import { test as base } from '@playwright/test';
import { AdGuardContainer } from './adguard-container';
import type { AdGuardApiClient } from '../shared/api/adguard-api';
import type { SystemdContainer } from './systemd-container';

type WorkerFixtures = {
  /** One AdGuard Home container per Playwright worker, reused across its tests. */
  aghWorker: AdGuardContainer;
};

type TestFixtures = {
  /** A reset AdGuard Home instance for this test (config restored to pristine). */
  agh: AdGuardContainer;
  /** Authenticated API client for {@link agh}. */
  api: AdGuardApiClient;
  /** A clean systemd host with no AGH installed (install/service E2E). */
  freshAgh: SystemdContainer;
};

export const test = base.extend<TestFixtures, WorkerFixtures>({
  aghWorker: [
    async ({}, use) => {
      const container = await AdGuardContainer.start();
      await use(container);
      await container.stop();
    },
    { scope: 'worker', timeout: 120_000 },
  ],

  agh: async ({ aghWorker }, use) => {
    await aghWorker.reset();
    await use(aghWorker);
  },

  api: async ({ agh }, use) => {
    await use(await agh.api());
  },

  // UI tests use relative navigation (page.goto('/')); point Playwright's
  // baseURL at the reset worker container so the browser hits this instance.
  baseURL: async ({ agh }, use) => {
    await use(agh.baseUrl);
  },

  freshAgh: async ({}, use) => {
    const { SystemdContainer } = await import('./systemd-container');
    const host = await SystemdContainer.start();
    await use(host);
    await host.stop();
  },
});

export { expect } from '@playwright/test';
