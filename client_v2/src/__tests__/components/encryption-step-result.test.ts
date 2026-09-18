import { describe, it, expect } from 'vitest';
import { mapStepResult } from 'panel/components/Encryption/blocks/SetupWizard/mapStepResult';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

/**
 * The copy the mapper reports, read from the same locale file the app loads.
 * Asserting via the keys keeps the tests about *which* message is produced, so
 * rewording it does not break them.
 */
import en from 'panel/__locales/en.json';
import { copy } from 'panel/__tests__/helpers/copy';

/** The message copy the mapper reports, named by key. */
const PARSE_CERT = copy('tls_setup_error_parse_cert');
const PARSE_KEY = copy('tls_setup_error_parse_key');
const READ_CERT = copy('tls_setup_error_read_cert');
const KEY_MISMATCH = copy('tls_setup_error_key_mismatch');
const CERT_UNTRUSTED = copy('tls_setup_warning_cert_untrusted');
const NO_IP = copy('tls_setup_warning_no_ip');

const certValues = {
    certificate_source: ENCRYPTION_SOURCE.CONTENT,
    key_source: ENCRYPTION_SOURCE.CONTENT,
};

const validStatus = {
    valid_chain: true,
    valid_cert: true,
    valid_key: true,
    valid_pair: true,
    subject: 'CN=example.com',
    warning_validation: '',
};

const SELF_SIGNED_WARNING =
    'validating certificate pair: certificate does not verify: x509: certificate signed by unknown authority';

const EXPIRED_WARNING = en.tls_setup_warning_cert_expired;

describe('mapStepResult — step 1 (certificate)', () => {
    it('maps an unparsed certificate to the certificate field', () => {
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_cert: false,
                valid_key: false,
                valid_pair: false,
                warning_validation:
                    'validating certificate pair: parsing certificate at index 0: x509: malformed certificate',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'certificate_chain',
            kind: 'error',
            message: PARSE_CERT,
        });
    });

    it('maps an untrusted chain to a non-blocking warning', () => {
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_key: false,
                valid_pair: false,
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate signed by unknown authority',
            },
            certValues,
        );
        expect(m?.kind).toBe('warning');
        expect(m?.field).toBe('certificate_chain');
        expect(m?.message).toBe(CERT_UNTRUSTED);
    });

    it('maps the missing-IP warning to the certificate field', () => {
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_key: false,
                valid_pair: false,
                warning_validation: 'certificates has no IP addresses',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'certificate_chain',
            kind: 'warning',
            message: NO_IP,
        });
    });

    it('maps an expired certificate to the validity warning', () => {
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_key: false,
                valid_pair: false,
                not_after: '2020-01-01T00:00:00Z',
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'certificate_chain',
            kind: 'warning',
            message: EXPIRED_WARNING,
        });
    });

    it('maps a not-yet-valid certificate to the same validity warning', () => {
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_key: false,
                valid_pair: false,
                not_before: '2099-01-01T00:00:00Z',
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
            },
            certValues,
        );
        expect(m?.kind).toBe('warning');
        expect(m?.message).toBe(EXPIRED_WARNING);
    });

    it('falls back to the untrusted warning when the certificate dates are unusable', () => {
        // No dates in the response, or a malformed one: the verify wording is
        // the only evidence, and it must not be read as an expiry.
        for (const dates of [{}, { not_after: '' }, { not_after: 'not-a-date' }]) {
            const m = mapStepResult(
                1,
                {
                    ...validStatus,
                    valid_key: false,
                    valid_pair: false,
                    ...dates,
                    warning_validation:
                        'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
                },
                certValues,
            );
            expect(m?.message).toBe(CERT_UNTRUSTED);
        }
    });

    it('keeps the untrusted warning for a valid-dated certificate', () => {
        // A chain that is simply not trusted — the dates are fine.
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_key: false,
                valid_pair: false,
                not_before: '2020-01-01T00:00:00Z',
                not_after: '2099-01-01T00:00:00Z',
                warning_validation: SELF_SIGNED_WARNING,
            },
            certValues,
        );
        expect(m?.message).toBe(CERT_UNTRUSTED);
    });

    it('returns nothing for a clean cert-only step 1 response', () => {
        expect(
            mapStepResult(1, { ...validStatus, valid_key: false, valid_pair: false }, certValues),
        ).toBeUndefined();
    });

    it('maps a silent parse failure to the parse message', () => {
        const m = mapStepResult(
            1,
            { ...validStatus, valid_cert: false, warning_validation: '' },
            certValues,
        );
        expect(m).toEqual({
            field: 'certificate_chain',
            kind: 'error',
            message: PARSE_CERT,
        });
    });

    it('attaches messages to the path field when the source is a file path', () => {
        const m = mapStepResult(
            1,
            {
                ...validStatus,
                valid_cert: false,
                warning_validation: 'reading cert file: no such file or directory',
            },
            { ...certValues, certificate_source: ENCRYPTION_SOURCE.PATH },
        );
        expect(m).toEqual({
            field: 'certificate_path',
            kind: 'error',
            message: READ_CERT,
        });
    });

    it('does not turn a failed check into a verdict on the certificate step', () => {
        // The check itself failed, so there is no certificate verdict to show:
        // the step stays clean and the configuration save is the authority.
        expect(mapStepResult(1, { error: 'network is unreachable' }, certValues)).toBeUndefined();
    });
});

