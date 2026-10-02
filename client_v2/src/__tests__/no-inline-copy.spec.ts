import { describe, it, expect } from 'vitest';
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

import en from 'panel/__locales/en.json';

/**
 * Tests must not re-state user-facing copy: `en.json` owns the wording, and a
 * literal duplicate turns every reword into a test failure.  Assert through
 * `panel/__tests__/helpers/copy` (`copy('key', values)`) or a `createIntlMock()`
 * instead, so the tests pin *which* message is produced rather than its text.
 *
 * This guard fails on a literal that matches a base-locale value — verbatim, or
 * with its `%placeholder%`s filled in, or as the argument of a query that reads
 * rendered text.
 */

/**
 * Copy shorter than this is usually a generic UI word (`Save`, `Block`,
 * `Upstream`) that fixtures reuse legitimately, and matching every use of it
 * would be noisy.  Text queries still catch the short ones (see `TEXT_QUERY`).
 */
const MIN_LENGTH = 15;

/** Shortest copy a query may name before it is worth a key lookup. */
const MIN_QUERY_LENGTH = 4;

/** Literal characters a template must keep for a filled-in match to count. */
const MIN_TEMPLATE_TEXT = 20;

const here = resolve(process.cwd(), 'src/__tests__');

if (!existsSync(here)) {
    throw new Error(`Expected to find the test directory at ${here}`);
}

/**
 * Files that may legitimately contain copy: the i18n tests exercise the locale
 * tables, and the helper's own test builds its expectations from them.
 */
const ALLOWED = new Set([
    'no-inline-copy.spec.ts',
    join('common', 'intl', 'locales.generated.spec.ts'),
    'intl.test.ts',
    join('helpers', 'copy.test.ts'),
]);

/** Every base-locale value → the keys that define it. */
const keysByValue = new Map<string, string[]>();

/** Templates with placeholders, for the filled-in case. */
const interpolated: { key: string; re: RegExp }[] = [];

/** DOM Testing Library's normalizer: collapse whitespace, then trim. */
const normalize = (text: string): string => text.replace(/\s+/g, ' ').trim();

for (const [key, value] of Object.entries(en)) {
    if (typeof value !== 'string') continue;

    const addKey = (text: string) =>
        keysByValue.set(text, [...new Set([...(keysByValue.get(text) ?? []), key])]);

    addKey(value);
    // A few values use `&nbsp;`; the queries read the rendered text, which is
    // collapsed, so the collapsed form is copy too.
    addKey(normalize(value));

    const literalText = value.replace(/%\w+%/g, '');
    if (value.includes('%') && literalText.trim().length >= MIN_TEMPLATE_TEXT) {
        const pattern = value
            .split(/%\w+%/)
            .map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
            .join('.+?');

        interpolated.push({ key, re: new RegExp(`^${pattern}$`, 's') });
        interpolated.push({ key, re: new RegExp(`^${normalize(pattern)}$`, 's') });
    }
}

const walk = (dir: string): string[] =>
    readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const path = join(dir, entry.name);
        if (entry.isDirectory()) return walk(path);
        if (!/\.(test|spec)\.tsx?$/.test(entry.name)) return [];

        return [path];
    });

/** Strips comments so prose in a spec is never mistaken for an assertion. */
const stripComments = (source: string): string =>
    source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1');

const literalsOf = (source: string): string[] => [
    ...source.matchAll(/'([^'\\\n]*)'|"([^"\\\n]*)"|`([^`\\]*)`/g),
].map((match) => match[1] ?? match[2] ?? match[3] ?? '');

/**
 * Queries and matchers whose argument is the copy a user reads.  These carry a
 * much lower length limit: `screen.getByText('Show more')` is exactly the
 * coupling this guard exists to prevent, and that string is nine characters.
 */
