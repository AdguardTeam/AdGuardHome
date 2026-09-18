import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@solidjs/testing-library';
import userEventBase from '@testing-library/user-event';

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
// The wizard debounces every per-step backend check.  Shrinking the constant
// keeps the behaviour under test — the assertions still wait for the check to
// land — without making every test pay the real 300ms of wall clock.
vi.mock('panel/helpers/constants', async (importOriginal) => ({
    ...(await importOriginal<typeof import('panel/helpers/constants')>()),
    DEBOUNCE_TIMEOUT: 20,
}));

import { TlsSetupWizard } from 'panel/components/Encryption/blocks/SetupWizard';
import { getTlsStatus } from 'panel/stores/encryption';

/**
 * The copy the wizard renders, read from the same locale file the app loads.
 * Asserting via the keys keeps the tests about *which* message is shown, so
 * rewording it does not break them.
 */
import en from 'panel/__locales/en.json';
import { copy, copyInDom } from 'panel/__tests__/helpers/copy';

/** Anchored match, so the label `Add` does not also match `Add anyway`. */
const exactly = (text: string) => new RegExp(`^${text}$`);

const ADD = copyInDom('add');
const ENABLE = copyInDom('enable');
const ADD_ANYWAY = copyInDom('tls_setup_add_anyway');
const ENABLE_ANYWAY = copyInDom('tls_setup_enable_anyway');
const CERT_UNTRUSTED = copyInDom('tls_setup_warning_cert_untrusted');
const CERT_EXPIRED = copyInDom('tls_setup_warning_cert_expired');
const PARSE_CERT = copyInDom('tls_setup_error_parse_cert');
const NOT_A_CERT = copyInDom('tls_setup_error_not_a_cert');
const KEY_MISMATCH = copyInDom('tls_setup_error_key_mismatch');
/** The warning the config step shows for a name the certificate does not cover. */
const NAME_MISMATCH = copyInDom('tls_setup_warning_server_name_mismatch', {
    hostname: 'dns.home.arpa',
});

/**
 * user-event waits a tick between every dispatched event, which costs ~2ms per
 * character — and every step-3 test types a 56-char certificate plus a 46-char
 * key.  The wizard does not depend on that pacing.
 */
const userEvent = {
    ...userEventBase,
    setup: (options?: Parameters<typeof userEventBase.setup>[0]) =>
        userEventBase.setup({ ...options, delay: null }),
};

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
    await user.type(screen.getByTestId('tls-setup-cert-content'), CERT);
    await user.click(screen.getByTestId('tls-setup-add'));
    await waitFor(() => {
        expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
    });
};

/** Fills cert + key and moves to step 3. */
const goToStep3 = async (user: ReturnType<typeof userEvent.setup>) => {
    await goToStep2(user);
    await user.type(screen.getByTestId('tls-setup-key-content'), KEY);
    await user.click(screen.getByTestId('tls-setup-add'));
    await waitFor(() => {
        expect(screen.getByTestId('tls-setup-step-config')).toBeInTheDocument();
    });
};

describe('TlsSetupWizard — shell & header', () => {
    beforeEach(() => vi.clearAllMocks());

    it('shows the step title, hides Go back on step 1 and shows it on steps 2-3', async () => {
        const user = userEvent.setup();
        renderWizard();

        expect(screen.getByTestId('tls-setup-title')).toHaveTextContent(
            copyInDom('tls_setup_cert_title'),
        );
        expect(screen.getByTestId('tls-setup-progress')).toHaveAttribute('aria-valuenow', '1');
        expect(screen.queryByTestId('tls-setup-back')).toBeNull();

        await goToStep2(user);
        expect(screen.getByTestId('tls-setup-back')).toBeInTheDocument();

        await user.type(screen.getByTestId('tls-setup-key-content'), KEY);
        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByTestId('tls-setup-back')).toBeInTheDocument();
        expect(screen.getByTestId('tls-setup-progress')).toHaveAttribute('aria-valuenow', '3');
    });

    it('Go back returns to the previous step preserving entered values', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);
        await user.click(screen.getByTestId('tls-setup-back'));

        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        expect(screen.getByTestId('tls-setup-cert-content')).toHaveValue(CERT);
    });
});

