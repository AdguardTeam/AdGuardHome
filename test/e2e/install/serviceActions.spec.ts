import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { waitForHttpOk, waitForHttpFailure } from './service-helpers.ts';

test('4040 — Start/Stop service action', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  assert.equal((await freshAgh.installService()).exitCode, 0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const stopResult = await freshAgh.serviceAction('stop');
  assert.equal(stopResult.exitCode, 0, stopResult.output);
  assert.match(stopResult.output, /control action[:=]\s*stop/i);
  await waitForHttpFailure(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  assert.equal((await freshAgh.serviceState()).serviceStatus, 'stopped');

  const startResult = await freshAgh.serviceAction('start');
  assert.equal(startResult.exitCode, 0, startResult.output);
  assert.match(startResult.output, /control action[:=]\s*start/i);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  assert.equal((await freshAgh.serviceState()).serviceStatus, 'running');
});

test('4041 — Restart service action', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  assert.equal((await freshAgh.installService()).exitCode, 0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const restartResult = await freshAgh.serviceAction('restart');
  assert.equal(restartResult.exitCode, 0, restartResult.output);
  assert.match(restartResult.output, /control action[:=]\s*restart/i);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  assert.equal((await freshAgh.serviceState()).serviceStatus, 'running');
});

test('4042 — Status service action', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  assert.equal((await freshAgh.installService()).exitCode, 0);

  const runningStatus = await freshAgh.serviceAction('status');
  assert.equal(runningStatus.exitCode, 0, runningStatus.output);
  assert.match(runningStatus.output, /running/i);

  assert.equal((await freshAgh.serviceAction('stop')).exitCode, 0);
  assert.match((await freshAgh.serviceAction('status')).output, /stopped|inactive|dead/i);
});
// Case 4043 (invalid-TLS reload flow) moved to the integration runtime:
// tests/dnsSettings/tls_reload.spec.ts (uses AdGuardContainer TLS helpers).
