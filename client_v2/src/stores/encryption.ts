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
    Omit<TlsConfig, 'port_https' | 'port_dns_over_tls' | 'port_dns_over_quic' | 'dns_names'>
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

const decodeResponse = (data: TlsConfig): TlsConfig => {
    const decoded: TlsConfig = { ...data };
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

export const setTlsConfig = async (values: TlsConfigBody, opts?: { silent?: boolean }) => {
    setState('processingConfig', true);
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
            ...Object.fromEntries(
                Object.entries(values).filter(([, v]) => v !== undefined),
            ),
        };

        const encoded = encodeRequest(fullValues);
        encoded.port_https = encoded.port_https || 0;
        encoded.port_dns_over_tls = encoded.port_dns_over_tls || 0;
        encoded.port_dns_over_quic = encoded.port_dns_over_quic || 0;

        const data = await tlsConfigure(encoded);
        const decoded = decodeResponse(data);

        redirectToCurrentProtocol(fullValues, dashboardState.httpPort);

        setState({ ...decoded, processingConfig: false });
        if (!opts?.silent) {
            addSuccessToast(intl.getMessage('settings_notify_changes_saved'));
        }
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfig', false);
    }
};

export const validateTlsConfig = async (values: TlsConfigBody) => {
    setState('processingValidate', true);
    try {
        const encoded = encodeRequest(values);
        // Normalise empty port strings to 0 before sending to the backend,
        // matching the behaviour in setTlsConfig.
        encoded.port_https = encoded.port_https || 0;
        encoded.port_dns_over_tls = encoded.port_dns_over_tls || 0;
        encoded.port_dns_over_quic = encoded.port_dns_over_quic || 0;
        const data = await tlsValidate(encoded);
        const decoded = decodeResponse(data);
        setState({ ...decoded, processingValidate: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingValidate', false);
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
 * Optimistically clears cert and key fields in the local store so
 * consumers reacting to certificate_chain / certificate_path
 * (e.g. certConfigured()) flip synchronously, avoiding a flash of
 * stale validation status while the async delete API call is in flight.
 */
export const clearCertOptimistically = () => {
    setState({
        certificate_chain: '',
        private_key: '',
        certificate_path: '',
        private_key_path: '',
        private_key_saved: false,
    });
};

export const encryptionState = untrack(() => state);