describe('TlsSetupWizard — step 1 (certificate)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('swaps textarea and path input when the source changes', async () => {
        const user = userEvent.setup();
        renderWizard();

        // Default: content — textarea with the design placeholder.
        const textarea = screen.getByTestId('tls-setup-cert-content');
        expect(textarea).toHaveAttribute('placeholder', '-----BEGIN CERTIFICATE-----');

        await user.click(screen.getByTestId('tls-setup-cert-source-path'));
        expect(screen.getByTestId('tls-setup-cert-path')).toBeInTheDocument();
        expect(screen.queryByTestId('tls-setup-cert-content')).toBeNull();

        await user.click(screen.getByTestId('tls-setup-cert-source-content'));
        expect(screen.getByTestId('tls-setup-cert-content')).toBeInTheDocument();
    });

    it('renders the letsencrypt.org link in the step description', () => {
        renderWizard();

        const link = screen.getByTestId('tls-setup-letsencrypt-link');
        expect(link).toHaveTextContent('letsencrypt.org');
        expect(link).toHaveAttribute('href', 'https://letsencrypt.org/');
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    });

    it('does not advance on empty input and shows an inline error', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByTestId('tls-setup-cert-content-error')).toBeInTheDocument();
        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        expect(screen.queryByTestId('tls-setup-step-key')).toBeNull();
    });

    it('does not flag an untouched field when the file picker takes focus', async () => {
        const user = userEvent.setup();
        renderWizard();

        // Focus the textarea, then reach for Browse: the native file dialog
        // takes focus away without any value having been entered.
        await user.click(screen.getByTestId('tls-setup-cert-content'));
        await user.click(screen.getByTestId('tls-setup-cert-dropzone'));

        expect(screen.queryByTestId('tls-setup-cert-content-error')).toBeNull();
        expect(mocks.tlsValidate).not.toHaveBeenCalled();

        // The required error still belongs to Add.
        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByTestId('tls-setup-cert-content-error')).toBeInTheDocument();
    });

    it('still reports content errors on blur', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(screen.getByTestId('tls-setup-cert-content'), 'not a pem block');
        await user.tab();

        expect(screen.getByTestId('tls-setup-cert-content-error')).toHaveTextContent(
            en.tls_setup_error_cert_incomplete,
        );
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

        expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        expect(screen.queryByTestId('tls-setup-form-message')).toBeNull();
        expect(screen.queryByTestId('tls-setup-key-content-error')).toBeNull();
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
            screen.getByTestId('tls-setup-cert-content'),
            '-----BEGIN CERTIFICATE-----\nbroken\n-----END CERTIFICATE-----',
        );
        await user.click(screen.getByTestId('tls-setup-add'));

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-cert-content-error')).toHaveTextContent(
                PARSE_CERT,
            );
        });
        // Still on step 1.
        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        // An error never turns Add into its warning state.
        expect(screen.getByTestId('tls-setup-add')).toHaveTextContent(exactly(ADD));
    });

    it('shows the self-signed warning on the first Add and advances on the second', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue({
            ...CERT_ONLY_STATUS,
            valid_chain: false,
            warning_validation: SELF_SIGNED_WARNING,
        });
        renderWizard();

        await user.type(screen.getByTestId('tls-setup-cert-content'), CERT);

        // Nothing is known to be wrong until the check runs: a plain Add.
        const add = screen.getByTestId('tls-setup-add');
        expect(add).toHaveTextContent(exactly(ADD));
        expect(add.className).not.toContain('warning');

        await user.click(add);

        // The warning is surfaced on this step instead of being carried to the
        // next one, and it never blocks — it just has to be seen once.
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-cert-content-warning')).toHaveTextContent(
                CERT_UNTRUSTED,
            );
        });
        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();

        // The button now spells out what going past the warning means.
        expect(add).toHaveTextContent(ADD_ANYWAY);
        expect(add.className).toContain('warning');

        await user.click(add);
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        });
    });

    it('blocks step 1 when the pasted content is not a certificate', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(screen.getByTestId('tls-setup-cert-content'), KEY);
        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByTestId('tls-setup-cert-content-error')).toHaveTextContent(NOT_A_CERT);
        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        expect(mocks.tlsValidate).not.toHaveBeenCalled();
    });

    it('blocks step 1 when the certificate is missing its closing line', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(
            screen.getByTestId('tls-setup-cert-content'),
            '-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKlM4NvZ5W4r',
        );
        await user.tab();

        expect(screen.getByTestId('tls-setup-cert-content-error')).toHaveTextContent(
            en.tls_setup_error_cert_incomplete,
        );
        // Truncated PEM is caught before the backend is asked to parse it.
        expect(mocks.tlsValidate).not.toHaveBeenCalled();
    });

    it('sends a cert-only, port-free payload on step 1', async () => {
        const user = userEvent.setup();
        renderWizard();

        await user.type(screen.getByTestId('tls-setup-cert-content'), CERT);
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
        await user.type(screen.getByTestId('tls-setup-key-content'), KEY);
        await user.click(screen.getByTestId('tls-setup-add'));

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-key-content-error')).toHaveTextContent(
                KEY_MISMATCH,
            );
        });
        expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument(); // stays on step 2
    });

    it('blocks step 2 when the key is missing its closing line', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep2(user);
        await user.type(
            screen.getByTestId('tls-setup-key-content'),
            '-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0B',
        );
        const callsBefore = mocks.tlsValidate.mock.calls.length;
        await user.click(screen.getByTestId('tls-setup-add'));

        expect(screen.getByTestId('tls-setup-key-content-error')).toHaveTextContent(
            en.tls_setup_error_key_incomplete,
        );
        expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        // The client gate held the step, so the truncated key was never sent.
        expect(mocks.tlsValidate.mock.calls.length).toBe(callsBefore);
    });
});

