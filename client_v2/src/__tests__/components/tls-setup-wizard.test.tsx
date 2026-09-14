import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

const mocks = vi.hoisted(() => ({
    tlsStatus: vi.fn(),
    tlsConfigure: vi.fn(),
    tlsValidate: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    redirectToCurrentProtocol: vi.fn(),
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
    dashboardState: { httpPort: 80, dnsPort: 53 },
}));
vi.mock('panel/helpers/helpers', () => ({
    redirectToCurrentProtocol: mocks.redirectToCurrentProtocol,
}));

import { TlsSetupWizard } from 'panel/components/Encryption/blocks/SetupWizard';
import { getTlsStatus } from 'panel/stores/encryption';

const CERT = '-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----';
const KEY = '-----BEGIN PRIVATE KEY-----\nxyz\n-----END PRIVATE KEY-----';

/** Full pair is valid — the default backend response for steps 1-2. */
const validStatus = {
    valid_chain: true,
    valid_cert: true,
    valid_key: true,
    valid_pair: true,
    subject: 'CN=example.com',
    warning_validation: '',
    certificate_chain: '',
    private_key: '',
};

/** Step 1 sends only the certificate: the backend reports no valid key. */
const CERT_ONLY_STATUS = {
    ...validStatus,
    valid_key: false,
    valid_pair: false,
};

const SELF_SIGNED_WARNING =
    'validating certificate pair: certificate does not verify: x509: certificate signed by unknown authority';

// Every step now hits the backend validate endpoint: default to a clean
// response unless a test overrides it.
beforeEach(() => {
    vi.clearAllMocks();
    mocks.tlsValidate.mockResolvedValue(validStatus);
});

const renderWizard = () => {
    const onClose = vi.fn();
    render(() => <TlsSetupWizard open onClose={onClose} />);
    return onClose;
};

/** Fills the certificate textarea and moves to step 2. */
const goToStep2 = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.type(screen.getByLabelText('Paste the certificate contents'), CERT);
    await user.click(screen.getByTestId('tls-setup-add'));
    await waitFor(() => {
        expect(screen.getByText('Add private key')).toBeInTheDocument();
    });
};

/** Fills cert + key and moves to step 3. */
const goToStep3 = async (user: ReturnType<typeof userEvent.setup>) => {
    await goToStep2(user);
    await user.type(screen.getByLabelText('Paste the key contents'), KEY);
    await user.click(screen.getByTestId('tls-setup-add'));
    await waitFor(() => {
        expect(screen.getByText('Enable encrypted DNS')).toBeInTheDocument();
    });
};

describe('TlsSetupWizard — shell & header', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders step 1 with the title and a 3-segment progress bar', () => {
        renderWizard();
        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();

        const progressbar = screen.getByRole('progressbar');
        expect(progressbar).toHaveAttribute('aria-valuenow', '1');
        // 3 segments: first the steps container, then its 3 pills.
        expect(progressbar.firstElementChild?.children).toHaveLength(3);
    });

    it('hides Go back on step 1 and shows it on steps 2-3', async () => {
        const user = userEvent.setup();
        renderWizard();

        expect(screen.queryByText('Go back')).not.toBeInTheDocument();

        await goToStep2(user);
        expect(screen.getByText('Go back')).toBeInTheDocument();

        await user.type(screen.getByLabelText('Paste the key contents'), KEY);
        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByText('Go back')).toBeInTheDocument();
        expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '3');
    });

    it('Go back returns to the previous step preserving entered values', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);
        await user.click(screen.getByText('Go back'));

        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();
        expect(
            (screen.getByLabelText('Paste the certificate contents') as HTMLTextAreaElement).value,
        ).toBe(CERT);
    });
});

