import { test, expect } from '../runtime/fixtures';
import { waitForHttpOk } from './service-helpers.ts';

test('4033 — Logs check', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  expect((await freshAgh.installService()).exitCode).toBe(0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const initialLogs = await freshAgh.serviceLogs();
  expect(initialLogs, 'Expected info logs after install').toMatch(/info/i);
  expect(initialLogs, 'Did not expect debug logs before enabling verbose').not.toMatch(/\[debug\]/i);

  expect((await freshAgh.serviceAction('stop')).exitCode).toBe(0);
  await freshAgh.setVerbose(true);
  expect((await freshAgh.serviceAction('start')).exitCode).toBe(0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const updatedLogs = await freshAgh.serviceLogs();
  expect(updatedLogs, 'Expected debug logs after enabling verbose').toMatch(/\[debug\]/i);
});
