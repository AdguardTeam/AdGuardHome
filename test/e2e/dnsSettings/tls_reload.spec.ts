import assert from 'node:assert/strict';
import { test } from '../runtime/fixtures';
import { AdGuardContainer } from '../runtime/adguard-container';

// Case 4043: starting with a broken TLS cert surfaces a TLS error; restoring a
// valid cert recovers the web interface.
test('4043 — Config reload on invalid TLS', async () => {
  test.setTimeout(120_000);
  const agh = await AdGuardContainer.startCustom({ extraPorts: [4433] });
  const tlsCfg = [
    'http:', '  address: 0.0.0.0:3000',
    'dns:', '  bind_hosts: [0.0.0.0]', '  port: 53', '  upstream_dns: [8.8.8.8]',
    'tls:', '  enabled: true', '  server_name: localhost', '  force_https: false',
    '  port_https: 4433', '  allow_unencrypted_doh: true',
    '  certificate_path: /opt/adguardhome/conf/cert.pem', '  private_key_path: /opt/adguardhome/conf/key.pem',
    'users:', '  - name: admin',
    '    password: $2b$12$aw6lk4Cdfc/b69rFQVqSrutVmh6UJ.ORxpQ10.fj685NVWmDiDO9O',
    'schema_version: 32', '',
  ].join('\n');
  try {
    await agh.setupTls();
    await agh.writeConfigAndRestart(tlsCfg);

    // Break the cert, stop AGH, then start it in the foreground with a timeout so
    // its TLS error is captured directly (and the process can't hang the exec).
    await agh.breakTls();
    const broken = await agh.exec(['bash', '-c',
      'pkill -x AdGuardHome 2>/dev/null; sleep 0.5; ' +
      'timeout 8 /opt/AdGuardHome/AdGuardHome --no-check-update ' +
      '-c /opt/adguardhome/conf/AdGuardHome.yaml -w /opt/adguardhome/work 2>&1 | head -200']);
    assert.match(broken.output, /tls|certificate|private key|pem|initializing|failed/i,
      `Expected a TLS error on broken-cert start, got: ${broken.output.slice(0, 300)}`);

    // Restore a valid cert and bring AGH back up.
    await agh.restoreTls();
    await agh.exec(['bash', '-c', '/usr/local/bin/agh-apply.sh']);
    const ok = await agh.exec(['bash', '-c', 'curl -fsS --max-time 5 -o /dev/null http://127.0.0.1:3000/ && echo up']);
    assert.match(ok.output, /up/, 'Expected AGH web to recover after restoring a valid cert');

    // The actual fix is TLS coming back: verify HTTPS on 4433 serves again.
    const https = await agh.exec(['bash', '-c', 'curl -fsSk --max-time 5 -o /dev/null https://127.0.0.1:4433/login.html && echo https-up']);
    assert.match(https.output, /https-up/, 'Expected HTTPS on 4433 to recover after restoring a valid cert');
  } finally {
    await agh.stop();
  }
});
