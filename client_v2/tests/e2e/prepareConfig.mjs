import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { SEEDED_ALLOW_FILTERS, SEEDED_FILTERS } from './seededFilters.mjs';

const requiredEnv = [
    'E2E_CONFIG_PATH',
    'E2E_WORK_DIR',
    'E2E_ADMIN_USERNAME',
    'E2E_ADMIN_PASSWORD_HASH',
    'E2E_HTTP_PORT',
    'E2E_DNS_PORT',
    'E2E_SCHEMA_VERSION',
];

for (const name of requiredEnv) {
    if (!process.env[name]) {
        throw new Error(`Missing required environment variable: ${name}`);
    }
}

const quoteYaml = (value) => `'${value.replaceAll("'", "''")}'`;

rmSync(process.env.E2E_WORK_DIR, { force: true, recursive: true });
mkdirSync(process.env.E2E_WORK_DIR, { mode: 0o700, recursive: true });

const filtersDir = join(process.env.E2E_WORK_DIR, 'data', 'filters');
mkdirSync(filtersDir, { mode: 0o700, recursive: true });

const filterYaml = ({ id, name, enabled }) =>
    [
        `  - enabled: ${enabled}`,
        `    url: ${quoteYaml(`https://filters.invalid/${id}.txt`)}`,
        `    name: ${quoteYaml(name)}`,
        `    id: ${id}`,
        `    last_updated: ${quoteYaml('2100-01-01T00:00:00Z')}`,
    ].join('\n');

[...SEEDED_FILTERS, ...SEEDED_ALLOW_FILTERS].forEach(({ id, name, rule }) => {
    writeFileSync(join(filtersDir, `${id}.txt`), `! Title: ${name}\n${rule}\n`, {
        encoding: 'utf8',
        mode: 0o600,
    });
});

const configBody = [
    'http:',
    `  address: ${quoteYaml(`127.0.0.1:${process.env.E2E_HTTP_PORT}`)}`,
    'users:',
    `  - name: ${quoteYaml(process.env.E2E_ADMIN_USERNAME)}`,
    `    password: ${quoteYaml(process.env.E2E_ADMIN_PASSWORD_HASH)}`,
    'dns:',
    '  bind_hosts:',
    `    - ${quoteYaml('127.0.0.1')}`,
    `  port: ${process.env.E2E_DNS_PORT}`,
    '  upstream_dns:',
    `    - ${quoteYaml('8.8.8.8')}`,
    'filters:',
    ...SEEDED_FILTERS.map(filterYaml),
    'whitelist_filters:',
    ...SEEDED_ALLOW_FILTERS.map(filterYaml),
    `schema_version: ${process.env.E2E_SCHEMA_VERSION}`,
    '',
].join('\n');

writeFileSync(process.env.E2E_CONFIG_PATH, configBody, {
    encoding: 'utf8',
    mode: 0o600,
});
