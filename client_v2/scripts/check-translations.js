// CLI entry point for auditing client_v2 translation usage against the base
// locales: src/__locales/en.json (UI strings) and
// src/__locales-services/en.json (Hostlists Registry blocked-service groups).
// It reports missing locale keys, unused locale keys, and dynamic intl usages that make the audit best-effort.
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import {
    auditTranslations,
    collectTranslationUsageFromFiles,
    formatAuditReport,
    formatPluralReport,
    listSourceFiles,
    loadLocaleMessages,
    loadTwoskyLocales,
    validatePlurals,
} from './translation-audit.js';

// Mirrors flattenServices in src/common/intl/index.ts.  This script runs on
// plain Node and cannot import the TypeScript module.
const flattenServices = (services) =>
    Object.fromEntries(
        Object.entries(services).map(([key, value]) => [
            key,
            typeof value === 'string' ? value : value.message,
        ]),
    );

export const runTranslationAudit = async ({
    rootDir = process.cwd(),
    write = (chunk) => {
        process.stdout.write(chunk);
    },
    writeError = (chunk) => {
        process.stderr.write(chunk);
    },
} = {}) => {
    try {
        const srcDir = path.join(rootDir, 'src');
        const localePath = path.join(srcDir, '__locales', 'en.json');
        const servicesLocalePath = path.join(srcDir, '__locales-services', 'en.json');
        const filePaths = await listSourceFiles(srcDir);
        const servicesMessages = flattenServices(await loadLocaleMessages(servicesLocalePath));
        const localeMessages = {
            ...(await loadLocaleMessages(localePath)),
            ...servicesMessages,
        };
        const usage = await collectTranslationUsageFromFiles(filePaths);

        const report = auditTranslations({ localeMessages, usage });

        write(`${formatAuditReport(report, { rootDir })}\n\n`);

        const repoRoot = path.resolve(rootDir, '..');
        const localesDir = path.join(srcDir, '__locales');
        const supportedLocales = await loadTwoskyLocales(repoRoot);
        const pluralKeys = new Set(
            usage.staticKeys
                .filter((entry) => entry.method === 'getPlural')
                .map((entry) => entry.key),
        );
        const pluralErrors = await validatePlurals(localesDir, supportedLocales, pluralKeys);
        write(`${formatPluralReport(pluralErrors)}\n`);

        return 0;
    } catch (error) {
        const message = error instanceof Error ? error.message : String(error);

        writeError(`Translation audit failed: ${message}\n`);

        return 1;
    }
};

const currentFilePath = fileURLToPath(import.meta.url);

if (process.argv[1] && path.resolve(process.argv[1]) === currentFilePath) {
    const exitCode = await runTranslationAudit();

    process.exitCode = exitCode;
}
