import { existsSync, unlinkSync } from 'fs';

export const CONFIG_FILE = '/tmp/AdGuard.temp.e2e.yaml';

async function globalTeardown() {
    // Remove the test config file
    if (existsSync(CONFIG_FILE)) {
        unlinkSync(CONFIG_FILE);
    }
}

export default globalTeardown;
