import { render, screen, fireEvent } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';

const mocks = vi.hoisted(() => ({
    accessList: vi.fn(),
    accessSet: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    accessList: mocks.accessList,
    accessSet: mocks.accessSet,
}));

import { TopClients } from 'panel/components/Dashboard/blocks/TopClients';
import { getAccessList } from 'panel/stores/access';
import { CLIENT_BLOCK_CONFIRM_SUBMIT_TEST_ID } from 'panel/common/ui/ClientBlockConfirm';
import theme from 'panel/lib/theme';
import { copyInDom } from 'panel/__tests__/helpers/copy';
import { mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

const CLIENT = '10.0.0.1';

const renderCard = () =>
    render(() => (
        <HashRouter>
            <Route
                path="/"
                component={() => (
                    <TopClients topClients={[{ name: CLIENT, count: 5 }]} numDnsQueries={100} />
                )}
            />
        </HashRouter>
    ));

/** The inline mobile action; the closed desktop dropdown renders a second copy. */
const actionButton = (key: 'block_client' | 'unblock_client') =>
    screen.getAllByRole('button', { name: copyInDom(key) })[0];

const loadAccessList = (disallowed: string[]) => {
    mocks.accessList.mockResolvedValue({
        allowed_clients: [],
        disallowed_clients: disallowed,
        blocked_hosts: [],
    });

    return getAccessList();
};

describe('Dashboard top clients block/unblock action', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockMatchMedia(false);
    });

    it('renders the block action as a button without the dropdown item background', async () => {
        await loadAccessList([]);
        renderCard();

        const button = actionButton('block_client');
        expect(button.tagName).toBe('BUTTON');
        expect(button.getAttribute('type')).toBe('button');
        // The dropdown row class paints `--default-dropdown-menu-background`.
        expect(theme.dropdown.item).toBeTruthy();
        expect(button.classList.contains(theme.dropdown.item)).toBe(false);
        // Red is the destructive block action.
        expect(button.className).toMatch(/tableRowMenuItemRed/);
    });

    it('renders the unblock action as a button without the dropdown item background', async () => {
        await loadAccessList([CLIENT]);
        renderCard();

        const button = actionButton('unblock_client');
        expect(button.tagName).toBe('BUTTON');
        expect(button.getAttribute('type')).toBe('button');
        expect(button.classList.contains(theme.dropdown.item)).toBe(false);
        // Green is the restoring unblock action.
        expect(button.className).toMatch(/tableRowMenuItemGreen/);
    });

    it('opens the confirm dialog from the inline action button', async () => {
        await loadAccessList([]);
        renderCard();

        await userEvent.click(actionButton('block_client'));
        expect(screen.getByTestId(CLIENT_BLOCK_CONFIRM_SUBMIT_TEST_ID)).toBeInTheDocument();
    });

    it('switches the action when the client becomes blocked', async () => {
        await loadAccessList([]);
        renderCard();
        expect(actionButton('block_client')).toBeInTheDocument();

        await loadAccessList([CLIENT]);

        expect(actionButton('unblock_client')).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: copyInDom('block_client') })).toBeNull();
    });

    it('does not scroll the page to the top when the action opens the dialog', async () => {
        await loadAccessList([]);
        renderCard();

        const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
        vi.useFakeTimers();
        try {
            fireEvent.click(actionButton('block_client'));

            // The row link scrolls to the top 100ms after a click.
            vi.advanceTimersByTime(150);

            expect(screen.getByTestId(CLIENT_BLOCK_CONFIRM_SUBMIT_TEST_ID)).toBeInTheDocument();
            expect(scrollTo).not.toHaveBeenCalled();
        } finally {
            vi.useRealTimers();
            scrollTo.mockRestore();
        }
    });
});
