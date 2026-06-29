import { GenericContainer, type StartedTestContainer, Wait } from 'testcontainers';
import { loginToAdGuardApi, type AdGuardApiClient } from '../shared/api/adguard-api';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../shared/adguard/admin.ts';

/** Retry a container start once on transient daemon errors (image resolution, races). */
export async function startWithRetry(
  factory: () => Promise<StartedTestContainer>,
): Promise<StartedTestContainer> {
  try {
    return await factory();
  } catch (first) {
    try {
      return await factory();
    } catch (second) {
      // Surface the original failure too — the retry often fails for the same reason.
      throw new Error(`container start failed twice; first: ${String(first)}; second: ${String(second)}`, { cause: second });
    }
  }
}

export interface DnsRecord {
  name: string;
  ttl: number;
  type: string;
  data: string;
}

const IMAGE = process.env.AGH_IMAGE ?? 'adguardhome-test:local';
const WEB_PORT = 3000;
const DNS_PORT = 53;
const CREDENTIALS = { username: ADMIN_USERNAME, password: ADMIN_PASSWORD };

/**
 * A running AdGuard Home instance backed by a testcontainer.
 * One per Playwright worker; reset between tests restores the pristine config.
 */
export class AdGuardContainer {
  private constructor(
    private readonly container: StartedTestContainer,
    readonly baseUrl: string,
    readonly dnsHost: string,
    readonly dnsPort: number,
  ) {}

  private static build(extraPorts: number[] = []): GenericContainer {
    return new GenericContainer(IMAGE)
      .withExposedPorts(WEB_PORT, DNS_PORT, ...extraPorts)
      // The image is built locally; never let testcontainers try to pull it
      // (avoids a manifest-resolution race under parallel workers).
      .withPullPolicy({ shouldPull: () => false })
      // Let AGH reach host-side mock upstreams via host.docker.internal everywhere.
      .withExtraHosts([{ host: 'host.docker.internal', ipAddress: 'host-gateway' }])
      .withWaitStrategy(Wait.forHttp('/', WEB_PORT).forStatusCodeMatching(() => true));
  }

  static async start(): Promise<AdGuardContainer> {
    const container = await startWithRetry(() => AdGuardContainer.build().start());
    const host = container.getHost();
    const baseUrl = `http://${host}:${container.getMappedPort(WEB_PORT)}`;
    return new AdGuardContainer(container, baseUrl, host, container.getMappedPort(DNS_PORT));
  }

  /** Boot with extra exposed ports and, optionally, a full custom config (applied via restart). */
  static async startCustom(opts: { config?: string; extraPorts?: number[] } = {}): Promise<AdGuardContainer> {
    const container = await startWithRetry(() => AdGuardContainer.build(opts.extraPorts ?? []).start());
    const host = container.getHost();
    const baseUrl = `http://${host}:${container.getMappedPort(WEB_PORT)}`;
    const agh = new AdGuardContainer(container, baseUrl, host, container.getMappedPort(DNS_PORT));
    if (opts.config) {
      try {
        await agh.writeConfigAndRestart(opts.config);
      } catch (err) {
        await agh.stop().catch(() => {});
        throw err;
      }
    }
    return agh;
  }

  /** The host-mapped port for a container port (e.g. an extra exposed pprof/https port). */
  mappedPort(containerPort: number): number {
    return this.container.getMappedPort(containerPort);
  }

  /** Read the live AdGuard Home config file. */
  async readConfig(): Promise<string> {
    const r = await this.exec(['cat', '/opt/adguardhome/conf/AdGuardHome.yaml']);
    if (r.exitCode !== 0) throw new Error(`readConfig failed: ${r.output}`);
    return r.output;
  }

  /** Replace the running config and restart AGH in-place (no pristine restore). */
  async writeConfigAndRestart(yaml: string): Promise<void> {
    const b64 = Buffer.from(yaml, 'utf8').toString('base64');
    const r = await this.exec(['bash', '-c',
      `echo '${b64}' | base64 -d > /opt/adguardhome/conf/AdGuardHome.yaml && /usr/local/bin/agh-apply.sh`]);
    if (r.exitCode !== 0) throw new Error(`writeConfigAndRestart failed (${r.exitCode}): ${r.output}`);
  }

  /** Authenticated API client (logs in as the pristine admin user). */
  async api(): Promise<AdGuardApiClient> {
    return loginToAdGuardApi(this.baseUrl, CREDENTIALS);
  }