describe('TlsSetupWizard — step 1 (certificate)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('swaps textarea and path input when the source changes', async () => {
        const user = userEvent.setup();
        renderWizard();

        // Default: content — textarea with the design placeholder.
        const textarea = screen.getByLabelText('Paste the certificate contents');
        expect(textarea).toHaveAttribute('placeholder', '-----BEGIN CERTIFICATE-----');

        await user.click(screen.getByText('Path to file on server'));
        expect(
            screen.getByLabelText('Enter full path to the certificate file'),
        ).toBeInTheDocument();
        expect(screen.queryByLabelText('Paste the certificate contents')).toBeNull();

        await user.click(screen.getByText('Certificate as text'));
        expect(screen.getByLabelText('Paste the certificate contents')).toBeInTheDocument();
    });

    it('renders the letsencrypt.org link in the step description', () => {
        renderWizard();

        const link = screen.getByRole('link', { name: 'letsencrypt.org' });
        expect(link).toHaveAttribute('href', 'https://letsencrypt.org/');
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    });

    it('does not advance on empty input and shows an inline error', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();
        expect(screen.queryByText('Add private key')).toBeNull();
    });

    it('does not flag an untouched field when the file picker takes focus', async () => {
        const user = userEvent.setup();
        renderWizard();

        // Focus the textarea, then reach for Browse: the native file dialog
        // takes focus away without any value having been entered.
        await user.click(screen.getByLabelText('Paste the certificate contents'));
        await user.click(screen.getByTestId('tls-setup-cert-dropzone'));

        expect(screen.queryByText('Fill out this field')).toBeNull();
        expect(mocks.tlsValidate).not.toHaveBeenCalled();

        // The required error still belongs to Add.
        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
    });

    it('still reports content errors on blur', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(screen.getByLabelText('Paste the certificate contents'), 'not a pem block');
        await user.tab();

        expect(screen.getByText('Enter the certificate contents with header')).toBeInTheDocument();
    });

    it('advances to step 2 on valid input', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);

        expect(screen.getByText('Add private key')).toBeInTheDocument();
    });

    it('does not block on a check that failed to run, leaving it to the save', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockRejectedValue(
            new Error(
                'http://127.0.0.1/control/tls/validate | Failed to unmarshal TLS config: boom | 400',
            ),
        );
        renderWizard();

        // The request carried no verdict about the certificate, so Add goes
        // through; if the configuration is really bad, the save reports it.
        await goToStep2(user);

        expect(screen.getByText('Add private key')).toBeInTheDocument();
        expect(screen.queryByText(/Certificate has issues/)).toBeNull();
        expect(screen.queryByText(/Unable to parse the certificate/)).toBeNull();
    });

    it('blocks step 1 with the backend parse error inline under the textarea', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue({
            ...CERT_ONLY_STATUS,
            valid_cert: false,
            warning_validation:
                'validating certificate pair: parsing certificate at index 0: x509: malformed certificate',
        });
        renderWizard();

        await user.type(
            screen.getByLabelText('Paste the certificate contents'),
            '-----BEGIN CERTIFICATE-----\nbroken\n-----END CERTIFICATE-----',
        );
        await user.click(screen.getByTestId('tls-setup-add'));

        await waitFor(() => {
            expect(
                screen.getByText('Unable to parse the certificate. The file may be corrupted'),
            ).toBeInTheDocument();
        });
        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument(); // still on step 1
        // An error never turns Add into its warning state.
        expect(screen.getByTestId('tls-setup-add')).toHaveTextContent(/^Add$/);
    });

    it('shows the self-signed warning on the first Add and advances on the second', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue({
            ...CERT_ONLY_STATUS,
            valid_chain: false,
            warning_validation: SELF_SIGNED_WARNING,
        });
        renderWizard();

        await user.type(screen.getByLabelText('Paste the certificate contents'), CERT);

        // Nothing is known to be wrong until the check runs: a plain Add.
        const add = screen.getByTestId('tls-setup-add');
        expect(add).toHaveTextContent(/^Add$/);
        expect(add.className).not.toContain('warning');

        await user.click(add);

        // The warning is surfaced on this step instead of being carried to the
        // next one, and it never blocks — it just has to be seen once.
        await waitFor(() => {
            expect(
                screen.getByText(
                    'This certificate is self-signed — it may not work on all devices. Make sure your devices will accept it',
                ),
            ).toBeInTheDocument();
        });
        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();

        // The button now spells out what going past the warning means.
        expect(add).toHaveTextContent('Add anyway');
        expect(add.className).toContain('warning');

        await user.click(add);
        await waitFor(() => {
            expect(screen.getByText('Add private key')).toBeInTheDocument();
        });
    });

    it('flips Add into the warning state on the blur check, before anything is clicked', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue({
            ...CERT_ONLY_STATUS,
            valid_chain: false,
            warning_validation: SELF_SIGNED_WARNING,
        });
        renderWizard();

        await user.type(screen.getByLabelText('Paste the certificate contents'), CERT);
        await user.tab(); // schedules the debounced per-step check

        const add = screen.getByTestId('tls-setup-add');
        await waitFor(() => {
            expect(add).toHaveTextContent('Add anyway');
        });
        expect(add.className).toContain('warning');
        expect(add).not.toBeDisabled();

        // The warning is already on screen, so a single click goes through.
        await user.click(add);
        await waitFor(() => {
            expect(screen.getByText('Add private key')).toBeInTheDocument();
        });
    });

    it('blocks step 1 when the pasted content is not a certificate', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(screen.getByLabelText('Paste the certificate contents'), KEY);
        await user.click(screen.getByTestId('tls-setup-add'));

        expect(
            screen.getByText(
                'This is not a certificate. Check that you pasted a certificate, not a key',
            ),
        ).toBeInTheDocument();
        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();
        expect(mocks.tlsValidate).not.toHaveBeenCalled();
    });

    it('sends a cert-only, port-free payload on step 1', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(screen.getByLabelText('Paste the certificate contents'), CERT);
        await user.tab();

        await waitFor(() => {
            expect(mocks.tlsValidate).toHaveBeenCalled();
        });
        const calls = mocks.tlsValidate.mock.calls;
        const payload = calls[calls.length - 1][0];
        expect(payload.enabled).toBe(false);
        expect(payload.serve_plain_dns).toBe(true);
        expect(payload.server_name).toBe('');
        expect(payload.private_key).toBe('');
        expect(payload.private_key_path).toBe('');
        expect(atob(payload.certificate_chain)).toBe(CERT);
    });
});

