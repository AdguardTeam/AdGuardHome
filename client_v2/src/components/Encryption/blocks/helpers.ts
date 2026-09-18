import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    STANDARD_HTTPS_PORT,
    DEBOUNCE_TIMEOUT,
    ENCRYPTION_SOURCE,
} from 'panel/helpers/constants';
import { validateTlsConfig, encryptionState } from 'panel/stores/encryption';
import { dashboardState } from 'panel/stores/dashboard';
import type { TlsConfig } from 'panel/api/model/tlsConfig';
import type { EncryptionFormValues, ExternalPorts } from '../validate';

/** Result of a backend TLS validation — decoded config or an error text. */
export type ValidationResult = TlsConfig | { error: string };

/** Wire format sent to the backend — EncryptionFormValues without the source tri-state fields. */
export type TlsSubmitValues = Omit<EncryptionFormValues, 'certificate_source' | 'key_source'>;

/**
 * Ports of the other AdGuard Home settings — the encrypted DNS ports may not
 * collide with them (the same rule the backend enforces).
 */
export const getExternalPorts = (): ExternalPorts => ({
    webUi: dashboardState.httpPort,
    plainDns: dashboardState.dnsPort,
    dnscrypt: Number(encryptionState.port_dnscrypt) || 0,
});

/**
 * Default TLS form values — used for initial state and "Reset DNS protocols".
 */
export const defaultTlsValues: Required<EncryptionFormValues> = {
    enabled: false,
    serve_plain_dns: true,
    server_name: '',
    force_https: false,
    port_https: STANDARD_HTTPS_PORT,
    port_dns_over_tls: DNS_OVER_TLS_PORT,
    port_dns_over_quic: DNS_OVER_QUIC_PORT,
    certificate_chain: '',
    private_key: '',
    certificate_path: '',
    private_key_path: '',
    certificate_source: ENCRYPTION_SOURCE.PATH,
    key_source: ENCRYPTION_SOURCE.PATH,
    private_key_saved: false,
};

/**
 * Strips the source tri-state into the wire format the backend expects.
 * - When source is PATH: empties the content field
 * - When source is CONTENT: empties the path field
 * - When private_key_saved is true: empties private_key (backend uses saved key)
 *
 * `private_key_saved` is always sent as an explicit boolean.  Omitting it
 * would let `setTlsConfig` merge in a stale `true` from the store, which makes
 * the backend replace the submitted key with the stored one — and reject the
 * pair when a key path is set, since key data and a key file cannot coexist.
 *
 * IMPORTANT: Keep in sync with the backend contract.
 */
export const getSubmitValues = (values: EncryptionFormValues): TlsSubmitValues => {
    const { certificate_source, key_source, private_key_saved, ...config } =
        values as EncryptionFormValues & Record<string, unknown>;
    if (certificate_source === ENCRYPTION_SOURCE.PATH) {
        config.certificate_chain = '';
    } else {
        config.certificate_path = '';
    }
    if (key_source === ENCRYPTION_SOURCE.PATH) {
        config.private_key = '';
        config.private_key_saved = false;
    } else {
        config.private_key_path = '';
        if (private_key_saved) {
            config.private_key = '';
        }
        config.private_key_saved = !!private_key_saved;
    }
    return config;
};

/** Steps of the TLS setup wizard. */
export type WizardStep = 1 | 2 | 3;

/**
 * Builds the payload for a per-step validation call.
 *
 * Steps 1 and 2 send `enabled: false` so the backend skips port checks
 * (`validateTLSSettings` returns early) and only reports certificate and key
 * diagnostics.  `server_name` is cleared before step 3 because it belongs to
 * the config step and would otherwise make the backend verify the chain
 * against a name the user has not confirmed yet.
 */
export const getStepValidationValues = (
    step: WizardStep,
    values: EncryptionFormValues,
): TlsSubmitValues => {
    const base: TlsSubmitValues = {
        ...getSubmitValues(values),
        serve_plain_dns: true,
    };

    if (step === 3) return { ...base, enabled: true };

    const payload: TlsSubmitValues = { ...base, enabled: false, server_name: '' };
    if (step === 1) {
        // No key at all on step 1: an empty key is skipped by the backend,
        // and `private_key_saved` must not make it read the stored one.
        payload.private_key = '';
        payload.private_key_path = '';
        payload.private_key_saved = false;
    }

    return payload;
};

/**
 * Creates a debounced TLS config validator.
 * Returns [validate(values), cancel()] tuple.
 * Call cancel() in onCleanup to avoid stale timeouts after unmount.
 */
export const createDebouncedValidator = (): [
    (values: EncryptionFormValues) => void,
    () => void,
] => {
    let timer: ReturnType<typeof setTimeout> | null = null;

    const validate = (values: EncryptionFormValues) => {
        if (timer) clearTimeout(timer);
        timer = setTimeout(() => {
            void validateTlsConfig(getSubmitValues(values));
        }, DEBOUNCE_TIMEOUT);
    };

    const cancel = () => {
        if (timer) clearTimeout(timer);
        timer = null;
    };

    return [validate, cancel];
};
