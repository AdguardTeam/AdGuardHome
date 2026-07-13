import { createSignal } from 'solid-js';

import { I18nInterface, translate } from '@adguard/translate';
import { BASE_LOCALE } from 'panel/helpers/twosky';

import en from 'panel/__locales/en.json';
import de from 'panel/__locales/de.json';
import es from 'panel/__locales/es.json';
import fr from 'panel/__locales/fr.json';
import it from 'panel/__locales/it.json';
import ja from 'panel/__locales/ja.json';
import ko from 'panel/__locales/ko.json';
import ptBr from 'panel/__locales/pt-br.json';
import ptPt from 'panel/__locales/pt-pt.json';
import ru from 'panel/__locales/ru.json';
import zhCn from 'panel/__locales/zh-cn.json';
import zhTw from 'panel/__locales/zh-tw.json';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';

export type LocalesType = ReturnType<I18nInterface['getUILanguage']>;

type LocalesTypes = Partial<Record<LocalesType, Record<string, string>>>;

const LOCALES = {
    en,
    de,
    es,
    fr,
    it,
    ja,
    ko,
    ru,
    'pt-br': ptBr,
    'pt-pt': ptPt,
    'zh-cn': zhCn,
    'zh-tw': zhTw,
};

const messages: LocalesTypes = LOCALES;

const resolveLanguage = (lng: string): LocalesType => {
    const l = lng.toLowerCase();

    if (messages[l as LocalesType]) {
        return l as LocalesType;
    }

    // Chinese locales
    if (l.startsWith('zh')) {
        if (l.includes('tw') || l.includes('hk') || l.includes('mo')) {
            return 'zh-tw' as LocalesType;
        }
        return 'zh-cn' as LocalesType;
    }

    // Portuguese locales
    if (l.startsWith('pt')) {
        if (l.includes('br')) {
            return 'pt-br' as LocalesType;
        }
        return 'pt-pt' as LocalesType;
    }

    // Try base language (e.g., en-us -> en)
    const base = l.split('-')[0] as LocalesType;
    if (messages[base]) {
        return base;
    }

    return BASE_LOCALE as LocalesType;
};

export const i18n = (lang: LocalesType) => {
    const resolved = resolveLanguage(lang);
    return {
        getMessage: (key: string) => messages[resolved]?.[key] || '',
        getUILanguage: () => resolved,
        getBaseMessage: (key: string) => messages.en![key] || key,
        getBaseUILanguage: () => BASE_LOCALE as LocalesType,
    };
};

const detectedLanguage = ((typeof window !== 'undefined' &&
    typeof localStorage !== 'undefined' &&
    LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LANGUAGE)) ||
    (typeof navigator !== 'undefined' && (navigator.language as string)) ||
    BASE_LOCALE) as LocalesType;

const initialLanguage: LocalesType = resolveLanguage(detectedLanguage);

// Solid-reactive language signal
const [lang, setLang] = createSignal<LocalesType>(initialLanguage);

/**
 * Creates default value functions for common HTML tags.
 * Mirrors createReactTranslator's createDefaultValues(), but returns real
 * DOM nodes via document.createElement instead of React element descriptors.
 *
 * Why document.createElement and not solid-js/h or <Dynamic>:
 * @adguard/translate is built around the React/Preact model where
 * createElement() returns a cheap descriptor object the framework reconciles
 * later. SolidJS has no equivalent — its JSX compiles to fine-grained template
 * cloning, and h()/Dynamic need a reactive owner. A plain DOM node has no
 * ownership entanglement: SolidJS just inserts and removes it
 * (normalizeIncomingArray keeps anything with a `.nodeType` as-is), which is
 * the most robust option for these static styling tags.
 *
 * `<br>` is intentionally NOT provided: the parser rejects `<br/>`/`<br>` as
 * unbalanced tags, and void tags must be plain strings (not functions), so a
 * `br` handler could never be invoked anyway.
 *
 * These are fallbacks for when a component doesn't pass a per-call handler.
 * The library merges user-supplied values over these, with user values winning.
 * For interactive tags (<a>, <button>), always pass a per-call JSX handler so
 * events go through SolidJS's delegation system.
 */
export const createSolidDefaultValues = () => ({
    // Inline elements
    strong: (children: string) => {
        const el = document.createElement('strong');
        el.textContent = children;
        return el;
    },
    code: (children: string) => {
        const el = document.createElement('code');
        el.textContent = children;
        return el;
    },
    b: (children: string) => {
        const el = document.createElement('b');
        el.textContent = children;
        return el;
    },
    i: (children: string) => {
        const el = document.createElement('i');
        el.textContent = children;
        return el;
    },
    em: (children: string) => {
        const el = document.createElement('em');
        el.textContent = children;
        return el;
    },

    // Block elements
    p: (children: string) => {
        const el = document.createElement('p');
        el.textContent = children;
        return el;
    },
});

/**
 * Custom message constructor for SolidJS.
 * When all formatted elements are strings, join them (backward compatible).
 * When JSX elements are present, return the array — SolidJS renders
 * arrays of strings + JSX elements natively.
 */
export const solidMessageConstructor = (formatted: unknown[]): string | unknown[] => {
    if (formatted.every((child) => typeof child === 'string')) {
        return (formatted as string[]).join('');
    }
    return formatted;
};

/**
 * Builds a @adguard/translate translator configured for SolidJS.
 *
 * Hand-rolled `createSolidTranslator` adapter: the library ships React/Preact
 * factories but no Solid one, so we reuse the generic createTranslator and
 * inject:
 *   - solidMessageConstructor: keeps arrays of nodes intact (the default
 *     join('') would stringify DOM/JSX nodes to "[object Object]");
 *   - createSolidDefaultValues(): renders styling tags as real DOM nodes.
 *
 * See createSolidDefaultValues for why document.createElement is used.
 */
const createSolidTranslator = (i18nInstance: ReturnType<typeof i18n>) =>
    translate.createTranslator<any>(
        i18nInstance,
        solidMessageConstructor,
        createSolidDefaultValues(),
    );

let translator = createSolidTranslator(i18n(initialLanguage));

const intl = {
    getMessage: (key: string, values?: any) => {
        lang(); // track the language signal for reactivity
        try {
            return translator.getMessage(key, values);
        } catch (e) {
            // eslint-disable-next-line no-console
            console.warn('[i18n] Missing placeholder value for key:', key, e);
            return key;
        }
    },

    getPlural: (key: string, number: number, values?: any) => {
        lang(); // track the language signal for reactivity
        try {
            return translator.getPlural(key, number, values);
        } catch (e) {
            // eslint-disable-next-line no-console
            console.warn('[i18n] Missing placeholder value for plural key:', key, e);
            return key;
        }
    },

    getUILanguage: () => lang(),

    getBaseMessage: (key: string) => {
        return messages.en?.[key] || key;
    },

    getBaseUILanguage: () => BASE_LOCALE as LocalesType,

    changeLanguage: (newLang: LocalesType) => {
        const resolved = resolveLanguage(newLang);
        translator = createSolidTranslator(i18n(resolved));
        setLang(resolved);
    },

    get translator() {
        return translator;
    },
};

export default intl;
