/**
 * CI freshness guard for the generated locale tables:
 * `src/common/intl/locales.generated.ts` and
 * `src/common/intl/services.groups.generated.ts`.
 *
 * Generates both to temp locations and compares them against the committed
 * versions.  No working-tree side effects — the real files are never modified
 * by this script.
 *
 * Usage:
 *   node ./scripts/check-locales.js
 *   npm run locales:check
 */

import { execFileSync } from 'node:child_process';
import { readFileSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { basename, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const GEN_SCRIPT = resolve(__dirname, 'generate-locales.js');
const INTL_DIR = resolve(__dirname, '..', 'src', 'common', 'intl');
const TMP_DIR = tmpdir();

/** The generated files to verify: where they live, where to test them, flag. */
const OUTPUTS = [
    {
        real: resolve(INTL_DIR, 'locales.generated.ts'),
        tmp: resolve(TMP_DIR, 'locales.generated.ts'),
        flag: '--out',
    },
    {
        real: resolve(INTL_DIR, 'services.groups.generated.ts'),
        tmp: resolve(TMP_DIR, 'services.groups.generated.ts'),
        flag: '--out-services',
    },
];

let exitCode = 0;

try {
    // Generate to temp files so we never touch the real working tree.
    execFileSync(
        process.execPath,
        [GEN_SCRIPT, ...OUTPUTS.flatMap(({ flag, tmp }) => [flag, tmp])],
        { stdio: 'inherit' },
    );

    for (const { real, tmp } of OUTPUTS) {
        // Compare contents directly — the files are small enough that a full
        // string comparison is both fast and precise.
        const generated = readFileSync(tmp, 'utf-8');
        const committed = readFileSync(real, 'utf-8');

        if (generated !== committed) {
            console.error('');
            console.error(`ERROR: ${basename(real)} is out of date.`);
            console.error('  Run:  npm run locales:generate');
            console.error('');
            exitCode = 1;
        } else {
            console.log(`${basename(real)} is up to date.`);
        }
    }
} finally {
    // Always clean up the temp files, even when the check fails.
    for (const { tmp } of OUTPUTS) {
        try {
            unlinkSync(tmp);
        } catch {
            // Best effort — the file might not exist if generate-locales itself
            // failed early.
        }
    }
}

process.exit(exitCode);
