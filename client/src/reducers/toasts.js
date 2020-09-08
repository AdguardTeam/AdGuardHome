import { handleActions } from 'redux-actions';
import { nanoid } from 'nanoid';

import {
    addErrorToast, addNoticeToast, addSuccessToast,
} from '../actions/toasts';
import { removeToast } from '../actions';
import { TOAST_TYPES } from '../helpers/constants';

const toasts = handleActions({
    [addErrorToast]: (state, { payload }) => {
        const message = payload.error.toString();
        console.error(payload.error);

        const errorToast = {
            id: nanoid(),
            message,
            options: payload.options,
            type: TOAST_TYPES.ERROR,
        };

        const newState = { ...state, notices: [...state.notices, errorToast] };
        return newState;
    },
    [addSuccessToast]: (state, { payload }) => {
        const successToast = {
            id: nanoid(),
            message: payload,
            type: TOAST_TYPES.SUCCESS,
        };

        const newState = { ...state, notices: [...state.notices, successToast] };
        return newState;
    },
    [addNoticeToast]: (state, { payload }) => {
        const noticeToast = {
            id: nanoid(),
            message: payload.error.toString(),
            options: payload.options,
            type: TOAST_TYPES.NOTICE,
        };

        const newState = { ...state, notices: [...state.notices, noticeToast] };
        return newState;
    },
    [removeToast]: (state, { payload }) => {
        const filtered = state.notices.filter((notice) => notice.id !== payload);
        const newState = { ...state, notices: filtered };
        return newState;
    },
}, { notices: [] });

export default toasts;