describe('TlsSetupWizard — step 2 (private key)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders two sources and no saved-key option', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);

        expect(screen.getByText('Private key as text')).toBeInTheDocument();
        expect(screen.getByText('Path to file on the server')).toBeInTheDocument();
        expect(screen.queryByText('Use an existing private key')).not.toBeInTheDocument();
    });

    it('gates Add on the key validation', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);
        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
        expect(screen.getByText('Add private key')).toBeInTheDocument();
    });

    it('does not flag an untouched key field when the file picker takes focus', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);
        await user.click(screen.getByLabelText('Paste the key contents'));
        await user.click(screen.getByTestId('tls-setup-key-dropzone'));

        expect(screen.queryByText('Fill out this field')).toBeNull();

        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
    });

    it('blocks step 2 when the key does not match the certificate', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValueOnce(validStatus);
        mocks.tlsValidate.mockResolvedValueOnce({
            ...validStatus,
            valid_pair: false,
            warning_validation:
                'validating certificate pair: certificate-key pair: tls: private key does not match public key',
        });
        renderWizard();

        await goToStep2(user);
        await user.type(screen.getByLabelText('Paste the key contents'), KEY);
        await user.click(screen.getByTestId('tls-setup-add'));

        await waitFor(() => {
            expect(
                screen.getByText(
                    'This private key does not match the certificate from the previous step',
                ),
            ).toBeInTheDocument();
        });
        expect(screen.getByText('Add private key')).toBeInTheDocument(); // stays on step 2
    });
});

