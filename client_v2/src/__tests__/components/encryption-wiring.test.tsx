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

import { Encryption } from 'panel/components/Encryption/Encryption';

const statusNoCert = {
    enabled: false,
    serve_plain_dns: true,
    force_https: false,
    server_name: '',
    port_https: 0,
    port_dns_over_tls: 0,
    port_dns_over_quic: 0,
    certificate_chain: '',
    certificate_path: '',
    private_key: '',
    private_key_path: '',
    private_key_saved: false,
    warning_validation: '',
};

describe('Encryption — TLS setup wizard wiring', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.tlsStatus.mockResolvedValue(statusNoCert);
    });

    it('opens the wizard via the plus button when no certificate is configured', async () => {
        const user = userEvent.setup();
        render(() => <Encryption />);

        await waitFor(() => {
            expect(screen.getByText('Add TLS certificate')).toBeInTheDocument();
        });

        await user.click(screen.getByText('Add TLS certificate'));

        expect(await screen.findByText('Add certificate')).toBeInTheDocument();
        expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '1');
    });

    it('opens the wizard when the Encrypted DNS switch is turned on without a cert', async () => {
        const user = userEvent.setup();
        render(() => <Encryption />);

        await waitFor(() => {
            expect(screen.getByText('Encrypted DNS')).toBeInTheDocument();
        });

        const switchInput = document.getElementById('encrypted_dns') as HTMLInputElement;
        expect(switchInput).not.toBeNull();
        await user.click(switchInput);

        expect(await screen.findByText('Add certificate')).toBeInTheDocument();
    });

    it('cancelling the wizard leaves the switch off', async () => {
        const user = userEvent.setup();
        render(() => <Encryption />);

        await waitFor(() => {
            expect(screen.getByText('Encrypted DNS')).toBeInTheDocument();
        });

        const switchInput = document.getElementById('encrypted_dns') as HTMLInputElement;
        await user.click(switchInput);

        expect(await screen.findByText('Add certificate')).toBeInTheDocument();

        // Cancel (secondary button in the wizard footer).
        const cancelButton = screen.getByRole('button', { name: 'Cancel' });
        await user.click(cancelButton);

        await waitFor(() => {
            expect(switchInput.checked).toBe(false);
        });
        expect(mocks.tlsConfigure).not.toHaveBeenCalled();
    });
});
