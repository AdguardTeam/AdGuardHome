import { render, fireEvent } from '@solidjs/testing-library';
import { describe, it, expect, vi, beforeAll } from 'vitest';

import { Form } from 'panel/components/Encryption/Form';
import type { EncryptionFormValues } from 'panel/components/Encryption/Form';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

// jsdom has no matchMedia; hooks like useIsMobile depend on it.
beforeAll(() => {
    if (!window.matchMedia) {
        window.matchMedia = (query: string) =>
            ({
                matches: false,
                media: query,
                onchange: null,
                addEventListener: () => {},
                removeEventListener: () => {},
                addListener: () => {},
                removeListener: () => {},
                dispatchEvent: () => false,
            }) as unknown as MediaQueryList;
    }
});

vi.mock('panel/stores/encryption', () => ({
    setTlsConfig: vi.fn(),
    validateTlsConfig: vi.fn(),
    resetValidationStatus: vi.fn(),
}));

const baseEncryption = {
    processing: false,
    processingConfig: false,
    processingValidate: false,
    enabled: false,
    serve_plain_dns: true,
    dns_names: null,
    force_https: false,
    issuer: '',
    key_type: '',
    not_after: '',
    not_before: '',
    port_dns_over_tls: 853,
    port_dns_over_quic: 853,
    port_https: 443,
    port_dnscrypt: 0,
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
} as any;

const enabledNoCert: EncryptionFormValues = {
    enabled: true,
    serve_plain_dns: true,
    server_name: 'dns.example.com',
    force_https: false,
    port_https: 443,
    port_dns_over_tls: 853,
    port_dns_over_quic: 853,
    certificate_chain: '',
    private_key: '',
    certificate_path: '',
    private_key_path: '',
    certificate_source: ENCRYPTION_SOURCE.CONTENT,
    key_source: ENCRYPTION_SOURCE.CONTENT,
    private_key_saved: false,
};

describe('Encryption Form', () => {
    const getByTestId = (testId: string): HTMLElement => {
        // The Input/Textarea components pass `name` but not `data-testid`
        // to the underlying DOM element. Map test ids to name selectors.
        const el = document.querySelector(`[name="${testId}"]`) as HTMLElement;
        if (!el) throw new Error(`Element not found: [name="${testId}"]`);
        return el;
    };

    it('does NOT fire the validate request on blur when the cert is empty', () => {
        const debounced = vi.fn();
        render(() => (
            <Form
                initialValues={enabledNoCert}
                encryption={baseEncryption}
                onSubmit={vi.fn()}
                debouncedConfigValidation={debounced}
            />
        ));

        const serverInput = getByTestId('server_name');
        fireEvent.change(serverInput, { target: { value: 'dns.example.com' } });
        fireEvent.blur(serverInput);

        expect(debounced).not.toHaveBeenCalled();
    });

    it('fires the validate request on blur once cert + key are filled', () => {
        const debounced = vi.fn();
        render(() => (
            <Form
                initialValues={enabledNoCert}
                encryption={baseEncryption}
                onSubmit={vi.fn()}
                debouncedConfigValidation={debounced}
            />
        ));

        // Fill both cert and key content textareas.
        const certInput = getByTestId('certificate_chain');
        fireEvent.change(certInput, {
            target: { value: '-----BEGIN CERTIFICATE-----\nabc' },
        });

        const keyInput = getByTestId('private_key');
        fireEvent.change(keyInput, {
            target: { value: '-----BEGIN PRIVATE KEY-----\nabc' },
        });

        // Blur on server_name (Input, which forwards onBlur) triggers
        // handleBlur which reads all form values including the now-filled
        // cert and key signals.
        const serverInput = getByTestId('server_name');
        fireEvent.blur(serverInput);

        expect(debounced).toHaveBeenCalledTimes(1);
    });
});
