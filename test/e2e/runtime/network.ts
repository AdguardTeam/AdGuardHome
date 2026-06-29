import { GenericContainer, Network, type StartedNetwork, type StartedTestContainer, Wait } from 'testcontainers';
import { startWithRetry } from './adguard-container';

const AGH_IMAGE = process.env.AGH_IMAGE ?? 'adguardhome-test:local';
const CLIENT_IMAGE = process.env.AGH_CLIENT_IMAGE ?? 'adguardhome-client:local';

export interface DnsResult {
  status: string;
  answers: string[];
  records: Array<{ name: string; ttl: number; type: string; data: string }>;
}

export class ClientContainer {
  constructor(private readonly c: StartedTestContainer, private readonly networkIp: string) {}

  /** The client's address on the shared network — what AGH sees as the query source. */
  ip(): string { return this.networkIp; }

  async exec(cmd: string[]): Promise<{ exitCode: number; output: string }> {
    const r = await this.c.exec(cmd);
    return { exitCode: r.exitCode, output: r.output };
  }

  /** Resolve a name via AGH over the network (default: plain DNS on `agh:53`). */
  async dnslookup(name: string, server = 'agh:53', opts: { type?: string } = {}): Promise<DnsResult> {
    const prefix = opts.type ? ['env', `RRTYPE=${opts.type}`] : [];
    const { output } = await this.exec([...prefix, 'dnslookup', name, server]);
    const status = output.match(/status:\s*([A-Z]+)/)?.[1] ?? 'UNKNOWN';
    const records: DnsResult['records'] = [];
    const idx = output.indexOf('ANSWER SECTION:');
    if (idx >= 0) {
      for (const line of output.slice(idx).split('\n').slice(1)) {
        if (line.trim().startsWith(';;')) break;
        const m = line.match(/^(\S+)\s+(\d+)\s+IN\s+(\S+)\s+(.+?)\s*$/);
        if (m) records.push({ name: m[1], ttl: Number(m[2]), type: m[3], data: m[4] });
      }
    }
    return { status, answers: records.filter((r) => r.type === 'A' || r.type === 'AAAA').map((r) => r.data), records };
  }
}

export interface AghCluster {
  network: StartedNetwork;
  aghContainer: StartedTestContainer;
  aghBaseUrl: string; // host-mapped web URL for API/UI
  client: ClientContainer;
  stop: () => Promise<void>;
}

/** Start AGH and a companion `dnslookup` client on a shared Docker network. */
export async function startCluster(): Promise<AghCluster> {
  const network = await new Network().start();
  let aghContainer: StartedTestContainer | undefined;
  let clientC: StartedTestContainer | undefined;
  try {
    aghContainer = await startWithRetry(() =>
      new GenericContainer(AGH_IMAGE)
        .withPullPolicy({ shouldPull: () => false })
        .withNetwork(network).withNetworkAliases('agh')
        .withExposedPorts(3000)
        .withWaitStrategy(Wait.forHttp('/', 3000).forStatusCodeMatching(() => true))
        .start());
    const aghBaseUrl = `http://${aghContainer.getHost()}:${aghContainer.getMappedPort(3000)}`;
    clientC = await startWithRetry(() =>
      new GenericContainer(CLIENT_IMAGE).withPullPolicy({ shouldPull: () => false }).withNetwork(network).start());
    const ip = clientC.getIpAddress(network.getName());
    const agh = aghContainer;
    const client = clientC;
    return {
      network,
      aghContainer: agh,
      aghBaseUrl,
      client: new ClientContainer(client, ip),
      stop: async () => {
        // Best-effort teardown: one failing stop must not skip the others, but
        // surface errors so a leak isn't completely silent.
        await client.stop().catch((e) => console.warn('[startCluster] client.stop failed:', e));
        await agh.stop().catch((e) => console.warn('[startCluster] agh.stop failed:', e));
        await network.stop().catch((e) => console.warn('[startCluster] network.stop failed:', e));
      },
    };
  } catch (err) {
    // Tear down whatever already started so a partial failure doesn't leak.
    await clientC?.stop().catch(() => {});
    await aghContainer?.stop().catch(() => {});
    await network.stop().catch(() => {});
    throw err;
  }
}
