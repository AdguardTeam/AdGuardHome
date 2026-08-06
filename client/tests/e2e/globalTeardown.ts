import { existsSync, unlinkSync } from 'fs';
import { CONFIG_FILE_PATH } from '../constants';

async function globalTeardown() {
    // Remove the test config file
    if (existsSync(CONFIG_FILE_PATH)) {
        unlinkSync(CONFIG_FILE_PATH);
    }
}

export default globalTeardown;
