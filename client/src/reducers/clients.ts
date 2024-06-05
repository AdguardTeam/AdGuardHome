import { handleActions } from 'redux-actions';

import * as actions from '../actions/clients';

const clients = handleActions(
    {
        [actions.addClientRequest.toString()]: (state: any) => ({
            ...state,
            processingAdding: true,
        }),
        [actions.addClientFailure.toString()]: (state: any) => ({
            ...state,
            processingAdding: false,
        }),
        [actions.addClientSuccess.toString()]: (state: any) => {
            const newState = {
                ...state,
                processingAdding: false,
            };
            return newState;
        },

        [actions.deleteClientRequest.toString()]: (state: any) => ({
            ...state,
            processingDeleting: true,
        }),
        [actions.deleteClientFailure.toString()]: (state: any) => ({
            ...state,
            processingDeleting: false,
        }),
        [actions.deleteClientSuccess.toString()]: (state: any) => {
            const newState = {
                ...state,
                processingDeleting: false,
            };
            return newState;
        },

        [actions.updateClientRequest.toString()]: (state: any) => ({
            ...state,
            processingUpdating: true,
        }),
        [actions.updateClientFailure.toString()]: (state: any) => ({
            ...state,
            processingUpdating: false,
        }),
        [actions.updateClientSuccess.toString()]: (state: any) => {
            const newState = {
                ...state,
                processingUpdating: false,
            };
            return newState;
        },

        [actions.toggleClientModal.toString()]: (state: any, { payload }: any) => {
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
    },
    {
        processing: true,
        processingAdding: false,
        processingDeleting: false,
        processingUpdating: false,
        isModalOpen: false,
        modalClientName: '',
        modalType: '',
    },
);

export default clients;
