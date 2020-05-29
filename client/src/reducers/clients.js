import { handleActions } from 'redux-actions';

import * as actions from '../actions/clients';

const clients = handleActions({
    [actions.addClientRequest]: (state) => ({ ...state, processingAdding: true }),
    [actions.addClientFailure]: (state) => ({ ...state, processingAdding: false }),
    [actions.addClientSuccess]: (state) => {
        const newState = {
            ...state,
            processingAdding: false,
        };
        return newState;
    },

    [actions.deleteClientRequest]: (state) => ({ ...state, processingDeleting: true }),
    [actions.deleteClientFailure]: (state) => ({ ...state, processingDeleting: false }),
    [actions.deleteClientSuccess]: (state) => {
        const newState = {
            ...state,
            processingDeleting: false,
        };
        return newState;
    },

    [actions.updateClientRequest]: (state) => ({ ...state, processingUpdating: true }),
    [actions.updateClientFailure]: (state) => ({ ...state, processingUpdating: false }),
    [actions.updateClientSuccess]: (state) => {
        const newState = {
            ...state,
            processingUpdating: false,
        };
        return newState;
    },

    [actions.toggleClientModal]: (state, { payload }) => {
        if (payload) {
            const newState = {
                ...state,
                modalType: payload.type || '',
                modalClientName: payload.name || '',
                isModalOpen: !state.isModalOpen,
            };
            return newState;
        }

        const newState = {
            ...state,
            isModalOpen: !state.isModalOpen,
        };
        return newState;
    },
}, {
    processing: true,
    processingAdding: false,
    processingDeleting: false,
    processingUpdating: false,
    isModalOpen: false,
    modalClientName: '',
    modalType: '',
});

export default clients;
