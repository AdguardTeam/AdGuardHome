import intl from 'panel/common/intl';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import type { EncryptionFormValues } from '../../validate';
import type { ValidationResult, WizardStep } from '../helpers';

export type MessageKind = 'error' | 'warning';

/**
 * A validation result bound to a wizard step.  The `field` decides where the
 * message belongs: the wizard takes the user to the step that owns it, so a
 * certificate problem is always fixed where the certificate is entered.
 */
export type StepMessage = {
    /** Field to render under; `undefined` means the message is form-level. */
    field?: keyof EncryptionFormValues;
    kind: MessageKind;
    message: string;
};

/** Fields rendered on each step, so messages can be attached to a visible field. */
export const STEP_FIELDS: Record<WizardStep, ReadonlySet<string>> = {
    1: new Set(['certificate_chain', 'certificate_path']),
    2: new Set(['private_key', 'private_key_path']),
    3: new Set(['server_name', 'port_https', 'port_dns_over_tls', 'port_dns_over_quic']),
};

type CertField = 'certificate_chain' | 'certificate_path';
type KeyField = 'private_key' | 'private_key_path';

const msg = (key: string, params?: Record<string, string>) => intl.getMessage(key, params);

const certField = (v: EncryptionFormValues): CertField =>
    v.certificate_source === ENCRYPTION_SOURCE.PATH ? 'certificate_path' : 'certificate_chain';

const keyFieldOf = (v: EncryptionFormValues): KeyField =>
    v.key_source === ENCRYPTION_SOURCE.PATH ? 'private_key_path' : 'private_key';

/**
 * Certificate complaints that mean the same thing on every step: the backend
 * could not read or parse the data at all, so the config cannot be saved.
 */
const certTextMessage = (text: string, field: CertField): StepMessage | undefined => {
    if (text.includes('reading cert file')) {
        return { field, kind: 'error', message: msg('tls_setup_error_read_cert') };
    }
    if (text.includes('empty certificate')) {
        return { field, kind: 'error', message: msg('tls_setup_error_not_a_cert') };
    }
    if (text.includes('parsing certificate at index')) {
        return { field, kind: 'error', message: msg('tls_setup_error_parse_cert') };
    }

    return undefined;
};

/**
 * True when the certificate in the response does not cover the current moment:
 * expired, or not valid yet.  Read from the certificate dates rather than the
 * verify wording, which differs per platform — and the backend words both cases
 * the same way, so one flag is enough.
 */
const certDatesInvalid = (notAfter?: string, notBefore?: string): boolean => {
    const now = Date.now();
    const after = Date.parse(notAfter ?? '');
    const before = Date.parse(notBefore ?? '');

    return (!Number.isNaN(after) && now > after) || (!Number.isNaN(before) && now < before);
};

/** Maps `warning_validation` / 400 text about the certificate. */
const mapCertText = (
    text: string,
    field: CertField,
    certValid: boolean,
    datesInvalid: boolean,
): StepMessage => {
    const common = certTextMessage(text, field);
    if (common) return common;

    if (text.includes('certificate does not verify')) {
        // A validity window that excludes now is worth naming: the generic
        // untrusted wording would send the user looking for a CA problem.
        if (datesInvalid) {
            return { field, kind: 'warning', message: msg('tls_setup_warning_cert_expired') };
        }

        // Non-critical for the backend: the chain is simply not trusted, or a
        // certificate in it is expired.  It warns on every step.
        return { field, kind: 'warning', message: msg('tls_setup_warning_cert_untrusted') };
    }
    if (text.includes('certificates has no IP addresses')) {
        return { field, kind: 'warning', message: msg('tls_setup_warning_no_ip') };
    }

    // A certificate that parsed only ever warns; an unparsed one blocks, since
    // the save would be refused as well.
    if (certValid) {
        return {
            field,
            kind: 'warning',
            message: text || msg('tls_setup_warning_cert_untrusted'),
        };
    }

    return { field, kind: 'error', message: msg('tls_setup_error_parse_cert') };
};

/** Maps `warning_validation` / 400 text about the private key and the pair. */
const mapKeyText = (text: string, field: KeyField): StepMessage => {
    if (text.includes('reading key file')) {
        return { field, kind: 'error', message: msg('tls_setup_error_read_key') };
    }
    if (text.includes('no valid keys were found')) {
        return { field, kind: 'error', message: msg('tls_setup_error_not_a_key') };
    }
    if (text.includes('parsing private key')) {
        return { field, kind: 'error', message: msg('tls_setup_error_parse_key') };
    }
    if (text.includes('ED25519 keys are not supported')) {
        return { field, kind: 'error', message: msg('tls_setup_error_ed25519_key') };
    }
    if (text.includes('certificate-key pair')) {
        return { field, kind: 'error', message: msg('tls_setup_error_key_mismatch') };
    }

    // Defensive: an empty backend text still blocks, with a generic message.
    return { field, kind: 'error', message: text || msg('tls_setup_error_parse_key') };
};

