/**
 * CI freshness guard for `src/common/intl/locales.generated.ts`.
 *
 * Generates the file to a temp location and compares it against the
 * committed version.  No working-tree side effects — the real file
 * is never modified by this script.
 *
 * Usage:
 *   node ./scripts/check-locales.js
 *   npm run locales:check
 */

import { execFileSync } from 'node:child_process';
import { readFileSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const GEN_SCRIPT = resolve(__dirname, 'generate-locales.js');
const REAL_FILE = resolve(__dirname, '..', 'src', 'common', 'intl', 'locales.generated.ts');
const TMP_FILE = resolve(tmpdir(), 'locales.generated.ts');

let exitCode = 0;

try {
    // Generate to a temp file so we never touch the real working tree.
    execFileSync(process.execPath, [GEN_SCRIPT, '--out', TMP_FILE], { stdio: 'inherit' });

    // Compare contents directly — the file is small enough that a full
    // string comparison is both fast and precise.
    const generated = readFileSync(TMP_FILE, 'utf-8');
    const committed = readFileSync(REAL_FILE, 'utf-8');

    if (generated !== committed) {
        console.error('');
        console.error('ERROR: locales.generated.ts is out of date.');
        console.error('  Run:  npm run locales:generate');
        console.error('');
        exitCode = 1;
    } else {
        console.log('locales.generated.ts is up to date.');
    }
} finally {
    // Always clean up the temp file, even when the check fails.
    try {
        unlinkSync(TMP_FILE);
    } catch {
        // Best effort — the file might not exist if generate-locales itself
        // failed early.
    }
}

process.exit(exitCode);
