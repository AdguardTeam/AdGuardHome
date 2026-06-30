export interface RetryOptions {
  attempts: number;
  delayMs: number;
}

export interface WaitForOptions {
  timeoutMs: number;
  intervalMs: number;
}

export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function retry<T>(
  operation: () => Promise<T>,
  options: RetryOptions,
): Promise<T> {
  let lastError: unknown;

  for (let attempt = 1; attempt <= options.attempts; attempt += 1) {
    try {
      return await operation();
    } catch (error) {
      lastError = error;
    }

    if (attempt < options.attempts) {
      await sleep(options.delayMs);
    }
  }

  throw lastError instanceof Error ? lastError : new Error(String(lastError));
}

export async function waitFor<T>(
  operation: () => Promise<T | undefined>,
  options: WaitForOptions,
): Promise<T> {
  const start = Date.now();

  while (Date.now() - start < options.timeoutMs) {
    const result = await operation();
    if (result !== undefined) {
      return result;
    }

    await sleep(options.intervalMs);
  }

  throw new Error(`Timed out after ${options.timeoutMs}ms`);
}
