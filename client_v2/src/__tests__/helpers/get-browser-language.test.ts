import { describe, expect, it, vi, afterEach } from 'vitest';

import { getBrowserLanguage } from '../../helpers/helpers';

// Mock the twosky LANGUAGES map to keep tests deterministic
vi.mock('../../helpers/twosky', () => ({
    LANGUAGES: {
        en: 'English',
        de: 'Deutsch',
        fr: 'Français',
        'zh-cn': '简体中文',
        'pt-br': 'Português (BR)',
    },
    LANGUAGE_NAMES: {
        en: 'English',
        de: 'Deutsch',
        fr: 'Français',
        'zh-cn': '简体中文',
        'pt-br': 'Português (BR)',
    },
    BASE_LOCALE: 'en',
}));

const { LocalStorageHelper } = await import('../../helpers/localStorageHelper');
const getStoredItemSpy = vi.spyOn(LocalStorageHelper, 'getItem');

const navigatorSpy = vi.spyOn(globalThis.navigator, 'language', 'get');

afterEach(() => {
    getStoredItemSpy.mockReset();
    navigatorSpy.mockReset();
});

describe('getBrowserLanguage', () => {
    describe('localStorage takes priority', () => {
        it('returns stored language when it matches a supported locale', () => {
            getStoredItemSpy.mockReturnValue('de');
            expect(getBrowserLanguage()).toBe('de');
        });

        it('falls past localStorage when the stored code is unsupported', () => {
            getStoredItemSpy.mockReturnValue('xx');
            navigatorSpy.mockReturnValue('fr');
            expect(getBrowserLanguage()).toBe('fr');
        });
    });

    describe('browser language detection', () => {
        it('matches full locale code like zh-cn', () => {
            navigatorSpy.mockReturnValue('zh-CN');
            expect(getBrowserLanguage()).toBe('zh-cn');
        });

        it('matches base language when full locale is unsupported', () => {
            navigatorSpy.mockReturnValue('fr-FR');
            expect(getBrowserLanguage()).toBe('fr');
        });

        it('matches base language from pt-BR', () => {
            navigatorSpy.mockReturnValue('pt-BR');
            // "pt-br" is a supported full locale, so it matches directly
            expect(getBrowserLanguage()).toBe('pt-br');
        });

        it('falls back to base language for unsupported region', () => {
            navigatorSpy.mockReturnValue('de-AT');
            // "de-at" is NOT in LANGUAGES, but base "de" is
            expect(getBrowserLanguage()).toBe('de');
        });
    });

    describe('fallback to en', () => {
        it('returns en when localStorage is empty and navigator is missing', () => {
            getStoredItemSpy.mockReturnValue(null);
            navigatorSpy.mockReturnValue('');
            expect(getBrowserLanguage()).toBe('en');
        });

        it('returns en when both sources yield unsupported codes', () => {
            getStoredItemSpy.mockReturnValue('yy');
            navigatorSpy.mockReturnValue('zz-ZZ');
            expect(getBrowserLanguage()).toBe('en');
        });
    });
});