describe('mapStepResult — step 2 (private key)', () => {
    it('maps a key/cert pair mismatch to the key field', () => {
        const m = mapStepResult(
            2,
            {
                ...validStatus,
                valid_pair: false,
                warning_validation:
                    'validating certificate pair: certificate-key pair: tls: private key does not match public key',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'private_key',
            kind: 'error',
            message: KEY_MISMATCH,
        });
    });

    it('maps an unparsed key on step 2 to a blocking error under the key field', () => {
        const m = mapStepResult(
            2,
            {
                ...validStatus,
                valid_key: false,
                valid_pair: false,
                warning_validation:
                    'validating certificate pair: parsing private key: tls: failed to find any PEM data in key input',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'private_key',
            kind: 'error',
            message: PARSE_KEY,
        });
    });

    it('maps an unsupported key type to the key field', () => {
        const m = mapStepResult(
            2,
            {
                ...validStatus,
                valid_key: false,
                warning_validation:
                    'validating certificate pair: ED25519 keys are not supported by browsers; did you mean to use X25519 for key exchange?',
            },
            certValues,
        );
        expect(m?.message).toBe(copy('tls_setup_error_ed25519_key'));
    });

    it('maps a pair failure without backend text to the key-mismatch message', () => {
        const m = mapStepResult(2, { ...validStatus, valid_pair: false }, certValues);
        expect(m).toEqual({
            field: 'private_key',
            kind: 'error',
            message: KEY_MISMATCH,
        });
    });

    it('keeps an expired certificate bound to the certificate field on step 2', () => {
        // The key is fine — only the chain fails to verify — so the warning
        // belongs under the certificate field of the previous step.
        const m = mapStepResult(
            2,
            {
                ...validStatus,
                valid_chain: false,
                not_after: '2020-01-01T00:00:00Z',
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
            },
            certValues,
        );
        expect(m).toEqual({ field: 'certificate_chain', kind: 'warning', message: EXPIRED_WARNING });
    });

    it('keeps the certificate error on the certificate field before step 2', () => {
        const m = mapStepResult(
            2,
            {
                ...validStatus,
                valid_cert: false,
                valid_key: false,
                valid_pair: false,
                warning_validation:
                    'validating certificate pair: parsing certificate at index 0: x509: malformed certificate',
            },
            certValues,
        );
        expect(m).toEqual({
            // Bound to the certificate field so the wizard shows it on step 1.
            field: 'certificate_chain',
            kind: 'error',
            message: PARSE_CERT,
        });
    });

    it('returns nothing when the pair is valid and there is no warning', () => {
        expect(mapStepResult(2, validStatus, certValues)).toBeUndefined();
    });
});

