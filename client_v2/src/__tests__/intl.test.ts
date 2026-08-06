import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { translate } from '@adguard/translate';
import { LocalStorageHelper } from '../helpers/localStorageHelper';

// Ensure the real constants module is available even when other test files
// (e.g., install-store.test.ts) mock it with a partial stub.
vi.mock('panel/helpers/constants', async (importOriginal) =>
    importOriginal<typeof import('panel/helpers/constants')>(),
);

import {
    createSolidDefaultValues,
    solidMessageConstructor,
    getInitialLanguage,
} from '../common/intl/index';

/** Helper to create real DOM elements for test assertions (SolidJS has no h() export) */
const h = (tag: string, props: Record<string, string> | null, children?: string) => {
    const el = document.createElement(tag);
    if (props) {
        Object.entries(props).forEach(([key, value]) => {
            el.setAttribute(key, value);
        });
    }
    if (children !== undefined) {
        el.textContent = children;
    }
    return el;
};

// Minimal i18n mock for testing the translator layer
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockI18n = (locale: string): any => ({
    getMessage: (key: string) => {
        const messages: Record<string, string> = {
            plain_only: 'Hello %name%',
            with_code: 'Use <code>%value%</code> for DNS queries',
            with_link: 'Learn more at <a>our docs</a>',
            with_mixed: '<code>%ipv4%</code>,<code>%ipv6%</code>: regular DNS',
            with_strong: 'This is <strong>important</strong>',
            with_b: 'This is <b>bold</b>',
            with_i: 'This is <i>italic</i>',
            with_p: '<p>Paragraph text</p>',
        };
        return messages[key] || '';
    },
    getUILanguage: () => locale,
    getBaseMessage: (key: string) => mockI18n('en').getMessage(key),
    getBaseUILanguage: () => 'en',
});

describe('solidMessageConstructor', () => {
    it('joins arrays of only strings', () => {
        const result = solidMessageConstructor(['Hello ', 'world', '!']);
        expect(result).toBe('Hello world!');
    });

    it('keeps arrays with non-string elements', () => {
        const el = h('code', null, 'test');
        const result = solidMessageConstructor(['Use ', el, ' for DNS']);
        expect(Array.isArray(result)).toBe(true);
        expect((result as unknown[])[1]).toBe(el);
    });
});

describe('createSolidDefaultValues', () => {
    const getTranslator = () =>
        translate.createTranslator<any>(
            mockI18n('en'),
            solidMessageConstructor,
            createSolidDefaultValues(),
        );

    it('returns plain string for translation without tags', () => {
        const t = getTranslator();
        const result = t.getMessage('plain_only', { name: 'World' });
        expect(result).toBe('Hello World');
    });

    it('renders <code> tag as SolidJS element (not HTML string)', () => {
        const t = getTranslator();
        const result = t.getMessage('with_code', { value: '1.2.3.4' });

        // Should be an array because of the code element
        expect(Array.isArray(result)).toBe(true);
        const arr = result as unknown[];
        expect(arr).toHaveLength(3); // 'Use ', <code>, ' for DNS queries'

        // The code element should be a real h() element, not a string
        expect(arr[1]).not.toBe('<code>1.2.3.4</code>');
        expect(typeof arr[1]).not.toBe('string');
    });

    it('supports per-call tag overrides (e.g., <a> with JSX handler)', () => {
        const t = getTranslator();
        const result = t.getMessage('with_link', {
            a: (text: string) => h('a', { href: 'https://example.com' }, text),
        });

        expect(Array.isArray(result)).toBe(true);
        const arr = result as unknown[];
        // The <a> element should be the SolidJS element from our handler
        expect(typeof arr[2]).not.toBe('string');
    });

    it('renders <strong> as SolidJS element', () => {
        const t = getTranslator();
        const result = t.getMessage('with_strong');
        expect(Array.isArray(result)).toBe(true);
    });

    it('renders <code> fallback without per-call handler', () => {
        const t = getTranslator();
        // Only pass ipv4, ipv6 — no 'code' handler
        const result = t.getMessage('with_mixed', {
            ipv4: '94.140.14.140',
            ipv6: '2a10:50c0::1:ff',
        });

        expect(Array.isArray(result)).toBe(true);
        const arr = result as unknown[];
        // Both <code> elements should be SolidJS elements from defaults
        expect(typeof arr[0]).not.toBe('string'); // first code
        expect(typeof arr[2]).not.toBe('string'); // second code
    });

    it('handles <b>, <i>, <p> tags via defaults', () => {
        const t = getTranslator();
        const boldResult = t.getMessage('with_b');
        expect(Array.isArray(boldResult)).toBe(true);

        const italicResult = t.getMessage('with_i');
        expect(Array.isArray(italicResult)).toBe(true);

        const pResult = t.getMessage('with_p');
        expect(Array.isArray(pResult)).toBe(true);
    });
});

