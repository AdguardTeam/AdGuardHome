import { chromium, FullConfig } from '@playwright/test';

import { ADMIN_USERNAME, ADMIN_PASSWORD, PORT } from '../constants';

import { existsSync, renameSync, unlinkSync } from 'fs';
import { CONFIG_FILE, TEMP_CONFIG_FILE } from './globalSetup';

async function globalTeardown() {
    // Remove the test config file
    if (existsSync(CONFIG_FILE)) {
        unlinkSync(CONFIG_FILE);
    }

    // Restore the original config file if it exists
    if (existsSync(TEMP_CONFIG_FILE)) {
        renameSync(TEMP_CONFIG_FILE, CONFIG_FILE);
    }
}

export default globalTeardown;