describe('TlsSetupWizard — warnings show only where they can be fixed', () => {
    beforeEach(() => vi.clearAllMocks());

    /** A valid pair issued by an unknown authority — the backend's answer to every step. */
    const warnOnSelfSignedCert = () =>
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation: SELF_SIGNED_WARNING,
        });

    /** Cert + Add twice: the first click only surfaces the warning. */
    const goToStep2PastWarning = async (user: ReturnType<typeof userEvent.setup>) => {
        await user.type(screen.getByLabelText('Paste the certificate contents'), CERT);
        await user.click(screen.getByTestId('tls-setup-add'));
        await waitFor(() => {
            expect(screen.getByText(/This certificate is self-signed/)).toBeInTheDocument();
        });
        await user.click(screen.getByTestId('tls-setup-add'));
        await waitFor(() => {
            expect(screen.getByText('Add private key')).toBeInTheDocument();
        });
    };

    it('does not carry the warning into step 2', async () => {
        const user = userEvent.setup();
        warnOnSelfSignedCert();
        renderWizard();

        await goToStep2PastWarning(user);

        expect(screen.queryByText(/This certificate is self-signed/)).toBeNull();
    });

    it('advances from step 2 in one click without repeating the warning', async () => {
        const user = userEvent.setup();
        warnOnSelfSignedCert();
        renderWizard();

        await goToStep2PastWarning(user);
        await user.type(screen.getByLabelText('Paste the key contents'), KEY);

        // Step 2 re-runs the certificate checks on the backend, but the
        // certificate is not editable here: the warning must not come back and
        // must not cost an extra click.
        await user.click(screen.getByTestId('tls-setup-add'));

        await waitFor(() => {
            expect(screen.getByText('Enable encrypted DNS')).toBeInTheDocument();
        });
        expect(screen.queryByText(/This certificate is self-signed/)).toBeNull();
    });

    it('does not bring the warning back when returning to step 1', async () => {
        const user = userEvent.setup();
        warnOnSelfSignedCert();
        renderWizard();

        await goToStep2PastWarning(user);
        await user.click(screen.getByText('Go back'));

        // The message was dropped on the way back and no check runs until the
        // field is touched again.
        expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();
        expect(screen.queryByText(/This certificate is self-signed/)).toBeNull();
    });
});

