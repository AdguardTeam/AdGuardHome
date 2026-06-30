import fs from 'node:fs/promises';
import path from 'node:path';

import jscodeshift from 'jscodeshift';

const j = jscodeshift.withParser('tsx');

const SOURCE_EXTENSIONS = new Set(['.ts', '.tsx']);
const TRANSLATION_METHODS = new Set(['getMessage', 'getPlural']);

const isIntlMethod = (callee) =>
    callee?.type === 'MemberExpression'
    && !callee.computed
    && callee.object?.type === 'Identifier'
    && callee.object.name === 'intl'
    && callee.property?.type === 'Identifier'
    && TRANSLATION_METHODS.has(callee.property.name);

const getLiteralKey = (argument) => {
    if (!argument) {
        return null;
    }

    if (argument.type === 'StringLiteral') {
        return argument.value;
    }

    if (argument.type === 'Literal' && typeof argument.value === 'string') {
        return argument.value;
    }

    if (argument.type === 'TemplateLiteral' && argument.expressions.length === 0) {
        return argument.quasis[0]?.value.cooked ?? '';
    }

    if (argument.type === 'ParenthesizedExpression') {
        return getLiteralKey(argument.expression);
    }

    return null;
};

const sortByFileAndLine = (left, right) =>
    left.filePath.localeCompare(right.filePath)
    || left.line - right.line
    || left.method.localeCompare(right.method);

const toRelativePath = (filePath, rootDir) => {
    if (!rootDir) {
        return filePath;
    }

    const relativePath = path.relative(rootDir, filePath);

    return relativePath === '' ? '.' : relativePath;
};

const formatNodeSource = (node) => j(node).toSource({ quote: 'single' });

export const collectTranslationUsageFromSource = (source, filePath) => {
    const staticKeys = [];
    const dynamicUsages = [];

    j(source)
        .find(j.CallExpression)
        .forEach((callPath) => {
            const { node } = callPath;

            if (!isIntlMethod(node.callee)) {
                return;
            }

            const method = node.callee.property.name;
            const line = node.loc?.start?.line ?? 0;
            const key = getLiteralKey(node.arguments[0]);

            if (key !== null) {
                staticKeys.push({ filePath, key, line, method });

                return;
            }

            dynamicUsages.push({
                expression: formatNodeSource(node),
                filePath,
                line,
                method,
            });
        });

    return {
        dynamicUsages: dynamicUsages.sort(sortByFileAndLine),
        staticKeys: staticKeys.sort(sortByFileAndLine),
    };
};

export const auditTranslations = ({ localeMessages, usage }) => {
    const localeKeys = Object.keys(localeMessages);
    const usedStaticKeySet = new Set(usage.staticKeys.map((entry) => entry.key));
    const missingKeys = [...usedStaticKeySet].filter((key) => !Object.hasOwn(localeMessages, key)).sort();
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
        'client_v3 translation audit',
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
            lines.push(`- ${toRelativePath(entry.filePath, rootDir)}:${entry.line} ${entry.expression}`);
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
