import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import React, { SetStateAction, Dispatch } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/reducers/modals';
import { removeFilter } from 'panel/actions/filtering';
import { RootState } from 'panel/initialState';

type Props = {
    filterToDelete: {
        url: string;
        name: string;
    };
    setFilterToDelete: Dispatch<SetStateAction<{ url: string; name: string }>>;
};

export const DeleteBlocklistModal = ({ filterToDelete, setFilterToDelete }: Props) => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);

    const { processingRemoveFilter } = filtering;

    if (!filterToDelete.url) {
        return null;
    }

    const handleDeleteClose = () => {
        setFilterToDelete({ url: '', name: '' });
        dispatch(closeModal());
    };

    const handleDeleteConfirm = () => {
        dispatch(removeFilter(filterToDelete.url));
        handleDeleteClose();
    };

    return (
        <ModalWrapper id={MODAL_TYPE.DELETE_BLOCKLIST}>
            <ConfirmDialog
                onClose={handleDeleteClose}
                onConfirm={handleDeleteConfirm}
                submitDisabled={processingRemoveFilter}
                buttonText={intl.getMessage('remove')}
                cancelText={intl.getMessage('cancel')}
                title={intl.getMessage('blocklist_remove')}
                text={intl.getMessage('blocklist_remove_desc', {
                    value: filterToDelete.name || filterToDelete.url,
                })}
                buttonVariant="danger"
            />
        </ModalWrapper>
    );
};
