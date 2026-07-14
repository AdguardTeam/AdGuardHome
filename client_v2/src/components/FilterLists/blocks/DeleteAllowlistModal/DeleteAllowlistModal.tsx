import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/stores/modals';
import { removeFilter, filteringState } from 'panel/stores/filtering';

type Props = {
    filterToDelete: {
        url: string;
        name: string;
    };
    setFilterToDelete: (value: { url: string; name: string }) => void;
};

export const DeleteAllowlistModal = (props: Props) => {
    const handleDeleteClose = () => {
        props.setFilterToDelete({ url: '', name: '' });
        closeModal();
    };

    const handleDeleteConfirm = () => {
        removeFilter(props.filterToDelete.url, true, props.filterToDelete.name);
        handleDeleteClose();
    };

    return (
        <Show when={props.filterToDelete.url}>
            <ModalWrapper id={MODAL_TYPE.DELETE_ALLOWLIST}>
                <ConfirmDialog
                    onClose={handleDeleteClose}
                    onConfirm={handleDeleteConfirm}
                    submitDisabled={filteringState.processingRemoveFilter}
                    buttonText={intl.getMessage('remove')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('allowlist_remove')}
                    text={intl.getMessage('allowlist_remove_desc', {
                        value: props.filterToDelete.name || props.filterToDelete.url,
                    })}
                    buttonVariant="danger"
                />
            </ModalWrapper>
        </Show>
    );
};
