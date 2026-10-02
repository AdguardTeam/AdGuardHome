import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

const mocks = vi.hoisted(() => ({
    setTlsConfig: vi.fn(),
    validateTlsConfig: vi.fn(),
    tlsStatus: vi.fn(),
    tlsConfigure: vi.fn(),
    tlsValidate: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    redirectToCurrentProtocol: vi.fn(),
    // The modal reads the TLS config from the store on open and validates the
    // edited name against the saved certificate; keep the store out of these
    // tests and drive both through this object.
    encryptionState: {
        server_name: 'dns.example.com',
        port_https: 443,
        port_dns_over_tls: 853,
        port_dns_over_quic: 853,
        port_dnscrypt: 0,
        certificate_chain: 'CERTIFICATE',
        certificate_path: '',
        private_key: 'PRIVATE KEY',
        private_key_path: '',
        private_key_saved: false,
        processingConfig: false,
        processingValidate: false,
    },
}));

vi.mock('panel/api/generated', () => ({
    tlsStatus: mocks.tlsStatus,
    tlsConfigure: mocks.tlsConfigure,
    tlsValidate: mocks.tlsValidate,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));
vi.mock('panel/stores/dashboard', () => ({
    getDnsStatus: vi.fn(),
    // The Web UI port is 80, the plain DNS port 53 — reusing either from the
    // encrypted DNS settings is what `validatePortConflicts` must catch.
    dashboardState: { httpPort: 80, dnsPort: 53 },
}));
vi.mock('panel/helpers/helpers', () => ({
    redirectToCurrentProtocol: mocks.redirectToCurrentProtocol,
}));
// The store mock keeps `validateTlsConfig` local: these tests pin what the
// modal sends and how it renders the verdict, not the store's HTTP plumbing.
vi.mock('panel/stores/encryption', () => ({
    encryptionState: mocks.encryptionState,
    setTlsConfig: mocks.setTlsConfig,
    validateTlsConfig: mocks.validateTlsConfig,
}));

import { ServerSettingsModal } from 'panel/components/Encryption/blocks/ServerSettingsModal';
import { ServerSettingsFields } from 'panel/components/Encryption/blocks/ServerSettingsFields';
import { copy, copyInDom } from 'panel/__tests__/helpers/copy';
import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    STANDARD_HTTPS_PORT,
} from 'panel/helpers/constants';

const renderModal = () => {
    const onClose = vi.fn();
    render(() => <ServerSettingsModal open onClose={onClose} />);
    return onClose;
};

const saveButton = () => screen.getByTestId('config-dialog-save');
const conflictMessage = copy('tls_setup_error_port_in_use');

/** What the backend reports when the certificate does not cover the name. */
const MISMATCH =
    'validating certificate pair: certificate does not verify: x509: certificate is valid for dns.example.com, not localhost';

/** A certificate verdict with no complaint — the saved pair is fine. */
const CERT_OK = { valid_cert: true, valid_key: true, valid_pair: true };

beforeEach(() => {
    vi.clearAllMocks();
    mocks.setTlsConfig.mockResolvedValue({ ok: true });
    mocks.validateTlsConfig.mockResolvedValue(CERT_OK);
    mocks.encryptionState.certificate_chain = 'CERTIFICATE';
    mocks.encryptionState.private_key = 'PRIVATE KEY';
});

describe('ServerSettingsModal — opening', () => {
    it('prefills the current settings from the store', () => {
        renderModal();

        expect(screen.getByText(copyInDom('encrypted_dns_settings'))).toBeInTheDocument();
        expect(screen.getByDisplayValue('dns.example.com')).toBeInTheDocument();
        expect(screen.getByDisplayValue('443')).toBeInTheDocument();
        expect(screen.getAllByDisplayValue('853')).toHaveLength(2);
    });

    it('renders a clear button in every field', () => {
        renderModal();

        // The server name is optional too, so it is cleared the same way the
        // three ports are.
        expect(screen.getAllByTestId('input-clear-button')).toHaveLength(4);
    });
});

