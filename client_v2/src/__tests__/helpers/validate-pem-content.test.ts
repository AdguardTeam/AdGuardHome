import { describe, it, expect, vi } from 'vitest';

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
    },
}));

import { validatePemContent } from 'panel/helpers/validators';

const makePem = (
    label: string,
    body: string,
    opts?: { lineEndings?: string; trailing?: string },
): string => {
    const le = opts?.lineEndings ?? '\n';
    const trailing = opts?.trailing ?? '';
    return `-----BEGIN ${label}-----` + le + body + le + `-----END ${label}-----` + trailing;
};

const CERT_BODY = 'MIIBxTCCAS6gAwIBAgIUNW1eQ0p0q5p0YfGq3Gq3Gq3Gq3EwCgYIKoZIzj0EAwIw';
const KEY_BODY = 'MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKj';

describe('validatePemContent', () => {
    // ── Positive cases ──────────────────────────────────────────────

    it('accepts a standard CERTIFICATE PEM block', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts an RSA PRIVATE KEY PEM block', () => {
        const pem = makePem('RSA PRIVATE KEY', KEY_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts an EC PRIVATE KEY PEM block', () => {
        const pem = makePem('EC PRIVATE KEY', KEY_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts an ENCRYPTED PRIVATE KEY PEM block', () => {
        const pem = makePem('ENCRYPTED PRIVATE KEY', KEY_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts body lines with trailing spaces', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY + '   ');
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts body lines with trailing tabs', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY + '\t\t');
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts blank lines between body rows', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY + '\n\n' + KEY_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts trailing newlines at end of file', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY, { trailing: '\n\n' });
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts trailing spaces and tabs at end of file', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY, { trailing: ' \t ' });
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts CRLF (\\r\\n) line endings', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY, { lineEndings: '\r\n' });
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts undefined input', () => {
        expect(validatePemContent(undefined)).toBeUndefined();
    });

    it('accepts two concatenated CERTIFICATE blocks', () => {
        const pem =
            makePem('CERTIFICATE', CERT_BODY, { trailing: '\n' }) +
            makePem('CERTIFICATE', KEY_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts three concatenated CERTIFICATE blocks', () => {
        const pem =
            makePem('CERTIFICATE', CERT_BODY, { trailing: '\n' }) +
            makePem('CERTIFICATE', KEY_BODY, { trailing: '\n' }) +
            makePem('CERTIFICATE', CERT_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    it('accepts a CERTIFICATE block concatenated with a PRIVATE KEY block', () => {
        const pem =
            makePem('CERTIFICATE', CERT_BODY, { trailing: '\n' }) +
            makePem('RSA PRIVATE KEY', KEY_BODY);
        expect(validatePemContent(pem)).toBeUndefined();
    });

    // ── Negative cases ──────────────────────────────────────────────

    it('rejects a plain non-PEM string', () => {
        expect(validatePemContent('not a pem')).toBe('encryption_invalid_data');
    });

    it('rejects a whitespace-only string', () => {
        expect(validatePemContent('   ')).toBe('encryption_invalid_data');
    });

    it('rejects lowercase labels', () => {
        const pem = makePem('begin certificate', CERT_BODY);
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });

    it('rejects PEM with no body between BEGIN and END', () => {
        const pem = '-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----';
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });

    it('rejects PEM missing the END line', () => {
        const pem = '-----BEGIN CERTIFICATE-----\n' + CERT_BODY;
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });

    it('rejects PEM missing the BEGIN line', () => {
        const pem = CERT_BODY + '\n-----END CERTIFICATE-----';
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });

    it('rejects PEM with no newline after BEGIN label', () => {
        const pem = '-----BEGIN CERTIFICATE-----' + CERT_BODY + '\n-----END CERTIFICATE-----';
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });

    it('rejects valid PEM with junk text before it', () => {
        const pem = 'junk text\n' + makePem('CERTIFICATE', CERT_BODY);
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });

    it('rejects valid PEM with junk text after it', () => {
        const pem = makePem('CERTIFICATE', CERT_BODY) + '\njunk text';
        expect(validatePemContent(pem)).toBe('encryption_invalid_data');
    });
});