describe('TlsSetupWizard — step 3 (config & enable)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('treats the server name as optional and prefills the default ports', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);

        expect(screen.getByText('Enable encrypted DNS')).toBeInTheDocument();
        // The warning state is an Add-button affair: Enable keeps its label.
        expect(screen.getByTestId('tls-setup-enable')).toHaveTextContent(/^Enable$/);
        // Empty is valid: the backend just turns off DDR, ClientID detection
        // and the certificate hostname check when there is no name.
        expect(screen.queryByText('Fill out this field')).toBeNull();
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });

        expect(screen.getByDisplayValue('443')).toBeInTheDocument();
        expect(screen.getAllByDisplayValue('853')).toHaveLength(2);
        expect(screen.getAllByTestId('input-clear-button')).toHaveLength(3);
    });

    it('normalizes the server name on blur', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        const serverName = screen.getByPlaceholderText('Enter your domain name');
        await user.type(serverName, 'https://example.com/');
        await user.tab();

        expect((serverName as HTMLInputElement).value).toBe('example.com');
    });

    it('fires a debounced validation with the exact save payload once the form is clean', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue(validStatus);
        renderWizard();

        await goToStep3(user);
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(mocks.tlsValidate.mock.calls.length).toBeGreaterThan(2);
        });
        const calls = mocks.tlsValidate.mock.calls;
        const payload = calls[calls.length - 1][0];
        expect(payload.enabled).toBe(true);
        expect(payload.serve_plain_dns).toBe(true);
        expect(atob(payload.certificate_chain)).toBe(CERT);
        expect(atob(payload.private_key)).toBe(KEY);
    });

    it('runs the pre-flight with an empty server name', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue(validStatus);
        renderWizard();

        await goToStep3(user);

        await waitFor(() => {
            expect(mocks.tlsValidate.mock.calls.length).toBeGreaterThan(2);
        });
        const calls = mocks.tlsValidate.mock.calls;
        const payload = calls[calls.length - 1][0];
        expect(payload.enabled).toBe(true);
        expect(payload.server_name).toBe('');
    });

    it('sends the user back to the key step when the pair breaks', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_pair: false,
            warning_validation:
                'validating certificate pair: certificate-key pair: x509: private key does not match public key',
        });
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        // The message belongs to the key field, so it is shown there — no
        // form-level error with a hint to go back.
        await waitFor(() => {
            expect(screen.getByText('Add private key')).toBeInTheDocument();
        });
        expect(
            screen.getByText(
                'This private key does not match the certificate from the previous step',
            ),
        ).toBeInTheDocument();
        expect(screen.queryByText(/Go back to a previous step/)).toBeNull();
    });

    it('sends the user back to the certificate step when the chain regresses', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_cert: false,
            warning_validation:
                'validating certificate pair: parsing certificate at index 0: x509: malformed certificate',
        });
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        // The message belongs to the certificate field, so the wizard goes back
        // to the step that owns it instead of blocking the config step.
        await waitFor(() => {
            expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();
        });
        expect(
            screen.getByText('Unable to parse the certificate. The file may be corrupted'),
        ).toBeInTheDocument();
        expect(screen.queryByText(/Go back to a previous step/)).toBeNull();
    });

    it('treats a non-critical verify failure as a warning and lets the user enable', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
        });
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        // The certificate step already warned about this chain, and the backend
        // accepts the config: the config step stays clean and Enable works.
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });
        expect(screen.getByText('Enable encrypted DNS')).toBeInTheDocument();
        expect(screen.queryByText(/Certificate has issues/)).toBeNull();
    });

    it('hides a repeated certificate warning on the config step', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation: SELF_SIGNED_WARNING,
        });
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(mocks.tlsValidate.mock.calls.length).toBeGreaterThan(2);
        });
        // The certificate warning was already shown on the step that collected
        // the certificate — repeating it here would be noise, and it never
        // blocks enabling.
        expect(screen.queryByText(/This certificate is self-signed/)).toBeNull();
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });
    });

    it('shows a hostname mismatch as a warning and still allows enabling', async () => {
        const user = userEvent.setup();
        const onClose = renderWizard();
        mocks.tlsConfigure.mockImplementation(async (v: unknown) => v);

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate is valid for example.com, not dns.home.arpa',
        });
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'dns.home.arpa');
        await user.tab();

        await waitFor(() => {
            expect(
                screen.getByText(
                    'The certificate is not valid for dns.home.arpa. Check the hostnames in the certificate',
                ),
            ).toBeInTheDocument();
        });

        // The certificate and key pair are valid, so the backend accepts the
        // config: a name that is missing from the certificate warns instead of
        // blocking Enable.
        const enable = screen.getByTestId('tls-setup-enable');
        expect(enable).not.toBeDisabled();

        await user.click(enable);
        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
        expect(onClose).toHaveBeenCalled();
    });

    it('maps a 400 port-busy error to the matching port field', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockRejectedValue(
            new Error(
                'http://127.0.0.1/control/tls/validate | port 853 for DNS-over-TLS is not available | 400',
            ),
        );
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(
                screen.getByText('Port 853 is not available for DNS-over-TLS'),
            ).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
    });

    it('shows a port conflict with another AGH setting under the port field', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        // dashboardState.httpPort is 80 — reusing it for HTTPS collides.
        const https = screen.getByDisplayValue('443');
        await user.clear(https);
        await user.type(https, '80');
        await user.tab();

        await waitFor(() => {
            expect(
                screen.getByText('This port is already used by another AdGuard Home setting'),
            ).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
    });

    it('shows the generic error inline when saving fails and keeps the dialog open', async () => {
        const user = userEvent.setup();
        const onClose = renderWizard();
        mocks.tlsConfigure.mockRejectedValue(
            new Error('http://127.0.0.1/control/tls/configure | saving: boom | 500'),
        );

        await goToStep3(user);
        const enable = screen.getByTestId('tls-setup-enable');
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();
        await waitFor(() => {
            expect(enable).not.toBeDisabled();
        });

        await user.click(enable);

        await waitFor(() => {
            expect(
                screen.getByText(
                    'Unable to enable encrypted DNS. Check the settings and try again',
                ),
            ).toBeInTheDocument();
        });
        expect(onClose).not.toHaveBeenCalled();
        // The wizard renders the error inline instead of toasting it.
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });

    it('Enable re-validates, then saves and closes the dialog', async () => {
        const user = userEvent.setup();
        const onClose = renderWizard();
        mocks.tlsValidate.mockResolvedValue(validStatus);
        mocks.tlsConfigure.mockImplementation(async (v: unknown) => v);

        await goToStep3(user);
        const enable = screen.getByTestId('tls-setup-enable');
        const serverName = screen.getByPlaceholderText('Enter your domain name');

        // A malformed name keeps Enable blocked …
        await user.type(serverName, 'not a domain');
        await user.tab();
        expect(screen.getByText('Invalid server name')).toBeInTheDocument();
        expect(enable).toBeDisabled();

        // … a valid one unblocks it and the save is sent.
        await user.clear(serverName);
        await user.type(serverName, 'example.com');
        await user.tab();
        await waitFor(() => {
            expect(enable).not.toBeDisabled();
        });

        await user.click(enable);

        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
        await waitFor(() => {
            expect(onClose).toHaveBeenCalled();
        });

        const saved = mocks.tlsConfigure.mock.calls[0][0];
        expect(saved.enabled).toBe(true);
        expect(saved.serve_plain_dns).toBe(true);
        expect(saved.server_name).toBe('example.com');
        expect(atob(saved.certificate_chain)).toBe(CERT);
        expect(atob(saved.private_key)).toBe(KEY);
        // Explicit false: a stale `true` would make the backend keep the
        // previously stored key instead of the one just entered.
        expect(saved.private_key_saved).toBe(false);
    });
});

