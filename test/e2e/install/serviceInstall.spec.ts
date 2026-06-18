import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { waitForHttpOk } from './service-helpers.ts';

test('4039 — Service install', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  const installResult = await freshAgh.installService();
  assert.equal(installResult.exitCode, 0, installResult.output);
  assert.match(installResult.output, /control action[:=]\s*install/i);
  assert.match(installResult.output, /starting service|started|service: started/i);

  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const runningState = await freshAgh.serviceState();
  assert.equal(runningState.serviceInstalled, true);
  assert.equal(runningState.serviceStatus, 'running');

  const repeated = await freshAgh.serviceAction('install');
  assert.notEqual(repeated.exitCode, 0, repeated.output);
  assert.match(repeated.output, /already|exists|installed|init already/i);

  const finalState = await freshAgh.serviceState();
  assert.equal(finalState.serviceInstalled, true);
  assert.equal(finalState.serviceStatus, 'running');
});