describe('TlsSetupWizard — expired certificate', () => {
    beforeEach(() => vi.clearAllMocks());

    /** A valid pair whose chain does not cover the current date. */
    const warnOnExpiredCert = () =>
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            not_after: '2020-01-01T00:00:00Z',
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
        });

    it('warns on step 1 and advances on the second Add', async () => {
        const user = userEvent.setup();
        warnOnExpiredCert();
        renderWizard();

        await user.type(screen.getByTestId('tls-setup-cert-content'), CERT);
        const add = screen.getByTestId('tls-setup-add');
        await user.click(add);

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-cert-content-warning')).toHaveTextContent(
                CERT_EXPIRED,
            );
        });
        // An expiry is not critical: the step warns and the button says so.
        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        expect(add).toHaveTextContent(ADD_ANYWAY);

        await user.click(add);
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        });
    });

    it('keeps the config step clean, so Enable does not ask for confirmation', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            not_after: '2020-01-01T00:00:00Z',
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
        });
        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();

        // The warning was already shown where the certificate is entered, is not
        // repeated here, and never blocks: the backend accepts an expired
        // certificate, so Enable stays a plain Enable rather than a confirmation.
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-config')).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        expect(screen.getByTestId('tls-setup-enable')).toHaveTextContent(exactly(ENABLE));
        expect(screen.queryByTestId('tls-setup-form-message')).toBeNull();
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
        await user.type(screen.getByTestId('tls-setup-cert-content'), CERT);
        await user.click(screen.getByTestId('tls-setup-add'));
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-cert-content-warning')).toBeInTheDocument();
        });
        await user.click(screen.getByTestId('tls-setup-add'));
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        });
    };

    it('does not carry the warning into step 2', async () => {
        const user = userEvent.setup();
        warnOnSelfSignedCert();
        renderWizard();

        await goToStep2PastWarning(user);

        expect(screen.queryByTestId('tls-setup-key-content-warning')).toBeNull();
    });

    it('advances from step 2 in one click without repeating the warning', async () => {
        const user = userEvent.setup();
        warnOnSelfSignedCert();
        renderWizard();

        await goToStep2PastWarning(user);
        await user.type(screen.getByTestId('tls-setup-key-content'), KEY);

        // Step 2 re-runs the certificate checks on the backend, but the
        // certificate is not editable here: the warning must not come back and
        // must not cost an extra click.
        await user.click(screen.getByTestId('tls-setup-add'));

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-config')).toBeInTheDocument();
        });
        expect(screen.queryByTestId('tls-setup-key-content-warning')).toBeNull();
    });

    it('does not bring the warning back when returning to step 1', async () => {
        const user = userEvent.setup();
        warnOnSelfSignedCert();
        renderWizard();

        await goToStep2PastWarning(user);
        await user.click(screen.getByTestId('tls-setup-back'));

        // The message was dropped on the way back and no check runs until the
        // field is touched again.
        expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        expect(screen.queryByTestId('tls-setup-cert-content-warning')).toBeNull();
    });
});

