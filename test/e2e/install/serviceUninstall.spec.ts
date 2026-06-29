import { test, expect } from '../runtime/fixtures';
import { waitForHttpFailure } from './service-helpers.ts';

test('4038 — Service uninstall', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  expect((await freshAgh.installService()).exitCode).toBe(0);

  const firstUninstall = await freshAgh.serviceAction('uninstall');
  expect(firstUninstall.exitCode, firstUninstall.output).toBe(0);
  expect(firstUninstall.output).toMatch(/control action[:=]\s*uninstall/i);

  await waitForHttpFailure(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  expect((await freshAgh.serviceState()).serviceInstalled).toBe(false);

  const secondUninstall = await freshAgh.serviceAction('uninstall');
  expect(secondUninstall.exitCode, secondUninstall.output).not.toBe(0);
  expect(secondUninstall.output).toMatch(/uninstall/i);
  expect(secondUninstall.output).toMatch(/no such file|not installed|does not exist|failed|init already/i);
});
