import { mkdirSync, rmSync, writeFileSync } from 'node:fs';

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
    `schema_version: ${process.env.E2E_SCHEMA_VERSION}`,
    '',
].join('\n');

writeFileSync(process.env.E2E_CONFIG_PATH, configBody, {
    encoding: 'utf8',
    mode: 0o600,
});