describe('TlsSetupWizard — step 3 (config & enable)', () => {
    beforeEach(() => vi.clearAllMocks());

    it('treats the server name as optional and prefills the default ports', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);

        expect(screen.getByTestId('tls-setup-step-config')).toBeInTheDocument();
        // No name, no warning: the backend skips the certificate hostname
        // check, so Enable stays plain.
        const enable = screen.getByTestId('tls-setup-enable');
        expect(enable).toHaveTextContent(/^Enable$/);
        expect(enable.className).not.toContain('warning');
        // Empty is valid: the backend just turns off DDR, ClientID detection
        // and the certificate hostname check when there is no name.
        expect(screen.queryByTestId('tls-setup-server-name-error')).toBeNull();
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });

        expect(screen.getByTestId('tls-setup-port-https')).toHaveValue(443);
        expect(screen.getByTestId('tls-setup-port-dns-over-tls')).toHaveValue(853);
        expect(screen.getByTestId('tls-setup-port-dns-over-quic')).toHaveValue(853);
        expect(screen.getAllByTestId('input-clear-button')).toHaveLength(3);
    });

    it('normalizes the server name on blur', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        const serverName = screen.getByTestId('tls-setup-server-name');
        await user.type(serverName, 'https://example.com/');
        await user.tab();

        expect(serverName).toHaveValue('example.com');
    });

    it('offers a clear button for the server name as soon as it has text', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);

        // The wizard opens with no name, so there is nothing to clear yet.
        expect(screen.getAllByTestId('input-clear-button')).toHaveLength(3);

        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');

        // Typing alone is enough: the field has content, so the button is there
        // before the value is committed to the form.
        const serverName = screen.getByTestId('tls-setup-server-name');
        await user.click(
            within(serverName.parentElement as HTMLElement).getByTestId('input-clear-button'),
        );

        expect(serverName).toHaveValue('');
    });

    it('fires a debounced pre-flight with the exact save payload once the form is clean', async () => {
        const user = userEvent.setup();
        mocks.tlsValidate.mockResolvedValue(validStatus);
        renderWizard();

        await goToStep3(user);

        // The pre-flight runs with an empty name too: the backend only skips
        // DDR, ClientID detection and the hostname check in that case.
        await waitFor(() => {
            const calls = mocks.tlsValidate.mock.calls;
            expect(calls[calls.length - 1][0].enabled).toBe(true);
        });
        const emptyNameCalls = mocks.tlsValidate.mock.calls;
        expect(emptyNameCalls[emptyNameCalls.length - 1][0].server_name).toBe('');

        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            const calls = mocks.tlsValidate.mock.calls;
            expect(calls[calls.length - 1][0].server_name).toBe('example.com');
        });
        const calls = mocks.tlsValidate.mock.calls;
        const payload = calls[calls.length - 1][0];
        expect(payload.enabled).toBe(true);
        expect(payload.serve_plain_dns).toBe(true);
        expect(atob(payload.certificate_chain)).toBe(CERT);
        expect(atob(payload.private_key)).toBe(KEY);
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
        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();

        // The message belongs to the key field, so it is shown there — no
        // form-level error with a hint to go back.
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-key-content-error')).toHaveTextContent(KEY_MISMATCH);
        expect(screen.queryByTestId('tls-setup-form-message')).toBeNull();
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
        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();

        // The message belongs to the certificate field, so the wizard goes back
        // to the step that owns it instead of blocking the config step.
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-step-certificate')).toBeInTheDocument();
        });
        expect(screen.getByTestId('tls-setup-cert-content-error')).toHaveTextContent(PARSE_CERT);
        expect(screen.queryByTestId('tls-setup-form-message')).toBeNull();
    });

    it('treats a non-critical verify failure as a warning and lets the user enable', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            not_after: '2020-01-01T00:00:00Z',
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate has expired or is not yet valid',
        });
        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();

        // The certificate step already warned about this chain, and the backend
        // accepts the config: the config step stays clean and Enable works.
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });
        expect(screen.getByTestId('tls-setup-step-config')).toBeInTheDocument();
        expect(screen.queryByTestId('tls-setup-server-name-warning')).toBeNull();
        expect(screen.queryByTestId('tls-setup-form-message')).toBeNull();
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
        await user.type(screen.getByTestId('tls-setup-server-name'), 'dns.home.arpa');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-server-name-warning')).toHaveTextContent(
                NAME_MISMATCH,
            );
        });

        // The certificate and key pair are valid, so the backend accepts the
        // config: a name that is missing from the certificate warns instead of
        // blocking Enable.  The label spells out what going past it means.
        const enable = screen.getByTestId('tls-setup-enable');
        expect(enable).not.toBeDisabled();
        expect(enable).toHaveTextContent(ENABLE_ANYWAY);
        expect(enable.className).toContain('warning');

        // The warning is already on screen, so a single click goes through.
        await user.click(enable);
        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
        expect(onClose).toHaveBeenCalled();
    });

    it('surfaces a mismatch found by the click and enables on the second one', async () => {
        const user = userEvent.setup();
        const onClose = renderWizard();
        mocks.tlsConfigure.mockImplementation(async (v: unknown) => v);

        // Every pre-flight is clean, so nothing is on screen when Enable is
        // clicked; only the check the click itself triggers warns.
        let preflightWithName = false;
        mocks.tlsValidate.mockImplementation(async (v: { server_name?: string }) => {
            if (v.server_name) preflightWithName = true;

            return validStatus;
        });

        await goToStep3(user);
        const serverName = screen.getByTestId('tls-setup-server-name');
        // Earlier cases save a server name through the shared store, so start
        // from a known empty field instead of appending to theirs.
        await user.clear(serverName);
        await user.type(serverName, 'dns.home.arpa');
        await user.tab();
        await waitFor(() => {
            expect(preflightWithName).toBe(true);
        });

        // Clean check: nothing is known to be wrong yet.
        const enable = screen.getByTestId('tls-setup-enable');
        expect(enable).toHaveTextContent(exactly(ENABLE));
        expect(enable.className).not.toContain('warning');

        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate is valid for example.com, not dns.home.arpa',
        });
        await user.click(enable);

        // The first click only puts the warning on screen — Enable must not
        // save past a warning the user has not seen.
        await waitFor(() => {
            expect(enable).toHaveTextContent(ENABLE_ANYWAY);
        });
        expect(enable.className).toContain('warning');
        expect(enable).not.toBeDisabled();
        expect(screen.getByTestId('tls-setup-server-name-warning')).toHaveTextContent(
            NAME_MISMATCH,
        );
        expect(mocks.tlsConfigure).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();

        // Seen once: the next click is the confirmation.
        await user.click(enable);
        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
        expect(onClose).toHaveBeenCalled();
    });

    it('drops the warning state again when the name is edited', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        mocks.tlsValidate.mockResolvedValue({
            ...validStatus,
            valid_chain: false,
            warning_validation:
                'validating certificate pair: certificate does not verify: x509: certificate is valid for example.com, not dns.home.arpa',
        });
        const serverName = screen.getByTestId('tls-setup-server-name');
        await user.type(serverName, 'dns.home.arpa');
        await user.tab();

        const enable = screen.getByTestId('tls-setup-enable');
        await waitFor(() => {
            expect(enable).toHaveTextContent(ENABLE_ANYWAY);
        });

        // Editing the name drops the message, so the button goes back to plain
        // Enable — the clean answer keeps the debounced re-check from putting
        // the warning straight back.
        mocks.tlsValidate.mockResolvedValue(validStatus);
        await user.clear(serverName);
        await user.type(serverName, 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(enable).toHaveTextContent(exactly(ENABLE));
        });
        expect(enable.className).not.toContain('warning');
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
        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-port-dns-over-tls-error')).toHaveTextContent(
                copyInDom('tls_setup_error_port_busy', { port: 853, protocol: 'DNS-over-TLS' }),
            );
        });
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
    });

    it('shows a port conflict with another AGH setting under the port field', async () => {
        const user = userEvent.setup();
        renderWizard();

        await goToStep3(user);
        // dashboardState.httpPort is 80 — reusing it for HTTPS collides.
        const https = screen.getByTestId('tls-setup-port-https');
        await user.clear(https);
        await user.type(https, '80');
        await user.tab();

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-port-https-error')).toBeInTheDocument();
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
        await user.type(screen.getByTestId('tls-setup-server-name'), 'example.com');
        await user.tab();
        await waitFor(() => {
            expect(enable).not.toBeDisabled();
        });

        await user.click(enable);

        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-form-message')).toBeInTheDocument();
        });
        // The failed save drops the pending state: no stuck loader, no stuck
        // disabled fields.
        expect(
            screen.getByTestId('tls-setup-enable').querySelector('[class*="loader"]'),
        ).toBeNull();
        expect(screen.getByTestId('tls-setup-port-https')).not.toBeDisabled();
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
        const serverName = screen.getByTestId('tls-setup-server-name');

        // A malformed name keeps Enable blocked …
        await user.type(serverName, 'not a domain');
        await user.tab();
        expect(screen.getByTestId('tls-setup-server-name-error')).toBeInTheDocument();
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

        // Two sources only — retrieving the stored key is never offered.
        expect(screen.getAllByRole('radio')).toHaveLength(2);

        // A key still has to be provided on this step.
        await user.click(screen.getByTestId('tls-setup-add'));
        expect(screen.getByTestId('tls-setup-step-key')).toBeInTheDocument();
        expect(screen.getByTestId('tls-setup-key-content-error')).toBeInTheDocument();
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
        const serverName = screen.getByTestId('tls-setup-server-name');
        await user.clear(serverName);
        await user.type(serverName, 'example.com');
        await user.tab();
        await waitFor(() => {
            expect(screen.getByTestId('tls-setup-enable')).not.toBeDisabled();
        });

        // Nothing warns on this step, so Enable keeps its plain label — an
        // amber "Enable anyway" would point at a warning that is not there.
        const enable = screen.getByTestId('tls-setup-enable');
        expect(enable).toHaveTextContent(/^Enable$/);
        expect(enable.className).not.toContain('warning');

        // The warning is not rendered on this step, so asking for a confirming
        // second click would leave the user clicking at nothing.
        await user.click(enable);
        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
    });
});

