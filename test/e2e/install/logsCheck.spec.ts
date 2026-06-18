import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { enableVerboseLogging } from './logsCheck.ts';
import { waitForHttpOk } from './service-helpers.ts';

test('4033 — Logs check', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  assert.equal((await freshAgh.installService()).exitCode, 0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const initialLogs = await freshAgh.serviceLogs();
  assert.match(initialLogs, /info/i, 'Expected info logs after install');
  assert.doesNotMatch(initialLogs, /\[debug\]/i, 'Did not expect debug logs before enabling verbose');

  assert.equal((await freshAgh.serviceAction('stop')).exitCode, 0);
  await freshAgh.setVerbose(true);
  assert.equal((await freshAgh.serviceAction('start')).exitCode, 0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const updatedLogs = await freshAgh.serviceLogs();
  assert.match(updatedLogs, /\[debug\]/i, 'Expected debug logs after enabling verbose');
});