describe('intl.getMessage — missing placeholder fallback', () => {
    it('returns the raw key and warns when a placeholder value is missing', () => {
        const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

        const t = translate.createTranslator<any>(
            mockI18n('en'),
            solidMessageConstructor,
            createSolidDefaultValues(),
        );

        // Mirror the try-catch pattern from intl's getMessage
        const safeGetMessage = (key: string, values?: any) => {
            try {
                return t.getMessage(key, values);
            } catch (e) {
                console.warn('[i18n] Missing placeholder value for key:', key, e);
                return key;
            }
        };

        // 'plain_only' has %name% but we pass no values → should return the key
        const result = safeGetMessage('plain_only');
        expect(result).toBe('plain_only');
        expect(warnSpy).toHaveBeenCalledTimes(1);
        expect(warnSpy.mock.calls[0][1]).toBe('plain_only');

        warnSpy.mockRestore();
    });

    it('still returns formatted string when all values are provided', () => {
        const t = translate.createTranslator<any>(
            mockI18n('en'),
            solidMessageConstructor,
            createSolidDefaultValues(),
        );

        const safeGetMessage = (key: string, values?: any) => {
            try {
                return t.getMessage(key, values);
            } catch {
                return key;
            }
        };

        const result = safeGetMessage('plain_only', { name: 'World' });
        expect(result).toBe('Hello World');
    });

    it('returns the raw key when getPlural is missing a required placeholder', () => {
        const mockPluralI18n = (locale: string): any => ({
            getMessage: (key: string) => {
                const messages: Record<string, string> = {
                    plural_with_value: 'Found %count% results for %query%',
                };
                return messages[key] || '';
            },
            getUILanguage: () => locale,
            getBaseMessage: (key: string) => mockPluralI18n('en').getMessage(key),
            getBaseUILanguage: () => 'en',
        });

        const pt = translate.createTranslator<any>(
            mockPluralI18n('en'),
            solidMessageConstructor,
            createSolidDefaultValues(),
        );

        const safeGetPlural = (key: string, number: number, values?: any) => {
            try {
                return pt.getPlural(key, number, values);
            } catch {
                return key;
            }
        };

        // 'query' is missing from values → should return the key
        const result = safeGetPlural('plural_with_value', 5);
        expect(result).toBe('plural_with_value');
    });
});

describe('getInitialLanguage', () => {
    const LANGUAGE_KEY = 'language';

    const setLocation = (href: string) => {
        delete (globalThis as any).location;
        (globalThis as any).location = new URL(href);
    };

    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
        // Default: a URL with no lang param (so tests without setLocation work)
        setLocation('http://127.0.0.1:3001/login.html');
    });

    afterEach(() => {
        localStorage.clear();
        delete (globalThis as any).location;
    });

    const storedLang = () => LocalStorageHelper.getItem<string>(LANGUAGE_KEY);

    it('resolves a valid URL query param and persists to localStorage', () => {
        setLocation('http://127.0.0.1:3001/login.html?lang=zh-cn');

        const result = getInitialLanguage();

        expect(result).toBe('zh-cn');
        expect(storedLang()).toBe('zh-cn');
    });

    it('resolves an invalid URL query param to en and persists en', () => {
        setLocation('http://127.0.0.1:3001/login.html?lang=garbage');

        const result = getInitialLanguage();

        expect(result).toBe('en');
        expect(storedLang()).toBe('en');
    });

    it('resolves abbreviated zh to zh-cn', () => {
        setLocation('http://127.0.0.1:3001/login.html?lang=zh');

        const result = getInitialLanguage();

        expect(result).toBe('zh-cn');
        expect(storedLang()).toBe('zh-cn');
    });

    it('falls back to localStorage when no URL param is present', () => {
        LocalStorageHelper.setItem(LANGUAGE_KEY, 'de');

        const result = getInitialLanguage();

        expect(result).toBe('de');
    });

    it('falls back to navigator.language when no URL param or localStorage', () => {
        vi.stubGlobal('navigator', { language: 'fr-FR' });

        const result = getInitialLanguage();

        expect(result).toBe('fr');
        vi.unstubAllGlobals();
    });

    it('falls back to en when no source provides a valid language', () => {
        // No URL param, no localStorage, stub navigator to a nonsense value
        vi.stubGlobal('navigator', { language: 'xx-XX' });

        const result = getInitialLanguage();

        expect(result).toBe('en');
        vi.unstubAllGlobals();
    });

    it('returns en when typeof window is undefined (SSR)', () => {
        const originalWindow = globalThis.window;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        delete (globalThis as any).window;

        const result = getInitialLanguage();

        expect(result).toBe('en');

        (globalThis as any).window = originalWindow;
    });
});
