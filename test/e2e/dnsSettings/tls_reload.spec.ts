import { test, expect } from '../runtime/fixtures';
import { AdGuardContainer } from '../runtime/adguard-container';
import { ADMIN_PASSWORD_HASH } from '../shared/adguard/admin.ts';

// Case 4043: starting with a broken TLS cert surfaces a TLS error; restoring a
// valid cert recovers the web interface.
test('4043 — Config reload on invalid TLS', async () => {
  test.setTimeout(120_000);
  const agh = await AdGuardContainer.startCustom({ extraPorts: [4433] });
  const tlsCfg = [
    'http:', '  address: 0.0.0.0:3000',
    // Upstream is incidental (this test only exercises TLS cert reload, never
    // resolves through it); a non-routable address keeps it off the internet.
    'dns:', '  bind_hosts: [0.0.0.0]', '  port: 53', '  upstream_dns: [192.0.2.1]',
    'tls:', '  enabled: true', '  server_name: localhost', '  force_https: false',
    '  port_https: 4433', '  allow_unencrypted_doh: true',
    '  certificate_path: /opt/adguardhome/conf/cert.pem', '  private_key_path: /opt/adguardhome/conf/key.pem',
    'users:', '  - name: admin',
    `    password: ${ADMIN_PASSWORD_HASH}`,
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
    // Match the cert/key failure specifically — 'initializing'/'failed' alone
    // would also match unrelated startup errors (port bind, config parse).
    expect(broken.output, `Expected a TLS cert error on broken-cert start, got: ${broken.output.slice(0, 300)}`).toMatch(/tls|certificate|private key|pem/i);

    // Restore a valid cert and bring AGH back up.
    await agh.restoreTls();
    await agh.exec(['bash', '-c', '/usr/local/bin/agh-apply.sh']);
    const ok = await agh.exec(['bash', '-c', 'curl -fsS --max-time 5 -o /dev/null http://127.0.0.1:3000/ && echo up']);
    expect(ok.output, 'Expected AGH web to recover after restoring a valid cert').toMatch(/up/);

    // The actual fix is TLS coming back: verify HTTPS on 4433 serves again.
    const https = await agh.exec(['bash', '-c', 'curl -fsSk --max-time 5 -o /dev/null https://127.0.0.1:4433/login.html && echo https-up']);
    expect(https.output, 'Expected HTTPS on 4433 to recover after restoring a valid cert').toMatch(/https-up/);
  } finally {
    await agh.stop();
  }
});
