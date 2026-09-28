import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

// Deep proxy: `theme` is written as theme.layout.container, theme.title.h4, etc.
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
            enabled: false,
            processingConfig: false,
            processingValidate: false,
            certificate_chain: '',
            certificate_path: '',
            private_key: '',
            private_key_path: '',
            private_key_saved: false,
            server_name: '',
            serve_plain_dns: true,
            force_https: false,
            port_https: 443,
            port_dns_over_tls: 853,
            port_dns_over_quic: 853,
            port_dnscrypt: 0,
        } as Record<string, unknown>,
        mocks: {
            getTlsStatus: vi.fn(),
            setTlsConfig: vi.fn(),
            validateTlsConfig: vi.fn(),
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

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: { httpPort: 80, dnsPort: 53 },
}));

// The page reads the `tlsWizard` query param to auto-open the wizard; no
// params here, and `setSearchParams` must be a no-op.
vi.mock('@solidjs/router', () => ({
    useSearchParams: () => [{}, vi.fn()],
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

// The child blocks have their own tests — stub them so this test only covers
// the page's switch logic.  Each stub reports its `open` state through
// `data-open`, which is what the assertions read.
vi.mock('panel/components/Encryption/blocks/PlainDnsToggle', () => ({
    PlainDnsToggle: () => <div data-testid="plain-dns-toggle" />,
}));
vi.mock('panel/components/Encryption/blocks/TlsCertSection', () => ({
    TlsCertSection: () => <div data-testid="tls-cert-section" />,
}));
vi.mock('panel/components/Encryption/blocks/ServerSettingsRow', () => ({
    ServerSettingsRow: () => <div data-testid="server-settings-row" />,
}));
vi.mock('panel/components/Encryption/blocks/RedirectToggle', () => ({
    RedirectToggle: () => <div data-testid="redirect-toggle" />,
}));
vi.mock('panel/components/Encryption/blocks/ResetDnsModal', () => ({
    ResetDnsModal: (props: { open: boolean }) => (
        <div data-testid="reset-dns-modal" data-open={String(props.open)} />
    ),
}));
vi.mock('panel/components/Encryption/blocks/ServerSettingsModal', () => ({
    ServerSettingsModal: (props: { open: boolean }) => (
        <div data-testid="server-settings-modal" data-open={String(props.open)} />
    ),
}));
vi.mock('panel/components/Encryption/blocks/SetupWizard', () => ({
    TlsSetupWizard: (props: { open: boolean }) => (
        <div data-testid="tls-wizard" data-open={String(props.open)} />
    ),
}));

import { Encryption } from 'panel/components/Encryption/Encryption';

const renderPage = async () => {
    render(() => <Encryption />);

    // The switch only renders after `getTlsStatus` resolves.
    const switchInput = await waitFor(() => screen.getByRole('checkbox'));

    return switchInput;
};

const openState = (testId: string) => screen.getByTestId(testId).getAttribute('data-open');

beforeEach(() => {
    vi.clearAllMocks();
    Object.assign(encryptionState, {
        enabled: false,
        processingConfig: false,
        certificate_chain: '',
        certificate_path: '',
        private_key: '',
        private_key_path: '',
        private_key_saved: false,
        server_name: '',
    });
    mocks.getTlsStatus.mockResolvedValue(undefined);
    mocks.setTlsConfig.mockResolvedValue({ ok: true });
});

describe('Encryption page — encrypted DNS switch', () => {
    it('enables encryption when a certificate and key are set, even without a server name', async () => {
        encryptionState.certificate_path = '/etc/certs/fullchain.pem';
        encryptionState.private_key_path = '/etc/certs/privkey.pem';
        encryptionState.server_name = '';

        const switchInput = await renderPage();
        await userEvent.click(switchInput);

        expect(mocks.setTlsConfig).toHaveBeenCalledWith({ enabled: true });
        expect(openState('server-settings-modal')).toBe('false');
        expect(openState('tls-wizard')).toBe('false');
    });

    it('enables encryption when a server name is set as well', async () => {
        encryptionState.certificate_path = '/etc/certs/fullchain.pem';
        encryptionState.private_key_path = '/etc/certs/privkey.pem';
        encryptionState.server_name = 'dns.example.com';

        const switchInput = await renderPage();
        await userEvent.click(switchInput);

        expect(mocks.setTlsConfig).toHaveBeenCalledWith({ enabled: true });
        expect(openState('server-settings-modal')).toBe('false');
        expect(openState('tls-wizard')).toBe('false');
    });

    it('opens the TLS wizard without saving when the key is missing', async () => {
        encryptionState.certificate_path = '/etc/certs/fullchain.pem';
        encryptionState.private_key_path = '';

        const switchInput = await renderPage();
        await userEvent.click(switchInput);

        expect(mocks.setTlsConfig).not.toHaveBeenCalled();
        expect(openState('tls-wizard')).toBe('true');
        expect(openState('server-settings-modal')).toBe('false');
        expect(screen.getByRole('checkbox')).not.toBeChecked();
    });
});