const PROTO_FIELDS: Record<string, keyof EncryptionFormValues> = {
    HTTPS: 'port_https',
    'DNS-over-TLS': 'port_dns_over_tls',
    'DNS-over-QUIC': 'port_dns_over_quic',
};

/** Maps plain-text 400 bodies and config-step warnings. */
const mapConfigText = (
    text: string,
    values: EncryptionFormValues,
): StepMessage | undefined => {
    const busy = text.match(/port (\d+) for (HTTPS|DNS-over-TLS|DNS-over-QUIC) is not available/);
    if (busy) {
        return {
            field: PROTO_FIELDS[busy[2]],
            kind: 'error',
            message: msg('tls_setup_error_port_busy', { port: busy[1], protocol: busy[2] }),
        };
    }

    const dup = text.match(/duplicated values: \[(\d+)/);
    if (dup) {
        // No field: the conflicting port is edited right here on step 3
        // (client-side `validatePortConflicts` normally catches this first; the
        // backend duplicate report also covers external settings).
        return {
            kind: 'error',
            message: msg('tls_setup_error_duplicate_port', { port: dup[1] }),
        };
    }

    // The same decisive certificate failures the certificate step reports, so
    // the wizard can take the user back there with the message attached.
    const certError = certTextMessage(text, certField(values));
    if (certError) return certError;

    if (text.includes('certificate does not verify')) {
        if (text.includes('certificate is valid for')) {
            // Not critical: the certificate and key pair are valid and
            // encrypted DNS works, the certificate is simply not valid for the
            // name the user typed.  The backend accepts such a config, so this
            // must not block Enable — it warns, like an untrusted chain.
            return {
                field: 'server_name',
                kind: 'warning',
                message: msg('tls_setup_warning_server_name_mismatch', {
                    hostname: String(values.server_name ?? ''),
                }),
            };
        }

        // Untrusted — `signed by unknown authority` on Linux, `certificate is
        // not trusted` on macOS — and every other verify complaint, an expired
        // certificate included: the backend treats all of them as
        // non-critical, and so does the certificate step.  Warn, never block.
        return { kind: 'warning', message: msg('tls_setup_warning_cert_untrusted') };
    }
    if (text.includes('certificates has no IP addresses')) {
        return { kind: 'warning', message: msg('tls_setup_warning_no_ip') };
    }
    if (text.includes('certificate-key pair') || text.includes('parsing private key')) {
        // The pair is checked on the key step, which is where the key lives.
        return {
            field: keyFieldOf(values),
            kind: 'error',
            message: msg('tls_setup_error_key_mismatch'),
        };
    }

    // Unclassified: the text is not a verdict we can turn into copy.  Stay
    // silent and let the save be the authority — a refused save reports
    // `tls_setup_error_enable_failed` on this step.
    return undefined;
};

/**
 * Turns a per-step backend result into the single inline message to show, or
 * undefined when the step is clean.  `kind === 'error'` blocks advancing.
 */
export const mapStepResult = (
    step: WizardStep,
    res: ValidationResult,
    values: EncryptionFormValues,
): StepMessage | undefined => {
    const cert = certField(values);
    const key = keyFieldOf(values);

    if ('error' in res) {
        const text = res.error;

        // The check itself failed — no certificate or key verdict was produced.
        // Steps 1 and 2 have no save of their own, so they stay clean and the
        // configuration save decides, reporting `tls_setup_error_enable_failed`
        // if the backend refuses.
        if (step !== 3) return undefined;

        return mapConfigText(text, values);
    }

    const text = res.warning_validation ?? '';
    // Only a certificate that parsed carries dates, so the flag stays false on
    // the responses that never produced a certificate verdict.
    const datesInvalid = certDatesInvalid(res.not_after, res.not_before);

    if (step === 1) {
        if (res.valid_cert && !text) return undefined;

        return mapCertText(text, cert, !!res.valid_cert, datesInvalid);
    }

    if (step === 2) {
        if (!res.valid_cert) return mapCertText(text, cert, false, datesInvalid);

        if (!res.valid_key) return mapKeyText(text, key);

        if (!res.valid_pair) {
            return text
                ? mapKeyText(text, key)
                : { field: key, kind: 'error', message: msg('tls_setup_error_key_mismatch') };
        }

        if (text) return mapCertText(text, cert, true, datesInvalid);

        return undefined;
    }

    const pairValid = !!(res.valid_cert && res.valid_key && res.valid_pair);
    if (!pairValid && !text) {
        // Defensive: the backend flagged a failure without a reason.  Point at
        // the half that failed, so the message lands where it is fixed.
        if (!res.valid_cert) {
            return {
                field: cert,
                kind: 'error',
                message: msg('tls_setup_error_parse_cert'),
            };
        }

        if (!res.valid_key) {
            return {
                field: key,
                kind: 'error',
                message: msg('tls_setup_error_parse_key'),
            };
        }

        return {
            field: key,
            kind: 'error',
            message: msg('tls_setup_error_key_mismatch'),
        };
    }

    return text ? mapConfigText(text, values) : undefined;
};
