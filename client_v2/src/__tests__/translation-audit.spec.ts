import { afterEach, describe, expect, it } from 'vitest';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

import { validatePlurals } from '../../scripts/translation-audit.js';

/** Locale dirs created by the tests, removed afterwards. */
const tempDirs: string[] = [];

const writeLocales = async (files: Record<string, Record<string, string>>) => {
    const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'agh-plurals-'));
    tempDirs.push(dir);

    for (const [locale, messages] of Object.entries(files)) {
        await fs.writeFile(path.join(dir, `${locale}.json`), JSON.stringify(messages));
    }

    return dir;
};

const germanLocales = new Map([['de', 'de']]);

describe('validatePlurals', () => {
    afterEach(async () => {
        await Promise.all(
            tempDirs.splice(0).map((dir) => fs.rm(dir, { force: true, recursive: true })),
        );
    });

    it('flags a plural key that lost its empty zero form', async () => {
        const dir = await writeLocales({
            de: { last_days: 'Letzter %count% Tag | Letzte %count% Tage' },
        });

        await expect(validatePlurals(dir, germanLocales, new Set(['last_days']))).resolves.toEqual([
            {
                file: 'de.json',
                key: 'last_days',
                value: 'Letzter %count% Tag | Letzte %count% Tage',
            },
        ]);
    });

    it('accepts the same value once the zero form is back', async () => {
        const dir = await writeLocales({
            de: { last_days: '| Letzter %count% Tag | Letzte %count% Tage' },
        });

        await expect(validatePlurals(dir, germanLocales, new Set(['last_days']))).resolves.toEqual(
            [],
        );
    });

    it('skips double-pipe strings that are not plural keys', async () => {
        const dir = await writeLocales({
            en: { user_rules_example_block: '<code>||%domain%^</code> Block access' },
        });

        await expect(validatePlurals(dir, new Map([['en', 'en']]), new Set())).resolves.toEqual([]);
    });
});
