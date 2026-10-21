import { render } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { describe, expect, it, vi } from 'vitest';

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
    it('logs out in the current tab', () => {
        const { container } = renderMenu();
        const logoutLink = container.querySelector('#sign_out');

        expect(logoutLink).not.toBeNull();
        expect(logoutLink).toHaveAttribute('href', 'control/logout');
        expect(logoutLink).not.toHaveAttribute('target');
        expect(logoutLink).toHaveAttribute('rel', 'external');
    });
});
