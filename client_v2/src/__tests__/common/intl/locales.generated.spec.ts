import { describe, it, expect } from 'vitest';

import { LOCALES, LOCALE_LOADERS, LOCALE_CODES } from 'panel/common/intl/locales.generated';
import twosky from 'Twosky';

const homeV2 = twosky.find((p) => p.project_id === 'home_v2')!;

describe('locales.generated.ts', () => {
    const allTwoskyCodes = Object.keys(homeV2.languages);
    const expectedCodes = allTwoskyCodes.sort();

    it('LOCALE_CODES lists every home_v2 language', () => {
        expect([...LOCALE_CODES].sort()).toEqual(expectedCodes);
    });

    it('LOCALE_LOADERS has a loader for every non-en locale', () => {
        const loaderCodes = Object.keys(LOCALE_LOADERS).sort();
        expect(loaderCodes).toEqual(expectedCodes.filter((c) => c !== 'en'));
    });

    it('LOCALES only contains the base locale (en) initially', () => {
        expect(Object.keys(LOCALES)).toEqual(['en']);
    });

    it('base locale (en) is present and non-empty', () => {
        expect(LOCALES.en).toBeDefined();
        expect(Object.keys(LOCALES.en).length).toBeGreaterThan(0);
    });
});
