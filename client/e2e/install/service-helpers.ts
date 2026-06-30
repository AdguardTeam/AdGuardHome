/** Poll an URL until it responds with one of the accepted statuses. */
export async function waitForHttpOk(url: string, opts: { timeoutMs?: number; expectedStatuses?: number[] } = {}): Promise<void> {
  const timeoutMs = opts.timeoutMs ?? 30_000;
  const expected = opts.expectedStatuses ?? [200];
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 2_000);
    try {
      const res = await fetch(url, { signal: controller.signal });
      if (expected.includes(res.status)) return;
    } catch {
      // not up yet
    } finally {
      clearTimeout(timer);
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`Timed out waiting for ${url} to become reachable`);
}

/** Poll an URL until it stops responding (connection refused). */
export async function waitForHttpFailure(url: string, opts: { timeoutMs?: number } = {}): Promise<void> {
  const timeoutMs = opts.timeoutMs ?? 30_000;
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 2_000);
    try {
      await fetch(url, { signal: controller.signal });
    } catch {
      return; // connection failed — service is down
    } finally {
      clearTimeout(timer);
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`Timed out waiting for ${url} to become unreachable`);
}
