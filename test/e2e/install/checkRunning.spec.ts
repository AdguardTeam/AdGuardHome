import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { waitForHttpOk } from './service-helpers.ts';

test('4034 — Detect another running AdGuardHome', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  assert.equal((await freshAgh.installService()).exitCode, 0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const processDetected = (await freshAgh.exec(['bash', '-c', 'pgrep -x AdGuardHome >/dev/null && echo yes || echo no'])).output.includes('yes');
  assert.equal(processDetected, true, 'AdGuardHome should appear in the process list');

  const portDetected = (await freshAgh.exec(['bash', '-c', '(ss -tln 2>/dev/null || netstat -tln 2>/dev/null) | grep -q ":3000" && echo yes || echo no'])).output.includes('yes');
  assert.equal(portDetected, true, 'AdGuardHome should be listening on the web port');
});
