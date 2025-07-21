import { handleActions } from 'redux-actions';

import * as actions from '../actions/rewrites';

const rewrites = handleActions(
    {
        [actions.getRewritesListRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.getRewritesListFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.getRewritesListSuccess.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                list: payload,
                processing: false,
            };
            return newState;
        },

        [actions.addRewriteRequest.toString()]: (state: any) => ({
            ...state,
            processingAdd: true,
        }),
        [actions.addRewriteFailure.toString()]: (state: any) => ({
            ...state,
            processingAdd: false,
        }),
        [actions.addRewriteSuccess.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                list: [...state.list, payload],
                processingAdd: false,
            };
            return newState;
        },

        [actions.deleteRewriteRequest.toString()]: (state: any) => ({
            ...state,
            processingDelete: true,
        }),
        [actions.deleteRewriteFailure.toString()]: (state: any) => ({
            ...state,
            processingDelete: false,
        }),
        [actions.deleteRewriteSuccess.toString()]: (state: any) => ({
            ...state,
            processingDelete: false,
        }),

        [actions.updateRewriteRequest.toString()]: (state: any) => ({
            ...state,
            processingUpdate: true,
        }),
        [actions.updateRewriteFailure.toString()]: (state: any) => ({
            ...state,
            processingUpdate: false,
        }),
        [actions.updateRewriteSuccess.toString()]: (state: any) => {
            const newState = {
                ...state,
                processingUpdate: false,
            };
            return newState;
        },

        [actions.toggleRewritesModal.toString()]: (state: any, { payload }: any) => {
            if (payload) {
                const newState = {
                    ...state,
                    modalType: payload.type || '',
                    isModalOpen: !state.isModalOpen,
                    currentRewrite: payload.currentRewrite,
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
        processingAdd: false,
        processingDelete: false,
        processingUpdate: false,
        isModalOpen: false,
        modalType: '',
        currentRewrite: {},
        list: [],
    },
);

export default rewrites;
