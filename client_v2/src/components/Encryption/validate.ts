import intl from 'panel/common/intl';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import {
    validateServerName,
    validatePort,
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

const validateCertPath = (value?: string): string | undefined => {
    if (value && validatePath(value)) {
        return intl.getMessage('tls_setup_error_read_cert');
    }
    return undefined;
};

const validateKeyPath = (value?: string): string | undefined => {
    if (value && validatePath(value)) {
        return intl.getMessage('tls_setup_error_read_key');
    }
    return undefined;
};

/** Matches any PEM private-key header, e.g. `-----BEGIN RSA PRIVATE KEY-----`. */
const R_PEM_KEY_HEADER = /-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----/;

/** Matches a PEM certificate header. */
const R_PEM_CERT_HEADER = /-----BEGIN CERTIFICATE-----/;

/** Matches a complete PEM certificate block: header, body, and closing line. */
const R_PEM_CERT_BLOCK = /-----BEGIN CERTIFICATE-----[\s\S]*-----END CERTIFICATE-----/;

/** Matches a complete PEM private-key block of any flavour. */
const R_PEM_KEY_BLOCK =
    /-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*-----END [A-Z0-9 ]*PRIVATE KEY-----/;

/**
 * Validates the certificate field content.  Only the cases the backend cannot
 * express are handled here — content that is obviously a private key, and
 * content that is not a complete PEM block (no header, or missing the closing
 * line to a partial copy).  Real parsing is left to the backend so the user
 * gets its precise message.
 */
const validateCertContent = (value?: string): string | undefined => {
    const required = validateRequiredValue(value);
    if (required) return required;

    const text = String(value);
    if (R_PEM_KEY_HEADER.test(text)) return intl.getMessage('tls_setup_error_not_a_cert');
    if (!R_PEM_CERT_BLOCK.test(text)) return intl.getMessage('tls_setup_error_cert_incomplete');

    return undefined;
};

/** Validates the private-key field content, mirroring [validateCertContent]. */
const validateKeyContent = (value?: string): string | undefined => {
    const required = validateRequiredValue(value);
    if (required) return required;

    const text = String(value);
    if (R_PEM_CERT_HEADER.test(text) && !R_PEM_KEY_HEADER.test(text)) {
        return intl.getMessage('tls_setup_error_not_a_key');
    }
    if (!R_PEM_KEY_BLOCK.test(text)) return intl.getMessage('tls_setup_error_key_incomplete');

    return undefined;
};

/**
 * Validates only the certificate fields (chain or path).
 * Used by step 1 of the TLS setup wizard.
 */
export const validateCertFields = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};

    if (values.certificate_source === ENCRYPTION_SOURCE.CONTENT) {
        const certErr = validateCertContent(values.certificate_chain);
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
 * Used by step 2 of the TLS setup wizard.
 */
export const validateKeyFields = (values: EncryptionFormValues): Record<string, string> => {
    const errs: Record<string, string> = {};

    if (values.private_key_saved) return errs;

    if (values.key_source === ENCRYPTION_SOURCE.CONTENT) {
        const keyErr = validateKeyContent(values.private_key);
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

/** Ports of the other AdGuard Home settings, used for conflict detection. */
export type ExternalPorts = {
    /** Web UI (HTTP/HTTPS) port. */
    webUi?: number;
    /** Plain DNS port. */
    plainDns?: number;
    /** DNSCrypt TCP port. */
    dnscrypt?: number;
};

/** Port inputs of the encrypted DNS settings — the fields owned by both hosts. */
export type PortField = 'port_https' | 'port_dns_over_tls' | 'port_dns_over_quic';

export type ServerSettingsField = 'server_name' | PortField;

export type ServerSettingsValues = Pick<EncryptionFormValues, ServerSettingsField>;

/**
 * Client-side rules for a single port input: the valid range plus the
 * browser-unsafe ports, which only apply to the HTTPS port.
 */
export const validatePortField = (field: PortField, value?: number): string | undefined =>
    validatePort(value) || (field === 'port_https' ? validateIsSafePort(value) : undefined);

/**
 * Mirrors the backend `validatePorts` (internal/home/web.go): ports may not
 * repeat within the TCP family {web UI, HTTPS, DoT, DNSCrypt, plain DNS} and
 * the UDP family {plain DNS, DoQ}.  Cross-family equality is allowed, zero
 * ports are skipped.
 *
 * Only the wizard's own port fields (`port_*`) are flagged — external
 * settings are read-only here, so a conflict between them is left to the
 * backend.
 */
export const validatePortConflicts = (
    values: EncryptionFormValues,
    externalPorts?: ExternalPorts,
): Record<string, string> => {
    const errs: Record<string, string> = {};
    const msg = intl.getMessage('tls_setup_error_port_in_use');

    const tcp: Record<string, number> = {
        port_https: Number(values.port_https) || 0,
        port_dns_over_tls: Number(values.port_dns_over_tls) || 0,
        webUi: externalPorts?.webUi || 0,
        plainDns: externalPorts?.plainDns || 0,
        dnscrypt: externalPorts?.dnscrypt || 0,
    };
    const udp: Record<string, number> = {
        port_dns_over_quic: Number(values.port_dns_over_quic) || 0,
        plainDns: externalPorts?.plainDns || 0,
    };

    for (const family of [tcp, udp]) {
        const byPort = new Map<number, string[]>();
        for (const [field, port] of Object.entries(family)) {
            if (!port) continue;
            byPort.set(port, [...(byPort.get(port) ?? []), field]);
        }
        for (const fields of byPort.values()) {
            if (fields.length < 2) continue;
            for (const field of fields.filter((f) => f.startsWith('port_'))) {
                errs[field] = msg;
            }
        }
    }

    return errs;
};

/**
 * Runs the client-side validation for the encrypted DNS server settings — the
 * server name and the three ports.  Shared by the TLS setup wizard's config
 * step and the encrypted DNS server settings modal, so both hosts accept and
 * reject exactly the same values.
 */
export const validateServerSettings = (
    values: ServerSettingsValues,
    externalPorts?: ExternalPorts,
): Record<string, string> => {
    const errs: Record<string, string> = {};

    // Server name — optional, format only.
    const serverNameErr = validateServerName(values.server_name);
    if (serverNameErr) errs.server_name = serverNameErr;

    // Ports — range and unsafe ports are checked client-side so invalid
    // values never reach the backend validate request.  Coerce to number
    // first: the store may hold empty strings before the first data load, and
    // `validatePort('')` would skip the range check.
    const ports: Record<PortField, number> = {
        port_https: Number(values.port_https) || 0,
        port_dns_over_tls: Number(values.port_dns_over_tls) || 0,
        port_dns_over_quic: Number(values.port_dns_over_quic) || 0,
    };

    for (const [field, port] of Object.entries(ports) as [PortField, number][]) {
        const portErr = validatePortField(field, port);
        if (portErr) errs[field] = portErr;
    }

    // Conflicts with the other AdGuard Home settings are mirrored from the
    // backend's port uniqueness rules (see `validatePortConflicts`).
    for (const [field, conflict] of Object.entries(validatePortConflicts(values, externalPorts))) {
        errs[field] = errs[field] ?? conflict;
    }

    return errs;
};

/**
 * Runs all client-side validation for the encryption form and returns a map of
 * field name -> error message. An empty object means the form is valid.
 *
 * Used by the wizard to gate advancing, the enable button and the debounced
 * backend validate request, so the rules live in one place.
 */
export const validateEncryptionForm = (
    values: EncryptionFormValues,
    externalPorts?: ExternalPorts,
): Record<string, string> => {
    const errs: Record<string, string> = {};

    Object.assign(
        errs,
        validateCertKeyFields(values),
        validateServerSettings(values, externalPorts),
    );

    // Plain DNS must be served when encryption is disabled.  Not part of
    // `validateServerSettings`: the server settings dialog cannot change it.
    const plainDnsErr = validatePlainDns(values.serve_plain_dns ?? false, values);
    if (plainDnsErr) errs.serve_plain_dns = plainDnsErr as string;

    return errs;
};
