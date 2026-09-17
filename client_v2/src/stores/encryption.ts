import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { tlsStatus, tlsConfigure, tlsValidate } from 'panel/api/generated';
import { addErrorToast, addSuccessToast } from './toasts';
import { dashboardState } from './dashboard';
import { redirectToCurrentProtocol } from '../helpers/helpers';
import intl from 'panel/common/intl';
import type { TlsConfig } from 'panel/api/model/tlsConfig';
import type { TlsConfigBody } from 'panel/api/model/tlsConfigBody';

type EncryptionState = Partial<
    Omit<
        TlsConfig,
        'port_https' | 'port_dns_over_tls' | 'port_dns_over_quic' | 'port_dnscrypt' | 'dns_names'
    >
> & {
    processing: boolean;
    processingConfig: boolean;
    processingValidate: boolean;
    status_cert: string;
    status_key: string;
    allow_unencrypted_doh: boolean;
    // All four port fields: number from API, string from form input (initialized as ''):
    port_https: number | string;
    port_dns_over_tls: number | string;
    port_dns_over_quic: number | string;
    port_dnscrypt: number | string;
    // Store initializes as null, API returns string[]:
    dns_names: string[] | null;
};

const initialState: EncryptionState = {
    processing: true,
    processingConfig: false,
    processingValidate: false,
    enabled: false,
    serve_plain_dns: false,
    dns_names: null,
    force_https: false,
    issuer: '',
    key_type: 'RSA',
    not_after: '',
    not_before: '',
    port_dns_over_tls: '',
    port_dns_over_quic: '',
    port_https: '',
    port_dnscrypt: '',
    subject: '',
    valid_chain: false,
    valid_key: false,
    valid_cert: false,
    valid_pair: false,
    status_cert: '',
    status_key: '',
    certificate_chain: '',
    private_key: '',
    server_name: '',
    warning_validation: '',
    certificate_path: '',
    private_key_path: '',
    private_key_saved: false,
    allow_unencrypted_doh: false,
};

const [state, setState] = createStore<EncryptionState>(initialState);

/**
 * Settings the backend marshals with `omitempty`: a cleared server name, a port
 * that is turned off, and a resolved validation warning with the certificate
 * metadata behind it all come back as missing keys rather than empty values.
 * The store merges every response into its state, so the absence has to be
 * spelled out — otherwise a cleared value would silently keep the previous one
 * until the page is reloaded and the state is built from scratch.
 *
 * Keep in sync with `tlsConfigSettings` in `internal/home/config.go` and with
 * `tlsConfigStatus` in `internal/home/tls.go`.
 */
const OMITTED_WHEN_EMPTY: Pick<
    TlsConfig,
    | 'server_name'
    | 'port_https'
    | 'port_dns_over_tls'
    | 'port_dns_over_quic'
    | 'warning_validation'
    | 'subject'
    | 'issuer'
    | 'key_type'
> = {
    server_name: '',
    port_https: 0,
    port_dns_over_tls: 0,
    port_dns_over_quic: 0,
    warning_validation: '',
    subject: '',
    issuer: '',
    // `key_type` is a union, not a string: the backend omits it for a key it
    // could not read, and the UI reads that absence as "unknown".
    key_type: undefined,
};

/**
 * Turns a TLS config response into store-ready values: the base64 payloads are
 * decoded, and the settings and status fields the backend omits when they are
 * empty are filled in.
 */
const decodeResponse = (data: TlsConfig): TlsConfig => {
    const decoded: TlsConfig = { ...OMITTED_WHEN_EMPTY, ...data };
    const fields = ['certificate_chain', 'private_key'] as const;
    fields.forEach((field) => {
        const value = decoded[field];
        if (typeof value === 'string') {
            try {
                decoded[field] = atob(value);
            } catch {
                // keep as is
            }
        }
    });
    return decoded;
};

const encodeRequest = (values: TlsConfig): TlsConfig => {
    const encoded: TlsConfig = { ...values };
    if (typeof encoded.certificate_chain === 'string') {
        encoded.certificate_chain = btoa(encoded.certificate_chain);
    }
    if (typeof encoded.private_key === 'string') {
        encoded.private_key = btoa(encoded.private_key);
    }
    return encoded;
};