describe('ServerSettingsModal — validation', () => {
    it('blocks saving and explains a port conflict with another AGH setting', async () => {
        const user = userEvent.setup();
        renderModal();

        expect(saveButton()).not.toBeDisabled();

        const https = screen.getByDisplayValue('443');
        await user.clear(https);
        await user.type(https, '80');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByText(conflictMessage)).toBeInTheDocument();
        });
        expect(saveButton()).toBeDisabled();
    });

    it('flags an invalid server name and blocks saving until it is fixed', async () => {
        const user = userEvent.setup();
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'not a domain');
        await user.tab();

        expect(screen.getByText(copyInDom('form_error_server_name'))).toBeInTheDocument();
        expect(saveButton()).toBeDisabled();
    });

    it('keeps the malformed server name quiet until blur', async () => {
        const user = userEvent.setup();
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'my server');

        // The format error belongs to blur — a half-typed name must not flash
        // an error mid-word.
        expect(screen.queryByText(copyInDom('form_error_server_name'))).toBeNull();
        expect(saveButton()).not.toBeDisabled();

        await user.tab();
        expect(screen.getByText(copyInDom('form_error_server_name'))).toBeInTheDocument();
        expect(saveButton()).toBeDisabled();
    });

    it('normalizes the server name on blur', async () => {
        const user = userEvent.setup();
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'https://example.com/');
        await user.tab();

        expect((serverName as HTMLInputElement).value).toBe('example.com');
        expect(saveButton()).not.toBeDisabled();
    });

    it('flags an out-of-range port', async () => {
        const user = userEvent.setup();
        renderModal();

        const dot = screen.getAllByDisplayValue('853')[0];
        await user.clear(dot);
        await user.type(dot, '99999');
        await user.tab();

        expect(screen.getByText(copyInDom('form_error_port_range'))).toBeInTheDocument();
        expect(saveButton()).toBeDisabled();
    });
});

describe('ServerSettingsModal — saving', () => {
    it('saves the four settings and closes the dialog', async () => {
        const user = userEvent.setup();
        const onClose = renderModal();

        await user.click(saveButton());

        // The save runs after the certificate check, so it settles a tick later.
        await waitFor(() => {
            expect(mocks.setTlsConfig).toHaveBeenCalledWith({
                server_name: 'dns.example.com',
                port_https: 443,
                port_dns_over_tls: 853,
                port_dns_over_quic: 853,
            });
        });
        expect(onClose).toHaveBeenCalled();
    });

    it('sends edited values and never falls back to a toast for validation errors', async () => {
        const user = userEvent.setup();
        const onClose = renderModal();

        const https = screen.getByDisplayValue('443');
        await user.clear(https);
        await user.type(https, '8443');
        await user.click(saveButton());

        await waitFor(() => {
            expect(mocks.setTlsConfig).toHaveBeenCalledWith({
                server_name: 'dns.example.com',
                port_https: 8443,
                port_dns_over_tls: 853,
                port_dns_over_quic: 853,
            });
        });
        expect(onClose).toHaveBeenCalled();
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });

    it('saves an empty server name after its clear button empties the field', async () => {
        const user = userEvent.setup();
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        const clearButton = within(serverName.parentElement as HTMLElement).getByTestId(
            'input-clear-button',
        );
        await user.click(clearButton);

        // The name is optional: clearing it must reach the form state, not just
        // the DOM, so the empty name is what gets saved.
        expect(serverName).toHaveValue('');
        expect(screen.getAllByDisplayValue('853')).toHaveLength(2);

        await user.click(saveButton());
        await waitFor(() => {
            expect(mocks.setTlsConfig).toHaveBeenCalledWith({
                server_name: '',
                port_https: 443,
                port_dns_over_tls: 853,
                port_dns_over_quic: 853,
            });
        });
    });
});

describe('ServerSettingsModal — the certificate warning', () => {
    const warningText = () =>
        copyInDom('tls_setup_warning_server_name_mismatch', { hostname: 'localhost' });

    it('warns under the server name when the saved certificate does not cover it', async () => {
        const user = userEvent.setup();
        mocks.validateTlsConfig.mockResolvedValue({ ...CERT_OK, warning_validation: MISMATCH });
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'localhost');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByTestId('server-settings-server-name-warning')).toHaveTextContent(
                warningText(),
            );
        });

        // The check asks about the edited name against the saved certificate
        // and key — the wizard's config-step payload.
        expect(mocks.validateTlsConfig).toHaveBeenCalledWith(
            expect.objectContaining({
                enabled: true,
                server_name: 'localhost',
                certificate_chain: 'CERTIFICATE',
                private_key: 'PRIVATE KEY',
            }),
            { persist: false },
        );
    });

    it('reveals a newly discovered warning on the first Save click and saves on the next', async () => {
        const user = userEvent.setup();
        const onClose = renderModal();

        // Clicking Save blurs the field first, so the blur check is still in
        // flight when the click handler runs — the real-world timing.  The
        // click's own check is the one that decides.
        let releaseBlurCheck: (value: unknown) => void = () => {};
        mocks.validateTlsConfig.mockImplementationOnce(
            () =>
                new Promise((resolve) => {
                    releaseBlurCheck = resolve;
                }),
        );
        mocks.validateTlsConfig.mockResolvedValue({ ...CERT_OK, warning_validation: MISMATCH });

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'localhost');

        await user.click(saveButton());

        await waitFor(() => {
            expect(screen.getByTestId('server-settings-server-name-warning')).toHaveTextContent(
                warningText(),
            );
        });
        expect(mocks.setTlsConfig).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
        expect(saveButton()).toHaveTextContent(copyInDom('save_anyway'));

        releaseBlurCheck(CERT_OK);

        // The warning is on screen now, so this click is the confirmation.
        await user.click(saveButton());

        await waitFor(() => {
            expect(mocks.setTlsConfig).toHaveBeenCalledWith({
                server_name: 'localhost',
                port_https: 443,
                port_dns_over_tls: 853,
                port_dns_over_quic: 853,
            });
        });
        expect(onClose).toHaveBeenCalled();
    });

    it('hides the warning as soon as the name is edited', async () => {
        const user = userEvent.setup();
        mocks.validateTlsConfig.mockResolvedValue({ ...CERT_OK, warning_validation: MISMATCH });
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'localhost');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByTestId('server-settings-server-name-warning')).toBeInTheDocument();
        });

        // The warning names the checked host, so it must not outlive the
        // value it was reported for.
        await user.type(serverName, 'x');

        expect(screen.queryByTestId('server-settings-server-name-warning')).toBeNull();
        expect(saveButton()).toHaveTextContent(copyInDom('save'));
    });

    it('does not ask the backend when no certificate is configured', async () => {
        const user = userEvent.setup();
        mocks.encryptionState.certificate_chain = '';
        renderModal();

        const serverName = screen.getByDisplayValue('dns.example.com');
        await user.clear(serverName);
        await user.type(serverName, 'localhost');
        await user.tab();

        // Without a certificate there is nothing to verify the name against.
        expect(mocks.validateTlsConfig).not.toHaveBeenCalled();
        expect(screen.queryByTestId('server-settings-server-name-warning')).toBeNull();
    });
});