  /** Restore the pristine config and restart AGH inside the container. */
  async reset(): Promise<void> {
    const result = await this.container.exec(['/usr/local/bin/agh-reset.sh']);
    if (result.exitCode !== 0) {
      throw new Error(`agh-reset failed (${result.exitCode}): ${result.output}`);
    }
  }

  async exec(cmd: string[]): Promise<{ exitCode: number; output: string }> {
    const result = await this.container.exec(cmd);
    return { exitCode: result.exitCode, output: result.output };
  }

  /**
   * Serve a filter list (e.g. a blocklist) from inside this container and return
   * a URL AGH can fetch. AGH resolves filter URLs through its own DNS (it does
   * not consult /etc/hosts) and the gateway IP is unreachable on Docker Desktop,
   * so serving over the container's own loopback is the portable approach.
   */
  async serveRules(filename: string, content: string): Promise<string> {
    if (!/^[\w.-]+$/.test(filename)) {
      throw new Error(`Unsafe rules filename: ${filename}`);
    }
    const b64 = Buffer.from(content, 'utf8').toString('base64');
    const r = await this.exec(['sh', '-c',
      `mkdir -p /tmp/rules && echo '${b64}' | base64 -d > /tmp/rules/${filename}` +
      ` && (pgrep -x busybox >/dev/null || setsid busybox httpd -p 127.0.0.1:8055 -h /tmp/rules </dev/null >/dev/null 2>&1 &)` +
      // Poll until the file is actually served instead of a fixed sleep.
      ` && for _ in $(seq 1 30); do curl -fsS -o /dev/null http://127.0.0.1:8055/${filename} && exit 0; sleep 0.1; done; exit 1`]);
    if (r.exitCode !== 0) {
      throw new Error(`serveRules(${filename}) failed to serve: ${r.output}`);
    }
    return `http://127.0.0.1:8055/${filename}`;
  }

  /**
   * Resolve a name through this AGH instance using `dnslookup` run inside the
   * container (against 127.0.0.1:53), so DNS tests never depend on host UDP
   * port mapping. Returns the DNS status code and any A/AAAA answer values.
   */
  async dnslookup(
    name: string,
    opts: { type?: string; timeoutSec?: number } = {},
  ): Promise<{ status: string; answers: string[]; records: DnsRecord[]; raw: string }> {
    const env: string[] = [];
    if (opts.type) env.push(`RRTYPE=${opts.type}`);
    if (opts.timeoutSec) env.push(`TIMEOUT=${opts.timeoutSec}`);
    const prefix = env.length ? ['env', ...env] : [];
    const { output } = await this.exec([...prefix, 'dnslookup', name, '127.0.0.1:53']);
    const status = output.match(/status:\s*([A-Z]+)/)?.[1] ?? 'UNKNOWN';
    const records: DnsRecord[] = [];
    const ansIdx = output.indexOf('ANSWER SECTION:');
    if (ansIdx >= 0) {
      for (const line of output.slice(ansIdx).split('\n').slice(1)) {
        if (line.trim().startsWith(';;')) break; // next section
        // e.g. "example.org.	300	IN	A	104.20.26.136"
        const m = line.match(/^(\S+)\s+(\d+)\s+IN\s+(\S+)\s+(.+?)\s*$/);
        if (m) records.push({ name: m[1], ttl: Number(m[2]), type: m[3], data: m[4] });
      }
    }
    const answers = records.filter((r) => r.type === 'A' || r.type === 'AAAA').map((r) => r.data);
    return { status, answers, records, raw: output };
  }

  /** Generate a self-signed cert/key inside the container. Returns their paths. */
  async setupTls(): Promise<{ certPath: string; keyPath: string }> {
    const certPath = '/opt/adguardhome/conf/cert.pem';
    const keyPath = '/opt/adguardhome/conf/key.pem';
    const r = await this.exec(['bash', '-c',
      `openssl req -x509 -newkey rsa:2048 -nodes -keyout ${keyPath} -out ${certPath} -days 1 -subj '/CN=localhost' 2>&1`]);
    if (r.exitCode !== 0) throw new Error(`setupTls failed: ${r.output}`);
    return { certPath, keyPath };
  }

  /** Overwrite the cert with garbage (for invalid-TLS reload tests). */
  async breakTls(): Promise<void> {
    const r = await this.exec(['bash', '-c', "echo 'not a cert' > /opt/adguardhome/conf/cert.pem"]);
    if (r.exitCode !== 0) throw new Error(`breakTls failed: ${r.output}`);
  }

  /** Regenerate a valid cert in place. */
  async restoreTls(): Promise<void> {
    await this.setupTls();
  }

  async stop(): Promise<void> {
    await this.container.stop();
  }
}
