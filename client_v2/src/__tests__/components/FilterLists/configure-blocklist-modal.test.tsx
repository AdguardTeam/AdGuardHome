import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ConfigureBlocklistModal } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { closeModal, openModal } from 'panel/stores/modals';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const mocks = vi.hoisted(() => ({
    apiAddURL: vi.fn(() => Promise.resolve(undefined)),
    apiSetURL: vi.fn(() => Promise.resolve(undefined)),
    apiStatus: vi.fn(() =>
        Promise.resolve({
            enabled: true,
            interval: 24,
            filters: [],
            whitelist_filters: [],
            user_rules: [],
        }),
    ),
}));

vi.mock('panel/api/generated', () => ({
    filteringStatus: mocks.apiStatus,
    filteringAddURL: mocks.apiAddURL,
    filteringSetRules: vi.fn(),
    filteringRemoveURL: vi.fn(),
    filteringSetURL: mocks.apiSetURL,
    filteringRefresh: vi.fn(),
    filteringConfig: vi.fn(),
    filteringCheckHost: vi.fn(),
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

const blocklist = {
    name: 'My blocklist',
    url: 'https://example.org/list.txt',
    enabled: true,
};

describe('ConfigureBlocklistModal', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        closeModal();
    });

    afterEach(() => {
        closeModal();
    });

    it('does not save a manual blocklist when the name is empty', async () => {
        openModal(MODAL_TYPE.ADD_BLOCKLIST);
        render(() => <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />);

        await userEvent.click(
            screen.getByRole('button', { name: copyInDom('blocklist_add_manual') }),
        );
        await userEvent.type(
            screen.getByLabelText(copyInDom('blocklist_url_file_path')),
            'https://example.org/list.txt',
        );
        await userEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiAddURL).not.toHaveBeenCalled();
    });

    it('does not save a manual blocklist when the url is empty', async () => {
        openModal(MODAL_TYPE.ADD_BLOCKLIST);
        render(() => <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />);

        await userEvent.click(
            screen.getByRole('button', { name: copyInDom('blocklist_add_manual') }),
        );
        await userEvent.type(screen.getByLabelText(copyInDom('name_label')), 'My blocklist');
        await userEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiAddURL).not.toHaveBeenCalled();
    });

    it('saves a manual blocklist when both fields are filled', async () => {
        openModal(MODAL_TYPE.ADD_BLOCKLIST);
        render(() => <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />);

        await userEvent.click(
            screen.getByRole('button', { name: copyInDom('blocklist_add_manual') }),
        );
        await userEvent.type(screen.getByLabelText(copyInDom('name_label')), blocklist.name);
        await userEvent.type(
            screen.getByLabelText(copyInDom('blocklist_url_file_path')),
            blocklist.url,
        );
        await userEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(mocks.apiAddURL).toHaveBeenCalledWith({
            name: blocklist.name,
            url: blocklist.url,
            whitelist: false,
        });
    });

    it('does not save an edited blocklist when the name is empty', async () => {
        openModal(MODAL_TYPE.EDIT_BLOCKLIST);
        render(() => (
            <ConfigureBlocklistModal modalId={MODAL_TYPE.EDIT_BLOCKLIST} filterToEdit={blocklist} />
        ));

        await userEvent.clear(screen.getByLabelText(copyInDom('name_label')));
        await userEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiSetURL).not.toHaveBeenCalled();
    });

    it('saves an edited blocklist keeping its enabled state', async () => {
        openModal(MODAL_TYPE.EDIT_BLOCKLIST);
        render(() => (
            <ConfigureBlocklistModal modalId={MODAL_TYPE.EDIT_BLOCKLIST} filterToEdit={blocklist} />
        ));

        await userEvent.type(screen.getByLabelText(copyInDom('name_label')), ' renamed');
        await userEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        expect(mocks.apiSetURL).toHaveBeenCalledWith({
            url: blocklist.url,
            data: {
                name: `${blocklist.name} renamed`,
                url: blocklist.url,
                enabled: true,
            },
            whitelist: false,
        });
    });
});
