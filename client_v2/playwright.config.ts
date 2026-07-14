import { defineConfig, devices } from '@playwright/test';

import path from 'path';
import { fileURLToPath } from 'url';
import { ADMIN_USERNAME, CONFIG_FILE_PATH, PORT, WORK_DIR_PATH } from './tests/constants';

const configDir = path.dirname(fileURLToPath(import.meta.url));
const ADMIN_PASSWORD_HASH = '$2a$10$82RqoFQEf8GcFZwhCk4GFu.KHavhWaNajpZxCkdYsHhToNRe8ljO2';
const DNS_PORT = 5353;
const CONFIG_SCHEMA_VERSION = 34;

const shellQuote = (value: string) => `'${value.replace(/'/g, `'\\''`)}'`;

const prepareConfigScriptPath = path.resolve(configDir, './tests/e2e/prepareConfig.mjs');
const prepareConfigEnv = [
    `E2E_CONFIG_PATH=${shellQuote(CONFIG_FILE_PATH)}`,
    `E2E_WORK_DIR=${shellQuote(WORK_DIR_PATH)}`,
    `E2E_ADMIN_USERNAME=${shellQuote(ADMIN_USERNAME)}`,
    `E2E_ADMIN_PASSWORD_HASH=${shellQuote(ADMIN_PASSWORD_HASH)}`,
    `E2E_HTTP_PORT=${shellQuote(String(PORT))}`,
    `E2E_DNS_PORT=${shellQuote(String(DNS_PORT))}`,
    `E2E_SCHEMA_VERSION=${shellQuote(String(CONFIG_SCHEMA_VERSION))}`,
].join(' ');

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
    testDir: './tests/e2e',
    globalSetup: path.resolve(configDir, './tests/e2e/globalSetup.ts'),
    globalTeardown: path.resolve(configDir, './tests/e2e/globalTeardown.ts'),
    timeout: 30000,
    /* Run tests in files in parallel */
    fullyParallel: true,
    /* Fail the build on CI if you accidentally left test.only in the source code. */
    forbidOnly: !!process.env.CI,
    /* Retry on CI only */
    retries: process.env.CI ? 2 : 0,
    /* Run tests in parallel on CI. */
    workers: process.env.CI ? 4 : undefined,
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
        command: `${prepareConfigEnv} node ${shellQuote(prepareConfigScriptPath)} && ./AdGuardHome --local-frontend -v -c ${CONFIG_FILE_PATH} --work-dir ${WORK_DIR_PATH}`,
        url: 'http://127.0.0.1:3000',
        cwd: path.resolve(configDir, '..'),
        reuseExistingServer: !process.env.CI,
        timeout: 50000,
    },
});
