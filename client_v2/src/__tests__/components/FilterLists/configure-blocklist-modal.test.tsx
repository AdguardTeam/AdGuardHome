import { render, screen, fireEvent } from '@solidjs/testing-library';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { ConfigureBlocklistModal } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal/ConfigureBlocklistModal';
import { openModal } from 'panel/stores/modals';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const mocks = vi.hoisted(() => ({
    apiSetURL: vi.fn(() => Promise.resolve(undefined)),
    apiGetFilteringStatus: vi.fn(() => Promise.resolve({})),
}));

vi.mock('panel/api/generated', () => ({
    filteringSetURL: mocks.apiSetURL,
    filteringStatus: mocks.apiGetFilteringStatus,
}));

vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: vi.fn(),
    addErrorToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

const filterUrl = 'https://example.org/list.txt';

const openEditBlocklistModal = async (filterToEdit: {
    url: string;
    name: string;
    enabled?: boolean;
}) => {
    openModal(MODAL_TYPE.EDIT_BLOCKLIST);
    render(() => (
        <ConfigureBlocklistModal
            modalId={MODAL_TYPE.EDIT_BLOCKLIST}
            filterToEdit={filterToEdit}
        />
    ));

    fireEvent.change(screen.getByLabelText(copyInDom('name_label')), {
        target: { value: 'Renamed list' },
    });
    fireEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

    await vi.waitFor(() => expect(mocks.apiSetURL).toHaveBeenCalled());
};

describe('ConfigureBlocklistModal rename', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('keeps an enabled blocklist enabled', async () => {
        await openEditBlocklistModal({
            url: filterUrl,
            name: 'Example list',
            enabled: true,
        });

        expect(mocks.apiSetURL).toHaveBeenCalledWith({
            url: filterUrl,
            data: { name: 'Renamed list', url: filterUrl, enabled: true },
            whitelist: false,
        });
    });

    it('keeps a disabled blocklist disabled', async () => {
        await openEditBlocklistModal({
            url: filterUrl,
            name: 'Example list',
            enabled: false,
        });

        expect(mocks.apiSetURL).toHaveBeenCalledWith({
            url: filterUrl,
            data: { name: 'Renamed list', url: filterUrl, enabled: false },
            whitelist: false,
        });
    });

    it('assumes enabled when the filter state is unknown', async () => {
        await openEditBlocklistModal({ url: filterUrl, name: 'Example list' });

        expect(mocks.apiSetURL).toHaveBeenCalledWith({
            url: filterUrl,
            data: { name: 'Renamed list', url: filterUrl, enabled: true },
            whitelist: false,
        });
    });
});
