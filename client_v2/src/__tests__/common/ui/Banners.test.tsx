import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { HashRouter, Route } from '@solidjs/router';

// We'll mock the stores at the module level and update them per test
const mockDashboardState = {
    isUpdateAvailable: false,
    newVersion: '',
    announcementUrl: '',
    canAutoUpdate: false,
    processingUpdate: false,
    processingVersion: false,
    dnsVersion: '',
};

const mockEncryptionState = {
    enabled: false,
    not_after: '',
    valid_cert: false,
};

vi.mock('panel/stores/dashboard', () => ({
    get dashboardState() {
        return mockDashboardState;
    },
    getUpdate: vi.fn(),
    getVersion: vi.fn(),
}));

vi.mock('panel/stores/encryption', () => ({
    get encryptionState() {
        return mockEncryptionState;
    },
}));

vi.mock('panel/common/intl', () => {
    const intl = {
        getMessage: (key: string, values?: any) => {
            const messages: Record<string, string> = {
                tls_certificate_expired: 'Your TLS certificate has expired',
                tls_certificate_expiring: 'Your TLS certificate is about to expire',
                update_available: `Version ${values?.version || ''} is available. Release notes`,
                update_button: 'Update',
                update_how_to: 'How to update',
                version_number: `Version ${values?.value || ''}`,
                check_updates_btn: 'Check for updates',
            };
            const msg = messages[key] || key;
            if (values?.a) {
                // When tag handlers are provided, return the full message
                return `Version ${values.version} is available. Release notes`;
            }
            return msg;
        },
        getUILanguage: () => 'en',
        changeLanguage: vi.fn(),
    };
    return { default: intl };
});

import { Banners } from 'panel/common/ui/Banners';
import { BANNER_TEST_VALUES } from 'panel/helpers/banners';

const renderBanners = (props?: { forceBanner?: any }) => {
    return render(() => (
        <HashRouter>
            <Route path="/" component={() => <Banners {...props} />} />
        </HashRouter>
    ));
};

const resetStores = () => {
    mockDashboardState.isUpdateAvailable = false;
    mockDashboardState.newVersion = '';
    mockDashboardState.announcementUrl = '';
    mockDashboardState.canAutoUpdate = false;
    mockDashboardState.processingUpdate = false;
    mockDashboardState.processingVersion = false;
    mockDashboardState.dnsVersion = '';
    mockEncryptionState.enabled = false;
    mockEncryptionState.not_after = '';
    mockEncryptionState.valid_cert = false;
};

