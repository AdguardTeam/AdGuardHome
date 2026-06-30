import { test, expect } from '../runtime/fixtures';
import { waitForHttpOk } from './service-helpers.ts';

test('4034 — Detect another running AdGuardHome', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  expect((await freshAgh.installService()).exitCode).toBe(0);
  await waitForHttpOk(`${freshAgh.appUrl()}/login.html`, { timeoutMs: 30_000 });

  const processDetected = (await freshAgh.exec(['bash', '-c', 'pgrep -x AdGuardHome >/dev/null && echo yes || echo no'])).output.includes('yes');
  expect(processDetected, 'AdGuardHome should appear in the process list').toBe(true);

  const portDetected = (await freshAgh.exec(['bash', '-c', '(ss -tln 2>/dev/null || netstat -tln 2>/dev/null) | grep -q ":3000" && echo yes || echo no'])).output.includes('yes');
  expect(portDetected, 'AdGuardHome should be listening on the web port').toBe(true);
});
