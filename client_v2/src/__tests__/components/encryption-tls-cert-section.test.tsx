import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

// Deep proxy: `theme` is written as theme.form.action, theme.dialog.body, etc.
// CSS modules resolve to empty objects under `css: false`, so either mock the
// exact class names or return the proxy itself for every property.
const { themeMock, encryptionState } = vi.hoisted(() => {
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
    };
});

vi.mock('panel/lib/theme', () => ({
    default: themeMock,
}));

vi.mock('panel/stores/encryption', () => ({
    encryptionState,
    setTlsConfig: vi.fn(),
    resetValidationStatus: vi.fn(),
    clearCertOptimistically: vi.fn(),
}));

// Mirrors the en.json entries so the assertions read like the real UI.
vi.mock('panel/common/intl', () => {
    const messages: Record<string, string> = {
        tls_certificate: 'TLS Certificate',
        encryption_certificates: 'Certificates',
        encryption_chain_valid: 'Certificate chain is valid',
        encryption_chain_invalid: 'Certificate chain is invalid',
        encryption_subject: 'Subject: %value%',
        encryption_issuer: 'Issuer: %value%',
        encryption_expire: 'Expires: %value%',
        encryption_hostnames: 'Hostnames: %value%',
        encryption_key_type: 'Encryption algorithm: %value%',
        encryption_certificate_has_issues: 'Certificate has issues',
        encryption_key_cert_mismatch: 'Private key does not match certificate',
    };

    return {
        default: {
            getMessage: (key: string, values?: Record<string, unknown>) => {
                const template = messages[key] ?? key;

                return Object.entries(values ?? {}).reduce(
                    (text, [name, value]) => text.replace(`%${name}%`, String(value)),
                    template,
                );
            },
            getUILanguage: () => 'en',
            changeLanguage: vi.fn(),
        },
    };
});

import { TlsCertSection } from 'panel/components/Encryption/blocks/TlsCertSection';

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

        expect(screen.getByText('Encryption algorithm: RSA')).toBeInTheDocument();
        expect(screen.getByText('Subject: example.org')).toBeInTheDocument();
    });

    it('renders the algorithm of an ECDSA key', () => {
        renderSection({ key_type: 'ECDSA' });

        expect(screen.getByText('Encryption algorithm: ECDSA')).toBeInTheDocument();
    });

    it('does not render the algorithm when the key type is unknown', () => {
        renderSection({ key_type: undefined });

        expect(algorithmLine()).not.toBeInTheDocument();
        // The certificate metadata is still rendered.
        expect(screen.getByText('Subject: example.org')).toBeInTheDocument();
    });

    it('does not render the algorithm when the private key is invalid', () => {
        // The store keeps a stale `key_type: 'RSA'` default, so an invalid key
        // must not leak it into the UI.
        renderSection({ valid_key: false, key_type: 'RSA' });

        expect(algorithmLine()).not.toBeInTheDocument();
        expect(screen.getByText('Certificate chain is valid')).toBeInTheDocument();
    });
});

describe('TlsCertSection — validation messages', () => {
    // The validation message is a state of its own: it replaces the
    // certificate block instead of stacking on top of it, so only one status
    // is ever visible.
    it('shows the mismatch message instead of the certificate block', () => {
        renderSection({ valid_pair: false, key_type: 'ECDSA' });

        expect(screen.getByText('Private key does not match certificate')).toBeInTheDocument();
        expect(screen.queryByText('Certificate chain is valid')).not.toBeInTheDocument();
        expect(screen.queryByText('Subject: example.org')).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });

    it('shows the warning message instead of the certificate block', () => {
        renderSection({ warning_validation: 'The certificate is not valid for example.org' });

        expect(
            screen.getByText('The certificate is not valid for example.org'),
        ).toBeInTheDocument();
        expect(screen.getByText('Certificate has issues')).toBeInTheDocument();
        expect(screen.queryByText('Certificate chain is valid')).not.toBeInTheDocument();
        expect(screen.queryByText('Subject: example.org')).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });

    it('shows the validation message alone when no certificate is configured', () => {
        renderSection({
            certificate_chain: '',
            certificate_path: '',
            warning_validation: 'The certificate is not valid for example.org',
        });

        expect(
            screen.getByText('The certificate is not valid for example.org'),
        ).toBeInTheDocument();
        expect(screen.queryByText('Certificate chain is valid')).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });
});

describe('TlsCertSection — no certificate', () => {
    it('renders no status block when there is neither a certificate nor a validation message', () => {
        renderSection({ certificate_chain: '', certificate_path: '' });

        expect(screen.getByTestId('tls-cert-section')).toBeInTheDocument();
        expect(screen.queryByText('Certificate chain is valid')).not.toBeInTheDocument();
        expect(algorithmLine()).not.toBeInTheDocument();
    });
});
