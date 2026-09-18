import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

import { copy } from 'panel/__tests__/helpers/copy';

const mockDashboardState = {
    dnsVersion: '',
    processingVersion: true,
    theme: 'light' as string | undefined,
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

// The footer renders localized copy; serve it from the base locale so the
// assertions can name the key instead of re-stating the text.
vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

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

// Replace the Ark-UI popover with a deterministic trigger/menu so tests can
// inspect the theme menu items and their active state.
vi.mock('panel/common/ui/Dropdown', () => ({
    Dropdown: (props: any) => (
        <>
            <button
                data-testid="dropdown-trigger"
                onClick={() => props.onOpenChange?.(!props.open)}
            >
                {props.children}
            </button>
            {props.open && <div data-testid="dropdown-menu">{props.menu}</div>}
        </>
    ),
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

        expect(
            screen.getByText(copy('version_number', { value: 'v1.0.0' })),
        ).toBeInTheDocument();
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
        expect(button.getAttribute('aria-label')).toBe(copy('check_updates_btn'));
    });

    it('hides check-updates button when checkUpdateFlag is false (Docker/Snap)', () => {
        mockDashboardState.dnsVersion = 'v1.0.0';
        mockDashboardState.processingVersion = false;
        mockDashboardState.checkUpdateFlag = false;

        render(() => <Footer />);

        expect(screen.queryByTestId('footer-check-updates')).not.toBeInTheDocument();
        // Version text is still visible
        expect(
            screen.getByText(copy('version_number', { value: 'v1.0.0' })),
        ).toBeInTheDocument();
    });

    it('highlights the current theme item instead of always System when not logged in', async () => {
        const user = userEvent.setup();
        localStorage.clear();
        mockDashboardState.name = '';
        mockDashboardState.theme = undefined;

        render(() => <Footer />);

        // Nothing stored, so the applied theme is Light. The trigger shows it.
        await user.click(screen.getByText(copy('light_theme')));

        const menu = () => within(screen.getByTestId('dropdown-menu'));

        // Light must be highlighted — not System.
        expect(menu().getByRole('button', { name: copy('light_theme') })).toHaveClass(
            'dropdownItemActive',
        );
        expect(menu().getByRole('button', { name: copy('system_theme') })).not.toHaveClass(
            'dropdownItemActive',
        );
        expect(menu().getByRole('button', { name: copy('dark_theme') })).not.toHaveClass(
            'dropdownItemActive',
        );

        // Choose Dark — the menu closes and the trigger updates.
        await user.click(menu().getByRole('button', { name: copy('dark_theme') }));
        await user.click(screen.getByText(copy('dark_theme')));

        // Dark must now be highlighted, not System.
        expect(menu().getByRole('button', { name: copy('dark_theme') })).toHaveClass(
            'dropdownItemActive',
        );
        expect(menu().getByRole('button', { name: copy('system_theme') })).not.toHaveClass(
            'dropdownItemActive',
        );
        expect(menu().getByRole('button', { name: copy('light_theme') })).not.toHaveClass(
            'dropdownItemActive',
        );
    });

    it('shows the persisted theme as active when not logged in', async () => {
        const user = userEvent.setup();
        localStorage.setItem('account_theme', JSON.stringify('dark'));
        mockDashboardState.name = '';
        mockDashboardState.theme = undefined;

        render(() => <Footer />);

        // The trigger reflects the persisted theme.
        expect(screen.getByText(copy('dark_theme'))).toBeInTheDocument();

        await user.click(screen.getByText(copy('dark_theme')));

        const menu = () => within(screen.getByTestId('dropdown-menu'));
        expect(menu().getByRole('button', { name: copy('dark_theme') })).toHaveClass(
            'dropdownItemActive',
        );
        expect(menu().getByRole('button', { name: copy('system_theme') })).not.toHaveClass(
            'dropdownItemActive',
        );
    });
});
