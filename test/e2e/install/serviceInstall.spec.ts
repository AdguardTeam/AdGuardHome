import { test, expect } from '../runtime/fixtures';
import { waitForHttpOk } from './service-helpers.ts';

test('4039 — Service install', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  const installResult = await freshAgh.installService();
  expect(installResult.exitCode, installResult.output).toBe(0);
  expect(installResult.output).toMatch(/control action[:=]\s*install/i);
  expect(installResult.output).toMatch(/starting service|started|service: started/i);

  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const runningState = await freshAgh.serviceState();
  expect(runningState.serviceInstalled).toBe(true);
  expect(runningState.serviceStatus).toBe('running');

  const repeated = await freshAgh.serviceAction('install');
  expect(repeated.exitCode, repeated.output).not.toBe(0);
  expect(repeated.output).toMatch(/already|exists|installed|init already/i);

  const finalState = await freshAgh.serviceState();
  expect(finalState.serviceInstalled).toBe(true);
  expect(finalState.serviceStatus).toBe('running');
});
