import { GenericContainer, type StartedTestContainer, Wait } from 'testcontainers';
import { startWithRetry } from './adguard-container';
import { ADMIN_PASSWORD_HASH } from '../shared/adguard/admin.ts';

const IMAGE = process.env.AGH_SYSTEMD_IMAGE ?? 'adguardhome-systemd:local';
const AGH_VERSION = process.env.ADGUARD_VERSION ?? 'v0.107.52';
// AGH_VERSION is interpolated into a download URL / shell command; constrain it.
if (!/^v?[\w.-]+$/.test(AGH_VERSION)) {
  throw new Error(`Invalid ADGUARD_VERSION: ${AGH_VERSION}`);
}

/**
 * A clean systemd host with no AdGuard Home installed. Install/service E2E tests
 * run the real installer (curl script or `AdGuardHome -s install`) against it.
 */
export class SystemdContainer {
  private constructor(
    private readonly container: StartedTestContainer,
    readonly host: string,
  ) {}

  static async start(): Promise<SystemdContainer> {
    const container = await startWithRetry(() =>
      new GenericContainer(IMAGE)
        .withPullPolicy({ shouldPull: () => false })
        .withPrivilegedMode(true)
        .withBindMounts([{ source: '/sys/fs/cgroup', target: '/sys/fs/cgroup', mode: 'rw' }])
        .withTmpFs({ '/run': '', '/run/lock': '', '/tmp': '' })
        .withExposedPorts(3000)
        // No port listens until AGH is installed; wait for systemd to finish booting.
        .withWaitStrategy(
          Wait.forSuccessfulCommand(
            "bash -c 'systemctl is-system-running 2>/dev/null | grep -qE \"running|degraded\"'",
          ).withStartupTimeout(60_000),
        )
        .start(),
    );
    return new SystemdContainer(container, container.getHost());
  }

  async exec(cmd: string[]): Promise<{ exitCode: number; output: string }> {
    const result = await this.container.exec(cmd);
    return { exitCode: result.exitCode, output: result.output };
  }

  mappedPort(port: number): number {
    return this.container.getMappedPort(port);
  }

  /** Unpack the AGH binary into /opt: prefer the build baked into the image
   *  (so tests exercise THIS checkout), fall back to a pinned download. */
  async installAghBinary(): Promise<void> {
    const r = await this.exec([
      'bash',
      '-c',
      'set -eo pipefail; ' +
      'if [ -f /opt/agh-dist/AdGuardHome_linux_amd64.tar.gz ]; then ' +
      '  tar -xz -f /opt/agh-dist/AdGuardHome_linux_amd64.tar.gz -C /opt; ' +
      'else ' +
      `  curl -fsSL --retry 3 --retry-connrefused --connect-timeout 10 --max-time 300 "https://github.com/AdguardTeam/AdGuardHome/releases/download/${AGH_VERSION}/AdGuardHome_linux_amd64.tar.gz" | tar -xz -C /opt; ` +
      'fi; ' +
      'test -x /opt/AdGuardHome/AdGuardHome',
    ]);
    if (r.exitCode !== 0) throw new Error(`AGH install failed: ${r.output}`);
  }

  /** Install AGH as a systemd service, pre-configured with the admin user. */
  async installService(): Promise<{ exitCode: number; output: string }> {
    await this.installAghBinary();
    const config = [
      'http:',
      '  address: 0.0.0.0:3000',
      'dns:',
      '  bind_hosts: [0.0.0.0]',
      '  port: 53',
      'users:',
      '  - name: admin',
      `    password: ${ADMIN_PASSWORD_HASH}`,
      'log:',
      '  file: /var/log/AdGuardHome.log',
      '  verbose: false',
      'schema_version: 20',
      '',
    ].join('\n');
    const b64 = Buffer.from(config, 'utf8').toString('base64');
    await this.exec(['bash', '-c', `echo '${b64}' | base64 -d > /opt/AdGuardHome/AdGuardHome.yaml`]);
    return this.serviceAction('install');
  }

  /** Run an AGH service control action and return its combined output. */
  async serviceAction(
    action: 'install' | 'uninstall' | 'start' | 'stop' | 'restart' | 'status' | 'reload',
  ): Promise<{ exitCode: number; output: string }> {
    return this.exec(['bash', '-c', `cd /opt/AdGuardHome && ./AdGuardHome -s ${action} -w /opt/AdGuardHome -c /opt/AdGuardHome/AdGuardHome.yaml 2>&1`]);
  }

  /** Snapshot of the systemd service install/run state. */
  async serviceState(): Promise<{ serviceInstalled: boolean; serviceStatus: 'running' | 'stopped' | 'missing' }> {
    const installed = (await this.exec(['bash', '-c', 'test -f /etc/systemd/system/AdGuardHome.service && echo yes || echo no'])).output.includes('yes');
    if (!installed) return { serviceInstalled: false, serviceStatus: 'missing' };
    const active = (await this.exec(['bash', '-c', 'systemctl is-active AdGuardHome 2>/dev/null || true'])).output.trim();
    return { serviceInstalled: true, serviceStatus: active === 'active' ? 'running' : 'stopped' };
  }

  /** Logs for the AGH service: the configured log file, plus journald as a fallback. */
  async serviceLogs(): Promise<string> {
    return (await this.exec(['bash', '-c',
      'cat /var/log/AdGuardHome.log 2>/dev/null; ' +
      'journalctl -u AdGuardHome --no-pager 2>/dev/null; ' +
      'journalctl _COMM=AdGuardHome --no-pager 2>/dev/null; ' +
      'true'])).output;
  }

  /** Toggle the `log.verbose` setting in the installed config. */
  async setVerbose(enabled: boolean): Promise<void> {
    await this.exec(['bash', '-c', `sed -i 's/^\\( *\\)verbose:.*/\\1verbose: ${enabled}/' /opt/AdGuardHome/AdGuardHome.yaml`]);
  }

  /** Install the binary and start AGH (as a plain process) in first-run install
   *  mode — no config file, so the setup wizard and /control/install/configure
   *  are exposed on port 3000. */
  async startUnconfigured(): Promise<void> {
    await this.installAghBinary();
    await this.exec(['bash', '-c',
      'mkdir -p /opt/AdGuardHome && rm -f /opt/AdGuardHome/AdGuardHome.yaml' +
      ' && setsid /opt/AdGuardHome/AdGuardHome --web-addr 0.0.0.0:3000 --no-check-update' +
      ' -w /opt/AdGuardHome </dev/null >/dev/null 2>&1 &']);
  }

  /** Run the official curl install script with the given flags inside the container. */
  async runCurlScript(flags: string): Promise<{ exitCode: number; output: string }> {
    if (!/^[\w -]*$/.test(flags)) {
      throw new Error(`Unsafe install-script flags: ${flags}`);
    }
    return this.exec(['bash', '-lc',
      `curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- ${flags}`]);
  }

  appUrl(): string {
    return `http://${this.host}:${this.mappedPort(3000)}`;
  }

  async stop(): Promise<void> {
    await this.container.stop();
  }
}