describe('mapStepResult — step 3 (config)', () => {
    it('maps a hostname mismatch at the config step to a non-blocking server name warning', () => {
        const m = mapStepResult(
            3,
            {
                ...validStatus,
                valid_chain: false,
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate is valid for example.com, not dns.home.arpa',
            },
            { ...certValues, server_name: 'dns.home.arpa' },
        );
        expect(m).toEqual({
            field: 'server_name',
            kind: 'warning',
            message: copy('tls_setup_warning_server_name_mismatch', {
                hostname: 'dns.home.arpa',
            }),
        });
    });

    it('maps a self-signed chain at the config step to a form-level warning', () => {
        const m = mapStepResult(
            3,
            {
                ...validStatus,
                valid_chain: false,
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate signed by unknown authority',
            },
            certValues,
        );
        expect(m).toEqual({
            kind: 'warning',
            message: CERT_UNTRUSTED,
        });
    });

    it('maps the macOS untrusted-chain wording to the same warning', () => {
        // Go's darwin verifier reports an untrusted chain as `certificate is
        // not trusted` instead of `signed by unknown authority`.
        const m = mapStepResult(
            3,
            {
                ...validStatus,
                valid_chain: false,
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: “agadguard.example.com” certificate is not trusted',
            },
            certValues,
        );
        expect(m).toEqual({
            kind: 'warning',
            message: CERT_UNTRUSTED,
        });
    });

    it('warns instead of blocking on an unrecognized verify complaint', () => {
        // An expired or revoked certificate is non-critical for the backend,
        // exactly like an untrusted chain.  The config step maps it to a
        // form-level warning, which `useStepCheck` never renders here, so the
        // wording does not matter — the certificate step is where it is shown.
        const m = mapStepResult(
            3,
            {
                ...validStatus,
                valid_chain: false,
                not_after: '2020-01-01T00:00:00Z',
                warning_validation:
                    'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
            },
            certValues,
        );
        expect(m).toEqual({
            kind: 'warning',
            message: CERT_UNTRUSTED,
        });
    });

    it('maps a 400 port-busy error to the matching port field', () => {
        const m = mapStepResult(
            3,
            { error: 'port 853 for DNS-over-TLS is not available' },
            certValues,
        );
        expect(m).toEqual({
            field: 'port_dns_over_tls',
            kind: 'error',
            message: copy('tls_setup_error_port_busy', { port: 853, protocol: 'DNS-over-TLS' }),
        });
    });

    it('maps a 400 HTTPS port-busy error to the HTTPS field', () => {
        const m = mapStepResult(3, { error: 'port 443 for HTTPS is not available' }, certValues);
        expect(m).toEqual({
            field: 'port_https',
            kind: 'error',
            message: copy('tls_setup_error_port_busy', { port: 443, protocol: 'HTTPS' }),
        });
    });

    it('flags the fields involved when the backend reports duplicate ports', () => {
        const m = mapStepResult(
            3,
            { error: 'validating tcp ports: duplicated values: [9000]' },
            certValues,
        );
        expect(m?.kind).toBe('error');
        expect(m?.message).toBe(copy('tls_setup_error_duplicate_port', { port: 9000 }));
    });

    it('maps a pair failure at the config step to the key field', () => {
        const m = mapStepResult(
            3,
            {
                ...validStatus,
                valid_pair: false,
                warning_validation:
                    'validating certificate pair: certificate-key pair: tls: private key does not match public key',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'private_key',
            kind: 'error',
            message: KEY_MISMATCH,
        });
    });

    it('points at the certificate when nothing is valid and the backend stays silent', () => {
        const m = mapStepResult(
            3,
            {
                ...validStatus,
                valid_cert: false,
                valid_key: false,
                valid_pair: false,
                warning_validation: '',
            },
            certValues,
        );
        expect(m).toEqual({
            field: 'certificate_chain',
            kind: 'error',
            message: PARSE_CERT,
        });
    });

    it('returns nothing when the config step is clean', () => {
        expect(mapStepResult(3, validStatus, certValues)).toBeUndefined();
    });
});
