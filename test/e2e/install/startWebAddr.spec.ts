import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { waitForHttpOk } from './service-helpers.ts';

test('4035 — Start via --web-addr', async ({ freshAgh }) => {
  test.setTimeout(180_000);
  // startUnconfigured launches AGH with `--web-addr 0.0.0.0:3000`, exercising the flag.
  await freshAgh.startUnconfigured();
  const url = freshAgh.appUrl();

  await waitForHttpOk(`${url}/install.html`, { timeoutMs: 30_000 });
  const pageHtml = await (await fetch(`${url}/install.html`)).text();
  assert.match(pageHtml, /AdGuard Home|installation/i);

  const proc = await freshAgh.exec(['bash', '-c', 'pgrep -x AdGuardHome >/dev/null && echo running || echo stopped']);
  assert.match(proc.output, /running/, 'AdGuardHome process should be running after --web-addr startup');
});
