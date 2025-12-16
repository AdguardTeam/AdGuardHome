import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import React, { SetStateAction, Dispatch } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/reducers/modals';
import { deleteRewrite } from 'panel/actions/rewrites';
import { RootState } from 'panel/initialState';

type Props = {
    rewriteToDelete: {
        answer: string;
        domain: string;
        enabled: boolean;
    };
    setRewriteToDelete: Dispatch<SetStateAction<{ answer: string; domain: string, enabled: boolean }>>;
};

export const DeleteRewriteModal = ({ rewriteToDelete, setRewriteToDelete }: Props) => {
    const dispatch = useDispatch();
    const { rewrites } = useSelector((state: RootState) => state);

    const { processingDelete } = rewrites;

    if (!rewriteToDelete.domain) {
        return null;
    }

    const handleDeleteClose = () => {
        setRewriteToDelete({ answer: '', domain: '', enabled: false });
        dispatch(closeModal());
    };

    const handleDeleteConfirm = () => {
        dispatch(deleteRewrite(rewriteToDelete));
        handleDeleteClose();
    };

    return (
        <ModalWrapper id={MODAL_TYPE.DELETE_REWRITE}>
            <ConfirmDialog
                onClose={handleDeleteClose}
                onConfirm={handleDeleteConfirm}
                submitDisabled={processingDelete}
                buttonText={intl.getMessage('remove')}
                cancelText={intl.getMessage('cancel')}
                title={intl.getMessage('rewrites_remove_title')}
                text={intl.getMessage('rewrites_remove_desc', {
                    value: rewriteToDelete.domain,
                })}
                buttonVariant="danger"
                submitTestId="rewrite-delete-confirm"
                cancelTestId="rewrite-delete-cancel"
            />
        </ModalWrapper>
    );
};
