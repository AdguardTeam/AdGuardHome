import { handleActions } from 'redux-actions';

import * as actions from '../actions/rewrites';

const rewrites = handleActions(
    {
        [actions.getRewritesListRequest]: (state) => ({ ...state, processing: true }),
        [actions.getRewritesListFailure]: (state) => ({ ...state, processing: false }),
        [actions.getRewritesListSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                list: payload,
                processing: false,
            };
            return newState;
        },

        [actions.addRewriteRequest]: (state) => ({ ...state, processingAdd: true }),
        [actions.addRewriteFailure]: (state) => ({ ...state, processingAdd: false }),
        [actions.addRewriteSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                list: [...state.list, payload],
                processingAdd: false,
            };
            return newState;
        },

        [actions.deleteRewriteRequest]: (state) => ({ ...state, processingDelete: true }),
        [actions.deleteRewriteFailure]: (state) => ({ ...state, processingDelete: false }),
        [actions.deleteRewriteSuccess]: (state) => ({ ...state, processingDelete: false }),

        [actions.toggleRewritesModal]: (state) => {
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
        isModalOpen: false,
        list: [],
    },
);

export default rewrites;
