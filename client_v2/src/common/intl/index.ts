import { createSignal } from 'solid-js';

import { I18nInterface, Locale, translate } from '@adguard/translate';
import { BASE_LOCALE } from 'panel/helpers/twosky';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { LANGUAGE_QUERY_PARAM } from 'panel/helpers/constants';

import {
    LOCALES,
    LOCALE_LOADERS,
    LOCALE_CODES,
    LocaleMessage,
    SERVICES_BASE_LOCALE,
    SERVICES_LOCALE_LOADERS,
    ServicesMessage,
} from 'panel/common/intl/locales.generated';

export type LocalesType = ReturnType<I18nInterface['getUILanguage']>;

/**
 * Converts the blocked-services locale format (`{ key: { message } }`) to the
 * flat format used by `@adguard/translate`.
 */
const flattenServices = (services: ServicesMessage): LocaleMessage =>
    Object.fromEntries(
        Object.entries(services).map(([key, value]) => [
            key,
            typeof value === 'string' ? value : value.message,
        ]),
    );

/**
 * The live message map — starts with only the base locale (`en`),
 * populated with additional locales at runtime via {@link preloadLocale}.
 *
 * The base locale also includes the blocked-services group names, which live
 * in a separate Hostlists Registry translation project.
 */
const messages: Record<string, LocaleMessage> = {
    ...LOCALES,
    [BASE_LOCALE]: {
        ...(LOCALES[BASE_LOCALE] ?? {}),
        ...flattenServices(SERVICES_BASE_LOCALE),
    },
};

/** Locales whose main UI strings have been loaded. */
const loadedLocales = new Set<string>([BASE_LOCALE]);

/** Locales whose blocked-services strings have been loaded. */
const loadedServices = new Set<string>([BASE_LOCALE]);

/**
 * Converts a hyphenated twosky locale code to the underscore format that
 * {@link https://github.com/AdguardTeam/translate @adguard/translate}
 * expects for plural-form lookups (e.g. pt-br → pt_br).
 */
const toTranslateLocale = (code: string): Locale => {
    // si-lk → am (Amharic): identical CLDR plural rules (one: n=0..1, other).
    // This only affects plural-form indexing — actual strings come from si-lk.json.
    //
    // TODO(ik): Contribute missing `si` locale to @adguard/translate
    if (code === 'si-lk') return 'am' as Locale;

    // zh-hk / sr-cs → parent locale
    if (code === 'zh-hk') return 'zh' as Locale;
    if (code === 'sr-cs') return 'sr' as Locale;

    return code.replace(/-/g, '_') as Locale;
};

const resolveLanguage = (lng: string): LocalesType => {
    const l = lng.toLowerCase();

    if (LOCALE_CODES.has(l)) {
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
    const base = l.split('-')[0];
    if (LOCALE_CODES.has(base)) {
        return base as LocalesType;
    }

    return BASE_LOCALE as LocalesType;
};

export const i18n = (lang: LocalesType) => {
    const resolved = resolveLanguage(lang);
    return {
        getMessage: (key: string) => messages[resolved]?.[key] || '',
        getUILanguage: () => toTranslateLocale(resolved),
        getBaseMessage: (key: string) => messages.en![key] || key,
        getBaseUILanguage: () => BASE_LOCALE as LocalesType,
    };
};

/**
 * Detects the initial language using a priority chain: URL query param
 * (set after cross-port redirect from the install wizard) → localStorage
 * → browser language → base locale.
 */
export const getInitialLanguage = (): LocalesType => {
    if (typeof window === 'undefined') {
        return BASE_LOCALE as LocalesType;
    }

    const urlLang = new URL(window.location.href).searchParams.get(LANGUAGE_QUERY_PARAM);
    if (urlLang) {
        const resolved = resolveLanguage(urlLang);
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, resolved);
        return resolved;
    }

    const stored = LocalStorageHelper.getItem<string>(LOCAL_STORAGE_KEYS.LANGUAGE);
    if (stored) {
        return resolveLanguage(stored);
    }

    if (typeof navigator !== 'undefined' && navigator.language) {
        return resolveLanguage(navigator.language);
    }

    return BASE_LOCALE as LocalesType;
};

export const initialLanguage: LocalesType = getInitialLanguage();

// Always start with English (the only locale guaranteed to be in `messages`
// at module-init time).  The App.onMount preloads the actual detected locale
// and triggers a reactive re-render via changeLanguage.
const [lang, setLang] = createSignal<LocalesType>(BASE_LOCALE as LocalesType);

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

let translator = createSolidTranslator(i18n(BASE_LOCALE as LocalesType));

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

    /**
     * Changes the active language.  If the locale has not been loaded yet
     * (common at boot), it is fetched on-demand via the lazy
     * `LOCALE_LOADERS` map before the translator is rebuilt.
     */
    changeLanguage: async (newLang: LocalesType) => {
        const resolved = resolveLanguage(newLang);
        try {
            await preloadLocale(resolved);
        } catch (err) {
            // eslint-disable-next-line no-console
            console.warn('[i18n] Failed to load locale:', resolved, err);
            return;
        }
        translator = createSolidTranslator(i18n(resolved));
        setLang(resolved);
    },

    get translator() {
        return translator;
    },
};

/**
 * Loads a non-base locale into the `messages` map so that subsequent
 * `getMessage`/`getPlural` calls can serve it.  The main UI strings and the
 * blocked-services group names are loaded independently, since they come from
 * different Twosky projects and may already be loaded separately.
 */
export const preloadLocale = async (code: string) => {
    const loadMain = loadedLocales.has(code) ? undefined : LOCALE_LOADERS[code];
    const loadServices = loadedServices.has(code) ? undefined : SERVICES_LOCALE_LOADERS[code];

    if (!loadMain && !loadServices) {
        return;
    }

    const [mainResult, servicesResult] = await Promise.allSettled([
        loadMain?.(),
        loadServices?.(),
    ]);

    if (mainResult.status === 'rejected') {
        throw mainResult.reason;
    }

    let merged: LocaleMessage = { ...(messages[code] ?? {}) };

    if (mainResult.value) {
        merged = {
            ...merged,
            ...((mainResult.value as { default?: LocaleMessage }).default ??
                (mainResult.value as LocaleMessage)),
        };
        loadedLocales.add(code);
    }

    if (servicesResult.status === 'rejected') {
        // The main locale still loads; service group names fall back to English.
        // eslint-disable-next-line no-console
        console.warn('[i18n] Failed to load blocked-services locale:', code, servicesResult.reason);
    } else if (servicesResult.value) {
        const services =
            (servicesResult.value as { default?: ServicesMessage }).default ??
            (servicesResult.value as ServicesMessage);

        merged = { ...merged, ...flattenServices(services) };
        loadedServices.add(code);
    }

    messages[code] = merged;
};

// Fire-and-forget: preload the browser-detected locale as soon as the module
// loads.  Every entry point (dashboard, login, install, forgot_password)
// benefits without needing its own onMount preload gate.  The app renders
// immediately with English; once the locale chunk loads, changeLanguage
// triggers a reactive re-render.
if (typeof window !== 'undefined') {
    intl.changeLanguage(initialLanguage).catch((err) => {
        console.warn('[i18n] Failed to preload locale:', err);
    });
}

export default intl;
