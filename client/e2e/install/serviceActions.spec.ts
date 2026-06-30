import { test, expect } from '../runtime/fixtures';
import { waitForHttpOk, waitForHttpFailure } from './service-helpers.ts';

test('4040 — Start/Stop service action', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  expect((await freshAgh.installService()).exitCode).toBe(0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const stopResult = await freshAgh.serviceAction('stop');
  expect(stopResult.exitCode, stopResult.output).toBe(0);
  expect(stopResult.output).toMatch(/control action[:=]\s*stop/i);
  await waitForHttpFailure(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  expect((await freshAgh.serviceState()).serviceStatus).toBe('stopped');

  const startResult = await freshAgh.serviceAction('start');
  expect(startResult.exitCode, startResult.output).toBe(0);
  expect(startResult.output).toMatch(/control action[:=]\s*start/i);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  expect((await freshAgh.serviceState()).serviceStatus).toBe('running');
});

test('4041 — Restart service action', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  expect((await freshAgh.installService()).exitCode).toBe(0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const restartResult = await freshAgh.serviceAction('restart');
  expect(restartResult.exitCode, restartResult.output).toBe(0);
  expect(restartResult.output).toMatch(/control action[:=]\s*restart/i);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  expect((await freshAgh.serviceState()).serviceStatus).toBe('running');
});

test('4042 — Status service action', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  expect((await freshAgh.installService()).exitCode).toBe(0);

  const runningStatus = await freshAgh.serviceAction('status');
  expect(runningStatus.exitCode, runningStatus.output).toBe(0);
  expect(runningStatus.output).toMatch(/running/i);

  expect((await freshAgh.serviceAction('stop')).exitCode).toBe(0);
  expect((await freshAgh.serviceAction('status')).output).toMatch(/stopped|inactive|dead/i);
});
// Case 4043 (invalid-TLS reload flow) moved to the integration runtime:
// tests/dnsSettings/tls_reload.spec.ts (uses AdGuardContainer TLS helpers).
