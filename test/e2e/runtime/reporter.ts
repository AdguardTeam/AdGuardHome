import type { Reporter, TestCase, TestResult, FullResult } from '@playwright/test/reporter';

/**
 * Prints a one-line-per-test summary at the end of the run, preserving the
 * testcase overview the suite previously produced via suite-summary.ts — but
 * from typed Playwright events instead of regex-parsed stdout.
 */
export default class SummaryReporter implements Reporter {
  // Keyed by test id so retries overwrite earlier attempts (one line per test).
  private lines = new Map<string, string>();

  formatLine(test: Pick<TestCase, 'title'>, result: Pick<TestResult, 'status'>): string {
    const mark =
      result.status === 'passed'
        ? '✔ PASS'
        : result.status === 'skipped'
          ? '﹣ SKIP'
          : '✖ FAIL';
    return `${mark}  ${test.title}`;
  }

  onTestEnd(test: TestCase, result: TestResult): void {
    this.lines.set(test.id, this.formatLine(test, result));
  }

  onEnd(result: FullResult): void {
    if (this.lines.size === 0) return;
    process.stdout.write('\n=== Testcase summary ===\n');
    process.stdout.write([...this.lines.values()].join('\n') + '\n');
    process.stdout.write(`=== Suite: ${result.status} ===\n`);
  }
}