export const getTlsStatus = async () => {
    setState('processing', true);
    try {
        const data = await tlsStatus();
        const decoded = decodeResponse(data);
        setState({ ...decoded, processing: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

/**
 * Saves the TLS configuration.
 *
 * With the default options it updates the store and shows toasts — existing
 * callers are unaffected.  With `suppressErrorToast` the error is returned
 * instead of toasted so callers can render it inline.  A save that turns
 * encryption on is confirmed with its own toast, whichever caller asked for
 * it.
 */
export const setTlsConfig = async (
    values: TlsConfigBody,
    opts?: { silent?: boolean; suppressErrorToast?: boolean },
): Promise<{ ok: true } | { ok: false; error: string }> => {
    setState('processingConfig', true);
    // Read before the save: the response below overwrites `state`, so after it
    // the previous value can no longer tell whether this call enabled anything.
    const wasEnabled = !!state.enabled;
    try {
        // Merge: start with all store values, then override with caller's
        // defined values (empty strings / false are intentional overrides).
        const fullValues: TlsConfig = {
            enabled: state.enabled,
            serve_plain_dns: state.serve_plain_dns,
            server_name: state.server_name,
            force_https: state.force_https,
            port_https: Number(state.port_https) || 0,
            port_dns_over_tls: Number(state.port_dns_over_tls) || 0,
            port_dns_over_quic: Number(state.port_dns_over_quic) || 0,
            certificate_chain: state.certificate_chain,
            private_key: state.private_key,
            certificate_path: state.certificate_path,
            private_key_path: state.private_key_path,
            private_key_saved: state.private_key_saved,
            ...Object.fromEntries(Object.entries(values).filter(([, v]) => v !== undefined)),
        };

        const encoded = encodeRequest(fullValues);
        encoded.port_https = encoded.port_https || 0;
        encoded.port_dns_over_tls = encoded.port_dns_over_tls || 0;
        encoded.port_dns_over_quic = encoded.port_dns_over_quic || 0;

        const data = await tlsConfigure(encoded);
        const decoded = decodeResponse(data);

        redirectToCurrentProtocol(fullValues, dashboardState.httpPort);

        // Turning encryption on is the outcome the user asked for, so it is
        // confirmed by name instead of the generic "changes saved".
        const justEnabled = !wasEnabled && !!fullValues.enabled;

        setState({ ...decoded, processingConfig: false });
        if (!opts?.silent) {
            addSuccessToast(
                justEnabled
                    ? intl.getMessage('encryption_enabled_toast')
                    : intl.getMessage('settings_notify_changes_saved'),
            );
        }

        return { ok: true };
    } catch (error) {
        if (!opts?.suppressErrorToast) {
            addErrorToast({ error });
        }
        setState('processingConfig', false);

        return { ok: false, error: extractBodyText(error) };
    }
};

/**
 * Extracts the body text from a customFetch error.
 *
 * customFetch throws `Error(`${url} | ${body} | ${status}`)` — there is no
 * `.body` property.  The body itself may contain `\n`-joined lines
 * (e.g. `errors.Join` for port probes), so we split on ` | `, rejoin the
 * middle segments and drop the trailing status segment.
 */
const extractBodyText = (error: unknown): string => {
    const message = error instanceof Error ? error.message : String(error);
    const segments = message.split(' | ');
    if (segments.length > 2) {
        return segments.slice(1, -1).join(' | ');
    }
    if (segments.length === 2) {
        return segments[1] ?? message;
    }
    return message;
};

/**
 * Validates TLS settings on the backend.
 *
 * With `persist` (default) it updates the store fields and shows error
 * toasts — existing callers are unaffected. With `persist: false` it is a
 * pure check: it never touches the store or the toasts and returns the
 * decoded `TlsConfig` on success or `{ error }` with the body text.
 */
export const validateTlsConfig = async (
    values: TlsConfigBody,
    opts?: { persist?: boolean },
): Promise<TlsConfig | { error: string }> => {
    const persist = opts?.persist ?? true;

    if (persist) {
        setState('processingValidate', true);
    }

    try {
        const encoded = encodeRequest(values);
        // Normalise empty port strings to 0 before sending to the backend,
        // matching the behaviour in setTlsConfig.
        encoded.port_https = encoded.port_https || 0;
        encoded.port_dns_over_tls = encoded.port_dns_over_tls || 0;
        encoded.port_dns_over_quic = encoded.port_dns_over_quic || 0;
        const data = await tlsValidate(encoded);
        const decoded = decodeResponse(data);
        if (persist) {
            setState({ ...decoded, processingValidate: false });
        }
        return decoded;
    } catch (error) {
        if (persist) {
            addErrorToast({ error });
            setState('processingValidate', false);
        }
        return { error: extractBodyText(error) };
    }
};

export const resetValidationStatus = () => {
    setState({
        warning_validation: '',
        valid_chain: false,
        valid_cert: false,
        valid_key: false,
        valid_pair: false,
        subject: '',
        issuer: '',
        key_type: undefined,
        not_after: '',
        not_before: '',
        dns_names: null,
    });
};

/**
 * Applies TLS values to the store before the backend has confirmed them, so
 * consumers reacting to the config (e.g. `certConfigured()`, the server
 * settings summary) already reflect the change while the save is in flight — a
 * save that rewrites the config restarts the DNS server and can take seconds.
 *
 * This state is transient: [setTlsConfig] overwrites it with the response, so
 * callers must pass the same values they send there.
 */
export const applyTlsOptimistically = (values: TlsConfigBody): void => {
    setState(values);
};

export const encryptionState = untrack(() => state);
