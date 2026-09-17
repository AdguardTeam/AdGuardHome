import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

// Deep proxy: `theme` is written as theme.form.action, theme.dialog.body, etc.
// CSS modules resolve to empty objects under `css: false`, so either mock the
// exact class names or return the proxy itself for every property.
const { themeMock, encryptionState, mocks } = vi.hoisted(() => {
    const proxy: any = new Proxy(
        {},
        {
            get: (_target, prop) => {
                if (prop === Symbol.toPrimitive || prop === 'toString') {
                    return () => '';
                }
                return proxy;
            },
        },
    );

    return {
        themeMock: proxy,
        encryptionState: {
            enabled: true,
            certificate_chain: '-----BEGIN CERTIFICATE-----',
            certificate_path: '',
            private_key: '-----BEGIN PRIVATE KEY-----',
            private_key_path: '',
            private_key_saved: false,
            valid_chain: true,
            valid_cert: true,
            valid_key: true,
            valid_pair: true,
            warning_validation: '',
            subject: 'example.org',
            issuer: 'Example CA',
            not_after: '2030-01-01T00:00:00Z',
            dns_names: ['example.org'] as string[] | null,
            key_type: 'RSA' as string | undefined,
        } as Record<string, unknown>,
        mocks: {
            setTlsConfig: vi.fn(),
            resetValidationStatus: vi.fn(),
            applyTlsOptimistically: vi.fn(),
        },
    };
});

vi.mock('panel/lib/theme', () => ({
    default: themeMock,
}));

vi.mock('panel/stores/encryption', () => ({
    encryptionState,
    ...mocks,
}));

// The section renders localized copy; serve it from the base locale so the
// assertions can name the key instead of re-stating the text.
vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { TlsCertSection } from 'panel/components/Encryption/blocks/TlsCertSection';
import { copy, copyInDom } from 'panel/__tests__/helpers/copy';

const baseState = {
    enabled: true,
    certificate_chain: '-----BEGIN CERTIFICATE-----',
    certificate_path: '',
    private_key: '-----BEGIN PRIVATE KEY-----',
    private_key_path: '',
    private_key_saved: false,
    valid_chain: true,
    valid_cert: true,
    valid_key: true,
    valid_pair: true,
    warning_validation: '',
    subject: 'example.org',
    issuer: 'Example CA',
    not_after: '2030-01-01T00:00:00Z',
    dns_names: ['example.org'] as string[] | null,
    key_type: 'RSA' as string | undefined,
};

const renderSection = (patch: Partial<typeof baseState> = {}) => {
    Object.assign(encryptionState, baseState, patch);

    return render(() => <TlsCertSection />);
};

const algorithmLine = () => screen.queryByText(/Encryption algorithm/);

beforeEach(() => {
    vi.clearAllMocks();
});

describe('TlsCertSection — encryption algorithm', () => {
    it('renders the algorithm of a valid private key', () => {
        renderSection();

        expect(
            screen.getByText(copyInDom('encryption_key_type', { value: 'RSA' })),
        ).toBeInTheDocument();
        expect(
            screen.getByText(copyInDom('encryption_subject', { value: 'example.org' })),
        ).toBeInTheDocument();
    });

    it('renders the algorithm of an ECDSA key', () => {
        renderSection({ key_type: 'ECDSA' });

        expect(
            screen.getByText(copyInDom('encryption_key_type', { value: 'ECDSA' })),
        ).toBeInTheDocument();
    });

    it('does not render the algorithm when the key type is unknown', () => {
        renderSection({ key_type: undefined });

        expect(algorithmLine()).not.toBeInTheDocument();
        // The certificate metadata is still rendered.
        expect(
            screen.getByText(copyInDom('encryption_subject', { value: 'example.org' })),
        ).toBeInTheDocument();
    });

    it('does not render the algorithm when the private key is invalid', () => {
        // The store keeps a stale `key_type: 'RSA'` default, so an invalid key
        // must not leak it into the UI.
        renderSection({ valid_key: false, key_type: 'RSA' });

        expect(algorithmLine()).not.toBeInTheDocument();
        expect(screen.getByText(copyInDom('encryption_chain_valid'))).toBeInTheDocument();
    });
});

