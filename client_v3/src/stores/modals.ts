import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { ModalType } from 'panel/helpers/constants';

type ModalsState = {
    modalId: ModalType | null;
};

const initialState: ModalsState = {
    modalId: null,
};

const [state, setState] = createStore<ModalsState>(initialState);

export const openModal = (modalId: ModalType) => {
    setState({ modalId });
};

export const closeModal = () => {
    setState({ modalId: null });
};

export const modalsState = untrack(() => state);
