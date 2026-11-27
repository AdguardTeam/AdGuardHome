import { fireEvent, render, screen, waitFor, within } from '@solidjs/testing-library';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
    accessList: vi.fn(),
    accessSet: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    accessList: mocks.accessList,
    accessSet: mocks.accessSet,
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { Access } from 'panel/components/DnsSettings/Access';
import { accessState, getAccessList } from 'panel/stores/access';
import { copyInDom, type CopyKey } from 'panel/__tests__/helpers/copy';

const SERVER_LISTS = {
    allowed_clients: ['10.0.0.5'],
    disallowed_clients: ['10.0.0.6'],
    blocked_hosts: ['blocked.example'],
};

const row = (title: CopyKey): HTMLElement => {
    const label = screen
        .getAllByText(copyInDom(title))
        .find((element) => !element.closest('[role="dialog"]'));
    const element = label?.closest('[role="button"]');

    if (!(element instanceof HTMLElement)) {
        throw new Error(`Access row not found: ${title}`);
    }

    return element;
};

const dialog = (title: CopyKey): HTMLElement | null => {
    const label = screen
        .getAllByText(copyInDom(title))
        .find((element) => element.closest('[role="dialog"]'));
    const element = label?.closest('[role="dialog"]');

    return element instanceof HTMLElement ? element : null;
};

const isDialogOpen = (title: CopyKey): boolean =>
    dialog(title)?.getAttribute('data-state') === 'open';

const editor = (id: string): HTMLTextAreaElement => {
    const element = document.querySelector(`textarea#${id}`);

    if (!(element instanceof HTMLTextAreaElement)) {
        throw new Error(`Access editor not found: ${id}`);
    }

    return element;
};

describe('Access', () => {
    beforeEach(async () => {
        vi.resetAllMocks();
        // Load empty lists through the store setter so every test starts from a
        // loaded, failed-read-free state.
        mocks.accessList.mockResolvedValue({
            allowed_clients: [],
            disallowed_clients: [],
            blocked_hosts: [],
        });
        await getAccessList();
    });

    it('seeds the editor from the server after a failed initial read', async () => {
        mocks.accessList.mockRejectedValueOnce(new Error('network'));
        await getAccessList();
        expect(accessState.allowed_clients).toBe('');

        render(() => <Access />);

        mocks.accessList.mockResolvedValue(SERVER_LISTS);
        fireEvent.click(row('dns_allowed_clients'));

        await waitFor(() => expect(isDialogOpen('dns_allowed_clients')).toBe(true));
        expect(editor('allowed_clients')).toHaveValue('10.0.0.5');
    });

    it('does not open the editor when the refresh fails', async () => {
        mocks.accessList.mockRejectedValueOnce(new Error('network'));
        await getAccessList();

        render(() => <Access />);

        mocks.accessList.mockRejectedValue(new Error('network'));
        fireEvent.click(row('dns_allowed_clients'));

        await waitFor(() => expect(mocks.addErrorToast).toHaveBeenCalledTimes(2));
        expect(isDialogOpen('dns_allowed_clients')).toBe(false);
        expect(editor('allowed_clients')).toHaveValue('');
    });

    it('keeps the editor open with the edit when the save fails', async () => {
        mocks.accessList.mockResolvedValue(SERVER_LISTS);

        render(() => <Access />);

        fireEvent.click(row('dns_allowed_clients'));
        await waitFor(() => expect(isDialogOpen('dns_allowed_clients')).toBe(true));

        fireEvent.change(editor('allowed_clients'), { target: { value: '1.1.1.1' } });

        mocks.accessList.mockRejectedValue(new Error('network'));

        fireEvent.click(
            within(dialog('dns_allowed_clients') as HTMLElement).getByTestId('config-dialog-save'),
        );

        await waitFor(() => expect(mocks.addErrorToast).toHaveBeenCalled());
        expect(mocks.accessSet).not.toHaveBeenCalled();
        expect(isDialogOpen('dns_allowed_clients')).toBe(true);
        expect(editor('allowed_clients')).toHaveValue('1.1.1.1');
    });
});
