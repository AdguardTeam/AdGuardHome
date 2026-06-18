import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { waitForHttpFailure } from './service-helpers.ts';

test('4038 — Service uninstall', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  assert.equal((await freshAgh.installService()).exitCode, 0);

  const firstUninstall = await freshAgh.serviceAction('uninstall');
  assert.equal(firstUninstall.exitCode, 0, firstUninstall.output);
  assert.match(firstUninstall.output, /control action[:=]\s*uninstall/i);

  await waitForHttpFailure(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });
  assert.equal((await freshAgh.serviceState()).serviceInstalled, false);

  const secondUninstall = await freshAgh.serviceAction('uninstall');
  assert.notEqual(secondUninstall.exitCode, 0, secondUninstall.output);
  assert.match(secondUninstall.output, /uninstall/i);
  assert.match(secondUninstall.output, /no such file|not installed|does not exist|failed|init already/i);
});
