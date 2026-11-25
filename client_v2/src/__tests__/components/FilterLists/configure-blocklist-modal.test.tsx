import { fireEvent, render, screen } from '@solidjs/testing-library';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ConfigureBlocklistModal } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { closeModal, openModal } from 'panel/stores/modals';
import { copyInDom } from 'panel/__tests__/helpers/copy';

// The catalog tab renders the whole blocklist catalog, which is irrelevant
// here and expensive to build in jsdom.
vi.mock('panel/components/FilterLists/blocks/ConfigureBlocklistModal/blocks/FiltersList', () => ({
    FiltersList: () => <div />,
}));

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

const typeInto = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), { target: { value } });

const openAddBlocklistManualTab = () => {
    openModal(MODAL_TYPE.ADD_BLOCKLIST);
    render(() => <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />);
    fireEvent.click(screen.getByRole('button', { name: copyInDom('blocklist_add_manual') }));
};

const openEditBlocklistModal = (
    filterToEdit: { name: string; url: string; enabled?: boolean } = blocklist,
) => {
    openModal(MODAL_TYPE.EDIT_BLOCKLIST);
    render(() => (
        <ConfigureBlocklistModal modalId={MODAL_TYPE.EDIT_BLOCKLIST} filterToEdit={filterToEdit} />
    ));
};

describe('ConfigureBlocklistModal', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        closeModal();
    });

    afterEach(() => {
        closeModal();
    });

    it('does not save a manual blocklist when the name is empty', () => {
        openAddBlocklistManualTab();

        typeInto(copyInDom('blocklist_url_file_path'), blocklist.url);
        fireEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiAddURL).not.toHaveBeenCalled();
    });

    it('does not save a manual blocklist when the url is empty', () => {
        openAddBlocklistManualTab();

        typeInto(copyInDom('name_label'), blocklist.name);
        fireEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiAddURL).not.toHaveBeenCalled();
    });

    it('saves a manual blocklist when both fields are filled', async () => {
        openAddBlocklistManualTab();

        typeInto(copyInDom('name_label'), blocklist.name);
        typeInto(copyInDom('blocklist_url_file_path'), blocklist.url);
        fireEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        await vi.waitFor(() =>
            expect(mocks.apiAddURL).toHaveBeenCalledWith({
                name: blocklist.name,
                url: blocklist.url,
                whitelist: false,
            }),
        );
    });

    it('does not save an edited blocklist when the name is empty', () => {
        openEditBlocklistModal();

        typeInto(copyInDom('name_label'), '');
        fireEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiSetURL).not.toHaveBeenCalled();
    });

    it('saves an edited blocklist keeping it enabled', async () => {
        openEditBlocklistModal();

        typeInto(copyInDom('name_label'), 'Renamed list');
        fireEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        await vi.waitFor(() =>
            expect(mocks.apiSetURL).toHaveBeenCalledWith({
                url: blocklist.url,
                data: {
                    name: 'Renamed list',
                    url: blocklist.url,
                    enabled: true,
                },
                whitelist: false,
            }),
        );
    });

    it('saves an edited blocklist keeping it disabled', async () => {
        openEditBlocklistModal({ ...blocklist, enabled: false });

        typeInto(copyInDom('name_label'), 'Renamed list');
        fireEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        await vi.waitFor(() =>
            expect(mocks.apiSetURL).toHaveBeenCalledWith({
                url: blocklist.url,
                data: {
                    name: 'Renamed list',
                    url: blocklist.url,
                    enabled: false,
                },
                whitelist: false,
            }),
        );
    });

    it('assumes an edited blocklist is enabled when its state is unknown', async () => {
        openEditBlocklistModal({ name: blocklist.name, url: blocklist.url });

        typeInto(copyInDom('name_label'), 'Renamed list');
        fireEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        await vi.waitFor(() =>
            expect(mocks.apiSetURL).toHaveBeenCalledWith({
                url: blocklist.url,
                data: {
                    name: 'Renamed list',
                    url: blocklist.url,
                    enabled: true,
                },
                whitelist: false,
            }),
        );
    });
});
