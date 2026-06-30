import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/stores/modals';
import { deleteRewrite, rewritesState } from 'panel/stores/rewrites';

type Props = {
    rewriteToDelete: {
        answer: string;
        domain: string;
        enabled: boolean;
    };
    setRewriteToDelete: (value: { answer: string; domain: string; enabled: boolean }) => void;
    onConfirm?: () => boolean | void | Promise<boolean | void>;
    onClose?: () => void;
};

export const DeleteRewriteModal = (props: Props) => {
    const handleDeleteClose = () => {
        props.setRewriteToDelete({ answer: '', domain: '', enabled: false });
        props.onClose?.();
        closeModal();
    };

    const handleDeleteConfirm = async () => {
        if (props.onConfirm) {
            const shouldClose = await props.onConfirm();

            if (shouldClose !== false) {
                handleDeleteClose();
            }

            return;
        }

        deleteRewrite(props.rewriteToDelete);
        handleDeleteClose();
    };

    return (
        <Show when={props.rewriteToDelete.domain}>
            <ModalWrapper id={MODAL_TYPE.DELETE_REWRITE}>
                <ConfirmDialog
                    onClose={handleDeleteClose}
                    onConfirm={handleDeleteConfirm}
                    submitDisabled={rewritesState.processingDelete}
                    buttonText={intl.getMessage('remove')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('rewrites_remove_title')}
                    text={intl.getMessage('rewrites_remove_desc', {
                        value: props.rewriteToDelete.domain,
                    })}
                    buttonVariant="danger"
                    submitTestId="rewrite-delete-confirm"
                    cancelTestId="rewrite-delete-cancel"
                />
            </ModalWrapper>
        </Show>
    );
};
