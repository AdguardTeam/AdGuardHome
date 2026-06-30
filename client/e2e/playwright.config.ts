import { defineConfig, devices } from '@playwright/test';

// Curated AdGuard Home black-box e2e suite: only the high-signal ("solid")
// testcases that map 1:1 to a catalogue case and fail on a real product
// regression. Each test title is "<caseId> — <name>".
export default defineConfig({
  testDir: '.',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // Each worker boots its own AGH container; cap concurrency to keep Docker sane.
  workers: process.env.CI ? 4 : 3,
  reporter: [
    ['./runtime/reporter.ts'],
    ['list'],
  ],
  use: { trace: 'on-first-retry' },
  projects: [
    {
      name: 'integration',
      testMatch: [
        'dnsSettings/**/*.spec.ts',
        'customRules/**/*.spec.ts',
        'dnsrewrites/**/*.spec.ts',
        'blocklists/**/*.spec.ts',
        'allowlists/**/*.spec.ts',
        'blockedServices/**/*.spec.ts',
        'clients/**/*.spec.ts',
        'generalSettings/**/*.spec.ts',
      ],
    },
    {
      // Privileged: boots a systemd host container (service install / runtime checks).
      name: 'install',
      testMatch: ['install/**/*.spec.ts'],
      fullyParallel: false,
      workers: 2,
    },
    {
      name: 'ui',
      testMatch: ['ui/**/*.spec.ts'],
      use: { ...devices['Desktop Chrome'] },
      fullyParallel: false,
    },
  ],
});
