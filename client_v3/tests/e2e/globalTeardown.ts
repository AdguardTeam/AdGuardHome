import { existsSync, rmSync, unlinkSync } from 'fs';
import { CONFIG_FILE_PATH, WORK_DIR_PATH } from '../constants';

async function globalTeardown() {
    // Remove the test config file
    if (existsSync(CONFIG_FILE_PATH)) {
        unlinkSync(CONFIG_FILE_PATH);
    }

    if (existsSync(WORK_DIR_PATH)) {
        rmSync(WORK_DIR_PATH, { force: true, recursive: true });
    }
}

export default globalTeardown;
