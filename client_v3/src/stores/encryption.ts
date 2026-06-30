import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import { getDnsStatus } from './dashboard';
import intl from 'panel/common/intl';

type EncryptionState = {
    processing: boolean;
    processingConfig: boolean;
    processingValidate: boolean;
    enabled: boolean;
    serve_plain_dns: boolean;
    dns_names: any;
    force_https: boolean;
    issuer: string;
    key_type: string;
    not_after: string;
    not_before: string;
    port_dns_over_tls: any;
    port_dns_over_quic: any;
    port_https: any;
    port_dnscrypt: any;
    subject: string;
    valid_chain: boolean;
    valid_key: boolean;
    valid_cert: boolean;
    valid_pair: boolean;
    status_cert: string;
    status_key: string;
    certificate_chain: string;
    private_key: string;
    server_name: string;
    warning_validation: string;
    certificate_path: string;
    private_key_path: string;
    private_key_saved: boolean;
    allow_unencrypted_doh: boolean;
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
    key_type: '',
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

const decodeResponse = (data: any) => {
    const fields = ['certificate_chain', 'private_key', 'server_name'];
    const decoded = { ...data };
    fields.forEach((field) => {
        if (decoded[field]) {
            try {
                decoded[field] = atob(decoded[field]);
            } catch {
                // keep as is
            }
        }
    });
    return decoded;
};

const encodeRequest = (values: any) => {
    const encoded = { ...values };
    if (encoded.certificate_chain) {
        encoded.certificate_chain = btoa(encoded.certificate_chain);
    }
    if (encoded.private_key) {
        encoded.private_key = btoa(encoded.private_key);
    }
    return encoded;
};

export const getTlsStatus = async () => {
    setState('processing', true);
    try {
        const data = await apiClient.getTlsStatus();
        const decoded = decodeResponse(data);
        setState({ ...decoded, processing: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const setTlsConfig = async (values: any) => {
    setState('processingConfig', true);
    try {
        const encoded = encodeRequest(values);
        encoded.port_https = encoded.port_https || 0;
        encoded.port_dns_over_tls = encoded.port_dns_over_tls || 0;
        encoded.port_dns_over_quic = encoded.port_dns_over_quic || 0;

        const data = await apiClient.setTlsConfig(encoded);
        const decoded = decodeResponse(data);

        if (values.enabled && values.force_https && window.location.protocol === 'http:') {
            window.location.reload();
            return;
        }

        setState({ ...decoded, processingConfig: false });
        addSuccessToast(intl.getMessage('encryption_config_saved'));
        // Refresh DNS status after TLS change
        await getDnsStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfig', false);
    }
};

export const validateTlsConfig = async (values: any) => {
    setState('processingValidate', true);
    try {
        const encoded = encodeRequest(values);
        // Normalise empty port strings to 0 before sending to the backend,
        // matching the behaviour in setTlsConfig.
        encoded.port_https = encoded.port_https || 0;
        encoded.port_dns_over_tls = encoded.port_dns_over_tls || 0;
        encoded.port_dns_over_quic = encoded.port_dns_over_quic || 0;
        const data = await apiClient.validateTlsConfig(encoded);
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
        key_type: '',
        not_after: '',
        not_before: '',
        dns_names: null,
    });
};

export const encryptionState = untrack(() => state);
