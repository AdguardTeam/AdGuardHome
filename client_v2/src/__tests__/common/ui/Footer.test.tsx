import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

const mockDashboardState = {
    dnsVersion: '',
    processingVersion: true,
    theme: 'light',
    language: 'en',
    name: '',
    checkUpdateFlag: true,
};

vi.mock('panel/stores/dashboard', () => ({
    get dashboardState() {
        return mockDashboardState;
    },
    getVersion: vi.fn(),
    changeTheme: vi.fn(),
    changeLanguage: vi.fn(),
}));

vi.mock('panel/common/intl', () => {
    const intl = {
        getMessage: (key: string, values?: any) => {
            const messages: Record<string, string> = {
                privacy_policy: 'Privacy Policy',
                report_an_issue: 'Report an issue',
                release_notes: 'Release notes',
                system_theme: 'System',
                dark_theme: 'Dark',
                light_theme: 'Light',
                version_number: `Version ${values?.value || ''}`,
                check_updates_btn: 'Check for updates',
            };
            return messages[key] || key;
        },
        getUILanguage: () => 'en',
        changeLanguage: vi.fn(),
    };
    return { default: intl };
});

vi.mock('panel/lib/theme', () => ({
    default: {
        link: { link: 'linkClass', noDecoration: 'noDecorationClass' },
        dropdown: {
            menu: 'dropdownMenu',
            item: 'dropdownItem',
            item_active: 'dropdownItemActive',
        },
    },
}));

import { Footer } from 'panel/common/ui/Footer';
import { getVersion } from 'panel/stores/dashboard';

describe('Footer', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockDashboardState.dnsVersion = '';
        mockDashboardState.processingVersion = true;
        mockDashboardState.checkUpdateFlag = true;
    });

    it('hides version badge when dnsVersion is empty', () => {
        mockDashboardState.dnsVersion = '';

        render(() => <Footer />);

        expect(screen.queryByText(/Version/)).not.toBeInTheDocument();
    });

    it('shows version badge when dnsVersion is populated', () => {
        mockDashboardState.dnsVersion = 'v1.0.0';

        render(() => <Footer />);

        expect(screen.getByText('Version v1.0.0')).toBeInTheDocument();
    });

    it('disables the check-updates button while processingVersion is true', () => {
        mockDashboardState.dnsVersion = 'v1.0.0';
        mockDashboardState.processingVersion = true;

        render(() => <Footer />);

        const button = screen.getByTestId('footer-check-updates');
        expect(button).toBeDisabled();
    });

    it('enables the check-updates button when processingVersion is false', () => {
        mockDashboardState.dnsVersion = 'v1.0.0';
        mockDashboardState.processingVersion = false;

        render(() => <Footer />);

        const button = screen.getByTestId('footer-check-updates');
        expect(button).not.toBeDisabled();
    });

    it('calls getVersion(true) when check-updates button is clicked', async () => {
        const user = userEvent.setup();
        mockDashboardState.dnsVersion = 'v1.0.0';
        mockDashboardState.processingVersion = false;

        render(() => <Footer />);

        const button = screen.getByTestId('footer-check-updates');
        await user.click(button);

        expect(getVersion).toHaveBeenCalledWith(true);
    });

    it('has aria-label on the check-updates button', () => {
        mockDashboardState.dnsVersion = 'v1.0.0';

        render(() => <Footer />);

        const button = screen.getByTestId('footer-check-updates');
        expect(button.getAttribute('aria-label')).toBe('Check for updates');
    });

    it('hides check-updates button when checkUpdateFlag is false (Docker/Snap)', () => {
        mockDashboardState.dnsVersion = 'v1.0.0';
        mockDashboardState.processingVersion = false;
        mockDashboardState.checkUpdateFlag = false;

        render(() => <Footer />);

        expect(screen.queryByTestId('footer-check-updates')).not.toBeInTheDocument();
        // Version text is still visible
        expect(screen.getByText('Version v1.0.0')).toBeInTheDocument();
    });
});
