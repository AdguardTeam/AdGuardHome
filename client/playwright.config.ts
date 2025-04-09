import { defineConfig, devices } from '@playwright/test';

import path from 'path';
import { CONFIG_FILE_PATH } from './tests/constants';

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
    testDir: './tests/e2e',
    globalSetup: path.resolve('./tests/e2e/globalSetup.ts'),
    globalTeardown: path.resolve('./tests/e2e/globalTeardown.ts'),
    timeout: 5000,
    /* Run tests in files in parallel */
    fullyParallel: true,
    /* Fail the build on CI if you accidentally left test.only in the source code. */
    forbidOnly: !!process.env.CI,
    /* Retry on CI only */
    retries: process.env.CI ? 2 : 0,
    /* Opt out of parallel tests on CI. */
    workers: process.env.CI ? 1 : undefined,
    /* Reporter to use. See https://playwright.dev/docs/test-reporters */
    reporter: 'html',
    /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
    use: {
        /* Base URL to use in actions like `await page.goto('/')`. */
        baseURL: 'http://127.0.0.1:3000',

        /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
        trace: 'on-first-retry',
        launchOptions: {
            headless: true,
        },
    },

    /* Configure projects for major browsers */
    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
        },
    ],

    webServer: {
        stdout: process.env.CI ? 'pipe' : 'ignore',
        command: `${!process.env.CI ? 'sudo ' : ''}./AdGuardHome --local-frontend -v -c ${CONFIG_FILE_PATH}`,
        url: 'http://127.0.0.1:3000',
        cwd: '..',
        reuseExistingServer: !process.env.CI,
        timeout: 10000,
    },
});
