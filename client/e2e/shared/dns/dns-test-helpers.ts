import type { AdGuardContainer } from '../../runtime/adguard-container';
import { waitFor } from '../polling/retry.ts';

interface PollOpts { timeoutMs?: number; intervalMs?: number }
const DEFAULTS: Required<PollOpts> = { timeoutMs: 15_000, intervalMs: 500 };

/** Resolve a name through the container and return trimmed, non-empty answer values. */
export async function resolveAnswers(agh: AdGuardContainer, domain: string, type: string = 'A'): Promise<string[]> {
  const { answers } = await agh.dnslookup(domain, { type });
  return answers.map((a) => a.trim()).filter((a) => a.length > 0);
}

/** Poll until the A/AAAA answers satisfy the predicate; returns those answers. */
export async function waitForAnswers(
  agh: AdGuardContainer,
  domain: string,
  type: 'A' | 'AAAA',
  predicate: (answers: string[]) => boolean,
  opts: PollOpts = {},
): Promise<string[]> {
  return waitFor(async () => {
    const { answers } = await agh.dnslookup(domain, { type });
    return predicate(answers) ? answers : undefined;
  }, { ...DEFAULTS, ...opts });
}

/** Poll until the answers + DNS status satisfy the predicate; returns both. */
export async function waitForDnsResult(
  agh: AdGuardContainer,
  domain: string,
  type: string,
  predicate: (answers: string[], rcode: string) => boolean,
  opts: PollOpts = {},
): Promise<{ answers: string[]; rcode: string }> {
  return waitFor(async () => {
    const { answers, status } = await agh.dnslookup(domain, { type, timeoutSec: 3 });
    return predicate(answers, status) ? { answers, rcode: status } : undefined;
  }, { ...DEFAULTS, ...opts });
}

/** Poll until the DNS status code equals the expected value. */
export async function waitForDnsStatus(agh: AdGuardContainer, domain: string, expected: string, opts: PollOpts = {}): Promise<void> {
  await waitFor(async () => {
    const { status } = await agh.dnslookup(domain, { type: 'A' });
    return status === expected ? status : undefined;
  }, { timeoutMs: 10_000, intervalMs: 500, ...opts });
}