describe('Banners', () => {
    beforeEach(() => {
        resetStores();
    });

    // ── Priority logic cases ──

    it('shows TLS expired banner when cert is expired', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() - 86400000).toISOString(); // 1 day ago

        renderBanners();

        expect(screen.getByTestId('banner-tls-expired')).toBeInTheDocument();
        expect(screen.getByText('Your TLS certificate has expired')).toBeInTheDocument();
    });

    it('shows TLS expiring banner when cert expires within 30 days', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() + 15 * 86400000).toISOString(); // 15 days

        renderBanners();

        expect(screen.getByTestId('banner-tls-expiring')).toBeInTheDocument();
        expect(screen.getByText('Your TLS certificate is about to expire')).toBeInTheDocument();
    });

    it('shows auto-update banner when update available and can auto-update', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() + 60 * 86400000).toISOString(); // 60 days
        mockDashboardState.isUpdateAvailable = true;
        mockDashboardState.newVersion = 'v1.0.0';
        mockDashboardState.canAutoUpdate = true;

        renderBanners();

        expect(screen.getByTestId('banner-update-auto')).toBeInTheDocument();
    });

    it('shows manual-update banner when update available but cannot auto-update', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() + 60 * 86400000).toISOString();
        mockDashboardState.isUpdateAvailable = true;
        mockDashboardState.newVersion = 'v1.0.0';
        mockDashboardState.canAutoUpdate = false;

        renderBanners();

        expect(screen.getByTestId('banner-update-manual')).toBeInTheDocument();
    });

    it('shows auto-update banner when TLS not enabled', () => {
        mockDashboardState.isUpdateAvailable = true;
        mockDashboardState.newVersion = 'v1.0.0';
        mockDashboardState.canAutoUpdate = true;

        renderBanners();

        expect(screen.getByTestId('banner-update-auto')).toBeInTheDocument();
    });

    it('falls through to update when not_after is invalid (NaN)', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = 'not-a-date';
        mockDashboardState.isUpdateAvailable = true;
        mockDashboardState.newVersion = 'v1.0.0';
        mockDashboardState.canAutoUpdate = true;

        renderBanners();

        expect(screen.getByTestId('banner-update-auto')).toBeInTheDocument();
    });

    it('renders nothing when no conditions met', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() + 60 * 86400000).toISOString();

        renderBanners();

        expect(screen.queryByTestId('banner-root')).not.toBeInTheDocument();
    });

    it('TLS expired takes priority over update available', () => {
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() - 86400000).toISOString(); // expired
        mockDashboardState.isUpdateAvailable = true;
        mockDashboardState.newVersion = 'v1.0.0';
        mockDashboardState.canAutoUpdate = true;

        renderBanners();

        // Only TLS expired banner should appear
        expect(screen.getByTestId('banner-tls-expired')).toBeInTheDocument();
        expect(screen.queryByTestId('banner-update-auto')).not.toBeInTheDocument();
    });

    // ── Dismiss cases ──

    it('hides the banner after clicking the close button', async () => {
        const user = userEvent.setup();
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() - 86400000).toISOString(); // expired

        renderBanners();

        expect(screen.getByTestId('banner-tls-expired')).toBeInTheDocument();
        const closeButton = screen.getByTestId('banner-tls-expired-close');
        await user.click(closeButton);

        expect(screen.queryByTestId('banner-tls-expired')).not.toBeInTheDocument();
        expect(screen.queryByTestId('banner-root')).not.toBeInTheDocument();
    });

    it('re-shows the banner when the underlying condition changes', async () => {
        const user = userEvent.setup();
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() - 86400000).toISOString(); // expired

        renderBanners();

        // Dismiss the expired banner
        const closeButton = screen.getByTestId('banner-tls-expired-close');
        await user.click(closeButton);
        expect(screen.queryByTestId('banner-tls-expired')).not.toBeInTheDocument();

        // Change condition: now cert is valid but expiring soon
        mockEncryptionState.not_after = new Date(Date.now() + 15 * 86400000).toISOString(); // 15 days
        // Re-render
        renderBanners();

        // Should now show the expiring banner
        expect(screen.getByTestId('banner-tls-expiring')).toBeInTheDocument();
        expect(screen.queryByTestId('banner-tls-expired')).not.toBeInTheDocument();
    });

    // ── forceBanner (dev test override) ──

    it('renders forced TLS expired banner regardless of store state', () => {
        renderBanners({ forceBanner: BANNER_TEST_VALUES.tlsExpired });
        expect(screen.getByTestId('banner-tls-expired')).toBeInTheDocument();
    });

    it('renders forced TLS expiring banner regardless of store state', () => {
        renderBanners({ forceBanner: BANNER_TEST_VALUES.tlsExpiring });
        expect(screen.getByTestId('banner-tls-expiring')).toBeInTheDocument();
    });

    it('renders forced auto-update banner', () => {
        renderBanners({ forceBanner: BANNER_TEST_VALUES.updateAuto });
        expect(screen.getByTestId('banner-update-auto')).toBeInTheDocument();
    });

    it('renders forced manual-update banner', () => {
        renderBanners({ forceBanner: BANNER_TEST_VALUES.updateManual });
        expect(screen.getByTestId('banner-update-manual')).toBeInTheDocument();
    });

    it('forceBanner overrides real banner conditions', () => {
        // Set up a real TLS expired condition
        mockEncryptionState.enabled = true;
        mockEncryptionState.valid_cert = true;
        mockEncryptionState.not_after = new Date(Date.now() - 86400000).toISOString();

        // But force auto-update banner instead
        renderBanners({ forceBanner: BANNER_TEST_VALUES.updateAuto });

        expect(screen.getByTestId('banner-update-auto')).toBeInTheDocument();
        expect(screen.queryByTestId('banner-tls-expired')).not.toBeInTheDocument();
    });
});
