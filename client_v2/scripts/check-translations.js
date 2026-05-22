// CLI entry point for auditing client_v2 translation usage against src/__locales/en.json.
// It reports missing locale keys, unused locale keys, and dynamic intl usages that make the audit best-effort.
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import {
    auditTranslations,
    collectTranslationUsageFromFiles,
    formatAuditReport,
    listSourceFiles,
    loadLocaleMessages,
} from './translation-audit.js';

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
        const filePaths = await listSourceFiles(srcDir);
        const localeMessages = await loadLocaleMessages(localePath);
        const usage = await collectTranslationUsageFromFiles(filePaths);
        const report = auditTranslations({ localeMessages, usage });

        write(`${formatAuditReport(report, { rootDir })}\n`);

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
