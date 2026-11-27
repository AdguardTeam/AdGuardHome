import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ConfigureRewritesModal } from 'panel/components/FilterLists/blocks/ConfigureRewritesModal';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { closeModal, openModal } from 'panel/stores/modals';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const mocks = vi.hoisted(() => ({
    onSubmit: vi.fn(() => true),
}));

vi.mock('panel/stores/rewrites', () => ({
    rewritesState: { list: [] },
    addRewrite: vi.fn(),
    updateRewrite: vi.fn(),
}));

const SELF_REWRITE = 'hook.example.com';

const renderModal = (
    modalId: 'ADD_REWRITE' | 'EDIT_REWRITE',
    rewriteToEdit?: { domain: string; answer: string; enabled: boolean },
) => {
    openModal(modalId);

    return render(() => (
        <ConfigureRewritesModal
            modalId={modalId}
            rewriteToEdit={rewriteToEdit}
            onSubmit={mocks.onSubmit}
        />
    ));
};

const submitSelfRewrite = async (buttonName: string) => {
    await userEvent.type(document.getElementById('domain') as HTMLInputElement, SELF_REWRITE);
    await userEvent.type(document.getElementById('answer') as HTMLInputElement, SELF_REWRITE);
    await userEvent.click(screen.getByRole('button', { name: buttonName }));
};

describe('ConfigureRewritesModal self-rewrite', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        closeModal();
    });

    afterEach(() => {
        closeModal();
    });

    it('submits a new rewrite whose answer repeats the domain', async () => {
        renderModal(MODAL_TYPE.ADD_REWRITE);

        await submitSelfRewrite(copyInDom('add'));

        expect(screen.queryByText(copyInDom('dns_rewrite_same'))).toBeNull();
        expect(mocks.onSubmit).toHaveBeenCalledWith({
            domain: SELF_REWRITE,
            answer: SELF_REWRITE,
            enabled: false,
        });
    });

    it('submits an edited rewrite whose answer repeats the domain', async () => {
        renderModal(MODAL_TYPE.EDIT_REWRITE, {
            domain: 'old.example.com',
            answer: '192.168.1.1',
            enabled: true,
        });

        const domainInput = document.getElementById('domain') as HTMLInputElement;
        const answerInput = document.getElementById('answer') as HTMLInputElement;
        await userEvent.clear(domainInput);
        await userEvent.clear(answerInput);
        await userEvent.type(domainInput, SELF_REWRITE);
        await userEvent.type(answerInput, SELF_REWRITE);
        await userEvent.click(screen.getByRole('button', { name: copyInDom('save') }));

        expect(screen.queryByText(copyInDom('dns_rewrite_same'))).toBeNull();
        expect(mocks.onSubmit).toHaveBeenCalledWith({
            domain: SELF_REWRITE,
            answer: SELF_REWRITE,
            enabled: true,
        });
    });
});
