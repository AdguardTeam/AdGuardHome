let cached: boolean | undefined;

/**
 * True when the Docker daemon runs a native Linux kernel reachable from the host,
 * so kernel ipset and reliable host<->container UDP/QUIC mapping work. False on
 * Docker Desktop for macOS/Windows, where those are unavailable.
 *
 * `process.platform === 'linux'` is a sufficient, dependency-free proxy: GitHub
 * `ubuntu-latest` runners are Linux; local macOS/Windows are not.
 */
export async function isLinuxDocker(): Promise<boolean> {
  if (cached !== undefined) return cached;
  cached = process.platform === 'linux';
  return cached;
}