const TEXT_QUERY =
    /(?:getByText|queryByText|findByText|getAllByText|queryAllByText|findAllByText|getByLabelText|getByPlaceholderText|toHaveTextContent|toBe|toContain)\s*[:(]\s*(['"])([^'"\n]*)\1/g;

/**
 * Accessible-name lookups.  `name` only counts inside a query call, so a data
 * fixture such as `{ name: 'Other' }` is not mistaken for copy.
 */
const NAME_QUERY =
    /(?:getByRole|getAllByRole|queryByRole|queryAllByRole|findByRole|findAllByRole|getByLabelText)\s*\([^;]*?\bname:\s*(['"])([^'"\n]*)\1/g;

const describeKey = (keys: string[]): string => keys.map((key) => `'${key}'`).join(' or ');

/**
 * The copy a spec re-states, with a suggested key for each case.  Split out so
 * the rules can be tested against synthetic sources below.
 */
const copyOffendersIn = (source: string): string[] => {
    const stripped = stripComments(source);
    const offenders: string[] = [];
    // One suggestion per literal: a query on long copy is also a plain literal,
    // and reporting both would just be noise.
    const reported = new Set<string>();

    const report = (literal: string, suggestion: string) => {
        reported.add(literal);
        offenders.push(suggestion);
    };

    for (const literal of literalsOf(stripped)) {
        if (literal.length < MIN_LENGTH) continue;

        const keys = keysByValue.get(literal);
        if (keys) {
            report(literal, `"${literal}" → use copy(${describeKey(keys)})`);

            continue;
        }

        const filled = interpolated.find(({ re }) => re.test(literal));
        if (filled) {
            report(literal, `"${literal}" → use copy('${filled.key}', …)`);
        }
    }

    for (const re of [TEXT_QUERY, NAME_QUERY]) {
        for (const [, , literal] of stripped.matchAll(re)) {
            if (literal.length < MIN_QUERY_LENGTH || reported.has(literal)) continue;

            const keys = keysByValue.get(normalize(literal));
            if (keys) {
                report(literal, `query on "${literal}" → use copyInDom(${describeKey(keys)})`);
            }
        }
    }

    return offenders;
};

describe('tests do not inline user-facing copy', () => {
    const files = walk(here).filter((path) => !ALLOWED.has(relative(here, path)));

    it('finds the specs to check', () => {
        expect(files.length).toBeGreaterThan(50);
    });

    it.each(files.map((path) => [relative(here, path), path] as const))('%s', (_, path) => {
        expect(copyOffendersIn(readFileSync(path, 'utf8'))).toEqual([]);
    });
});

describe('the guard itself', () => {
    // A guard that never fires is worthless: these pin the rules so a change in
    // the matching code cannot silently disable them.
    it('flags copy asserted through an exact matcher', () => {
        const source = `expect(errs.server_name).toBe('${en.form_error_server_name}');`;

        expect(copyOffendersIn(source)).toEqual([
            expect.stringContaining('form_error_server_name'),
        ]);
    });

    it('flags copy asserted through a text query', () => {
        const source = `screen.getByText('${en.show_more}');`;

        expect(copyOffendersIn(source)).toEqual([expect.stringContaining('show_more')]);
    });

    it('flags copy used as an accessible name', () => {
        const source = `screen.getByRole('button', { name: '${en.save}' });`;

        expect(copyOffendersIn(source)).toEqual([expect.stringContaining("'save'")]);
    });

    it('flags copy with its placeholders filled in', () => {
        const source = `expect(m?.message).toBe('${en.tls_setup_error_duplicate_port.replace('%port%', '9000')}');`;

        expect(copyOffendersIn(source)).toEqual([
            expect.stringContaining('tls_setup_error_duplicate_port'),
        ]);
    });

    it('accepts keys, fixtures, and plain data', () => {
        const source = [
            `expect(errs.server_name).toBe(copy('form_error_server_name'));`,
            `screen.getByText(copyInDom('show_more'));`,
            `expect(screen.getByText('a.org')).toBeInTheDocument();`,
            `const client = { name: 'Other', ids: ['192.168.1.50'] };`,
            `expect(normalizeMac(input)).toBe('AA:BB:CC:DD:EE:FF');`,
        ].join('\n');

        expect(copyOffendersIn(source)).toEqual([]);
    });
});
