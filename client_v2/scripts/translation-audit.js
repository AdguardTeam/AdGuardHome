import fs from 'node:fs/promises';
import path from 'node:path';

import { validator } from '@adguard/translate';
import ts from 'typescript';

const SOURCE_EXTENSIONS = new Set(['.ts', '.tsx']);
const TRANSLATION_METHODS = new Set(['getMessage', 'getPlural']);

/**
 * Loads the locale-to-underscore map from .twosky.json in the repo root.
 * Returns a Map where keys are hyphenated locale codes (e.g. "pt-br") and
 * values are @adguard/translate locale codes (e.g. "pt_br"), or `null` for
 * unsupported locales.
 *
 * @param {string} repoRoot - Path to the repository root (contains .twosky.json).
 * @returns {Promise<Map<string, string|null>>}
 */
export const loadTwoskyLocales = async (repoRoot) => {
    const twoskyPath = path.join(repoRoot, '.twosky.json');
    const raw = await fs.readFile(twoskyPath, 'utf8');
    const projects = JSON.parse(raw);

    const homeV2 = projects.find((p) => p.project_id === 'home_v2');
    if (!homeV2) {
        throw new Error('home_v2 project not found in .twosky.json');
    }

    /**
     * Converts a hyphenated twosky locale code to @adguard/translate format.
     * Most become underscore (pt-br → pt_br).  Special cases for locales that
     * the library handles under a parent code, or doesn't handle at all.
     */
    const toTranslateLocale = (code) => {
        // Sinhala → Amharic: identical CLDR plural rules (one: n=0..1, other)
        // This only affects plural-form indexing; actual strings come from si-lk.json
        //
        // TODO(ik): Contribute missing `si` locale to @adguard/translate
        if (code === 'si-lk') {
            return 'am';
        }
        // Hong Kong & Serbian Cyrillic → parent locale
        if (code === 'zh-hk') {
            return 'zh';
        }
        if (code === 'sr-cs') {
            return 'sr';
        }

        return code.replace(/-/g, '_');
    };

    const map = new Map();
    for (const code of Object.keys(homeV2.languages)) {
        map.set(code, toTranslateLocale(code));
    }

    return map;
};

const isIntlMethod = (node) => {
    if (!ts.isPropertyAccessExpression(node)) {
        return false;
    }

    return (
        ts.isIdentifier(node.expression) &&
        node.expression.text === 'intl' &&
        ts.isIdentifier(node.name) &&
        TRANSLATION_METHODS.has(node.name.text)
    );
};

const getLiteralKey = (node) => {
    if (!node) {
        return null;
    }

    if (ts.isStringLiteral(node)) {
        return node.text;
    }

    if (ts.isNoSubstitutionTemplateLiteral(node)) {
        return node.text;
    }

    if (ts.isParenthesizedExpression(node)) {
        return getLiteralKey(node.expression);
    }

    return null;
};

const sortByFileAndLine = (left, right) =>
    left.filePath.localeCompare(right.filePath) ||
    left.line - right.line ||
    left.method.localeCompare(right.method);

const toRelativePath = (filePath, rootDir) => {
    if (!rootDir) {
        return filePath;
    }

    const relativePath = path.relative(rootDir, filePath);

    return relativePath === '' ? '.' : relativePath;
};

const formatNodeSource = (node, sourceFile) => sourceFile.text.slice(node.pos, node.end);

export const collectTranslationUsageFromSource = (source, filePath) => {
    const staticKeys = [];
    const dynamicUsages = [];

    const sourceFile = ts.createSourceFile(filePath, source, ts.ScriptTarget.Latest, true);

    const visit = (node) => {
        if (ts.isCallExpression(node) && isIntlMethod(node.expression)) {
            const method = node.expression.name.text;
            const pos = sourceFile.getLineAndCharacterOfPosition(node.pos);
            const line = pos.line + 1;
            const key = getLiteralKey(node.arguments[0]);

            if (key !== null) {
                staticKeys.push({ filePath, key, line, method });
            } else {
                dynamicUsages.push({
                    expression: formatNodeSource(node, sourceFile),
                    filePath,
                    line,
                    method,
                });
            }
        }

        // Always recurse into children to find nested intl.getMessage() calls
        ts.forEachChild(node, visit);
    };

    ts.forEachChild(sourceFile, visit);

    return {
        dynamicUsages: dynamicUsages.sort(sortByFileAndLine),
        staticKeys: staticKeys.sort(sortByFileAndLine),
    };
};

export const auditTranslations = ({ localeMessages, usage }) => {
    const localeKeys = Object.keys(localeMessages);
    const usedStaticKeySet = new Set(usage.staticKeys.map((entry) => entry.key));
    const missingKeys = [...usedStaticKeySet]
        .filter((key) => !Object.hasOwn(localeMessages, key))
        .sort();
    const unusedKeys = localeKeys.filter((key) => !usedStaticKeySet.has(key)).sort();

    return {
        dynamicUsages: [...usage.dynamicUsages].sort(sortByFileAndLine),
        localeKeyCount: localeKeys.length,
        missingKeys,
        resultsAreIncomplete: usage.dynamicUsages.length > 0,
        staticKeyCount: usage.staticKeys.length,
        unusedKeys,
    };
};

