import intl from 'panel/common/intl';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import {
    validateServerName,
    validatePort,
    validateIsSafePort,
    validatePlainDns,
    validateRequiredValue,
    validatePath,
    validatePemContent,
} from 'panel/helpers/validators';

export type EncryptionFormValues = {
    enabled?: boolean;
    serve_plain_dns?: boolean;
    server_name?: string;
    force_https?: boolean;
    port_https?: number;
    port_dns_over_tls?: number;
    port_dns_over_quic?: number;
    certificate_chain?: string;
    private_key?: string;
    certificate_path?: string;
    private_key_path?: string;
    certificate_source?: string;
    key_source?: string;
    private_key_saved?: boolean;
};

const validateCertPath = (value?: string): string | undefined => {
    if (value && validatePath(value)) {
        return intl.getMessage('encryption_unable_read_cert');
    }
    return undefined;
};

const validateKeyPath = (value?: string): string | undefined => {
    if (value && validatePath(value)) {
        return intl.getMessage('encryption_unable_read_key');
    }
    return undefined;
};

/**
 * Validates only the certificate fields (chain or path).
 * Used by step 1 of the Add TLS Certificate modal.
 */
export const validateCertFields = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};

    if (values.certificate_source === ENCRYPTION_SOURCE.CONTENT) {
        const certErr =
            validateRequiredValue(values.certificate_chain) ||
            validatePemContent(values.certificate_chain);
        if (certErr) errs.certificate_chain = certErr;
    } else {
        const certPathErr =
            validateRequiredValue(values.certificate_path) ||
            validateCertPath(values.certificate_path);
        if (certPathErr) errs.certificate_path = certPathErr;
    }

    return errs;
};

/**
 * Validates only the private key fields (key or path).
 * Used by step 2 of the Add TLS Certificate modal.
 */
export const validateKeyFields = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};

    if (values.private_key_saved) return errs;

    if (values.key_source === ENCRYPTION_SOURCE.CONTENT) {
        const keyErr =
            validateRequiredValue(values.private_key) || validatePemContent(values.private_key);
        if (keyErr) errs.private_key = keyErr;
    } else if (values.key_source === ENCRYPTION_SOURCE.PATH) {
        const keyPathErr =
            validateRequiredValue(values.private_key_path) ||
            validateKeyPath(values.private_key_path);
        if (keyPathErr) errs.private_key_path = keyPathErr;
    }

    return errs;
};

/**
 * Validates certificate and private key fields together.
 */
export const validateCertKeyFields = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};
    Object.assign(errs, validateCertFields(values), validateKeyFields(values));
    return errs;
};

/**
 * Runs all client-side validation for the encryption form and returns a map of
 * field name -> error message. An empty object means the form is valid.
 *
 * Used by both `handleBlur` (to gate the debounced backend validate request)
 * and `onFormSubmit`, so the rules live in one place.
 */
export const validateEncryptionForm = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};

    // Delegate cert/key validation to the shared helper.
    Object.assign(errs, validateCertKeyFields(values));

    // Server name — optional, format only.
    const serverNameErr = validateServerName(values.server_name);
    if (serverNameErr) errs.server_name = serverNameErr;

    // Ports — range, unsafe, and equality are checked fully client-side so
    // invalid ports never reach the backend validate request.
    // Coerce to number first: the store may hold empty strings before the
    // first data load, and `validatePort('')` would skip the range check.
    const portHttps = Number(values.port_https) || 0;
    const portDot = Number(values.port_dns_over_tls) || 0;
    const portDoq = Number(values.port_dns_over_quic) || 0;

    const portHttpsErr = validatePort(portHttps) || validateIsSafePort(portHttps);
    if (portHttpsErr) errs.port_https = portHttpsErr as string;

    const portDotErr = validatePort(portDot);
    if (portDotErr) errs.port_dns_over_tls = portDotErr as string;

    const portDoqErr = validatePort(portDoq);
    if (portDoqErr) errs.port_dns_over_quic = portDoqErr as string;

    if (portDot && portHttps && portDot === portHttps) {
        errs.port_dns_over_tls = intl.getMessage('form_error_equal');
        errs.port_https = intl.getMessage('form_error_equal');
    }

    // Plain DNS must be served when encryption is disabled.
    const plainDnsErr = validatePlainDns(values.serve_plain_dns ?? false, values);
    if (plainDnsErr) errs.serve_plain_dns = plainDnsErr as string;

    return errs;
};