// These last two blocks touch the real store, a module singleton shared by
// every test above: keep them below the tests that need a pristine store.
describe('TlsSetupWizard — stored key is never offered', () => {
    beforeEach(() => vi.clearAllMocks());

    it('does not offer to reuse the stored key even when the backend reports one', async () => {
        const user = userEvent.setup();
        // A key entered as text is persisted in the configuration file, but
        // `marshalTLS` never returns its contents — only this flag.
        mocks.tlsStatus.mockResolvedValue({ ...validStatus, private_key_saved: true });
        await getTlsStatus();

        renderWizard();
        await goToStep2(user);

        expect(screen.queryByText('Use an existing private key')).not.toBeInTheDocument();

        // A key still has to be provided on this step.
        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByText('Add private key')).toBeInTheDocument();
        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
    });
});

describe('TlsSetupWizard — a hidden warning does not cost an extra click', () => {
    beforeEach(() => vi.clearAllMocks());

    it('enables on the first click when the only warning is not shown', async () => {
        const user = userEvent.setup();
        mocks.tlsConfigure.mockImplementation(async (v: unknown) => v);
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation: SELF_SIGNED_WARNING,
        });
        const serverName = screen.getByPlaceholderText('Enter your domain name');
        await user.clear(serverName);
        await user.type(serverName, 'example.com');
        await user.tab();
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });

        // The warning is not rendered on this step, so asking for a confirming
        // second click would leave the user clicking at nothing.
        await user.click(screen.getByTestId('tls-setup-enable'));
        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
    });
});
