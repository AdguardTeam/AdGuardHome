import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    STANDARD_HTTPS_PORT,
    DEBOUNCE_TIMEOUT,
    ENCRYPTION_SOURCE,
} from 'panel/helpers/constants';
import { validateTlsConfig } from 'panel/stores/encryption';
import type { EncryptionFormValues } from '../validate';

/** Wire format sent to the backend — EncryptionFormValues without the source tri-state fields. */
export type TlsSubmitValues = Omit<EncryptionFormValues, 'certificate_source' | 'key_source'>;

/**
 * Default TLS form values — used for initial state and "Reset DNS protocols".
 * Mirrors `defaultValues` from the original Form.tsx.
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
 * IMPORTANT: Keep in sync with the backend contract. Matches the original
 * getSubmitValues in index.tsx exactly.
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
    } else if (key_source === ENCRYPTION_SOURCE.SAVED) {
        config.private_key_path = '';
        config.private_key = '';
        config.private_key_saved = private_key_saved;
    } else {
        config.private_key_path = '';
        if (private_key_saved) {
            config.private_key = '';
            config.private_key_saved = private_key_saved;
        }
    }
    return config;
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
            const submitValues = getSubmitValues(values);
            if (submitValues.enabled) {
                validateTlsConfig(submitValues);
            }
        }, DEBOUNCE_TIMEOUT);
    };

    const cancel = () => {
        if (timer) clearTimeout(timer);
        timer = null;
    };

    return [validate, cancel];
};
