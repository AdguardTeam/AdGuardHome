import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ConfigureAllowlistModal } from 'panel/components/FilterLists/blocks/ConfigureAllowlistModal';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { closeModal, openModal } from 'panel/stores/modals';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const mocks = vi.hoisted(() => ({
    apiAddURL: vi.fn(() => Promise.resolve(undefined)),
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
    filteringSetURL: vi.fn(),
    filteringRefresh: vi.fn(),
    filteringConfig: vi.fn(),
    filteringCheckHost: vi.fn(),
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

describe('ConfigureAllowlistModal', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        closeModal();
        openModal(MODAL_TYPE.ADD_ALLOWLIST);
    });

    afterEach(() => {
        closeModal();
    });

    it('does not save when the name is empty', async () => {
        render(() => <ConfigureAllowlistModal modalId={MODAL_TYPE.ADD_ALLOWLIST} />);

        await userEvent.type(
            screen.getByLabelText(copyInDom('blocklist_url_file_path')),
            'https://example.org/list.txt',
        );
        await userEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(screen.getByText(copyInDom('form_error_required'))).toBeInTheDocument();
        expect(mocks.apiAddURL).not.toHaveBeenCalled();
    });

    it('saves the allowlist when both fields are filled', async () => {
        render(() => <ConfigureAllowlistModal modalId={MODAL_TYPE.ADD_ALLOWLIST} />);

        await userEvent.type(screen.getByLabelText(copyInDom('name_label')), 'My allowlist');
        await userEvent.type(
            screen.getByLabelText(copyInDom('blocklist_url_file_path')),
            'https://example.org/list.txt',
        );
        await userEvent.click(screen.getByRole('button', { name: copyInDom('add') }));

        expect(mocks.apiAddURL).toHaveBeenCalledWith({
            name: 'My allowlist',
            url: 'https://example.org/list.txt',
            whitelist: true,
        });
    });
});