export const formatAuditReport = (report, { rootDir } = {}) => {
    const lines = [
        'client_v2 translation audit',
        `Static translation calls: ${report.staticKeyCount}`,
        `Locale keys in en.json: ${report.localeKeyCount}`,
        `Dynamic translation calls: ${report.dynamicUsages.length}`,
        '',
    ];

    if (report.resultsAreIncomplete) {
        lines.push(
            'WARNING: Dynamic translation usages were found. Missing and unused results are best-effort until those calls are rewritten with string literals.',
            '',
        );
    }

    if (report.missingKeys.length === 0) {
        lines.push('Missing translation keys: none', '');
    } else {
        lines.push('Missing translation keys:');
        report.missingKeys.forEach((key) => {
            lines.push(`- ${key}`);
        });
        lines.push('');
    }

    if (report.unusedKeys.length === 0) {
        lines.push('Unused translation keys: none', '');
    } else {
        lines.push('Unused translation keys:');
        report.unusedKeys.forEach((key) => {
            lines.push(`- ${key}`);
        });
        lines.push('');
    }

    if (report.dynamicUsages.length === 0) {
        lines.push('Dynamic translation usages: none');
    } else {
        lines.push('Dynamic translation usages:');
        report.dynamicUsages.forEach((entry) => {
            lines.push(
                `- ${toRelativePath(entry.filePath, rootDir)}:${entry.line} ${entry.expression}`,
            );
        });
    }

    return lines.join('\n').trimEnd();
};

export const loadLocaleMessages = async (localePath) => {
    const fileContents = await fs.readFile(localePath, 'utf8');

    return JSON.parse(fileContents);
};

export const listSourceFiles = async (directory) => {
    const entries = await fs.readdir(directory, { withFileTypes: true });
    const files = await Promise.all(
        entries.map(async (entry) => {
            const entryPath = path.join(directory, entry.name);

            if (entry.isDirectory()) {
                if (entry.name === '__tests__') {
                    return [];
                }

                return listSourceFiles(entryPath);
            }

            if (!SOURCE_EXTENSIONS.has(path.extname(entry.name)) || entry.name.endsWith('.d.ts')) {
                return [];
            }

            return [entryPath];
        }),
    );

    return files.flat().sort();
};

export const collectTranslationUsageFromFiles = async (filePaths) => {
    const usage = {
        dynamicUsages: [],
        staticKeys: [],
    };

    const fileUsages = await Promise.all(
        filePaths.map(async (filePath) => {
            const source = await fs.readFile(filePath, 'utf8');

            return collectTranslationUsageFromSource(source, filePath);
        }),
    );

    fileUsages.forEach((fileUsage) => {
        usage.staticKeys.push(...fileUsage.staticKeys);
        usage.dynamicUsages.push(...fileUsage.dynamicUsages);
    });

    usage.staticKeys.sort(sortByFileAndLine);
    usage.dynamicUsages.sort(sortByFileAndLine);

    return usage;
};

/**
 * Validates plural forms in all locale files against the @adguard/translate
 * library's plural rules (CLDR-based). Returns an array of errors grouped by
 * locale file and key.
 *
 * @param {string} localesDir - Path to __locales directory
 * @param {{write?: (chunk: string) => void}} [options]
 * @returns {Promise<Array<{file: string, key: string, value: string}>>}
 */
export const validatePlurals = async (localesDir, supportedLocales) => {
    const errors = [];
    const entries = await fs.readdir(localesDir, { withFileTypes: true });

    for (const entry of entries) {
        if (!entry.isFile() || !entry.name.endsWith('.json')) {
            continue;
        }

        const localeName = entry.name.replace(/\.json$/, '');
        const localeCode = supportedLocales.get(localeName);

        // `undefined` means not in twosky list; `null` means unsupported
        if (localeCode === undefined || localeCode === null) {
            continue;
        }

        const localePath = path.join(localesDir, entry.name);
        const messages = await loadLocaleMessages(localePath);

        for (const [key, value] of Object.entries(messages)) {
            // Only validate strings that START with "|" (plural form separator).
            // Strings containing "||" (like filter rule examples) are not plurals.
            if (typeof value !== 'string' || !value.trimStart().startsWith('|')) {
                continue;
            }

            if (!validator.isPluralFormValid(value, localeCode, key)) {
                errors.push({ file: entry.name, key, value });
            }
        }
    }

    return errors;
};

/**
 * Formats the plural validation report.
 *
 * @param {Array<{file: string, key: string, value: string}>} errors
 * @returns {string}
 */
export const formatPluralReport = (errors) => {
    if (errors.length === 0) {
        return 'Plural form validation: all locales passed';
    }

    const byFile = new Map();
    for (const err of errors) {
        const list = byFile.get(err.file) || [];
        list.push(err);
        byFile.set(err.file, list);
    }

    const lines = [`Plural form errors: ${errors.length} keys`];

    for (const [file, errs] of byFile) {
        lines.push(`\n  ${file}:`);
        for (const { key, value } of errs) {
            const pipeCount = (value.match(/\|/g) || []).length;
            lines.push(`    ${key} (${pipeCount} pipes)`);
            lines.push(`      "${value.slice(0, 80)}${value.length > 80 ? '…' : ''}"`);
        }
    }

    return lines.join('\n');
};
