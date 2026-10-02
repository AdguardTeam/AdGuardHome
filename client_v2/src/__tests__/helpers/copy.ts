import { vi } from 'vitest';

import enJson from 'panel/__locales/en.json';

/**
 * The base locale, re-exported so tests can assert a key's copy without a
 * second import path.
 */
export const en = enJson;

export type CopyKey = keyof typeof en;

/** Values interpolated into `%name%` placeholders, as `intl.getMessage` takes. */
export type CopyValues = Record<string, string | number>;

/** Replaces every `%name%` with the matching value, ignoring the rest. */
const interpolate = (template: string, values?: CopyValues): string => {
    if (!values) return template;

    return Object.entries(values).reduce(
        (text, [name, value]) => text.replaceAll(`%${name}%`, String(value)),
        template,
    );
};

/**
 * The copy `en.json` defines for `key`, exactly as the app receives it.
 *
 * Use it for values a function returns — a validator message, a store field —
 * and assert against a literal only when that literal is not copy at all
 * (fixture data such as a domain, an IP address, or a test-only label).
 */
export const copy = (key: CopyKey, values?: CopyValues): string =>
    interpolate(en[key], values);

/**
 * Collapses runs of whitespace the way DOM Testing Library does when it reads
 * an element's text.
 *
 * A few `en.json` values use a non-breaking space (`&nbsp;`) for typography.
 * The queries normalize the *rendered* text — so the DOM shows a plain space —
 * but compare the matcher string as written, which means asserting with the raw
 * value fails even though the element is on screen.
 */
const normalizeForDom = (text: string): string => text.replace(/\s+/g, ' ').trim();

/**
 * The copy `en.json` defines for `key` as it reads in the DOM, for
 * `getByText`-style assertions.
 */
export const copyInDom = (key: CopyKey, values?: CopyValues): string =>
    normalizeForDom(copy(key, values));

/**
 * Selects the English form of a pipe-delimited plural key.
 *
 * Mirrors `@adguard/translate`: English declares exactly three forms
 * (`zero | singular | plural`), the selected form is trimmed, and `count` is
 * forced to `number` — so an empty zero form renders as an empty string for 0,
 * exactly like the real `intl.getPlural`.
 */
export const copyPlural = (key: CopyKey, number: number, values?: CopyValues): string => {
    const forms = en[key].split('|');
    const index = number === 0 ? 0 : number === 1 ? 1 : 2;
    const form = (forms[index] ?? forms[forms.length - 1] ?? '').trim();

    return interpolate(form, { ...values, count: number });
};

const lookup = (key: string): CopyKey => {
    if (!(key in en)) {
        throw new Error(`Unknown intl key: ${key}`);
    }

    return key as CopyKey;
};

/**
 * A `panel/common/intl` mock backed by the real base locale, so component tests
 * render the copy the app ships.  Use it from a hoisted async factory:
 *
 * ```ts
 * vi.mock('panel/common/intl', async () =>
 *     (await import('panel/__tests__/helpers/copy')).createIntlMock(),
 * );
 * ```
 *
 * An unknown key throws rather than rendering the key, so a renamed or removed
 * key fails the test instead of silently changing what is displayed.
 */
export const createIntlMock = () => ({
    default: {
        getMessage: (key: string, values?: CopyValues): string =>
            copy(lookup(key), values),
        getPlural: (key: string, number: number, values?: CopyValues): string =>
            copyPlural(lookup(key), number, values),
        getBaseMessage: (key: string): string => en[key as CopyKey] ?? key,
        getUILanguage: (): string => 'en',
        changeLanguage: vi.fn(),
    },
});
