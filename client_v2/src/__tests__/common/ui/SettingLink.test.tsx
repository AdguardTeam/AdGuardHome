import { render, screen } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { describe, expect, it } from 'vitest';

import { SettingLink } from 'panel/common/ui/SettingLink';
import { RoutePath } from 'panel/components/Routes/Paths';

const renderLink = (props: { disabled?: boolean } = {}) => {
    const history = createMemoryHistory();
    history.set({ value: '/' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route
                path="/"
                component={() => (
                    <SettingLink
                        to={RoutePath.Clients}
                        title="Title"
                        description="Description"
                        disabled={props.disabled}
                        data-testid="row"
                    />
                )}
            />
        </MemoryRouter>
    ));
};

describe('SettingLink', () => {
    it('renders the title and description', () => {
        renderLink();

        expect(screen.getByText('Title')).toBeInTheDocument();
        expect(screen.getByText('Description')).toBeInTheDocument();
    });

    it('renders a router link when enabled', () => {
        renderLink();

        const row = screen.getByTestId('row');
        expect(row.tagName).toBe('A');
        expect(row).toHaveAttribute('href');
    });

    it('renders a non-interactive row with the arrow when disabled', () => {
        renderLink({ disabled: true });

        const row = screen.getByTestId('row');
        expect(row.tagName).toBe('DIV');
        expect(row.querySelector('svg')).not.toBeNull();
        expect(screen.queryByRole('link')).toBeNull();
    });
});