describe('ServerSettingsFields — the settings shared with the wizard', () => {
    const values = { server_name: 'example.com', port_https: 443 };

    it('prefixes the field ids when the host asks for it', () => {
        // The wizard prefixes its ids with `tls_setup_` so the fields keep the
        // DOM ids the design/tests rely on.
        render(() => (
            <ServerSettingsFields
                idPrefix="tls_setup_"
                values={values}
                onFieldChange={vi.fn()}
                onFieldBlur={vi.fn()}
            />
        ));

        expect(document.getElementById('tls_setup_server_name')).not.toBeNull();
        expect(document.getElementById('tls_setup_port_https')).not.toBeNull();
        expect(document.getElementById('tls_setup_port_dns_over_tls')).not.toBeNull();
        expect(document.getElementById('tls_setup_port_dns_over_quic')).not.toBeNull();
    });

    it('renders the clear buttons only when the host asks for them', () => {
        render(() => (
            <ServerSettingsFields values={values} onFieldChange={vi.fn()} onFieldBlur={vi.fn()} />
        ));

        expect(screen.queryAllByTestId('input-clear-button')).toHaveLength(0);
    });

    it('forwards field changes and blurs with the field name', async () => {
        const user = userEvent.setup();
        const onFieldChange = vi.fn();
        const onFieldBlur = vi.fn();
        render(() => (
            <ServerSettingsFields
                values={values}
                onFieldChange={onFieldChange}
                onFieldBlur={onFieldBlur}
            />
        ));

        const https = screen.getByDisplayValue('443');
        await user.clear(https);
        await user.type(https, '8443');
        await user.tab();

        expect(onFieldChange).toHaveBeenCalledWith('port_https', '8443');
        expect(onFieldBlur).toHaveBeenCalledWith('port_https');
    });

    it('reports the live server name on every keystroke when the host asks for it', async () => {
        const user = userEvent.setup();
        const onFieldInput = vi.fn();
        render(() => (
            <ServerSettingsFields
                values={values}
                onFieldChange={vi.fn()}
                onFieldBlur={vi.fn()}
                onFieldInput={onFieldInput}
            />
        ));

        // `onFieldChange` only fires on `change` (blur), so the host needs the
        // input stream to keep a warning about the typed name up to date.
        await user.type(screen.getByDisplayValue('example.com'), 'x');

        expect(onFieldInput).toHaveBeenLastCalledWith('server_name', 'example.comx');
    });

    it('fills the default ports in the tooltips from the shared constants', () => {
        // The tooltips are rendered with their text nodes even when closed.
        render(() => (
            <ServerSettingsFields values={values} onFieldChange={vi.fn()} onFieldBlur={vi.fn()} />
        ));

        const labels = [...document.querySelectorAll('label')]
            .map((label) => label.textContent)
            .join(' ');

        expect(labels).toContain(
            copyInDom('encryption_https_tooltip', { port: STANDARD_HTTPS_PORT }),
        );
        expect(labels).toContain(copyInDom('encryption_dot_tooltip', { port: DNS_OVER_TLS_PORT }));
        expect(labels).toContain(copyInDom('encryption_doq_tooltip', { port: DNS_OVER_QUIC_PORT }));
        expect(labels).not.toContain('%port%');
    });
});
