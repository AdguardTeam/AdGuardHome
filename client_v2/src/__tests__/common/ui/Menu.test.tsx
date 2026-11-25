import { render } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
    dashboardState: {
        processingProfile: false,
        profileFailed: false,
        name: 'admin',
    },
}));

vi.mock('panel/stores/dashboard', () => ({ dashboardState: mocks.dashboardState }));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { Menu } from 'panel/common/ui/Menu';

const renderMenu = () => {
    const history = createMemoryHistory();
    history.set({ value: '/' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route
                path="/*"
                component={() => <Menu accountSubMenu={false} setAccountSubMenu={vi.fn()} />}
            />
        </MemoryRouter>
    ));
};

describe('Menu', () => {
    beforeEach(() => {
        mocks.dashboardState.processingProfile = false;
        mocks.dashboardState.profileFailed = false;
        mocks.dashboardState.name = 'admin';
    });

    it('logs out in the current tab', () => {
        const { container } = renderMenu();
        const logoutLink = container.querySelector('#sign_out');

        expect(logoutLink).not.toBeNull();
        expect(logoutLink).toHaveAttribute('href', 'control/logout');
        expect(logoutLink).not.toHaveAttribute('target');
        expect(logoutLink).toHaveAttribute('rel', 'external');
    });

    it('hides the logout link when authentication is disabled', () => {
        mocks.dashboardState.name = '';

        const { container } = renderMenu();

        expect(container.querySelector('#sign_out')).toBeNull();
    });

    it('hides the logout link until the profile is loaded', () => {
        mocks.dashboardState.processingProfile = true;

        const { container } = renderMenu();

        expect(container.querySelector('#sign_out')).toBeNull();
    });

    it('keeps the logout link when the profile could not be loaded', () => {
        mocks.dashboardState.profileFailed = true;
        mocks.dashboardState.name = '';

        const { container } = renderMenu();

        expect(container.querySelector('#sign_out')).not.toBeNull();
    });
});
