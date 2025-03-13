import { test } from '@playwright/test';
import { execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

test.describe('Uninstall and Cleanup', () => {
    test('should uninstall AdGuard Home and clean up files', async ({ page }) => {
        // This test would normally call the uninstaller and verify files are removed
        // In a real implementation, this would perform actual uninstallation steps
        
        // Example of how you might check if files were removed:
        // const adguardPath = '/opt/AdGuardHome';
        // const exists = fs.existsSync(adguardPath);
        // expect(exists).toBeFalsy();
    });
});
