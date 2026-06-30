import intl from 'panel/common/intl';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import {
    validateServerName,
    validatePort,
    validatePortTLS,
    validatePortQuic,
    validateIsSafePort,
    validatePlainDns,
    validateRequiredValue,
    validatePath,
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

/**
 * Runs all client-side validation for the encryption form and returns a map of
 * field name -> error message. An empty object means the form is valid.
 *
 * Used by both `handleBlur` (to gate the debounced backend validate request)
 * and `onFormSubmit`, so the rules live in one place.
 */
export const validateEncryptionForm = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};

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

    const portDotErr = validatePortTLS(portDot);
    if (portDotErr) errs.port_dns_over_tls = portDotErr as string;

    const portDoqErr = validatePortQuic(portDoq);
    if (portDoqErr) errs.port_dns_over_quic = portDoqErr as string;

    if (portDot && portHttps && portDot === portHttps) {
        errs.port_dns_over_tls = intl.getMessage('form_error_equal');
        errs.port_https = intl.getMessage('form_error_equal');
    }

    // Plain DNS must be served when encryption is disabled.
    const plainDnsErr = validatePlainDns(values.serve_plain_dns ?? false, values);
    if (plainDnsErr) errs.serve_plain_dns = plainDnsErr as string;

    // When encryption is enabled, a certificate and a private key are required
    // before the backend validate request is meaningful.
    if (values.enabled) {
        if (values.certificate_source === ENCRYPTION_SOURCE.CONTENT) {
            const certErr = validateRequiredValue(values.certificate_chain);
            if (certErr) errs.certificate_chain = certErr;
        } else {
            const certPathErr =
                validateRequiredValue(values.certificate_path) ||
                validatePath(values.certificate_path);
            if (certPathErr) errs.certificate_path = certPathErr;
        }

        if (!values.private_key_saved) {
            if (values.key_source === ENCRYPTION_SOURCE.CONTENT) {
                const keyErr = validateRequiredValue(values.private_key);
                if (keyErr) errs.private_key = keyErr;
            } else {
                const keyPathErr =
                    validateRequiredValue(values.private_key_path) ||
                    validatePath(values.private_key_path);
                if (keyPathErr) errs.private_key_path = keyPathErr;
            }
        }
    }

    return errs;
};
