import { createAction, handleActions } from 'redux-actions';
import { ModalType } from 'panel/helpers/constants';
import { ModalsData } from 'panel/initialState';

export const openModal = createAction('OPEN_MODAL', (modalId: ModalType) => ({
    modalId,
}));
export const closeModal = createAction('CLOSE_MODAL');

const initialState: ModalsData = {
    modalId: null,
};

const modals = handleActions(
    {
        [openModal.toString()]: (state: ModalsData, { payload }: { payload: { modalId: ModalType } }) => ({
            modalId: payload.modalId,
        }),
        [closeModal.toString()]: (): ModalsData => ({
            modalId: null,
        }),
    },
    initialState,
);

export default modals;