describe('TlsSetupWizard — enabling is confirmed by name', () => {
    beforeEach(() => vi.clearAllMocks());

    it('shows the encrypted DNS toast when the enable goes through', async () => {
        const user = userEvent.setup();
        // Start from a disabled configuration: only the save that turns
        // encryption on is confirmed with its own message.
        mocks.tlsStatus.mockResolvedValue({ ...validStatus, enabled: false });
        await getTlsStatus();
        mocks.tlsConfigure.mockImplementation(async (v: unknown) => v);

        const onClose = renderWizard();
        await goToStep3(user);

        const enable = screen.getByTestId('tls-setup-enable');
        await waitFor(() => {
            expect(enable).not.toBeDisabled();
        });
        await user.click(enable);

        await waitFor(() => {
            expect(onClose).toHaveBeenCalled();
        });
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(copy('encryption_enabled_toast'));
    });
});

describe('TlsSetupWizard — the enable button reports that it is working', () => {
    beforeEach(() => vi.clearAllMocks());

    it('shows a loader on Enable until the save settles', async () => {
        const user = userEvent.setup();
        // Hold the save open so the pending state can be observed.
        let finishSave: () => void = () => {};
        mocks.tlsConfigure.mockImplementation(
            (v: unknown) =>
                new Promise((resolve) => {
                    finishSave = () => resolve(v);
                }),
        );

        const onClose = renderWizard();
        await goToStep3(user);

        const enable = screen.getByTestId('tls-setup-enable');
        await waitFor(() => {
            expect(enable).not.toBeDisabled();
        });
        expect(enable.querySelector('[class*="loader"]')).toBeNull();

        await user.click(enable);

        await waitFor(() => {
            expect(mocks.tlsConfigure).toHaveBeenCalled();
        });
        expect(
            screen.getByTestId('tls-setup-enable').querySelector('[class*="loader"]'),
        ).not.toBeNull();
        // The dialog is busy: the button cannot be pressed again and the
        // fields cannot change under the save.
        expect(screen.getByTestId('tls-setup-enable')).toBeDisabled();
        expect(screen.getByTestId('tls-setup-port-https')).toBeDisabled();

        finishSave();

        await waitFor(() => {
            expect(onClose).toHaveBeenCalled();
        });
    });
});
