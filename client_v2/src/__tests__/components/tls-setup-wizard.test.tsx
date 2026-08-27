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
    dashboardState: { httpPort: 80 },
}));
vi.mock('panel/helpers/helpers', () => ({
    redirectToCurrentProtocol: mocks.redirectToCurrentProtocol,
}));

import { TlsSetupWizard } from 'panel/components/Encryption/blocks/SetupWizard';

const CERT = '-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----';
const KEY = '-----BEGIN PRIVATE KEY-----\nxyz\n-----END PRIVATE KEY-----';

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

const renderWizard = () => {
    const onClose = vi.fn();
    render(() => <TlsSetupWizard open onClose={onClose} />);
    return onClose;
};

/** Fills the certificate textarea and moves to step 2. */
const goToStep2 = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.type(screen.getByLabelText('Paste the certificate contents'), CERT);
    await user.click(screen.getByTestId('tls-setup-add'));
};

/** Fills cert + key and moves to step 3. */
const goToStep3 = async (user: ReturnType<typeof userEvent.setup>) => {
    await goToStep2(user);
    await user.type(screen.getByLabelText('Paste the key contents'), KEY);
    await user.click(screen.getByTestId('tls-setup-add'));
};

describe('TlsSetupWizard — shell & header', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders step 1 with the title and a 3-segment progress bar', () => {
        renderWizard();
        expect(screen.getByText('Add certificate')).toBeInTheDocument();

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

        expect(screen.getByText('Add certificate')).toBeInTheDocument();
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

        await user.click(screen.getByText('Path to file on the server'));
        expect(screen.getByLabelText('Full path to the certificate file')).toBeInTheDocument();
        expect(screen.queryByLabelText('Paste the certificate contents')).toBeNull();

        await user.click(screen.getByText('Certificate as text'));
        expect(screen.getByLabelText('Paste the certificate contents')).toBeInTheDocument();
    });

    it('does not advance on empty input and shows an inline error', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
        expect(screen.getByText('Add certificate')).toBeInTheDocument();
        expect(screen.queryByText('Add private key')).toBeNull();
    });

    it('advances to step 2 on valid input', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);

        expect(screen.getByText('Add private key')).toBeInTheDocument();
    });
});

describe('TlsSetupWizard — step 2 (private key)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders three sources and disables the saved-key option when no key is saved', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);

        expect(screen.getByText('Private key as text')).toBeInTheDocument();
        expect(screen.getByText('Path to file on the server')).toBeInTheDocument();

        const savedLabel = screen.getByText('Use an existing private key').closest('label');
        expect(savedLabel).not.toBeNull();
        expect(savedLabel?.querySelector('input')).toBeDisabled();
    });

    it('gates Add on the key validation', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);
        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByText('Fill out this field')).toBeInTheDocument();
        expect(screen.getByText('Add private key')).toBeInTheDocument();
    });
});

describe('TlsSetupWizard — step 3 (config & enable)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('requires the server name and prefills the default ports', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);

        expect(screen.getByText('Enable encrypted DNS')).toBeInTheDocument();
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
        expect(screen.getByText('Fill out this field')).toBeInTheDocument();

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
            expect(mocks.tlsValidate).toHaveBeenCalled();
        });
        const payload = mocks.tlsValidate.mock.calls[0][0];
        expect(payload.enabled).toBe(true);
        expect(payload.serve_plain_dns).toBe(true);
        expect(atob(payload.certificate_chain)).toBe(CERT);
        expect(atob(payload.private_key)).toBe(KEY);
    });

    it('maps invalid backend flags to a blocking error block', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_pair: false,
            warning_validation:
                'validating certificate pair: certificate-key pair: x509: private key does not match public key',
        });
        renderWizard();

        await goToStep3(user);
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByText('Private key does not match certificate')).toBeInTheDocument();
        });
        expect(screen.getByText('Go back to a previous step to fix the issue')).toBeInTheDocument();
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
    });

    it('shows a warning and still allows enabling for an untrusted chain', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate signed by unknown authority',
        });
        renderWizard();

        await goToStep3(user);
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(
                screen.getByText(
                    'The certificate chain cannot be verified: it may be self-signed, expired, or for a different hostname',
                ),
            ).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
    });

    it('maps a 400 port-busy error to an inline message', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockRejectedValue(
            new Error(
                'http://127.0.0.1/control/tls/validate | port 443 for HTTPS is not available | 400',
            ),
        );
        renderWizard();

        await goToStep3(user);
        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByText('Port 443 is not available for HTTPS')).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
    });

    it('Enable re-validates, then saves and closes the dialog', async () => {
        const user = userEvent.setup();
        const onClose = renderWizard();
        mocks.tlsValidate.mockResolvedValue(validStatus);
        mocks.tlsConfigure.mockImplementation(async (v: unknown) => v);

        await goToStep3(user);
        const enable = screen.getByTestId('tls-setup-enable');
        expect(enable).toBeDisabled();

        await user.type(screen.getByPlaceholderText('Enter your domain name'), 'example.com');
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
    });
});