describe('TlsCertSection — validation messages', () => {
    /** The backend text the section renders verbatim as the status message. */
    const serverNameWarning = copy('tls_setup_warning_server_name_mismatch', {
        hostname: 'example.org',
    });

    // The validation message is a state of its own: it replaces the
    // certificate block instead of stacking on top of it, so only one status
    // is ever visible.
    it('shows the mismatch message instead of the certificate block', () => {
        renderSection({ valid_pair: false, key_type: 'ECDSA' });

        expect(screen.getByText(copyInDom('encryption_key_cert_mismatch'))).toBeInTheDocument();
        expect(screen.queryByText(copyInDom('encryption_chain_valid'))).not.toBeInTheDocument();
        expect(
            screen.queryByText(copyInDom('encryption_subject', { value: 'example.org' })),
        ).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });

    it('shows the warning message instead of the certificate block', () => {
        renderSection({ warning_validation: serverNameWarning });

        expect(screen.getByText(serverNameWarning)).toBeInTheDocument();
        expect(
            screen.getByText(copyInDom('encryption_certificate_has_issues')),
        ).toBeInTheDocument();
        expect(screen.queryByText(copyInDom('encryption_chain_valid'))).not.toBeInTheDocument();
        expect(
            screen.queryByText(copyInDom('encryption_subject', { value: 'example.org' })),
        ).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });

    it('shows the validation message alone when no certificate is configured', () => {
        renderSection({
            certificate_chain: '',
            certificate_path: '',
            warning_validation: serverNameWarning,
        });

        expect(screen.getByText(serverNameWarning)).toBeInTheDocument();
        expect(screen.queryByText(copyInDom('encryption_chain_valid'))).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });
});

describe('TlsCertSection — no certificate', () => {
    it('renders no status block when there is neither a certificate nor a validation message', () => {
        renderSection({ certificate_chain: '', certificate_path: '' });

        expect(screen.getByTestId('tls-cert-section')).toBeInTheDocument();
        expect(screen.queryByText(copyInDom('encryption_chain_valid'))).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });
});

describe('TlsCertSection — remove certificate', () => {
    /**
     * The whole config the removal writes: encryption off, and every value the
     * certificate came with back to its default.  Spelled out rather than
     * derived from the defaults, because the request replaces the entire TLS
     * config — a field missing here is a field left behind on a server that no
     * longer serves TLS.
     */
    const REMOVE_PAYLOAD = {
        enabled: false,
        serve_plain_dns: true,
        server_name: '',
        force_https: false,
        port_https: 443,
        port_dns_over_tls: 853,
        port_dns_over_quic: 853,
        certificate_chain: '',
        private_key: '',
        certificate_path: '',
        private_key_path: '',
        private_key_saved: false,
    };

    const openConfirm = async (user: ReturnType<typeof userEvent.setup>) => {
        await user.click(screen.getByTestId('tls-cert-remove'));
    };

    const confirmRemove = async (user: ReturnType<typeof userEvent.setup>) => {
        await user.click(screen.getByRole('button', { name: copyInDom('yes_remove') }));
    };

    it('asks for confirmation instead of removing right away', async () => {
        const user = userEvent.setup();
        renderSection();

        await openConfirm(user);

        expect(screen.getByText(copyInDom('remove_tls_certificate'))).toBeInTheDocument();
        expect(mocks.setTlsConfig).not.toHaveBeenCalled();
    });

    it('turns encryption off and resets the TLS data', async () => {
        const user = userEvent.setup();
        renderSection();

        await openConfirm(user);
        await confirmRemove(user);

        expect(mocks.setTlsConfig).toHaveBeenCalledWith(REMOVE_PAYLOAD);
        expect(mocks.resetValidationStatus).toHaveBeenCalledTimes(1);
    });

    it('clears the data in the store before the save it sends', async () => {
        const user = userEvent.setup();
        renderSection();

        await openConfirm(user);
        await confirmRemove(user);

        // The save restarts the DNS server and can take seconds, so the page
        // must not keep showing what is being removed for that long.
        expect(mocks.applyTlsOptimistically).toHaveBeenCalledWith(REMOVE_PAYLOAD);
        expect(mocks.applyTlsOptimistically.mock.invocationCallOrder[0]).toBeLessThan(
            mocks.setTlsConfig.mock.invocationCallOrder[0],
        );
    });

    it('closes the confirmation after the removal', async () => {
        const user = userEvent.setup();
        renderSection();

        await openConfirm(user);
        await confirmRemove(user);

        expect(screen.queryByText(copyInDom('remove_tls_certificate'))).not.toBeInTheDocument();
    });

    it('keeps the config when the removal is cancelled', async () => {
        const user = userEvent.setup();
        renderSection();

        await openConfirm(user);
        await user.click(screen.getByRole('button', { name: copyInDom('cancel') }));

        expect(mocks.setTlsConfig).not.toHaveBeenCalled();
        expect(mocks.applyTlsOptimistically).not.toHaveBeenCalled();
        expect(screen.queryByText(copyInDom('remove_tls_certificate'))).not.toBeInTheDocument();
    });
});
